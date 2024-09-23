package rabbitmq

import (
	"fmt"
	"github.com/streadway/amqp"
	"time"
)

func SetupRabbitMQ(url string) (*amqp.Channel, *amqp.Connection, error) {
	var conn *amqp.Connection
	var err error

	for attempts := 0; attempts < 5; attempts++ {
		conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		time.Sleep(time.Duration(30+attempts*2) * time.Second)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ after 3 attempts: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
	}

	// Declare the queue
	_, err = ch.QueueDeclare(
		"source_data_queue",
		true,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return ch, conn, nil
}

func CloseRabbitMQ(ch *amqp.Channel, conn *amqp.Connection) error {
	var errs []error

	// Cancel all consumers on the channel
	if err := ch.Cancel("", false); err != nil {
		errs = append(errs, fmt.Errorf("error cancelling RabbitMQ consumption: %w", err))
	}

	// Attempt to close the channel
	if err := ch.Close(); err != nil {
		errs = append(errs, fmt.Errorf("error closing RabbitMQ channel: %w", err))
	}

	// Close the connection
	if err := conn.Close(); err != nil {
		errs = append(errs, fmt.Errorf("error closing RabbitMQ connection: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred during RabbitMQ shutdown: %v", errs)
	}

	return nil
}
