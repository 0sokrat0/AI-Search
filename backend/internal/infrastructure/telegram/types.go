package telegram

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageHandler interface {
	Handle(ctx context.Context, msgs []PendingMsg) error
}

type PendingMsg struct {
	MessageID      int64
	SenderID       int64
	SenderName     string
	SenderUsername string
	SenderPeerType string
	ChatPeerType   string
	IsScam         bool
	IsFake         bool
	IsPremium      bool
	IsDM           bool
	Text           string
	ChatTitle      string
	PeerID         int64
	Date           int
}

type AccountStatus string

const (
	StatusActive       AccountStatus = "active"
	StatusAuthorized   AccountStatus = "authorized"
	StatusStarting     AccountStatus = "starting"
	StatusUnauthorized AccountStatus = "unauthorized"
	StatusAuthPending  AccountStatus = "auth_pending"
	StatusDisabled     AccountStatus = "disabled"
)

type AccountConfig struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Phone              string             `bson:"phone" json:"phone"`
	SessionData        []byte             `bson:"session_data,omitempty" json:"-"`
	Proxy              string             `bson:"proxy" json:"proxy"`
	ProxyFallback      string             `bson:"proxy_fallback,omitempty" json:"proxy_fallback,omitempty"`
	Status             AccountStatus      `bson:"status" json:"status"`
	QRUrl              string             `bson:"qr_url,omitempty" json:"qr_url,omitempty"`
	WaitingForPassword bool               `bson:"waiting_for_password,omitempty" json:"waiting_for_password,omitempty"`
	Name               string             `bson:"name" json:"name"`
	Username           string             `bson:"username" json:"username"`
	AvatarURL          string             `bson:"avatar_url" json:"avatar_url"`
	CreatedAt          time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt          time.Time          `bson:"updated_at" json:"updated_at"`
}
