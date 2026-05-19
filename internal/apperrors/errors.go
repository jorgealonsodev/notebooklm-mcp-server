// Package apperrors defines sentinel error types for the notebooklm-mcp-server.
package apperrors

import "errors"

const defaultRateLimitMsg = "NotebookLM rate limit reached (50 queries/day for free accounts)"

// RateLimitError is returned when NotebookLM's daily query quota is exhausted.
// Free accounts are limited to 50 queries/day.
type RateLimitError struct {
	msg string
}

// NewRateLimitError constructs a RateLimitError. Pass an empty string to use
// the canonical default message.
func NewRateLimitError(msg string) error {
	if msg == "" {
		msg = defaultRateLimitMsg
	}
	return &RateLimitError{msg: msg}
}

func (e *RateLimitError) Error() string { return e.msg }

// AuthenticationError is returned when Google authentication fails.
// SuggestCleanup hints that the caller should run the cleanup workflow
// (useful when upgrading from an old installation).
type AuthenticationError struct {
	msg            string
	SuggestCleanup bool
}

// NewAuthenticationError constructs an AuthenticationError.
func NewAuthenticationError(msg string, suggestCleanup bool) error {
	return &AuthenticationError{msg: msg, SuggestCleanup: suggestCleanup}
}

func (e *AuthenticationError) Error() string { return e.msg }

// IsRateLimit reports whether err (or any error in its chain) is a
// *RateLimitError.
func IsRateLimit(err error) bool {
	var rle *RateLimitError
	return errors.As(err, &rle)
}
