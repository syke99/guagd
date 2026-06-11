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

	data := GaragePageData{
		Username:        user.Username,
		IsOwner:         isOwner,
		IsAuthenticated: isAuthenticated,
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
