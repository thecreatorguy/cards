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
	Leader string
	cancelListeners map[int]chan bool
	nextListenerID int
	Cancelled bool
}

type PlayerInfo struct {
	NumCards int `json:"numCards"`
	Score int `json:"score"`
	RoundPoints int `json:"roundPoints"`
	Lead bool `json:"lead"`
}

type HeartsGameInfo struct {
	Name string `json:"name"`
	PlayerInfo map[string]PlayerInfo `json:"playerInfo"`
	PlayerOrder []string `json:"playerOrder"`
	PassDirection PassDirection `json:"passDirection"`
	CurrentTrick Deck `json:"currentTrick"`
	HeartsBroken bool `json:"heartsBroken"`
	MaxPoints int `json:"maxPoints"`
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
		cancelListeners: map[int]chan bool{},
		Cancelled: false,
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
			RoundPoints: player.roundPoints,
			Lead: name == g.Leader,
		}
	}

	return &HeartsGameInfo{
		Name: decider.GetName(),
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

func (g *HeartsGame) GameOver() bool {
	return g.Cancelled || g.Loser() != nil
}

func (g *HeartsGameInfo) Loser() string {
	for name, p := range g.PlayerInfo {
		if p.Score >= g.MaxPoints {
			return name
		}
	}
	return ""
}

func (g *HeartsGame) Start() chan bool {
	completeChannel := make(chan bool)
	go func() {
		for !g.GameOver() {
			if g.PlayRound() {
				break
			}
		}
		for _, p := range g.Players {
			p.Decider.Notify(g)
		}
		completeChannel <- true
	}()
	
	return completeChannel
}

func (g *HeartsGame) Cancel() {
	g.Cancelled = true
	for _, c := range g.cancelListeners {
		c <- true
	}
}

func (g *HeartsGame) GetCancelListener() (int, chan bool) {
	ret := make(chan bool)
	g.cancelListeners[g.nextListenerID] = ret
	g.nextListenerID++
	return g.nextListenerID-1, ret
}

func (g *HeartsGame) RemoveCancelListener(id int) {
	delete(g.cancelListeners, id)
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

func (g *HeartsGame) PlayRound() bool {
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

	if cancelled := g.PassCards(); cancelled {
		return true
	}
	

	// Play starts with the 2 of clubs. Hearts must be broken to play hearts leading. 13 tricks are played
	var leader int
	for i := 0; i < 4; i++ {
		if g.GetPlayer(i).Hand.Contains(Two, Clubs) {
			leader = i
		}
	}
	var cancelled bool
	for trick := 0; trick < 13; trick++ {
		leader, cancelled = g.PlayTrick(leader)
		if cancelled {
			return true
		}
	}
	g.Leader = ""

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

	return false
}

func (g *HeartsGame) PassCards() bool {
	if g.PassDirection != NoPass {
		// Depending on the pass direction, pass 3 cards
		resultsChannels := []chan *Deck{make(chan *Deck), make(chan *Deck), make(chan *Deck), make(chan *Deck)}
		for i := 0; i < 4; i++ {
			go func(i int) {
				cards, cancelled := g.GetPlayer(i).PassCards(g)
				if cancelled {
					resultsChannels[i] <- nil
				}
				resultsChannels[i] <- &cards
			}(i)
		}
		passedCards := make([]Deck, 4)
		for i := 0; i < 4; i++ {
			res := <-resultsChannels[i]
			if res == nil {
				return true
			}
			passedCards[i] = *res 
		}

		switch (g.PassDirection) {
		case PassLeft:
			for i := 0; i < 4; i++ {
				g.GetPlayer(i).GetPassedCards(passedCards[(i+3)%4], g)
			}
			g.PassDirection = PassRight
		case PassRight:
			for i := 0; i < 4; i++ {
				g.GetPlayer(i).GetPassedCards(passedCards[(i+1)%4], g)
			}
			g.PassDirection = PassAcross
		case PassAcross:
			for i := 0; i < 4; i++ {
				g.GetPlayer(i).GetPassedCards(passedCards[(i+2)%4], g)
			}
			g.PassDirection = NoPass
		}

		for i := 0; i < 4; i++ {
			g.GetPlayer(i).Hand.Sort()
		}
	} else {
		g.PassDirection = PassLeft
	}
	g.NotifyAll()

	return false
}

func (g *HeartsGame) PlayTrick(leader int) (int, bool) {
	g.Leader = g.GetPlayer(leader).Decider.GetName()

	var highestTrump int
	var highestValue int
	for i := 0; i < 4; i++ {
		currentPlayer := g.GetPlayer((i + leader) % 4)
		card, cancelled := currentPlayer.PlayOnTrick(g)
		if cancelled {
			return 0, true
		}

		g.CurrentTrick = append(g.CurrentTrick, card)
		if card.Suit == *g.LeadSuit() && highestValue < card.ValueIndex() {
			highestTrump = i
			highestValue = card.ValueIndex()
		}
		g.NotifyAll()
	}

	leader = (leader + highestTrump) % 4
	
	g.GetPlayer(leader).roundPoints += PointValue(g.CurrentTrick)
	
	g.CurrentTrick = Deck{}
	g.NotifyAll()

	return leader, false
}

func (p *Player) GetAnswer(q Question, game *HeartsGame) (Answer, bool) {
	ansChan := make(chan Answer)
	go func() {
		ansChan <- p.Decider.Decide(q, game)
	}()
	
	var answer Answer
	cancelID, cancelChan := game.GetCancelListener()
	select {
	case answer = <-ansChan:
		game.RemoveCancelListener(cancelID)
		return answer, false
	case <-cancelChan:
		return nil, true
	}
}

func (p *Player) PassCards(game *HeartsGame) (Deck, bool) {
	for {
		answer, cancelled := p.GetAnswer(PassCardsQuestion, game)
		if cancelled {
			return Deck{}, true
		}

		indices, ok := answer.([]int)
		for !ok || len(indices) != 3 {
			p.Decider.ShowInfo("Could not understand answer")
			continue
		}
		
		sort.Slice(indices, func(i, j int) bool {return indices[i] < indices[j]})

		cards := Deck{p.Hand[indices[0]], p.Hand[indices[1]], p.Hand[indices[2]]}
		res := append(p.Hand[:indices[0]], p.Hand[indices[0]+1:indices[1]]...)
		res = append(res, p.Hand[indices[1]+1:indices[2]]...)
		res = append(res, p.Hand[indices[2]+1:]...)
		p.Hand = res

		return cards, false
	}
}

func (p *Player) GetPassedCards(cards []Card, game *HeartsGame) {
	p.Hand = append(p.Hand, cards...)
}

func (p *Player) PlayOnTrick(game *HeartsGame) (Card, bool) {
	for {
		answer, cancelled := p.GetAnswer(PlayOnTrickQuestion, game)
		if cancelled {
			return Card{}, true
		}

		index, ok := answer.(int)
		if !ok {
			p.Decider.ShowInfo("Could not understand answer")
			continue
		}

		card := p.Hand[index]
		if game.LeadSuit() == nil {
			if game.FirstTrick() && (card.Suit != Clubs || card.Value != Two) {
				p.Decider.ShowInfo("Must lead with the 2 of clubs")
				continue
			}
			if card.Suit == Hearts && !game.HeartsBroken {
				p.Decider.ShowInfo("Hearts not broken, lead with another suit")
				continue
			}

			p.Hand = append(p.Hand[:index], p.Hand[index+1:]...)
			return card, false
		} else {
			leadSuit := *game.LeadSuit()
			isLeadSuit := leadSuit == card.Suit
			if !isLeadSuit && p.HasSuit(leadSuit) {
				p.Decider.ShowInfo("Must play the lead suit: " + string(leadSuit))
				continue
			}

			playingHeart := card.Suit == Hearts || card.Suit == Spades && card.Value == Queen
			hasNonHeartCard := p.HasSuit(Clubs) || p.HasSuit(Diamonds) || p.Hand.ContainsNonQueenSpade()
			if game.FirstTrick() && playingHeart && hasNonHeartCard {
				p.Decider.ShowInfo("Cannot play heart or QoS on the first trick unless you have no alternative")
				continue
			}
			
			if playingHeart {
				game.HeartsBroken = true
			}
			p.Hand = append(p.Hand[:index], p.Hand[index+1:]...)
			return card, false
		}
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