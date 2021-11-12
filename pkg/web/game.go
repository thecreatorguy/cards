package web

import (
	"sync"
	"time"

	"github.com/thecreatorguy/cards/pkg/game"
)

const (
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

type LobbyMessage struct {
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

type Player struct {
	Name string `json:"name"`
	CPU bool `json:"cpu"`
	Session *Session `json:"-"`
	AnswerChannel chan Message `json:"-"`
}

var Lobbies = map[string]*Lobby{}

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


func (l *Lobby) SendMessage(s *Session, m Message) {
	l.lock.Lock()
	l.listener <- LobbyMessage{s, m}
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


func (l *Lobby) Run() {
	go func() {
		for {
			lm, ok := <-l.listener
			if !ok {
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
						break
					}
					if len(l.Players) > 4 {
						s.SendInfo("Too many players")
						break
					}
					deciders := []game.Decider{}
					for _, p := range l.Players {
						if !p.CPU {
							deciders = append(deciders, p)
						} else {
							deciders = append(deciders, &game.RandomCPU{ID: p.Name})
						}
					}
					l.Game = *game.NewHeartsGame(deciders, l.Settings.MaxPoints)
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
				close(l.listener)
				break
			}
		}
	}()
}

func (l *Lobby) GetPlayer(s *Session) *Player {
	for _, p := range l.Players {
		if s == p.Session {
			return p
		}
	}
	return nil
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
	p.Session.lobby.SendMessage(p.Session, Message{Code: RefreshCode})
}

func (p *Player) Cleanup(gs game.GameState) {
	p.Notify(gs)
}
