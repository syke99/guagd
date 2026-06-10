package models

type UserSignUpPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
	AcctType string `json:"acct_type"`
}
