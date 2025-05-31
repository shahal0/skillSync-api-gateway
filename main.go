package main

import (
	"log"
	"os"
	"skillsync-api-gateway/clients"
	"skillsync-api-gateway/routes"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Initialize gRPC clients
	clients.InitClients()

	// Create Gin router with default middleware
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all origins
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Setup API routes
	routes.SetupRoutes(r)     // Auth routes
	routes.SetupJobRoutes(r)  // Job routes

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8008"
	}

	// Start the server
	log.Printf("Starting API Gateway server on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
