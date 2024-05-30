package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

const queueURL = "amqp://guest:guest@localhost:5672/"

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(s routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(s)
		return pubsub.ACK
	}
}

func handlerMove(gs *gamelogic.GameState, pch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(mv gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")
		result := gs.HandleMove(mv)
		if result == gamelogic.MoveOutComeSafe || result == gamelogic.MoveOutcomeSamePlayer {
			return pubsub.ACK
		}
		if result == gamelogic.MoveOutcomeMakeWar {
			err := pubsub.PublishJSON(
				pch,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.GetUsername(),
				gamelogic.RecognitionOfWar{
					Attacker: mv.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				fmt.Println("cant publish make war, requeue")
				return pubsub.NACK_Requeue
			}

			return pubsub.ACK
		}
		return pubsub.NACK_Discard
	}
}

func handlerWar(gs *gamelogic.GameState, channel *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func(rw gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")
		result, winner, loser := gs.HandleWar(rw)
		switch result {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NACK_Requeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NACK_Discard
		case gamelogic.WarOutcomeOpponentWon:
			err := pubsub.PublishGOB(
				channel,
				routing.ExchangePerilTopic,
				routing.GameLogSlug+"."+gs.GetUsername(),
				routing.GameLog{
					CurrentTime: time.Now(),
					Message:     winner + " won a war against " + loser,
					Username:    gs.GetUsername(),
				},
			)
			if err != nil {
				fmt.Println("failed to publlish game log")
			}
			return pubsub.ACK
		case gamelogic.WarOutcomeYouWon:
			err := pubsub.PublishJSON(
				channel,
				routing.ExchangePerilTopic,
				routing.GameLogSlug+"."+gs.GetUsername(),
				routing.GameLog{
					CurrentTime: time.Now(),
					Message:     winner + " won a war against " + loser,
					Username:    gs.GetUsername(),
				},
			)
			if err != nil {
				fmt.Println("failed to publlish game log")
			}
			return pubsub.ACK
		case gamelogic.WarOutcomeDraw:
			err := pubsub.PublishJSON(
				channel,
				routing.ExchangePerilTopic,
				routing.GameLogSlug+"."+gs.GetUsername(),
				routing.GameLog{
					CurrentTime: time.Now(),
					Message:     "A war between" + winner + " and " + loser + " resulted in a draw",
					Username:    gs.GetUsername(),
				},
			)
			if err != nil {
				fmt.Println("failed to publlish game log")
			}
			return pubsub.ACK
		}

		fmt.Println("error: unknown war outcome")
		return pubsub.NACK_Discard
	}
}

func main() {
	fmt.Println("Starting Peril client...")

	conn, err := amqp.Dial(queueURL)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("channel.open: %s", err)
	}

	fmt.Println("Peril game client connected to RabbitMQ!")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}

	state := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+username,
		routing.PauseKey,
		pubsub.QUEUE_Transient,
		handlerPause(state),
	)

	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+username,
		routing.ArmyMovesPrefix+".*",
		pubsub.QUEUE_Transient,
		handlerMove(state, channel),
	)

	if err != nil {
		log.Fatalf("could not subscribe to move: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix+".*",
		pubsub.QUEUE_Durable,
		handlerWar(state, channel),
	)

	if err != nil {
		log.Fatalf("could not subscribe to war: %v", err)
	}

Loop:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}

		command := words[0]

		switch command {
		case "quit":
			gamelogic.PrintQuit()
			break Loop
		case "spawn":
			err := state.CommandSpawn(words)
			if err != nil {
				fmt.Println(err)
			}
		case "move":
			mv, err := state.CommandMove(words)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("ok", mv)
			}
			err = pubsub.PublishJSON(
				channel,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+mv.Player.Username,
				mv,
			)
			if err != nil {
				fmt.Println("publish error", err)
			} else {
				fmt.Println("publish move success")
			}
		case "status":
			state.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			if len(words) != 2 {
				fmt.Println("error: Usage spam N")
			}
			nStr := words[1]
			n, err := strconv.Atoi(nStr)
			if err != nil {
				fmt.Println("error: Usage spam N, N is integer number")
			}

			for j := 0; j < n; j++ {
				log := gamelogic.GetMaliciousLog()
				pubsub.PublishGOB(
					channel,
					routing.ExchangePerilTopic,
					routing.GameLogSlug+"."+username,
					routing.GameLog{
						CurrentTime: time.Now(),
						Message:     log,
						Username:    username,
					},
				)
			}
		default:
			fmt.Println("command unknown")
			continue
		}
	}
}
