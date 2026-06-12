package models

type AccountInfo struct {
	AccountID string `db:"id"`
	Username  string `db:"username"`
	AcctType  string `db:"acct_type"`
}

type AccountSearchResult struct {
	Username string `db:"username"`
	AcctType string `db:"acct_type"`
}
