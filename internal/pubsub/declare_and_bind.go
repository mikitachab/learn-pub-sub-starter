package pubsub

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
	channel, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("cant create channel: %v", err)
	}

	durable := false
	autoDelete := false
	exclusive := false

	if simpleQueueType == QUEUE_Durable {
		durable = true
	}

	if simpleQueueType == QUEUE_Transient {
		autoDelete = true
		exclusive = true
	}

	queue, err := channel.QueueDeclare(
		queueName,
		durable,
		autoDelete,
		exclusive,
		false,
		amqp.Table{"x-dead-letter-exchange": "peril_dlx"},
	)

	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue: %v", err)
	}

	err = channel.QueueBind(
		queueName,
		key,
		exchange,
		false,
		nil,
	)

	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %v", err)
	}

	return channel, queue, nil
}
