package client

import (
	"io/fs"
	"log"
	"net/http"
)

func (c *client) Handlers() map[string]http.HandlerFunc {
	sub, err := fs.Sub(landing, "landing")
	if err != nil {
		log.Printf("error loading landing fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	fileServer := http.FileServer(http.FS(sub))
	landingRoute := c.baseRoute + "landing/"

	return map[string]http.HandlerFunc{
		landingRoute: func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix(landingRoute, fileServer).ServeHTTP(w, r)
		},
		c.baseRoute + "signup":         c.signup,
		c.baseRoute + "signup/success": c.signupSuccess,
		c.baseRoute + "signup/failure": c.signupFailure,
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
