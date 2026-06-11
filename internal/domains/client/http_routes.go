package client

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"

	"guagd/internal/pkg/middleware"
)

func prefixRoute(prefix, route string) string {
	if prefix == "/" {
		prefix = ""
	}
	return fmt.Sprintf("%s/%s", prefix, route)
}

func (c *client) Handlers() map[string]http.HandlerFunc {
	sub, err := fs.Sub(landing, "pages/landing")
	if err != nil {
		log.Printf("error loading landing fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	assetsSub, err := fs.Sub(assets, "assets")
	if err != nil {
		log.Printf("error loading assets fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	appSub, err := fs.Sub(app, "app")
	if err != nil {
		log.Printf("error loading app fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	fileServer := http.FileServer(http.FS(sub))
	assetsServer := http.FileServer(http.FS(assetsSub))
	appServer := http.FileServer(http.FS(appSub))
	landingRoute := prefixRoute(c.baseRoute, "pages/landing/")
	assetsRoute := prefixRoute(c.baseRoute, "assets/")
	appRoute := prefixRoute(c.baseRoute, "app/")

	routes := map[string]http.HandlerFunc{
		"/": func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, app, "app/index.html")
		},
		appRoute: func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix(appRoute, appServer).ServeHTTP(w, r)
		},
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
		prefixRoute(c.baseRoute, "waitlist"):         c.waitlist,
		prefixRoute(c.baseRoute, "waitlist/success"): c.waitlistSuccess,
		prefixRoute(c.baseRoute, "waitlist/failure"): c.waitlistFailure,
		prefixRoute(c.baseRoute, "signup"):           c.signupPage,
		prefixRoute(c.baseRoute, "signup/failure"):   c.signupFailure,
		prefixRoute(c.baseRoute, "signin"):           c.signinPage,
		prefixRoute(c.baseRoute, "signin/failure"):   c.signinFailure,
		prefixRoute(c.baseRoute, "track/visit"):   c.trackVisit,
		prefixRoute(c.baseRoute, "access"):        c.accessPage,
		"/garage/{username}": func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("HX-Request") != "true" {
				http.ServeFileFS(w, r, app, "app/index.html")
				return
			}
			c.garage.GaragePage(w, r)
		},
		"/api/v1/garage/layout":                    middleware.RequireAuth(c.garage.SaveLayout),
		"/api/v1/garage/theme":                     middleware.RequireAuth(c.garage.SaveTheme),
		"/hq/{username}":                           c.hq.HQPage,
	}

	return routes
}

// serveFragment serves an HTML fragment for HTMX swaps, or the full app shell
// for direct navigation (refresh, typed URL). HTMX always sends HX-Request: true.
func (c *client) serveFragment(w http.ResponseWriter, r *http.Request, path string) {
	if r.Header.Get("HX-Request") == "true" {
		http.ServeFileFS(w, r, landing, path)
		return
	}
	http.ServeFileFS(w, r, app, "app/index.html")
}

func (c *client) waitlist(w http.ResponseWriter, r *http.Request) {
	c.serveFragment(w, r, "pages/landing/waitlist/signup.html")
}

func (c *client) waitlistSuccess(w http.ResponseWriter, r *http.Request) {
	c.serveFragment(w, r, "pages/landing/waitlist/success.html")
}

func (c *client) waitlistFailure(w http.ResponseWriter, r *http.Request) {
	c.serveFragment(w, r, "pages/landing/waitlist/failure.html")
}

func (c *client) signupPage(w http.ResponseWriter, r *http.Request) {
	c.serveFragment(w, r, "pages/landing/signup/signup.html")
}

func (c *client) signupFailure(w http.ResponseWriter, r *http.Request) {
	c.serveFragment(w, r, "pages/landing/signup/failure.html")
}

func (c *client) signinPage(w http.ResponseWriter, r *http.Request) {
	c.serveFragment(w, r, "pages/landing/signin/signin.html")
}

func (c *client) signinFailure(w http.ResponseWriter, r *http.Request) {
	c.serveFragment(w, r, "pages/landing/signin/failure.html")
}

func (c *client) accessPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html><html><body><script>localStorage.setItem('gaugd_early_access','true');window.location.href='/';</script></body></html>`)
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

	sessionRequired := false
	sessionContainer, err := session.GetSession(r, w, &sessmodels.VerifySessionOptions{
		SessionRequired: &sessionRequired,
	})
	if err == nil && sessionContainer != nil {
		userID := sessionContainer.GetUserID()
		if err := c.db.Exec(
			r.Context(),
			`UPDATE accounts SET visitor_id = $1
			 WHERE supertokens_id = $2
			 AND COALESCE(visitor_id, '') != $1`,
			cookie.Value,
			userID,
		); err != nil {
			log.Printf("track visit: update visitor_id: %s", err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
