package user

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const testBaseRoute = "/users/"

func TestNewUserClient(t *testing.T) {
	c := NewUserClient(testBaseRoute)
	if c == nil {
		t.Fatal("expected non-nil userClient")
	}
}

func TestHandlers(t *testing.T) {
	c := NewUserClient(testBaseRoute)
	handlers := c.Handlers()

	if len(handlers) == 0 {
		t.Fatal("expected at least one handler")
	}

	if _, ok := handlers[testBaseRoute+"register"]; !ok {
		t.Errorf("expected handlers to contain '%s'", testBaseRoute+"register")
	}
}

func TestRegister(t *testing.T) {
	t.Run("logs email from request body", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		t.Cleanup(func() { log.SetOutput(nil) })

		form := url.Values{}
		form.Set("email", "test@example.com")

		r := httptest.NewRequest(http.MethodPost, testBaseRoute+"register", strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		c := NewUserClient(testBaseRoute)
		c.register(w, r)

		if !strings.Contains(buf.String(), "test@example.com") {
			t.Errorf("expected log output to contain email, got: %s", buf.String())
		}
	})

	t.Run("empty email", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		t.Cleanup(func() { log.SetOutput(nil) })

		r := httptest.NewRequest(http.MethodPost, testBaseRoute+"register", nil)
		w := httptest.NewRecorder()

		c := NewUserClient(testBaseRoute)
		c.register(w, r)

		if !strings.Contains(buf.String(), "email:") {
			t.Errorf("expected log output, got: %s", buf.String())
		}
	})
}
