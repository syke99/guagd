package models

type ModUpload struct {
	ID          string `json:"id"           db:"id"`
	ModID       string `json:"mod_id"       db:"mod_id"`
	ObjectKey   string `json:"object_key"   db:"object_key"`
	Name        string `json:"name"         db:"name"`
	UploadType  string `json:"upload_type"  db:"upload_type"`
	ContentType string `json:"content_type" db:"content_type"`
	URL         string `json:"url"          db:"-"`
}
