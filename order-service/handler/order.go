package handler

import (
	"log"
	"time"

	"bytemart/order-service/db"
	"bytemart/order-service/rabbitmq"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type OrderHandler struct {
	publisher *rabbitmq.Publisher
}

func NewOrderHandler(pub *rabbitmq.Publisher) *OrderHandler {
	return &OrderHandler{publisher: pub}
}

type createOrderRequest struct {
	UserID    string  `json:"user_id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Total     float64 `json:"total"`
}

type OrderEvent struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Total     float64   `json:"total"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	var req createOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Request Body isn't valid",
		})
	}

	if req.UserID == "" || req.ProductID == "" || req.Quantity <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "user_id, product_id, and quantity is not nullable",
		})
	}

	order := db.Order{
		ID:        uuid.New().String(),
		UserID:    req.UserID,
		ProductID: req.ProductID,
		Quantity:  req.Quantity,
		Total:     req.Total,
		Status:    "pending",
	}

	if result := db.DB.Create(&order); result.Error != nil {
		log.Printf("Failed to save order: %v", result.Error)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save order",
		})
	}

	go func() {
		event := OrderEvent{
			OrderID:   order.ID,
			UserID:    order.UserID,
			ProductID: order.ProductID,
			Quantity:  order.Quantity,
			Total:     order.Total,
			Status:    order.Status,
			CreatedAt: order.CreatedAt,
		}
		if err := h.publisher.Publish("order.created", event); err != nil {
			log.Printf("Failed to pubish event: %v", err)
		}
	}()

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Order successfully created",
		"order":   order,
	})
}

func (h *OrderHandler) GetOrders(c *fiber.Ctx) error {
	var orders []db.Order
	if result := db.DB.Order("created_at DESC").Find(&orders); result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Unable to fetch orders data",
		})
	}
	return c.JSON(orders)
}
