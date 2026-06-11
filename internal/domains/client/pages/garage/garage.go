package garage

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

var garageTemplate = template.Must(
	template.Must(template.New("").Parse(shared.NavTemplate)).
		ParseFS(templates, "templates/garage.html"),
)

type GarageClient struct {
	db db.DB
}

func NewGarageClient(db db.DB) *GarageClient {
	return &GarageClient{db: db}
}

type GarageUser struct {
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

type GaragePageData struct {
	Username        string
	IsOwner         bool
	IsAuthenticated bool
	Layout          []LayoutItem
	SafeCSS         template.CSS
}

var defaultLayout = []LayoutItem{
	{Component: "profile-header", X: 0, Y: 0, W: 12, H: 3},
	{Component: "car-list", X: 0, Y: 3, W: 12, H: 6},
}

func (g *GarageClient) getUserByUsername(ctx context.Context, username string) (*GarageUser, error) {
	var user GarageUser
	err := g.db.QueryRow(
		ctx,
		"SELECT supertokens_id, username FROM accounts WHERE username = $1",
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

func (g *GarageClient) getGarageLayout(ctx context.Context, supertokensID string) ([]LayoutItem, map[string]map[string]string, error) {
	var layoutJSON, themeJSON string
	err := g.db.QueryRow(
		ctx,
		"SELECT layout::text, theme::text FROM garage_layouts WHERE supertokens_id = $1",
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

func (g *GarageClient) upsertLayout(ctx context.Context, supertokensID string, layout []LayoutItem) error {
	b, err := json.Marshal(layout)
	if err != nil {
		return err
	}
	return g.db.Exec(
		ctx,
		`INSERT INTO garage_layouts (supertokens_id, layout)
		 VALUES ($1, $2::jsonb)
		 ON CONFLICT (supertokens_id) DO UPDATE
		 SET layout = EXCLUDED.layout`,
		supertokensID, string(b),
	)
}

func (g *GarageClient) upsertTheme(ctx context.Context, supertokensID string, theme map[string]map[string]string) error {
	b, err := json.Marshal(theme)
	if err != nil {
		return err
	}
	return g.db.Exec(
		ctx,
		`INSERT INTO garage_layouts (supertokens_id, theme)
		 VALUES ($1, $2::jsonb)
		 ON CONFLICT (supertokens_id) DO UPDATE
		 SET theme = EXCLUDED.theme`,
		supertokensID, string(b),
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
