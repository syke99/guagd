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

	data := models.GaragePageData{
		Username:        user.Username,
		IsOwner:         isOwner,
		IsAuthenticated: isAuthenticated,
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
