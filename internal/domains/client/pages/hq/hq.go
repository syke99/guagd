package hq

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5"

	"guagd/internal/domains/client/pages/shared"
	"guagd/internal/pkg/db"
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

type HQUser struct {
	SupertokensID string
	Username      string
}

type LayoutItem struct {
	Component string `json:"component"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	W         int    `json:"w"`
	H         int    `json:"h"`
}

type HQMember struct {
	Username string
}

type HQPageData struct {
	Username        string
	IsOwner         bool
	IsAuthenticated bool
	MemberCount     int
	Members         []HQMember
	Layout          []LayoutItem
	SafeCSS         template.CSS
}

var defaultLayout = []LayoutItem{
	{Component: "hq-profile-header", X: 0, Y: 0, W: 12, H: 3},
	{Component: "member-grid", X: 0, Y: 3, W: 12, H: 8},
}

func (h *HQClient) getUserByUsername(ctx context.Context, username string) (*HQUser, error) {
	var user HQUser
	err := h.db.QueryRow(
		ctx,
		"SELECT supertokens_id, username FROM accounts WHERE username = $1 AND acct_type = 'club'",
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
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

func (h *HQClient) getHQLayout(ctx context.Context, supertokensID string) ([]LayoutItem, map[string]map[string]string, error) {
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
		return append([]LayoutItem{}, defaultLayout...), map[string]map[string]string{}, nil
	}

	var layout []LayoutItem
	if err := json.Unmarshal([]byte(layoutJSON), &layout); err != nil || len(layout) == 0 {
		layout = append([]LayoutItem{}, defaultLayout...)
	}

	var theme map[string]map[string]string
	if err := json.Unmarshal([]byte(themeJSON), &theme); err != nil {
		theme = map[string]map[string]string{}
	}

	return layout, theme, nil
}

func (h *HQClient) upsertLayout(ctx context.Context, supertokensID string, layout []LayoutItem) error {
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

func (h *HQClient) getMembers(ctx context.Context, clubID string) ([]HQMember, error) {
	var members []HQMember
	err := h.db.Query(
		ctx,
		`SELECT a.username FROM club_memberships cm
		 JOIN accounts a ON a.supertokens_id = cm.member_id
		 WHERE cm.club_id = $1
		 ORDER BY cm.created_at`,
		func(rows pgx.Rows) error {
			for rows.Next() {
				var m HQMember
				if err := rows.Scan(&m.Username); err != nil {
					return err
				}
				members = append(members, m)
			}
			return rows.Err()
		},
		clubID,
	)
	return members, err
}

func (h *HQClient) searchNonMembers(ctx context.Context, clubID, q string) ([]HQMember, error) {
	var results []HQMember
	err := h.db.Query(ctx,
		`SELECT username FROM accounts
		 WHERE username ILIKE $1
		 AND username IS NOT NULL
		 AND acct_type = 'driver'
		 AND supertokens_id NOT IN (
		   SELECT member_id FROM club_memberships WHERE club_id = $2
		 )
		 ORDER BY username LIMIT 10`,
		func(rows pgx.Rows) error {
			for rows.Next() {
				var m HQMember
				if err := rows.Scan(&m.Username); err != nil {
					return err
				}
				results = append(results, m)
			}
			return rows.Err()
		},
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

func buildThemeCSS(theme map[string]map[string]string) template.CSS {
	var sb strings.Builder
	if global, ok := theme["global"]; ok && len(global) > 0 {
		sb.WriteString(":root {\n")
		for prop, val := range global {
			if p := sanitizeCSSProp(prop); p != "" {
				if v := sanitizeCSSValue(val); v != "" {
					fmt.Fprintf(&sb, "  %s: %s;\n", p, v)
				}
			}
		}
		sb.WriteString("}\n")
	}
	for component, styles := range theme {
		if component == "global" || len(styles) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "#gs-%s {\n", component)
		for prop, val := range styles {
			if p := sanitizeCSSProp(prop); p != "" {
				if v := sanitizeCSSValue(val); v != "" {
					fmt.Fprintf(&sb, "  %s: %s;\n", p, v)
				}
			}
		}
		sb.WriteString("}\n")
	}
	return template.CSS(sb.String())
}

func sanitizeCSSProp(p string) string {
	for _, ch := range p {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '-' {
			return ""
		}
	}
	return p
}

func sanitizeCSSValue(v string) string {
	for _, dangerous := range []string{"(", ")", ";", "{", "}", "<", ">", `"`, "'", `\`, "\n", "\r"} {
		if strings.Contains(v, dangerous) {
			return ""
		}
	}
	return strings.TrimSpace(v)
}
