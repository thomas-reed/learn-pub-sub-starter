package main

import (
	"fmt"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerWar(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func(rw gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")
		outcome, winner, loser := gs.HandleWar(rw)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			err := PublishGameLog(
				ch,
				gs.Player.Username,
				fmt.Sprintf("%s won a war against %s\n", winner, loser),
			)
			if err != nil {
				fmt.Printf("Error: %v", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			err := PublishGameLog(
				ch,
				gs.Player.Username,
				fmt.Sprintf("%s won a war against %s\n", winner, loser),
			)
			if err != nil {
				fmt.Printf("Error: %v", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			err := PublishGameLog(
				ch,
				gs.Player.Username,
				fmt.Sprintf("A war between %s and %s resulted in a draw\n", winner, loser),
			)
			if err != nil {
				fmt.Printf("Error: %v", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		default:
			return pubsub.NackDiscard
		}
	}
}

func PublishGameLog(ch *amqp.Channel, user, message string) error {
	err := pubsub.PublishGob(
		ch,
		routing.ExchangePerilTopic,
		routing.GameLogSlug + "." + user,
		routing.GameLog {
			CurrentTime: time.Now(),
			Message: message,
			Username: user,
		},
	)
	if err != nil {
		return fmt.Errorf("Error publishing game log: %w", err)
	}
	return nil
}