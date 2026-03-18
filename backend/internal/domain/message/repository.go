package message

import (
	"context"
	"time"
)

type Repository interface {
	Save(ctx context.Context, m *Message) error
	FindByID(ctx context.Context, tenantID, id string) (*Message, error)
	FindByTelegramID(ctx context.Context, tenantID string, chatID, messageID int64) (*Message, error)
	FindByChat(ctx context.Context, tenantID string, chatID int64, limit, offset int) ([]*Message, error)
	FindBySender(ctx context.Context, tenantID string, senderID int64, limit, offset int) ([]*Message, error)
	FindUnclassified(ctx context.Context, tenantID string, limit int) ([]*Message, error)
	List(ctx context.Context, tenantID string, f ListFilter) ([]*Message, error)
	ListPage(ctx context.Context, tenantID string, f ListFilter) (*ListPage, error)
	CountByTenantToday(ctx context.Context, tenantID string) (int64, error)
	ExistsByTelegramID(ctx context.Context, tenantID string, chatID, messageID int64) (bool, error)
	SetFlag(ctx context.Context, tenantID, id, field string, value bool) error
	SetClassification(ctx context.Context, tenantID, id string, isLead bool) error
	SetSemanticDirection(ctx context.Context, tenantID, id string, direction *string) error
	CountSenderInChats(ctx context.Context, tenantID string, senderIDs []int64) (map[int64]int, error)
	DeleteNoise(ctx context.Context, tenantID string, olderThan time.Duration, ids []string) (int64, error)
	GetIngestStats(ctx context.Context, tenantID string, days int) (*IngestStats, error)
	GetChartData(ctx context.Context, tenantID string, from, to time.Time) ([]ChartDayBucket, error)
}

type ListFilter struct {
	ChatID   *int64
	SenderID *int64
	FromDate *time.Time
	ToDate   *time.Time
	Limit    int
	Offset   int
	Cursor   string
}

type ListPage struct {
	Items      []*Message `json:"items"`
	NextCursor string     `json:"nextCursor"`
}

type IngestStats struct {
	Period          string         `json:"period"`
	TotalSignals    int64          `json:"totalSignals"`
	SignalsToday    int64          `json:"signalsToday"`
	SignalsLastHour int64          `json:"signalsLastHour"`
	AvgPerHour      float64        `json:"avgPerHour"`
	UniqueChats     int64          `json:"uniqueChats"`
	UniqueSenders   int64          `json:"uniqueSenders"`
	LeadCandidates  int64          `json:"leadCandidates"`
	TeamMessages    int64          `json:"teamMessages"`
	IgnoredMessages int64          `json:"ignoredMessages"`
	LastSignalAt    *time.Time     `json:"lastSignalAt"`
	Hourly          []HourlyBucket `json:"hourly"`
	TopChats        []ChatBucket   `json:"topChats"`
}

type HourlyBucket struct {
	Hour  string `json:"hour"`
	Count int64  `json:"count"`
}

type ChatBucket struct {
	ChatID    int64  `json:"chatId"`
	ChatTitle string `json:"chatTitle"`
	Count     int64  `json:"count"`
}

type ChartDayBucket struct {
	Day       string `json:"day"`
	Total     int64  `json:"total"`
	Target    int64  `json:"target"`
	Traders   int64  `json:"traders"`
	Merchants int64  `json:"merchants"`
	PSOffers  int64  `json:"psOffers"`
}
