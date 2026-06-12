package models

import "html/template"

type GarageUser struct {
	SupertokensID string `db:"supertokens_id"`
	Username      string `db:"username"`
}

type GaragePageData struct {
	Username        string
	IsOwner         bool
	IsAuthenticated bool
	CarCount        int
	Cars            []Car
	Layout          []LayoutItem
	SafeCSS         template.CSS
	CoverPhotoURL   string
}
