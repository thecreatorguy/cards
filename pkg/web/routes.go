package web

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

const (
	AssetsPrefix = "/assets"
)

var (
	//go:embed static
	assets embed.FS
	//go:embed views
	views embed.FS
)

func AddRoutes(r *mux.Router, basePath string) {
	fsys, err := fs.Sub(assets, "static")
	if err != nil {
		panic(err)
	}

	templates := template.Must(template.ParseFS(views, "views/*"))

	r.PathPrefix(basePath + AssetsPrefix).Handler(
		http.StripPrefix(basePath + AssetsPrefix, http.FileServer(http.FS(fsys))),
	)

	r.HandleFunc(basePath + "/waitingroom", handleWaitingRoom(basePath, templates)).Methods("GET")
	r.HandleFunc(basePath + "/lobby/list", handleLobbyList).Methods("GET")
	r.HandleFunc(basePath + "/game", handleGame(basePath, templates)).Methods("GET")
	r.HandleFunc(basePath + "/game/websocket", makeConnection).Methods("GET")
}

// handleApp returns a handler that returns the index page with the correct assets path filled in
func handleWaitingRoom(basePath string, templates *template.Template) func(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, "waitingroom", struct{BasePath string; AssetsPrefix string}{
		basePath, 
		basePath + AssetsPrefix,
	})
	if err != nil {
		log.Fatal(err)
		return nil
	}

	lobbyMenu := buf.Bytes()
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(lobbyMenu)
	}
}

func handleLobbyList(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetUnstartedLobbies())
}

// handleApp returns a handler that returns the index page with the correct assets path filled in
func handleGame(basePath string, templates *template.Template) func(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, "game", struct{BasePath string; AssetsPrefix string}{
		basePath, 
		basePath + AssetsPrefix,
	})
	if err != nil {
		log.Fatal(err)
		return nil
	}

	game := buf.Bytes()
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(game)
	}
}
