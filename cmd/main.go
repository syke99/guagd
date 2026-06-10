package main

import (
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli/v2"

	"guagd/cmd/config"
	"guagd/internal/domains/client"
	"guagd/internal/domains/user"
	"guagd/internal/pkg/db"
	"guagd/internal/server"
)

func main() {
	app := &cli.App{
		Name: "guagd",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "path to a config file (json or yaml)",
			},
		},
		Action: func(c *cli.Context) error {
			var cfg config.Config
			var err error

			if path := c.String("config"); path != "" {
				cfg, err = config.Load(config.WithConfigFile(path))
			} else {
				cfg, err = config.Load()
			}
			if err != nil {
				return err
			}

			database, err := db.Connect(cfg.DatabaseURL)
			if err != nil {
				return err
			}

			mux := http.NewServeMux()

			srv, err := server.NewServer(mux, cfg.ServerPort)
			if err != nil {
				return err
			}

			clientDomain := client.NewClient("/", cfg.PublicURL, database)
			userClient := user.NewUserClient("api/v1/users/", database)

			srv.RegisterRoutes(clientDomain)
			srv.RegisterRoutes(userClient)

			return srv.Serve()
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
