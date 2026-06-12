package models

import "html/template"

type GarageUser struct {
	AccountID     string `db:"account_id"`
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
