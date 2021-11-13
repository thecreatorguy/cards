package game

// type Game interface {

// }

type GameState interface {
	GetDeciderInfo(Decider) interface{}
}

type Question string
type Answer interface{}

type Decider interface {
	Decide(Question, GameState) Answer
	ShowInfo(string)
	GetName() string
	Notify(GameState)
}
