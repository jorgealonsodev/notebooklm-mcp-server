package browser

import (
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

func TestSafeSleep_CompletesWhenPageAlive(t *testing.T) {
	alive := true
	check := func() bool { return alive }

	start := time.Now()
	SafeSleep(check, 200*time.Millisecond)
	elapsed := time.Since(start)

	// Should have slept approximately 200ms (allow ±100ms for timing)
	if elapsed < 100*time.Millisecond {
		t.Errorf("SafeSleep returned too early: %v", elapsed)
	}
}

func TestSafeSleep_ReturnsEarlyWhenPageDies(t *testing.T) {
	var alive int32 = 1
	check := func() bool { return atomic.LoadInt32(&alive) == 1 }

	// Kill the page after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		atomic.StoreInt32(&alive, 0)
	}()

	start := time.Now()
	SafeSleep(check, 500*time.Millisecond)
	elapsed := time.Since(start)

	// Should have returned early (well before 500ms)
	if elapsed > 300*time.Millisecond {
		t.Errorf("SafeSleep did not return early after page death: %v", elapsed)
	}
}
