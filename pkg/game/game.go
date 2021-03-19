package game

import (
	"math/rand"
	"os"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/gorilla/websocket"
)

const (
	BuildingTag = "building"
	SpaceTag 	= "space"
	ScienceTag 	= "science"
	PlantTag 	= "plant"
	MicrobeTag 	= "microbe"
	AnimalTag 	= "animal"
	PowerTag 	= "power"
	JovianTag 	= "jovian"
	EarthTag 	= "earth"
	CityTag 	= "city"
	EventTag 	= "event"
	Wildtag 	= "wild"
)

const (
	LeftPassDirection 	= "left"
	RightPassDirection 	= "right"
)

const (
	AutomatedType 	= "automated"
	ActiveType 		= "active"
	EventType 		= "event"
)

var Games		 	= map[string]*Game{}
var Corporations	= []*Corporation{}
var BaseCards 		= []*Card{}
var CorporateCards 	= []*Card{}
var PreludeCards 	= []*Prelude{}

type GlobalRequirement struct {
	Label string `json:"label"`
	Value int `json:"value"`
	Maximum bool `json:"maximum"`
}

type VictoryPoints struct {
	Value float32 `json:"value"`
	Per string `json:"per"`
}

type Markers struct {
	Count int `json:"count"`
	Type int `json:"type"`
}

type Card struct {
	Name string `json:"name"`
	Cost int `json:"cost"`
	Requirement GlobalRequirement `json:"requirement"`
	Tags []string `json:"tags"`
	Type string `json:"type"`
	Content []interface{} `json:"content"`
	VictoryPoints VictoryPoints `json:"victory_points"`
}

type PlayedCard struct {
	*Card `json:"card"`
	Used bool `json:"used"`
	Markers Markers `json:"markers"`
}

type Corporation struct {
	Name string `json:"name"`
	Tags []string `json:"tags"`
	StartMaterials []string `json:"start_materials"`
}

type Prelude struct {
	Name string `json:"name"`
	Tags []string `json:"tags"`
}

type Player struct {
	ID string `json:"id"`
	Conn *websocket.Conn `json:"-"`
	Hand []*Card `json:"-"`
	PlayedCards []PlayedCard `json:"played_cards"`
}

type Game struct {
	LobbyName string `json:"lobby_name"`
	Started bool `json:"started"`
	Deck []*Card `json:"-"`
	CorporationDeck []*Corporation `json:"-"`
	PreludeDeck  []*Prelude `json:"-"`
	Players map[string]*Player `json:"players"`
	PlayerOrder []string `json:"player_order"`
	Player1 int `json:"player1"`
	CurrentPlayer int `json:"current_player"`
	PassDirection string `json:"pass_direction"`
}

func init() {
	rand.Seed(time.Now().UnixNano())

	file, err := os.Open("./data/base-cards.yaml")
	if err != nil {
		panic(err)
	}
	err = yaml.NewDecoder(file).Decode(&BaseCards)
	if err != nil {
		panic(err)
	}

	file, err = os.Open("./data/corporate-cards.yaml")
	if err != nil {
		panic(err)
	}
	err = yaml.NewDecoder(file).Decode(&CorporateCards)
	if err != nil {
		panic(err)
	}


}

func GetUnstartedGames() []*Game {
	unstarted := []*Game{}
	for _, g := range Games {
		if (!g.Started) {
			unstarted = append(unstarted, g)
		}
	}

	return unstarted
}

func NewGame(conn *websocket.Conn, playerID, lobbyName string) *Game {
	p := &Player{Conn: conn, ID: playerID}

	deck := []*Card{}
	deck = append(deck, BaseCards...)

	rand.Shuffle(len(deck), func(i, j int) {deck[i], deck[j] = deck[j], deck[i]})

	g := &Game{
		LobbyName: lobbyName,
		Started: false,
		Deck: deck,
		Players: map[string]*Player{playerID: p},
		PassDirection: LeftPassDirection,
	}

	go g.Start()

	return g
}

func (g *Game) Done() bool {
	return len(g.Players) == 0
}

func (g *Game) Join(conn *websocket.Conn, id string) bool {
	p := &Player{Conn: conn, ID: id}
	if len(g.Players) < 5 {
		g.Players[id] = p
		return true
	}
	return false
}

func (g *Game) Leave(id string) {
	delete(g.Players, id)
}

func (g *Game) Start() {
	Games[g.LobbyName] = g

	for !g.Done() {
		break
	}

	delete(Games, g.LobbyName)
}

func (g *Game) HandleMessage(m Message, ID string) bool {

	return false
}