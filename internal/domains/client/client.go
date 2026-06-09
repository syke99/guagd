package client

import (
	"embed"
)

//go:embed all:landing
var landing embed.FS

//go:embed all:assets
var assets embed.FS

type client struct {
	baseRoute string
}

func NewClient(baseRoute string) *client {
	return &client{baseRoute: baseRoute}
}
