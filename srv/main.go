package main

import (
	"io/fs"
	"log"
	"net/http"

	assets "guagd"
)

func main() {
	landing, err := fs.Sub(assets.Landing, "client/landing")
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", http.FileServer(http.FS(landing)))

	log.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
