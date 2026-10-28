package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/archon/backend/internal/db"
	"github.com/archon/backend/internal/recipes"
	"github.com/archon/backend/internal/ssh"
)

type RecipeHandler struct {
	db *db.DatabaseService
}

func NewRecipeHandler(database *db.DatabaseService) *RecipeHandler {
	return &RecipeHandler{db: database}
}

// ListRecipes returns all available cleanup recipes
func (h *RecipeHandler) ListRecipes(c *fiber.Ctx) error {
	serverID := c.Params("id")

	// Check server exists and cleanup_enabled
	var cleanupEnabled bool
	err := h.db.DB.QueryRow(`SELECT cleanup_enabled FROM servers WHERE id = $1`, serverID).Scan(&cleanupEnabled)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Server not found"})
	}

	// Return recipes with enabled status
	recipeList := make([]fiber.Map, 0, len(recipes.AllRecipes))
	for _, recipe := range recipes.AllRecipes {
		recipeList = append(recipeList, fiber.Map{
			"id":           recipe.ID,
			"name":         recipe.Name,
			"description":  recipe.Description,
			"category":     recipe.Category,
			"safety_level": recipe.SafetyLevel,
			"reversible":   recipe.Reversible,
			"enabled":      cleanupEnabled, // recipe aktif hanya kalau cleanup_enabled
		})
	}

	return c.JSON(fiber.Map{
		"server_id":       serverID,
		"recipes":         recipeList,
		"cleanup_enabled": cleanupEnabled,
		"note":            "Kategori B: aksi tidak bisa dikembalikan. Dry-run wajib sebelum eksekusi.",
	})
}

// EstimateRecipe runs dry-run estimate for a recipe
func (h *RecipeHandler) EstimateRecipe(c *fiber.Ctx) error {
	serverID := c.Params("id")
	recipeID := c.Params("recipe_id")

	// Get recipe
	recipe, err := recipes.GetRecipeByID(recipeID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	// Get connection details
	var host, username, privateKeyEncrypted, hostFingerprint string
	var port int
	err = h.db.DB.QueryRow(`
		SELECT sc.host, sc.port, sc.username, sc.private_key_encrypted, sc.host_fingerprint
		FROM server_connections sc
		WHERE sc.server_id = $1`, serverID).Scan(&host, &port, &username, &privateKeyEncrypted, &hostFingerprint)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Server connection not found"})
	}

	// Create SSH executor
	config := &ssh.ConnectionConfig{
		Host:            host,
		Port:            port,
		Username:        username,
		PrivateKey:      []byte(privateKeyEncrypted),
		HostFingerprint: hostFingerprint,
		Timeout:         30 * time.Second,
		MaxOutputBytes:  1024 * 1024,
	}

	executor, err := ssh.NewExecutor(config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create SSH executor"})
	}

	if err := executor.Connect(); err != nil {
		return c.Status(503).JSON(fiber.Map{"error": "SSH connection failed", "detail": err.Error()})
	}
	defer executor.Close()

	// Execute estimate command
	result, err := executor.Execute(recipe.EstimateCmd)
	if err != nil || result.ExitCode != 0 {
		return c.Status(500).JSON(fiber.Map{
			"error":  "Estimate command failed",
			"detail": result.Stderr,
		})
	}

	// Parse estimate (rough - output varies by command)
	estimateStr := strings.TrimSpace(result.Stdout)
	estimatedBytes, _ := strconv.ParseInt(estimateStr, 10, 64)

	return c.JSON(fiber.Map{
		"server_id":        serverID,
		"recipe_id":        recipeID,
		"recipe_name":      recipe.Name,
		"estimated_bytes":  estimatedBytes,
		"estimated_mb":     float64(estimatedBytes) / 1024 / 1024,
		"raw_output":       estimateStr,
		"note":             "Estimasi kasar. Jalankan dengan confirmed:true untuk eksekusi.",
	})
}

// ExecuteRecipe runs cleanup recipe (requires confirmed:true)
func (h *RecipeHandler) ExecuteRecipe(c *fiber.Ctx) error {
	serverID := c.Params("id")
	recipeID := c.Params("recipe_id")

	var req struct {
		Confirmed bool `json:"confirmed"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Prinsip #1: wajib confirmed:true
	if !req.Confirmed {
		return c.Status(400).JSON(fiber.Map{
			"error": "confirmed:true required",
			"note":  "Kategori B tidak bisa dikembalikan. Pastikan sudah dry-run dan review estimasi.",
		})
	}

	// Check cleanup_enabled (prinsip #9: default read-only)
	var cleanupEnabled bool
	err := h.db.DB.QueryRow(`SELECT cleanup_enabled FROM servers WHERE id = $1`, serverID).Scan(&cleanupEnabled)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Server not found"})
	}
	if !cleanupEnabled {
		return c.Status(403).JSON(fiber.Map{
			"error": "Cleanup not enabled for this server",
			"note":  "Aktifkan cleanup di settings server terlebih dahulu.",
		})
	}

	// Get recipe
	recipe, err := recipes.GetRecipeByID(recipeID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": err.Error()})
	}

	// Get connection details
	var host, username, privateKeyEncrypted, hostFingerprint string
	var port int
	err = h.db.DB.QueryRow(`
		SELECT sc.host, sc.port, sc.username, sc.private_key_encrypted, sc.host_fingerprint
		FROM server_connections sc
		WHERE sc.server_id = $1`, serverID).Scan(&host, &port, &username, &privateKeyEncrypted, &hostFingerprint)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Server connection not found"})
	}

	// Create SSH executor
	config := &ssh.ConnectionConfig{
		Host:            host,
		Port:            port,
		Username:        username,
		PrivateKey:      []byte(privateKeyEncrypted),
		HostFingerprint: hostFingerprint,
		Timeout:         60 * time.Second,
		MaxOutputBytes:  1024 * 1024,
	}

	executor, err := ssh.NewExecutor(config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create SSH executor"})
	}

	if err := executor.Connect(); err != nil {
		return c.Status(503).JSON(fiber.Map{"error": "SSH connection failed", "detail": err.Error()})
	}
	defer executor.Close()

	// Execute cleanup command
	startTime := time.Now()
	result, err := executor.Execute(recipe.Command)
	durationMs := time.Since(startTime).Milliseconds()

	if err != nil || result.ExitCode != 0 {
		return c.Status(500).JSON(fiber.Map{
			"error":      "Cleanup command failed",
			"detail":     result.Stderr,
			"exit_code":  result.ExitCode,
		})
	}

	// Log cleanup action to audit trail
	var actionID string
	err = h.db.DB.QueryRow(`
		INSERT INTO cleanup_actions (server_id, category, recipe_id, triggered_by, executed_at)
		VALUES ($1, 'recipe', $2, 'manual', $3)
		RETURNING id`, serverID, recipeID, startTime).Scan(&actionID)
	if err != nil {
		fmt.Printf("Warning: failed to log cleanup action: %v\n", err)
	}

	// Log SSH command
	auditLogger := ssh.NewAuditLogger(h.db.DB)
	auditLogger.LogExecution(context.Background(), &ssh.CommandLog{
		ServerID:        serverID,
		CommandTemplate: fmt.Sprintf("recipe:%s", recipeID),
		Parameters:      map[string]interface{}{"recipe_id": recipeID},
		ExecutedCommand: recipe.Command,
		ExitCode:        result.ExitCode,
		DurationMs:      durationMs,
		TriggeredBy:     "cleanup_recipe",
		ExecutedAt:      startTime,
	})

	return c.JSON(fiber.Map{
		"server_id":   serverID,
		"recipe_id":   recipeID,
		"recipe_name": recipe.Name,
		"action_id":   actionID,
		"status":      "success",
		"stdout":      result.Stdout,
		"duration_ms": durationMs,
		"note":        "Cleanup berhasil. Aksi tidak bisa dikembalikan (kategori B).",
	})
}
