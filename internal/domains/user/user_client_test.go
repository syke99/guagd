package user

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

const testBaseRoute = "/users/"

func TestNewUserClient(t *testing.T) {
	c := NewUserClient(testBaseRoute, nil)
	if c == nil {
		t.Fatal("expected non-nil userClient")
	}
}

func TestHandlers(t *testing.T) {
	c := NewUserClient(testBaseRoute, nil)
	handlers := c.Handlers()

	if len(handlers) == 0 {
		t.Fatal("expected at least one handler")
	}

	if _, ok := handlers[testBaseRoute+"addWaitlist"]; !ok {
		t.Errorf("expected handlers to contain '%s'", testBaseRoute+"addWaitlist")
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

		c := NewUserClient(testBaseRoute, &mockDB{})
		c.addWaitlist(w, r)

		if !strings.Contains(buf.String(), "test@example.com") {
			t.Errorf("expected log to contain email, got: %s", buf.String())
		}
		if !strings.Contains(buf.String(), "Test User") {
			t.Errorf("expected log to contain name, got: %s", buf.String())
		}

		loc := w.Header().Get("HX-Location")
		if !strings.Contains(loc, "/signup/success") {
			t.Errorf("expected HX-Location to point to /signup/success, got: %s", loc)
		}
	})

	t.Run("invalid body redirects to failure", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, testBaseRoute+"addWaitlist", strings.NewReader("not json"))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		c := NewUserClient(testBaseRoute, nil)
		c.addWaitlist(w, r)

		loc := w.Header().Get("HX-Location")
		if !strings.Contains(loc, "/signup/failure") {
			t.Errorf("expected HX-Location to point to /signup/failure, got: %s", loc)
		}
	})

	t.Run("empty body redirects to failure", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, testBaseRoute+"addWaitlist", nil)
		w := httptest.NewRecorder()

		c := NewUserClient(testBaseRoute, nil)
		c.addWaitlist(w, r)

		loc := w.Header().Get("HX-Location")
		if !strings.Contains(loc, "/signup/failure") {
			t.Errorf("expected HX-Location to point to /signup/failure, got: %s", loc)
		}
	})
}

func TestRedirect(t *testing.T) {
	t.Run("sets HX-Location header as JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		resp := models.HTMXRedirectResponse{Path: "/signup/success", Target: "#hero-right"}

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
