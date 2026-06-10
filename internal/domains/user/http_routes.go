package user

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/supertokens/supertokens-golang/recipe/emailpassword"
	"github.com/supertokens/supertokens-golang/recipe/session"

	"guagd/internal/pkg/models"
)

func prefixRoute(prefix, route string) string {
	return strings.TrimRight(prefix, "/") + "/" + route
}

func (u *userClient) Handlers() map[string]http.HandlerFunc {
	routes := map[string]http.HandlerFunc{
		prefixRoute(u.baseRoute, "waitlist/add"): u.addWaitlist,
		prefixRoute(u.baseRoute, "signin"):       u.signIn,
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

func (u *userClient) signIn(w http.ResponseWriter, r *http.Request) {
	var payload models.UserSignInPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		redirect(w, models.HTMXRedirectResponse{Path: "/signin/failure", Target: "#signin-result"})
		return
	}

	log.Printf("signIn: email=%s", payload.Email)

	result, err := emailpassword.SignIn("public", payload.Email, payload.Password)
	if err != nil || result.WrongCredentialsError != nil {
		redirect(w, models.HTMXRedirectResponse{Path: "/signin/failure", Target: "#signin-result"})
		return
	}

	if _, err := session.CreateNewSession(r, w, "public", result.OK.User.ID, nil, nil); err != nil {
		log.Printf("signIn: create session: %s", err)
		redirect(w, models.HTMXRedirectResponse{Path: "/signin/failure", Target: "#signin-result"})
		return
	}

	w.Header().Set("HX-Redirect", "/garage")
	w.WriteHeader(http.StatusOK)
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
