package requests

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"

	"guagd/internal/pkg/db"
	"guagd/internal/pkg/models"
)

// RequestsPage serves /requests?u={short_id}. Session-required; routes to driver or shop view.
func (rc *RequestsClient) RequestsPage(w http.ResponseWriter, r *http.Request) {
	sess, err := rc.sessions.GetOptionalSession(r, w)
	if err != nil || sess == nil {
		http.Redirect(w, r, "/signin", http.StatusFound)
		return
	}

	shortID := r.URL.Query().Get("u")
	if len(shortID) < 8 {
		http.NotFound(w, r)
		return
	}
	shortID = shortID[:8]

	acct, err := rc.getAccountByShortID(r.Context(), shortID)
	if err != nil || acct.SupertokensID != sess.GetUserID() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	var reqs []models.VerificationRequest
	switch acct.AcctType {
	case "shop":
		reqs, err = rc.getShopRequests(r.Context(), acct.AccountID)
	default:
		reqs, err = rc.getDriverRequests(r.Context(), acct.AccountID)
	}
	if err != nil {
		log.Printf("requestsPage: get requests: %s", err)
		reqs = []models.VerificationRequest{}
	}

	avatarURL := ""
	if acct.AvatarKey != "" {
		avatarURL = rc.storage.AccountPhotoURL(acct.AvatarKey)
	}

	data := models.RequestsPageData{
		AcctType:        acct.AcctType,
		AccountShortID:  shortID,
		Username:        acct.Username,
		AvatarURL:       avatarURL,
		IsAuthenticated: true,
		Requests:        reqs,
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store")
	if err := requestsTemplate.ExecuteTemplate(w, "requests.html", data); err != nil {
		log.Printf("requestsPage: render: %s", err)
	}
}

// WizardCarsFragment returns the car selector HTML for step 1 of the create wizard.
func (rc *RequestsClient) WizardCarsFragment(w http.ResponseWriter, r *http.Request) {
	sess, _ := rc.sessions.GetOptionalSession(r, w)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	shortID := r.URL.Query().Get("u")
	if len(shortID) < 8 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	acct, err := rc.getAccountByShortID(r.Context(), shortID[:8])
	if err != nil || acct.SupertokensID != sess.GetUserID() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	cars, err := rc.getDriverCars(r.Context(), acct.AccountID)
	if err != nil {
		log.Printf("wizardCars: %s", err)
		cars = []models.Car{}
	}

	w.Header().Set("Content-Type", "text/html")
	if err := requestsTemplate.ExecuteTemplate(w, "wizard-cars", cars); err != nil {
		log.Printf("wizardCars: render: %s", err)
	}
}

// WizardRecordsFragment returns mod+maintenance checkboxes for step 2.
func (rc *RequestsClient) WizardRecordsFragment(w http.ResponseWriter, r *http.Request) {
	sess, _ := rc.sessions.GetOptionalSession(r, w)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	shortID := r.URL.Query().Get("u")
	carID := r.URL.Query().Get("car_id")
	if len(shortID) < 8 || carID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	acct, err := rc.getAccountByShortID(r.Context(), shortID[:8])
	if err != nil || acct.SupertokensID != sess.GetUserID() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	records, err := rc.getCarRecords(r.Context(), acct.AccountID, carID)
	if err != nil {
		log.Printf("wizardRecords: %s", err)
		records = []models.CarRecord{}
	}

	w.Header().Set("Content-Type", "text/html")
	if err := requestsTemplate.ExecuteTemplate(w, "wizard-records", records); err != nil {
		log.Printf("wizardRecords: render: %s", err)
	}
}

// ListDriverRequests returns the driver's outgoing requests as an HTML fragment.
func (rc *RequestsClient) ListDriverRequests(w http.ResponseWriter, r *http.Request) {
	sess, _ := rc.sessions.GetOptionalSession(r, w)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	shortID := r.URL.Query().Get("u")
	if len(shortID) < 8 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	acct, err := rc.getAccountByShortID(r.Context(), shortID[:8])
	if err != nil || acct.SupertokensID != sess.GetUserID() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	reqs, err := rc.getDriverRequests(r.Context(), acct.AccountID)
	if err != nil {
		log.Printf("listDriverRequests: %s", err)
		reqs = []models.VerificationRequest{}
	}

	w.Header().Set("Content-Type", "text/html")
	data := models.RequestsPageData{
		AcctType:       acct.AcctType,
		AccountShortID: shortID[:8],
		Requests:       reqs,
	}
	if err := requestsTemplate.ExecuteTemplate(w, "driver-request-list", data); err != nil {
		log.Printf("listDriverRequests: render: %s", err)
	}
}

// ListShopRequests returns the shop's incoming requests as an HTML fragment.
func (rc *RequestsClient) ListShopRequests(w http.ResponseWriter, r *http.Request) {
	sess, _ := rc.sessions.GetOptionalSession(r, w)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	shortID := r.URL.Query().Get("u")
	if len(shortID) < 8 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	acct, err := rc.getAccountByShortID(r.Context(), shortID[:8])
	if err != nil || acct.SupertokensID != sess.GetUserID() || acct.AcctType != "shop" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	reqs, err := rc.getShopRequests(r.Context(), acct.AccountID)
	if err != nil {
		log.Printf("listShopRequests: %s", err)
		reqs = []models.VerificationRequest{}
	}

	w.Header().Set("Content-Type", "text/html")
	data := models.RequestsPageData{
		AcctType:       acct.AcctType,
		AccountShortID: shortID[:8],
		Requests:       reqs,
	}
	if err := requestsTemplate.ExecuteTemplate(w, "shop-request-list", data); err != nil {
		log.Printf("listShopRequests: render: %s", err)
	}
}

// CreateRequest handles POST /api/v1/driver/requests/create.
func (rc *RequestsClient) CreateRequest(w http.ResponseWriter, r *http.Request) {
	sess, _ := rc.sessions.GetOptionalSession(r, w)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload models.CreateRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if payload.ShopUsername == "" || len(payload.VerifIDs) == 0 {
		http.Error(w, "shop_username and verif_ids required", http.StatusBadRequest)
		return
	}
	if payload.ServiceType != "past" && payload.ServiceType != "appointment" {
		http.Error(w, "invalid service_type", http.StatusBadRequest)
		return
	}
	if payload.WorkType != "own_work" && payload.WorkType != "other_shop_work" {
		http.Error(w, "invalid work_type", http.StatusBadRequest)
		return
	}

	var requesterID string
	if err := rc.db.QueryRow(r.Context(),
		`SELECT id::text FROM accounts WHERE supertokens_id = $1`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return fmt.Errorf("not found")
			}
			return rows.Scan(&requesterID)
		},
		sess.GetUserID(),
	); err != nil {
		http.Error(w, "account not found", http.StatusUnauthorized)
		return
	}

	shopID, err := rc.getShopIDByUsername(r.Context(), payload.ShopUsername)
	if err != nil {
		http.Error(w, "shop not found", http.StatusBadRequest)
		return
	}

	if err := rc.createRequest(r.Context(), requesterID, shopID,
		payload.ServiceType, payload.WorkType, payload.Notes, payload.VerifIDs,
	); err != nil {
		log.Printf("createRequest: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return updated request list fragment
	reqs, _ := rc.getDriverRequests(r.Context(), requesterID)
	data := models.RequestsPageData{
		AccountShortID: r.URL.Query().Get("u"),
		Requests:       reqs,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := requestsTemplate.ExecuteTemplate(w, "driver-request-list", data); err != nil {
		log.Printf("createRequest: render: %s", err)
	}
}

// RespondToItems handles POST /api/v1/shop/requests/respond.
func (rc *RequestsClient) RespondToItems(w http.ResponseWriter, r *http.Request) {
	sess, _ := rc.sessions.GetOptionalSession(r, w)
	if sess == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var payload models.RespondPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if len(payload.Items) == 0 {
		http.Error(w, "items required", http.StatusBadRequest)
		return
	}

	var shopID string
	if err := rc.db.QueryRow(r.Context(),
		`SELECT id::text FROM accounts WHERE supertokens_id = $1 AND acct_type = 'shop'`,
		func(rows pgx.Rows) error {
			if !rows.Next() {
				return fmt.Errorf("not found")
			}
			return rows.Scan(&shopID)
		},
		sess.GetUserID(),
	); err != nil {
		http.Error(w, "shop account not found", http.StatusForbidden)
		return
	}

	if err := rc.respondToItems(r.Context(), shopID, payload.Items); err != nil {
		log.Printf("respondToItems: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Return updated shop request list
	reqs, _ := rc.getShopRequests(r.Context(), shopID)
	shortID := r.URL.Query().Get("u")
	data := models.RequestsPageData{
		AcctType:       "shop",
		AccountShortID: shortID,
		Requests:       reqs,
	}
	w.Header().Set("Content-Type", "text/html")
	if err := requestsTemplate.ExecuteTemplate(w, "shop-request-list", data); err != nil {
		log.Printf("respondToItems: render: %s", err)
	}
}

// ShopSearchFragment returns shop search results with data-id attributes for wizard use.
func (rc *RequestsClient) ShopSearchFragment(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	type shopResult struct {
		AccountID string `db:"account_id"`
		Username  string `db:"username"`
	}
	var results []shopResult
	if err := rc.db.Query(r.Context(),
		`SELECT id::text AS account_id, username
		 FROM accounts
		 WHERE username ILIKE $1 AND acct_type = 'shop'
		 ORDER BY username LIMIT 10`,
		db.WithResultsOf(&results),
		"%"+q+"%",
	); err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	for _, res := range results {
		fmt.Fprintf(w,
			`<div class="shop-search-result" data-id="%s" data-username="%s" onclick="selectShop(this)">@%s</div>`,
			html.EscapeString(res.AccountID),
			html.EscapeString(res.Username),
			html.EscapeString(res.Username),
		)
	}
}
