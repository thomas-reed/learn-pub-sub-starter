package main

import (
	"fmt"
	"log"
	"strconv"

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

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	gs := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		amqpConn,
		routing.ExchangePerilDirect,
		routing.PauseKey + "." + username,
		routing.PauseKey,
		pubsub.TransientQueue,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to pause!")
	}

	err = pubsub.SubscribeJSON(
		amqpConn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix + "." + username,
		routing.ArmyMovesPrefix + ".*",
		pubsub.TransientQueue,
		handlerMove(gs, ch),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to move!")
	}

	err = pubsub.SubscribeJSON(
		amqpConn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix,
		routing.WarRecognitionsPrefix + ".*",
		pubsub.DurableQueue,
		handlerWar(gs, ch),
	)
	if err != nil {
		log.Fatalf("Failed to subscribe to war!")
	}
	
	for {
		cmd := gamelogic.GetInput()
		if len(cmd) == 0 {
			continue
		}
		switch cmd[0] {
		case "help":
    	gamelogic.PrintClientHelp()
		case "spawn":
			err = gs.CommandSpawn(cmd)
			if err != nil {
				log.Println(err)
			}
		case "move":
			move, err := gs.CommandMove(cmd)
			if err != nil {
				log.Println(err)
			}
			if err = pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix + "." + username,
				move,
			); err != nil {
				log.Println(err)
			} else {
				fmt.Println("Published move!")
			}
		case "status":
			gs.CommandStatus()
		case "spam":
			if len(cmd) < 2 {
				fmt.Println("Spam usage: spam <num>")
				continue
			}
			num, err := strconv.Atoi(cmd[1])
			if err != nil {
				fmt.Println("Invalid number of spams - Spam usage: spam <num>")
				continue
			}
			for i := 0; i < num; i++ {
				err := PublishGameLog(ch, gs.Player.Username, gamelogic.GetMaliciousLog())
				if err != nil {
					fmt.Printf("Error publishing gamelog: %v\n", err)
				}
			}
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Printf("Unknown cmd - %v\n", cmd[0])
			continue
		}
	}
}
