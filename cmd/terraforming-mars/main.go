package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/thecreatorguy/terraforming-mars/pkg/game"
)

func main() {
	r := mux.NewRouter()

	game.AddRoutes(r, "")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8890"
	}
	fmt.Printf("Listening on port %s...", port)
	server := &http.Server{
		Handler:        r,
		Addr: 			fmt.Sprintf(":%s", port),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(server.ListenAndServe())
}