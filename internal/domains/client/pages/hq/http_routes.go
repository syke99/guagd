package hq

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"

	"guagd/internal/pkg/middleware"
)

func (h *HQClient) HQPage(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.PathValue("username"), "@")
	if username == "" {
		http.NotFound(w, r)
		return
	}

	user, err := h.getUserByUsername(r.Context(), username)
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

	layout, theme, err := h.getHQLayout(r.Context(), user.SupertokensID)
	if err != nil {
		log.Printf("hqPage: get layout: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	members, err := h.getMembers(r.Context(), user.SupertokensID)
	if err != nil {
		log.Printf("hqPage: get members: %s", err)
		members = []HQMember{}
	}

	data := HQPageData{
		Username:        user.Username,
		IsOwner:         isOwner,
		IsAuthenticated: isAuthenticated,
		MemberCount:     len(members),
		Members:         members,
		Layout:          layout,
		SafeCSS:         buildThemeCSS(theme),
	}

	w.Header().Set("Content-Type", "text/html")
	if err := hqTemplate.ExecuteTemplate(w, "hq.html", data); err != nil {
		log.Printf("hqPage: render: %s", err)
	}
}

func (h *HQClient) SaveLayout(w http.ResponseWriter, r *http.Request) {
	var layout []LayoutItem
	if err := json.NewDecoder(r.Body).Decode(&layout); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	if err := h.upsertLayout(r.Context(), userID, layout); err != nil {
		log.Printf("hq saveLayout: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HQClient) SaveTheme(w http.ResponseWriter, r *http.Request) {
	var theme map[string]map[string]string
	if err := json.NewDecoder(r.Body).Decode(&theme); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value(middleware.ContextKeyUserID).(string)
	if err := h.upsertTheme(r.Context(), userID, theme); err != nil {
		log.Printf("hq saveTheme: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HQClient) SearchMembers(w http.ResponseWriter, r *http.Request) {
	sessionRequired := true
	sessionContainer, err := session.GetSession(r, w, &sessmodels.VerifySessionOptions{
		SessionRequired: &sessionRequired,
	})
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	payload := sessionContainer.GetAccessTokenPayload()
	if t, _ := payload["acct_type"].(string); t != "club" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	clubID := sessionContainer.GetUserID()

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	results, err := h.searchNonMembers(r.Context(), clubID, q)
	if err != nil {
		log.Printf("searchMembers: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	for _, m := range results {
		fmt.Fprintf(w, `<a class="search-result-item" href="#">@%s<span class="search-result-badge">Garage</span></a>`,
			m.Username)
	}
}

func (h *HQClient) ListMembers(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimPrefix(r.URL.Query().Get("username"), "@")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	user, err := h.getUserByUsername(r.Context(), username)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	members, err := h.getMembers(r.Context(), user.SupertokensID)
	if err != nil {
		log.Printf("listMembers: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

func (h *HQClient) AddMember(w http.ResponseWriter, r *http.Request) {
	sessionRequired := true
	sessionContainer, err := session.GetSession(r, w, &sessmodels.VerifySessionOptions{
		SessionRequired: &sessionRequired,
	})
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	payload := sessionContainer.GetAccessTokenPayload()
	acctType, _ := payload["acct_type"].(string)
	if acctType != "club" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	clubUsername, _ := payload["username"].(string)
	clubID := sessionContainer.GetUserID()

	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	memberUsername := strings.TrimPrefix(strings.TrimSpace(body.Username), "@")
	if memberUsername == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	if err := h.addMember(r.Context(), clubID, memberUsername); err != nil {
		log.Printf("hq addMember: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/hq/@"+clubUsername)
	w.WriteHeader(http.StatusOK)
}

func (h *HQClient) RemoveMember(w http.ResponseWriter, r *http.Request) {
	sessionRequired := true
	sessionContainer, err := session.GetSession(r, w, &sessmodels.VerifySessionOptions{
		SessionRequired: &sessionRequired,
	})
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	payload := sessionContainer.GetAccessTokenPayload()
	acctType, _ := payload["acct_type"].(string)
	if acctType != "club" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	clubUsername, _ := payload["username"].(string)
	clubID := sessionContainer.GetUserID()

	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	memberUsername := strings.TrimPrefix(strings.TrimSpace(body.Username), "@")
	if memberUsername == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	if err := h.removeMember(r.Context(), clubID, memberUsername); err != nil {
		log.Printf("hq removeMember: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("HX-Redirect", "/hq/@"+clubUsername)
	w.WriteHeader(http.StatusOK)
}
