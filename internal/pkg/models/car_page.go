package models

type VerificationCounts struct {
	Documented int `db:"documented"`
	Verified   int `db:"verified"`
	Performed  int `db:"performed"`
}

func (v VerificationCounts) Total() int {
	return v.Documented + v.Verified + v.Performed
}

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
	Verifications   VerificationCounts
	AvatarURL       string
	IsAuthenticated bool
}
