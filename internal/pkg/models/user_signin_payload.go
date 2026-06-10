package models

type UserSignInPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
