package session

import (
	"testing"
)

func TestNewBrowserSession(t *testing.T) {
	s := NewBrowserSession("https://notebooklm.google.com")

	if s.ID == "" {
		t.Error("session ID should not be empty")
	}
	if len(s.ID) != 8 {
		t.Errorf("session ID should be 8 chars, got %d: %s", len(s.ID), s.ID)
	}
	if s.NotebookURL != "https://notebooklm.google.com" {
		t.Errorf("NotebookURL = %q, want %q", s.NotebookURL, "https://notebooklm.google.com")
	}
	if s.MessageCount != 0 {
		t.Errorf("MessageCount = %d, want 0", s.MessageCount)
	}
	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if s.LastActivity.IsZero() {
		t.Error("LastActivity should be set")
	}
}

func TestSessionIDFormat(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s := NewBrowserSession("https://example.com")
		if seen[s.ID] {
			t.Errorf("duplicate session ID: %s", s.ID)
		}
		seen[s.ID] = true

		// Verify ID is hex
		for _, c := range s.ID {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("non-hex char in ID: %c", c)
			}
		}
	}
}

func TestSessionTouch(t *testing.T) {
	s := NewBrowserSession("https://example.com")
	oldActivity := s.LastActivity

	// Simulate some time passing
	s.LastActivity = s.LastActivity.Add(-10)

	s.Touch()

	if !s.LastActivity.After(oldActivity) {
		t.Error("Touch() should update LastActivity")
	}
}

func TestSessionIncrementMessages(t *testing.T) {
	s := NewBrowserSession("https://example.com")

	s.IncrementMessages()
	if s.MessageCount != 1 {
		t.Errorf("MessageCount = %d, want 1", s.MessageCount)
	}

	s.IncrementMessages()
	s.IncrementMessages()
	if s.MessageCount != 3 {
		t.Errorf("MessageCount = %d, want 3", s.MessageCount)
	}
}

func TestSessionCloseNilPage(t *testing.T) {
	s := NewBrowserSession("https://example.com")
	s.Page = nil // no page set

	err := s.Close()
	if err != nil {
		t.Errorf("Close() with nil page should not error: %v", err)
	}
}

func TestSessionResetNilPage(t *testing.T) {
	s := NewBrowserSession("https://example.com")
	s.Page = nil
	s.MessageCount = 5

	err := s.Reset()
	if err != nil {
		t.Errorf("Reset() with nil page should not error: %v", err)
	}
	if s.MessageCount != 0 {
		t.Errorf("Reset() should reset MessageCount even with nil page")
	}
}

func TestSessionCloseCallsPageClose(t *testing.T) {
	s := NewBrowserSession("https://example.com")
	mp := &mockPage{}
	s.Page = mp

	err := s.Close()
	if err != nil {
		t.Errorf("Close() error: %v", err)
	}
	if !mp.IsClosed() {
		t.Error("Close() should call page.Close()")
	}
}

func TestSessionResetCallsPageReload(t *testing.T) {
	s := NewBrowserSession("https://example.com")
	mp := &mockPage{}
	s.Page = mp
	s.MessageCount = 5

	err := s.Reset()
	if err != nil {
		t.Errorf("Reset() error: %v", err)
	}
	if !mp.reloadCalled {
		t.Error("Reset() should call page.Reload()")
	}
	if s.MessageCount != 0 {
		t.Errorf("MessageCount after Reset = %d, want 0", s.MessageCount)
	}
}
