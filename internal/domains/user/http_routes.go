package user

import (
	"encoding/json"
	"guagd/internal/pkg/models"
	"log"
	"net/http"
)

func (u *userClient) Handlers() map[string]http.HandlerFunc {
	routes := map[string]http.HandlerFunc{
		"register": u.register,
	}

	prefixed := make(map[string]http.HandlerFunc, len(routes))
	for route, handler := range routes {
		prefixed[u.baseRoute+route] = handler
	}

	return prefixed
}

func (u *userClient) register(w http.ResponseWriter, r *http.Request) {
	var payload models.UserRegisterPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		redirect(w, models.HTMXRedirectResponse{Path: "/signup/failure", Target: "#hero-right"})
		return
	}

	log.Printf("register: name=%s email=%s", payload.Name, payload.Email)

	if err := u.registerUser(r.Context(), payload.Name, payload.Email); err != nil {
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
