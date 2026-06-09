package user

import (
	"encoding/json"
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
	var payload struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("HX-Location", `{"path":"/signup/failure","target":"#hero-right"}`)
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Printf("register: name=%s email=%s", payload.Name, payload.Email)

	// TODO: persist user
	w.Header().Set("HX-Location", `{"path":"/signup/success","target":"#hero-right"}`)
	w.WriteHeader(http.StatusOK)
}
