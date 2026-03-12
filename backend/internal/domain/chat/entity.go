package chat

import (
	"time"

	"github.com/google/uuid"
)

type ChatType string

const (
	ChatTypePrivate    ChatType = "private"
	ChatTypeGroup      ChatType = "group"
	ChatTypeSupergroup ChatType = "supergroup"
	ChatTypeChannel    ChatType = "channel"
)

type Chat struct {
	id              string
	tenantID        string
	chatID          int64
	chatTitle       string
	chatType        ChatType
	isActive        bool
	monitoringSince time.Time
	lastMessageID   *int64
	lastCheckAt     *time.Time
	totalMessages   int64
	totalLeads      int64
	createdAt       time.Time
	updatedAt       time.Time
}

func New(tenantID string, chatID int64, chatTitle string, chatType ChatType) (*Chat, error) {
	if tenantID == "" {
		return nil, ErrInvalidTenantID
	}
	if chatID == 0 {
		return nil, ErrInvalidChatID
	}
	if chatTitle == "" {
		return nil, ErrInvalidChatTitle
	}

	now := time.Now()
	return &Chat{
		id:              uuid.New().String(),
		tenantID:        tenantID,
		chatID:          chatID,
		chatTitle:       chatTitle,
		chatType:        chatType,
		isActive:        true,
		monitoringSince: now,
		totalMessages:   0,
		totalLeads:      0,
		createdAt:       now,
		updatedAt:       now,
	}, nil
}

func Restore(
	id, tenantID string,
	chatID int64,
	chatTitle string,
	chatType ChatType,
	isActive bool,
	monitoringSince time.Time,
	lastMessageID *int64,
	lastCheckAt *time.Time,
	totalMessages, totalLeads int64,
	createdAt, updatedAt time.Time,
) *Chat {
	return &Chat{
		id:              id,
		tenantID:        tenantID,
		chatID:          chatID,
		chatTitle:       chatTitle,
		chatType:        chatType,
		isActive:        isActive,
		monitoringSince: monitoringSince,
		lastMessageID:   lastMessageID,
		lastCheckAt:     lastCheckAt,
		totalMessages:   totalMessages,
		totalLeads:      totalLeads,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}

func (c *Chat) ID() string                 { return c.id }
func (c *Chat) TenantID() string           { return c.tenantID }
func (c *Chat) ChatID() int64              { return c.chatID }
func (c *Chat) ChatTitle() string          { return c.chatTitle }
func (c *Chat) ChatType() ChatType         { return c.chatType }
func (c *Chat) IsActive() bool             { return c.isActive }
func (c *Chat) MonitoringSince() time.Time { return c.monitoringSince }
func (c *Chat) LastMessageID() *int64      { return c.lastMessageID }
func (c *Chat) LastCheckAt() *time.Time    { return c.lastCheckAt }
func (c *Chat) TotalMessages() int64       { return c.totalMessages }
func (c *Chat) TotalLeads() int64          { return c.totalLeads }
func (c *Chat) CreatedAt() time.Time       { return c.createdAt }
func (c *Chat) UpdatedAt() time.Time       { return c.updatedAt }

func (c *Chat) UpdateTitle(title string) error {
	if title == "" {
		return ErrInvalidChatTitle
	}
	c.chatTitle = title
	c.updatedAt = time.Now()
	return nil
}

func (c *Chat) RecordCheck(lastMessageID int64) {
	now := time.Now()
	c.lastMessageID = &lastMessageID
	c.lastCheckAt = &now
	c.updatedAt = now
}

func (c *Chat) IncrementMessageCount() {
	c.totalMessages++
	c.updatedAt = time.Now()
}

func (c *Chat) IncrementLeadCount() {
	c.totalLeads++
	c.updatedAt = time.Now()
}

func (c *Chat) Activate() {
	if !c.isActive {
		c.isActive = true
		c.monitoringSince = time.Now()
		c.updatedAt = time.Now()
	}
}

func (c *Chat) Deactivate() {
	c.isActive = false
	c.updatedAt = time.Now()
}

func (c *Chat) GetLeadConversionRate() float64 {
	if c.totalMessages == 0 {
		return 0.0
	}
	return float64(c.totalLeads) / float64(c.totalMessages) * 100
}

func (c *Chat) HasNewMessages() bool {
	if c.lastCheckAt == nil {
		return true
	}
	return time.Since(*c.lastCheckAt) > 5*time.Minute
}

func (c *Chat) IsGroup() bool {
	return c.chatType == ChatTypeGroup || c.chatType == ChatTypeSupergroup
}

func (c *Chat) IsChannel() bool {
	return c.chatType == ChatTypeChannel
}

func (c *Chat) GetMonitoringDuration() time.Duration {
	return time.Since(c.monitoringSince)
}
