package models

type Maintenance struct {
	ID            string `json:"id"             db:"id"`
	CarID         string `json:"car_id"         db:"car_id"`
	Name          string `json:"name"           db:"name"`
	Category      string `json:"category"       db:"category"`
	ServiceDate   string `json:"service_date"   db:"service_date"`
	Mileage       int    `json:"mileage"        db:"mileage"`
	Cost          int    `json:"cost"           db:"cost"`
	Notes         string `json:"notes"          db:"notes"`
	UploadCount   int    `json:"upload_count"   db:"upload_count"`
	VerifLevel    string `json:"verif_level"    db:"verif_level"`
	VerifShopName string `json:"verif_shop_name" db:"verif_shop_name"`
}
