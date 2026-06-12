package main

import (
	"log"
	"net/http"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/supertokens/supertokens-golang/supertokens"

	"guagd/cmd/auth"
	"guagd/cmd/config"
	"guagd/internal/domains/account"
	"guagd/internal/domains/client"
	"guagd/internal/domains/upload"
	"guagd/internal/pkg/db"
	"guagd/internal/pkg/storage"
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

			store, err := storage.New(storage.Config{
				AccountID:       cfg.R2AccountID,
				AccessKeyID:     cfg.R2AccessKeyID,
				SecretAccessKey: cfg.R2SecretAccessKey,
				CarPhotos: storage.BucketConfig{
					Name:      cfg.R2CarPhotosBucketName,
					PublicURL: cfg.R2CarPhotosBucketPublicURL,
				},
				AccountPhotos: storage.BucketConfig{
					Name:      cfg.R2AccountPhotosBucketName,
					PublicURL: cfg.R2AccountPhotosBucketPublicURL,
				},
			})
			if err != nil {
				return err
			}
			mux := http.NewServeMux()

			srv, err := server.NewServer(mux, cfg.ServerPort)
			if err != nil {
				return err
			}

			auth.Init(cfg.SuperTokensCoreURL, cfg.PublicURL, cfg.SuperTokensAPIKey)

			clientDomain := client.NewClient("/", cfg.PublicURL, database, store, cfg.HeroBuildID, cfg.HeroGarageID, cfg.HeroClubID)
			accountClient := account.NewAccountClient("/api/v1/accounts/", database)
			uploadClient := upload.NewUploadClient(store)

			srv.RegisterRoutes(clientDomain)
			srv.RegisterRoutes(accountClient)
			srv.RegisterRoutes(uploadClient)
			srv.Wrap(supertokens.Middleware)

			return srv.Serve()
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
