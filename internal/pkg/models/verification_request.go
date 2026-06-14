package models

type VerificationRequest struct {
	ID          string                    `db:"id"`
	ShopName    string                    `db:"shop_name"`
	ShopID      string                    `db:"shop_id"`
	ServiceType string                    `db:"service_type"`
	WorkType    string                    `db:"work_type"`
	Status      string                    `db:"status"`
	Notes       string                    `db:"notes"`
	CreatedAt   string                    `db:"created_at"`
	Items       []VerificationRequestItem `db:"-"`
}

type VerificationRequestItem struct {
	ID          string `db:"id"`
	RequestID   string `db:"request_id"`
	Status      string `db:"status"`
	DenialNotes string `db:"denial_notes"`
	RecordName  string `db:"record_name"`
	RecordType  string `db:"record_type"`
	Category    string `db:"category"`
	VerifLevel  string `db:"verif_level"`
}

type CarRecord struct {
	VerifID    string `db:"verif_id"`
	RecordName string `db:"record_name"`
	RecordType string `db:"record_type"`
	Category   string `db:"category"`
	VerifLevel string `db:"verif_level"`
}

type CreateRequestPayload struct {
	ShopUsername string   `json:"shop_username"`
	ServiceType  string   `json:"service_type"`
	WorkType     string   `json:"work_type"`
	Notes        string   `json:"notes"`
	VerifIDs     []string `json:"verif_ids"`
}

type RespondItem struct {
	ItemID      string `json:"item_id"`
	Status      string `json:"status"`
	DenialNotes string `json:"denial_notes"`
}

type RespondPayload struct {
	Items []RespondItem `json:"items"`
}

type RequestsPageData struct {
	AcctType        string
	AccountShortID  string
	Username        string
	AvatarURL       string
	IsAuthenticated bool
	Requests        []VerificationRequest
}
