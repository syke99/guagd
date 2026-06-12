package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"guagd/internal/pkg/db"
	"guagd/internal/pkg/sessions"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

type mockDB struct{}

func (m *mockDB) Exec(_ context.Context, _ string, _ ...any) error                  { return nil }
func (m *mockDB) Query(_ context.Context, _ string, _ db.Results, _ ...any) error   { return nil }
func (m *mockDB) QueryRow(_ context.Context, _ string, _ db.Result, _ ...any) error { return nil }

type mockSession struct{ userID string }

func (s *mockSession) GetUserID() string                             { return s.userID }
func (s *mockSession) GetAccessTokenPayload() map[string]interface{} { return nil }
func (s *mockSession) RevokeSession() error                          { return nil }

type mockGetter struct{ sess sessions.Session }

func (g *mockGetter) GetSession(_ *http.Request, _ http.ResponseWriter) (sessions.Session, error) {
	return g.sess, nil
}
func (g *mockGetter) GetOptionalSession(_ *http.Request, _ http.ResponseWriter) (sessions.Session, error) {
	return g.sess, nil
}

// bare client with no dependencies — sufficient for static-serving handlers
func bare() *client {
	return &client{baseRoute: "/", db: &mockDB{}, sessions: &mockGetter{}}
}

// ── prefixRoute ───────────────────────────────────────────────────────────────

func TestPrefixRoute(t *testing.T) {
	cases := []struct{ prefix, route, want string }{
		{"/", "accounts", "/accounts"},
		{"/api", "accounts", "/api/accounts"},
		{"/api/", "accounts", "/api/accounts"},
		{"/", "", "/"},
	}
	for _, c := range cases {
		got := prefixRoute(c.prefix, c.route)
		if got != c.want {
			t.Errorf("prefixRoute(%q, %q) = %q, want %q", c.prefix, c.route, got, c.want)
		}
	}
}

// ── newVisitorID ──────────────────────────────────────────────────────────────

func TestNewVisitorID(t *testing.T) {
	id := newVisitorID()
	if id == "" {
		t.Fatal("expected non-empty visitor ID")
	}
	parts := strings.Split(id, "-")
	if len(parts) != 5 {
		t.Errorf("expected UUID-like format with 5 parts, got %q", id)
	}
	// should be unique across calls
	if id2 := newVisitorID(); id == id2 {
		t.Errorf("expected unique IDs, got same value twice: %q", id)
	}
}

// ── serveFragment ─────────────────────────────────────────────────────────────

func TestServeFragment_HTMXRequest(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/waitlist", nil)
	r.Header.Set("HX-Request", "true")
	bare().serveFragment(w, r, "pages/landing/waitlist/signup.html")
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestServeFragment_DirectNav(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/waitlist", nil)
	// no HX-Request header → serves app shell
	bare().serveFragment(w, r, "pages/landing/waitlist/signup.html")
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ── page handlers ─────────────────────────────────────────────────────────────

func testHandler(t *testing.T, fn http.HandlerFunc, path string) {
	t.Helper()
	for _, htmx := range []string{"true", ""} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, path, nil)
		if htmx != "" {
			r.Header.Set("HX-Request", htmx)
		}
		fn(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("path=%q HX-Request=%q: expected 200, got %d", path, htmx, w.Code)
		}
	}
}

func TestWaitlist(t *testing.T)        { testHandler(t, bare().waitlist, "/waitlist") }
func TestWaitlistSuccess(t *testing.T) { testHandler(t, bare().waitlistSuccess, "/waitlist/success") }
func TestWaitlistFailure(t *testing.T) { testHandler(t, bare().waitlistFailure, "/waitlist/failure") }
func TestSignupPage(t *testing.T)      { testHandler(t, bare().signupPage, "/signup") }
func TestSignupFailure(t *testing.T)   { testHandler(t, bare().signupFailure, "/signup/failure") }
func TestSigninPage(t *testing.T)      { testHandler(t, bare().signinPage, "/signin") }
func TestSigninFailure(t *testing.T)   { testHandler(t, bare().signinFailure, "/signin/failure") }

// ── accessPage ────────────────────────────────────────────────────────────────

func TestAccessPage(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/access", nil)
	bare().accessPage(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "localStorage") {
		t.Errorf("expected localStorage script in response")
	}
}

// ── trackVisit ────────────────────────────────────────────────────────────────

func TestTrackVisit_BadReferer(t *testing.T) {
	c := &client{
		baseRoute: "/",
		publicURL: "https://gaugd.com",
		db:        &mockDB{},
		sessions:  &mockGetter{},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/track/visit", nil)
	r.Header.Set("Referer", "https://example.com/other")
	c.trackVisit(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestTrackVisit_NoCookie(t *testing.T) {
	c := &client{
		baseRoute: "/",
		db:        &mockDB{},
		sessions:  &mockGetter{},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/track/visit", nil)
	c.trackVisit(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestTrackVisit_NoSession(t *testing.T) {
	c := &client{
		baseRoute: "/",
		db:        &mockDB{},
		sessions:  &mockGetter{sess: nil},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/track/visit", nil)
	r.AddCookie(&http.Cookie{Name: "visitor_id", Value: "test-visitor-123"})
	c.trackVisit(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestTrackVisit_WithSession(t *testing.T) {
	c := &client{
		baseRoute: "/",
		db:        &mockDB{},
		sessions:  &mockGetter{sess: &mockSession{userID: "user-abc"}},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/track/visit", nil)
	r.AddCookie(&http.Cookie{Name: "visitor_id", Value: "test-visitor-456"})
	c.trackVisit(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestTrackVisit_MatchingReferer(t *testing.T) {
	c := &client{
		baseRoute: "/",
		publicURL: "https://gaugd.com",
		db:        &mockDB{},
		sessions:  &mockGetter{},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/track/visit", nil)
	r.Header.Set("Referer", "https://gaugd.com/garage/@alice")
	r.AddCookie(&http.Cookie{Name: "visitor_id", Value: "test-visitor-789"})
	c.trackVisit(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
