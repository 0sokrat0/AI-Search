package lead

import (
	"context"
	"sort"
	"strings"
	"time"

	"MRG/internal/domain/contact"
	"MRG/internal/domain/lead"
	"MRG/internal/domain/message"
)

type sieveMetaTeacher interface {
	AddReferencePointWithMeta(ctx context.Context, text string, isLead bool, direction string) error
}

type UseCase struct {
	leads    lead.Repository
	messages message.Repository
	contacts contact.Repository
	sieve    lead.Sieve
}

func New(leads lead.Repository, messages message.Repository, contacts contact.Repository) *UseCase {
	return &UseCase{leads: leads, messages: messages, contacts: contacts}
}

func (uc *UseCase) WithSieve(sieve lead.Sieve) *UseCase {
	uc.sieve = sieve
	return uc
}

func (uc *UseCase) List(ctx context.Context, tenantID string, f lead.ListFilter) ([]*lead.Lead, error) {
	return uc.leads.List(ctx, tenantID, f)
}

func (uc *UseCase) ListPage(ctx context.Context, tenantID string, f lead.ListFilter) (*lead.ListPage, error) {
	return uc.leads.ListPage(ctx, tenantID, f)
}

func (uc *UseCase) GetByID(ctx context.Context, tenantID, id string) (*lead.Lead, error) {
	l, err := uc.leads.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, lead.ErrLeadNotFound
	}
	return l, nil
}

func (uc *UseCase) Approve(ctx context.Context, tenantID, id string) (*lead.Lead, error) {
	return uc.setFeedback(ctx, tenantID, id, true)
}

func (uc *UseCase) Reject(ctx context.Context, tenantID, id string) (*lead.Lead, error) {
	return uc.setFeedback(ctx, tenantID, id, false)
}
func (uc *UseCase) MarkControversial(ctx context.Context, tenantID, id string) (*lead.Lead, error) {
	l, err := uc.leads.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, lead.ErrLeadNotFound
	}
	l.MarkAsControversial()
	if err := uc.leads.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (uc *UseCase) setFeedback(ctx context.Context, tenantID, id string, good bool) (*lead.Lead, error) {
	l, err := uc.leads.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, lead.ErrLeadNotFound
	}
	if good {
		l.Approve()
	} else {
		l.Reject()
	}
	if err := uc.leads.Update(ctx, l); err != nil {
		return nil, err
	}

	if uc.sieve != nil && l.Text() != "" {
		go func() {
			if teacher, ok := uc.sieve.(sieveMetaTeacher); ok {
				_ = teacher.AddReferencePointWithMeta(context.Background(), l.Text(), good, l.SemanticDirection())
				return
			}
			_ = uc.sieve.AddReferencePoint(context.Background(), l.Text(), good)
		}()
	}
	return l, nil
}

func (uc *UseCase) AdvanceStatus(ctx context.Context, tenantID, id string, to lead.Status) (*lead.Lead, error) {
	l, err := uc.leads.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, lead.ErrLeadNotFound
	}
	if err := l.Advance(to); err != nil {
		return nil, err
	}
	if err := uc.leads.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (uc *UseCase) SetStatus(ctx context.Context, tenantID, id string, to lead.Status) (*lead.Lead, error) {
	l, err := uc.leads.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, lead.ErrLeadNotFound
	}
	if err := l.SetStatus(to); err != nil {
		return nil, err
	}
	if err := uc.leads.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (uc *UseCase) ClaimOwnership(ctx context.Context, tenantID, id, ownerID, ownerName string) (*lead.Lead, error) {
	return uc.leads.ClaimOwnership(ctx, tenantID, id, ownerID, ownerName)
}

func (uc *UseCase) ReleaseOwnership(ctx context.Context, tenantID, id, ownerID string) (*lead.Lead, error) {
	return uc.leads.ReleaseOwnership(ctx, tenantID, id, ownerID)
}

func (uc *UseCase) SetMerchant(ctx context.Context, tenantID, id, merchantID string) (*lead.Lead, error) {
	l, err := uc.leads.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, lead.ErrLeadNotFound
	}
	l.SetMerchant(merchantID)
	if err := uc.leads.Update(ctx, l); err != nil {
		return nil, err
	}

	if err := uc.bindSenderToMerchant(ctx, tenantID, l.SenderID(), l.SenderName(), l.SenderUsername(), merchantID); err != nil {
		return l, nil
	}

	go func() {
		otherLeads, err := uc.leads.FindBySender(context.Background(), tenantID, l.SenderID(), 100, 0)
		if err == nil {
			for _, ol := range otherLeads {
				if ol.ID() != l.ID() && ol.MerchantID() != merchantID {
					ol.SetMerchant(merchantID)
					_ = uc.leads.Update(context.Background(), ol)
				}
			}
		}
	}()

	return l, nil
}

func (uc *UseCase) SetContactMerchant(ctx context.Context, tenantID string, senderID int64, name, username, merchantID string) error {
	if err := uc.bindSenderToMerchant(ctx, tenantID, senderID, name, username, merchantID); err != nil {
		return err
	}

	go func() {
		otherLeads, err := uc.leads.FindBySender(context.Background(), tenantID, senderID, 100, 0)
		if err == nil {
			for _, ol := range otherLeads {
				if ol.MerchantID() != merchantID {
					ol.SetMerchant(merchantID)
					_ = uc.leads.Update(context.Background(), ol)
				}
			}
		}
	}()

	return nil
}

func (uc *UseCase) bindSenderToMerchant(ctx context.Context, tenantID string, senderID int64, name, username, merchantID string) error {
	c, err := uc.contacts.FindBySenderID(ctx, tenantID, senderID)
	if err != nil {
		return err
	}

	if c == nil {
		c = contact.New(tenantID, senderID, name, username)
		c.SetMerchant(merchantID)
		return uc.contacts.Save(ctx, c)
	}

	c.UpdateInfo(name, username)
	c.SetMerchant(merchantID)
	return uc.contacts.Update(ctx, c)
}

func (uc *UseCase) Delete(ctx context.Context, tenantID, id string) error {
	l, err := uc.leads.FindByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if l == nil {
		return lead.ErrLeadNotFound
	}
	return uc.leads.DeleteByID(ctx, tenantID, id)
}

type Signal struct {
	ID                string    `json:"id"`
	ChatID            int64     `json:"chat_id"`
	ChatTitle         string    `json:"chat_title"`
	Text              string    `json:"text"`
	Score             float64   `json:"score"`
	IsLead            bool      `json:"is_lead"`
	SemanticDirection string    `json:"semantic_direction"`
	SemanticCategory  string    `json:"semantic_category"`
	CreatedAt         time.Time `json:"created_at"`
}

type Brief struct {
	Lead         *lead.Lead `json:"lead"`
	Signals      []Signal   `json:"signals"`
	SignalsCount int        `json:"signals_count"`
	LastSeenAt   time.Time  `json:"last_seen_at"`
}

func (uc *UseCase) GetBrief(ctx context.Context, tenantID, id string) (*Brief, error) {
	l, err := uc.leads.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, lead.ErrLeadNotFound
	}

	senderLeads, err := uc.leads.FindBySender(ctx, tenantID, l.SenderID(), 100, 0)
	if err != nil {
		return nil, err
	}

	leadByMessageID := make(map[string]*lead.Lead, len(senderLeads))
	for _, sl := range senderLeads {
		leadByMessageID[sl.MessageID()] = sl
	}

	senderMsgs, err := uc.messages.FindBySender(ctx, tenantID, l.SenderID(), 100, 0)
	if err != nil {
		return nil, err
	}
	signals := make([]Signal, 0, len(senderMsgs))
	lastSeen := l.CreatedAt()

	for _, m := range senderMsgs {
		msgID := m.ID()
		linkedLead, isLead := leadByMessageID[msgID]
		score := 0.0
		semanticDirection := ""
		semanticCategory := ""
		if isLead {
			score = linkedLead.Score()
			semanticDirection = linkedLead.SemanticDirection()
			semanticCategory = linkedLead.SemanticCategory()
		} else if m.SimilarityScore() != nil {
			score = *m.SimilarityScore()
		}
		if semanticDirection == "" && m.SemanticDirection() != nil {
			semanticDirection = *m.SemanticDirection()
		}
		if semanticCategory == "" {
			semanticCategory = directionToCategory(semanticDirection)
			if semanticCategory == "" {
				semanticCategory = "stream"
			}
		}

		signals = append(signals, Signal{
			ID:                msgID,
			ChatID:            m.ChatID(),
			ChatTitle:         m.ChatTitle(),
			Text:              m.Text(),
			Score:             score,
			IsLead:            isLead,
			SemanticDirection: semanticDirection,
			SemanticCategory:  semanticCategory,
			CreatedAt:         m.CreatedAt(),
		})
		if m.CreatedAt().After(lastSeen) {
			lastSeen = m.CreatedAt()
		}
	}

	sort.SliceStable(signals, func(i, j int) bool {
		return signals[i].CreatedAt.After(signals[j].CreatedAt)
	})

	total := len(signals)
	if total > 20 {
		signals = signals[:20]
	}

	return &Brief{
		Lead:         l,
		Signals:      signals,
		SignalsCount: total,
		LastSeenAt:   lastSeen,
	}, nil
}

func directionToCategory(direction string) string {
	switch {
	case direction == "":
		return ""
	case containsAnyFold(direction, "trader_search", "search_trader", "ищу трейдера", "поиск трейдера"):
		return "trader_search"
	case containsAnyFold(direction, "merchant", "merch", "мерч"):
		return "merchants"
	case containsAnyFold(direction, "trader", "трейдер", "buyer", "арбитраж"):
		return "traders"
	case containsAnyFold(direction, "processing", "request", "процесс", "запрос"):
		return "merchants"
	case containsAnyFold(direction, "offer", "provider", "предлож", "ps"):
		return "ps_offers"
	case containsAnyFold(direction, "noise", "spam", "шум"):
		return "noise"
	default:
		return ""
	}
}

func containsAnyFold(s string, parts ...string) bool {
	ls := strings.ToLower(s)
	for _, p := range parts {
		if strings.Contains(ls, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

func (uc *UseCase) UpdateCategory(ctx context.Context, tenantID, id, category string) (*lead.Lead, error) {
	l, err := uc.leads.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, lead.ErrLeadNotFound
	}
	l.SetSemanticCategory(category)
	if err := uc.leads.Update(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (uc *UseCase) GetStats(ctx context.Context, tenantID string, days int) (*lead.Stats, error) {
	return uc.leads.GetStats(ctx, tenantID, days)
}
