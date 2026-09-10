package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/hermes/backend/internal/db"
	"github.com/hermes/backend/internal/handlers"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env (abaikan error: env vars bisa datang dari environment)
	_ = godotenv.Load()

	app := fiber.New(fiber.Config{
		AppName: "Hermes Backend API v1.0",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, X-API-Key",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "hermes-backend",
		})
	})

	// Database connection (Supabase via DATABASE_URL dari .env)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	dbService, err := db.NewDatabaseService(dsn)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer dbService.Close()
	log.Println("Connected to database (Supabase)")

	// API v1 routes
	api := app.Group("/api/v1")

	// Agent endpoints
	agentHandler := handlers.NewAgentHandler(dbService)
	api.Post("/agent/register", agentHandler.Register)
	api.Post("/agent/:server_id/metrics", agentHandler.ReceiveMetrics)
	api.Post("/agent/:server_id/file-candidates", agentHandler.ReceiveFileCandidates)
	api.Post("/agent/:server_id/backup-status", agentHandler.ReceiveBackupStatus)

	// Server management endpoints
	serverHandler := handlers.NewServerHandler(dbService)
	api.Get("/servers", serverHandler.GetServers)
	api.Get("/servers/:id/prediction", serverHandler.GetPrediction)
	api.Get("/servers/:id/cleanup-candidates", serverHandler.GetCleanupCandidates)
	api.Post("/servers/:id/cleanup", serverHandler.TriggerCleanup)

	// Cleanup management
	cleanupHandler := handlers.NewCleanupHandler(dbService)
	api.Post("/cleanup-actions/:id/restore", cleanupHandler.RestoreFiles)

	// Notification webhook
	notifyHandler := handlers.NewNotificationHandler()
	api.Post("/notifications/whatsapp/webhook", notifyHandler.WhatsAppWebhook)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001" // 3000 dipakai dashboard Next.js
	}

	log.Printf("Starting server on port %s", port)
	log.Fatal(app.Listen(":" + port))
}