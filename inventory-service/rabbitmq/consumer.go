package rabbitmq

import (
	"encoding/json"
	"log"

	"bytemart/inventory-service/db"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type orderEvent struct {
	OrderID   string `json:"order_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
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

	ch.ExchangeDeclare(
		"bytemart.events",
		"topic",
		true,
		false, false, false, nil,
	)

	// Queue BERBEDA dari notification!
	q, err := ch.QueueDeclare(
		"inventory.queue",
		true,
		false, false, false, nil,
	)
	if err != nil {
		return err
	}

	ch.QueueBind(q.Name, "order.created", "bytemart.events", false, nil)
	ch.Qos(1, 0, false)

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	log.Println("📦 Inventory Service siap mendengarkan event order.created...")

	go func() {
		for msg := range msgs {
			var event orderEvent
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				log.Printf("❌ Gagal parse event: %v", err)
				msg.Nack(false, false)
				continue
			}

			if err := processStockDeduction(event); err != nil {
				log.Printf("❌ Gagal proses stok: %v", err)
				msg.Nack(false, true)
				continue
			}

			msg.Ack(false)
		}
	}()

	return nil
}

func processStockDeduction(event orderEvent) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var product db.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&product, "id = ?", event.ProductID).Error; err != nil {
			saveLog(tx, event, 0, 0, "produk tidak ditemukan")
			log.Printf("⚠️  Produk %s tidak ditemukan", event.ProductID)
			return nil
		}

		var note string
		var deducted int

		if product.Stock < event.Quantity {
			note = "stok tidak mencukupi"
			log.Printf("Stok %s tidak cukup! Tersedia: %d, Diminta: %d",
				event.ProductID, product.Stock, event.Quantity)
		} else {
			product.Stock -= event.Quantity
			deducted = event.Quantity
			note = "stok berhasil dikurangi"
			tx.Save(&product)
			log.Printf("Stok %s dikurangi %d, sisa: %d",
				event.ProductID, event.Quantity, product.Stock)
		}

		saveLog(tx, event, deducted, product.Stock, note)
		return nil
	})
}

func saveLog(tx *gorm.DB, event orderEvent, deducted, remaining int, note string) {
	tx.Create(&db.StockLog{
		OrderID:   event.OrderID,
		ProductID: event.ProductID,
		Deducted:  deducted,
		Remaining: remaining,
		Note:      note,
	})
}
