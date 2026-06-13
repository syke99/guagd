package garage

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"

	"github.com/jackc/pgx/v5"

	"guagd/internal/domains/client/pages/shared"
	"guagd/internal/pkg/db"
	"guagd/internal/pkg/models"
	"guagd/internal/pkg/sessions"
	"guagd/internal/pkg/storage"
)

//go:embed templates/*
var templates embed.FS

var garageTemplate = template.Must(
	template.Must(template.New("").Parse(shared.NavTemplate)).
		ParseFS(templates, "templates/garage.html"),
)

type GarageClient struct {
	db       db.DB
	storage  *storage.Client
	sessions sessions.Getter
}

func NewGarageClient(db db.DB, store *storage.Client, sg sessions.Getter) *GarageClient {
	return &GarageClient{db: db, storage: store, sessions: sg}
}

var defaultLayout = []models.LayoutItem{
	{Component: "profile-header", X: 0, Y: 0, W: 12, H: 3},
	{Component: "car-list", X: 0, Y: 3, W: 12, H: 6},
}

func (g *GarageClient) getUserByUsername(ctx context.Context, username string) (*models.GarageUser, error) {
	var user models.GarageUser
	err := g.db.QueryRow(
		ctx,
		"SELECT id::text AS account_id, supertokens_id, username FROM accounts WHERE username = $1",
		db.WithResultOf(&user),
		username,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (g *GarageClient) getGarageLayout(ctx context.Context, accountID string) ([]models.LayoutItem, map[string]map[string]string, string, string, error) {
	var layoutJSON, themeJSON string
	err := g.db.QueryRow(
		ctx,
		"SELECT layout::text, theme::text FROM garage_layouts WHERE account_id = $1",
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&layoutJSON, &themeJSON)
		},
		accountID,
	)
	if err != nil {
		return append([]models.LayoutItem{}, defaultLayout...), map[string]map[string]string{}, "", "", nil
	}

	var layout []models.LayoutItem
	if err := json.Unmarshal([]byte(layoutJSON), &layout); err != nil || len(layout) == 0 {
		layout = append([]models.LayoutItem{}, defaultLayout...)
	}

	var theme map[string]map[string]string
	if err := json.Unmarshal([]byte(themeJSON), &theme); err != nil {
		theme = map[string]map[string]string{}
	}

	var bannerKey, avatarKey string
	_ = g.db.QueryRow(
		ctx,
		"SELECT COALESCE(banner_key, ''), COALESCE(avatar_key, '') FROM account_photos WHERE account_id = $1",
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return nil
			}
			return rows.Scan(&bannerKey, &avatarKey)
		},
		accountID,
	)

	coverPhotoURL := ""
	if bannerKey != "" {
		coverPhotoURL = g.storage.AccountPhotoURL(bannerKey)
	}
	avatarURL := ""
	if avatarKey != "" {
		avatarURL = g.storage.AccountPhotoURL(avatarKey)
	}

	return layout, theme, coverPhotoURL, avatarURL, nil
}

func (g *GarageClient) upsertLayout(ctx context.Context, accountID string, layout []models.LayoutItem) error {
	b, err := json.Marshal(layout)
	if err != nil {
		return err
	}
	return g.db.Exec(
		ctx,
		`INSERT INTO garage_layouts (account_id, layout)
		 VALUES ($1, $2::jsonb)
		 ON CONFLICT (account_id) DO UPDATE
		 SET layout = EXCLUDED.layout`,
		accountID, string(b),
	)
}

func (g *GarageClient) upsertTheme(ctx context.Context, accountID string, theme map[string]map[string]string) error {
	b, err := json.Marshal(theme)
	if err != nil {
		return err
	}
	return g.db.Exec(
		ctx,
		`INSERT INTO garage_layouts (account_id, theme)
		 VALUES ($1, $2::jsonb)
		 ON CONFLICT (account_id) DO UPDATE
		 SET theme = EXCLUDED.theme`,
		accountID, string(b),
	)
}

func (g *GarageClient) getCars(ctx context.Context, accountID string) ([]models.Car, error) {
	var cars []models.Car
	err := g.db.Query(
		ctx,
		`SELECT c.id::text,
		        c.year,
		        c.make,
		        c.model,
		        COALESCE(c.trim, '')        AS trim,
		        COALESCE(c.mileage, 0)      AS mileage,
		        COALESCE(p.object_key, '')  AS object_key
		 FROM cars c
		 LEFT JOIN car_photos p ON p.car_id = c.id AND p.is_primary = true
		 WHERE c.owner_id = $1
		 ORDER BY c.created_at`,
		db.WithResultsOf(&cars),
		accountID,
	)
	for i := range cars {
		if cars[i].ObjectKey != "" {
			cars[i].PrimaryPhotoURL = g.storage.CarPhotoURL(cars[i].ObjectKey)
		}
	}
	return cars, err
}

func (g *GarageClient) addCar(ctx context.Context, accountID string, car models.Car) (models.Car, error) {
	var created models.Car
	err := g.db.QueryRow(
		ctx,
		`INSERT INTO cars (owner_id, year, make, model, trim, mileage)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, 0))
		 RETURNING id::text,
		           year,
		           make,
		           model,
		           COALESCE(trim, '')   AS trim,
		           COALESCE(mileage, 0) AS mileage,
		           ''                   AS object_key`,
		db.WithResultOf(&created),
		accountID, car.Year, car.Make, car.Model, car.Trim, car.Mileage,
	)
	return created, err
}

func (g *GarageClient) removeCar(ctx context.Context, accountID, carID string) error {
	return g.db.Exec(
		ctx,
		`DELETE FROM cars WHERE id = $1 AND owner_id = $2`,
		carID, accountID,
	)
}

func (g *GarageClient) updateCar(ctx context.Context, accountID string, car models.Car) (models.Car, error) {
	var updated models.Car
	err := g.db.QueryRow(
		ctx,
		`UPDATE cars
		 SET year = $3, make = $4, model = $5, trim = NULLIF($6, ''), mileage = NULLIF($7, 0)
		 WHERE id = $1 AND owner_id = $2
		 RETURNING id::text,
		           year,
		           make,
		           model,
		           COALESCE(trim, '')   AS trim,
		           COALESCE(mileage, 0) AS mileage,
		           ''                   AS object_key`,
		db.WithResultOf(&updated),
		car.ID, accountID, car.Year, car.Make, car.Model, car.Trim, car.Mileage,
	)
	return updated, err
}

func (g *GarageClient) getCarPhotos(ctx context.Context, carID string) ([]models.CarPhoto, error) {
	var photos []models.CarPhoto
	err := g.db.Query(
		ctx,
		`SELECT id::text,
		        car_id::text AS car_id,
		        object_key,
		        is_primary
		 FROM car_photos WHERE car_id = $1
		 ORDER BY is_primary DESC, uploaded_at ASC`,
		db.WithResultsOf(&photos),
		carID,
	)
	for i := range photos {
		photos[i].URL = g.storage.CarPhotoURL(photos[i].ObjectKey)
	}
	return photos, err
}

func (g *GarageClient) addCarPhoto(ctx context.Context, accountID, carID, objectKey string, isPrimary bool) (models.CarPhoto, error) {
	var owned int
	err := g.db.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM cars WHERE id = $1 AND owner_id = $2`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&owned)
		},
		carID, accountID,
	)
	if err != nil || owned == 0 {
		return models.CarPhoto{}, fmt.Errorf("car not found")
	}

	if isPrimary {
		_ = g.db.Exec(ctx, `UPDATE car_photos SET is_primary = false WHERE car_id = $1`, carID)
	}

	var photo models.CarPhoto
	err = g.db.QueryRow(
		ctx,
		`INSERT INTO car_photos (car_id, object_key, is_primary)
		 VALUES ($1, $2, $3)
		 RETURNING id::text,
		           car_id::text AS car_id,
		           object_key,
		           is_primary`,
		db.WithResultOf(&photo),
		carID, objectKey, isPrimary,
	)
	return photo, err
}

func (g *GarageClient) removeCarPhoto(ctx context.Context, accountID, photoID string) error {
	var objectKey string
	err := g.db.QueryRow(
		ctx,
		`DELETE FROM car_photos
		 USING cars
		 WHERE car_photos.id = $1
		   AND car_photos.car_id = cars.id
		   AND cars.owner_id = $2
		 RETURNING car_photos.object_key`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return nil
			}
			return rows.Scan(&objectKey)
		},
		photoID, accountID,
	)
	if err != nil {
		return err
	}
	if objectKey != "" {
		if err := g.storage.DeleteCarPhoto(ctx, objectKey); err != nil {
			log.Printf("removeCarPhoto: delete from R2: %s", err)
		}
	}
	return nil
}

func (g *GarageClient) setCarPhotoPrimary(ctx context.Context, accountID, carID, photoID string) error {
	var owned int
	err := g.db.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM car_photos p
		 JOIN cars c ON c.id = p.car_id
		 WHERE p.id = $1 AND c.id = $2 AND c.owner_id = $3`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&owned)
		},
		photoID, carID, accountID,
	)
	if err != nil || owned == 0 {
		return fmt.Errorf("photo not found")
	}

	if err := g.db.Exec(ctx, `UPDATE car_photos SET is_primary = false WHERE car_id = $1`, carID); err != nil {
		return err
	}
	return g.db.Exec(ctx, `UPDATE car_photos SET is_primary = true WHERE id = $1`, photoID)
}

func (g *GarageClient) saveAvatar(ctx context.Context, accountID, objectKey string) error {
	return g.db.Exec(
		ctx,
		`INSERT INTO account_photos (account_id, avatar_key, updated_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (account_id) DO UPDATE
		 SET avatar_key = EXCLUDED.avatar_key, updated_at = now()`,
		accountID, objectKey,
	)
}

func (g *GarageClient) removeAvatar(ctx context.Context, accountID string) error {
	return g.db.Exec(
		ctx,
		`UPDATE account_photos SET avatar_key = NULL, updated_at = now() WHERE account_id = $1`,
		accountID,
	)
}

func (g *GarageClient) saveCoverPhoto(ctx context.Context, accountID, objectKey string) error {
	return g.db.Exec(
		ctx,
		`INSERT INTO account_photos (account_id, banner_key, updated_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (account_id) DO UPDATE
		 SET banner_key = EXCLUDED.banner_key, updated_at = now()`,
		accountID, objectKey,
	)
}

func (g *GarageClient) removeCoverPhoto(ctx context.Context, accountID string) error {
	return g.db.Exec(
		ctx,
		`UPDATE account_photos SET banner_key = NULL, updated_at = now() WHERE account_id = $1`,
		accountID,
	)
}

func (g *GarageClient) getMods(ctx context.Context, carID string) ([]models.Mod, error) {
	var mods []models.Mod
	err := g.db.Query(ctx,
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

func (g *GarageClient) getCarUploads(ctx context.Context, modID string) ([]models.CarUpload, error) {
	var uploads []models.CarUpload
	err := g.db.Query(ctx,
		`SELECT id::text, mod_id::text AS mod_id, object_key, name, upload_type, content_type
		 FROM car_uploads WHERE mod_id = $1 ORDER BY uploaded_at ASC`,
		db.WithResultsOf(&uploads),
		modID,
	)
	for i := range uploads {
		uploads[i].URL = g.storage.CarFileURL(uploads[i].ObjectKey)
	}
	return uploads, err
}

func (g *GarageClient) addCarUpload(ctx context.Context, accountID, modID, objectKey, name, uploadType, contentType string) (models.CarUpload, error) {
	var owned int
	err := g.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM car_mods cm
		 JOIN cars c ON c.id = cm.car_id
		 WHERE cm.id = $1 AND c.owner_id = $2`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&owned)
		},
		modID, accountID,
	)
	if err != nil || owned == 0 {
		return models.CarUpload{}, fmt.Errorf("mod not found")
	}

	var upload models.CarUpload
	err = g.db.QueryRow(ctx,
		`INSERT INTO car_uploads (mod_id, object_key, name, upload_type, content_type)
		 VALUES ($1::uuid, $2, $3, $4, $5)
		 RETURNING id::text, mod_id::text AS mod_id, object_key, name, upload_type, content_type`,
		db.WithResultOf(&upload),
		modID, objectKey, name, uploadType, contentType,
	)
	if err == nil {
		upload.URL = g.storage.CarFileURL(upload.ObjectKey)
	}
	return upload, err
}

func (g *GarageClient) removeCarUpload(ctx context.Context, accountID, uploadID string) error {
	var objectKey string
	err := g.db.QueryRow(ctx,
		`DELETE FROM car_uploads
		 WHERE id = $1::uuid
		   AND (
		     mod_id IN (SELECT cm.id FROM car_mods cm JOIN cars c ON c.id = cm.car_id WHERE c.owner_id = $2)
		     OR maintenance_id IN (SELECT cm.id FROM car_maintenance cm JOIN cars c ON c.id = cm.car_id WHERE c.owner_id = $2)
		   )
		 RETURNING object_key`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return nil
			}
			return rows.Scan(&objectKey)
		},
		uploadID, accountID,
	)
	if err != nil {
		return err
	}
	if objectKey != "" {
		if err := g.storage.DeleteCarFile(ctx, objectKey); err != nil {
			log.Printf("removeCarUpload: delete from R2: %s", err)
		}
	}
	return nil
}

func (g *GarageClient) addMod(ctx context.Context, accountID, carID string, m models.Mod) (models.Mod, error) {
	var owned int
	err := g.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM cars WHERE id = $1 AND owner_id = $2`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&owned)
		},
		carID, accountID,
	)
	if err != nil || owned == 0 {
		return models.Mod{}, fmt.Errorf("car not found")
	}

	// Normalize month-only dates from the browser's <input type="month">
	if len(m.InstallDate) == 7 {
		m.InstallDate = m.InstallDate + "-01"
	}

	var created models.Mod
	err = g.db.QueryRow(ctx,
		`INSERT INTO car_mods (car_id, name, category, install_date, mileage_at_install, cost, notes)
		 VALUES ($1, $2, $3, NULLIF($4,'')::date, NULLIF($5,0), NULLIF($6,0), NULLIF($7,''))
		 RETURNING id::text,
		           car_id::text                     AS car_id,
		           name,
		           category,
		           COALESCE(install_date::text, '')  AS install_date,
		           COALESCE(mileage_at_install, 0)   AS mileage_at_install,
		           COALESCE(cost, 0)                 AS cost,
		           COALESCE(notes, '')               AS notes,
		           0                                 AS upload_count`,
		db.WithResultOf(&created),
		carID, m.Name, m.Category, m.InstallDate, m.MileageAtInstall, m.Cost, m.Notes,
	)
	return created, err
}

func (g *GarageClient) removeMod(ctx context.Context, accountID, modID string) error {
	return g.db.Exec(ctx,
		`DELETE FROM car_mods cm
		 USING cars
		 WHERE cm.id = $1
		   AND cm.car_id = cars.id
		   AND cars.owner_id = $2`,
		modID, accountID,
	)
}

func (g *GarageClient) getMaintenance(ctx context.Context, carID string) ([]models.Maintenance, error) {
	var records []models.Maintenance
	err := g.db.Query(ctx,
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

func (g *GarageClient) addMaintenance(ctx context.Context, accountID, carID string, m models.Maintenance) (models.Maintenance, error) {
	var owned int
	err := g.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM cars WHERE id = $1 AND owner_id = $2`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&owned)
		},
		carID, accountID,
	)
	if err != nil || owned == 0 {
		return models.Maintenance{}, fmt.Errorf("car not found")
	}

	var created models.Maintenance
	err = g.db.QueryRow(ctx,
		`INSERT INTO car_maintenance (car_id, name, category, service_date, mileage, cost, notes)
		 VALUES ($1, $2, $3, NULLIF($4,'')::date, NULLIF($5,0), NULLIF($6,0), NULLIF($7,''))
		 RETURNING id::text,
		           car_id::text                    AS car_id,
		           name,
		           category,
		           COALESCE(service_date::text, '') AS service_date,
		           COALESCE(mileage, 0)             AS mileage,
		           COALESCE(cost, 0)                AS cost,
		           COALESCE(notes, '')              AS notes,
		           0                               AS upload_count`,
		db.WithResultOf(&created),
		carID, m.Name, m.Category, m.ServiceDate, m.Mileage, m.Cost, m.Notes,
	)
	return created, err
}

func (g *GarageClient) removeMaintenance(ctx context.Context, accountID, recordID string) error {
	return g.db.Exec(ctx,
		`DELETE FROM car_maintenance cm
		 USING cars
		 WHERE cm.id = $1
		   AND cm.car_id = cars.id
		   AND cars.owner_id = $2`,
		recordID, accountID,
	)
}

func (g *GarageClient) getMaintenanceUploads(ctx context.Context, maintenanceID string) ([]models.CarUpload, error) {
	var uploads []models.CarUpload
	err := g.db.Query(ctx,
		`SELECT id::text, maintenance_id::text AS mod_id, object_key, name, upload_type, content_type
		 FROM car_uploads WHERE maintenance_id = $1 ORDER BY uploaded_at ASC`,
		db.WithResultsOf(&uploads),
		maintenanceID,
	)
	for i := range uploads {
		uploads[i].URL = g.storage.CarFileURL(uploads[i].ObjectKey)
	}
	return uploads, err
}

func (g *GarageClient) addMaintenanceUpload(ctx context.Context, accountID, maintenanceID, objectKey, name, uploadType, contentType string) (models.CarUpload, error) {
	var owned int
	err := g.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM car_maintenance cm
		 JOIN cars c ON c.id = cm.car_id
		 WHERE cm.id = $1 AND c.owner_id = $2`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&owned)
		},
		maintenanceID, accountID,
	)
	if err != nil || owned == 0 {
		return models.CarUpload{}, fmt.Errorf("maintenance record not found")
	}

	var upload models.CarUpload
	err = g.db.QueryRow(ctx,
		`INSERT INTO car_uploads (maintenance_id, object_key, name, upload_type, content_type)
		 VALUES ($1::uuid, $2, $3, $4, $5)
		 RETURNING id::text, maintenance_id::text AS mod_id, object_key, name, upload_type, content_type`,
		db.WithResultOf(&upload),
		maintenanceID, objectKey, name, uploadType, contentType,
	)
	if err == nil {
		upload.URL = g.storage.CarFileURL(upload.ObjectKey)
	}
	return upload, err
}
