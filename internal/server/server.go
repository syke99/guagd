package server

import (
	"errors"
	"guagd/internal/domains"
	"log"
	"net/http"
)

type Server interface {
	RegisterRoutes(domain domains.Domain) Server
	Serve() error
}

type server struct {
	mux  *http.ServeMux
	port string
}

func NewServer(mux *http.ServeMux, port string) (Server, error) {
	if mux == nil {
		return nil, errors.New("mux must not be nil")
	}

	if port == "" {
		return nil, errors.New("port must not be empty")
	}

	return &server{mux: mux, port: port}, nil
}

func (s *server) RegisterRoutes(domain domains.Domain) Server {
	for route, handler := range domain.Handlers() {
		if route == "" || handler == nil {
			continue
		}
		s.mux.HandleFunc(route, handler)
	}
	return s
}

func (s *server) Serve() error {
	log.Printf("Listening on %s", s.port)
	return http.ListenAndServe(s.port, s.mux)
}
