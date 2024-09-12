package rabbitmq

import (
	"fmt"
	"github.com/streadway/amqp"
)

func SetupRabbitMQ(url string) (*amqp.Channel, *amqp.Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		err := conn.Close()
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("failed to open RabbitMQ channel: %w", err)
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
