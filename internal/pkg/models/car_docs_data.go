package models

type CarDocsData struct {
	Uploads  []CarUpload
	IsOwner  bool
	Type     string
	RecordID string
}
