package models

type Mod struct {
	ID               string `json:"id"                 db:"id"`
	CarID            string `json:"car_id"             db:"car_id"`
	Name             string `json:"name"               db:"name"`
	Category         string `json:"category"           db:"category"`
	InstallDate      string `json:"install_date"       db:"install_date"`
	MileageAtInstall int    `json:"mileage_at_install" db:"mileage_at_install"`
	Cost             int    `json:"cost"               db:"cost"`
	Notes            string `json:"notes"              db:"notes"`
	UploadCount      int    `json:"upload_count"       db:"upload_count"`
}
