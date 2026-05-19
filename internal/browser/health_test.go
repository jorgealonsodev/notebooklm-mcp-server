package browser

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsRecoverableError(t *testing.T) {
	tests := []struct {
		name   string
		errMsg string
		want   bool
	}{
		{"target closed", "Target closed", true},
		{"page closed", "Page closed", true},
		{"browser closed", "Browser closed", true},
		{"websocket disconnected", "WebSocket disconnected", true},
		{"target closed lowercase", "target closed", true},
		{"page closed mixed case", "page Closed", true},
		{"normal error", "navigation timeout", false},
		{"empty error", "", false},
		{"selector not found", "selector not found", false},
		{"protocol error", "Protocol error (Runtime.callFunctionOn): Target closed.", true},
		{"page unresponsive", "page is unresponsive", false},
		{"context destroyed", "Browser context has been closed", true},
		{"connection error", "connection refused", false},
		{"timeout", "waiting for selector 'button' timed out", false},
		{"substring in words", "re-target closed-now", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRecoverableError(tt.errMsg)
			if got != tt.want {
				t.Errorf("IsRecoverableError(%q) = %v, want %v", tt.errMsg, got, tt.want)
			}
		})
	}
}

func TestIsRecoverableError_NilLike(t *testing.T) {
	// Edge case: very long error messages
	longMsg := ""
	for i := 0; i < 1000; i++ {
		longMsg += "x"
	}
	longMsg += " Target closed "
	if !IsRecoverableError(longMsg) {
		t.Error("long error message containing recoverable pattern should be detected")
	}

	// Unicode characters
	if IsRecoverableError("🚀 Target closed 🚀") != true {
		t.Error("unicode surrounding recoverable pattern should be detected")
	}
}

func TestSafeSleep_CompletesWhenPageAlive(t *testing.T) {
	ctx := context.Background()
	alive := true
	check := func() bool { return alive }

	start := time.Now()
	SafeSleep(ctx, check, 200*time.Millisecond)
	elapsed := time.Since(start)

	// Should have slept approximately 200ms (allow ±100ms for timing)
	if elapsed < 100*time.Millisecond {
		t.Errorf("SafeSleep returned too early: %v", elapsed)
	}
}

func TestSafeSleep_ReturnsEarlyWhenPageDies(t *testing.T) {
	ctx := context.Background()
	var alive int32 = 1
	check := func() bool { return atomic.LoadInt32(&alive) == 1 }

	// Kill the page after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		atomic.StoreInt32(&alive, 0)
	}()

	start := time.Now()
	SafeSleep(ctx, check, 500*time.Millisecond)
	elapsed := time.Since(start)

	// Should have returned early (well before 500ms)
	if elapsed > 300*time.Millisecond {
		t.Errorf("SafeSleep did not return early after page death: %v", elapsed)
	}
}

func TestSafeSleep_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	alive := true
	check := func() bool { return alive }

	// Cancel context after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	SafeSleep(ctx, check, 500*time.Millisecond)
	elapsed := time.Since(start)

	// Should return early due to context cancellation
	if elapsed > 200*time.Millisecond {
		t.Errorf("SafeSleep did not return early on context cancel: %v", elapsed)
	}
}

func TestSafeSleep_ShortDuration(t *testing.T) {
	ctx := context.Background()
	alive := true
	check := func() bool { return alive }

	// Duration shorter than poll interval — should still work
	start := time.Now()
	SafeSleep(ctx, check, 50*time.Millisecond)
	elapsed := time.Since(start)

	if elapsed < 30*time.Millisecond {
		t.Errorf("SafeSleep with short duration returned too early: %v", elapsed)
	}
}

func TestSafeSleep_AlreadyDead(t *testing.T) {
	ctx := context.Background()
	check := func() bool { return false } // Page already dead

	start := time.Now()
	SafeSleep(ctx, check, 500*time.Millisecond)
	elapsed := time.Since(start)

	// Should return almost immediately
	if elapsed > 50*time.Millisecond {
		t.Errorf("SafeSleep with dead page should return immediately: %v", elapsed)
	}
}
