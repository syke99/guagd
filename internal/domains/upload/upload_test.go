package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"guagd/internal/pkg/middleware"
)

func newPresignRequest(body string, userID string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/upload/presign", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if userID != "" {
		r = r.WithContext(context.WithValue(r.Context(), middleware.ContextKeyUserID, userID))
	}
	return r
}

func TestPresign_BadBody(t *testing.T) {
	u := &UploadClient{}
	w := httptest.NewRecorder()
	r := newPresignRequest("not json", "user-1")
	u.Presign(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPresign_UnsupportedContentType(t *testing.T) {
	u := &UploadClient{}
	for _, ct := range []string{"image/gif", "application/pdf", "text/plain", ""} {
		body, _ := json.Marshal(map[string]string{"entity_type": "car", "content_type": ct})
		w := httptest.NewRecorder()
		u.Presign(w, newPresignRequest(string(body), "user-1"))
		if w.Code != http.StatusBadRequest {
			t.Errorf("content_type %q: expected 400, got %d", ct, w.Code)
		}
	}
}

func TestPresign_InvalidEntityType(t *testing.T) {
	u := &UploadClient{}
	for _, et := range []string{"shop", "admin", ""} {
		body, _ := json.Marshal(map[string]string{"entity_type": et, "content_type": "image/jpeg"})
		w := httptest.NewRecorder()
		u.Presign(w, newPresignRequest(string(body), "user-1"))
		if w.Code != http.StatusBadRequest {
			t.Errorf("entity_type %q: expected 400, got %d", et, w.Code)
		}
	}
}

func TestPresign_AllowedContentTypes(t *testing.T) {
	allowed := []string{"image/jpeg", "image/png", "image/webp"}
	for _, ct := range allowed {
		body, _ := json.Marshal(map[string]string{"entity_type": "car", "content_type": ct})
		w := httptest.NewRecorder()
		// storage is nil — will error, but we get past all validation
		func() {
			defer func() { recover() }()
			u := &UploadClient{}
			u.Presign(w, newPresignRequest(string(body), "user-1"))
		}()
		// 400 would mean content type was rejected — that's the failure we're guarding against
		if w.Code == http.StatusBadRequest && strings.Contains(w.Body.String(), "unsupported") {
			t.Errorf("content_type %q should be allowed but was rejected", ct)
		}
	}
}

func TestPresign_ValidEntityTypes(t *testing.T) {
	for _, et := range []string{"car", "account"} {
		body, _ := json.Marshal(map[string]string{"entity_type": et, "content_type": "image/jpeg"})
		w := httptest.NewRecorder()
		func() {
			defer func() { recover() }()
			u := &UploadClient{}
			u.Presign(w, newPresignRequest(string(body), "user-1"))
		}()
		if w.Code == http.StatusBadRequest && strings.Contains(w.Body.String(), "entity_type") {
			t.Errorf("entity_type %q should be valid but was rejected", et)
		}
	}
}

func TestPhotoContentTypesMap(t *testing.T) {
	expected := map[string]string{
		"image/jpeg": "jpg",
		"image/png":  "png",
		"image/webp": "webp",
	}
	for ct, ext := range expected {
		if got, ok := photoContentTypes[ct]; !ok || got != ext {
			t.Errorf("photoContentTypes[%q] = %q, want %q", ct, got, ext)
		}
	}
	if len(photoContentTypes) != len(expected) {
		var buf bytes.Buffer
		for k := range photoContentTypes {
			buf.WriteString(k)
			buf.WriteString(" ")
		}
		t.Errorf("unexpected extra entries in photoContentTypes: %s", buf.String())
	}
}

func TestCarFileContentTypesMap(t *testing.T) {
	expected := map[string]string{
		"image/jpeg":      "jpg",
		"image/png":       "png",
		"image/webp":      "webp",
		"application/pdf": "pdf",
	}
	for ct, ext := range expected {
		if got, ok := carFileContentTypes[ct]; !ok || got != ext {
			t.Errorf("carFileContentTypes[%q] = %q, want %q", ct, got, ext)
		}
	}
	if len(carFileContentTypes) != len(expected) {
		var buf bytes.Buffer
		for k := range carFileContentTypes {
			buf.WriteString(k)
			buf.WriteString(" ")
		}
		t.Errorf("unexpected extra entries in carFileContentTypes: %s", buf.String())
	}
}
