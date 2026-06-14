package requests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"guagd/internal/pkg/db"
	"guagd/internal/pkg/models"
	"guagd/internal/pkg/sessions"
	"guagd/internal/pkg/storage"
)

// ── mocks ──────────────────────────────────────────────────────────────────────

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

type mockSession struct{ userID string }

func (s *mockSession) GetUserID() string                             { return s.userID }
func (s *mockSession) GetAccessTokenPayload() map[string]interface{} { return nil }
func (s *mockSession) RevokeSession() error                          { return nil }

type mockGetter struct {
	sess sessions.Session
}

func (g *mockGetter) GetSession(_ *http.Request, _ http.ResponseWriter) (sessions.Session, error) {
	return g.sess, nil
}
func (g *mockGetter) GetOptionalSession(_ *http.Request, _ http.ResponseWriter) (sessions.Session, error) {
	return g.sess, nil
}

func newClient() *RequestsClient {
	store, _ := storage.New(storage.Config{
		AccountID: "fake", AccessKeyID: "fake", SecretAccessKey: "fake",
		CarPhotos:     storage.BucketConfig{Name: "cars", PublicURL: "https://cars.example.com"},
		AccountPhotos: storage.BucketConfig{Name: "accounts", PublicURL: "https://accounts.example.com"},
	})
	return &RequestsClient{db: &mockDB{}, sessions: &mockGetter{}, storage: store}
}

func newClientWithSession(userID string) *RequestsClient {
	store, _ := storage.New(storage.Config{
		AccountID: "fake", AccessKeyID: "fake", SecretAccessKey: "fake",
		CarPhotos:     storage.BucketConfig{Name: "cars", PublicURL: "https://cars.example.com"},
		AccountPhotos: storage.BucketConfig{Name: "accounts", PublicURL: "https://accounts.example.com"},
	})
	return &RequestsClient{db: &mockDB{}, sessions: &mockGetter{sess: &mockSession{userID: userID}}, storage: store}
}

// ── RequestsPage ──────────────────────────────────────────────────────────────

func TestRequestsPage_NoSession(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/requests?u=abc12345", nil)
	newClient().RequestsPage(w, r)
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d", w.Code)
	}
}

func TestRequestsPage_MissingShortID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/requests", nil)
	newClientWithSession("user-1").RequestsPage(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRequestsPage_ShortIDTooShort(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/requests?u=abc", nil)
	newClientWithSession("user-1").RequestsPage(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── WizardCarsFragment ────────────────────────────────────────────────────────

func TestWizardCarsFragment_NoSession(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/driver/requests/wizard/cars?u=abc12345", nil)
	newClient().WizardCarsFragment(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWizardCarsFragment_MissingShortID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/driver/requests/wizard/cars", nil)
	newClientWithSession("user-1").WizardCarsFragment(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── WizardRecordsFragment ─────────────────────────────────────────────────────

func TestWizardRecordsFragment_NoSession(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/driver/requests/wizard/records?u=abc12345&car_id=c1", nil)
	newClient().WizardRecordsFragment(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWizardRecordsFragment_MissingCarID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/driver/requests/wizard/records?u=abc12345", nil)
	newClientWithSession("user-1").WizardRecordsFragment(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── ListDriverRequests ────────────────────────────────────────────────────────

func TestListDriverRequests_NoSession(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/driver/requests?u=abc12345", nil)
	newClient().ListDriverRequests(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListDriverRequests_MissingShortID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/driver/requests", nil)
	newClientWithSession("user-1").ListDriverRequests(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── ListShopRequests ──────────────────────────────────────────────────────────

func TestListShopRequests_NoSession(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/shop/requests?u=abc12345", nil)
	newClient().ListShopRequests(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListShopRequests_MissingShortID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/shop/requests", nil)
	newClientWithSession("user-1").ListShopRequests(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── CreateRequest ─────────────────────────────────────────────────────────────

func TestCreateRequest_NoSession(t *testing.T) {
	body, _ := json.Marshal(models.CreateRequestPayload{
		ShopUsername: "shopx",
		ServiceType:  "past",
		WorkType:     "own_work",
		VerifIDs:     []string{"v1"},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/driver/requests/create", bytes.NewReader(body))
	newClient().CreateRequest(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreateRequest_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/driver/requests/create", bytes.NewReader([]byte("not json")))
	newClientWithSession("user-1").CreateRequest(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateRequest_MissingShopUsername(t *testing.T) {
	body, _ := json.Marshal(models.CreateRequestPayload{
		ServiceType: "past",
		WorkType:    "own_work",
		VerifIDs:    []string{"v1"},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/driver/requests/create", bytes.NewReader(body))
	newClientWithSession("user-1").CreateRequest(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateRequest_MissingVerifIDs(t *testing.T) {
	body, _ := json.Marshal(models.CreateRequestPayload{
		ShopUsername: "shopx",
		ServiceType:  "past",
		WorkType:     "own_work",
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/driver/requests/create", bytes.NewReader(body))
	newClientWithSession("user-1").CreateRequest(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateRequest_InvalidServiceType(t *testing.T) {
	body, _ := json.Marshal(models.CreateRequestPayload{
		ShopUsername: "shopx",
		ServiceType:  "invalid",
		WorkType:     "own_work",
		VerifIDs:     []string{"v1"},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/driver/requests/create", bytes.NewReader(body))
	newClientWithSession("user-1").CreateRequest(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateRequest_InvalidWorkType(t *testing.T) {
	body, _ := json.Marshal(models.CreateRequestPayload{
		ShopUsername: "shopx",
		ServiceType:  "past",
		WorkType:     "bad_type",
		VerifIDs:     []string{"v1"},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/driver/requests/create", bytes.NewReader(body))
	newClientWithSession("user-1").CreateRequest(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── RespondToItems ────────────────────────────────────────────────────────────

func TestRespondToItems_NoSession(t *testing.T) {
	body, _ := json.Marshal(models.RespondPayload{
		Items: []models.RespondItem{{ItemID: "i1", Status: "approved"}},
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/shop/requests/respond", bytes.NewReader(body))
	newClient().RespondToItems(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRespondToItems_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/shop/requests/respond", bytes.NewReader([]byte("bad")))
	newClientWithSession("shop-1").RespondToItems(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRespondToItems_EmptyItems(t *testing.T) {
	body, _ := json.Marshal(models.RespondPayload{Items: []models.RespondItem{}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/shop/requests/respond", bytes.NewReader(body))
	newClientWithSession("shop-1").RespondToItems(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── ShopSearchFragment ────────────────────────────────────────────────────────

func TestShopSearchFragment_EmptyQuery(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/driver/requests/wizard/shops", nil)
	newClient().ShopSearchFragment(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestShopSearchFragment_WithQuery(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/driver/requests/wizard/shops?q=testshop", nil)
	newClient().ShopSearchFragment(w, r)
	// mockDB.Query returns nil (no results), so we get 204 from the no-content path
	// or 200 with empty body — either is acceptable; just verify no server error
	if w.Code >= http.StatusInternalServerError {
		t.Errorf("unexpected server error: %d", w.Code)
	}
}
