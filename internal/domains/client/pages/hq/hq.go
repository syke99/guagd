package hq

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"

	"github.com/jackc/pgx/v5"

	"guagd/internal/domains/client/pages/shared"
	"guagd/internal/pkg/db"
	"guagd/internal/pkg/models"
)

//go:embed templates/*
var templates embed.FS

var hqTemplate = template.Must(
	template.Must(template.New("").Parse(shared.NavTemplate)).
		ParseFS(templates, "templates/hq.html"),
)

type HQClient struct {
	db db.DB
}

func NewHQClient(db db.DB) *HQClient {
	return &HQClient{db: db}
}

var defaultLayout = []models.LayoutItem{
	{Component: "hq-profile-header", X: 0, Y: 0, W: 12, H: 3},
	{Component: "member-grid", X: 0, Y: 3, W: 12, H: 8},
}

func (h *HQClient) getUserByUsername(ctx context.Context, username string) (*models.HQUser, error) {
	var user models.HQUser
	err := h.db.QueryRow(
		ctx,
		"SELECT supertokens_id, username FROM accounts WHERE username = $1 AND acct_type = 'club'",
		db.WithResultOf(&user),
		username,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (h *HQClient) getHQLayout(ctx context.Context, supertokensID string) ([]models.LayoutItem, map[string]map[string]string, error) {
	var layoutJSON, themeJSON string
	err := h.db.QueryRow(
		ctx,
		"SELECT layout::text, theme::text FROM hq_layouts WHERE supertokens_id = $1",
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&layoutJSON, &themeJSON)
		},
		supertokensID,
	)
	if err != nil {
		return append([]models.LayoutItem{}, defaultLayout...), map[string]map[string]string{}, nil
	}

	var layout []models.LayoutItem
	if err := json.Unmarshal([]byte(layoutJSON), &layout); err != nil || len(layout) == 0 {
		layout = append([]models.LayoutItem{}, defaultLayout...)
	}

	var theme map[string]map[string]string
	if err := json.Unmarshal([]byte(themeJSON), &theme); err != nil {
		theme = map[string]map[string]string{}
	}

	return layout, theme, nil
}

func (h *HQClient) upsertLayout(ctx context.Context, supertokensID string, layout []models.LayoutItem) error {
	b, err := json.Marshal(layout)
	if err != nil {
		return err
	}
	return h.db.Exec(
		ctx,
		`INSERT INTO hq_layouts (supertokens_id, layout)
		 VALUES ($1, $2::jsonb)
		 ON CONFLICT (supertokens_id) DO UPDATE
		 SET layout = EXCLUDED.layout, updated_at = now()`,
		supertokensID, string(b),
	)
}

func (h *HQClient) upsertTheme(ctx context.Context, supertokensID string, theme map[string]map[string]string) error {
	b, err := json.Marshal(theme)
	if err != nil {
		return err
	}
	return h.db.Exec(
		ctx,
		`INSERT INTO hq_layouts (supertokens_id, theme)
		 VALUES ($1, $2::jsonb)
		 ON CONFLICT (supertokens_id) DO UPDATE
		 SET theme = EXCLUDED.theme, updated_at = now()`,
		supertokensID, string(b),
	)
}

func (h *HQClient) getMembers(ctx context.Context, clubID string) ([]models.HQMember, error) {
	var members []models.HQMember
	err := h.db.Query(
		ctx,
		`SELECT a.username FROM club_memberships cm
		 JOIN accounts a ON a.supertokens_id = cm.member_id
		 WHERE cm.club_id = $1
		 ORDER BY cm.created_at`,
		db.WithResultsOf(&members),
		clubID,
	)
	return members, err
}

func (h *HQClient) searchNonMembers(ctx context.Context, clubID, q string) ([]models.HQMember, error) {
	var results []models.HQMember
	err := h.db.Query(ctx,
		`SELECT username FROM accounts
		 WHERE username ILIKE $1
		 AND username IS NOT NULL
		 AND acct_type = 'driver'
		 AND supertokens_id NOT IN (
		   SELECT member_id FROM club_memberships WHERE club_id = $2
		 )
		 ORDER BY username LIMIT 10`,
		db.WithResultsOf(&results),
		"%"+q+"%", clubID,
	)
	return results, err
}

func (h *HQClient) addMember(ctx context.Context, clubID, memberUsername string) error {
	return h.db.Exec(
		ctx,
		`INSERT INTO club_memberships (club_id, member_id)
		 SELECT $1, supertokens_id FROM accounts
		 WHERE username = $2 AND acct_type = 'driver'
		 ON CONFLICT DO NOTHING`,
		clubID, memberUsername,
	)
}

func (h *HQClient) removeMember(ctx context.Context, clubID, memberUsername string) error {
	return h.db.Exec(
		ctx,
		`DELETE FROM club_memberships
		 WHERE club_id = $1
		 AND member_id = (SELECT supertokens_id FROM accounts WHERE username = $2)`,
		clubID, memberUsername,
	)
}

