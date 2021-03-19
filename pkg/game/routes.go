package game

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	
	AssetsPrefix = "/assets"
)

var (
	//go:embed static
	Assets embed.FS
	//go:embed views
	Views embed.FS
)


func AddRoutes(r *mux.Router, baseURL string) {
	// BaseURL = baseURL
	fs.Sub()
	fs := http.StripPrefix(baseURL + AssetsPrefix, http.FileServer(http.Dir(assetsDir)))
	r.PathPrefix(baseURL + AssetsPrefix).Handler(fs)

	r.HandleFunc(baseURL + "/lobby", handleLobbyMenu(indexTemplateFile, baseURL)).Methods("GET")
	r.HandleFunc(baseURL + "/game", handleGame()).Methods("GET")
	r.HandleFunc(baseURL + "/game/websocket", makeConnection).Methods("GET")
}

// handleApp returns a handler that returns the index page with the correct assets path filled in
func handleLobbyMenu(indexTemplateFile, baseURL string) func(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	pageTemplate := template.Must(template.ParseFiles(indexTemplateFile))
	err := pageTemplate.ExecuteTemplate(&buf, "index", struct{BaseURL string; AssetsPrefix string}{
		baseURL, 
		baseURL + AssetsPrefix,
	})
	if err != nil {
		log.Fatal(err)
		return nil
	}

	index := buf.Bytes()
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(index)
	}
}

// handleApp returns a handler that returns the index page with the correct assets path filled in
func handleGame(indexTemplateFile, baseURL string) func(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	pageTemplate := template.Must(template.ParseFiles(indexTemplateFile))
	err := pageTemplate.ExecuteTemplate(&buf, "index", struct{BaseURL string; AssetsPrefix string}{
		baseURL, 
		baseURL + AssetsPrefix,
	})
	if err != nil {
		log.Fatal(err)
		return nil
	}

	index := buf.Bytes()
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(index)
	}
}
