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
	"github.com/thecreatorguy/cards/pkg/game"
)

var TimeoutDuration = time.Second * 5

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

type MessageCode string
const (
	// Two Way Codes
	ErrorMessageCode = MessageCode("error")
	PingCode = MessageCode("ping")
	PongCode = MessageCode("pong")

	// Recieving Codes
	HostGameCode = MessageCode("host_game")
	JoinGameCode = MessageCode("join_game")
	RefreshCode = MessageCode("refresh")
	UpdateLobbySettingsCode = MessageCode("update_lobby_settings")
	StartGameCode = MessageCode("start_game")
	PassedCardsCode = MessageCode("passed_cards")
	PlayedCardCode = MessageCode("played_card")

	// Sending Codes
	InfoCode = MessageCode("info")
	UpdateLobbyCode = MessageCode("update_lobby")
	UpdateCode = MessageCode("update")
	PassCardsCode = MessageCode("pass_cards")
	PlayCardCode = MessageCode("play_card")
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

type LobbyMessage struct {
	Dead bool
	Source *Session
	Message Message
}

type GameState string
const (
	InLobbyState = GameState("in_lobby")
	InGameState = GameState("in_game")
)

type Settings struct {
	MaxPoints int `json:"max_points"`
}

type Player struct {
	Name string `json:"name"`
	CPU bool `json:"cpu"`
	Session *Session `json:"-"`
	AnswerChannel chan Message `json:"-"`
}
type Lobby struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Settings Settings `json:"settings"`
	State GameState `json:"state"`
	Players []*Player `json:"players"`
	Game game.HeartsGame `json:"-"`
	listener chan LobbyMessage `json:"-"`
	lock *sync.Mutex `json:"-"`
}

var Lobbies = map[string]*Lobby{}
var SessionToLobby = map[string]*Lobby{}

var (
	ErrTimeout = errors.New("timed out")
)

//----------------------------------------------------//
//------------------- Connection ---------------------//
//----------------------------------------------------//

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
		fmt.Println("timed out")
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
	s.lobby.SendMessage(LobbyMessage{false, s, Message{Code: RefreshCode}})
	s.Listen()
}

func (s *Session) Cleanup() {
	delete(Sessions, s.ID)
}

func (s *Session) Listen() {
	for {
		var m Message
		err := s.conn.ReadJSON(&m)
		closed := websocket.IsCloseError(err, closedStatuses...)
		if closed {
			s.Cleanup()
			break
		}
		if err != nil {
			s.SendError(FailedDecodingError, err.Error())
			// closed, err := s.SendError(FailedDecodingError, err.Error())
			// if closed {
			// 	s.Cleanup()
			// 	break
			// }
			// if err != nil {
			// 	log.Println(err)
			// }
			continue
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
			s.lobby.SendMessage(LobbyMessage{Dead: false, Source: s, Message: m})
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

//----------------------------------------------------//
//---------------------- Lobby -----------------------//
//----------------------------------------------------//

func (l *Lobby) Started() bool {
	return l.State != InLobbyState
}

func GetUnstartedLobbies() []*Lobby {
	unstarted := []*Lobby{}
	for _, l := range Lobbies {
		if (!l.Started()) {
			unstarted = append(unstarted, l)
		}
	}
	return unstarted
}

func StartNewLobby(host *Session, nickname string, lobbyName string) {
	lobby := &Lobby{
		ID: RandomString(12),
		Name: lobbyName,
		Settings: Settings{MaxPoints: 100},
		State: InLobbyState,
		Players: []*Player{{Name: nickname, CPU: false, Session: host, AnswerChannel: make(chan Message)}},
		listener: make(chan LobbyMessage),
		lock: &sync.Mutex{},
	}
	host.lobby = lobby

	Lobbies[lobby.ID] = lobby
	lobby.UpdateAll()
	lobby.Run()
}

func (l *Lobby) Join(joiner *Session, nickname string) {
	l.Players = append(l.Players, &Player{Name: nickname, CPU: false, Session: joiner, AnswerChannel: make(chan Message)})
	joiner.lobby = l
	l.UpdateAll()
}

func (l *Lobby) AddCPU(nickname string) {
	l.Players = append(l.Players, &Player{Name: nickname, CPU: true})
	l.UpdateAll()
}

func (l *Lobby) RemoveCPU(nickname string) {
	index := -1
	for i, p := range l.Players {
		if p.Name == nickname && p.CPU {
			index = i
		}
	}
	if index == -1 {
		return
	}

	l.Players = append(l.Players[:index], l.Players[index+1:]...)
	l.UpdateAll()
}


func (l *Lobby) SendMessage(m LobbyMessage) {
	l.lock.Lock()
	l.listener <- m
	l.lock.Unlock()
}

func (l *Lobby) Alive() bool {
	if len(l.Players) == 0 {
		return false
	}
	for _, p := range l.Players {
		if p.Session.Alive() {
			return true
		}
	}
	return false
}

func (l *Lobby) GetPlayer(s *Session) *Player {
	for _, p := range l.Players {
		if s == p.Session {
			return p
		}
	}
	return nil
}


func (l *Lobby) Run() {
	go func() {
		for {
			lm := <-l.listener
			if lm.Dead {
				break
			}

			s := lm.Source
			p := l.GetPlayer(s)
			m := lm.Message
			switch l.State {
			case InLobbyState:
				switch m.Code {
				case RefreshCode:
					l.Update(p)

				case UpdateLobbySettingsCode:
					var pyld struct {
						MaxPoints *int `json:"max_points,omitempty"`
						PSI1 *int `json:"player_swap_index_1,omitempty"`
						PSI2 *int `json:"player_swap_index_2,omitempty"`
						AddCPU *string `json:"add_cpu"`
						RemoveCPU *string `json:"remove_cpu"`
					}
					m.GetContent(&pyld)

					if pyld.MaxPoints != nil {
						l.Settings.MaxPoints = *pyld.MaxPoints
					}
					if pyld.PSI1 != nil {
						l.Players[*pyld.PSI1], l.Players[*pyld.PSI2] = l.Players[*pyld.PSI2], l.Players[*pyld.PSI1]
					}
					if pyld.AddCPU != nil {
						l.AddCPU(*pyld.AddCPU)
					}
					if pyld.RemoveCPU != nil {
						l.RemoveCPU(*pyld.RemoveCPU)
					}

					l.UpdateAll()

				case StartGameCode:
					if len(l.Players) < 4 {
						s.SendInfo("Too few players")
					}
					if len(l.Players) > 4 {
						s.SendInfo("Too many players")
					}
					deciders := []game.Decider{}
					for _, p := range l.Players {
						if !p.CPU {
							deciders = append(deciders, p)
						} else {
							deciders = append(deciders, &game.RandomCPU{ID: p.Name})
						}
					}
					l.Game = *game.NewHeartsGame(deciders, 100)
					l.State = InGameState
					go l.Game.Start() // TODO: make sure this goroutine can stop with the lobby
				default:
					s.SendInvalidCodeError(m.Code)
				}
			case InGameState:
				switch m.Code {
				case RefreshCode:
					l.Update(p)

				case PassedCardsCode:
					p.AnswerChannel <- m

				case PlayedCardCode:
					p.AnswerChannel <- m

				default:
					s.SendInvalidCodeError(m.Code)
				}
			default:
				panic("Shouldn't have gotten here")
			}
		}
	}()
	
	go func() {
		for {
			time.Sleep(30 * time.Second)
			if !l.Alive() {
				l.SendMessage(LobbyMessage{Dead: true})
				break
			}
		}
	}()
}

func (l *Lobby) Update(p *Player) {
	if p.CPU {
		return
	}

	switch l.State {
	case InLobbyState:
		p.Session.SendNewMessage(UpdateLobbyCode, l)
	case InGameState:
		p.Session.SendNewMessage(UpdateCode, l.Game.GetDeciderInfo(p))
	}
}

func (l *Lobby) UpdateAll() {
	for _, s := range l.Players {
		l.Update(s)
	}
}


//----------------------------------------------------//
//--------------------- Player -----------------------//
//----------------------------------------------------//

func (p *Player) Decide(q game.Question, g game.GameState) game.Answer {
	hg := g.GetDeciderInfo(p).(*game.HeartsGameInfo)
	p.Session.SendNewMessage(UpdateCode, hg)


	switch q {
	case game.PassCardsQuestion:
		p.Session.SendMessage(Message{Code: PassCardsCode})
		m := <-p.AnswerChannel
		var payload struct{Cards []int `json:"cards"`}
		m.GetContent(&payload)
		return payload.Cards

	case game.PlayOnTrickQuestion:
		p.Session.SendMessage(Message{Code: PlayCardCode})
		m := <-p.AnswerChannel
		var payload struct{Card int `json:"card"`}
		m.GetContent(&payload)		
		return payload.Card
	}

	return nil
}

func (p *Player) ShowInfo(info string) {
	p.Session.SendInfo(info)
}

func (p *Player) GetName() string {
	return p.Name
}

func (p *Player) Notify(gs game.GameState) {
	p.Session.lobby.SendMessage(LobbyMessage{Dead: false, Source: p.Session, Message: Message{Code: RefreshCode}})
}

func (p *Player) Cleanup(gs game.GameState) {
	p.Notify(gs)
}
