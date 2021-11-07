package game

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
	AutomatedType 	= "automated"
	ActiveType 		= "active"
	EventType 		= "event"
)

var Corporations		= []*Corporation{}
var Preludes			= []*Prelude{}
var BaseCards 			= []*Card{}
var CorporateCards 		= []*Card{}
var PreludeCards 		= []*Prelude{}

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
	Content interface{} `json:"content"`
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
	Content interface{} `json:"content"`
}

type PlayerCorporation struct {
	*Corporation `json:"corporation"`
	Used bool `json:"used"`
}

type Prelude struct {
	Name string `json:"name"`
	Tags []string `json:"tags"`
	Content interface{} `json:"content"`
}

func getSelectedCard(cards []*Card, index int) (chosen *Card, notChosen []*Card) {
	chosen = cards[index]
	notChosen = append(append([]*Card{}, cards[:index]...), cards[index+1:]...)
	return
}

func getSelectedCards(cards []*Card, indices []int) (chosen []*Card, notChosen []*Card) {
	for i, c := range cards {
		if intIn(i, indices) {
			chosen = append(chosen, c)
		} else {
			notChosen = append(notChosen, c)
		}
	}
	return
}

func intIn(inval int, set []int) bool {
	for _, v := range set {
		if v == inval {
			return true
		}
	}
	return false
}