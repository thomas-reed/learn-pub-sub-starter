package main

import (
	"fmt"
	"log"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	const amqpConnStr = "amqp://guest:guest@localhost:5672/"

	amqpConn, err := amqp.Dial(amqpConnStr)
	if err != nil {
		log.Fatalf("Error connecting to RabbitMQ server: %v", err)
	}
	defer amqpConn.Close()
	fmt.Println("Connected to RabbitMQ server!")

	ch, err := amqpConn.Channel()
	if err != nil {
		log.Fatalf("Error creating channel: %v", err)
	}

	_, queue, err := pubsub.DeclareAndBind(
		amqpConn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug + ".*",
		pubsub.DurableQueue,
	)
	if err != nil {
		log.Fatalf("Failed to declare and bind channel/queue: %v", err)
	}
	defer ch.Close()
	fmt.Printf("Queue %v declared and bound!\n", queue.Name)

	err = pubsub.SubscribeGob(
		amqpConn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug + ".*",
		pubsub.DurableQueue,
		handlerGameLog(),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to game_log!")
	}

	gamelogic.PrintServerHelp()
	for {
		cmd := gamelogic.GetInput()
		if len(cmd) == 0 {
			continue
		}
		switch cmd[0] {
		case "help":
    	gamelogic.PrintServerHelp()
		case "pause":
			fmt.Println("Sending pause cmd..")
			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: true,
				},
			)
			if err != nil {
				log.Printf("Error sending pause cmd: %v", err)
			}
		case "resume":
			fmt.Println("Sending resume cmd..")
			err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{
					IsPaused: false,
				},
			)
			if err != nil {
				log.Printf("Error sending resume cmd: %v", err)
			}
		case "quit":
			fmt.Println("Exiting..")
			return
		default:
			fmt.Printf("Unknown cmd - %v\n", cmd[0])
			continue
		}
	}
}
