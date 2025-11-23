package queueing

import (
	"log/slog"

	"github.com/rabbitmq/amqp091-go"
)

type Client struct {
	conn *amqp091.Connection
	ch   *amqp091.Channel
	url  string
}

func NewRabbitClient(url string) (*Client, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		slog.Error("Error creating RabbitMQ connection", slog.String("error", err.Error()))
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		slog.Error("Error creating RabbitMQ channel", slog.String("error", err.Error()))
		conn.Close()
		return nil, err
	}

	ch.Qos(1, 0, false)

	return &Client{
		conn: conn,
		ch:   ch,
		url:  url,
	}, nil
}

func (c *Client) Close() error {
	if err := c.ch.Close(); err != nil {
		slog.Error("Error closing RabbitMQ channel", slog.String("error", err.Error()))
		c.conn.Close()
		return err
	}
	return c.conn.Close()
}

// Sends a byte payload to the named queue, declaring the queue if needed.
func (c *Client) Publish(queue string, body []byte) error {
	slog.Info("Declaring queue", slog.String("queue", queue))
	_, err := c.ch.QueueDeclare(
		queue, // name
		false, // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		slog.Error("Failed to declare queue", slog.String("queue", queue), slog.String("error", err.Error()))
		return err
	}

	slog.Info("Publishing message to RabbitMQ", slog.String("queue", queue), slog.Int("body_size", len(body)))
	err = c.ch.Publish(
		"",
		queue,
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	return err
}

// Registers a consumer for the given queue name, processing messages with the provided handler function.
func (c *Client) StartConsumer(queue string, handler func([]byte) error) error {
	_, err := c.ch.QueueDeclare(
		queue, // name
		false, // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		slog.Error("Failed to declare queue", slog.String("queue", queue), slog.String("error", err.Error()))
		return err
	}

	msgs, err := c.ch.Consume(
		queue, // name
		"",    // consumer
		true,  // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		slog.Error("Failed to consume queue", slog.String("queue", queue), slog.String("error", err.Error()))
		return err
	}

	go func() {
		for d := range msgs {
			go func(b []byte) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("panic in consumer handler", slog.Any("recover", r))
					}
				}()

				handler(b)
			}(d.Body)
		}
		slog.Info("consumer channel closed for queue", slog.String("queue", queue))
	}()

	return nil
}
