package signal

import (
	"context"

	"MRG/internal/domain/message"
)

type BindContactInput struct {
	TenantID   string
	MessageID  string
	MerchantID string
}

type InboxQuery struct {
	TenantID     string
	Limit        int
	Offset       int
	Tab          string
	Category     string
	ShowArchived bool
}

type DTO struct {
	ID                     string   `json:"id"`
	ChatTitle              string   `json:"chatTitle"`
	FromName               string   `json:"fromName"`
	Contact                string   `json:"contact"`
	SenderTelegramID       int64    `json:"senderTelegramId"`
	MerchantID             string   `json:"merchantId"`
	Text                   string   `json:"text"`
	Date                   string   `json:"date"`
	LeadID                 *string  `json:"leadId"`
	LeadScore              *float64 `json:"leadScore"`
	SimilarityScore        *float64 `json:"similarityScore"`
	ClassifiedAsLead       *bool    `json:"classifiedAsLead"`
	SemanticDirection      *string  `json:"semanticDirection"`
	SemanticCategory       string   `json:"semanticCategory"`
	ClassificationReason   string   `json:"classificationReason"`
	TraderScore            float64  `json:"traderScore"`
	MerchantScore          float64  `json:"merchantScore"`
	ProcessingRequestScore float64  `json:"processingRequestScore"`
	PSOfferScore           float64  `json:"psOfferScore"`
	NoiseScore             float64  `json:"noiseScore"`
	PrimaryLabel           string   `json:"primaryLabel"`
	PrimaryPercent         int      `json:"primaryPercent"`
	IsIgnored              bool     `json:"isIgnored"`
	IsTeamMember           bool     `json:"isTeamMember"`
	IsSpamSender           bool     `json:"isSpamSender"`
	IsViewed               bool     `json:"isViewed"`
	IsNew                  bool     `json:"isNew"`
	OtherChatsCount        int      `json:"otherChatsCount"`
	SemanticFlags          []string `json:"semanticFlags"`
}

type FeedbackInput struct {
	TenantID      string
	MessageID     string
	IsLead        bool
	Category      string
	SemanticFlags []string
}

type FeedbackResult struct {
	OK            bool     `json:"ok"`
	IsLead        bool     `json:"is_lead"`
	LeadID        *string  `json:"lead_id"`
	SemanticFlags []string `json:"semantic_flags"`
}

type FlagInput struct {
	TenantID  string
	MessageID string
	Field     string
	Value     bool
}

type ServiceAPI interface {
	BindContact(ctx context.Context, in BindContactInput) error
	GetInbox(ctx context.Context, q InboxQuery) ([]DTO, error)
	GetStats(ctx context.Context, tenantID string, days int) (*message.IngestStats, error)
	FeedbackSignal(ctx context.Context, in FeedbackInput) (FeedbackResult, error)
	FlagSignal(ctx context.Context, in FlagInput) error
	GetSenderHistory(ctx context.Context, tenantID, senderID string, limit int) ([]DTO, error)
}
