package message

import "errors"

var (
	ErrMessageNotFound  = errors.New("message not found")
	ErrInvalidTenantID  = errors.New("invalid tenant ID")
	ErrInvalidChatID    = errors.New("invalid chat ID")
	ErrInvalidMessageID = errors.New("invalid message ID")
	ErrDuplicateMessage = errors.New("duplicate message")
)
