package garage

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
	"guagd/internal/pkg/models"
	"guagd/internal/pkg/sessions"
	"guagd/internal/pkg/storage"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

type mockDB struct{}

func (m *mockDB) Exec(_ context.Context, _ string, _ ...any) error                  { return nil }
func (m *mockDB) Query(_ context.Context, _ string, _ db.Results, _ ...any) error   { return nil }
func (m *mockDB) QueryRow(_ context.Context, _ string, _ db.Result, _ ...any) error { return nil }

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

func newClient() *GarageClient {
	store, _ := storage.New(storage.Config{
		AccountID: "fake", AccessKeyID: "fake", SecretAccessKey: "fake",
		CarPhotos:     storage.BucketConfig{Name: "cars", PublicURL: "https://cars.example.com"},
		AccountPhotos: storage.BucketConfig{Name: "accounts", PublicURL: "https://accounts.example.com"},
	})
	return &GarageClient{db: &mockDB{}, sessions: &mockGetter{}, storage: store}
}

func withAccountID(r *http.Request, id string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.ContextKeyAccountID, id))
}

// ── SaveLayout ────────────────────────────────────────────────────────────────

func TestSaveLayout_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/layout", strings.NewReader("not json"))
	newClient().SaveLayout(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSaveLayout_NoAccountInContext(t *testing.T) {
	body, _ := json.Marshal([]models.LayoutItem{{Component: "car-list", X: 0, Y: 0, W: 12, H: 6}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/layout", bytes.NewReader(body))
	newClient().SaveLayout(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSaveLayout_Valid(t *testing.T) {
	body, _ := json.Marshal([]models.LayoutItem{{Component: "car-list", X: 0, Y: 0, W: 12, H: 6}})
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodPost, "/api/v1/garage/layout", bytes.NewReader(body)), "acct-1")
	newClient().SaveLayout(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// ── SaveTheme ─────────────────────────────────────────────────────────────────

func TestSaveTheme_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/theme", strings.NewReader("not json"))
	newClient().SaveTheme(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSaveTheme_NoAccountInContext(t *testing.T) {
	body, _ := json.Marshal(map[string]map[string]string{"global": {"--accent": "#e85d04"}})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/theme", bytes.NewReader(body))
	newClient().SaveTheme(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSaveTheme_Valid(t *testing.T) {
	body, _ := json.Marshal(map[string]map[string]string{"global": {"--accent": "#e85d04"}})
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodPost, "/api/v1/garage/theme", bytes.NewReader(body)), "acct-1")
	newClient().SaveTheme(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// ── AddCar ────────────────────────────────────────────────────────────────────

func TestAddCar_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/add", strings.NewReader("bad"))
	newClient().AddCar(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddCar_MissingFields(t *testing.T) {
	cases := []models.Car{
		{Year: 0, Make: "Toyota", Model: "Supra"},
		{Year: 1993, Make: "", Model: "Supra"},
		{Year: 1993, Make: "Toyota", Model: ""},
	}
	for _, c := range cases {
		body, _ := json.Marshal(c)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/add", bytes.NewReader(body))
		newClient().AddCar(w, r)
		if w.Code != http.StatusBadRequest {
			t.Errorf("car %+v: expected 400, got %d", c, w.Code)
		}
	}
}

func TestAddCar_NoAccountInContext(t *testing.T) {
	body, _ := json.Marshal(models.Car{Year: 1993, Make: "Toyota", Model: "Supra"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/add", bytes.NewReader(body))
	newClient().AddCar(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ── RemoveCar ─────────────────────────────────────────────────────────────────

func TestRemoveCar_BadJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/remove", strings.NewReader("bad"))
	newClient().RemoveCar(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRemoveCar_MissingID(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"id": ""})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/remove", bytes.NewReader(body))
	newClient().RemoveCar(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRemoveCar_NoAccountInContext(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"id": "car-uuid"})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/remove", bytes.NewReader(body))
	newClient().RemoveCar(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRemoveCar_Valid(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"id": "car-uuid"})
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/remove", bytes.NewReader(body)), "acct-1")
	newClient().RemoveCar(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// ── GetCarPhotos ──────────────────────────────────────────────────────────────

func TestGetCarPhotos_MissingCarID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/garage/cars/photos", nil)
	newClient().GetCarPhotos(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── AddCarPhoto ───────────────────────────────────────────────────────────────

func TestAddCarPhoto_MissingCarID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/photos/add", strings.NewReader(`{"object_key":"k"}`))
	newClient().AddCarPhoto(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddCarPhoto_MissingObjectKey(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/photos/add?car_id=c1", strings.NewReader(`{"object_key":""}`))
	newClient().AddCarPhoto(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAddCarPhoto_NoAccountInContext(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/photos/add?car_id=c1", strings.NewReader(`{"object_key":"key.jpg"}`))
	newClient().AddCarPhoto(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ── RemoveCarPhoto ────────────────────────────────────────────────────────────

func TestRemoveCarPhoto_MissingPhotoID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/photos/remove", nil)
	newClient().RemoveCarPhoto(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRemoveCarPhoto_NoAccountInContext(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/photos/remove?photo_id=p1", nil)
	newClient().RemoveCarPhoto(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ── SaveCoverPhoto / SaveAvatar ───────────────────────────────────────────────

func TestSaveCoverPhoto_MissingKey(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cover", strings.NewReader(`{"object_key":""}`))
	newClient().SaveCoverPhoto(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSaveAvatar_MissingKey(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/avatar", strings.NewReader(`{"object_key":""}`))
	newClient().SaveAvatar(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── GaragePage session branching ──────────────────────────────────────────────

func TestGaragePage_NoSession(t *testing.T) {
	g := &GarageClient{db: &mockDB{}, sessions: &mockGetter{sess: nil}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/garage/@testuser", nil)
	r.Header.Set("HX-Request", "true")
	r.SetPathValue("username", "@testuser")
	g.GaragePage(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("expected Cache-Control: no-store")
	}
}

func TestGaragePage_SessionError(t *testing.T) {
	g := &GarageClient{db: &mockDB{}, sessions: &mockGetter{err: errors.New("no session")}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/garage/@testuser", nil)
	r.SetPathValue("username", "@testuser")
	g.GaragePage(w, r)
	// error from GetOptionalSession is silently ignored; page still renders
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// ── SetCarPhotoPrimary ────────────────────────────────────────────────────────

func TestSetCarPhotoPrimary_MissingParams(t *testing.T) {
	cases := []string{
		"/api/v1/garage/cars/photos/primary",
		"/api/v1/garage/cars/photos/primary?car_id=c1",
		"/api/v1/garage/cars/photos/primary?photo_id=p1",
	}
	for _, url := range cases {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, url, nil)
		newClient().SetCarPhotoPrimary(w, r)
		if w.Code != http.StatusBadRequest {
			t.Errorf("url %q: expected 400, got %d", url, w.Code)
		}
	}
}

func TestSetCarPhotoPrimary_NoAccountInContext(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cars/photos/primary?car_id=c1&photo_id=p1", nil)
	newClient().SetCarPhotoPrimary(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ── RemoveAvatar ──────────────────────────────────────────────────────────────

func TestRemoveAvatar_NoAccountInContext(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/garage/avatar", nil)
	newClient().RemoveAvatar(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRemoveAvatar_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodDelete, "/api/v1/garage/avatar", nil), "acct-1")
	newClient().RemoveAvatar(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// ── RemoveCoverPhoto ──────────────────────────────────────────────────────────

func TestRemoveCoverPhoto_NoAccountInContext(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/garage/cover", nil)
	newClient().RemoveCoverPhoto(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRemoveCoverPhoto_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodDelete, "/api/v1/garage/cover", nil), "acct-1")
	newClient().RemoveCoverPhoto(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

// ── GetCarPhotos valid path ───────────────────────────────────────────────────

func TestGetCarPhotos_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/garage/cars/photos?car_id=c1", nil)
	newClient().GetCarPhotos(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

// ── SaveAvatar valid path ─────────────────────────────────────────────────────

func TestSaveAvatar_NoAccountInContext(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/avatar", strings.NewReader(`{"object_key":"avatar.jpg"}`))
	newClient().SaveAvatar(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSaveAvatar_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodPost, "/api/v1/garage/avatar", strings.NewReader(`{"object_key":"avatar.jpg"}`)), "acct-1")
	newClient().SaveAvatar(w, r)
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

// ── SaveCoverPhoto valid path ─────────────────────────────────────────────────

func TestSaveCoverPhoto_NoAccountInContext(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/garage/cover", strings.NewReader(`{"object_key":"cover.jpg"}`))
	newClient().SaveCoverPhoto(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSaveCoverPhoto_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	r := withAccountID(httptest.NewRequest(http.MethodPost, "/api/v1/garage/cover", strings.NewReader(`{"object_key":"cover.jpg"}`)), "acct-1")
	newClient().SaveCoverPhoto(w, r)
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
