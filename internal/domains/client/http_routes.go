package client

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"
)

func prefixRoute(prefix, route string) string {
	if prefix == "/" {
		prefix = ""
	}
	return fmt.Sprintf("%s/%s", prefix, route)
}

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
	landingRoute := prefixRoute(c.baseRoute, "landing/")
	assetsRoute := prefixRoute(c.baseRoute, "assets/")

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
		prefixRoute(c.baseRoute, "signup"):         c.signup,
		prefixRoute(c.baseRoute, "signup/success"): c.signupSuccess,
		prefixRoute(c.baseRoute, "signup/failure"): c.signupFailure,
		prefixRoute(c.baseRoute, "track/visit"):    c.trackVisit,
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
	if ref := r.Referer(); c.publicURL != "" && !strings.HasPrefix(ref, c.publicURL) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	cookie, err := r.Cookie("visitor_id")
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := c.db.Exec(
		r.Context(),
		`INSERT INTO page_events (visitor_id, event)
		 SELECT $1, $2
		 WHERE NOT EXISTS (
		   SELECT 1 FROM page_events
		   WHERE visitor_id = $1
		   AND event = 'visit'
		   AND created_at > now() - interval '24 hours'
		 )`,
		cookie.Value,
		"visit",
	); err != nil {
		log.Printf("track visit: %s", err)
	}

	w.WriteHeader(http.StatusNoContent)
}
