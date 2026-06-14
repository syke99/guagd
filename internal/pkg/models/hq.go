package models

import "html/template"

type HQUser struct {
	AccountID     string `db:"account_id"`
	SupertokensID string `db:"supertokens_id"`
	Username      string `db:"username"`
}

type HQMember struct {
	Username      string `db:"username"`
	CoverPhotoKey string `db:"cover_photo_key"`
	CoverPhotoURL string `db:"-"`
	AvatarKey     string `db:"avatar_key"`
	AvatarURL     string `db:"-"`
}

type HQPageData struct {
	Username           string
	IsOwner            bool
	IsAuthenticated    bool
	AccountShortID     string
	MemberCount        int
	Members            []HQMember
	Layout             []LayoutItem
	SafeCSS            template.CSS
	CoverPhotoURL      string
	AvatarURL          string
}
