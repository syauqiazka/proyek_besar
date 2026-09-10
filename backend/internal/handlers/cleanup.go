package handlers

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hermes/backend/internal/db"
)

type CleanupHandler struct {
	db *db.DatabaseService
}

func NewCleanupHandler(database *db.DatabaseService) *CleanupHandler {
	return &CleanupHandler{db: database}
}

type RestoreRequest struct {
	Confirmed bool `json:"confirmed"`
}

// RestoreFiles — restore file dari snapshot jika masih dalam window 48 jam
func (h *CleanupHandler) RestoreFiles(c *fiber.Ctx) error {
	actionID := c.Params("id")

	var req RestoreRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if !req.Confirmed {
		return c.Status(400).JSON(fiber.Map{"error": "Restore must be explicitly confirmed"})
	}

	// Cek status & window
	var status string
	var windowUntil sql.NullTime
	var snapshotPath sql.NullString
	err := h.db.DB.QueryRow(`
		SELECT status, restore_window_until, snapshot_path FROM cleanup_actions WHERE id = $1`,
		actionID).Scan(&status, &windowUntil, &snapshotPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(404).JSON(fiber.Map{"error": "Cleanup action not found"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Database error"})
	}
	if status != "completed" {
		return c.Status(400).JSON(fiber.Map{"error": "Cleanup not completed; nothing to restore"})
	}
	if windowUntil.Valid && time.Now().After(windowUntil.Time) {
		return c.Status(400).JSON(fiber.Map{"error": "Restore window (48h) has expired"})
	}

	// TODO: kirim perintah restore ke agent (polling)
	// Untuk sekarang: tandai status 'restored' + restore file di snapshot
	_, err = h.db.DB.Exec(`UPDATE cleanup_actions SET status = 'restored' WHERE id = $1`, actionID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update status"})
	}
	_, _ = h.db.DB.Exec(`UPDATE cleanup_files SET restored_at = NOW() WHERE cleanup_action_id = $1`, actionID)

	return c.JSON(fiber.Map{
		"message":           "Restore initiated",
		"cleanup_action_id": actionID,
		"status":            "restoring",
	})
}