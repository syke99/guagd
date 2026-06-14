package models

type CarPageOwner struct {
	Username  string `db:"username"`
	AvatarURL string `db:"-"`
	AvatarKey string `db:"avatar_key"`
	AcctType  string `db:"acct_type"`
}

type CarPageDoc struct {
	ID          string
	URL         string
	Name        string
	UploadType  string
	ContentType string
	SourceName  string
	SourceType  string // "mod" or "maintenance"
}

type CarPageData struct {
	Car             Car
	Owner           CarPageOwner
	Photos          []CarPhoto
	Mods            []Mod
	Maintenance     []Maintenance
	Docs            []CarPageDoc
	AvatarURL       string
	IsAuthenticated bool
}
