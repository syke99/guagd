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

	appSub, err := fs.Sub(app, "app")
	if err != nil {
		log.Printf("error loading app fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	fileServer := http.FileServer(http.FS(sub))
	assetsServer := http.FileServer(http.FS(assetsSub))
	appServer := http.FileServer(http.FS(appSub))
	landingRoute := prefixRoute(c.baseRoute, "landing/")
	assetsRoute := prefixRoute(c.baseRoute, "assets/")
	appRoute := prefixRoute(c.baseRoute, "app/")

	return map[string]http.HandlerFunc{
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
		prefixRoute(c.baseRoute, "signup"):         c.signupPage,
		prefixRoute(c.baseRoute, "signup/failure"): c.signupFailure,
		prefixRoute(c.baseRoute, "signin"):         c.signinPage,
		prefixRoute(c.baseRoute, "signin/failure"): c.signinFailure,
		prefixRoute(c.baseRoute, "track/visit"):    c.trackVisit,
	}
}

func (c *client) waitlist(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, landing, "landing/waitlist/signup.html")
}

func (c *client) waitlistSuccess(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, landing, "landing/waitlist/success.html")
}

func (c *client) waitlistFailure(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, landing, "landing/waitlist/failure.html")
}

func (c *client) signupPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, landing, "landing/signup/signup.html")
}

func (c *client) signupFailure(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, landing, "landing/signup/failure.html")
}

func (c *client) signinPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, landing, "landing/signin/signin.html")
}

func (c *client) signinFailure(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, landing, "landing/signin/failure.html")
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

	// TODO: after we get supertokens all set up, we're going
	// TODO: to grab the user id from the request context here;
	// TODO: we're then going to do one of three things:
	// TODO: 1. if the user does not have a visitor id set, we'll
	// TODO:	insert it into that user's visitor_id column
	// TODO: 2. if the user has a visitor id, but it's not the same
	// TODO:	as the current one, we update the user's visitor id
	// TODO: 3. if the user has a visitor id and it is the same,
	// TODO:	as the current one, nothing happens

	w.WriteHeader(http.StatusNoContent)
}
