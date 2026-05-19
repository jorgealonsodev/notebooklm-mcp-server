package apperrors_test

import (
	"errors"
	"testing"

	"github.com/jorge/notebooklm-mcp-server/internal/apperrors"
)

func TestRateLimitError_IsError(t *testing.T) {
	err := apperrors.NewRateLimitError("")
	if err == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestRateLimitError_DefaultMessage(t *testing.T) {
	err := apperrors.NewRateLimitError("")
	want := "NotebookLM rate limit reached (50 queries/day for free accounts)"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}
}

func TestRateLimitError_CustomMessage(t *testing.T) {
	err := apperrors.NewRateLimitError("custom message")
	if err.Error() != "custom message" {
		t.Errorf("got %q, want %q", err.Error(), "custom message")
	}
}

func TestRateLimitError_ErrorsAs(t *testing.T) {
	err := apperrors.NewRateLimitError("")
	var rle *apperrors.RateLimitError
	if !errors.As(err, &rle) {
		t.Error("errors.As should match *RateLimitError")
	}
}

func TestAuthenticationError_IsError(t *testing.T) {
	err := apperrors.NewAuthenticationError("bad creds", false)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
}

func TestAuthenticationError_Message(t *testing.T) {
	err := apperrors.NewAuthenticationError("bad creds", false)
	if err.Error() != "bad creds" {
		t.Errorf("got %q, want %q", err.Error(), "bad creds")
	}
}

func TestAuthenticationError_SuggestCleanupFalse(t *testing.T) {
	err := apperrors.NewAuthenticationError("x", false)
	var ae *apperrors.AuthenticationError
	if !errors.As(err, &ae) {
		t.Fatal("errors.As should match *AuthenticationError")
	}
	if ae.SuggestCleanup {
		t.Error("SuggestCleanup should be false")
	}
}

func TestAuthenticationError_SuggestCleanupTrue(t *testing.T) {
	err := apperrors.NewAuthenticationError("x", true)
	var ae *apperrors.AuthenticationError
	if !errors.As(err, &ae) {
		t.Fatal("errors.As should match *AuthenticationError")
	}
	if !ae.SuggestCleanup {
		t.Error("SuggestCleanup should be true")
	}
}

func TestIsRateLimit(t *testing.T) {
	if !apperrors.IsRateLimit(apperrors.NewRateLimitError("")) {
		t.Error("IsRateLimit should return true for RateLimitError")
	}
	if apperrors.IsRateLimit(apperrors.NewAuthenticationError("x", false)) {
		t.Error("IsRateLimit should return false for AuthenticationError")
	}
	if apperrors.IsRateLimit(nil) {
		t.Error("IsRateLimit should return false for nil")
	}
}
