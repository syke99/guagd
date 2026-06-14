package account

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"guagd/internal/pkg/db"
	"guagd/internal/pkg/models"
)

type mockDB struct{}

func (m *mockDB) Exec(_ context.Context, _ string, _ ...any) error                  { return nil }
func (m *mockDB) Query(_ context.Context, _ string, _ db.Results, _ ...any) error   { return nil }
func (m *mockDB) QueryRow(_ context.Context, _ string, _ db.Result, _ ...any) error { return nil }
func (m *mockDB) BeginTx(_ context.Context) (db.Tx, error)                          { return &mockTx{}, nil }

type mockTx struct{}

func (t *mockTx) Exec(_ context.Context, _ string, _ ...any) error                  { return nil }
func (t *mockTx) Query(_ context.Context, _ string, _ db.Results, _ ...any) error   { return nil }
func (t *mockTx) QueryRow(_ context.Context, _ string, _ db.Result, _ ...any) error { return nil }
func (t *mockTx) Commit(_ context.Context) error                                     { return nil }
func (t *mockTx) Rollback(_ context.Context) error                                   { return nil }

var _ db.DB = (*mockDB)(nil)

// mockAuth controls what each auth operation returns.
type mockAuth struct {
	signUpID        string
	signUpEmail     bool
	signUpErr       error
	signInID        string
	signInWrong     bool
	signInErr       error
	createSessionErr error
	revokeErr       error
}

func (a *mockAuth) SignUp(_, _ string) (string, bool, error) {
	return a.signUpID, a.signUpEmail, a.signUpErr
}
func (a *mockAuth) SignIn(_, _ string) (string, bool, error) {
	return a.signInID, a.signInWrong, a.signInErr
}
func (a *mockAuth) CreateSession(_ *http.Request, _ http.ResponseWriter, _ string, _ map[string]any) error {
	return a.createSessionErr
}
func (a *mockAuth) RevokeSession(_ *http.Request, _ http.ResponseWriter) error {
	return a.revokeErr
}

func newClient(a *mockAuth) *accountClient {
	return &accountClient{baseRoute: testBaseRoute, db: &mockDB{}, auth: a}
}

const testBaseRoute = "/accounts/"

func TestNewAccountClient(t *testing.T) {
	c := NewAccountClient(testBaseRoute, nil)
	if c == nil {
		t.Fatal("expected non-nil accountClient")
	}
}

func TestHandlers(t *testing.T) {
	c := NewAccountClient(testBaseRoute, nil)
	handlers := c.Handlers()

	if len(handlers) == 0 {
		t.Fatal("expected at least one handler")
	}

	if _, ok := handlers[testBaseRoute+"waitlist/add"]; !ok {
		t.Errorf("expected handlers to contain '%s'", testBaseRoute+"waitlist/add")
	}
}

func TestAddWaitlist(t *testing.T) {
	t.Run("logs name and email and redirects to success", func(t *testing.T) {
		var buf bytes.Buffer
		orig := log.Writer()
		log.SetOutput(&buf)
		t.Cleanup(func() { log.SetOutput(orig) })

		payload := models.UserRegisterPayload{Name: "Test User", Email: "test@example.com"}
		body, _ := json.Marshal(payload)

		r := httptest.NewRequest(http.MethodPost, testBaseRoute+"addWaitlist", bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c := NewAccountClient(testBaseRoute, &mockDB{})
		c.addWaitlist(w, r)

		if !strings.Contains(buf.String(), "test@example.com") {
			t.Errorf("expected log to contain email, got: %s", buf.String())
		}
		if !strings.Contains(buf.String(), "Test User") {
			t.Errorf("expected log to contain name, got: %s", buf.String())
		}

		loc := w.Header().Get("HX-Location")
		if !strings.Contains(loc, "/waitlist/success") {
			t.Errorf("expected HX-Location to point to /waitlist/success, got: %s", loc)
		}
	})

	t.Run("invalid body redirects to failure", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, testBaseRoute+"addWaitlist", strings.NewReader("not json"))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c := NewAccountClient(testBaseRoute, nil)
		c.addWaitlist(w, r)

		loc := w.Header().Get("HX-Location")
		if !strings.Contains(loc, "/waitlist/failure") {
			t.Errorf("expected HX-Location to point to /waitlist/failure, got: %s", loc)
		}
	})

	t.Run("empty body redirects to failure", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, testBaseRoute+"addWaitlist", nil)
		w := httptest.NewRecorder()

		c := NewAccountClient(testBaseRoute, nil)
		c.addWaitlist(w, r)

		loc := w.Header().Get("HX-Location")
		if !strings.Contains(loc, "/waitlist/failure") {
			t.Errorf("expected HX-Location to point to /waitlist/failure, got: %s", loc)
		}
	})
}

func TestRedirect(t *testing.T) {
	t.Run("sets HX-Location header as JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := models.HTMXRedirectResponse{Path: "/waitlist/success", Target: "#hero-right"}

		redirect(w, resp)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}

		loc := w.Header().Get("HX-Location")
		var parsed models.HTMXRedirectResponse
		if err := json.Unmarshal([]byte(loc), &parsed); err != nil {
			t.Fatalf("HX-Location was not valid JSON: %s", loc)
		}
		if parsed.Path != resp.Path {
			t.Errorf("expected path %s, got %s", resp.Path, parsed.Path)
		}
		if parsed.Target != resp.Target {
			t.Errorf("expected target %s, got %s", resp.Target, parsed.Target)
		}
	})
}

func TestSearchPageURL(t *testing.T) {
	cases := []struct{ acctType, want string }{
		{"driver", "/garage/@alice"},
		{"club", "/hq/@alice"},
		{"shop", "/shop/@alice"},
		{"unknown", "/garage/@alice"},
		{"", "/garage/@alice"},
	}
	for _, c := range cases {
		got := searchPageURL("alice", c.acctType)
		if got != c.want {
			t.Errorf("searchPageURL(%q) = %q, want %q", c.acctType, got, c.want)
		}
	}
}

func TestSearchPageLabel(t *testing.T) {
	cases := []struct{ acctType, want string }{
		{"driver", "Garage"},
		{"club", "HQ"},
		{"shop", "Shop"},
		{"unknown", "Garage"},
		{"", "Garage"},
	}
	for _, c := range cases {
		got := searchPageLabel(c.acctType)
		if got != c.want {
			t.Errorf("searchPageLabel(%q) = %q, want %q", c.acctType, got, c.want)
		}
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	c := NewAccountClient(testBaseRoute, &mockDB{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, testBaseRoute+"search?q=", nil)
	c.search(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestSearch_ValidQuery(t *testing.T) {
	c := NewAccountClient(testBaseRoute, &mockDB{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, testBaseRoute+"search?q=alice", nil)
	c.search(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html" {
		t.Errorf("expected text/html, got %q", ct)
	}
}

// ── signUp ────────────────────────────────────────────────────────────────────

func signUpBody(t *testing.T, email, password, username, acctType string) *bytes.Reader {
	t.Helper()
	b, _ := json.Marshal(models.UserSignUpPayload{
		Email: email, Password: password, Username: username, AcctType: acctType,
	})
	return bytes.NewReader(b)
}

func TestSignUp_InvalidBody(t *testing.T) {
	c := newClient(&mockAuth{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader("bad json"))
	c.signUp(w, r)
	if !strings.Contains(w.Header().Get("HX-Location"), "/signup/failure") {
		t.Errorf("expected signup/failure redirect, got %q", w.Header().Get("HX-Location"))
	}
}

func TestSignUp_InvalidAcctType(t *testing.T) {
	c := newClient(&mockAuth{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signup", signUpBody(t, "a@b.com", "pass", "alice", "admin"))
	c.signUp(w, r)
	if !strings.Contains(w.Header().Get("HX-Location"), "/signup/failure") {
		t.Errorf("expected signup/failure redirect")
	}
}

func TestSignUp_AuthError(t *testing.T) {
	c := newClient(&mockAuth{signUpErr: errors.New("supertokens down")})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signup", signUpBody(t, "a@b.com", "pass", "alice", "driver"))
	c.signUp(w, r)
	if !strings.Contains(w.Header().Get("HX-Location"), "/signup/failure") {
		t.Errorf("expected signup/failure redirect")
	}
}

func TestSignUp_EmailExists(t *testing.T) {
	c := newClient(&mockAuth{signUpEmail: true})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signup", signUpBody(t, "a@b.com", "pass", "alice", "driver"))
	c.signUp(w, r)
	if !strings.Contains(w.Header().Get("HX-Location"), "/signup/failure") {
		t.Errorf("expected signup/failure redirect")
	}
}

func TestSignUp_Success_Driver(t *testing.T) {
	c := newClient(&mockAuth{signUpID: "uid-1"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signup", signUpBody(t, "a@b.com", "pass", "alice", "driver"))
	c.signUp(w, r)
	if got := w.Header().Get("HX-Redirect"); got != "/garage/@alice" {
		t.Errorf("expected /garage/@alice, got %q", got)
	}
}

func TestSignUp_Success_Club(t *testing.T) {
	c := newClient(&mockAuth{signUpID: "uid-2"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signup", signUpBody(t, "a@b.com", "pass", "racers", "club"))
	c.signUp(w, r)
	if got := w.Header().Get("HX-Redirect"); got != "/hq/@racers" {
		t.Errorf("expected /hq/@racers, got %q", got)
	}
}

// ── signIn ────────────────────────────────────────────────────────────────────

func signInBody(t *testing.T, email, password string) *bytes.Reader {
	t.Helper()
	b, _ := json.Marshal(models.UserSignInPayload{Email: email, Password: password})
	return bytes.NewReader(b)
}

func TestSignIn_InvalidBody(t *testing.T) {
	c := newClient(&mockAuth{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader("bad json"))
	c.signIn(w, r)
	if !strings.Contains(w.Header().Get("HX-Location"), "/signin/failure") {
		t.Errorf("expected signin/failure redirect")
	}
}

func TestSignIn_WrongCredentials(t *testing.T) {
	c := newClient(&mockAuth{signInWrong: true})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signin", signInBody(t, "a@b.com", "wrong"))
	c.signIn(w, r)
	if !strings.Contains(w.Header().Get("HX-Location"), "/signin/failure") {
		t.Errorf("expected signin/failure redirect")
	}
}

func TestSignIn_Success_Driver(t *testing.T) {
	c := newClient(&mockAuth{signInID: "uid-1"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signin", signInBody(t, "a@b.com", "pass"))
	c.signIn(w, r)
	if got := w.Header().Get("HX-Redirect"); !strings.HasPrefix(got, "/garage/@") {
		t.Errorf("expected /garage/@... redirect, got %q", got)
	}
}

// ── signOut ───────────────────────────────────────────────────────────────────

func TestSignOut_Success(t *testing.T) {
	c := newClient(&mockAuth{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signout", nil)
	c.signOut(w, r)
	if got := w.Header().Get("HX-Redirect"); got != "/" {
		t.Errorf("expected / redirect, got %q", got)
	}
}

func TestSignOut_RevokeError(t *testing.T) {
	c := newClient(&mockAuth{revokeErr: errors.New("revoke failed")})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/signout", nil)
	c.signOut(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
