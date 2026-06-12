package models

type AccountInfo struct {
	Username string `db:"username"`
	AcctType string `db:"acct_type"`
}

type AccountSearchResult struct {
	Username string `db:"username"`
	AcctType string `db:"acct_type"`
}
