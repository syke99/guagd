package account

import (
	"bytes"
	"context"
	"encoding/json"
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

var _ db.DB = (*mockDB)(nil)

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
		log.SetOutput(&buf)
		t.Cleanup(func() { log.SetOutput(nil) })

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
