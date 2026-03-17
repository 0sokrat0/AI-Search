package signal

import "errors"

var (
	ErrSignalNotFound   = errors.New("signal not found")
	ErrSieveUnavailable = errors.New("vector sieve not configured")
	ErrEmptySignalText  = errors.New("signal has no text")
	ErrInvalidFlagField = errors.New("field must be is_ignored, is_team_member, is_viewed or is_spam_sender")
	ErrInvalidSenderID  = errors.New("invalid sender id")
)
