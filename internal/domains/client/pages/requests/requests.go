package requests

import (
	"context"
	"embed"
	"fmt"
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

var requestsTemplate = template.Must(
	template.Must(template.New("").Parse(shared.NavTemplate)).
		ParseFS(templates, "templates/requests.html"),
)

type RequestsClient struct {
	db       db.DB
	storage  *storage.Client
	sessions sessions.Getter
}

func NewRequestsClient(database db.DB, store *storage.Client, sg sessions.Getter) *RequestsClient {
	return &RequestsClient{db: database, storage: store, sessions: sg}
}

type accountInfo struct {
	AccountID     string `db:"account_id"`
	SupertokensID string `db:"supertokens_id"`
	Username      string `db:"username"`
	AcctType      string `db:"acct_type"`
	AvatarKey     string `db:"avatar_key"`
}

func (rc *RequestsClient) getAccountByShortID(ctx context.Context, shortID string) (accountInfo, error) {
	var info accountInfo
	err := rc.db.QueryRow(ctx,
		`SELECT a.id::text AS account_id,
		        a.supertokens_id,
		        a.username,
		        a.acct_type,
		        COALESCE(ap.avatar_key, '') AS avatar_key
		 FROM accounts a
		 LEFT JOIN account_photos ap ON ap.account_id = a.id
		 WHERE a.id::text LIKE $1 || '%'
		 LIMIT 1`,
		db.WithResultOf(&info),
		shortID,
	)
	return info, err
}

func (rc *RequestsClient) getShopIDByUsername(ctx context.Context, username string) (string, error) {
	var id string
	err := rc.db.QueryRow(ctx,
		`SELECT id::text FROM accounts WHERE username = $1 AND acct_type = 'shop'`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&id)
		},
		username,
	)
	return id, err
}

func (rc *RequestsClient) getDriverRequests(ctx context.Context, accountID string) ([]models.VerificationRequest, error) {
	var reqs []models.VerificationRequest
	err := rc.db.Query(ctx,
		`SELECT vr.id::text,
		        a.username                    AS shop_name,
		        vr.shop_id::text              AS shop_id,
		        vr.service_type,
		        vr.work_type,
		        vr.status,
		        COALESCE(vr.notes, '')         AS notes,
		        vr.created_at::text           AS created_at
		 FROM verification_requests vr
		 JOIN accounts a ON a.id = vr.shop_id
		 WHERE vr.requester_id = $1::uuid
		 ORDER BY vr.created_at DESC`,
		db.WithResultsOf(&reqs),
		accountID,
	)
	if err != nil || len(reqs) == 0 {
		return reqs, err
	}

	ids := make([]string, len(reqs))
	for i, r := range reqs {
		ids[i] = r.ID
	}
	items, err := rc.getItemsForRequests(ctx, ids)
	if err != nil {
		return reqs, nil
	}
	byReqID := make(map[string][]models.VerificationRequestItem)
	for _, it := range items {
		byReqID[it.RequestID] = append(byReqID[it.RequestID], it)
	}
	for i := range reqs {
		reqs[i].Items = byReqID[reqs[i].ID]
	}
	return reqs, nil
}

func (rc *RequestsClient) getShopRequests(ctx context.Context, shopID string) ([]models.VerificationRequest, error) {
	var reqs []models.VerificationRequest
	err := rc.db.Query(ctx,
		`SELECT vr.id::text,
		        a.username                    AS shop_name,
		        vr.shop_id::text              AS shop_id,
		        vr.service_type,
		        vr.work_type,
		        vr.status,
		        COALESCE(vr.notes, '')         AS notes,
		        vr.created_at::text           AS created_at
		 FROM verification_requests vr
		 JOIN accounts a ON a.id = vr.requester_id
		 WHERE vr.shop_id = $1::uuid
		 ORDER BY vr.created_at DESC`,
		db.WithResultsOf(&reqs),
		shopID,
	)
	if err != nil || len(reqs) == 0 {
		return reqs, err
	}

	ids := make([]string, len(reqs))
	for i, r := range reqs {
		ids[i] = r.ID
	}
	items, err := rc.getItemsForRequests(ctx, ids)
	if err != nil {
		return reqs, nil
	}
	byReqID := make(map[string][]models.VerificationRequestItem)
	for _, it := range items {
		byReqID[it.RequestID] = append(byReqID[it.RequestID], it)
	}
	for i := range reqs {
		reqs[i].Items = byReqID[reqs[i].ID]
	}
	return reqs, nil
}

func (rc *RequestsClient) getItemsForRequests(ctx context.Context, requestIDs []string) ([]models.VerificationRequestItem, error) {
	var items []models.VerificationRequestItem
	err := rc.db.Query(ctx,
		`SELECT vri.id::text,
		        vri.request_id::text                                   AS request_id,
		        vri.status,
		        COALESCE(vri.denial_notes, '')                          AS denial_notes,
		        COALESCE(cm.name, mnt.name, '')                         AS record_name,
		        CASE WHEN cm.id IS NOT NULL THEN 'mod' ELSE 'maintenance' END AS record_type,
		        COALESCE(cm.category, mnt.category, '')                 AS category,
		        cv.level                                                AS verif_level
		 FROM verification_request_items vri
		 JOIN car_verifications cv ON cv.id = vri.verification_id
		 LEFT JOIN car_mods cm         ON cm.id  = cv.mod_id
		 LEFT JOIN car_maintenance mnt ON mnt.id = cv.maintenance_id
		 WHERE vri.request_id = ANY($1::uuid[])
		 ORDER BY vri.created_at ASC`,
		db.WithResultsOf(&items),
		requestIDs,
	)
	return items, err
}

func (rc *RequestsClient) getDriverCars(ctx context.Context, accountID string) ([]models.Car, error) {
	var cars []models.Car
	err := rc.db.Query(ctx,
		`SELECT c.id::text,
		        c.year,
		        c.make,
		        c.model,
		        COALESCE(c.trim, '')       AS trim,
		        COALESCE(c.mileage, 0)    AS mileage,
		        COALESCE(p.object_key, '') AS object_key
		 FROM cars c
		 LEFT JOIN car_photos p ON p.car_id = c.id AND p.is_primary = true
		 WHERE c.owner_id = $1::uuid
		 ORDER BY c.created_at`,
		db.WithResultsOf(&cars),
		accountID,
	)
	return cars, err
}

func (rc *RequestsClient) getCarRecords(ctx context.Context, accountID, carID string) ([]models.CarRecord, error) {
	var records []models.CarRecord

	// Mods
	var mods []models.CarRecord
	if err := rc.db.Query(ctx,
		`SELECT cv.id::text AS verif_id,
		        cm.name     AS record_name,
		        'mod'       AS record_type,
		        cm.category,
		        cv.level    AS verif_level
		 FROM car_mods cm
		 JOIN car_verifications cv ON cv.mod_id = cm.id
		 WHERE cm.car_id = $1::uuid
		   AND EXISTS (SELECT 1 FROM cars WHERE id = $1::uuid AND owner_id = $2::uuid)
		 ORDER BY cm.created_at ASC`,
		db.WithResultsOf(&mods),
		carID, accountID,
	); err != nil {
		return nil, err
	}

	// Maintenance
	var maints []models.CarRecord
	if err := rc.db.Query(ctx,
		`SELECT cv.id::text  AS verif_id,
		        mnt.name     AS record_name,
		        'maintenance' AS record_type,
		        mnt.category,
		        cv.level     AS verif_level
		 FROM car_maintenance mnt
		 JOIN car_verifications cv ON cv.maintenance_id = mnt.id
		 WHERE mnt.car_id = $1::uuid
		   AND EXISTS (SELECT 1 FROM cars WHERE id = $1::uuid AND owner_id = $2::uuid)
		 ORDER BY mnt.created_at ASC`,
		db.WithResultsOf(&maints),
		carID, accountID,
	); err != nil {
		return nil, err
	}

	records = append(mods, maints...)
	return records, nil
}

func (rc *RequestsClient) createRequest(ctx context.Context, requesterID, shopID, serviceType, workType, notes string, verifIDs []string) error {
	if len(verifIDs) == 0 {
		return fmt.Errorf("no verif_ids provided")
	}

	tx, err := rc.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var requestID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO verification_requests (requester_id, shop_id, service_type, work_type, status, notes)
		 VALUES ($1::uuid, $2::uuid, $3, $4, 'requested', NULLIF($5,''))
		 RETURNING id::text`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return pgx.ErrNoRows
			}
			return rows.Scan(&requestID)
		},
		requesterID, shopID, serviceType, workType, notes,
	); err != nil {
		return err
	}

	for _, vid := range verifIDs {
		if err := tx.Exec(ctx,
			`INSERT INTO verification_request_items (request_id, verification_id)
			 VALUES ($1::uuid, $2::uuid)`,
			requestID, vid,
		); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (rc *RequestsClient) respondToItems(ctx context.Context, shopID string, items []models.RespondItem) error {
	tx, err := rc.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, item := range items {
		if item.Status != "approved" && item.Status != "denied" {
			continue
		}

		// Verify the item belongs to a request directed at this shop
		var ok int
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM verification_request_items vri
			 JOIN verification_requests vr ON vr.id = vri.request_id
			 WHERE vri.id = $1::uuid AND vr.shop_id = $2::uuid`,
			func(rows pgx.Rows) error {
				if !rows.Next() {
					return pgx.ErrNoRows
				}
				return rows.Scan(&ok)
			},
			item.ItemID, shopID,
		); err != nil || ok == 0 {
			continue
		}

		if err := tx.Exec(ctx,
			`UPDATE verification_request_items
			 SET status = $2, denial_notes = NULLIF($3,''), updated_at = NOW()
			 WHERE id = $1::uuid`,
			item.ItemID, item.Status, item.DenialNotes,
		); err != nil {
			return err
		}

		if item.Status == "approved" {
			// Determine level from request work_type
			if err := tx.Exec(ctx,
				`UPDATE car_verifications cv
				 SET level = CASE
				       WHEN vr.work_type = 'own_work' THEN 'verified'
				       ELSE 'performed'
				     END,
				     shop_id    = vr.shop_id,
				     updated_at = NOW()
				 FROM verification_request_items vri
				 JOIN verification_requests vr ON vr.id = vri.request_id
				 WHERE vri.id      = $1::uuid
				   AND cv.id       = vri.verification_id`,
				item.ItemID,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}
