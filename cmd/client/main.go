package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

const queueURL = "amqp://guest:guest@localhost:5672/"

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) {
	return func(s routing.PlayingState) {
		defer fmt.Print("> ")
		gs.HandlePause(s)
	}
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) {
	return func(mv gamelogic.ArmyMove) {
		defer fmt.Print("> ")
		gs.HandleMove(mv)
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
		handlerMove(state),
	)

	if err != nil {
		log.Fatalf("could not subscribe to pause: %v", err)
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
			fmt.Println("Spamming not allowed yet!")
		default:
			fmt.Println("command unknown")
			continue
		}
	}
}
