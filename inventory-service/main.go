package main

import (
	"log"
	"os"

	"bytemart/inventory-service/db"
	"bytemart/inventory-service/rabbitmq"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	if err := db.Connect(); err != nil {
		log.Fatalf("Database error: %v", err)
	}

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://bytemart:arzaka22@localhost:5672/"
	}

	if err := rabbitmq.StartConsumer(rabbitURL); err != nil {
		log.Fatalf("RabbitMQ consumer error: %v", err)
	}

	app := fiber.New(fiber.Config{
		AppName: "ByteMart Inventory Service v1.0",
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[INVENTORY] ${time} | ${status} | ${latency} | ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET",
	}))

	api := app.Group("/api")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "inventory-service",
		})
	})

	api.Get("/stock", func(c *fiber.Ctx) error {
		var products []db.Product
		if result := db.DB.Find(&products); result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch stock data",
			})
		}
		return c.JSON(products)
	})

	api.Get("/stock-logs", func(c *fiber.Ctx) error {
		var logs []db.StockLog
		if result := db.DB.Order("created_at DESC").Find(&logs); result.Error != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch logs",
			})
		}
		return c.JSON(logs)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3003"
	}

	log.Printf("Inventory Service (Fiber) running on :%s", port)
	log.Fatal(app.Listen(":" + port))	
}