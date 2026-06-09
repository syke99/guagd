package domains

import "net/http"

type Domain interface {
	Handlers() map[string]http.HandlerFunc
}
