package signal

import (
	"context"
	"fmt"
	"strings"
	"time"

	"MRG/internal/domain/contact"
	"MRG/internal/domain/lead"
	"MRG/internal/domain/message"
	settings_store "MRG/internal/infrastructure/storage/settings"
	"MRG/internal/infrastructure/textnorm"
	lead_usecase "MRG/internal/usecase/lead"
)

type Service struct {
	messageRepo message.Repository
	contactRepo contact.Repository
	leadRepo    lead.Repository
	leadUC      *lead_usecase.UseCase
	sieve       lead.Sieve
	settings    *settings_store.Store
}

func NewService(
	messageRepo message.Repository,
	contactRepo contact.Repository,
	leadRepo lead.Repository,
	leadUC *lead_usecase.UseCase,
	sieve lead.Sieve,
	settings *settings_store.Store,
) *Service {
	return &Service{
		messageRepo: messageRepo,
		contactRepo: contactRepo,
		leadRepo:    leadRepo,
		leadUC:      leadUC,
		sieve:       sieve,
		settings:    settings,
	}
}

func (s *Service) BindContact(ctx context.Context, in BindContactInput) error {
	msg, err := s.messageRepo.FindByID(ctx, in.TenantID, in.MessageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return ErrSignalNotFound
	}

	return s.leadUC.SetContactMerchant(ctx, in.TenantID, msg.SenderID(), msg.SenderName(), msg.SenderUsername(), in.MerchantID)
}

func (s *Service) GetInbox(ctx context.Context, q InboxQuery) (*InboxPage, error) {
	f := message.ListFilter{
		Limit:    q.Limit,
		Offset:   q.Offset,
		Cursor:   q.Cursor,
		FromDate: q.FromDate,
		ToDate:   q.ToDate,
	}
	page, err := s.messageRepo.ListPage(ctx, q.TenantID, f)
	if err != nil {
		return nil, err
	}
	msgs := page.Items

	msgIDs := make([]string, len(msgs))
	for i, m := range msgs {
		msgIDs[i] = m.ID()
	}
	msgToLead, err := s.leadRepo.FindByMessageIDs(ctx, q.TenantID, msgIDs)
	if err != nil {
		return nil, err
	}

	senderIDs := make([]int64, 0, len(msgs))
	seen := map[int64]bool{}
	for _, m := range msgs {
		if !seen[m.SenderID()] {
			senderIDs = append(senderIDs, m.SenderID())
			seen[m.SenderID()] = true
		}
	}
	senderChatCount, err := s.messageRepo.CountSenderInChats(ctx, q.TenantID, senderIDs)
	if err != nil {
		return nil, err
	}

	out := make([]DTO, 0, len(msgs))
	for _, m := range msgs {
		if !q.ShowArchived && m.IsIgnored() {
			continue
		}
		dto := toSignalDTO(m, senderChatCount[m.SenderID()])
		if ref, ok := msgToLead[m.ID()]; ok {
			dto.LeadID = &ref.ID
			dto.LeadScore = &ref.Score
		}
		s.applyInboxCategory(ctx, &dto)
		if !matchInboxTab(dto, q.Tab) || !matchInboxCategory(dto, q.Category) {
			continue
		}
		out = append(out, dto)
	}
	return &InboxPage{
		Items:      out,
		NextCursor: page.NextCursor,
	}, nil
}

func (s *Service) GetStats(ctx context.Context, tenantID string, days int) (*message.IngestStats, error) {
	return s.messageRepo.GetIngestStats(ctx, tenantID, days)
}

func (s *Service) GetChart(ctx context.Context, tenantID string, from, to time.Time) ([]message.ChartDayBucket, error) {
	return s.messageRepo.GetChartData(ctx, tenantID, from, to)
}

type sieveMetaTeacher interface {
	AddReferencePointWithMeta(ctx context.Context, text string, isLead bool, direction string) error
}

type sieveMetaTeacherWithFlags interface {
	AddReferencePointWithMetaFlags(ctx context.Context, text string, isLead bool, direction string, flags []string) error
}

func (s *Service) FeedbackSignal(ctx context.Context, in FeedbackInput) (FeedbackResult, error) {
	if s.sieve == nil {
		return FeedbackResult{}, ErrSieveUnavailable
	}

	msg, err := s.messageRepo.FindByID(ctx, in.TenantID, in.MessageID)
	if err != nil {
		return FeedbackResult{}, err
	}
	if msg == nil {
		return FeedbackResult{}, ErrSignalNotFound
	}
	if msg.Text() == "" {
		return FeedbackResult{}, ErrEmptySignalText
	}

	normalizedCategory, hasCategory := normalizeCategory(in.Category)
	isLeadForTraining := in.IsLead
	if hasCategory {
		isLeadForTraining = normalizedCategory != "noise"
	}

	directionForTraining := ""
	if hasCategory {
		directionForTraining = categoryToDirection(normalizedCategory)
	}
	flagsForTraining := normalizeSemanticFlags(in.SemanticFlags)
	flagsForTraining = mergeSemanticFlags(flagsForTraining, suggestSemanticFlags(msg.Text(), normalizedCategory, directionForTraining))

	embeddingText := textnorm.ForEmbedding(msg.Text())
	if embeddingText == "" {
		embeddingText = msg.Text()
	}

	if teacher, ok := s.sieve.(sieveMetaTeacherWithFlags); ok {
		if err := teacher.AddReferencePointWithMetaFlags(ctx, embeddingText, isLeadForTraining, directionForTraining, flagsForTraining); err != nil {
			return FeedbackResult{}, err
		}
	} else if teacher, ok := s.sieve.(sieveMetaTeacher); ok {
		if err := teacher.AddReferencePointWithMeta(ctx, embeddingText, isLeadForTraining, directionForTraining); err != nil {
			return FeedbackResult{}, err
		}
	} else if err := s.sieve.AddReferencePoint(ctx, embeddingText, isLeadForTraining); err != nil {
		return FeedbackResult{}, err
	}

	if hasCategory {
		if err := s.messageRepo.SetSemanticDirection(ctx, in.TenantID, in.MessageID, &directionForTraining); err != nil {
			return FeedbackResult{}, err
		}
	}

	if err := s.messageRepo.SetClassification(ctx, in.TenantID, in.MessageID, isLeadForTraining); err != nil {
		return FeedbackResult{}, err
	}

	score := 1.0
	if msg.SimilarityScore() != nil {
		score = *msg.SimilarityScore()
	}

	existing, err := s.leadRepo.FindByMessageID(ctx, in.TenantID, in.MessageID)
	if err != nil {
		return FeedbackResult{}, err
	}

	if isLeadForTraining {
		var l *lead.Lead
		if existing != nil {
			l = existing
			l.MarkAsConfirmed()
		} else {
			l, err = lead.Detect(
				in.TenantID, msg.ID(),
				msg.ChatID(), msg.ChatTitle(),
				msg.SenderID(), msg.SenderName(), msg.SenderUsername(), msg.Text(),
				score,
			)
			if err != nil {
				return FeedbackResult{}, err
			}
			l.MarkAsConfirmed()
		}

		if directionForTraining != "" {
			l.SetSemanticDirection(directionForTraining)
		}
		if in.Category != "" {
			l.SetSemanticCategory(in.Category)
		}

		var saveErr error
		if existing != nil {
			saveErr = s.leadRepo.Update(ctx, l)
		} else {
			saveErr = s.leadRepo.Save(ctx, l)
		}
		if saveErr != nil {
			return FeedbackResult{}, saveErr
		}
		leadID := l.ID()
		return FeedbackResult{OK: true, IsLead: true, LeadID: &leadID, SemanticFlags: flagsForTraining}, nil
	}

	// Case: isLeadForTraining = false (False Positive)
	var l *lead.Lead
	if existing != nil {
		l = existing
		l.MarkAsFalsePositive()
	} else {
		l, err = lead.Detect(
			in.TenantID, msg.ID(),
			msg.ChatID(), msg.ChatTitle(),
			msg.SenderID(), msg.SenderName(), msg.SenderUsername(), msg.Text(),
			score,
		)
		if err != nil {
			return FeedbackResult{}, err
		}
		l.MarkAsFalsePositive()
	}

	if directionForTraining != "" {
		l.SetSemanticDirection(directionForTraining)
	}
	if in.Category != "" {
		l.SetSemanticCategory(in.Category)
	} else {
		l.SetSemanticCategory("noise")
	}

	var saveErr error
	if existing != nil {
		saveErr = s.leadRepo.Update(ctx, l)
	} else {
		saveErr = s.leadRepo.Save(ctx, l)
	}
	if saveErr != nil {
		return FeedbackResult{}, saveErr
	}

	return FeedbackResult{OK: true, IsLead: false, SemanticFlags: flagsForTraining}, nil
}

func (s *Service) FlagSignal(ctx context.Context, in FlagInput) error {
	if in.Field != "is_ignored" && in.Field != "is_team_member" && in.Field != "is_viewed" && in.Field != "is_spam_sender" {
		return ErrInvalidFlagField
	}
	if in.Field == "is_spam_sender" {
		return s.setSpamSender(ctx, in)
	}
	return s.messageRepo.SetFlag(ctx, in.TenantID, in.MessageID, in.Field, in.Value)
}

func (s *Service) setSpamSender(ctx context.Context, in FlagInput) error {
	msg, err := s.messageRepo.FindByID(ctx, in.TenantID, in.MessageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return ErrSignalNotFound
	}
	if err := s.messageRepo.SetFlag(ctx, in.TenantID, in.MessageID, "is_spam_sender", in.Value); err != nil {
		return err
	}
	if s.contactRepo == nil {
		return nil
	}
	c, err := s.contactRepo.FindBySenderID(ctx, in.TenantID, msg.SenderID())
	if err != nil {
		return err
	}
	if c == nil {
		c = contact.New(in.TenantID, msg.SenderID(), msg.SenderName(), msg.SenderUsername())
	}
	c.SetSpam(in.Value)
	return s.contactRepo.Update(ctx, c)
}

func (s *Service) GetSenderHistory(ctx context.Context, tenantID, senderIDStr string, limit int) ([]DTO, error) {
	var senderID int64
	if _, err := fmt.Sscanf(senderIDStr, "%d", &senderID); err != nil || senderID == 0 {
		return nil, ErrInvalidSenderID
	}

	msgs, err := s.messageRepo.FindBySender(ctx, tenantID, senderID, limit, 0)
	if err != nil {
		return nil, err
	}

	out := make([]DTO, 0, len(msgs))
	for _, m := range msgs {
		dto := toSignalDTO(m, 0)
		s.applyInboxCategory(ctx, &dto)
		out = append(out, dto)
	}
	return out, nil
}

func toSignalDTO(m *message.Message, otherChatsCount int) DTO {
	name := strings.TrimSpace(m.SenderName())
	if name == "" || strings.HasPrefix(name, "Dialog ") || strings.HasPrefix(name, "User ") {
		name = strings.TrimPrefix(strings.TrimSpace(m.SenderUsername()), "@")
	}
	if name == "" || strings.HasPrefix(name, "Dialog ") {
		chatTitle := strings.TrimSpace(m.ChatTitle())
		if chatTitle != "" && chatTitle != "Unknown" && !strings.HasPrefix(chatTitle, "Dialog ") {
			name = chatTitle
		}
	}
	if name == "" {
		name = fmt.Sprintf("User %d", m.SenderID())
	}
	contact := m.SenderUsername()
	if contact != "" && !strings.HasPrefix(contact, "@") {
		contact = "@" + contact
	}
	return DTO{
		ID:                     m.ID(),
		ChatTitle:              m.ChatTitle(),
		ChatPeerType:           m.ChatPeerType(),
		FromName:               name,
		Contact:                contact,
		SenderTelegramID:       m.SenderID(),
		Text:                   m.Text(),
		Date:                   m.CreatedAt().UTC().Format(time.RFC3339),
		SimilarityScore:        m.SimilarityScore(),
		ClassifiedAsLead:       m.ClassifiedAsLead(),
		SemanticDirection:      m.SemanticDirection(),
		SemanticCategory:       "",
		ClassificationReason:   "",
		TraderScore:            0,
		MerchantScore:          0,
		ProcessingRequestScore: 0,
		PSOfferScore:           0,
		NoiseScore:             0,
		PrimaryLabel:           "",
		PrimaryPercent:         0,
		IsIgnored:              m.IsIgnored(),
		IsTeamMember:           m.IsTeamMember(),
		IsSpamSender:           m.IsSpamSender(),
		IsViewed:               m.IsViewed(),
		IsNew:                  !m.IsViewed(),
		OtherChatsCount:        otherChatsCount,
		SemanticFlags:          nil,
	}
}

func (s *Service) applyInboxCategory(ctx context.Context, dto *DTO) {
	assignCategoryScores(dto)
	assignPrimaryLabel(dto)

	dto.SemanticFlags = suggestSemanticFlags(dto.Text, "", "")

	if dto.IsIgnored {
		dto.SemanticCategory = "noise"
		dto.ClassificationReason = "Помечено менеджером как скрытое"
		return
	}
	if dto.IsTeamMember {
		dto.SemanticCategory = "noise"
		dto.ClassificationReason = "Отправитель помечен как участник команды"
		return
	}

	if dto.LeadID != nil {
		if dto.SemanticDirection != nil {
			if category, ok := mapDirectionToCategory(*dto.SemanticDirection); ok {
				dto.SemanticCategory = category
				if category == "noise" {
					dto.ClassificationReason = "Сигнал вручную помечен как шум"
					return
				}
				dto.ClassificationReason = "Есть связанный лид, категория зафиксирована по semantic_direction"
				dto.SemanticFlags = mergeSemanticFlags(dto.SemanticFlags, suggestSemanticFlags(dto.Text, category, *dto.SemanticDirection))
				return
			}
		}

		bestCategory, bestScore := bestBusinessCategory(*dto)
		if bestCategory != "" {
			dto.SemanticCategory = bestCategory
			dto.ClassificationReason = fmt.Sprintf("Есть связанный лид, используем лучшую бизнес-категорию %s (%.2f)", bestCategory, bestScore)
			dto.SemanticFlags = mergeSemanticFlags(dto.SemanticFlags, suggestSemanticFlags(dto.Text, bestCategory, ""))
			return
		}

		dto.SemanticCategory = "merchants"
		dto.ClassificationReason = "Есть связанный лид, noise для него недопустим"
		return
	}

	bestCategory, bestScore := bestBusinessCategory(*dto)
	threshold := s.categoryThreshold(ctx, bestCategory)
	if bestCategory != "" && bestScore >= threshold {
		dto.SemanticCategory = bestCategory
		dto.SemanticFlags = mergeSemanticFlags(dto.SemanticFlags, suggestSemanticFlags(dto.Text, bestCategory, ""))
		switch {
		case dto.SemanticDirection != nil && strings.TrimSpace(*dto.SemanticDirection) != "":
			dto.ClassificationReason = fmt.Sprintf("RAG направление: %s (%.2f >= %.2f)", *dto.SemanticDirection, bestScore, threshold)
			dto.SemanticFlags = mergeSemanticFlags(dto.SemanticFlags, suggestSemanticFlags(dto.Text, bestCategory, *dto.SemanticDirection))
		case dto.LeadID != nil:
			dto.ClassificationReason = fmt.Sprintf("Есть запись лида и score %.2f >= %.2f", bestScore, threshold)
		default:
			dto.ClassificationReason = fmt.Sprintf("Score %.2f >= %.2f", bestScore, threshold)
		}
		return
	}

	dto.SemanticCategory = "noise"
	if bestCategory == "" {
		dto.ClassificationReason = "Нет уверенной семантической категории"
		return
	}
	dto.ClassificationReason = fmt.Sprintf("Score %.2f ниже порога %.2f для %s", bestScore, threshold, bestCategory)
}

func assignPrimaryLabel(dto *DTO) {
	if dto.IsIgnored || dto.IsTeamMember {
		dto.PrimaryLabel = "noise"
		dto.PrimaryPercent = 100
		return
	}
	if dto.SemanticDirection != nil {
		if category, ok := mapDirectionToCategory(*dto.SemanticDirection); ok && category == "noise" {
			dto.PrimaryLabel = "noise"
			dto.PrimaryPercent = 100
			return
		}
	}
	if dto.LeadID != nil {
		dto.PrimaryLabel = "lead"
		dto.PrimaryPercent = 99
		return
	}

	bestLabel := "noise"
	bestScore := dto.NoiseScore
	candidates := []struct {
		label string
		score float64
	}{
		{label: "merchant", score: dto.MerchantScore},
		{label: "trader", score: dto.TraderScore},
		{label: "ps_offer", score: dto.PSOfferScore},
	}
	for _, c := range candidates {
		if c.score > bestScore {
			bestScore = c.score
			bestLabel = c.label
		}
	}
	dto.PrimaryLabel = bestLabel
	dto.PrimaryPercent = int(clamp01(bestScore)*100 + 0.5)
}

func matchInboxTab(dto DTO, tab string) bool {
	switch tab {
	case "lead":
		return isLeadTabSignal(dto)
	default:
		return true
	}
}

func matchInboxCategory(dto DTO, category string) bool {
	if category == "all" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(dto.SemanticCategory), category)
}

func isLeadTabSignal(dto DTO) bool {
	if dto.IsIgnored || dto.IsTeamMember {
		return false
	}
	if dto.LeadID != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(dto.SemanticCategory)) {
	case "trader_search", "traders", "merchants", "ps_offers":
		return true
	default:
		return false
	}
}

func mapDirectionToCategory(direction string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "trader_search", "search_trader", "search_traders":
		return "trader_search", true
	case "trader", "traders":
		return "traders", true
	case "merchant", "merchants", "merch":
		return "merchants", true
	case "processing_request", "processing_requests", "request_processing":
		return "merchants", true
	case "ps_offer", "ps_offers":
		return "ps_offers", true
	case "noise", "spam":
		return "noise", true
	default:
		return "", false
	}
}

func assignCategoryScores(dto *DTO) {
	dto.TraderScore = 0
	dto.MerchantScore = 0
	dto.ProcessingRequestScore = 0
	dto.PSOfferScore = 0
	dto.NoiseScore = 1

	score := 0.0
	if dto.SimilarityScore != nil {
		score = clamp01(*dto.SimilarityScore)
	}

	if dto.SemanticDirection != nil {
		if category, ok := mapDirectionToCategory(*dto.SemanticDirection); ok {
			switch category {
			case "trader_search", "traders":
				dto.TraderScore = score
				dto.NoiseScore = clamp01(1 - score)
			case "merchants":
				dto.MerchantScore = score
				dto.NoiseScore = clamp01(1 - score)
			case "ps_offers":
				dto.PSOfferScore = score
				dto.NoiseScore = clamp01(1 - score)
			case "noise":
				dto.NoiseScore = maxf(dto.NoiseScore, score)
			}
		}
	}

	if dto.IsIgnored || dto.IsTeamMember {
		dto.NoiseScore = 1
	}

	if dto.LeadID != nil {
		dto.NoiseScore = minf(dto.NoiseScore, 0.05)
	}
}

func normalizeCategory(category string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "trader_search", "search_trader", "search_traders":
		return "trader_search", true
	case "traders", "trader":
		return "traders", true
	case "merchant", "merchants", "merch":
		return "merchants", true
	case "processing_requests", "processing_request", "request":
		return "merchants", true
	case "ps_offers", "ps_offer", "offer":
		return "ps_offers", true
	case "noise", "spam":
		return "noise", true
	default:
		return "", false
	}
}

func categoryToDirection(category string) string {
	switch category {
	case "trader_search":
		return "trader_search"
	case "traders":
		return "trader"
	case "merchants":
		return "merchant"
	case "ps_offers":
		return "ps_offer"
	default:
		return "noise"
	}
}

func bestBusinessCategory(dto DTO) (string, float64) {
	candidates := []struct {
		category string
		score    float64
	}{
		{category: "traders", score: dto.TraderScore},
		{category: "merchants", score: dto.MerchantScore},
		{category: "ps_offers", score: dto.PSOfferScore},
	}

	bestCategory := ""
	bestScore := 0.0
	for _, candidate := range candidates {
		if candidate.score > bestScore {
			bestCategory = candidate.category
			bestScore = candidate.score
		}
	}
	return bestCategory, bestScore
}

func (s *Service) categoryThreshold(ctx context.Context, category string) float64 {
	if s.settings == nil {
		return 0.60
	}

	switch category {
	case "trader_search":
		return s.settings.GetFloat(ctx, "trader_threshold", 0.60)
	case "traders":
		return s.settings.GetFloat(ctx, "trader_threshold", 0.60)
	case "merchants":
		return s.settings.GetFloat(ctx, "merchant_threshold", 0.60)
	case "ps_offers":
		return s.settings.GetFloat(ctx, "ps_offer_threshold", 0.60)
	default:
		return 0.60
	}
}

func normalizeSemanticFlags(flags []string) []string {
	if len(flags) == 0 {
		return nil
	}
	out := make([]string, 0, len(flags))
	seen := make(map[string]struct{}, len(flags))
	for _, f := range flags {
		v := strings.ToLower(strings.TrimSpace(f))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func mergeSemanticFlags(base, extra []string) []string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(base)+len(extra))
	out := make([]string, 0, len(base)+len(extra))
	for _, src := range [][]string{base, extra} {
		for _, f := range src {
			v := strings.ToLower(strings.TrimSpace(f))
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func suggestSemanticFlags(text, category, direction string) []string {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return nil
	}
	flags := make([]string, 0, 6)
	add := func(v string) {
		flags = append(flags, v)
	}

	if strings.Contains(lower, "telegram") &&
		(strings.Contains(lower, "terms of service") ||
			strings.Contains(lower, "appeal") ||
			strings.Contains(lower, "moderator") ||
			strings.Contains(lower, "suspend")) {
		add("system_notice")
		add("compliance")
	}

	if strings.Contains(lower, "http://") || strings.Contains(lower, "https://") || strings.Contains(lower, "t.me/") {
		add("has_links")
	}
	if strings.Contains(lower, "@") || strings.Contains(lower, "dm") || strings.Contains(lower, "в лс") || strings.Contains(lower, "пишите") {
		add("contact_exchange")
	}
	if strings.Contains(lower, "комисс") || strings.Contains(lower, "rate") || strings.Contains(lower, "ставк") {
		add("pricing")
	}
	if strings.Contains(lower, "ищу") || strings.Contains(lower, "нужен") || strings.Contains(lower, "need") || strings.Contains(lower, "looking for") {
		add("request_intent")
	}
	if strings.Contains(lower, "предлага") || strings.Contains(lower, "offer") || strings.Contains(lower, "we provide") {
		add("offer_intent")
	}

	// Task 4: Search for trader / Trader
	if strings.Contains(lower, "трейдер") || strings.Contains(lower, "trader") ||
		strings.Contains(lower, "ищу трейдера") || strings.Contains(lower, "search trader") {
		add("trader_search")
	}

	// Traffic detection (Point 4)
	if strings.Contains(lower, "трафик") || strings.Contains(lower, "traffic") ||
		strings.Contains(lower, "реквизит") || strings.Contains(lower, "рекв") ||
		strings.Contains(lower, "траф") || strings.Contains(lower, "обработка") ||
		strings.Contains(lower, "processing") {
		add("has_traffic")
	}

	// Merchant intent hints
	if strings.Contains(lower, "подключить") || strings.Contains(lower, "интеграция") ||
		strings.Contains(lower, "платежка") || strings.Contains(lower, "эквайринг") {
		add("merchant_intent")
	}

	switch category {
	case "noise":
		add("noise")
	case "trader_search":
		add("trader_search")
	case "traders":
		add("trader")
	case "merchants":
		add("merchant")
	case "ps_offers":
		add("ps_offer")
	}
	if direction != "" {
		add("direction_" + strings.ToLower(strings.TrimSpace(direction)))
	}

	return normalizeSemanticFlags(flags)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxf(vals ...float64) float64 {
	out := 0.0
	for _, v := range vals {
		if v > out {
			out = v
		}
	}
	return out
}
