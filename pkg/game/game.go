package game

import (
	_ "embed"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/go-yaml/yaml"
)

const (
	LeftPassDirection 	= "left"
	RightPassDirection 	= "right"
)

type Player struct {
	*Session `json:"-"`
	Nickname string `json:"nickname"`
	Corporation Corporation `json:"corporation"`
	Hand []*Card `json:"-"`
	PossibleCards []*Card `json:"-"`
	PlayedCards []PlayedCard `json:"played_cards"`
	PlayedPreludes []Prelude `json:"played_preludes"`
}

type Settings struct {
	UseCorporateCards bool `json:"use_corporate_cards"`
	UsePreludeCards bool `json:"use_prelude_cards"`
}

type GameState string
const (
	InLobbyState = GameState("in_lobby")
	FirstGenState = GameState("first_gen")
	InGenState = GameState("in_gen")
	BetweenGensState = GameState("between_gens")
)

type Game struct {
	LobbyName string `json:"lobby_name"`
	Settings Settings `json:"settings"`
	State GameState `json:"state"`
	Deck []*Card `json:"-"`
	DiscardPile []*Card `json:"-"`
	CorporationDeck []*Corporation `json:"-"`
	PreludeDeck  []*Prelude `json:"-"`
	Players []*Player `json:"players"`
	Host *Player `json:"-"`
	Player1 int `json:"player1"`
	CurrentPlayer int `json:"current_player"`
	PassDirection string `json:"pass_direction"`
	Lock *sync.Mutex `json:"-"`
}

var Games = map[string]*Game{}
var SessionToGame = map[string]*Game{}

//go:embed data/corporation-cards.yaml
var corporationCardsFile string
//go:embed data/base-cards.yaml
var baseCardsFile string
//go:embed data/corporate-cards.yaml
var corporateCardsFile string
//go:embed data/prelude-cards.yaml
var preludeCardsFile string

func init() {
	rand.Seed(time.Now().UnixNano())

	err := yaml.NewDecoder(strings.NewReader(corporationCardsFile)).Decode(&CorporationCards)
	if err != nil {
		panic(err)
	}
	err = yaml.NewDecoder(strings.NewReader(baseCardsFile)).Decode(&BaseCards)
	if err != nil {
		panic(err)
	}
	err = yaml.NewDecoder(strings.NewReader(corporateCardsFile)).Decode(&CorporateCards)
	if err != nil {
		panic(err)
	}
	err = yaml.NewDecoder(strings.NewReader(preludeCardsFile)).Decode(&PreludeCards)
	if err != nil {
		panic(err)
	}
}

func GetUnstartedGames() []*Game {
	unstarted := []*Game{}
	for _, g := range Games {
		if (!g.Started()) {
			unstarted = append(unstarted, g)
		}
	}

	return unstarted
}

func NewGame(host *Session, lobbyName, nickname string) {
	p := &Player{Session: host, Nickname: nickname}

	g := &Game{
		LobbyName: lobbyName,
		State: InLobbyState,
		Settings: Settings{
			UseCorporateCards: true,
			UsePreludeCards: true,
		},
		Players: []*Player{p},
		Host: p,
		Lock: &sync.Mutex{},
	}

	go g.Run()
}

func (g *Game) Started() bool {
	return g.State != InLobbyState
}

func (g *Game) Join(s *Session, nickname string) bool {
	p := &Player{Session: s, Nickname: nickname}
	if len(g.Players) >= 5 {
		return false
	}

	g.Players = append(g.Players, p)
	SessionToGame[p.Session.ID] = g
	g.UpdatePlayers()
	return true	
}

type UpdateLobbySettingsPayload struct {
	UseCorporateCards *bool `json:"use_corporate_cards,omitempty"`
	UsePreludeCards *bool `json:"use_prelude_cards,omitempty"`
	PlayerSwapIndex1 *int `json:"player_swap_index_1,omitempty"`
	PlayerSwapIndex2 *int `json:"player_swap_index_2,omitempty"`
}

func (g *Game) Run() {
	Games[g.LobbyName] = g
	for _, p := range g.Players {
		SessionToGame[p.ID] = g
	}
	g.UpdatePlayers()

	for !g.Done() {
		switch g.State {
		case InLobbyState:
			m, _, _ := g.Host.ReceiveMessage() // TODO: handle host dropping out
			switch m.Code {
			case UpdateLobbySettingsCode:
				g.UpdateLobbySettings(m)
			case StartGameCode:
				g.Start()
			default:
				g.Host.SendInvalidCode()
			}
		case FirstGenState:
		case BetweenGensState:
		case InGenState:
		}
	}

	for _, p := range g.Players {
		delete(SessionToGame, p.Session.ID)
	}
	delete(Games, g.LobbyName)
}

func (g *Game) UpdateLobbySettings(m Message) {
	var pyld UpdateLobbySettingsPayload
	m.GetContent(&pyld)
	if pyld.UseCorporateCards != nil {
		g.Settings.UseCorporateCards = *pyld.UseCorporateCards
	}
	if pyld.UsePreludeCards != nil {
		g.Settings.UsePreludeCards = *pyld.UsePreludeCards
	}
	if pyld.PlayerSwapIndex1 != nil {
		temp := g.Players[*pyld.PlayerSwapIndex1]
		g.Players[*pyld.PlayerSwapIndex1] = g.Players[*pyld.PlayerSwapIndex2]
		g.Players[*pyld.PlayerSwapIndex2] = temp
	}
	g.UpdatePlayers()
}

type FirstGenPayload struct {
	Corporations []*Corporation `json:"corporations"`
	Cards []*Card `json:"cards"`
	Preludes []*Prelude `json:"preludes"`
}

func (g *Game) Start() { 
	// Initialize the game state, shuffle cards
	corporations := append([]*Corporation{}, CorporationCards...)
	rand.Shuffle(len(corporations), func(i, j int) {corporations[i], corporations[j] = corporations[j], corporations[i]})
	g.CorporationDeck = corporations

	deck := append([]*Card{}, BaseCards...)
	if g.Settings.UseCorporateCards {
		deck = append(deck, CorporateCards...)
	}
	rand.Shuffle(len(deck), func(i, j int) {deck[i], deck[j] = deck[j], deck[i]})
	g.Deck = deck

	if g.Settings.UsePreludeCards {
		preludes := append([]*Prelude{}, PreludeCards...)
		g.PreludeDeck = preludes
	}

	g.DiscardPile = []*Card{}

	g.Player1 = rand.Intn(len(g.Players))
	g.CurrentPlayer = g.Player1
	g.PassDirection = LeftPassDirection
	g.State = FirstGenState

	// Handle first generation
	var wg sync.WaitGroup
	wg.Add(len(g.Players))
	for _, p := range g.Players {
		go func(p *Player) {
			payload := FirstGenPayload{
				Corporations: g.DealCorporations(3),
				Cards: g.DealCards(10),
				Preludes: g.DealPreludes(4),
			}
			p.SendMessage(Message{FirstGenCode, payload})

			p.ReceiveMessage()
			// TODO parse out what was passed back
			wg.Done()
		}(p)
	}
	wg.Wait()

	g.UpdatePlayers()
}

func (g *Game) DealCorporations(num int) []*Corporation {
	g.Lock.Lock()
	corporations := g.CorporationDeck[:num]
	g.CorporationDeck = g.CorporationDeck[num:]
	g.Lock.Unlock()
	return corporations
}

func (g *Game) DealCards(num int) []*Card {
	g.Lock.Lock()
	cards := g.Deck[:num]
	g.Deck = g.Deck[num:]
	g.Lock.Unlock()
	return cards
}

func (g *Game) DealPreludes(num int) []*Prelude {
	g.Lock.Lock()
	preludes := g.PreludeDeck[:num]
	g.PreludeDeck = g.PreludeDeck[num:]
	g.Lock.Unlock()
	return preludes
}

func (g *Game) Discard(discarded ...*Card) {
	g.Lock.Lock()
	g.DiscardPile = append(g.DiscardPile, discarded...)
	g.Lock.Unlock()
}

func (g *Game) Done() bool {
	noneAlive := true
	for _, p := range g.Players {
		if p.Alive() {
			noneAlive = false
			break
		}
	}
	return len(g.Players) == 0 || noneAlive
}

func (g *Game) SendToAll(m Message) {
	for _, p := range g.Players {
		p.SendMessage(m)
	}
}

// func (g *Game) SendToAllExcept(m Message, except *Player) {
// 	for _, p := range g.Players {
// 		if p != except {
// 			p.SendMessage(m)
// 		}
// 	}
// }

type UpdatePayload struct {
	Game *Game `json:"game"`
	Hand []*Card `json:"hand"`
}

func (g *Game) UpdatePlayers() {
	for _, p := range g.Players {
		p.SendMessage(Message{UpdateCode, UpdatePayload{g, p.Hand}})
	}
}