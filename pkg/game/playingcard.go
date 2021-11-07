package game

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
)

type Suit string
const (
	Hearts   = 	Suit("hearts")
	Diamonds = 	Suit("diamonds")
	Clubs    = 	Suit("clubs")
	Spades   = 	Suit("spades")
)
var Suits = []Suit{Clubs, Diamonds, Spades, Hearts}

type CardValue string
const (
	Ace = CardValue("ace")
	Two = CardValue("2")
	Three = CardValue("3")
	Four = CardValue("4")
	Five = CardValue("5")
	Six = CardValue("6")
	Seven = CardValue("7")
	Eight = CardValue("8")
	Nine = CardValue("9")
	Ten = CardValue("10")
	Jack = CardValue("jack")
	Queen = CardValue("queen")
	King = CardValue("king")
)
var CardValues = []CardValue{Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King, Ace}

type Card struct {
	Suit Suit
	Value CardValue
}

type Deck []Card

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewDeck() Deck {
	cards := []Card{}
	for _, s := range Suits {
		for _, v := range CardValues {
			cards = append(cards, Card{s, v})
		}
	}
	return cards
}

func (d Deck) Shuffle() {
	rand.Shuffle(len(d), func(i, j int) { d[i], d[j] = d[j], d[i] })
}

func (d *Deck) Deal() Card {
	c := (*d)[0]
	*d = (*d)[1:]
	return c
}

func (d Deck) Empty() bool {
	return len(d) == 0
}

func (d Deck) Contains(v CardValue, s Suit) bool {
	for _, c := range d {
		if c.Suit == s && c.Value == v {
			return true
		}
	}
	return false
}

func (d Deck) Sort() {
	sort.Slice(d, func(i, j int) bool {
		if d[i].SuitIndex() != d[j].SuitIndex() {
			return d[i].SuitIndex() < d[j].SuitIndex()
		}
		return d[i].ValueIndex() < d[j].ValueIndex()
	})
}

func (c Card) SuitIndex() int {
	for i, s := range Suits {
		if s == c.Suit {
			return i
		}
	}
	return -1
}

func (c Card) ValueIndex() int {
	for i, v := range CardValues {
		if v == c.Value {
			return i
		}
	}
	return -1
}

func (d *Deck) Play(c Card) {
	*d = append(*d, c)
}

func (c Card) String() string {
	return fmt.Sprintf("[%s %s]", c.Suit, c.Value)
}

func (d Deck) String() string {
	ret := []string{}
	for _, c := range d {
		ret = append(ret, fmt.Sprintf("%v", c.String()))
	}
	return "{" + strings.Join(ret, ", ") + "}"
}

func (d Deck) NumberedString() string {
	ret := []string{}
	for num, c := range d {
		ret = append(ret, fmt.Sprintf("%v:%v", num, c.String()))
	}
	return "{" + strings.Join(ret, ", ") + "}"
}

func (d Deck) ContainsNonQueenSpade() bool {
	for _, c := range d {
		if c.Suit == Spades && c.Value != Queen {
			return true
		}
	}
	return false
}