package client

import (
	"crypto/rand"
	"embed"
	"fmt"

	"guagd/internal/domains/client/garage"
	"guagd/internal/pkg/db"
)

//go:embed all:landing
var landing embed.FS

//go:embed all:assets
var assets embed.FS

//go:embed all:app
var app embed.FS

type client struct {
	baseRoute string
	publicURL string
	db        db.DB
	garage    *garage.GarageClient
}

func NewClient(baseRoute string, publicURL string, db db.DB) *client {
	return &client{
		baseRoute: baseRoute,
		publicURL: publicURL,
		db:        db,
		garage:    garage.NewGarageClient(db),
	}
}

func newVisitorID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
