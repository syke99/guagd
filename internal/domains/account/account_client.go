package account

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"guagd/internal/pkg/db"
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

type accountInfo struct {
	Username string
	AcctType string
}

type searchResult struct {
	Username string
	AcctType string
}

func (u *accountClient) searchAccounts(ctx context.Context, q, acctType string) ([]searchResult, error) {
	var results []searchResult
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
	err := u.db.Query(ctx, query,
		func(rows pgx.Rows) error {
			for rows.Next() {
				var r searchResult
				if err := rows.Scan(&r.Username, &r.AcctType); err != nil {
					return err
				}
				results = append(results, r)
			}
			return rows.Err()
		},
		args...,
	)
	return results, err
}

func (u *accountClient) getAccountBySupertokensID(ctx context.Context, supertokensID string) (accountInfo, error) {
	var info accountInfo
	err := u.db.QueryRow(ctx,
		"SELECT username, acct_type FROM accounts WHERE supertokens_id = $1",
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return fmt.Errorf("account not found")
			}
			return rows.Scan(&info.Username, &info.AcctType)
		},
		supertokensID,
	)
	if err != nil {
		return accountInfo{}, err
	}
	return info, nil
}
