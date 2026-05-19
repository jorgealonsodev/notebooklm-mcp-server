// Package browser manages a shared Playwright browser context with anti-detection,
// channel fallback, page health checks, and singleton locking.
package browser

import (
	"strings"
	"time"
)

// recoverableErrors lists error substrings that indicate a recoverable
// browser/page closure rather than a logic failure.
var recoverableErrors = []string{
	"target closed",
	"page closed",
	"browser closed",
	"websocket disconnected",
}

// IsRecoverableError returns true if the error message indicates a recoverable
// browser lifecycle event (page/browser closed, websocket disconnected).
func IsRecoverableError(errMsg string) bool {
	lower := strings.ToLower(errMsg)
	for _, pattern := range recoverableErrors {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// SafeSleep sleeps for the given duration but checks every pollInterval/10
// whether the page is still alive. If the page dies during the sleep, it
// returns early.
func SafeSleep(isPageAlive func() bool, duration time.Duration) {
	const pollInterval = 100 * time.Millisecond
	polls := int(duration / pollInterval)
	if polls < 1 {
		time.Sleep(duration)
		return
	}
	for i := 0; i < polls; i++ {
		if !isPageAlive() {
			return
		}
		// Check every 10th poll to avoid overhead
		if i%10 == 0 {
			if !isPageAlive() {
				return
			}
		}
		time.Sleep(pollInterval)
	}
}
