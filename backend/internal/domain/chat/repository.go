package chat

import "context"

type Repository interface {
	Create(ctx context.Context, chat *Chat) error

	Update(ctx context.Context, chat *Chat) error

	Delete(ctx context.Context, id string) error

	FindByID(ctx context.Context, id string) (*Chat, error)

	FindByChatID(ctx context.Context, tenantID string, chatID int64) (*Chat, error)

	FindByTenantID(ctx context.Context, tenantID string, filters ListFilters) ([]*Chat, error)

	FindActive(ctx context.Context, tenantID string) ([]*Chat, error)

	CountByTenantID(ctx context.Context, tenantID string) (int64, error)

	CountActiveByTenantID(ctx context.Context, tenantID string) (int64, error)

	ExistsByChatID(ctx context.Context, tenantID string, chatID int64) (bool, error)
}

type ListFilters struct {
	IsActive *bool
	ChatType *ChatType
	Limit    int
	Offset   int
}
