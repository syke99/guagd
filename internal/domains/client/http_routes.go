package client

import (
	"io/fs"
	"log"
	"net/http"
	"time"
)

func (c *client) Handlers() map[string]http.HandlerFunc {
	sub, err := fs.Sub(landing, "landing")
	if err != nil {
		log.Printf("error loading landing fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	assetsSub, err := fs.Sub(assets, "assets")
	if err != nil {
		log.Printf("error loading assets fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	fileServer := http.FileServer(http.FS(sub))
	assetsServer := http.FileServer(http.FS(assetsSub))
	landingRoute := c.baseRoute + "landing/"
	assetsRoute := c.baseRoute + "assets/"

	return map[string]http.HandlerFunc{
		landingRoute: func(w http.ResponseWriter, r *http.Request) {
			if _, err := r.Cookie("visitor_id"); err != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     "visitor_id",
					Value:    newVisitorID(),
					Path:     "/",
					Expires:  time.Now().Add(365 * 24 * time.Hour),
					HttpOnly: false,
					SameSite: http.SameSiteLaxMode,
				})
			}
			http.StripPrefix(landingRoute, fileServer).ServeHTTP(w, r)
		},
		assetsRoute: func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix(assetsRoute, assetsServer).ServeHTTP(w, r)
		},
		c.baseRoute + "signup":         c.signup,
		c.baseRoute + "signup/success": c.signupSuccess,
		c.baseRoute + "signup/failure": c.signupFailure,
		c.baseRoute + "track/visit":    c.trackVisit,
	}
}

func (c *client) signup(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, landing, "landing/signup/signup.html")
}

func (c *client) signupSuccess(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, landing, "landing/signup/success.html")
}

func (c *client) signupFailure(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, landing, "landing/signup/failure.html")
}

func (c *client) trackVisit(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("visitor_id")
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := c.db.Exec(
		r.Context(),
		"INSERT INTO page_events (visitor_id, event) VALUES ($1, $2)",
		cookie.Value,
		"visit",
	); err != nil {
		log.Printf("track visit: %s", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
