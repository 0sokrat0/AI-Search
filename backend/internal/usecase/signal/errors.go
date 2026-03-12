package signal

import "errors"

var (
	ErrSignalNotFound   = errors.New("signal not found")
	ErrSieveUnavailable = errors.New("vector sieve not configured")
	ErrEmptySignalText  = errors.New("signal has no text")
	ErrInvalidFlagField = errors.New("field must be is_ignored, is_team_member or is_viewed")
	ErrInvalidSenderID  = errors.New("invalid sender id")
)
