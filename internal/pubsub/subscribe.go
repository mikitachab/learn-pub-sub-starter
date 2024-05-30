package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ACK          = iota
	NACK_Discard = iota
	NACK_Requeue = iota
)

type Acktype = int

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int,
	handler func(T) Acktype,
) error {
	channel, _, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
	)
	if err != nil {
		return err
	}

	err = channel.Qos(10, 0, false)

	if err != nil {
		return err
	}

	deliveryChan, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range deliveryChan {
			var data T
			json.Unmarshal(msg.Body, &data)
			ack := handler(data)
			fmt.Println(ack)
			if ack == ACK {
				msg.Ack(false)
				fmt.Println(queueName, "ACK")
			} else if ack == NACK_Discard {
				msg.Nack(false, false)
				fmt.Println(queueName, "NACK_Discard")
			} else if ack == NACK_Requeue {
				msg.Nack(false, true)
				fmt.Println(queueName, "NACK_Requeue")
			}
		}
	}()

	return nil
}

func SubscribeGOB[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType int,
	handler func(T) Acktype,
) error {
	channel, _, err := DeclareAndBind(
		conn,
		exchange,
		queueName,
		key,
		simpleQueueType,
	)
	if err != nil {
		return err
	}

	err = channel.Qos(10, 0, false)

	if err != nil {
		return err
	}

	deliveryChan, err := channel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for msg := range deliveryChan {
			var data T
			buffer := bytes.NewBuffer(msg.Body)
			dec := gob.NewDecoder(buffer)
			err := dec.Decode(&data)
			if err != nil {
				fmt.Println("error while decoding message from queue")
			}
			ack := handler(data)
			fmt.Println(ack)
			if ack == ACK {
				msg.Ack(false)
				fmt.Println(queueName, "ACK")
			} else if ack == NACK_Discard {
				msg.Nack(false, false)
				fmt.Println(queueName, "NACK_Discard")
			} else if ack == NACK_Requeue {
				msg.Nack(false, true)
				fmt.Println(queueName, "NACK_Requeue")
			}
		}
	}()

	return nil
}
