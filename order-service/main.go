package main

import (
	"log"
	"os"

	"bytemart/order-service/db"
	"bytemart/order-service/handler"
	"bytemart/order-service/rabbitmq"

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

	publisher, err := rabbitmq.NewPublisher(rabbitURL)
	if err != nil {
		log.Fatalf("RabbitMQ error: %v", err)
	}
	defer publisher.Close()

	app := fiber.New(fiber.Config{
		AppName: "ByteMart Order Service v1.0",
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[ORDER] ${time} | ${status} | ${latency} | ${method} ${path}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
		AllowMethods: "GET, POST",
	}))

	orderHandler := handler.NewOrderHandler(publisher)

	api := app.Group("/api")
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "order-service"})
	})
	api.Post("/orders", orderHandler.CreateOrder)
	api.Get("/orders", orderHandler.GetOrders)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	log.Printf("Order Service (Fiber) running on :%s", port)
	log.Fatal(app.Listen(":" + port))
}
