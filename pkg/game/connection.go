package game

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

const (
	NewGameCode = MessageCode("new_game")
	JoinGameCode = MessageCode("join_game")
	RequestLobbiesCode = MessageCode("request_lobbies")
	ErrorMessageCode = MessageCode("error")
)

type MessageCode string

const (
	MenuState = ConnectionState("menu")
	InGameState = ConnectionState("in_game")
)

type ConnectionState string

type Message struct {
	Code MessageCode `json:"code"`
	Content interface{} `json:"content"`
}
const (
	FailedDecodingError = ErrorCode("failed_decoding")
	InvalidMessageCodeError = ErrorCode("invalid_message_code")
)

type ErrorCode string

type ErrorMessage struct {
	Code ErrorCode `json:"code"`
	Text string `json:"text"`
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
	conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }
	id := "temp"

	var m Message
	var state ConnectionState = MenuState
	var g *Game
	for {
		err = conn.ReadJSON(&m)
		if err != nil {
			conn.WriteJSON(Message{Code: ErrorMessageCode, Content: ErrorMessage{FailedDecodingError, err.Error()}})
			continue
		}

		switch state {
		case MenuState:
			switch m.Code {
			case NewGameCode:
				var lobbyName string
				m.GetContent(&lobbyName)
				g = NewGame(conn, id, lobbyName)
				conn.WriteJSON(Message{Code: NewGameCode})
				state = InGameState

			case JoinGameCode:
				var lobbyName string
				m.GetContent(&lobbyName)
				g = NewGame(conn, id, lobbyName)
				conn.WriteJSON(Message{Code: NewGameCode})
				state = InGameState
		
			case RequestLobbiesCode:
				conn.WriteJSON(Message{RequestLobbiesCode, GetUnstartedGames()})
			
			default:
				conn.WriteJSON(Message{Code: ErrorMessageCode, Content: ErrorMessage{InvalidMessageCodeError, ""}})
			}

		case InGameState:
			done := g.HandleMessage(m, id)
			if (done) {
				g = nil
				state = MenuState
			}
		}
	}
}
