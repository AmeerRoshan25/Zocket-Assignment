package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"product-management-system/db"
	"product-management-system/queue"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8" // Redis client
)

var redisClient *redis.Client
var ctx = context.Background()
var logger = logrus.New()

func initLogger() {
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel) // Adjust the logging level as needed
}

// Initialize Redis
func initRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password by default
		DB:       0,                // Default DB
	})
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		logger.WithError(err).Fatal("Could not connect to Redis")
	}
	logger.Info("Connected to Redis successfully!")
}

func main() {
	initLogger()
	queue.InitLogger() // Initialize the shared logger

	// Log application start
	logger.Info("Starting the Product Management System application...")
	// Initialize the database
	db.InitDB()
	defer db.DB.Close()

	// Initialize Redis
	initRedis()

	// Initialize RabbitMQ
	queue.InitQueue()
	defer queue.Conn.Close()
	defer queue.Channel.Close()

	// Start RabbitMQ consumer in a goroutine
	go queue.StartConsumer("image_processing")

	// Create a Gin router
	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	// Create a new user
	r.POST("/users", func(c *gin.Context) {
		var user struct {
			UserID   int    `json:"user_id"`
			UserName string `json:"user_name"`
		}

		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.DB.Exec(context.Background(), `
			INSERT INTO users (user_id, user_name) VALUES ($1, $2)`,
			user.UserID, user.UserName,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
	})

	// Create a new product
	r.POST("/products", func(c *gin.Context) {
		logger.Info("Received request to create a new product")

		var product struct {
			UserID             int      `json:"user_id"`
			ProductName        string   `json:"product_name"`
			ProductDescription string   `json:"product_description"`
			ProductImages      []string `json:"product_images"`
			ProductPrice       float64  `json:"product_price"`
		}

		if err := c.BindJSON(&product); err != nil {
			logger.WithError(err).Error("Failed to parse product creation request")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Process and log further steps
		logger.WithFields(logrus.Fields{
			"user_id":      product.UserID,
			"product_name": product.ProductName,
		}).Info("Product creation in progress...")
	})

	// Get product by ID with caching
	r.GET("/products/:id", func(c *gin.Context) {
		productID := c.Param("id")

		// Check cache
		cachedProduct, err := redisClient.Get(ctx, "product:"+productID).Result()
		if err == nil {
			var product map[string]interface{}
			json.Unmarshal([]byte(cachedProduct), &product)
			c.JSON(http.StatusOK, gin.H{"data": product, "source": "cache"})
			return
		}

		// Fetch from database
		var product struct {
			ID                 int      `json:"id"`
			UserID             int      `json:"user_id"`
			ProductName        string   `json:"product_name"`
			ProductDescription string   `json:"product_description"`
			ProductImages      []string `json:"product_images"`
			ProductPrice       float64  `json:"product_price"`
		}
		err = db.DB.QueryRow(ctx, `
			SELECT id, user_id, product_name, product_description, product_images, product_price
			FROM products WHERE id = $1
		`, productID).Scan(&product.ID, &product.UserID, &product.ProductName, &product.ProductDescription, &product.ProductImages, &product.ProductPrice)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		// Cache the product
		productJSON, _ := json.Marshal(product)
		redisClient.Set(ctx, "product:"+productID, productJSON, 10*time.Minute)

		c.JSON(http.StatusOK, gin.H{"data": product, "source": "database"})
	})

	// Fetch products by user ID with caching
	r.GET("/products", func(c *gin.Context) {
		userID := c.Query("user_id")
		minPrice := c.Query("min_price")
		maxPrice := c.Query("max_price")
		productName := c.Query("product_name")

		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			return
		}

		// Build the query dynamically
		query := "SELECT id, product_name, product_description, product_price FROM products WHERE user_id = $1"
		args := []interface{}{userID}
		argIndex := 2

		if minPrice != "" {
			query += " AND product_price >= $" + fmt.Sprintf("%d", argIndex)
			args = append(args, minPrice)
			argIndex++
		}
		if maxPrice != "" {
			query += " AND product_price <= $" + fmt.Sprintf("%d", argIndex)
			args = append(args, maxPrice)
			argIndex++
		}
		if productName != "" {
			query += " AND product_name ILIKE $" + fmt.Sprintf("%d", argIndex)
			args = append(args, "%"+productName+"%")
			argIndex++
		}

		rows, err := db.DB.Query(ctx, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		products := []map[string]interface{}{}
		for rows.Next() {
			var product struct {
				ID                 int     `json:"id"`
				ProductName        string  `json:"product_name"`
				ProductDescription string  `json:"product_description"`
				ProductPrice       float64 `json:"product_price"`
			}
			if err := rows.Scan(&product.ID, &product.ProductName, &product.ProductDescription, &product.ProductPrice); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			products = append(products, map[string]interface{}{
				"id":                  product.ID,
				"product_name":        product.ProductName,
				"product_description": product.ProductDescription,
				"product_price":       product.ProductPrice,
			})
		}

		c.JSON(http.StatusOK, products)
	})

	r.PUT("/products/:id", func(c *gin.Context) {
		productID := c.Param("id")

		var updatedProduct struct {
			UserID             int      `json:"user_id"`
			ProductName        string   `json:"product_name"`
			ProductDescription string   `json:"product_description"`
			ProductImages      []string `json:"product_images"`
			ProductPrice       float64  `json:"product_price"`
		}
		if err := c.BindJSON(&updatedProduct); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		_, err := db.DB.Exec(ctx, `
			UPDATE products
			SET user_id = $1, product_name = $2, product_description = $3, product_images = $4, product_price = $5
			WHERE id = $6
		`, updatedProduct.UserID, updatedProduct.ProductName, updatedProduct.ProductDescription, updatedProduct.ProductImages, updatedProduct.ProductPrice, productID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
			return
		}

		// Invalidate the Redis cache for the updated product
		redisKey := "product:" + productID
		if err := redisClient.Del(ctx, redisKey).Err(); err != nil {
			log.Printf("Failed to invalidate cache for product %s: %v", productID, err)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully"})
	})

	r.DELETE("/products/:id", func(c *gin.Context) {
		productID := c.Param("id")

		_, err := db.DB.Exec(ctx, `DELETE FROM products WHERE id = $1`, productID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
			return
		}

		// Invalidate the Redis cache for the deleted product
		redisKey := "product:" + productID
		if err := redisClient.Del(ctx, redisKey).Err(); err != nil {
			log.Printf("Failed to invalidate cache for product %s: %v", productID, err)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
	})

	r.Run(":8080")
}
