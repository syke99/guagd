package garage

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"

	"guagd/internal/pkg/middleware"
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

	sessionRequired := false
	sessionContainer, _ := session.GetSession(r, w, &sessmodels.VerifySessionOptions{
		SessionRequired: &sessionRequired,
	})

	isAuthenticated := sessionContainer != nil
	isOwner := isAuthenticated && sessionContainer.GetUserID() == user.SupertokensID

	layout, theme, coverPhotoURL, err := g.getGarageLayout(r.Context(), user.SupertokensID)
	if err != nil {
		log.Printf("garagePage: get layout: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	cars, err := g.getCars(r.Context(), user.SupertokensID)
	if err != nil {
		log.Printf("garagePage: get cars: %s", err)
		cars = []Car{}
	}

	data := GaragePageData{
		Username:        user.Username,
		IsOwner:         isOwner,
		IsAuthenticated: isAuthenticated,
		CarCount:        len(cars),
		Cars:            cars,
		Layout:          layout,
		SafeCSS:         buildThemeCSS(theme),
		CoverPhotoURL:   coverPhotoURL,
	}

	w.Header().Set("Content-Type", "text/html")
	if err := garageTemplate.ExecuteTemplate(w, "garage.html", data); err != nil {
		log.Printf("garagePage: render: %s", err)
	}
}

func (g *GarageClient) SaveLayout(w http.ResponseWriter, r *http.Request) {
	var layout []LayoutItem
	if err := json.NewDecoder(r.Body).Decode(&layout); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	if err := g.upsertLayout(r.Context(), userID, layout); err != nil {
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

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	if err := g.upsertTheme(r.Context(), userID, theme); err != nil {
		log.Printf("saveTheme: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (g *GarageClient) AddCar(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Year    int    `json:"year"`
		Make    string `json:"make"`
		Model   string `json:"model"`
		Trim    string `json:"trim"`
		Mileage int    `json:"mileage"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Year == 0 || strings.TrimSpace(body.Make) == "" || strings.TrimSpace(body.Model) == "" {
		http.Error(w, "year, make, and model required", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	car, err := g.addCar(r.Context(), userID, Car{
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

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	if err := g.removeCar(r.Context(), userID, body.ID); err != nil {
		log.Printf("removeCar: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	photo, err := g.addCarPhoto(r.Context(), userID, carID, body.ObjectKey, body.IsPrimary)
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

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	if err := g.removeCarPhoto(r.Context(), userID, photoID); err != nil {
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

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	if err := g.setCarPhotoPrimary(r.Context(), userID, carID, photoID); err != nil {
		log.Printf("setCarPhotoPrimary: %s", err)
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

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	if err := g.saveCoverPhoto(r.Context(), userID, body.ObjectKey); err != nil {
		log.Printf("saveCoverPhoto: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
