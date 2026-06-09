package user

import (
	"context"

	"guagd/internal/pkg/db"
)

type userClient struct {
	baseRoute string
	db        db.DB
}

func NewUserClient(baseRoute string, db db.DB) *userClient {
	return &userClient{baseRoute: baseRoute, db: db}
}

func (u *userClient) registerUser(ctx context.Context, name, email, visitorId string) error {
	return u.db.Exec(
		ctx,
		"INSERT INTO users (name, email, visitor_id) VALUES ($1, $2, $3)",
		name,
		email,
		visitorId,
	)
}
