package account

import (
	"context"

	"guagd/internal/pkg/db"
	"guagd/internal/pkg/models"
)

type accountClient struct {
	baseRoute string
	db        db.DB
}

func NewAccountClient(baseRoute string, db db.DB) *accountClient {
	return &accountClient{baseRoute: baseRoute, db: db}
}

func (u *accountClient) createAccount(ctx context.Context, supertokensID, username, email, acctType string) error {
	return u.db.Exec(
		ctx,
		`INSERT INTO accounts (supertokens_id, username, email, acct_type)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (email) DO UPDATE
		 SET supertokens_id = EXCLUDED.supertokens_id,
		     username       = EXCLUDED.username,
		     acct_type      = EXCLUDED.acct_type`,
		supertokensID,
		username,
		email,
		acctType,
	)
}

func (u *accountClient) registerUser(ctx context.Context, name, email, visitorId string) error {
	return u.db.Exec(
		ctx,
		"INSERT INTO accounts (name, email, visitor_id) VALUES ($1, $2, $3)",
		name,
		email,
		visitorId,
	)
}

func (u *accountClient) searchAccounts(ctx context.Context, q, acctType string) ([]models.AccountSearchResult, error) {
	var results []models.AccountSearchResult
	var query string
	var args []any
	if acctType != "" {
		query = `SELECT username, acct_type FROM accounts
		 WHERE username ILIKE $1
		 AND username IS NOT NULL
		 AND acct_type = $2
		 ORDER BY username
		 LIMIT 10`
		args = []any{"%" + q + "%", acctType}
	} else {
		query = `SELECT username, acct_type FROM accounts
		 WHERE username ILIKE $1
		 AND username IS NOT NULL
		 ORDER BY username
		 LIMIT 10`
		args = []any{"%" + q + "%"}
	}
	err := u.db.Query(ctx, query, db.WithResultsOf(&results), args...)
	return results, err
}

func (u *accountClient) getAccountBySupertokensID(ctx context.Context, supertokensID string) (models.AccountInfo, error) {
	var info models.AccountInfo
	err := u.db.QueryRow(ctx,
		"SELECT id::text, username, acct_type FROM accounts WHERE supertokens_id = $1",
		db.WithResultOf(&info),
		supertokensID,
	)
	if err != nil {
		return models.AccountInfo{}, err
	}
	return info, nil
}
