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
	"guagd/internal/pkg/storage"
)

//go:embed templates/*
var templates embed.FS

var garageTemplate = template.Must(
	template.Must(template.New("").Parse(shared.NavTemplate)).
		ParseFS(templates, "templates/garage.html"),
)

type GarageClient struct {
	db      db.DB
	storage *storage.Client
}

func NewGarageClient(db db.DB, store *storage.Client) *GarageClient {
	return &GarageClient{db: db, storage: store}
}

var defaultLayout = []models.LayoutItem{
	{Component: "profile-header", X: 0, Y: 0, W: 12, H: 3},
	{Component: "car-list", X: 0, Y: 3, W: 12, H: 6},
}

func (g *GarageClient) getUserByUsername(ctx context.Context, username string) (*models.GarageUser, error) {
	var user models.GarageUser
	err := g.db.QueryRow(
		ctx,
		"SELECT supertokens_id, username FROM accounts WHERE username = $1",
		db.WithResultOf(&user),
		username,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (g *GarageClient) getGarageLayout(ctx context.Context, supertokensID string) ([]models.LayoutItem, map[string]map[string]string, string, error) {
	var layoutJSON, themeJSON string
	var coverPhotoKey string
	err := g.db.QueryRow(
		ctx,
		"SELECT layout::text, theme::text, COALESCE(cover_photo_key, '') FROM garage_layouts WHERE supertokens_id = $1",
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&layoutJSON, &themeJSON, &coverPhotoKey)
		},
		supertokensID,
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

	coverPhotoURL := ""
	if coverPhotoKey != "" {
		coverPhotoURL = g.storage.AccountPhotoURL(coverPhotoKey)
	}

	return layout, theme, coverPhotoURL, nil
}

func (g *GarageClient) upsertLayout(ctx context.Context, supertokensID string, layout []models.LayoutItem) error {
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

func (g *GarageClient) getCars(ctx context.Context, ownerID string) ([]models.Car, error) {
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
		ownerID,
	)
	for i := range cars {
		if cars[i].ObjectKey != "" {
			cars[i].PrimaryPhotoURL = g.storage.CarPhotoURL(cars[i].ObjectKey)
		}
	}
	return cars, err
}

func (g *GarageClient) addCar(ctx context.Context, ownerID string, car models.Car) (models.Car, error) {
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

func (g *GarageClient) addCarPhoto(ctx context.Context, ownerID, carID, objectKey string, isPrimary bool) (models.CarPhoto, error) {
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
		carID, ownerID,
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

func (g *GarageClient) removeCarPhoto(ctx context.Context, ownerID, photoID string) error {
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
		photoID, ownerID,
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

func (g *GarageClient) setCarPhotoPrimary(ctx context.Context, ownerID, carID, photoID string) error {
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
		photoID, carID, ownerID,
	)
	if err != nil || owned == 0 {
		return fmt.Errorf("photo not found")
	}

	if err := g.db.Exec(ctx, `UPDATE car_photos SET is_primary = false WHERE car_id = $1`, carID); err != nil {
		return err
	}
	return g.db.Exec(ctx, `UPDATE car_photos SET is_primary = true WHERE id = $1`, photoID)
}

func (g *GarageClient) saveCoverPhoto(ctx context.Context, ownerID, objectKey string) error {
	return g.db.Exec(
		ctx,
		`INSERT INTO garage_layouts (supertokens_id, cover_photo_key)
		 VALUES ($1, $2)
		 ON CONFLICT (supertokens_id) DO UPDATE
		 SET cover_photo_key = EXCLUDED.cover_photo_key`,
		ownerID, objectKey,
	)
}
