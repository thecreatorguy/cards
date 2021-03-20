package game

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type ErrorCode string
const (
	FailedDecodingError = ErrorCode("failed_decoding")
	InvalidMessageCodeError = ErrorCode("invalid_message_code")
)

type ErrorMessage struct {
	Code ErrorCode `json:"code"`
	Text string `json:"text"`
}

type MessageCode string
const (
	JoinGameCode = MessageCode("join_game")
	ErrorMessageCode = MessageCode("error")
)
type Message struct {
	Code MessageCode `json:"code"`
	Content interface{} `json:"content"`
}

func (m Message) GetContent(v interface{}) {
	data, _ := json.Marshal(m.Content)
	json.Unmarshal(data, v)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func makeConnection(w http.ResponseWriter, r *http.Request) {
	var session string
	for _, c := range r.Cookies() {
		if c.Name == "session" {
			session = c.Value
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }

	var m Message
	var g *Game
	for {
		err = conn.ReadJSON(&m)
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived, websocket.CloseGoingAway) {
				break
			}
			conn.WriteJSON(Message{Code: ErrorMessageCode, Content: ErrorMessage{FailedDecodingError, err.Error()}})
			continue
		}

		if (m.Code == JoinGameCode) {
			log.Println(m)
		}

		g.HandleMessage(m, session)
	}
}

func addToLobby() {
	
}
