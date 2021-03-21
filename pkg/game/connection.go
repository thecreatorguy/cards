package game

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

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
	PingCode = MessageCode("ping")
	SetupCode = MessageCode("setup")
	HostGameCode = MessageCode("host_game")
	JoinGameCode = MessageCode("join_game")
	UpdateCode = MessageCode("update")
	UpdateLobbySettingsCode = MessageCode("update_lobby_settings")
	StartGameCode = MessageCode("start_game")
	FirstGenCode = MessageCode("first_gen")
	ErrorMessageCode = MessageCode("error")
)
type Message struct {
	Code MessageCode `json:"code"`
	Content interface{} `json:"content,omitempty"`
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
	var id string
	for _, c := range r.Cookies() {
		if c.Name == "session" {
			id = c.Value
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }

	if s, ok := Sessions[id]; ok {
		// check if original is still alive, if it is then terminate this one
		// if not, replace connection. if in game, let game know. if not, start lobby
		s.Conn = conn
	} else {
		s := NewSession(id, conn)
		s.SendMessage(Message{Code: SetupCode})
		s.Lobby()
	}
	
}

var Sessions = map[string]*Session{}

type Session struct {
	ID string
	Conn *websocket.Conn
	ReadLock *sync.Mutex
	WriteLock *sync.Mutex
}

func NewSession(id string, conn *websocket.Conn) *Session {
	s := &Session{id, conn, &sync.Mutex{}, &sync.Mutex{}}
	Sessions[id] = s
	return s
}

func (s *Session) Lobby() {
	for {
		m, closed, err := s.ReceiveMessage()
		if closed {
			delete(Sessions, s.ID)
			break
		}
		if err != nil {
			closed, err := s.SendMessage(Message{ErrorMessageCode, ErrorMessage{FailedDecodingError, err.Error()}})
			if closed {
				break
			}
			if err != nil {
				log.Println(err)
			}
			continue
		}

		switch m.Code {
		case HostGameCode:
			var payload struct{Nickname string `json:"nickname"`; LobbyName string `json:"lobby_name"`}
			m.GetContent(&payload)
			NewGame(s, payload.LobbyName, payload.Nickname)
			return
		case JoinGameCode:
			var payload struct{Nickname string `json:"nickname"`; LobbyName string `json:"lobby_name"`}
			m.GetContent(&payload)
			Games[payload.LobbyName].Join(s, payload.Nickname)
			return
		default:
			s.SendInvalidCode()
		}		
	}
}

var closedStatuses = []int{
	websocket.CloseNormalClosure,
	websocket.CloseNoStatusReceived,
	websocket.CloseGoingAway,
}

func (s *Session) ReceiveMessage() (Message, bool, error) {
	var m Message

	s.ReadLock.Lock()
	err := s.Conn.ReadJSON(&m)
	s.ReadLock.Unlock()
	
	return m, websocket.IsCloseError(err, closedStatuses...), err
}

func (s *Session) SendMessage(m Message) (bool, error) {
	s.WriteLock.Lock()
	err := s.Conn.WriteJSON(m)
	s.WriteLock.Unlock()

	return websocket.IsCloseError(err, closedStatuses...), err
}

func (s *Session) Alive() bool {
	closed, err := s.SendMessage(Message{Code: PingCode})
	if closed || err != nil {
		return false
	}

	_, closed, err = s.ReceiveMessage()
	return !closed && err == nil
}

func (s *Session) SendInvalidCode() {
	s.SendMessage(Message{ErrorMessageCode, ErrorMessage{InvalidMessageCodeError, "Invalid message code"}})
}