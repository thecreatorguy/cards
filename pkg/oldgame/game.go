package game

import (
	_ "embed"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/thecreatorguy/cards/pkg/jsonyaml"
)

//-------------------------------------------------------------------
//---------------------   Types and Constants   ---------------------
//-------------------------------------------------------------------

const (
	LeftPassDirection 	= "left"
	RightPassDirection 	= "right"
)

type Player struct {
	*Session `json:"-"`
	Nickname string `json:"nickname"`
	Corporation PlayerCorporation `json:"corporation"`
	Hand []*Card `json:"-"`
	PossibleCards []*Card `json:"-"`
	PlayedCards []PlayedCard `json:"played_cards"`
	PlayedPreludes []*Prelude `json:"played_preludes"`
	Passed bool `json:"passed"`
}

type Settings struct {
	UseCorporateCards bool `json:"use_corporate_cards"`
	UsePreludeCards bool `json:"use_prelude_cards"`
}

type GameState string
const (
	InLobbyState = GameState("in_lobby")
	InGameState = GameState("in_game")
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
	lock *sync.Mutex `json:"-"`
}

var Games = map[string]*Game{}
var SessionToGame = map[string]*Game{}

//-------------------------------------------------------------------
//--------------------------   Init Cards   -------------------------
//-------------------------------------------------------------------

//go:embed data/corporations.yaml
var corporationsFile []byte
//go:embed data/preludes.yaml
var preludesFile []byte
//go:embed data/base-cards.yaml
var baseCardsFile []byte
//go:embed data/corporate-cards.yaml
var corporateCardsFile []byte
//go:embed data/prelude-cards.yaml
var preludeCardsFile []byte

func init() {
	rand.Seed(time.Now().UnixNano())

	err := jsonyaml.Unmarshal(corporationsFile, &Corporations)
	if err != nil {
		panic(err)
	}
	err = jsonyaml.Unmarshal(preludesFile, &Preludes)
	if err != nil {
		panic(err)
	}
	err = jsonyaml.Unmarshal(baseCardsFile, &BaseCards)
	if err != nil {
		panic(err)
	}
	err = jsonyaml.Unmarshal(corporateCardsFile, &CorporateCards)
	if err != nil {
		panic(err)
	}
	err = jsonyaml.Unmarshal(preludeCardsFile, &PreludeCards)
	if err != nil {
		panic(err)
	}
}

//-------------------------------------------------------------------
//----------------------------   In Lobby   -------------------------
//-------------------------------------------------------------------

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
		lock: &sync.Mutex{},
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

func (g *Game) Run() {
	Games[g.LobbyName] = g
	for _, p := range g.Players {
		SessionToGame[p.ID] = g
	}
	g.UpdatePlayers()

	runloop:
	for !g.Done() {
		switch g.State {
		case InLobbyState:
			m, ok := g.GetMessage(g.Host)
			if !ok {
				break runloop
			}
			switch m.Code {
			case UpdateLobbySettingsCode:
				g.UpdateLobbySettings(m)
			case StartGameCode:
				g.Start()
			default:
				g.Host.SendInvalidCode()
			}
		case InGameState:
			done := g.PlayTurn()
			if done {
				g.NextGeneration()
			}
		}
	}
	
	for _, p := range g.Players {
		p.Cleanup()
		delete(SessionToGame, p.ID)
	}
	delete(Games, g.LobbyName)
}

func (g *Game) UpdateLobbySettings(m Message) {
	var pyld struct {
		UseCorporateCards *bool `json:"use_corporate_cards,omitempty"`
		UsePreludeCards *bool `json:"use_prelude_cards,omitempty"`
		PlayerSwapIndex1 *int `json:"player_swap_index_1,omitempty"`
		PlayerSwapIndex2 *int `json:"player_swap_index_2,omitempty"`
	}
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

//-------------------------------------------------------------------
//----------------------------   In Game   --------------------------
//-------------------------------------------------------------------

func (g *Game) Start() { 
	g.State = InGameState

	// Initialize the game state, shuffle cards
	corporations := append([]*Corporation{}, Corporations...)
	rand.Shuffle(len(corporations), func(i, j int) {corporations[i], corporations[j] = corporations[j], corporations[i]})
	g.CorporationDeck = corporations

	deck := append([]*Card{}, BaseCards...)
	if g.Settings.UseCorporateCards {
		deck = append(deck, CorporateCards...)
	}
	rand.Shuffle(len(deck), func(i, j int) {deck[i], deck[j] = deck[j], deck[i]})
	g.Deck = deck

	if g.Settings.UsePreludeCards {
		preludes := append([]*Prelude{}, Preludes...)
		g.PreludeDeck = preludes
	}

	g.DiscardPile = []*Card{}

	g.Player1 = rand.Intn(len(g.Players))
	g.CurrentPlayer = g.Player1
	g.PassDirection = LeftPassDirection

	// Handle first generation
	var wg sync.WaitGroup
	wg.Add(len(g.Players))
	for _, p := range g.Players {
		go func(p *Player) {
			deal := struct {
				Corporations []*Corporation `json:"corporations"`
				Cards []*Card `json:"cards"`
				Preludes []*Prelude `json:"preludes"`
			}{
				Corporations: g.DrawCorporations(3),
				Cards: g.DrawCards(10),
				Preludes: g.DrawPreludes(4),
			}
			
			closed, err := p.SendMessage(Message{FirstGenCode, deal})
			if closed {
				log.Println("closed")
			}
			if err != nil {
				log.Println(err)
			}

			m, ok := g.GetMessage(p)
			if !ok {
				return
			}

			var resp struct {
				Corporation int `json:"corporation"`
				Cards []int `json:"cards"`
				Preludes []int `json:"preludes"`
			}
			m.GetContent(&resp)

			p.Corporation = PlayerCorporation{deal.Corporations[resp.Corporation], false}

			hand, discarded := getSelectedCards(deal.Cards, resp.Cards)
			p.Hand = hand
			g.Discard(discarded...)

			preludes := []*Prelude{}
			for _, pr := range resp.Preludes {
				preludes = append(preludes, deal.Preludes[pr])
			}
			p.PlayedPreludes = preludes

			wg.Done()
		}(p)
	}
	wg.Wait()

	g.UpdatePlayers()
}

func (g *Game) PlayTurn() bool {
	p := g.Players[g.CurrentPlayer]

	p.SendMessage(Message{Code: YourTurnCode})
	// loop to process until message to pass or done
	receiveLoop:
	for {
		m, ok := g.GetMessage(p)
		if !ok {
			return false
		}
		switch m.Code {
		case PlayCardCode:
			var pyld struct{Card int `json:"card"`}
			m.GetContent(&pyld)
			var selected *Card
			selected, p.Hand = getSelectedCard(p.Hand, pyld.Card)
			p.PlayedCards = append(p.PlayedCards, PlayedCard{Card: selected})

		case DrawCardsCode:
			var pyld struct{Amount int `json:"amount"`}
			m.GetContent(&pyld)
			deal := g.DrawCards(pyld.Amount)
			p.SendMessage(Message{DrawCardsCode, struct{Cards []*Card `json:"cards"`}{deal}})

			m, ok = g.GetMessage(p)
			if !ok {
				return false
			}
			var chosenPyld struct{Cards []int `json:"cards"`}
			m.GetContent(&chosenPyld)
			chosen, discarded := getSelectedCards(p.Hand, chosenPyld.Cards)
			p.Hand = append(p.Hand, chosen...)
			g.Discard(discarded...)

		case DiscardCardsCode:
			var pyld struct{Cards []int `json:"cards"`}
			m.GetContent(&pyld)
			var selected []*Card
			selected, p.Hand = getSelectedCards(p.Hand, pyld.Cards)
			g.Discard(selected...)

		case DrawPreludesCode:
			var pyld struct{Amount int `json:"amount"`}
			m.GetContent(&pyld)
			deal := g.DrawPreludes(pyld.Amount)
			p.SendMessage(Message{DrawPreludesCode, struct{Preludes []*Prelude `json:"prelude"`}{deal}})

			m, ok = g.GetMessage(p)
			if !ok {
				return false
			}
			var chosenPyld struct{Preludes []int `json:"preludes"`}
			m.GetContent(&chosenPyld)
			for _, pr := range chosenPyld.Preludes {
				p.PlayedPreludes = append(p.PlayedPreludes, deal[pr])
			}

		case DoneTurnCode:
			break receiveLoop

		case PassCode:
			p.Passed = true
			break receiveLoop
		}
		g.UpdatePlayers()
	}

	// If all players are passed, the round is done
	done := true
	for _, pl := range g.Players {
		if !pl.Passed {
			done = false
			break
		}
	}
	if done {
		return true
	}

	for {
		g.CurrentPlayer = (g.CurrentPlayer + 1) % len(g.Players)
		if !g.Players[g.CurrentPlayer].Passed {
			break
		}
	}
	return false
}

func (g *Game) NextGeneration() {
	//draw cards and pass
	cardSets := [][]*Card{}
	for i := 0; i < len(g.Players); i++ {
		cardSets = append(cardSets, g.DrawCards(4))
	}

	for _, p := range g.Players {
		p.PossibleCards = []*Card{}
	}

	for i := 0; i < 4; i++ {
		var wg sync.WaitGroup
		wg.Add(len(g.Players))
		for j, p := range g.Players {
			var cards []*Card
			if g.PassDirection == LeftPassDirection { // Left, or clockwise, is defined as passing in the positive direction
				cards = cardSets[(i+j) % len(g.Players)]
			} else {
				cards = cardSets[(i-j) % len(g.Players)]
			}
			
			go func(cards *[]*Card, p *Player) {
				p.SendMessage(Message{BetweenGensCode, *cards})
				m, ok := g.GetMessage(p)
				if !ok {
					return
				}
				if (i < 4) {
					var choice struct{Card int `json:"card"`}
					m.GetContent(choice)
					var chosen *Card
					chosen, *cards = getSelectedCard(*cards, choice.Card)
					p.PossibleCards = append(p.PossibleCards, chosen)
				} else {
					var choice struct{Cards []int `json:"cards"`}
					m.GetContent(choice)
					selected, discarded := getSelectedCards(p.PossibleCards, choice.Cards)
					p.Hand = append(p.Hand, selected...)
					g.Discard(discarded...)
				}
				
				wg.Done()
			}(&cards, p)
		}
		wg.Wait()
		if g.Done() {
			return
		}
	}

	g.Player1 = (g.Player1 + 1) % len(g.Players)
	g.CurrentPlayer = g.Player1
	if g.PassDirection == LeftPassDirection {
		g.PassDirection = RightPassDirection
	} else {
		g.PassDirection = LeftPassDirection
	}
	
	g.UpdatePlayers()
}

//-------------------------------------------------------------------
//----------------------------   Helpers   --------------------------
//-------------------------------------------------------------------

func GetUnstartedGames() []*Game {
	unstarted := []*Game{}
	for _, g := range Games {
		if (!g.Started()) {
			unstarted = append(unstarted, g)
		}
	}

	return unstarted
}

func (g *Game) DrawCorporations(num int) []*Corporation {
	g.lock.Lock()
	corporations := g.CorporationDeck[:num]
	g.CorporationDeck = g.CorporationDeck[num:]
	g.lock.Unlock()
	return corporations
}

func (g *Game) DrawPreludes(num int) []*Prelude {
	g.lock.Lock()
	preludes := g.PreludeDeck[:num]
	g.PreludeDeck = g.PreludeDeck[num:]
	g.lock.Unlock()
	return preludes
}

func (g *Game) DrawCards(num int) []*Card {
	g.lock.Lock()
	if num > len(g.Deck) {
		rand.Shuffle(len(g.DiscardPile), func(i, j int) {g.DiscardPile[i], g.DiscardPile[j] = g.DiscardPile[j], g.DiscardPile[i]})
		g.Deck = append(g.Deck, g.DiscardPile...)
		g.DiscardPile = []*Card{}
	}
	cards := g.Deck[:num]
	g.Deck = g.Deck[num:]
	g.lock.Unlock()
	return cards
}

func (g *Game) Discard(discarded ...*Card) {
	g.lock.Lock()
	g.DiscardPile = append(g.DiscardPile, discarded...)
	g.lock.Unlock()
}

func (g *Game) Update(p *Player) {
	p.SendMessage(Message{UpdateCode, struct {Game *Game `json:"game"`;	Hand []*Card `json:"hand"`}{g, p.Hand}})
}

func (g *Game) UpdatePlayers() {
	for _, p := range g.Players {
		g.Update(p)
	}
}

func (g *Game) GetMessage(p *Player) (Message, bool) {
	m, closed, _ := p.ReceiveMessage()

	if closed {
		if g.Done() || !p.WaitForReconnect() {
			g.WakeThreads()
			return m, false
		}
		g.Update(p)
		return g.GetMessage(p)
	}

	return m, true
}

func (g *Game) NumDisconnected() int {
	disconnected := 0
	for _, p := range g.Players {
		if !p.Alive() {
			disconnected += 1
		}
	}
	return disconnected
}

func (g *Game) Done() bool {
	return g.NumDisconnected() == len(g.Players)
}

func (g *Game) WakeThreads() {
	for _, p := range g.Players {
		p.Wake()
	}
}
