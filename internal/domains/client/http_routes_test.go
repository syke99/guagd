package client

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// bare client with no dependencies — sufficient for static-serving handlers
func bare() *client {
	return &client{baseRoute: "/"}
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
