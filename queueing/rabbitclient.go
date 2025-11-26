package queueing

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/rabbitmq/amqp091-go"
)

type Client struct {
	conn *amqp091.Connection
	ch   *amqp091.Channel
	url  string
	mu   sync.Mutex
}

func NewRabbitClient(url string) (*Client, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("error creating RabbitMQ connection: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error creating RabbitMQ channel: %w", err)
	}

	// Only allow one message to be processed at a time
	if err := ch.Qos(1, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("error setting Qos: %w", err)
	}

	return &Client{
		conn: conn,
		ch:   ch,
		url:  url,
	}, nil
}

func (c *Client) Close() error {
	if err := c.ch.Close(); err != nil {
		// We still try to close the connection even if channel close fails
		c.conn.Close()
		return fmt.Errorf("error closing RabbitMQ channel: %w", err)
	}
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("error closing RabbitMQ connection: %w", err)
	}
	return nil
}

// Ensures that a queue with the given name exists.
func (c *Client) EnsureQueue(queue string) error {
	_, err := c.ch.QueueDeclare(
		queue, // name
		false, // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", queue, err)
	}
	return nil
}

// Sends a byte payload to the named queue.
func (c *Client) Publish(queue string, body []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.ch.Publish(
		"",
		queue,
		false,
		false,
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message to queue %s: %w", queue, err)
	}

	return nil
}

// Registers a consumer for the given queue name, processing messages with the provided handler function.
func (c *Client) StartConsumer(queue string, handler func([]byte) error) error {
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
		return fmt.Errorf("failed to consume queue %s: %w", queue, err)
	}

	go func() {
		for d := range msgs {
			go func(b []byte) {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("panic in consumer handler", slog.Any("recover", r))
					}
				}()

				if err := handler(b); err != nil {
					slog.Error("Consumer handler failed", slog.String("queue", queue), slog.String("error", err.Error()))
				}
			}(d.Body)
		}
		slog.Info("consumer channel closed for queue", slog.String("queue", queue))
	}()

	return nil
}
