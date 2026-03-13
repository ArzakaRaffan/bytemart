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

type Product struct {
	ID        string    `gorm:"primaryKey;type:varchar(20)" json:"product_id"`
	Name      string    `gorm:"not null"                    json:"name"`
	Stock     int       `gorm:"not null;default:0"          json:"stock"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"              json:"updated_at"`
}

type StockLog struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OrderID   string    `gorm:"not null;index"           json:"order_id"`
	ProductID string    `gorm:"not null"                 json:"product_id"`
	Deducted  int       `gorm:"not null"                 json:"deducted"`
	Remaining int       `gorm:"not null"                 json:"remaining"`
	Note      string    `gorm:"not null"                 json:"note"`
	CreatedAt time.Time `gorm:"autoCreateTime"           json:"created_at"`
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
		log.Printf("Waiting for postgres. Attempt #%d/10", i+1)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("Database connection failed: %w", err)
	}

	if err := DB.AutoMigrate(&Product{}, &StockLog{}); err != nil {
		return fmt.Errorf("Migration Failed: %w", err)
	}

	seedProducts()

	log.Println("Inventory Service added to postgres")
	return nil
}

func seedProducts() {
	var count int64
	DB.Model(&Product{}).Count(&count)
	if count > 0 {
		return
	}

	products := []Product{
		{ID: "PROD-001", Name: "Laptop Gaming", Stock: 100},
		{ID: "PROD-002", Name: "Wireless Mouse", Stock: 50},
		{ID: "PROD-003", Name: "Mechanical Keyboard", Stock: 200},
	}
	DB.Create(&products)
	log.Println("Seeding data successfully added")
}
