package game

import (
	"bytes"
	"html/template"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func AddRoutes(r *mux.Router, baseURL, appPath, indexTemplateFile, assetsDir string) {
	assetsPrefix := "/assets"
	fs := http.StripPrefix(baseURL + assetsPrefix, http.FileServer(http.Dir(assetsDir)))
	r.PathPrefix(baseURL + assetsPrefix).Handler(fs)

	r.HandleFunc(baseURL + appPath, handleRoot(indexTemplateFile, baseURL, assetsPrefix)).Methods("GET")
	r.HandleFunc(baseURL + "/game", handleGame).Methods("GET")
}

// handleRoot returns a handler that returns the index page with the correct assets path filled in
func handleRoot(indexTemplateFile, baseURL, assetsPrefix string) func(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	pageTemplate := template.Must(template.ParseFiles(indexTemplateFile))
	err := pageTemplate.ExecuteTemplate(&buf, "index", struct{BaseURL string; AssetsPrefix string}{
		baseURL, 
		baseURL + assetsPrefix,
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

func handleGame(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }

	messageType, p, err := conn.ReadMessage()
	if err != nil {
		log.Println(err);
		return;
	}
	log.Println(messageType)
	log.Println(string(p));
	if err := conn.WriteMessage(messageType, p); err != nil {
        log.Println(err)
        return
    }
	conn.Close();
}
