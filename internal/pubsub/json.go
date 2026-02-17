package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Acktype int

const (
	Ack Acktype = iota
	NackRequeue
	NackDiscard
)

func PublishJSON[T any](
	ch *amqp.Channel,
	exchange, key string,
	val T,
) error {
	json, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("Error marshalling value: %w", err)
	}
	ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body: json,
		},
	)
	return nil
}

func SubscribeJSON[T any](
  conn *amqp.Connection,
  exchange,
  queueName,
  key string,
  queueType SimpleQueueType,
  handler func(T) Acktype,
) error {
	ch, queue, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		queueType,
	)
	if err != nil {
		return fmt.Errorf("Error declaring/binding queue: %w", err)
	}

	messageCh, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("Error consuming queue: %w", err)
	}

	go func() {
		defer ch.Close()
		for msg := range messageCh {
			var msgData T
			if err := json.Unmarshal(msg.Body, &msgData); err != nil {
				log.Printf("Error occurred unmarshalling message data: %v\n", err)
				msg.Nack(false, false) // don't requeue bad messages
				continue
			}
			ack := handler(msgData)
			switch ack {
			case Ack:
				msg.Ack(false)
			case NackRequeue:
				msg.Nack(false, true)
			case NackDiscard:
				msg.Nack(false, false)
			}
		}
	}()

	return nil
}
