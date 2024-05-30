package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	body, err := json.Marshal(val)
	if err != nil {
		return err
	}

	msg := amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}

	ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		msg,
	)

	return nil
}

func PublishGOB[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)

	err := enc.Encode(val)
	if err != nil {
		return err
	}

	msg := amqp.Publishing{
		ContentType: "application/gob",
		Body:        buffer.Bytes(),
	}

	ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		msg,
	)

	return nil
}
