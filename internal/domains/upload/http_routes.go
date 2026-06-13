package upload

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"

	"guagd/internal/pkg/middleware"
)

var photoContentTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

var modFileContentTypes = map[string]string{
	"image/jpeg":      "jpg",
	"image/png":       "png",
	"image/webp":      "webp",
	"application/pdf": "pdf",
}

func (u *UploadClient) Presign(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EntityType  string `json:"entity_type"`
		ContentType string `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch body.EntityType {
	case "car", "account", "mod":
	default:
		http.Error(w, "entity_type must be 'car', 'account', or 'mod'", http.StatusBadRequest)
		return
	}

	allowedTypes := photoContentTypes
	if body.EntityType == "mod" {
		allowedTypes = modFileContentTypes
	}

	ext, ok := allowedTypes[body.ContentType]
	if !ok {
		http.Error(w, "unsupported content type", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	objectKey := fmt.Sprintf("%s/%s.%s", userID, uuid.New().String(), ext)

	var (
		uploadURL string
		err       error
	)
	switch body.EntityType {
	case "car":
		uploadURL, err = u.storage.PresignCarPhotoUpload(r.Context(), objectKey, body.ContentType)
	case "account":
		uploadURL, err = u.storage.PresignAccountPhotoUpload(r.Context(), objectKey, body.ContentType)
	case "mod":
		uploadURL, err = u.storage.PresignModFileUpload(r.Context(), objectKey, body.ContentType)
	}
	if err != nil {
		log.Printf("presign: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"upload_url": uploadURL,
		"object_key": objectKey,
	})
}
