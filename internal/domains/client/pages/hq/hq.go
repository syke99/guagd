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
	"guagd/internal/pkg/sessions"
	"guagd/internal/pkg/storage"
)

//go:embed templates/*
var templates embed.FS

var hqTemplate = template.Must(
	template.Must(template.New("").Parse(shared.NavTemplate)).
		ParseFS(templates, "templates/hq.html"),
)

var hqMemberCardTemplate = template.Must(
	template.New("").ParseFS(templates, "templates/hq-member-card-fragment.html"),
)

type HQClient struct {
	db       db.DB
	storage  *storage.Client
	sessions sessions.Getter
}

func NewHQClient(db db.DB, store *storage.Client, sg sessions.Getter) *HQClient {
	return &HQClient{db: db, storage: store, sessions: sg}
}

var defaultLayout = []models.LayoutItem{
	{Component: "hq-profile-header", X: 0, Y: 0, W: 12, H: 3},
	{Component: "member-grid", X: 0, Y: 3, W: 12, H: 8},
}

func (h *HQClient) getAccountIDBySupertokensID(ctx context.Context, supertokensID string) (string, error) {
	var id string
	err := h.db.QueryRow(ctx,
		`SELECT id::text FROM accounts WHERE supertokens_id = $1`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&id)
		},
		supertokensID,
	)
	return id, err
}

func (h *HQClient) getUserByUsername(ctx context.Context, username string) (*models.HQUser, error) {
	var user models.HQUser
	err := h.db.QueryRow(
		ctx,
		"SELECT id::text AS account_id, supertokens_id, username FROM accounts WHERE username = $1 AND acct_type = 'club'",
		db.WithResultOf(&user),
		username,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (h *HQClient) getHQLayout(ctx context.Context, accountID string) ([]models.LayoutItem, map[string]map[string]string, string, error) {
	var layoutJSON, themeJSON string
	err := h.db.QueryRow(
		ctx,
		"SELECT layout::text, theme::text FROM hq_layouts WHERE account_id = $1",
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&layoutJSON, &themeJSON)
		},
		accountID,
	)
	if err != nil {
		return append([]models.LayoutItem{}, defaultLayout...), map[string]map[string]string{}, "", nil
	}

	var layout []models.LayoutItem
	if err := json.Unmarshal([]byte(layoutJSON), &layout); err != nil || len(layout) == 0 {
		layout = append([]models.LayoutItem{}, defaultLayout...)
	}

	var theme map[string]map[string]string
	if err := json.Unmarshal([]byte(themeJSON), &theme); err != nil {
		theme = map[string]map[string]string{}
	}

	var bannerKey string
	_ = h.db.QueryRow(
		ctx,
		"SELECT COALESCE(banner_key, '') FROM account_photos WHERE account_id = $1",
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return nil
			}
			return rows.Scan(&bannerKey)
		},
		accountID,
	)

	coverPhotoURL := ""
	if bannerKey != "" {
		coverPhotoURL = h.storage.AccountPhotoURL(bannerKey)
	}

	return layout, theme, coverPhotoURL, nil
}

func (h *HQClient) saveCoverPhoto(ctx context.Context, accountID, objectKey string) error {
	return h.db.Exec(
		ctx,
		`INSERT INTO account_photos (account_id, banner_key, updated_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (account_id) DO UPDATE
		 SET banner_key = EXCLUDED.banner_key, updated_at = now()`,
		accountID, objectKey,
	)
}

func (h *HQClient) removeCoverPhoto(ctx context.Context, accountID string) error {
	return h.db.Exec(
		ctx,
		`UPDATE account_photos SET banner_key = NULL, updated_at = now() WHERE account_id = $1`,
		accountID,
	)
}

func (h *HQClient) upsertLayout(ctx context.Context, accountID string, layout []models.LayoutItem) error {
	b, err := json.Marshal(layout)
	if err != nil {
		return err
	}
	return h.db.Exec(
		ctx,
		`INSERT INTO hq_layouts (account_id, layout)
		 VALUES ($1, $2::jsonb)
		 ON CONFLICT (account_id) DO UPDATE
		 SET layout = EXCLUDED.layout, updated_at = now()`,
		accountID, string(b),
	)
}

func (h *HQClient) upsertTheme(ctx context.Context, accountID string, theme map[string]map[string]string) error {
	b, err := json.Marshal(theme)
	if err != nil {
		return err
	}
	return h.db.Exec(
		ctx,
		`INSERT INTO hq_layouts (account_id, theme)
		 VALUES ($1, $2::jsonb)
		 ON CONFLICT (account_id) DO UPDATE
		 SET theme = EXCLUDED.theme, updated_at = now()`,
		accountID, string(b),
	)
}

func (h *HQClient) getMembers(ctx context.Context, clubAccountID string) ([]models.HQMember, error) {
	var members []models.HQMember
	err := h.db.Query(
		ctx,
		`SELECT a.username,
		        COALESCE(ap.banner_key, '') AS cover_photo_key,
		        COALESCE(ap.avatar_key, '') AS avatar_key
		 FROM club_memberships cm
		 JOIN accounts a ON a.id = cm.member_id
		 LEFT JOIN account_photos ap ON ap.account_id = cm.member_id
		 WHERE cm.club_id = $1
		 ORDER BY cm.created_at`,
		db.WithResultsOf(&members),
		clubAccountID,
	)
	for i := range members {
		if members[i].CoverPhotoKey != "" {
			members[i].CoverPhotoURL = h.storage.AccountPhotoURL(members[i].CoverPhotoKey)
		}
		if members[i].AvatarKey != "" {
			members[i].AvatarURL = h.storage.AccountPhotoURL(members[i].AvatarKey)
		}
	}
	return members, err
}

func (h *HQClient) getMemberByUsername(ctx context.Context, username string) (models.HQMember, error) {
	var member models.HQMember
	err := h.db.QueryRow(ctx,
		`SELECT a.username,
		        COALESCE(ap.banner_key, '') AS cover_photo_key,
		        COALESCE(ap.avatar_key, '') AS avatar_key
		 FROM accounts a
		 LEFT JOIN account_photos ap ON ap.account_id = a.id
		 WHERE a.username = $1`,
		db.WithResultOf(&member),
		username,
	)
	if err != nil {
		return models.HQMember{}, err
	}
	if member.CoverPhotoKey != "" {
		member.CoverPhotoURL = h.storage.AccountPhotoURL(member.CoverPhotoKey)
	}
	if member.AvatarKey != "" {
		member.AvatarURL = h.storage.AccountPhotoURL(member.AvatarKey)
	}
	return member, nil
}

func (h *HQClient) searchNonMembers(ctx context.Context, clubAccountID, q string) ([]models.HQMember, error) {
	var results []models.HQMember
	err := h.db.Query(ctx,
		`SELECT a.username,
		        COALESCE(ap.banner_key, '') AS cover_photo_key,
		        COALESCE(ap.avatar_key, '') AS avatar_key
		 FROM accounts a
		 LEFT JOIN account_photos ap ON ap.account_id = a.id
		 WHERE a.username ILIKE $1
		 AND a.username IS NOT NULL
		 AND a.acct_type = 'driver'
		 AND a.id NOT IN (
		   SELECT member_id FROM club_memberships WHERE club_id = $2
		 )
		 ORDER BY a.username LIMIT 10`,
		db.WithResultsOf(&results),
		"%"+q+"%", clubAccountID,
	)
	return results, err
}


func (h *HQClient) addMember(ctx context.Context, clubAccountID, memberUsername string) error {
	return h.db.Exec(
		ctx,
		`INSERT INTO club_memberships (club_id, member_id)
		 SELECT $1, id FROM accounts
		 WHERE username = $2 AND acct_type = 'driver'
		 ON CONFLICT DO NOTHING`,
		clubAccountID, memberUsername,
	)
}

func (h *HQClient) removeMember(ctx context.Context, clubAccountID, memberUsername string) error {
	return h.db.Exec(
		ctx,
		`DELETE FROM club_memberships
		 WHERE club_id = $1
		 AND member_id = (SELECT id FROM accounts WHERE username = $2)`,
		clubAccountID, memberUsername,
	)
}
