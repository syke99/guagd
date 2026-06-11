package upload

import (
	"net/http"

	"guagd/internal/pkg/middleware"
	"guagd/internal/pkg/storage"
)

type UploadClient struct {
	storage *storage.Client
}

func NewUploadClient(store *storage.Client) *UploadClient {
	return &UploadClient{storage: store}
}

func (u *UploadClient) Handlers() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"/api/v1/upload/presign": middleware.RequireAuth(u.Presign),
	}
}
