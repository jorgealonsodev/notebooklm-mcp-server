package session

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
)

// mockBrowserManager implements BrowserManager for testing.
type mockBrowserManager struct {
	mu          sync.Mutex
	pages       []*mockPage
	newPageErr  error
	healthy     bool
	closeCalled bool
}

func (m *mockBrowserManager) Launch(ctx context.Context) error       { return nil }
func (m *mockBrowserManager) Close() error {
	m.mu.Lock()
	m.closeCalled = true
	m.mu.Unlock()
	return nil
}
func (m *mockBrowserManager) Healthy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.healthy
}
func (m *mockBrowserManager) NewPage(ctx context.Context) (pageOps, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.newPageErr != nil {
		return nil, m.newPageErr
	}
	page := &mockPage{}
	m.pages = append(m.pages, page)
	return page, nil
}

// mockPage implements pageOps for testing.
type mockPage struct {
	mu           sync.Mutex
	closed       bool
	reloadErr    error
	closeErr     error
	reloadCalled bool
}

func (m *mockPage) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return m.closeErr
}

func (m *mockPage) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *mockPage) Reload(opts ...interface{}) error {
	m.mu.Lock()
	m.reloadCalled = true
	m.mu.Unlock()
	return m.reloadErr
}

func TestNewManager(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           5,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestManagerCreate(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           5,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	sess, err := mgr.Create("https://notebooklm.google.com")
	if err != nil {
		t.Fatalf("Create() error: %v", err)
	}
	if sess == nil {
		t.Fatal("Create() returned nil session")
	}
	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}
}

func TestManagerGet(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           5,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	sess, err := mgr.Create("https://notebooklm.google.com")
	if err != nil {
		t.Fatal(err)
	}

	got, err := mgr.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if got.ID != sess.ID {
		t.Errorf("Get() ID = %q, want %q", got.ID, sess.ID)
	}
}

func TestManagerGetNotFound(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           5,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	_, err := mgr.Get("nonexistent")
	if err == nil {
		t.Error("Get() should error for nonexistent session")
	}
}

func TestManagerClose(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           5,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	sess, err := mgr.Create("https://notebooklm.google.com")
	if err != nil {
		t.Fatal(err)
	}

	err = mgr.Close(sess.ID)
	if err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Session should be removed
	_, err = mgr.Get(sess.ID)
	if err == nil {
		t.Error("session should be removed after Close()")
	}
}

func TestManagerList(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           5,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	mgr.Create("https://nb1.com")
	mgr.Create("https://nb2.com")

	list := mgr.List()
	if len(list) != 2 {
		t.Errorf("List() returned %d sessions, want 2", len(list))
	}
}

func TestManagerStats(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           5,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	s1, _ := mgr.Create("https://nb1.com")
	s2, _ := mgr.Create("https://nb2.com")
	s1.IncrementMessages()
	s1.IncrementMessages()
	s2.IncrementMessages()

	stats := mgr.Stats()
	if stats.Active != 2 {
		t.Errorf("Stats().Active = %d, want 2", stats.Active)
	}
	if stats.TotalMessages != 3 {
		t.Errorf("Stats().TotalMessages = %d, want 3", stats.TotalMessages)
	}
}

func TestManagerCloseAll(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           5,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	mgr.Create("https://nb1.com")
	mgr.Create("https://nb2.com")

	err := mgr.CloseAll()
	if err != nil {
		t.Fatalf("CloseAll() error: %v", err)
	}

	list := mgr.List()
	if len(list) != 0 {
		t.Errorf("List() after CloseAll() = %d, want 0", len(list))
	}
}

func TestManagerMaxSessions(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           2,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	mgr.Create("https://nb1.com")
	mgr.Create("https://nb2.com")

	_, err := mgr.Create("https://nb3.com")
	if err == nil {
		t.Error("Create() should error when max sessions reached")
	}
}

func TestManagerConcurrentAccess(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           20,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess, err := mgr.Create("https://nb.com")
			if err != nil {
				t.Errorf("Create() error: %v", err)
				return
			}
			_ = mgr.Close(sess.ID)
		}()
	}
	wg.Wait()

	// All sessions should be closed
	if len(mgr.List()) != 0 {
		t.Errorf("expected 0 sessions after concurrent create/close, got %d", len(mgr.List()))
	}
}

func TestManagerStartStopCleanup(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           5,
		SessionTimeoutSeconds: 1, // 1 second timeout for fast test
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	sess, _ := mgr.Create("https://nb.com")

	// Manually set LastActivity to trigger expiry
	mgr.sessions[sess.ID].LastActivity = time.Now().Add(-2 * time.Second)

	// Call cleanup directly (the ticker interval is floored at 60s, too slow for tests)
	mgr.cleanupExpired()

	// Session should be cleaned up
	_, err := mgr.Get(sess.ID)
	if err == nil {
		t.Error("expired session should be cleaned up")
	}

	// Verify StartCleanup/StopCleanup don't panic
	mgr.StartCleanup()
	mgr.StopCleanup()
}

func TestManagerReset(t *testing.T) {
	cfg := config.Config{
		MaxSessions:           5,
		SessionTimeoutSeconds: 900,
	}
	browser := &mockBrowserManager{healthy: true}

	mgr := NewManager(cfg, browser)
	sess, _ := mgr.Create("https://nb.com")
	sess.IncrementMessages()
	sess.IncrementMessages()

	err := mgr.Reset(sess.ID)
	if err != nil {
		t.Fatalf("Reset() error: %v", err)
	}

	// Message count should be reset
	got, _ := mgr.Get(sess.ID)
	if got.MessageCount != 0 {
		t.Errorf("MessageCount after Reset = %d, want 0", got.MessageCount)
	}

	// Reload should have been called
	mp := sess.Page.(*mockPage)
	if !mp.reloadCalled {
		t.Error("Reset() should call page.Reload()")
	}
}

func TestManagerCleanupInterval(t *testing.T) {
	tests := []struct {
		timeoutSeconds int
		wantInterval   time.Duration
	}{
		{900, 300 * time.Second},  // min(450, 300) = 300, max(60, 300) = 300
		{120, 60 * time.Second},   // min(60, 300) = 60, max(60, 60) = 60
		{30, 60 * time.Second},    // min(15, 300) = 15, max(60, 15) = 60
		{1800, 300 * time.Second}, // min(900, 300) = 300, max(60, 300) = 300
	}

	for _, tt := range tests {
		got := cleanupInterval(time.Duration(tt.timeoutSeconds) * time.Second)
		if got != tt.wantInterval {
			t.Errorf("cleanupInterval(%v) = %v, want %v", tt.timeoutSeconds, got, tt.wantInterval)
		}
	}
}
