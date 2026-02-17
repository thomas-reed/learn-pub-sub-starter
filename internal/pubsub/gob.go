package pubsub

import (
	"context"
	"encoding/gob"
	"bytes"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishGob[T any](
	ch *amqp.Channel,
	exchange, key string,
	val T,
) error {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(val)
	if err != nil {
		return fmt.Errorf("Error encoding gob: %w", err)
	}
	
	ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/gob",
			Body: buffer.Bytes(),
		},
	)
	return nil
}

func SubscribeGob[T any](
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

	err = ch.Qos(10, 0, false)
	if err != nil {
		return fmt.Errorf("Error setting QoS prefetch count: %w", err)
	}
	messageCh, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("Error consuming queue: %w", err)
	}

	go func() {
		defer ch.Close()
		for msg := range messageCh {
			buffer := bytes.NewBuffer(msg.Body)
			decoder := gob.NewDecoder(buffer)
			var msgData T
			err := decoder.Decode(&msgData)
			if err != nil {
				fmt.Printf("Error occurred decoding message data: %v\n", err)
				msg.Nack(false, false) // don't requeue bad messages
				continue	
			}
			ack := handler(msgData)
			switch ack {
			case Ack:
				fmt.Println("sending Ack..")
				msg.Ack(false)
			case NackRequeue:
				fmt.Println("sending Nack, requeuing..")
				msg.Nack(false, true)
			case NackDiscard:
				fmt.Println("sending Nack, discarding..")
				msg.Nack(false, false)
			}
		}
	}()

	return nil
}