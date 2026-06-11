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

type Car struct {
	ID      string `json:"id"`
	Year    int    `json:"year"`
	Make    string `json:"make"`
	Model   string `json:"model"`
	Trim    string `json:"trim"`
	Mileage int    `json:"mileage"`
}

type GaragePageData struct {
	Username        string
	IsOwner         bool
	IsAuthenticated bool
	CarCount        int
	Cars            []Car
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

func (g *GarageClient) getCars(ctx context.Context, ownerID string) ([]Car, error) {
	var cars []Car
	err := g.db.Query(
		ctx,
		`SELECT id::text, year, make, model, COALESCE(trim, ''), COALESCE(mileage, 0)
		 FROM cars WHERE owner_id = $1 ORDER BY created_at`,
		func(rows pgx.Rows) error {
			for rows.Next() {
				var c Car
				if err := rows.Scan(&c.ID, &c.Year, &c.Make, &c.Model, &c.Trim, &c.Mileage); err != nil {
					return err
				}
				cars = append(cars, c)
			}
			return rows.Err()
		},
		ownerID,
	)
	return cars, err
}

func (g *GarageClient) addCar(ctx context.Context, ownerID string, car Car) (Car, error) {
	var created Car
	err := g.db.QueryRow(
		ctx,
		`INSERT INTO cars (owner_id, year, make, model, trim, mileage)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, 0))
		 RETURNING id::text, year, make, model, COALESCE(trim, ''), COALESCE(mileage, 0)`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&created.ID, &created.Year, &created.Make, &created.Model, &created.Trim, &created.Mileage)
		},
		ownerID, car.Year, car.Make, car.Model, car.Trim, car.Mileage,
	)
	return created, err
}

func (g *GarageClient) removeCar(ctx context.Context, ownerID, carID string) error {
	return g.db.Exec(
		ctx,
		`DELETE FROM cars WHERE id = $1 AND owner_id = $2`,
		carID, ownerID,
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
