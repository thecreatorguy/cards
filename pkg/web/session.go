package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var TimeoutDuration = time.Second * 5

type MessageCode string
const (
	ErrorMessageCode = MessageCode("error")
	PingCode = MessageCode("ping")
	PongCode = MessageCode("pong")
	HostGameCode = MessageCode("host_game")
	JoinGameCode = MessageCode("join_game")
	ReconnectedCode = MessageCode("reconnected")
)

type Message struct {
	ID string `json:"id"`
	Code MessageCode `json:"code"`
	Content interface{} `json:"content,omitempty"`
}

func (m Message) GetContent(v interface{}) {
	data, _ := json.Marshal(m.Content)
	json.Unmarshal(data, v)
}

type ErrorCode string
const (
	FailedDecodingError = ErrorCode("failed_decoding")
	InvalidMessageCodeError = ErrorCode("invalid_message_code")
	InvalidLobbyError = ErrorCode("invalid_lobby")
)

type ErrorMessage struct {
	Code ErrorCode `json:"code"`
	Text string `json:"text"`
}

var (
	ErrTimeout = errors.New("timed out")
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Session struct {
	ID string `json:"id"`
	conn *websocket.Conn `json:"-"`
	writeLock *sync.Mutex `json:"-"`
	recieveChannels map[string]chan Message `json:"-"`
	lobby *Lobby `json:"-"`
}

var Sessions = map[string]*Session{}

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
		s.Reconnect(conn)
	} else {
		s := NewSession(id, conn)
		s.Listen()
	}
	
}

func NewSession(id string, conn *websocket.Conn) *Session {
	s := &Session{
		ID: id,
		conn: conn,
		writeLock: &sync.Mutex{},
		recieveChannels: map[string]chan Message{},
	}
	Sessions[id] = s
	return s
}

var closedStatuses = []int{
	websocket.CloseNormalClosure,
	websocket.CloseNoStatusReceived,
	websocket.CloseGoingAway,
}

func (s *Session) SendMessage(m Message) (bool, error) {
	if m.ID == "" {
		m.ID = RandomString(15)
	}
	s.writeLock.Lock()
	err := s.conn.WriteJSON(m)
	s.writeLock.Unlock()

	return websocket.IsCloseError(err, closedStatuses...), err
}

func (s *Session) SendNewMessage(code MessageCode, content interface{}) (string, bool, error) {
	id := NewID()
	closed, err := s.SendMessage(Message{id, code, content})
	return id, closed, err
}

func (s *Session) SendError(ec ErrorCode, text string) {
	s.SendNewMessage(ErrorMessageCode, ErrorMessage{ec, text})
}

func (s *Session) SendInvalidCodeError(code MessageCode) {
	s.SendError(InvalidMessageCodeError, fmt.Sprintf("[%v] is an invalid code", code))
}

func (s *Session) SendInfo(text string) {
	s.SendNewMessage(InfoCode, text)
}

func (s *Session) SendAndRecieveMessage(code MessageCode, content interface{}) (Message, bool, error) {
	id := NewID()
	s.recieveChannels[id] = make(chan Message)
	closed, err := s.SendMessage(Message{id, code, content})
	if closed || err != nil {
		return Message{}, closed, err
	}

	select {
	case rm := <-s.recieveChannels[id]:
		delete(s.recieveChannels, id)
		return rm, false, nil

	case <-time.After(TimeoutDuration):
		s.recieveChannels[id] = nil
		return Message{}, false, ErrTimeout
	}	
}

func (s *Session) Reply(m Message, code MessageCode, content interface{}) (bool, error) {
	return s.SendMessage(Message{m.ID, code, content})
}

func (s *Session) Alive() bool {
	_, closed, err := s.SendAndRecieveMessage(PingCode, nil)
	return !closed && err == nil
}

func (s *Session) Reconnect(conn *websocket.Conn) {
	s.conn = conn
	s.lobby.SendMessage(s, Message{Code: ReconnectedCode})
	s.Listen()
}

func (s *Session) Cleanup() {
	s.conn.WriteControl(websocket.CloseMessage, []byte{}, time.Now().Add(time.Second * 4))
	s.conn.Close()
	delete(Sessions, s.ID)
}

func (s *Session) Listen() {
	for {
		var m Message
		err := s.conn.ReadJSON(&m)
		closed := websocket.IsCloseError(err, closedStatuses...)
		if closed {
			if s.lobby == nil {
				s.Cleanup()
			}
			break
		}
		if err != nil {
			// TODO: fix this to look for more close errors, and retry on regular error
			s.SendError(FailedDecodingError, err.Error())
			if s.lobby == nil {
				s.Cleanup()
			}
			break
		}

		// If waiting for reply, send message to that thread
		if ch, ok := s.recieveChannels[m.ID]; ok {
			if ch == nil {
				delete(s.recieveChannels, m.ID)
			} else {
				ch <- m
			}
			continue
		}

		// If we are being pinged, reply with a pong
		if m.Code == PingCode {
			s.Reply(m, PongCode, nil)
			continue
		}
			
		// If we are in a lobby, that lobby should handle all messages
		if s.lobby != nil {
			s.lobby.SendMessage(s, m)
			continue
		}

		// Otherwise, we are in the init screen, which should connect us with a lobby
		switch m.Code {
		case HostGameCode:
			var payload struct{Nickname string `json:"nickname"`; LobbyName string `json:"lobby_name"`}
			m.GetContent(&payload)
			StartNewLobby(s, payload.Nickname, payload.LobbyName)
		case JoinGameCode:
			var payload struct{Nickname string `json:"nickname"`; Lobby string `json:"lobby"`}
			m.GetContent(&payload)
			if l, ok := Lobbies[payload.Lobby]; ok {
				l.Join(s, payload.Nickname)
			} else {
				s.SendError(InvalidLobbyError, fmt.Sprintf("[%s] is an invalid lobby ID", payload.Lobby))
			}
		default:
			s.SendInvalidCodeError(m.Code)
		}
	}
}
