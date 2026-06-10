package user

import (
	"encoding/json"
	"guagd/internal/pkg/models"
	"log"
	"net/http"
	"strings"
)

func prefixRoute(prefix, route string) string {
	return strings.TrimRight(prefix, "/") + "/" + route
}

func (u *userClient) Handlers() map[string]http.HandlerFunc {
	routes := map[string]http.HandlerFunc{
		prefixRoute(u.baseRoute, "waitlist/add"): u.addWaitlist,
	}

	return routes
}

func (u *userClient) addWaitlist(w http.ResponseWriter, r *http.Request) {
	var payload models.UserRegisterPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		redirect(w, models.HTMXRedirectResponse{Path: "/signup/failure", Target: "#hero-right"})
		return
	}

	log.Printf("addWaitlist: name=%s email=%s visitor_id=%s", payload.Name, payload.Email, payload.VisitorID)

	if err := u.registerUser(r.Context(), payload.Name, payload.Email, payload.VisitorID); err != nil {
		redirect(w, models.HTMXRedirectResponse{Path: "/signup/failure", Target: "#hero-right"})
		return
	}

	redirect(w, models.HTMXRedirectResponse{Path: "/signup/success", Target: "#hero-right"})
}

func redirect(w http.ResponseWriter, resp models.HTMXRedirectResponse) {
	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Location", string(b))
	w.WriteHeader(http.StatusOK)
}
