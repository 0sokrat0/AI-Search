package user

import "time"

type Invite struct {
	id        string
	tenantID  string
	role      Role
	tokenHash string
	createdBy string
	createdAt time.Time
	expiresAt time.Time
	usedAt    *time.Time
	usedBy    string
}

func NewInvite(id, tenantID string, role Role, tokenHash, createdBy string, expiresAt time.Time) *Invite {
	now := time.Now().UTC()
	return &Invite{
		id:        id,
		tenantID:  tenantID,
		role:      NormalizeRole(role),
		tokenHash: tokenHash,
		createdBy: createdBy,
		createdAt: now,
		expiresAt: expiresAt.UTC(),
	}
}

func RestoreInvite(id, tenantID string, role Role, tokenHash, createdBy string, createdAt, expiresAt time.Time, usedAt *time.Time, usedBy string) *Invite {
	return &Invite{
		id:        id,
		tenantID:  tenantID,
		role:      NormalizeRole(role),
		tokenHash: tokenHash,
		createdBy: createdBy,
		createdAt: createdAt,
		expiresAt: expiresAt,
		usedAt:    usedAt,
		usedBy:    usedBy,
	}
}

func (i *Invite) ID() string           { return i.id }
func (i *Invite) TenantID() string     { return i.tenantID }
func (i *Invite) Role() Role           { return i.role }
func (i *Invite) TokenHash() string    { return i.tokenHash }
func (i *Invite) CreatedBy() string    { return i.createdBy }
func (i *Invite) CreatedAt() time.Time { return i.createdAt }
func (i *Invite) ExpiresAt() time.Time { return i.expiresAt }
func (i *Invite) UsedAt() *time.Time   { return i.usedAt }
func (i *Invite) UsedBy() string       { return i.usedBy }

func (i *Invite) IsExpired(now time.Time) bool {
	return !i.expiresAt.IsZero() && now.After(i.expiresAt)
}

func (i *Invite) IsUsed() bool {
	return i.usedAt != nil
}

func (i *Invite) MarkUsed(userID string, at time.Time) {
	when := at.UTC()
	i.usedAt = &when
	i.usedBy = userID
}
