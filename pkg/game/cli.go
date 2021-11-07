package game

import (
	"bufio"
	"fmt"
	"os"
)

type CLIPlayer struct {
	Name string
}

func (p *CLIPlayer) Decide(q Question, g GameState) Answer {
	hg := g.GetDeciderInfo(p).(*HeartsGameInfo)
	p.Display(hg)

	reader := bufio.NewReader(os.Stdin)

	switch q {
	case PassCardsQuestion:
		fmt.Printf("Passing %v. Pass 3 cards by index, separated by spaces: ", hg.PassDirection)
		indices := make([]int, 3)
		input, _ := reader.ReadString('\n')
		fmt.Sscanf(input, "%v %v %v", &indices[0], &indices[1], &indices[2])
		return indices 

	case PlayOnTrickQuestion:
		fmt.Println("Play a card by index:")
		var idx int
		input, _ := reader.ReadString('\n')
		fmt.Sscanf(input, "%v", &idx)
		
		return idx

	}

	return nil
}

func (p *CLIPlayer) ShowInfo(info string) {
	println(info)
}

func (p *CLIPlayer) GetName() string {
	return p.Name
}

func (p *CLIPlayer) Notify(g GameState) {
	hg := g.GetDeciderInfo(p).(*HeartsGameInfo)
	p.Display(hg)
}

func (p *CLIPlayer) Cleanup(g GameState) {
	hg := g.GetDeciderInfo(p).(*HeartsGameInfo)
	p.Display(hg)

	fmt.Println("-----------")
	fmt.Println("Game Over!")
	fmt.Printf("Loser: %v\n", hg.Loser())
}


func (p *CLIPlayer) Display(hg *HeartsGameInfo) {
	fmt.Println("-------------")

	fmt.Print("Scores: ")
	for name, player := range hg.PlayerInfo {
		fmt.Printf("( %v: %v ) ", name, player.Score)
	}
	fmt.Println()

	fmt.Printf("Current Trick: %v\n", hg.CurrentTrick)
	
	fmt.Printf("Hand: %v\n", hg.Hand.NumberedString())
}