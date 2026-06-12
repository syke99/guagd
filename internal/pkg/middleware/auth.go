package middleware

import (
	"context"
	"net/http"

	"github.com/supertokens/supertokens-golang/recipe/session"
)

type contextKey string

const (
	ContextKeyUserID    contextKey = "userID"
	ContextKeyAccountID contextKey = "accountID"
)

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionContainer, err := session.GetSession(r, w, nil)
		if err != nil {
			w.Header().Set("HX-Redirect", "/signin")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ContextKeyUserID, sessionContainer.GetUserID())
		if payload := sessionContainer.GetAccessTokenPayload(); payload != nil {
			if accountID, ok := payload["account_id"].(string); ok && accountID != "" {
				ctx = context.WithValue(ctx, ContextKeyAccountID, accountID)
			}
		}
		next(w, r.WithContext(ctx))
	}
}
