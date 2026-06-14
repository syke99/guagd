package garage

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"guagd/internal/pkg/css"
	"guagd/internal/pkg/middleware"
	"guagd/internal/pkg/models"
)

func (g *GarageClient) GaragePage(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.PathValue("username"), "@")
	if username == "" {
		http.NotFound(w, r)
		return
	}

	user, err := g.getUserByUsername(r.Context(), username)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	sessionContainer, _ := g.sessions.GetOptionalSession(r, w)

	isAuthenticated := sessionContainer != nil
	isOwner := isAuthenticated && sessionContainer.GetUserID() == user.SupertokensID

	layout, theme, coverPhotoURL, avatarURL, err := g.getGarageLayout(r.Context(), user.AccountID)
	if err != nil {
		log.Printf("garagePage: get layout: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	cars, err := g.getCars(r.Context(), user.AccountID)
	if err != nil {
		log.Printf("garagePage: get cars: %s", err)
		cars = make([]models.Car, 0)
	}

	var accountShortID string
	if isAuthenticated {
		if isOwner {
			if len(user.AccountID) >= 8 {
				accountShortID = user.AccountID[:8]
			}
		} else {
			if id, err := g.getAccountIDBySupertokensID(r.Context(), sessionContainer.GetUserID()); err == nil && len(id) >= 8 {
				accountShortID = id[:8]
			}
		}
	}

	data := models.GaragePageData{
		Username:        user.Username,
		IsOwner:         isOwner,
		IsAuthenticated: isAuthenticated,
		AccountShortID:  accountShortID,
		CarCount:        len(cars),
		Cars:            cars,
		Layout:          layout,
		SafeCSS:         css.BuildTheme(theme),
		CoverPhotoURL:   coverPhotoURL,
		AvatarURL:       avatarURL,
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store")
	if err := garageTemplate.ExecuteTemplate(w, "garage.html", data); err != nil {
		log.Printf("garagePage: render: %s", err)
	}
}

func accountID(r *http.Request) (string, bool) {
	v, ok := r.Context().Value(middleware.ContextKeyAccountID).(string)
	return v, ok && v != ""
}

func (g *GarageClient) SaveLayout(w http.ResponseWriter, r *http.Request) {
	var layout []models.LayoutItem
	if err := json.NewDecoder(r.Body).Decode(&layout); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.upsertLayout(r.Context(), id, layout); err != nil {
		log.Printf("saveLayout: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) SaveTheme(w http.ResponseWriter, r *http.Request) {
	var theme map[string]map[string]string
	if err := json.NewDecoder(r.Body).Decode(&theme); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.upsertTheme(r.Context(), id, theme); err != nil {
		log.Printf("saveTheme: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) AddCar(w http.ResponseWriter, r *http.Request) {
	var body models.Car
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Year == 0 || strings.TrimSpace(body.Make) == "" || strings.TrimSpace(body.Model) == "" {
		http.Error(w, "year, make, and model required", http.StatusBadRequest)
		return
	}

	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	car, err := g.addCar(r.Context(), id, models.Car{
		Year:    body.Year,
		Make:    strings.TrimSpace(body.Make),
		Model:   strings.TrimSpace(body.Model),
		Trim:    strings.TrimSpace(body.Trim),
		Mileage: body.Mileage,
	})
	if err != nil {
		log.Printf("addCar: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(car)
}

func (g *GarageClient) UpdateCar(w http.ResponseWriter, r *http.Request) {
	var body models.Car
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.ID == "" || body.Year == 0 || strings.TrimSpace(body.Make) == "" || strings.TrimSpace(body.Model) == "" {
		http.Error(w, "id, year, make, and model required", http.StatusBadRequest)
		return
	}

	accountID, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	car, err := g.updateCar(r.Context(), accountID, models.Car{
		ID:      body.ID,
		Year:    body.Year,
		Make:    strings.TrimSpace(body.Make),
		Model:   strings.TrimSpace(body.Model),
		Trim:    strings.TrimSpace(body.Trim),
		Mileage: body.Mileage,
	})
	if err != nil {
		log.Printf("updateCar: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(car)
}

func (g *GarageClient) RemoveCar(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.ID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.removeCar(r.Context(), id, body.ID); err != nil {
		log.Printf("removeCar: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) GetCarPhotos(w http.ResponseWriter, r *http.Request) {
	carID := r.URL.Query().Get("car_id")
	if carID == "" {
		http.Error(w, "car_id required", http.StatusBadRequest)
		return
	}

	photos, err := g.getCarPhotos(r.Context(), carID)
	if err != nil {
		log.Printf("getCarPhotos: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if photos == nil {
		photos = []models.CarPhoto{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(photos)
}

func (g *GarageClient) AddCarPhoto(w http.ResponseWriter, r *http.Request) {
	carID := r.URL.Query().Get("car_id")
	if carID == "" {
		http.Error(w, "car_id required", http.StatusBadRequest)
		return
	}

	var body struct {
		ObjectKey string `json:"object_key"`
		IsPrimary bool   `json:"is_primary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ObjectKey) == "" {
		http.Error(w, "object_key required", http.StatusBadRequest)
		return
	}

	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	photo, err := g.addCarPhoto(r.Context(), id, carID, body.ObjectKey, body.IsPrimary)
	if err != nil {
		log.Printf("addCarPhoto: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	photo.URL = g.storage.CarPhotoURL(photo.ObjectKey)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(photo)
}

func (g *GarageClient) RemoveCarPhoto(w http.ResponseWriter, r *http.Request) {
	photoID := r.URL.Query().Get("photo_id")
	if photoID == "" {
		http.Error(w, "photo_id required", http.StatusBadRequest)
		return
	}

	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.removeCarPhoto(r.Context(), id, photoID); err != nil {
		log.Printf("removeCarPhoto: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) SetCarPhotoPrimary(w http.ResponseWriter, r *http.Request) {
	carID := r.URL.Query().Get("car_id")
	photoID := r.URL.Query().Get("photo_id")
	if carID == "" || photoID == "" {
		http.Error(w, "car_id and photo_id required", http.StatusBadRequest)
		return
	}

	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.setCarPhotoPrimary(r.Context(), id, carID, photoID); err != nil {
		log.Printf("setCarPhotoPrimary: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) SaveAvatar(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ObjectKey string `json:"object_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ObjectKey) == "" {
		http.Error(w, "object_key required", http.StatusBadRequest)
		return
	}

	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.saveAvatar(r.Context(), id, body.ObjectKey); err != nil {
		log.Printf("saveAvatar: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": g.storage.AccountPhotoURL(body.ObjectKey)})
}

func (g *GarageClient) RemoveAvatar(w http.ResponseWriter, r *http.Request) {
	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.removeAvatar(r.Context(), id); err != nil {
		log.Printf("removeAvatar: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) SaveCoverPhoto(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ObjectKey string `json:"object_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ObjectKey) == "" {
		http.Error(w, "object_key required", http.StatusBadRequest)
		return
	}

	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.saveCoverPhoto(r.Context(), id, body.ObjectKey); err != nil {
		log.Printf("saveCoverPhoto: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": g.storage.AccountPhotoURL(body.ObjectKey)})
}

func (g *GarageClient) RemoveCoverPhoto(w http.ResponseWriter, r *http.Request) {
	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.removeCoverPhoto(r.Context(), id); err != nil {
		log.Printf("removeCoverPhoto: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) CarModalFragment(w http.ResponseWriter, r *http.Request) {
	carID := r.URL.Query().Get("car_id")
	if carID == "" {
		http.Error(w, "car_id required", http.StatusBadRequest)
		return
	}

	car, ownerSupertokensID, err := g.getCarForModal(r.Context(), carID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	sessionContainer, _ := g.sessions.GetOptionalSession(r, w)
	isOwner := sessionContainer != nil && sessionContainer.GetUserID() == ownerSupertokensID

	type result[T any] struct {
		val T
		err error
	}
	photoCh := make(chan result[[]models.CarPhoto], 1)
	modCh := make(chan result[[]models.Mod], 1)
	maintCh := make(chan result[[]models.Maintenance], 1)
	verifCh := make(chan result[models.VerificationCounts], 1)

	go func() {
		v, e := g.getCarPhotos(r.Context(), carID)
		photoCh <- result[[]models.CarPhoto]{v, e}
	}()
	go func() {
		v, e := g.getMods(r.Context(), carID)
		if v == nil {
			v = []models.Mod{}
		}
		modCh <- result[[]models.Mod]{v, e}
	}()
	go func() {
		v, e := g.getMaintenance(r.Context(), carID)
		if v == nil {
			v = []models.Maintenance{}
		}
		maintCh <- result[[]models.Maintenance]{v, e}
	}()
	go func() {
		v, e := g.getVerificationCounts(r.Context(), carID)
		verifCh <- result[models.VerificationCounts]{v, e}
	}()

	photos := (<-photoCh).val
	mods := (<-modCh).val
	maintenance := (<-maintCh).val
	verifications := (<-verifCh).val

	var primary *models.CarPhoto
	var nonPrimary []models.CarPhoto
	for i := range photos {
		if photos[i].IsPrimary {
			primary = &photos[i]
		} else {
			nonPrimary = append(nonPrimary, photos[i])
		}
	}
	data := models.CarModalData{
		Car:             car,
		PrimaryPhoto:    primary,
		Photos:          nonPrimary,
		TotalPhotoCount: len(photos),
		Mods:            mods,
		Maintenance:     maintenance,
		Verifications:   verifications,
		IsOwner:         isOwner,
		MaxPhotos:       10,
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store")
	if err := carModalTemplate.ExecuteTemplate(w, "car-modal-card.html", data); err != nil {
		log.Printf("carModalFragment: render: %s", err)
	}
}

func (g *GarageClient) CarDocsFragment(w http.ResponseWriter, r *http.Request) {
	recordType := r.URL.Query().Get("type")
	recordID := r.URL.Query().Get("record_id")
	if (recordType != "mod" && recordType != "maintenance") || recordID == "" {
		http.Error(w, "type and record_id required", http.StatusBadRequest)
		return
	}

	ownerSupertokensID, err := g.getOwnerForRecord(r.Context(), recordType, recordID)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	sessionContainer, _ := g.sessions.GetOptionalSession(r, w)
	isOwner := sessionContainer != nil && sessionContainer.GetUserID() == ownerSupertokensID

	var uploads []models.CarUpload
	if recordType == "maintenance" {
		uploads, err = g.getMaintenanceUploads(r.Context(), recordID)
	} else {
		uploads, err = g.getCarUploads(r.Context(), recordID)
	}
	if err != nil {
		log.Printf("CarDocsFragment: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-store")
	if err := carDocsTemplate.ExecuteTemplate(w, "car-docs-fragment.html", models.CarDocsData{
		Uploads:  uploads,
		IsOwner:  isOwner,
		Type:     recordType,
		RecordID: recordID,
	}); err != nil {
		log.Printf("CarDocsFragment: render: %s", err)
	}
}

func (g *GarageClient) GetCarMods(w http.ResponseWriter, r *http.Request) {
	carID := r.URL.Query().Get("car_id")
	if carID == "" {
		http.Error(w, "car_id required", http.StatusBadRequest)
		return
	}
	mods, err := g.getMods(r.Context(), carID)
	if err != nil {
		log.Printf("getCarMods: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if mods == nil {
		mods = []models.Mod{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mods)
}

func (g *GarageClient) AddCarMod(w http.ResponseWriter, r *http.Request) {
	carID := r.URL.Query().Get("car_id")
	if carID == "" {
		http.Error(w, "car_id required", http.StatusBadRequest)
		return
	}
	var body models.Mod
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if body.Category == "" {
		body.Category = "Other"
	}
	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	mod, err := g.addMod(r.Context(), id, carID, body)
	if err != nil {
		log.Printf("addCarMod: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mod)
}

func (g *GarageClient) RemoveCarMod(w http.ResponseWriter, r *http.Request) {
	modID := r.URL.Query().Get("mod_id")
	if modID == "" {
		http.Error(w, "mod_id required", http.StatusBadRequest)
		return
	}
	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.removeMod(r.Context(), id, modID); err != nil {
		log.Printf("removeCarMod: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) GetCarUploads(w http.ResponseWriter, r *http.Request) {
	modID := r.URL.Query().Get("mod_id")
	if modID == "" {
		http.Error(w, "mod_id required", http.StatusBadRequest)
		return
	}
	uploads, err := g.getCarUploads(r.Context(), modID)
	if err != nil {
		log.Printf("getCarUploads: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if uploads == nil {
		uploads = []models.CarUpload{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uploads)
}

func (g *GarageClient) AddCarUpload(w http.ResponseWriter, r *http.Request) {
	modID := r.URL.Query().Get("mod_id")
	if modID == "" {
		http.Error(w, "mod_id required", http.StatusBadRequest)
		return
	}
	var body struct {
		ObjectKey   string `json:"object_key"`
		Name        string `json:"name"`
		UploadType  string `json:"upload_type"`
		ContentType string `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ObjectKey) == "" || strings.TrimSpace(body.Name) == "" {
		http.Error(w, "object_key and name required", http.StatusBadRequest)
		return
	}
	if body.UploadType == "" {
		body.UploadType = "Receipt"
	}
	if body.ContentType == "" {
		body.ContentType = "application/octet-stream"
	}
	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	upload, err := g.addCarUpload(r.Context(), id, modID, body.ObjectKey, body.Name, body.UploadType, body.ContentType)
	if err != nil {
		log.Printf("addCarUpload: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(upload)
}

func (g *GarageClient) RemoveCarUpload(w http.ResponseWriter, r *http.Request) {
	uploadID := r.URL.Query().Get("upload_id")
	if uploadID == "" {
		http.Error(w, "upload_id required", http.StatusBadRequest)
		return
	}
	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.removeCarUpload(r.Context(), id, uploadID); err != nil {
		log.Printf("removeCarUpload: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) GetMaintenance(w http.ResponseWriter, r *http.Request) {
	carID := r.URL.Query().Get("car_id")
	if carID == "" {
		http.Error(w, "car_id required", http.StatusBadRequest)
		return
	}
	records, err := g.getMaintenance(r.Context(), carID)
	if err != nil {
		log.Printf("getMaintenance: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if records == nil {
		records = []models.Maintenance{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

func (g *GarageClient) AddMaintenance(w http.ResponseWriter, r *http.Request) {
	carID := r.URL.Query().Get("car_id")
	if carID == "" {
		http.Error(w, "car_id required", http.StatusBadRequest)
		return
	}
	var body models.Maintenance
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	if body.Category == "" {
		body.Category = "Other"
	}
	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	record, err := g.addMaintenance(r.Context(), id, carID, body)
	if err != nil {
		log.Printf("addMaintenance: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}

func (g *GarageClient) RemoveMaintenance(w http.ResponseWriter, r *http.Request) {
	recordID := r.URL.Query().Get("record_id")
	if recordID == "" {
		http.Error(w, "record_id required", http.StatusBadRequest)
		return
	}
	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	if err := g.removeMaintenance(r.Context(), id, recordID); err != nil {
		log.Printf("removeMaintenance: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) GetMaintenanceUploads(w http.ResponseWriter, r *http.Request) {
	maintenanceID := r.URL.Query().Get("maintenance_id")
	if maintenanceID == "" {
		http.Error(w, "maintenance_id required", http.StatusBadRequest)
		return
	}
	uploads, err := g.getMaintenanceUploads(r.Context(), maintenanceID)
	if err != nil {
		log.Printf("getMaintenanceUploads: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if uploads == nil {
		uploads = []models.CarUpload{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(uploads)
}

func (g *GarageClient) AddMaintenanceUpload(w http.ResponseWriter, r *http.Request) {
	maintenanceID := r.URL.Query().Get("maintenance_id")
	if maintenanceID == "" {
		http.Error(w, "maintenance_id required", http.StatusBadRequest)
		return
	}
	var body struct {
		ObjectKey   string `json:"object_key"`
		Name        string `json:"name"`
		UploadType  string `json:"upload_type"`
		ContentType string `json:"content_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.ObjectKey) == "" || strings.TrimSpace(body.Name) == "" {
		http.Error(w, "object_key and name required", http.StatusBadRequest)
		return
	}
	if body.UploadType == "" {
		body.UploadType = "Receipt"
	}
	if body.ContentType == "" {
		body.ContentType = "application/octet-stream"
	}
	id, ok := accountID(r)
	if !ok {
		http.Error(w, "session expired; please sign out and sign back in", http.StatusUnauthorized)
		return
	}
	upload, err := g.addMaintenanceUpload(r.Context(), id, maintenanceID, body.ObjectKey, body.Name, body.UploadType, body.ContentType)
	if err != nil {
		log.Printf("addMaintenanceUpload: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(upload)
}
