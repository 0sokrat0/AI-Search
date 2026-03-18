package lead

import (
	"time"

	"github.com/google/uuid"
)

type Lead struct {
	id                  string
	tenantID            string
	messageID           string
	chatID              int64
	chatTitle           string
	senderID            int64
	senderName          string
	senderUsername      string
	text                string
	geo                 []string
	products            []string
	semanticDirection   string
	semanticCategory    string
	merchantID          string
	status              Status
	qualificationSource QualificationSource
	score               float64
	userFeedback        *bool
	categoryAssignedAt  *time.Time
	createdAt           time.Time
	updatedAt           time.Time
}

func Detect(
	tenantID, messageID string,
	chatID int64, chatTitle string,
	senderID int64, senderName, senderUsername, text string,
	score float64,
) (*Lead, error) {
	if tenantID == "" {
		return nil, ErrInvalidTenantID
	}
	if messageID == "" {
		return nil, ErrInvalidMessageID
	}
	now := time.Now()
	return &Lead{
		id:               uuid.New().String(),
		tenantID:         tenantID,
		messageID:        messageID,
		chatID:           chatID,
		chatTitle:        chatTitle,
		senderID:         senderID,
		senderName:       senderName,
		senderUsername:   senderUsername,
		text:             text,
		geo:              []string{},
		products:         []string{},
		semanticCategory: "leads",
		status:           StatusDetected,
		score:            score,
		createdAt:        now,
		updatedAt:        now,
	}, nil
}

func Restore(
	id, tenantID, messageID string,
	chatID int64, chatTitle string,
	senderID int64, senderName, senderUsername, text string,
	geo, products []string,
	semanticDirection string,
	semanticCategory string,
	merchantID string,
	status Status,
	qualificationSource QualificationSource,
	score float64,
	userFeedback *bool,
	categoryAssignedAt *time.Time,
	createdAt, updatedAt time.Time,
) *Lead {
	return &Lead{
		id:                  id,
		tenantID:            tenantID,
		messageID:           messageID,
		chatID:              chatID,
		chatTitle:           chatTitle,
		senderID:            senderID,
		senderName:          senderName,
		senderUsername:      senderUsername,
		text:                text,
		geo:                 ensureSlice(geo),
		products:            ensureSlice(products),
		semanticDirection:   semanticDirection,
		semanticCategory:    semanticCategory,
		merchantID:          merchantID,
		status:              status,
		qualificationSource: qualificationSource,
		score:               score,
		userFeedback:        userFeedback,
		categoryAssignedAt:  categoryAssignedAt,
		createdAt:           createdAt,
		updatedAt:           updatedAt,
	}
}

func (l *Lead) ID() string                               { return l.id }
func (l *Lead) TenantID() string                         { return l.tenantID }
func (l *Lead) MessageID() string                        { return l.messageID }
func (l *Lead) ChatID() int64                            { return l.chatID }
func (l *Lead) ChatTitle() string                        { return l.chatTitle }
func (l *Lead) SenderID() int64                          { return l.senderID }
func (l *Lead) SenderName() string                       { return l.senderName }
func (l *Lead) SenderUsername() string                   { return l.senderUsername }
func (l *Lead) Text() string                             { return l.text }
func (l *Lead) Geo() []string                            { return copySlice(l.geo) }
func (l *Lead) Products() []string                       { return copySlice(l.products) }
func (l *Lead) SemanticDirection() string                { return l.semanticDirection }
func (l *Lead) SemanticCategory() string                 { return l.semanticCategory }
func (l *Lead) MerchantID() string                       { return l.merchantID }
func (l *Lead) Status() Status                           { return l.status }
func (l *Lead) QualificationSource() QualificationSource { return l.qualificationSource }
func (l *Lead) Score() float64                           { return l.score }
func (l *Lead) UserFeedback() *bool                      { return l.userFeedback }
func (l *Lead) CategoryAssignedAt() *time.Time           { return l.categoryAssignedAt }
func (l *Lead) CreatedAt() time.Time                     { return l.createdAt }
func (l *Lead) UpdatedAt() time.Time                     { return l.updatedAt }
func (l *Lead) Priority() Priority                       { return PriorityFromScore(l.score) }

func (l *Lead) SenderIdentifier() string {
	if l.senderUsername != "" {
		return "@" + l.senderUsername
	}
	return l.senderName
}

func (l *Lead) Tag(geo, products []string) {
	l.geo = ensureSlice(geo)
	l.products = ensureSlice(products)
	l.updatedAt = time.Now()
}

func (l *Lead) SetMerchant(merchantID string) {
	l.merchantID = merchantID
	l.updatedAt = time.Now()
}

func (l *Lead) SetSemanticDirection(direction string) {
	l.semanticDirection = direction
	l.updatedAt = time.Now()
}

func (l *Lead) SetSemanticCategory(category string) {
	l.semanticCategory = category
	now := time.Now()
	l.categoryAssignedAt = &now
	l.updatedAt = now
}

func (l *Lead) Advance(to Status) error {
	if err := validateTransition(l.status, to); err != nil {
		return err
	}
	l.status = to
	l.updatedAt = time.Now()
	return nil
}

func (l *Lead) SetStatus(to Status) error {
	if !IsValidStatus(to) {
		return ErrInvalidStatus
	}
	l.status = to
	l.updatedAt = time.Now()
	return nil
}

func (l *Lead) Approve() {
	l.setFeedback(true)
	l.qualificationSource = QualificationSourceManual
	l.status = StatusConfirmed
}

func (l *Lead) Reject() {
	l.setFeedback(false)
	l.qualificationSource = QualificationSourceNone
	l.status = StatusFalsePositive
}

func (l *Lead) MarkAsConfirmed() {
	l.setFeedback(true)
	l.qualificationSource = QualificationSourceManual
	l.status = StatusConfirmed
}

func (l *Lead) MarkAsControversial() {
	l.userFeedback = nil
	l.status = StatusControversial
	l.updatedAt = time.Now()
}

func (l *Lead) MarkAsFalsePositive() {
	l.setFeedback(false)
	l.qualificationSource = QualificationSourceNone
	l.status = StatusFalsePositive
}

func (l *Lead) MarkAsAIQualified() {
	l.userFeedback = nil
	l.qualificationSource = QualificationSourceAI
	l.updatedAt = time.Now()
}

func (l *Lead) setFeedback(good bool) {
	l.userFeedback = &good
	l.updatedAt = time.Now()
}

func (l *Lead) IsReviewed() bool { return l.userFeedback != nil }
func (l *Lead) IsApproved() bool { return l.userFeedback != nil && *l.userFeedback }
