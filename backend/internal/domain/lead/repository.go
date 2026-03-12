package lead

import (
	"context"
	"time"
)

type MessageLeadRef struct {
	ID    string
	Score float64
}

type Repository interface {
	Save(ctx context.Context, l *Lead) error
	Update(ctx context.Context, l *Lead) error
	DeleteByID(ctx context.Context, tenantID, id string) error
	DeleteByMessageID(ctx context.Context, tenantID, messageID string) error
	FindByID(ctx context.Context, tenantID, id string) (*Lead, error)
	FindByMessageID(ctx context.Context, tenantID, messageID string) (*Lead, error)
	FindByMessageIDs(ctx context.Context, tenantID string, messageIDs []string) (map[string]MessageLeadRef, error)
	FindBySender(ctx context.Context, tenantID string, senderID int64, limit, offset int) ([]*Lead, error)
	List(ctx context.Context, tenantID string, f ListFilter) ([]*Lead, error)
	Count(ctx context.Context, tenantID string, f ListFilter) (int64, error)
	GetStats(ctx context.Context, tenantID string, days int) (*Stats, error)
}

type Stats struct {
	Period           string        `json:"period"`
	TotalDetected    int64         `json:"totalDetected"`
	Approved         int64         `json:"approved"`
	Rejected         int64         `json:"rejected"`
	Pending          int64         `json:"pending"`
	AvgScore         float64       `json:"avgScore"`
	AvgScoreApproved float64       `json:"avgScoreApproved"`
	AvgScoreRejected float64       `json:"avgScoreRejected"`
	Buckets          []ScoreBucket `json:"buckets"`
}

type ScoreBucket struct {
	From     float64 `json:"from"`
	To       float64 `json:"to"`
	Count    int64   `json:"count"`
	Approved int64   `json:"approved"`
	Rejected int64   `json:"rejected"`
}

type ListFilter struct {
	Status            *Status
	MerchantID        *string
	SemanticDirection *string
	ChatID            *int64
	MinScore          *float64
	FromDate          *time.Time
	ToDate            *time.Time
	Reviewed          *bool
	Limit             int
	Offset            int
}
