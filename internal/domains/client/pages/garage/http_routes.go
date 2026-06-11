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

	layout, theme, err := g.getGarageLayout(r.Context(), user.SupertokensID)
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
