package settings

import "errors"

var (
	ErrInvalidThreshold = errors.New("threshold must be a float between 0 and 1")
	ErrInvalidWindow    = errors.New("sender_window_seconds must be between 5 and 3600")
)
