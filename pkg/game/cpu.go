package game

import (
	"math/rand"
)

type RandomCPU struct {
	ID string
}

func RandomDecision(d Decider, q Question, g GameState) Answer {
	hg := g.GetDeciderInfo(d).(*HeartsGameInfo)

	switch q {
	case PassCardsQuestion:
		nums := map[int]interface{}{}
		for len(nums) < 3 {
			nums[rand.Int() % 13] = 0
		}
		numsSlice := []int{}
		for num := range nums {
			numsSlice = append(numsSlice, num) 
		}
		return numsSlice

	case PlayOnTrickQuestion:
		if len(hg.Hand) == 13 && hg.Hand.Contains(Two, Clubs) {
			return 0
		}
		return rand.Int() % len(hg.Hand)
	}

	return nil
}

func (cpu *RandomCPU) Decide(q Question, g GameState) Answer {
	return RandomDecision(cpu, q, g)
}

func (cpu *RandomCPU) ShowInfo(string) {}

func (cpu *RandomCPU) GetName() string {
	return cpu.ID
}

func (cpu *RandomCPU) Notify(GameState) {}
