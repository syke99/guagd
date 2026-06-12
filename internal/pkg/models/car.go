package models

type Car struct {
	ID              string `json:"id"                db:"id"`
	Year            int    `json:"year"              db:"year"`
	Make            string `json:"make"              db:"make"`
	Model           string `json:"model"             db:"model"`
	Trim            string `json:"trim"              db:"trim"`
	Mileage         int    `json:"mileage"           db:"mileage"`
	ObjectKey       string `json:"object_key"        db:"object_key"`
	PrimaryPhotoURL string `json:"primary_photo_url" db:"-"`
}

type CarPhoto struct {
	ID        string `json:"id"         db:"id"`
	CarID     string `json:"car_id"     db:"car_id"`
	ObjectKey string `json:"object_key" db:"object_key"`
	URL       string `json:"url"        db:"-"`
	IsPrimary bool   `json:"is_primary" db:"is_primary"`
}
