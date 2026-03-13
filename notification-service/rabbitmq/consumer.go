package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"

	"bytemart/notification-service/db"

	amqp "github.com/rabbitmq/amqp091-go"
)

type orderEvent struct {
	OrderID   string  `json:"order_id"`
	UserID    string  `json:"user_id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Total     float64 `json:"total"`
}

func StartConsumer(url string) error {
	conn, err := amqp.Dial(url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	// Declare exchange yang SAMA dengan producer
	ch.ExchangeDeclare(
		"bytemart.events",
		"topic",
		true,
		false, false, false, nil,
	)

	// Buat queue khusus notification
	q, err := ch.QueueDeclare(
		"notification.queue",
		true, // durable
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Bind queue ke exchange dengan routing key "order.created"
	ch.QueueBind(q.Name, "order.created", "bytemart.events", false, nil)

	// Batasi 1 pesan diproses sekaligus
	ch.Qos(1, 0, false)

	// Mulai consume
	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Println("📬 Notification Service is listening order.created notifications...")

	go func() {
		for msg := range msgs {
			var event orderEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("Failed to parse event: %v", err)
				msg.Nack(false, false)
				continue
			}

			shortID := event.OrderID
			if len(shortID) > 8 {
				shortID = shortID[:8]
			}

			notif := db.Notification{
				OrderID: event.OrderID,
				UserID:  event.UserID,
				Message: fmt.Sprintf(
					"Hello, %s! Order #%s for %d items worth Rp%.0f has been received.",
					event.UserID, shortID, event.Quantity, event.Total,
				),
			}

			if result := db.DB.Create(&notif); result.Error != nil {
				log.Printf("Failed to save notifications: %v", result.Error)
				msg.Nack(false, true)
				continue
			}

			log.Printf("Notification added for[%s]: %s", event.UserID, notif.Message)
			msg.Ack(false)
		}
	}()

	return nil
}
