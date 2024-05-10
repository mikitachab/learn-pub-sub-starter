package main

import (
	"fmt"
	"log"

	game "github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

const queueURL = "amqp://guest:guest@localhost:5672/"

func main() {
	conn, err := amqp.Dial(queueURL)
	if err != nil {
		fmt.Println("cannot connect to rabbit")
		fmt.Println(err)
		return
	}

	defer conn.Close()

	fmt.Println("connection success")
	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("channel.open: %s", err)
	}

	game.PrintServerHelp()

Loop:
	for {
		words := game.GetInput()
		if len(words) == 0 {
			continue
		}

		command := words[0]

		switch command {
		case "pause":
			pubsub.PublishJSON(
				channel,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{IsPaused: true},
			)
		case "resume":
			pubsub.PublishJSON(
				channel,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{IsPaused: false},
			)
		case "exit":
			break Loop
		default:
			fmt.Println("unknown command")
			continue
		}
	}
}
