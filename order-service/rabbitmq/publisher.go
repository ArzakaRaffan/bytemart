package rabbitmq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(
		"bytemart.events", // nama exchange
		"topic",           // tipe topic
		true,              // durable
		false, false, false, nil,
	)
	if err != nil {
		return nil, err
	}

	log.Println("RabbitMQ Publisher is ready")
	return &Publisher{conn: conn, channel: ch}, nil
}

func (p *Publisher) Publish(routingKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(ctx,
		"bytemart.events", // exchange yang kita buat tadi
		routingKey,        // contoh: "order.created"
		false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		},
	)

	if err == nil {
		log.Printf("Event published [%s]: %s", routingKey, string(body))
	}
	return err
}

func (p *Publisher) Close() {
	p.channel.Close()
	p.conn.Close()
}
