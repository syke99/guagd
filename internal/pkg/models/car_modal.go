package models

type CarModalData struct {
	Car             Car
	PrimaryPhoto    *CarPhoto
	Photos          []CarPhoto // all non-primary photos
	TotalPhotoCount int        // all photos including primary
	Mods            []Mod
	Maintenance     []Maintenance
	IsOwner         bool
	MaxPhotos       int
}
