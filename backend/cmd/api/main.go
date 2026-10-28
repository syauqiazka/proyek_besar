package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/archon/backend/internal/db"
	"github.com/archon/backend/internal/handlers"
	"github.com/archon/backend/internal/prediction"
)

func main() {
	// Load .env (abaikan error: env vars bisa datang dari environment)
	_ = godotenv.Load()

	app := fiber.New(fiber.Config{
		AppName: "Archon Backend API v1.0",
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
			"service": "archon-backend",
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

	// Start prediction scheduler (background cron)
	predictionScheduler := prediction.NewScheduler(dbService.DB)
	if err := predictionScheduler.Start(); err != nil {
		log.Printf("Warning: failed to start prediction scheduler: %v", err)
	}
	defer predictionScheduler.Stop()

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

	// Onboarding & connection endpoints
	onboardingHandler := handlers.NewOnboardingHandler(dbService)
	api.Post("/servers", onboardingHandler.CreateServer)
	api.Post("/servers/:id/connection-test", onboardingHandler.TestConnection)
	api.Patch("/servers/:id/settings", onboardingHandler.UpdateServerSettings)
	api.Get("/servers/:id/command-logs", onboardingHandler.GetCommandLogs)

	// SSH command execution endpoints
	sshHandler := handlers.NewSSHHandler(dbService)
	api.Post("/servers/:id/ssh/execute", sshHandler.ExecuteCommand)
	api.Post("/servers/:id/scan", sshHandler.ScanDiskUsage)

	// SSH access helper (local terminal shortcuts)
	sshAccessHandler := handlers.NewSSHAccessHandler(dbService)
	api.Get("/servers/:id/ssh-command", sshAccessHandler.GetSSHCommand)
	api.Get("/servers/:id/ssh-test", sshAccessHandler.QuickTestConnection)

	// Cleanup management
	cleanupHandler := handlers.NewCleanupHandler(dbService)
	api.Post("/cleanup-actions/:id/restore", cleanupHandler.RestoreFiles)

	// Cleanup recipes (kategori B)
	recipeHandler := handlers.NewRecipeHandler(dbService)
	api.Get("/servers/:id/cleanup-recipes", recipeHandler.ListRecipes)
	api.Get("/servers/:id/cleanup-recipes/:recipe_id/estimate", recipeHandler.EstimateRecipe)
	api.Post("/servers/:id/cleanup-recipes/:recipe_id/execute", recipeHandler.ExecuteRecipe)

	// API Keys endpoints
	apiKeyHandler := handlers.NewApiKeyHandler(dbService)
	api.Get("/api-keys", apiKeyHandler.GetApiKeys)
	api.Delete("/api-keys/:id", apiKeyHandler.DeleteApiKey)

	// Settings endpoints
	settingsHandler := handlers.NewSettingsHandler(dbService.GetDB())
	api.Post("/settings/test-telegram", settingsHandler.TestTelegram)
	api.Post("/settings/telegram/connect", settingsHandler.StartTelegramConnection)
	api.Get("/settings/telegram/status", settingsHandler.GetTelegramConnectionStatus)
	api.Post("/settings/notification", settingsHandler.SaveNotificationSettings)
	api.Get("/settings/notification", settingsHandler.GetNotificationSettings)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001" // 3000 dipakai dashboard Next.js
	}

	log.Printf("Starting server on port %s", port)
	log.Fatal(app.Listen(":" + port))
}