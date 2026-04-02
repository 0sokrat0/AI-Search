package lead

import "errors"

var (
	ErrLeadNotFound            = errors.New("lead not found")
	ErrInvalidTenantID         = errors.New("invalid tenant ID")
	ErrInvalidMessageID        = errors.New("invalid message ID")
	ErrInvalidStatus           = errors.New("invalid status")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrLeadAlreadyAssigned     = errors.New("lead already assigned")
)
