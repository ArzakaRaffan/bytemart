package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

type Order struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"order_id"`
	UserID    string    `gorm:"not null"                    json:"user_id"`
	ProductID string    `gorm:"not null"                    json:"product_id"`
	Quantity  int       `gorm:"not null"                    json:"quantity"`
	Total     float64   `gorm:"not null"                    json:"total"`
	Status    string    `gorm:"default:pending"             json:"status"`
	CreatedAt time.Time `gorm:"autoCreateTime"              json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"              json:"updated_at"`
}

func Connect() error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	var err error
	for i := 0; i < 10; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			break
		}
		log.Printf("Waiting for postgres... tries #%d/10", i+1)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("Connection Error: %w", err)
	}

	if err := DB.AutoMigrate(&Order{}); err != nil {
		return fmt.Errorf("Migration Error: %w", err)
	}

	log.Println("Order Service connected to Postgres")
	return nil
}
