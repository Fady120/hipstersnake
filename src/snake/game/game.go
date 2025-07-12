package game

import (
	"log"
	"math/rand"
	"snake/src/snake/player"
	"sync"
	"time"
)

var (
	pairingChannels    = map[string]chan *player.Player{}
	pairingChannelLock = sync.Mutex{}
)

type Game struct {
	Width     int            `json:"width"`
	Height    int            `json:"height"`
	PlayerOne *player.Player `json:"playerOne"`
	PlayerTwo *player.Player `json:"playerTwo"`
	HasEnded  bool           `json:"hasEnded"`
	Food      [][2]int       `json:"food"`
}

func getPairingChannel(key string) chan *player.Player {
	pairingChannelLock.Lock()
	defer pairingChannelLock.Unlock()
	if _, found := pairingChannels[key]; !found {
		pairingChannels[key] = make(chan *player.Player, 0)
	}
	return pairingChannels[key]
}

func Pair(player *player.Player) (myName, theirName string) {
	pairingChannel := getPairingChannel(player.Path)
	select {
	case pairingChannel <- player:
		myName = "playerTwo"
		theirName = "playerOne"
	case otherPlayer := <-pairingChannel:
		go create(player, otherPlayer)
		myName = "playerOne"
		theirName = "playerTwo"
	}
	return myName, theirName
}

func create(PlayerOne *player.Player, PlayerTwo *player.Player) {
	if PlayerOne.Disconnected || PlayerTwo.Disconnected {
		if !PlayerTwo.Disconnected {
			Pair(PlayerTwo)
		}
		if !PlayerOne.Disconnected {
			Pair(PlayerOne)
		}
		return
	}
	PlayerOne.Position = [][2]int{
		{3, 7},
		{3, 6},
		{3, 5},
		{3, 4},
		{3, 3},
	}
	PlayerOne.Heading = "down"
	PlayerTwo.Position = [][2]int{
		{46, 42},
		{46, 43},
		{46, 44},
		{46, 45},
		{46, 46},
	}
	PlayerTwo.Heading = "up"
	game := &Game{
		Width:     50,
		Height:    50,
		PlayerOne: PlayerOne,
		PlayerTwo: PlayerTwo,
		Food:      [][2]int{},
	}
	game.run()
}

func (game *Game) run() {
	timeInterval := 2e8
	moveTicker := time.Tick(time.Duration(timeInterval))
	foodTicker := time.Tick(1e9)
	for {
		select {
		case <-moveTicker:
			game.PlayerOne.AdvancePosition()
			game.PlayerTwo.AdvancePosition()
			game.checkForLoser()
			game.PlayerOne.ToClient <- game
			game.PlayerTwo.ToClient <- game
			if game.HasEnded {
				err := StoreScore("playerOne", len(game.PlayerOne.Position))
				if err != nil {
					log.Printf("Failed to save playerOne's score: %v", err)
				}
				err = StoreScore("playerTwo", len(game.PlayerTwo.Position))
				if err != nil {
					log.Printf("Failed to save playerTwo's score: %v", err)
				}
				close(game.PlayerOne.ToClient)
				close(game.PlayerTwo.ToClient)
				return
			} else {
				game.eatFood()
				if game.PlayerOne.JustAte {
					timeInterval = timeInterval * 97 / 100
				}
				if game.PlayerTwo.JustAte {
					timeInterval = timeInterval * 97 / 100
				}
				if game.PlayerOne.JustAte || game.PlayerTwo.JustAte {
					moveTicker = time.Tick(time.Duration(timeInterval))
				}
			}
		case <-foodTicker:
			x := rand.Int() % game.Width
			y := rand.Int() % game.Height
			game.Food = append(game.Food, [2]int{x, y})
		case update := <-game.PlayerOne.FromClient:
			game.PlayerOne.UpdateHeading(update)
		case update := <-game.PlayerTwo.FromClient:
			game.PlayerTwo.UpdateHeading(update)
		}
	}
}

func (game *Game) eatFood() {
	remainingFood := [][2]int{}
	for _, location := range game.Food {
		if game.PlayerTwo.Position[0] == location {
			game.PlayerTwo.JustAte = true
		} else if game.PlayerOne.Position[0] == location {
			game.PlayerOne.JustAte = true
		} else {
			remainingFood = append(remainingFood, location)
		}
	}
	game.Food = remainingFood
}

func (game *Game) checkForLoser() {
	game.PlayerOne.LostGame = game.PlayerOne.ExceededBounds(game.Width, game.Height) || game.PlayerOne.CollidedInto(game.PlayerTwo) || game.PlayerOne.HitSelf()
	game.PlayerTwo.LostGame = game.PlayerTwo.ExceededBounds(game.Width, game.Height) || game.PlayerTwo.CollidedInto(game.PlayerOne) || game.PlayerTwo.HitSelf()
	game.HasEnded = game.PlayerTwo.LostGame || game.PlayerOne.LostGame
}
