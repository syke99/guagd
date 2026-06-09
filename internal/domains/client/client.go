package client

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed all:landing
var landing embed.FS

type Client interface {
	Handlers() map[string]http.HandlerFunc
}

type client struct {
	baseRoute string
}

func NewClient(baseRoute string) *client {
	return &client{baseRoute: baseRoute}
}

func (c *client) Handlers() map[string]http.HandlerFunc {
	sub, err := fs.Sub(landing, "landing")
	if err != nil {
		log.Printf("error loading landing fs: %s", err)
		return map[string]http.HandlerFunc{}
	}

	fileServer := http.FileServer(http.FS(sub))
	route := c.baseRoute + "landing/"

	return map[string]http.HandlerFunc{
		route: func(w http.ResponseWriter, r *http.Request) {
			http.StripPrefix(route, fileServer).ServeHTTP(w, r)
		},
	}
}
