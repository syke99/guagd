package car

import (
	"context"
	"embed"
	"html/template"
	"strconv"
	"strings"
	"time"

	"guagd/internal/domains/client/pages/shared"
	"guagd/internal/pkg/db"
	"guagd/internal/pkg/models"
	"guagd/internal/pkg/sessions"
	"guagd/internal/pkg/storage"
)

//go:embed templates/*
var templates embed.FS

var carPageTemplate = template.Must(
	template.Must(template.New("").Funcs(carFuncMap).Parse(shared.NavTemplate)).
		ParseFS(templates, "templates/car.html"),
)

var carFuncMap = template.FuncMap{
	"trustLevel": func(count int) string {
		if count > 0 {
			return "documented"
		}
		return "reported"
	},
	"trustLabel": func(count int) string {
		if count > 0 {
			return "Documented"
		}
		return "Reported"
	},
	"formatModMeta": func(m models.Mod) string {
		var parts []string
		if m.InstallDate != "" {
			if s := fmtDateStr(m.InstallDate); s != "" {
				parts = append(parts, s)
			}
		}
		if m.MileageAtInstall != 0 {
			parts = append(parts, fmtMileage(m.MileageAtInstall))
		}
		if m.Cost != 0 {
			parts = append(parts, fmtCost(m.Cost))
		}
		return strings.Join(parts, " · ")
	},
	"formatMaintMeta": func(m models.Maintenance) string {
		var parts []string
		if m.ServiceDate != "" {
			if s := fmtDateStr(m.ServiceDate); s != "" {
				parts = append(parts, s)
			}
		}
		if m.Mileage != 0 {
			parts = append(parts, fmtMileage(m.Mileage))
		}
		if m.Cost != 0 {
			parts = append(parts, fmtCost(m.Cost))
		}
		return strings.Join(parts, " · ")
	},
	"docIcon": func(contentType string) string {
		switch {
		case strings.HasPrefix(contentType, "image/"):
			return "🖼"
		case contentType == "application/pdf":
			return "📄"
		default:
			return "📎"
		}
	},
}

func fmtDateStr(s string) string {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return s
	}
	return t.Format("Jan 2006")
}

func fmtMileage(n int) string { return fmtNumber(n) + " mi" }
func fmtCost(n int) string    { return "$" + fmtNumber(n) }

func fmtNumber(n int) string {
	s := strconv.Itoa(n)
	var b []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b = append(b, ',')
		}
		b = append(b, byte(c))
	}
	return string(b)
}

type CarPageClient struct {
	db       db.DB
	storage  *storage.Client
	sessions sessions.Getter
}

func NewCarPageClient(db db.DB, store *storage.Client, sg sessions.Getter) *CarPageClient {
	return &CarPageClient{db: db, storage: store, sessions: sg}
}

func (c *CarPageClient) getCarByShortID(ctx context.Context, shortID string) (models.Car, error) {
	var car models.Car
	err := c.db.QueryRow(ctx,
		`SELECT id::text,
		        year,
		        make,
		        model,
		        COALESCE(trim, '')    AS trim,
		        COALESCE(mileage, 0)  AS mileage,
		        ''                    AS object_key
		 FROM cars
		 WHERE id::text LIKE $1 || '%'
		 LIMIT 1`,
		db.WithResultOf(&car),
		shortID,
	)
	return car, err
}

func (c *CarPageClient) getOwner(ctx context.Context, carID string) (models.CarPageOwner, error) {
	var owner models.CarPageOwner
	err := c.db.QueryRow(ctx,
		`SELECT a.username,
		        a.acct_type,
		        COALESCE(ap.avatar_key, '') AS avatar_key
		 FROM cars ca
		 JOIN accounts a ON a.id = ca.owner_id
		 LEFT JOIN account_photos ap ON ap.account_id = a.id
		 WHERE ca.id = $1`,
		db.WithResultOf(&owner),
		carID,
	)
	if owner.AvatarKey != "" {
		owner.AvatarURL = c.storage.AccountPhotoURL(owner.AvatarKey)
	}
	return owner, err
}

func (c *CarPageClient) getPhotos(ctx context.Context, carID string) ([]models.CarPhoto, error) {
	var photos []models.CarPhoto
	err := c.db.Query(ctx,
		`SELECT id::text,
		        car_id::text AS car_id,
		        object_key,
		        is_primary
		 FROM car_photos
		 WHERE car_id = $1
		 ORDER BY is_primary DESC, uploaded_at ASC`,
		db.WithResultsOf(&photos),
		carID,
	)
	for i := range photos {
		photos[i].URL = c.storage.CarPhotoURL(photos[i].ObjectKey)
	}
	return photos, err
}

func (c *CarPageClient) getMods(ctx context.Context, carID string) ([]models.Mod, error) {
	var mods []models.Mod
	err := c.db.Query(ctx,
		`SELECT cm.id::text,
		        cm.car_id::text                     AS car_id,
		        cm.name,
		        cm.category,
		        COALESCE(cm.install_date::text, '')  AS install_date,
		        COALESCE(cm.mileage_at_install, 0)   AS mileage_at_install,
		        COALESCE(cm.cost, 0)                 AS cost,
		        COALESCE(cm.notes, '')               AS notes,
		        COUNT(mu.id)                         AS upload_count
		 FROM car_mods cm
		 LEFT JOIN car_uploads mu ON mu.mod_id = cm.id
		 WHERE cm.car_id = $1
		 GROUP BY cm.id
		 ORDER BY cm.created_at ASC`,
		db.WithResultsOf(&mods),
		carID,
	)
	return mods, err
}

func (c *CarPageClient) getMaintenance(ctx context.Context, carID string) ([]models.Maintenance, error) {
	var records []models.Maintenance
	err := c.db.Query(ctx,
		`SELECT cm.id::text,
		        cm.car_id::text                    AS car_id,
		        cm.name,
		        cm.category,
		        COALESCE(cm.service_date::text, '') AS service_date,
		        COALESCE(cm.mileage, 0)             AS mileage,
		        COALESCE(cm.cost, 0)                AS cost,
		        COALESCE(cm.notes, '')              AS notes,
		        COUNT(cu.id)                        AS upload_count
		 FROM car_maintenance cm
		 LEFT JOIN car_uploads cu ON cu.maintenance_id = cm.id
		 WHERE cm.car_id = $1
		 GROUP BY cm.id
		 ORDER BY cm.created_at ASC`,
		db.WithResultsOf(&records),
		carID,
	)
	return records, err
}

type rawDoc struct {
	ID              string `db:"id"`
	ObjectKey       string `db:"object_key"`
	Name            string `db:"name"`
	UploadType      string `db:"upload_type"`
	ContentType     string `db:"content_type"`
	ModName         string `db:"mod_name"`
	MaintenanceName string `db:"maintenance_name"`
}

func (c *CarPageClient) getDocs(ctx context.Context, carID string) ([]models.CarPageDoc, error) {
	var rows []rawDoc
	err := c.db.Query(ctx,
		`SELECT cu.id::text,
		        cu.object_key,
		        cu.name,
		        cu.upload_type,
		        cu.content_type,
		        COALESCE(cm.name, '')   AS mod_name,
		        COALESCE(mnt.name, '')  AS maintenance_name
		 FROM car_uploads cu
		 LEFT JOIN car_mods cm         ON cm.id  = cu.mod_id
		 LEFT JOIN car_maintenance mnt ON mnt.id = cu.maintenance_id
		 WHERE cm.car_id = $1 OR mnt.car_id = $1
		 ORDER BY cu.uploaded_at ASC`,
		db.WithResultsOf(&rows),
		carID,
	)
	if err != nil {
		return nil, err
	}

	docs := make([]models.CarPageDoc, 0, len(rows))
	for _, row := range rows {
		sourceType := "mod"
		sourceName := row.ModName
		if row.MaintenanceName != "" {
			sourceType = "maintenance"
			sourceName = row.MaintenanceName
		}
		docs = append(docs, models.CarPageDoc{
			ID:          row.ID,
			URL:         c.storage.CarFileURL(row.ObjectKey),
			Name:        row.Name,
			UploadType:  row.UploadType,
			ContentType: row.ContentType,
			SourceName:  sourceName,
			SourceType:  sourceType,
		})
	}
	return docs, nil
}
