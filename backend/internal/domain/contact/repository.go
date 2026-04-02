package contact

import "context"

type Repository interface {
	Save(ctx context.Context, c *Contact) error
	Update(ctx context.Context, c *Contact) error
	FindBySenderID(ctx context.Context, tenantID string, senderID int64) (*Contact, error)
	ClaimOwnership(ctx context.Context, tenantID string, senderID int64, ownerID, ownerName string) (*Contact, error)
}
