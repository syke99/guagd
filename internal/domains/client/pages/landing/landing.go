package landing

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"guagd/internal/pkg/db"
	"guagd/internal/pkg/storage"
)

//go:embed landing.html
var landingHTML string

type LandingClient struct {
	db        db.DB
	store     *storage.Client
	publicURL string
	buildID   string
	garageID  string
	clubID    string
	tmpl      *template.Template
}

func NewLandingClient(database db.DB, store *storage.Client, publicURL, buildID, garageID, clubID string) (*LandingClient, error) {
	tmpl, err := template.New("landing").Parse(landingHTML)
	if err != nil {
		return nil, fmt.Errorf("parsing landing template: %w", err)
	}
	return &LandingClient{
		db:        database,
		store:     store,
		publicURL: publicURL,
		buildID:   buildID,
		garageID:  garageID,
		clubID:    clubID,
		tmpl:      tmpl,
	}, nil
}

// HeroData is the template data passed to landing.html.
type HeroData struct {
	Build  *HeroBuild
	Garage *HeroGarage
	Club   *HeroClub
}

type HeroBuild struct {
	CarName          string
	CoverPhotoURL    template.URL
	Photos           []template.URL
	ShareURL         string
	ModCount         int
	MaintenanceCount int
}

type HeroGarage struct {
	Username      string
	AvatarURL     template.URL
	CoverPhotoURL template.URL
	CarCount      int
	Cars          []HeroGarageCar
}

type HeroGarageCar struct {
	Name     string
	ThumbURL template.URL
}

type HeroClub struct {
	Username      string
	CoverPhotoURL template.URL
	MemberCount   int
	Members       []HeroClubMember
}

type HeroClubMember struct {
	Username  string
	CarCount  int
	AvatarURL template.URL
}

func (l *LandingClient) LandingPage(w http.ResponseWriter, r *http.Request) {
	data := HeroData{}

	if l.buildID != "" {
		if build, err := l.fetchBuild(r.Context(), l.buildID); err != nil {
			log.Printf("landing: fetch build %s: %s", l.buildID, err)
		} else {
			data.Build = build
		}
	}

	if l.garageID != "" {
		if garage, err := l.fetchGarage(r.Context(), l.garageID); err != nil {
			log.Printf("landing: fetch garage %s: %s", l.garageID, err)
		} else {
			data.Garage = garage
		}
	}

	if l.clubID != "" {
		if club, err := l.fetchClub(r.Context(), l.clubID); err != nil {
			log.Printf("landing: fetch club %s: %s", l.clubID, err)
		} else {
			data.Club = club
		}
	}

	w.Header().Set("Content-Type", "text/html")
	if err := l.tmpl.Execute(w, data); err != nil {
		log.Printf("landing: render: %s", err)
	}
}

// ── DB queries ─────────────────────────────────────────────────────────────────

type buildRow struct {
	Year     int    `db:"year"`
	Make     string `db:"make"`
	Model    string `db:"model"`
	CarID    string `db:"car_id"`
	Username string `db:"username"`
	CoverKey string `db:"cover_key"`
}

type photoRow struct {
	ObjectKey string `db:"object_key"`
}

func (l *LandingClient) fetchBuild(ctx context.Context, carID string) (*HeroBuild, error) {
	var row buildRow
	if err := l.db.QueryRow(ctx,
		`SELECT c.year, c.make, c.model, c.id::text AS car_id, a.username,
		        COALESCE(cp.object_key, '') AS cover_key
		 FROM cars c
		 JOIN accounts a ON a.id = c.owner_id
		 LEFT JOIN car_photos cp ON cp.car_id = c.id AND cp.is_primary = true
		 WHERE c.id = $1`,
		db.WithResultOf(&row),
		carID,
	); err != nil {
		return nil, err
	}

	build := &HeroBuild{
		CarName: fmt.Sprintf("%d %s %s", row.Year, row.Make, row.Model),
	}

	if row.CoverKey != "" {
		build.CoverPhotoURL = template.URL(l.store.CarPhotoURL(row.CoverKey))
	}

	shortID := row.CarID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	build.ShareURL = fmt.Sprintf("%s/garage/@%s?car=%s", l.publicURL, row.Username, shortID)

	var photos []photoRow
	_ = l.db.Query(ctx,
		`SELECT object_key FROM car_photos
		 WHERE car_id = $1 AND is_primary = false
		 ORDER BY uploaded_at ASC`,
		db.WithResultsOf(&photos),
		carID,
	)
	for i, p := range photos {
		if i >= 3 {
			break
		}
		build.Photos = append(build.Photos, template.URL(l.store.CarPhotoURL(p.ObjectKey)))
	}

	var modCount struct {
		Count int `db:"count"`
	}
	_ = l.db.QueryRow(ctx,
		`SELECT COUNT(*)::int AS count FROM car_mods WHERE car_id = $1`,
		db.WithResultOf(&modCount), carID,
	)
	build.ModCount = modCount.Count

	var maintCount struct {
		Count int `db:"count"`
	}
	_ = l.db.QueryRow(ctx,
		`SELECT COUNT(*)::int AS count FROM car_maintenance WHERE car_id = $1`,
		db.WithResultOf(&maintCount), carID,
	)
	build.MaintenanceCount = maintCount.Count

	return build, nil
}

type garageAccountRow struct {
	Username  string `db:"username"`
	AvatarKey string `db:"avatar_key"`
	BannerKey string `db:"banner_key"`
}

type garageCarRow struct {
	Year     int    `db:"year"`
	Make     string `db:"make"`
	Model    string `db:"model"`
	ThumbKey string `db:"thumb_key"`
}

func (l *LandingClient) fetchGarage(ctx context.Context, accountID string) (*HeroGarage, error) {
	var row garageAccountRow
	if err := l.db.QueryRow(ctx,
		`SELECT a.username,
		        COALESCE(ap.avatar_key, '') AS avatar_key,
		        COALESCE(ap.banner_key, '') AS banner_key
		 FROM accounts a
		 LEFT JOIN account_photos ap ON ap.account_id = a.id
		 WHERE a.id = $1`,
		db.WithResultOf(&row),
		accountID,
	); err != nil {
		return nil, err
	}

	garage := &HeroGarage{Username: row.Username}
	if row.AvatarKey != "" {
		garage.AvatarURL = template.URL(l.store.AccountPhotoURL(row.AvatarKey))
	}
	if row.BannerKey != "" {
		garage.CoverPhotoURL = template.URL(l.store.AccountPhotoURL(row.BannerKey))
	}

	var cars []garageCarRow
	_ = l.db.Query(ctx,
		`SELECT c.year, c.make, c.model,
		        COALESCE(cp.object_key, '') AS thumb_key
		 FROM cars c
		 LEFT JOIN car_photos cp ON cp.car_id = c.id AND cp.is_primary = true
		 WHERE c.owner_id = $1
		 ORDER BY c.created_at ASC
		 LIMIT 4`,
		db.WithResultsOf(&cars),
		accountID,
	)
	garage.CarCount = len(cars)
	for _, c := range cars {
		gc := HeroGarageCar{Name: fmt.Sprintf("%d %s %s", c.Year, c.Make, c.Model)}
		if c.ThumbKey != "" {
			gc.ThumbURL = template.URL(l.store.CarPhotoURL(c.ThumbKey))
		}
		garage.Cars = append(garage.Cars, gc)
	}

	return garage, nil
}

type clubAccountRow struct {
	Username  string `db:"username"`
	BannerKey string `db:"banner_key"`
}

type memberCountRow struct {
	Count int `db:"count"`
}

type memberRow struct {
	Username  string `db:"username"`
	CarCount  int    `db:"car_count"`
	AvatarKey string `db:"avatar_key"`
}

func (l *LandingClient) fetchClub(ctx context.Context, accountID string) (*HeroClub, error) {
	var row clubAccountRow
	if err := l.db.QueryRow(ctx,
		`SELECT a.username, COALESCE(ap.banner_key, '') AS banner_key
		 FROM accounts a
		 LEFT JOIN account_photos ap ON ap.account_id = a.id
		 WHERE a.id = $1 AND a.acct_type = 'club'`,
		db.WithResultOf(&row),
		accountID,
	); err != nil {
		return nil, err
	}

	club := &HeroClub{Username: row.Username}
	if row.BannerKey != "" {
		club.CoverPhotoURL = template.URL(l.store.AccountPhotoURL(row.BannerKey))
	}

	var countRow memberCountRow
	_ = l.db.QueryRow(ctx,
		`SELECT COUNT(*)::int AS count FROM club_memberships WHERE club_id = $1`,
		db.WithResultOf(&countRow),
		accountID,
	)
	club.MemberCount = countRow.Count

	var members []memberRow
	_ = l.db.Query(ctx,
		`SELECT a.username,
		        (SELECT COUNT(*)::int FROM cars WHERE owner_id = a.id) AS car_count,
		        COALESCE(ap.avatar_key, '') AS avatar_key
		 FROM club_memberships cm
		 JOIN accounts a ON a.id = cm.member_id
		 LEFT JOIN account_photos ap ON ap.account_id = a.id
		 WHERE cm.club_id = $1
		 ORDER BY cm.created_at ASC
		 LIMIT 4`,
		db.WithResultsOf(&members),
		accountID,
	)
	for _, m := range members {
		hm := HeroClubMember{Username: m.Username, CarCount: m.CarCount}
		if m.AvatarKey != "" {
			hm.AvatarURL = template.URL(l.store.AccountPhotoURL(m.AvatarKey))
		}
		club.Members = append(club.Members, hm)
	}

	return club, nil
}
