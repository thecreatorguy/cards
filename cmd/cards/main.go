package main

import "github.com/thecreatorguy/cards/pkg/game"

func main() {
	// r := mux.NewRouter()

	// game.AddRoutes(r, "")

	// port := os.Getenv("PORT")
	// if port == "" {
	// 	port = "8890"
	// }
	
	// fmt.Printf("Listening on port %s...", port)
	// server := &http.Server{
	// 	Handler:        r,
	// 	Addr: 			fmt.Sprintf(":%s", port),
	// 	ReadTimeout:    10 * time.Second,
	// 	WriteTimeout:   10 * time.Second,
	// 	MaxHeaderBytes: 1 << 20,
	// }

	// log.Fatal(server.ListenAndServe())

	game := game.NewDefaultHeartsGame("Tim")
	game.Start()
}