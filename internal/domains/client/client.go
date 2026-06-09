package client

import (
	"embed"
)

//go:embed all:landing
var landing embed.FS

type client struct {
	baseRoute string
}

func NewClient(baseRoute string) *client {
	return &client{baseRoute: baseRoute}
}
