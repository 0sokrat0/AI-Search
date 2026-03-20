package message

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Metadata struct {
	Geo      string `json:"geo,omitempty"`
	Intent   string `json:"intent,omitempty"`
	Service  string `json:"service,omitempty"`
	Currency string `json:"currency,omitempty"`
}

type MediaType string

const (
	MediaTypeNone MediaType = "none"
)

type Message struct {
	id                string
	tenantID          string
	chatID            int64
	chatTitle         string
	messageID         int64
	senderID          int64
	senderName        string
	senderUsername    string
	chatPeerType      string
	isScam            bool
	isFake            bool
	isPremium         bool
	text              string
	mediaType         MediaType
	createdAt         time.Time
	isIgnored         bool
	isTeamMember      bool
	isSpamSender      bool
	isDM              bool
	isViewed          bool
	viewedAt          *time.Time
	similarityScore   *float64
	classifiedAsLead  *bool
	semanticDirection *string
	metadata          Metadata
}

func New(
	tenantID string,
	chatID int64,
	chatTitle string,
	messageID, senderID int64,
	senderName, senderUsername, text string,
	mediaType MediaType,
) (*Message, error) {
	if tenantID == "" {
		return nil, ErrInvalidTenantID
	}
	if chatID == 0 {
		return nil, ErrInvalidChatID
	}
	if messageID == 0 {
		return nil, ErrInvalidMessageID
	}
	return &Message{
		id:             uuid.New().String(),
		tenantID:       tenantID,
		chatID:         chatID,
		chatTitle:      chatTitle,
		messageID:      messageID,
		senderID:       senderID,
		senderName:     senderName,
		senderUsername: senderUsername,
		text:           text,
		mediaType:      mediaType,
		createdAt:      time.Now(),
	}, nil
}

func Restore(
	id, tenantID string,
	chatID int64,
	chatTitle string,
	messageID, senderID int64,
	senderName, senderUsername, text string,
	mediaType MediaType,
	createdAt time.Time,
	isIgnored, isTeamMember, isSpamSender, isDM, isViewed bool,
	viewedAt *time.Time,
	similarityScore *float64,
	classifiedAsLead *bool,
	semanticDirection *string,
	meta Metadata,
) *Message {
	return &Message{
		id:                id,
		tenantID:          tenantID,
		chatID:            chatID,
		chatTitle:         chatTitle,
		messageID:         messageID,
		senderID:          senderID,
		senderName:        senderName,
		senderUsername:    senderUsername,
		text:              text,
		mediaType:         mediaType,
		createdAt:         createdAt,
		isIgnored:         isIgnored,
		isTeamMember:      isTeamMember,
		isSpamSender:      isSpamSender,
		isDM:              isDM,
		isViewed:          isViewed,
		viewedAt:          viewedAt,
		similarityScore:   similarityScore,
		classifiedAsLead:  classifiedAsLead,
		semanticDirection: semanticDirection,
		metadata:          meta,
	}
}

func (m *Message) ID() string                 { return m.id }
func (m *Message) TenantID() string           { return m.tenantID }
func (m *Message) ChatID() int64              { return m.chatID }
func (m *Message) ChatTitle() string          { return m.chatTitle }
func (m *Message) MessageID() int64           { return m.messageID }
func (m *Message) SenderID() int64            { return m.senderID }
func (m *Message) SenderName() string         { return m.senderName }
func (m *Message) SenderUsername() string     { return m.senderUsername }
func (m *Message) ChatPeerType() string       { return m.chatPeerType }
func (m *Message) IsScam() bool               { return m.isScam }
func (m *Message) IsFake() bool               { return m.isFake }
func (m *Message) IsPremium() bool            { return m.isPremium }
func (m *Message) Text() string               { return m.text }
func (m *Message) MediaType() MediaType       { return m.mediaType }
func (m *Message) CreatedAt() time.Time       { return m.createdAt }
func (m *Message) HasMedia() bool             { return m.mediaType != MediaTypeNone }
func (m *Message) IsIgnored() bool            { return m.isIgnored }
func (m *Message) IsTeamMember() bool         { return m.isTeamMember }
func (m *Message) IsSpamSender() bool         { return m.isSpamSender }
func (m *Message) IsViewed() bool             { return m.isViewed }
func (m *Message) ViewedAt() *time.Time       { return m.viewedAt }
func (m *Message) SimilarityScore() *float64  { return m.similarityScore }
func (m *Message) ClassifiedAsLead() *bool    { return m.classifiedAsLead }
func (m *Message) SemanticDirection() *string { return m.semanticDirection }
func (m *Message) Metadata() Metadata         { return m.metadata }

func (m *Message) IsDM() bool               { return m.isDM }
func (m *Message) SetChatPeerType(v string) { m.chatPeerType = v }
func (m *Message) SetIgnored(v bool)        { m.isIgnored = v }
func (m *Message) SetTeamMember(v bool)     { m.isTeamMember = v }
func (m *Message) SetSpamSender(v bool)     { m.isSpamSender = v }
func (m *Message) SetIsDM(v bool)           { m.isDM = v }
func (m *Message) SetViewed(v bool) {
	m.isViewed = v
	if v {
		now := time.Now().UTC()
		m.viewedAt = &now
		return
	}
	m.viewedAt = nil
}

func (m *Message) SetSenderTrust(scam, fake, premium bool) {
	m.isScam = scam
	m.isFake = fake
	m.isPremium = premium
}

func (m *Message) SetMetadata(meta Metadata) {
	m.metadata = meta
}

func (m *Message) SetClassification(score float64, isLead bool) {
	m.similarityScore = &score
	m.classifiedAsLead = &isLead
}

func (m *Message) SetClassificationWithDirection(score float64, isLead bool, direction string) {
	m.SetClassification(score, isLead)
	direction = strings.TrimSpace(direction)
	if direction == "" {
		m.semanticDirection = nil
		return
	}
	m.semanticDirection = &direction
}

func (m *Message) SenderIdentifier() string {
	if m.senderUsername != "" {
		return "@" + m.senderUsername
	}
	return m.senderName
}
