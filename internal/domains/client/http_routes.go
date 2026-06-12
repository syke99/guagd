package client

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func prefixRoute(prefix, route string) string {
	return strings.TrimRight(prefix, "/") + "/" + route
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

	sessionContainer, _ := c.sessions.GetOptionalSession(r, w)
	if sessionContainer != nil {
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
