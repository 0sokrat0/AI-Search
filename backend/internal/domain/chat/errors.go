package chat

import "errors"

var (
	ErrChatNotFound = errors.New("chat not found")

	ErrInvalidTenantID = errors.New("invalid tenant ID")

	ErrInvalidChatID = errors.New("invalid chat ID")

	ErrInvalidChatTitle = errors.New("invalid chat title")

	ErrDuplicateChat = errors.New("chat already exists")

	ErrChatNotActive = errors.New("chat is not active")
)
