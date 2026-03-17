package contact

import (
	"time"
)

type Contact struct {
	tenantID       string
	senderID       int64
	senderName     string
	senderUsername string
	merchantID     string
	isTeamMember   bool
	isSpam         bool
	createdAt      time.Time
	updatedAt      time.Time
}

func New(tenantID string, senderID int64, name, username string) *Contact {
	now := time.Now()
	return &Contact{
		tenantID:       tenantID,
		senderID:       senderID,
		senderName:     name,
		senderUsername: username,
		createdAt:      now,
		updatedAt:      now,
	}
}

func Restore(
	tenantID string,
	senderID int64,
	name, username string,
	merchantID string,
	isTeamMember bool,
	isSpam bool,
	createdAt, updatedAt time.Time,
) *Contact {
	return &Contact{
		tenantID:       tenantID,
		senderID:       senderID,
		senderName:     name,
		senderUsername: username,
		merchantID:     merchantID,
		isTeamMember:   isTeamMember,
		isSpam:         isSpam,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}

func (c *Contact) TenantID() string       { return c.tenantID }
func (c *Contact) SenderID() int64        { return c.senderID }
func (c *Contact) SenderName() string     { return c.senderName }
func (c *Contact) SenderUsername() string { return c.senderUsername }
func (c *Contact) MerchantID() string     { return c.merchantID }
func (c *Contact) IsTeamMember() bool     { return c.isTeamMember }
func (c *Contact) IsSpam() bool           { return c.isSpam }
func (c *Contact) CreatedAt() time.Time   { return c.createdAt }
func (c *Contact) UpdatedAt() time.Time   { return c.updatedAt }

func (c *Contact) SetMerchant(merchantID string) {
	c.merchantID = merchantID
	c.updatedAt = time.Now()
}

func (c *Contact) SetTeamMember(isTeam bool) {
	c.isTeamMember = isTeam
	c.updatedAt = time.Now()
}

func (c *Contact) SetSpam(v bool) {
	c.isSpam = v
	c.updatedAt = time.Now()
}

func (c *Contact) UpdateInfo(name, username string) {
	if name != "" {
		c.senderName = name
	}
	if username != "" {
		c.senderUsername = username
	}
	c.updatedAt = time.Now()
}
