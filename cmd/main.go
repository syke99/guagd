package main

import (
	"log"
	"net/http"

	"guagd/internal/domains/client"
	"guagd/internal/domains/user"
	"guagd/internal/server"
)

func main() {
	mux := http.NewServeMux()

	srv, err := server.NewServer(mux, ":8080")
	if err != nil {
		log.Fatal(err)
	}

	clientDomain := client.NewClient("/")
	userClient := user.NewUserClient("/users/")

	srv.RegisterRoutes(clientDomain.Handlers())
	srv.RegisterRoutes(userClient.Handlers())

	if err := srv.Serve(); err != nil {
		log.Fatal(err)
	}
}
