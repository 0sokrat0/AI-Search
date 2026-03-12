package user

import "context"

type Repository interface {
	Create(ctx context.Context, user *User) error

	Update(ctx context.Context, user *User) error

	Delete(ctx context.Context, id string) error

	FindByID(ctx context.Context, id string) (*User, error)

	FindByEmail(ctx context.Context, email string) (*User, error)

	FindByTelegramID(ctx context.Context, tenantID string, telegramUserID int64) (*User, error)

	IsTeamMember(ctx context.Context, tenantID string, telegramUserID int64) (bool, error)

	FindByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*User, error)

	CountByTenantID(ctx context.Context, tenantID string) (int64, error)

	ExistsByEmail(ctx context.Context, email string) (bool, error)

	CreateInvite(ctx context.Context, invite *Invite) error

	FindInviteByTokenHash(ctx context.Context, tokenHash string) (*Invite, error)

	UpdateInvite(ctx context.Context, invite *Invite) error
}
