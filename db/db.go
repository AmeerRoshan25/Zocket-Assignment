package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is a global variable to hold the database connection pool
var DB *pgxpool.Pool

// InitDB initializes the database connection pool
func InitDB() {
	// Database connection string
	dsn := "postgres://postgres:ameer@25@localhost:5432/product_management"

	// Create a connection pool
	var err error
	DB, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := DB.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}

	fmt.Println("Connected to the database successfully!")
}
