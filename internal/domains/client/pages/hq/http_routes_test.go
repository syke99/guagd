package hq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"guagd/internal/pkg/db"
	"guagd/internal/pkg/middleware"
	"guagd/internal/pkg/sessions"
	"guagd/internal/pkg/storage"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

type mockDB struct{}

func (m *mockDB) Exec(_ context.Context, _ string, _ ...any) error                  { return nil }
func (m *mockDB) Query(_ context.Context, _ string, _ db.Results, _ ...any) error   { return nil }
func (m *mockDB) QueryRow(_ context.Context, _ string, _ db.Result, _ ...any) error { return nil }

type errDB struct{}

func (m *errDB) Exec(_ context.Context, _ string, _ ...any) error                  { return errors.New("db error") }
func (m *errDB) Query(_ context.Context, _ string, _ db.Results, _ ...any) error   { return errors.New("db error") }
func (m *errDB) QueryRow(_ context.Context, _ string, _ db.Result, _ ...any) error { return errors.New("db error") }

type mockSession struct {
	userID  string
	payload map[string]interface{}
}

func (s *mockSession) GetUserID() string                             { return s.userID }
func (s *mockSession) GetAccessTokenPayload() map[string]interface{} { return s.payload }
func (s *mockSession) RevokeSession() error                          { return nil }

type mockGetter struct {
	sess sessions.Session
	err  error
}

func (g *mockGetter) GetSession(_ *http.Request, _ http.ResponseWriter) (sessions.Session, error) {
	return g.sess, g.err
}
func (g *mockGetter) GetOptionalSession(_ *http.Request, _ http.ResponseWriter) (sessions.Session, error) {
	return g.sess, g.err
}

func newClient(sg sessions.Getter) *HQClient {
	store, _ := storage.New(storage.Config{
		AccountID: "fake", AccessKeyID: "fake", SecretAccessKey: "fake",
		AccountPhotos: storage.BucketConfig{Name: "accounts", PublicURL: "https://accounts.example.com"},
	})
	return &HQClient{db: &mockDB{}, sessions: sg, storage: store}
}

func withAccountID(r *http.Request, id string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.ContextKeyAccountID, id))
}

// ── SaveLayout ────────────────────────────────────────────────────────────────

func TestSaveLayout_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/layout", strings.NewReader("bad"))
	newClient(nil).SaveLayout(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSaveLayout_NoAccount(t *testing.T) {
	body, _ := json.Marshal([]interface{}{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/layout", bytes.NewReader(body))
	newClient(nil).SaveLayout(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSaveLayout_Valid(t *testing.T) {
	body, _ := json.Marshal([]interface{}{})
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodPost, "/api/v1/hq/layout", bytes.NewReader(body)), "acct-1")
	newClient(nil).SaveLayout(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// ── SearchMembers ─────────────────────────────────────────────────────────────

func TestSearchMembers_NoSession(t *testing.T) {
	g := &mockGetter{err: errors.New("no session")}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/hq/members/search?q=foo", nil)
	newClient(g).SearchMembers(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSearchMembers_NonClubAccount(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "driver", "account_id": "acct-1"}}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/hq/members/search?q=foo", nil)
	newClient(g).SearchMembers(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestSearchMembers_MissingAccountID(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "club", "account_id": ""}}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/hq/members/search?q=foo", nil)
	newClient(g).SearchMembers(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSearchMembers_EmptyQuery(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "club", "account_id": "acct-1"}}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/hq/members/search?q=", nil)
	newClient(g).SearchMembers(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestSearchMembers_ValidQuery(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "club", "account_id": "acct-1"}}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/hq/members/search?q=foo", nil)
	newClient(g).SearchMembers(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ── AddMember ─────────────────────────────────────────────────────────────────

func TestAddMember_NoSession(t *testing.T) {
	g := &mockGetter{err: errors.New("no session")}
	body, _ := json.Marshal(map[string]string{"username": "someone"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/members/add", bytes.NewReader(body))
	newClient(g).AddMember(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAddMember_NonClubAccount(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "driver", "account_id": "acct-1"}}}
	body, _ := json.Marshal(map[string]string{"username": "someone"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/members/add", bytes.NewReader(body))
	newClient(g).AddMember(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAddMember_MissingAccountID(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "club", "account_id": ""}}}
	body, _ := json.Marshal(map[string]string{"username": "someone"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/members/add", bytes.NewReader(body))
	newClient(g).AddMember(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAddMember_MissingUsername(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "club", "account_id": "acct-1", "username": "myclub"}}}
	body, _ := json.Marshal(map[string]string{"username": ""})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/members/add", bytes.NewReader(body))
	newClient(g).AddMember(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddMember_Valid(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "club", "account_id": "acct-1", "username": "myclub"}}}
	body, _ := json.Marshal(map[string]string{"username": "newmember"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/members/add", bytes.NewReader(body))
	newClient(g).AddMember(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("HX-Redirect"), "myclub") {
		t.Errorf("expected HX-Redirect to contain club username, got %q", w.Header().Get("HX-Redirect"))
	}
}

// ── RemoveMember ──────────────────────────────────────────────────────────────

func TestRemoveMember_NoSession(t *testing.T) {
	g := &mockGetter{err: errors.New("no session")}
	body, _ := json.Marshal(map[string]string{"username": "someone"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/members/remove", bytes.NewReader(body))
	newClient(g).RemoveMember(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRemoveMember_NonClubAccount(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "driver", "account_id": "acct-1"}}}
	body, _ := json.Marshal(map[string]string{"username": "someone"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/members/remove", bytes.NewReader(body))
	newClient(g).RemoveMember(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestRemoveMember_MissingUsername(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "club", "account_id": "acct-1", "username": "myclub"}}}
	body, _ := json.Marshal(map[string]string{"username": ""})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/members/remove", bytes.NewReader(body))
	newClient(g).RemoveMember(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRemoveMember_Valid(t *testing.T) {
	g := &mockGetter{sess: &mockSession{payload: map[string]interface{}{"acct_type": "club", "account_id": "acct-1", "username": "myclub"}}}
	body, _ := json.Marshal(map[string]string{"username": "oldmember"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/members/remove", bytes.NewReader(body))
	newClient(g).RemoveMember(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("HX-Redirect"), "myclub") {
		t.Errorf("expected HX-Redirect to contain club username, got %q", w.Header().Get("HX-Redirect"))
	}
}

// ── HQPage session branching ──────────────────────────────────────────────────

func TestHQPage_NoSession(t *testing.T) {
	g := &mockGetter{sess: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/hq/@testclub", nil)
	r.SetPathValue("username", "@testclub")
	newClient(g).HQPage(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("expected Cache-Control: no-store")
	}
}

// ── SaveCoverPhoto ────────────────────────────────────────────────────────────

func TestHQSaveCoverPhoto_MissingKey(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/cover", strings.NewReader(`{"object_key":""}`))
	newClient(nil).SaveCoverPhoto(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHQSaveCoverPhoto_NoAccount(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/cover", strings.NewReader(`{"object_key":"k.jpg"}`))
	newClient(nil).SaveCoverPhoto(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHQSaveCoverPhoto_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodPost, "/api/v1/hq/cover", strings.NewReader(`{"object_key":"cover.jpg"}`)), "acct-1")
	newClient(nil).SaveCoverPhoto(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response not JSON: %v", err)
	}
	if resp["url"] == "" {
		t.Errorf("expected non-empty url in response")
	}
}

// ── SaveTheme ─────────────────────────────────────────────────────────────────

func TestHQSaveTheme_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/theme", strings.NewReader("bad"))
	newClient(nil).SaveTheme(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHQSaveTheme_NoAccount(t *testing.T) {
	body, _ := json.Marshal(map[string]map[string]string{"global": {"--accent": "#e85d04"}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/hq/theme", bytes.NewReader(body))
	newClient(nil).SaveTheme(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHQSaveTheme_Valid(t *testing.T) {
	body, _ := json.Marshal(map[string]map[string]string{"global": {"--accent": "#e85d04"}})
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodPost, "/api/v1/hq/theme", bytes.NewReader(body)), "acct-1")
	newClient(nil).SaveTheme(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// ── RemoveCoverPhoto ──────────────────────────────────────────────────────────

func TestHQRemoveCoverPhoto_NoAccount(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/hq/cover", nil)
	newClient(nil).RemoveCoverPhoto(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHQRemoveCoverPhoto_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodDelete, "/api/v1/hq/cover", nil), "acct-1")
	newClient(nil).RemoveCoverPhoto(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// ── ListMembers ───────────────────────────────────────────────────────────────

func TestListMembers_MissingUsername(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/hq/members", nil)
	newClient(nil).ListMembers(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListMembers_NotFound(t *testing.T) {
	c := &HQClient{db: &errDB{}, sessions: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/hq/members?username=nobody", nil)
	c.ListMembers(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestListMembers_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/hq/members?username=myclub", nil)
	newClient(nil).ListMembers(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}
