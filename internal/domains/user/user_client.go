package user

import (
	"log"
	"net/http"
)

type UserClient interface {
	Handlers() map[string]http.HandlerFunc
}

type userClient struct {
	baseRoute string
}

func NewUserClient(baseRoute string) *userClient {
	return &userClient{baseRoute: baseRoute}
}

func (u *userClient) Handlers() map[string]http.HandlerFunc {
	routes := map[string]http.HandlerFunc{
		"register": u.register,
	}

	prefixed := make(map[string]http.HandlerFunc, len(routes))
	for route, handler := range routes {
		prefixed[u.baseRoute+route] = handler
	}

	return prefixed
}

func (u *userClient) register(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	log.Printf("email: %s", email)
}
