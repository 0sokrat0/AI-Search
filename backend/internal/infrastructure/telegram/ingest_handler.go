package telegram

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"MRG/internal/domain/contact"
	"MRG/internal/domain/lead"
	"MRG/internal/domain/message"
	"MRG/internal/domain/user"
	"MRG/internal/infrastructure/storage/settings"
	"MRG/internal/infrastructure/textnorm"

	"go.uber.org/zap"
)

type IngestHandler struct {
	tenantID    string
	log         *zap.Logger
	repo        message.Repository
	userRepo    user.Repository
	leadRepo    lead.Repository
	contactRepo contact.Repository
	sieve       lead.Sieve
	settings    *settings.Store

	cleanupMu     sync.Mutex
	lastCleanupAt time.Time
}

type sieveMetaDetector interface {
	DetectLeadWithMeta(ctx context.Context, text string, senderKey string) (isLead bool, score float32, semanticDirection string, err error)
}

func NewIngestHandler(
	tenantID string,
	log *zap.Logger,
	repo message.Repository,
	userRepo user.Repository,
	leadRepo lead.Repository,
	contactRepo contact.Repository,
	sieve lead.Sieve,
	settings *settings.Store,
) *IngestHandler {
	return &IngestHandler{
		tenantID:    tenantID,
		log:         log,
		repo:        repo,
		userRepo:    userRepo,
		leadRepo:    leadRepo,
		contactRepo: contactRepo,
		sieve:       sieve,
		settings:    settings,
	}
}

func (h *IngestHandler) Handle(ctx context.Context, msgs []PendingMsg) error {
	h.maybeCleanupOldNoise(ctx)

	ignoreKeywords := ""
	if h.settings != nil {
		ignoreKeywords = h.settings.GetString(ctx, "ignore_keywords", "")
	}
	keywords := strings.Split(ignoreKeywords, ",")
	var activeKeywords []string
	for _, k := range keywords {
		if s := strings.TrimSpace(k); s != "" {
			activeKeywords = append(activeKeywords, strings.ToLower(s))
		}
	}

	for _, m := range msgs {
		text := strings.TrimSpace(m.Text)
		if text == "" {
			continue
		}
		if strings.TrimSpace(m.SenderUsername) == "" && m.IsDM {
			h.log.Debug("message skipped: sender has no direct username",
				zap.Int64("sender_id", m.SenderID),
				zap.Int64("peer_id", m.PeerID),
				zap.Int64("message_id", m.MessageID),
			)
			continue
		}

		if h.isTeamMember(ctx, m.SenderID) {
			continue
		}

		if h.isSpamSender(ctx, m.SenderID) {
			h.log.Debug("message skipped: sender is spam", zap.Int64("sender_id", m.SenderID))
			continue
		}

		// Keyword filtering
		lowerText := strings.ToLower(text)
		ignoredByKeyword := false
		for _, k := range activeKeywords {
			if strings.Contains(lowerText, k) {
				ignoredByKeyword = true
				break
			}
		}
		if ignoredByKeyword {
			h.log.Debug("message ignored by keyword", zap.Int64("msg_id", m.MessageID), zap.String("text", text))
			continue
		}

		var similarityScore *float64
		var classifiedAsLead *bool
		var semanticDirection *string

		classifyText := textnorm.ForEmbedding(text)
		if classifyText == "" {
			classifyText = text
		}

		if h.sieve != nil {
			var (
				isLead bool
				score  float32
				dir    string
				err    error
			)
			if sieveWithMeta, ok := h.sieve.(sieveMetaDetector); ok {
				isLead, score, dir, err = sieveWithMeta.DetectLeadWithMeta(ctx, classifyText, senderKey(m))
			} else {
				isLead, score, err = h.sieve.DetectLead(ctx, classifyText, senderKey(m))
			}

			if err == nil {
				score64 := float64(score)
				similarityScore = &score64
				classifiedAsLead = &isLead
				if strings.TrimSpace(dir) != "" {
					d := strings.TrimSpace(dir)
					semanticDirection = &d
				}
				h.log.Debug("telegram message classified",
					zap.Int64("sender_id", m.SenderID),
					zap.Int64("peer_id", m.PeerID),
					zap.Int64("message_id", m.MessageID),
					zap.Bool("is_lead", isLead),
					zap.Float64("score", score64),
					zap.String("direction", strings.TrimSpace(dir)),
				)
			} else {
				h.log.Debug("telegram message classification failed",
					zap.Error(err),
					zap.Int64("sender_id", m.SenderID),
					zap.Int64("peer_id", m.PeerID),
					zap.Int64("message_id", m.MessageID),
				)
			}
		}

		storeText := classifyText
		msg, err := message.New(
			h.tenantID,
			m.PeerID,
			m.ChatTitle,
			m.MessageID,
			m.SenderID,
			m.SenderName,
			m.SenderUsername,
			storeText,
			message.MediaTypeNone,
		)
		if err != nil {
			h.log.Debug("skip invalid telegram message",
				zap.Error(err),
				zap.Int64("sender_id", m.SenderID),
				zap.Int64("peer_id", m.PeerID),
				zap.Int64("message_id", m.MessageID),
			)
			continue
		}

		msg.SetSenderTrust(m.IsScam, m.IsFake, m.IsPremium)
		msg.SetChatPeerType(m.ChatPeerType)
		msg.SetIsDM(m.IsDM)

		if similarityScore != nil && classifiedAsLead != nil {
			msg.SetClassificationWithDirection(*similarityScore, *classifiedAsLead, "")
			if semanticDirection != nil {
				msg.SetClassificationWithDirection(*similarityScore, *classifiedAsLead, *semanticDirection)
			}
		}

		if err := h.repo.Save(ctx, msg); err != nil {
			h.log.Warn("save message failed",
				zap.Error(err),
				zap.Int64("msg_id", m.MessageID),
				zap.Int64("sender_id", m.SenderID),
				zap.Int64("peer_id", m.PeerID),
			)
			continue
		}

		if classifiedAsLead != nil && *classifiedAsLead {
			l, err := lead.Detect(h.tenantID, msg.ID(), msg.ChatID(), msg.ChatTitle(), msg.SenderID(), msg.SenderName(), msg.SenderUsername(), msg.Text(), *similarityScore)
			if err == nil {

				merchantID := ""
				if h.contactRepo != nil {
					if c, err := h.contactRepo.FindBySenderID(ctx, h.tenantID, msg.SenderID()); err == nil && c != nil && c.MerchantID() != "" {
						merchantID = c.MerchantID()
					}
				}
				if merchantID != "" {
					l.SetMerchant(merchantID)
				}
				l.MarkAsAIQualified()

				if semanticDirection != nil {
					l.SetSemanticDirection(*semanticDirection)
					switch normalizeLeadCategory(*semanticDirection) {
					case "trader_search", "merchants", "traders", "ps_offers":
						l.SetSemanticCategory(normalizeLeadCategory(*semanticDirection))
					}
				}

				if err := h.leadRepo.Save(ctx, l); err != nil {
					h.log.Warn("save lead failed",
						zap.Error(err),
						zap.Int64("sender_id", m.SenderID),
						zap.Int64("peer_id", m.PeerID),
						zap.Int64("message_id", m.MessageID),
					)
				}
			}
		}
	}

	return nil
}

func (h *IngestHandler) maybeCleanupOldNoise(ctx context.Context) {
	const (
		cleanupEvery = 30 * time.Minute
		noiseTTL     = 72 * time.Hour
	)
	now := time.Now()
	if h.settings != nil && !h.settings.GetBool(ctx, "noise_cleanup_enabled", true) {
		return
	}

	h.cleanupMu.Lock()
	if !h.lastCleanupAt.IsZero() && now.Sub(h.lastCleanupAt) < cleanupEvery {
		h.cleanupMu.Unlock()
		return
	}
	h.lastCleanupAt = now
	h.cleanupMu.Unlock()

	deleted, err := h.repo.DeleteNoise(ctx, h.tenantID, noiseTTL, nil)
	if err != nil {
		h.log.Warn("cleanup old noise failed", zap.Error(err))
		return
	}
	if deleted > 0 {
		h.log.Info("cleanup old noise done", zap.Int64("deleted", deleted), zap.Duration("ttl", noiseTTL))
	}
}

func (h *IngestHandler) isTeamMember(ctx context.Context, senderID int64) bool {
	if h.userRepo == nil || senderID == 0 {
		return false
	}
	ok, err := h.userRepo.IsTeamMember(ctx, h.tenantID, senderID)
	if err != nil {
		return false
	}
	return ok
}

func (h *IngestHandler) isSpamSender(ctx context.Context, senderID int64) bool {
	if h.contactRepo == nil || senderID == 0 {
		return false
	}
	c, err := h.contactRepo.FindBySenderID(ctx, h.tenantID, senderID)
	if err != nil || c == nil {
		return false
	}
	return c.IsSpam()
}

func senderKey(m PendingMsg) string {
	if m.SenderID != 0 {
		return strconv.FormatInt(m.SenderID, 10)
	}
	return strconv.FormatInt(m.PeerID, 10)
}

func normalizeLeadCategory(direction string) string {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "merchant", "merchants", "merch":
		return "merchants"
	case "trader_search", "search_trader", "search_traders", "looking_for_trader":
		return "trader_search"
	case "trader", "traders":
		return "traders"
	case "processing_request", "processing_requests", "request_processing":
		return "merchants"
	case "ps_offer", "ps_offers", "offer", "provider":
		return "ps_offers"
	default:
		return ""
	}
}

var _ MessageHandler = (*IngestHandler)(nil)
