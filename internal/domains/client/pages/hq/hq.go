package hq

import (
	"context"
	"embed"
	"fmt"
	"html/template"

	"github.com/jackc/pgx/v5"

	"guagd/internal/pkg/db"
)

//go:embed templates/*
var templates embed.FS

var hqTemplate = template.Must(template.ParseFS(templates, "templates/hq.html"))

type HQClient struct {
	db db.DB
}

func NewHQClient(db db.DB) *HQClient {
	return &HQClient{db: db}
}

type HQUser struct {
	SupertokensID string
	Username      string
}

type HQPageData struct {
	Username string
	IsOwner  bool
}

func (h *HQClient) getUserByUsername(ctx context.Context, username string) (*HQUser, error) {
	var user HQUser
	err := h.db.QueryRow(
		ctx,
		"SELECT supertokens_id, username FROM accounts WHERE username = $1",
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return fmt.Errorf("user not found")
			}
			return rows.Scan(&user.SupertokensID, &user.Username)
		},
		username,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
