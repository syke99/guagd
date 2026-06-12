package models

import "html/template"

type HQUser struct {
	SupertokensID string `db:"supertokens_id"`
	Username      string `db:"username"`
}

type HQMember struct {
	Username string `db:"username"`
}

type HQPageData struct {
	Username        string
	IsOwner         bool
	IsAuthenticated bool
	MemberCount     int
	Members         []HQMember
	Layout          []LayoutItem
	SafeCSS         template.CSS
}
