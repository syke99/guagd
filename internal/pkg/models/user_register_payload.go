package models

type UserRegisterPayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
