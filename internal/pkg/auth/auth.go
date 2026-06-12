package auth

import (
	"net/http"

	"github.com/supertokens/supertokens-golang/recipe/emailpassword"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"
)

type Auth interface {
	SignUp(email, password string) (userID string, emailExists bool, err error)
	SignIn(email, password string) (userID string, wrongCreds bool, err error)
	CreateSession(r *http.Request, w http.ResponseWriter, userID string, claims map[string]any) error
	RevokeSession(r *http.Request, w http.ResponseWriter) error
}

type SuperTokensAuth struct{}

func (a *SuperTokensAuth) SignUp(email, password string) (string, bool, error) {
	result, err := emailpassword.SignUp("public", email, password)
	if err != nil {
		return "", false, err
	}
	if result.EmailAlreadyExistsError != nil {
		return "", true, nil
	}
	return result.OK.User.ID, false, nil
}

func (a *SuperTokensAuth) SignIn(email, password string) (string, bool, error) {
	result, err := emailpassword.SignIn("public", email, password)
	if err != nil {
		return "", false, err
	}
	if result.WrongCredentialsError != nil {
		return "", true, nil
	}
	return result.OK.User.ID, false, nil
}

func (a *SuperTokensAuth) CreateSession(r *http.Request, w http.ResponseWriter, userID string, claims map[string]any) error {
	_, err := session.CreateNewSession(r, w, "public", userID, claims, nil)
	return err
}

func (a *SuperTokensAuth) RevokeSession(r *http.Request, w http.ResponseWriter) error {
	sessionRequired := false
	sc, err := session.GetSession(r, w, &sessmodels.VerifySessionOptions{
		SessionRequired: &sessionRequired,
	})
	if err != nil || sc == nil {
		return nil
	}
	return sc.RevokeSession()
}
