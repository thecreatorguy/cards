package game

import (
	"math/rand"
	"sort"
)

type Player struct {
	Decider Decider
	Hand Deck
	Score int
	roundPoints int
}

type PassDirection string
const (
	PassLeft = PassDirection("left")
	PassRight = PassDirection("right")
	PassAcross = PassDirection("across")
	NoPass = PassDirection("no_pass")
)

const (
	PassCardsQuestion = Question("pass_cards")
	PlayOnTrickQuestion = Question("play_on_trick")
)

type HeartsGame struct {
	Players map[string]*Player
	PlayerOrder []string
	PassDirection PassDirection
	CurrentTrick Deck
	HeartsBroken bool
	MaxPoints int
}

type PlayerInfo struct {
	NumCards int `json:"numCards"`
	Score int `json:"score"`
}

type HeartsGameInfo struct {
	PlayerInfo map[string]PlayerInfo `json:"playerInfo"`
	PlayerOrder []string `json:"playerOrder"`
	PassDirection PassDirection `json:"passDirection"`
	CurrentTrick Deck `json:"currentTrick"`
	HeartsBroken bool `json:"HeartsBroken"`
	MaxPoints int `json:"MaxPoints"`
	Hand Deck `json:"hand"`
}

func NewHeartsGame(deciders []Decider, maxPoints int) *HeartsGame {
	players := map[string]*Player{}
	playerOrder := []string{}
	for _, d := range deciders {
		players[d.GetName()] = &Player{Decider: d}
		playerOrder = append(playerOrder, d.GetName())
	}

	return &HeartsGame{
		Players: players,
		PlayerOrder: playerOrder,
		PassDirection: PassLeft,
		MaxPoints: maxPoints,
	}
}

func NewDefaultHeartsGame(name string) *HeartsGame {
	deciders := []Decider{&RandomCPU{"Alice"}, &RandomCPU{"Bob"}, &RandomCPU{"Charlie"}, &CLIPlayer{name}}
	rand.Shuffle(len(deciders), func(i, j int) { deciders[i], deciders[j] = deciders[j], deciders[i] })
	
	return NewHeartsGame(deciders, 100)
}

func (g *HeartsGame) GetDeciderInfo(decider Decider) interface{} {
	playerInfo := map[string]PlayerInfo{}
	for name, player := range g.Players {
		playerInfo[name] = PlayerInfo{
			NumCards: len(player.Hand),
			Score: player.Score,
		}
	}

	return &HeartsGameInfo{
		PlayerInfo: playerInfo,
		PlayerOrder: g.PlayerOrder,
		PassDirection: g.PassDirection,
		CurrentTrick: g.CurrentTrick,
		HeartsBroken: g.HeartsBroken,
		MaxPoints: g.MaxPoints,
		Hand: g.Players[decider.GetName()].Hand,
	}
}

func (g *HeartsGame) GetPlayer(i int) *Player {
	return g.Players[g.PlayerOrder[i]]
}

func (g *HeartsGame) GetOrder(name string) int {
	for i := 0; i < 4; i++ {
		if g.PlayerOrder[i] == name {
			return i
		}
	}
	return -1
}

func (g *HeartsGame) Loser() *Player {
	for _, p := range g.Players {
		if p.Score >= g.MaxPoints {
			return p
		}
	}
	return nil
}

func (g *HeartsGameInfo) Loser() string {
	for name, p := range g.PlayerInfo {
		if p.Score >= g.MaxPoints {
			return name
		}
	}
	return ""
}

func (g *HeartsGame) Start() {
	for g.Loser() == nil {
		g.PlayRound()
	}
	for _, p := range g.Players {
		p.Decider.Cleanup(g)
	}
}

func (g *HeartsGame) NotifyAll() {
	for _, p := range g.Players {
		p.Decider.Notify(g)
	}
}

func (g *HeartsGame) FirstTrick() bool {
	for _, p := range g.Players {
		if len(p.Hand) == 13 {
			return true
		}
	}
	return false
}

func (g *HeartsGame) PlayRound() {

	// Hand out the next set of cards
	d := NewDeck()
	d.Shuffle()
	pi := 0
	for !d.Empty() {
		g.GetPlayer(pi).Hand = append(g.GetPlayer(pi).Hand, d.Deal())
		pi = (pi + 1) % 4
	}
	for i := 0; i < 4; i++ {
		g.GetPlayer(i).Hand.Sort()
	}

	if g.PassDirection != NoPass {
		// Depending on the pass direction, pass 3 cards
		resultsChannels := []chan Deck{make(chan Deck), make(chan Deck), make(chan Deck), make(chan Deck)}
		for i := 0; i < 4; i++ {
			go func(i int) {
				resultsChannels[i] <- g.GetPlayer(i).PassCards(g)
			}(i)
		}
		passedCards := make([][]Card, 4)
		for i := 0; i < 4; i++ {
			passedCards[i] = <-resultsChannels[i]
		}

		switch (g.PassDirection) {
		case PassLeft:
			g.GetPlayer(0).GetPassedCards(passedCards[3], g)
			g.GetPlayer(1).GetPassedCards(passedCards[0], g)
			g.GetPlayer(2).GetPassedCards(passedCards[1], g)
			g.GetPlayer(3).GetPassedCards(passedCards[2], g)
			g.PassDirection = PassRight
		case PassRight:
			g.GetPlayer(0).GetPassedCards(passedCards[1], g)
			g.GetPlayer(1).GetPassedCards(passedCards[2], g)
			g.GetPlayer(2).GetPassedCards(passedCards[3], g)
			g.GetPlayer(3).GetPassedCards(passedCards[0], g)
			g.PassDirection = PassAcross
		case PassAcross:
			g.GetPlayer(0).GetPassedCards(passedCards[3], g)
			g.GetPlayer(1).GetPassedCards(passedCards[0], g)
			g.GetPlayer(2).GetPassedCards(passedCards[1], g)
			g.GetPlayer(3).GetPassedCards(passedCards[2], g)
			g.PassDirection = NoPass
		}

		for i := 0; i < 4; i++ {
			g.GetPlayer(i).Hand.Sort()
		}
	} else {
		g.PassDirection = PassLeft
	}
	

	// Play starts with the 2 of clubs. Hearts must be broken to play hearts leading. 13 tricks are played
	var leader int
	for i := 0; i < 4; i++ {
		if g.GetPlayer(i).Hand.Contains(Two, Clubs) {
			leader = i
		}
		g.GetPlayer(i).roundPoints = 0
	}
	for trick := 0; trick < 13; trick++ {
		g.CurrentTrick = Deck{}
		var highestTrump int
		var highestValue int
		for i := 0; i < 4; i++ {
			currentPlayer := g.GetPlayer((i + leader) % 4)
			card := currentPlayer.PlayOnTrick(g)
			g.CurrentTrick = append(g.CurrentTrick, card)
			if card.Suit == *g.LeadSuit() && highestValue < card.ValueIndex() {
				highestTrump = i
				highestValue = card.ValueIndex()
			}
			g.NotifyAll()
		}

		leader = (leader + highestTrump) % 4
		g.GetPlayer(leader).roundPoints += PointValue(g.CurrentTrick)
		

		g.NotifyAll()

	}

	// Score the round
	var shotTheMoon *Player
	for _, p := range g.Players {
		if p.roundPoints == 26 {
			shotTheMoon = p
		}
	}

	if shotTheMoon != nil {
		for _, p := range g.Players {
			if p != shotTheMoon {
				p.Score += 26
			}
		}
	} else {
		for _, p := range g.Players {
			p.Score += p.roundPoints
		}
	}

	for i := 0; i < 4; i++ {
		g.GetPlayer(i).roundPoints = 0
	}

	g.NotifyAll()
}

func (p *Player) PassCards(game *HeartsGame) Deck {
	answer := p.Decider.Decide(PassCardsQuestion, game)
	indices, ok := answer.([]int)

	for !ok || len(indices) != 3 {
		p.Decider.ShowInfo("Could not understand answer")

		answer = p.Decider.Decide(PassCardsQuestion, game)
		indices, ok = answer.([]int)
	}
	
	sort.Slice(indices, func(i, j int) bool {return indices[i] < indices[j]})

	cards := Deck{p.Hand[indices[0]], p.Hand[indices[1]], p.Hand[indices[2]]}
	res := append(p.Hand[:indices[0]], p.Hand[indices[0]+1:indices[1]]...)
	res = append(res, p.Hand[indices[1]+1:indices[2]]...)
	res = append(res, p.Hand[indices[2]+1:]...)
	p.Hand = res

	return cards
}

func (p *Player) GetPassedCards(cards []Card, game *HeartsGame) {
	p.Hand = append(p.Hand, cards...)
	p.Decider.Notify(game)
}

func (p *Player) PlayOnTrick(game *HeartsGame) Card {
	for {
		answer := p.Decider.Decide(PlayOnTrickQuestion, game)
		index, ok := answer.(int)

		if !ok {
			p.Decider.ShowInfo("Could not understand answer")
			continue
		}

		card := p.Hand[index]
		if game.LeadSuit() == nil {
			if game.FirstTrick() && (card.Suit != Clubs && card.Value != Two) {
				p.Decider.ShowInfo("Must play 2 of clubs")
				continue
			}
			leadingNotHearts := card.Suit != Hearts
			leadingHearts := card.Suit == Hearts && game.HeartsBroken
			if leadingNotHearts || leadingHearts {
				p.Hand = append(p.Hand[:index], p.Hand[index+1:]...)
				return card
			}
		} else {
			leadSuit := *game.LeadSuit()
			isLeadSuit := leadSuit == card.Suit
			playingCardOutOfSuit := !p.HasSuit(leadSuit)
			firstTrickHearts := game.FirstTrick() && (card.Suit == Hearts || card.Suit == Spades && card.Value == Queen)
			if firstTrickHearts && (p.HasSuit(Clubs) || p.HasSuit(Diamonds) || p.Hand.ContainsNonQueenSpade()) {
				p.Decider.ShowInfo("Cannot play heart on the first trick unless you have no alternative")
				continue
			}
			if isLeadSuit || playingCardOutOfSuit {
				if playingCardOutOfSuit && (card.Suit == Hearts || card.Suit == Spades && card.Value == Queen) {
					game.HeartsBroken = true
				}
				p.Hand = append(p.Hand[:index], p.Hand[index+1:]...)
				return card
			}
		}

		p.Decider.ShowInfo("Invalid Card")
	}
}

func (g *HeartsGame) LeadSuit() *Suit {
	if len(g.CurrentTrick) == 0 {
		return nil
	}
	return &g.CurrentTrick[0].Suit
}

func (p *Player) HasSuit(s Suit) bool {
	for _, c := range p.Hand {
		if c.Suit == s {
			return true
		}
	}
	return false
}

func PointValue(d Deck) int {
	points := 0

	for _, c := range d {
		if c.Suit == Hearts {
			points += 1
		}
		if  c.Suit == Spades && c.Value == Queen {
			points += 13
		}
	}

	return points
}