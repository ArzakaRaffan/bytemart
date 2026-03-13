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

type Notification struct {
	ID      uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID string    `gorm:"not null;index"           json:"order_id"`
	UserID  string    `gorm:"not null"                 json:"user_id"`
	Message string    `gorm:"not null"                 json:"message"`
	SentAt  time.Time `gorm:"autoCreateTime"           json:"sent_at"`
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
		log.Printf("Waiting for PostgreSQL... tries #%d/10", i+1)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("Failed to connect to database: %w", err)
	}

	if err := DB.AutoMigrate(&Notification{}); err != nil {
		return fmt.Errorf("Failed to migrate: %w", err)
	}

	log.Println("Notification Service Connected to PostgreSQL")
	return nil
}
