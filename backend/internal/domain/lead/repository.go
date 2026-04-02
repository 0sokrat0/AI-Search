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
	FindByTextHash(ctx context.Context, tenantID, textHash string) (*Lead, error)
	ClaimOwnership(ctx context.Context, tenantID, id, ownerID, ownerName string) (*Lead, error)
	ReleaseOwnership(ctx context.Context, tenantID, id, ownerID string) (*Lead, error)
	List(ctx context.Context, tenantID string, f ListFilter) ([]*Lead, error)
	ListPage(ctx context.Context, tenantID string, f ListFilter) (*ListPage, error)
	Count(ctx context.Context, tenantID string, f ListFilter) (int64, error)
	GetStats(ctx context.Context, tenantID string, days int) (*Stats, error)
}

type Stats struct {
	Period             string                 `json:"period"`
	TotalDetected      int64                  `json:"totalDetected"`
	Approved           int64                  `json:"approved"`
	Rejected           int64                  `json:"rejected"`
	Pending            int64                  `json:"pending"`
	AIQualified        int64                  `json:"aiQualified"`
	ManualApproved     int64                  `json:"manualApproved"`
	AvgScore           float64                `json:"avgScore"`
	AvgScoreApproved   float64                `json:"avgScoreApproved"`
	AvgScoreRejected   float64                `json:"avgScoreRejected"`
	Buckets            []ScoreBucket          `json:"buckets"`
	DetectedByCategory CategoryDistribution   `json:"detectedByCategory"`
	ApprovedByCategory CategoryDistribution   `json:"approvedByCategory"`
	RejectedByCategory CategoryDistribution   `json:"rejectedByCategory"`
	Series             []CategorySeriesBucket `json:"series"`
}

type ScoreBucket struct {
	From     float64 `json:"from"`
	To       float64 `json:"to"`
	Count    int64   `json:"count"`
	Approved int64   `json:"approved"`
	Rejected int64   `json:"rejected"`
}

type CategoryDistribution struct {
	TraderSearch int64 `json:"traderSearch"`
	Traders      int64 `json:"traders"`
	Merchants    int64 `json:"merchants"`
	PSOffers     int64 `json:"psOffers"`
	Other        int64 `json:"other"`
}

type CategorySeriesBucket struct {
	Day          string `json:"day"`
	TraderSearch int64  `json:"traderSearch"`
	Traders      int64  `json:"traders"`
	Merchants    int64  `json:"merchants"`
	PSOffers     int64  `json:"psOffers"`
	Other        int64  `json:"other"`
}

type ListFilter struct {
	Status            *Status
	MerchantID        *string
	SemanticCategory  *string
	SemanticDirection *string
	ChatID            *int64
	MinScore          *float64
	FromDate          *time.Time
	ToDate            *time.Time
	Reviewed          *bool
	QualifiedOnly     bool
	Limit             int
	Offset            int
	Cursor            string
}

type ListPage struct {
	Items      []*Lead `json:"items"`
	NextCursor string  `json:"nextCursor"`
}
