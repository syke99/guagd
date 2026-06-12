package account

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"

	"guagd/internal/pkg/models"
)

func prefixRoute(prefix, route string) string {
	return strings.TrimRight(prefix, "/") + "/" + route
}

func (u *accountClient) Handlers() map[string]http.HandlerFunc {
	routes := map[string]http.HandlerFunc{
		prefixRoute(u.baseRoute, "waitlist/add"): u.addWaitlist,
		prefixRoute(u.baseRoute, "signup"):       u.signUp,
		prefixRoute(u.baseRoute, "signin"):       u.signIn,
		prefixRoute(u.baseRoute, "signout"):      u.signOut,
		prefixRoute(u.baseRoute, "search"):       u.search,
	}

	return routes
}

func (u *accountClient) addWaitlist(w http.ResponseWriter, r *http.Request) {
	var payload models.UserRegisterPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		redirect(w, models.HTMXRedirectResponse{Path: "/waitlist/failure", Target: "#hero-right"})
		return
	}

	log.Printf("addWaitlist: name=%s email=%s visitor_id=%s", payload.Name, payload.Email, payload.VisitorID)

	if err := u.registerUser(r.Context(), payload.Name, payload.Email, payload.VisitorID); err != nil {
		redirect(w, models.HTMXRedirectResponse{Path: "/waitlist/failure", Target: "#hero-right"})
		return
	}

	redirect(w, models.HTMXRedirectResponse{Path: "/waitlist/success", Target: "#hero-right"})
}

func (u *accountClient) signUp(w http.ResponseWriter, r *http.Request) {
	var payload models.UserSignUpPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		redirect(w, models.HTMXRedirectResponse{Path: "/signup/failure", Target: "#signup-result"})
		return
	}

	if payload.AcctType != "driver" && payload.AcctType != "club" {
		redirect(w, models.HTMXRedirectResponse{Path: "/signup/failure", Target: "#signup-result"})
		return
	}

	log.Printf("signUp: email=%s username=%s acct_type=%s", payload.Email, payload.Username, payload.AcctType)

	userID, emailExists, err := u.auth.SignUp(payload.Email, payload.Password)
	if err != nil || emailExists {
		redirect(w, models.HTMXRedirectResponse{Path: "/signup/failure", Target: "#signup-result"})
		return
	}

	if err := u.createAccount(r.Context(), userID, payload.Username, payload.Email, payload.AcctType); err != nil {
		log.Printf("signUp: db insert: %s", err)
		redirect(w, models.HTMXRedirectResponse{Path: "/signup/failure", Target: "#signup-result"})
		return
	}

	info, err := u.getAccountBySupertokensID(r.Context(), userID)
	if err != nil {
		log.Printf("signUp: get account: %s", err)
		redirect(w, models.HTMXRedirectResponse{Path: "/signup/failure", Target: "#signup-result"})
		return
	}

	if err := u.auth.CreateSession(r, w, userID, map[string]any{
		"account_id": info.AccountID,
		"username":   payload.Username,
		"acct_type":  payload.AcctType,
	}); err != nil {
		log.Printf("signUp: create session: %s", err)
		redirect(w, models.HTMXRedirectResponse{Path: "/signup/failure", Target: "#signup-result"})
		return
	}

	dest := "/garage/@" + payload.Username
	if payload.AcctType == "club" {
		dest = "/hq/@" + payload.Username
	}
	w.Header().Set("HX-Redirect", dest)
	w.WriteHeader(http.StatusOK)
}

func (u *accountClient) signIn(w http.ResponseWriter, r *http.Request) {
	var payload models.UserSignInPayload

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		redirect(w, models.HTMXRedirectResponse{Path: "/signin/failure", Target: "#signin-result"})
		return
	}

	log.Printf("signIn: email=%s", payload.Email)

	userID, wrongCreds, err := u.auth.SignIn(payload.Email, payload.Password)
	if err != nil || wrongCreds {
		redirect(w, models.HTMXRedirectResponse{Path: "/signin/failure", Target: "#signin-result"})
		return
	}

	info, err := u.getAccountBySupertokensID(r.Context(), userID)
	if err != nil {
		log.Printf("signIn: get account: %s", err)
		redirect(w, models.HTMXRedirectResponse{Path: "/signin/failure", Target: "#signin-result"})
		return
	}

	if err := u.auth.CreateSession(r, w, userID, map[string]any{
		"account_id": info.AccountID,
		"username":   info.Username,
		"acct_type":  info.AcctType,
	}); err != nil {
		log.Printf("signIn: create session: %s", err)
		redirect(w, models.HTMXRedirectResponse{Path: "/signin/failure", Target: "#signin-result"})
		return
	}

	dest := "/garage/@" + info.Username
	if info.AcctType == "club" {
		dest = "/hq/@" + info.Username
	}
	w.Header().Set("HX-Redirect", dest)
	w.WriteHeader(http.StatusOK)
}

func (u *accountClient) signOut(w http.ResponseWriter, r *http.Request) {
	if err := u.auth.RevokeSession(r, w); err != nil {
		log.Printf("signOut: revoke session: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (u *accountClient) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	acctType := r.URL.Query().Get("type")

	results, err := u.searchAccounts(r.Context(), q, acctType)
	if err != nil {
		log.Printf("search: %s", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	for _, res := range results {
		url := searchPageURL(res.Username, res.AcctType)
		label := searchPageLabel(res.AcctType)
		fmt.Fprintf(w, `<a class="search-result-item" href="%s">@%s<span class="search-result-badge">%s</span></a>`,
			url, html.EscapeString(res.Username), label)
	}
}

func searchPageURL(username, acctType string) string {
	switch acctType {
	case "club":
		return "/hq/@" + username
	case "shop":
		return "/shop/@" + username
	default:
		return "/garage/@" + username
	}
}

func searchPageLabel(acctType string) string {
	switch acctType {
	case "club":
		return "HQ"
	case "shop":
		return "Shop"
	default:
		return "Garage"
	}
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
