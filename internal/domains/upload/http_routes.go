package upload

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"

	"guagd/internal/pkg/middleware"
)

var allowedContentTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
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

	ext, ok := allowedContentTypes[body.ContentType]
	if !ok {
		http.Error(w, "unsupported content type", http.StatusBadRequest)
		return
	}

	if body.EntityType != "car" && body.EntityType != "account" {
		http.Error(w, "entity_type must be 'car' or 'account'", http.StatusBadRequest)
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
