package sessions

import (
	"net/http"

	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"
)

// Session is the subset of sessmodels.SessionContainer that handlers need.
type Session interface {
	GetUserID() string
	GetAccessTokenPayload() map[string]interface{}
	RevokeSession() error
}

// Getter abstracts session.GetSession so handlers can be tested without a
// running SuperTokens sidecar.
type Getter interface {
	// GetSession requires a valid session; returns an error if none exists.
	GetSession(r *http.Request, w http.ResponseWriter) (Session, error)
	// GetOptionalSession returns nil, nil when no session is present.
	GetOptionalSession(r *http.Request, w http.ResponseWriter) (Session, error)
}

// SuperTokensGetter is the production implementation backed by the SuperTokens SDK.
type SuperTokensGetter struct{}

func (s *SuperTokensGetter) GetSession(r *http.Request, w http.ResponseWriter) (Session, error) {
	required := true
	sc, err := session.GetSession(r, w, &sessmodels.VerifySessionOptions{SessionRequired: &required})
	if err != nil || sc == nil {
		return nil, err
	}
	return &stSession{sc}, nil
}

func (s *SuperTokensGetter) GetOptionalSession(r *http.Request, w http.ResponseWriter) (Session, error) {
	required := false
	sc, err := session.GetSession(r, w, &sessmodels.VerifySessionOptions{SessionRequired: &required})
	if err != nil || sc == nil {
		return nil, err
	}
	return &stSession{sc}, nil
}

// stSession wraps sessmodels.SessionContainer (which uses function fields, not methods)
// so it satisfies the Session interface.
type stSession struct{ sc sessmodels.SessionContainer }

func (s *stSession) GetUserID() string                        { return s.sc.GetUserID() }
func (s *stSession) GetAccessTokenPayload() map[string]interface{} { return s.sc.GetAccessTokenPayload() }
func (s *stSession) RevokeSession() error                     { return s.sc.RevokeSession() }
