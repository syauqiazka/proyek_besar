package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hermes/backend/internal/db"
)

type ApiKeyHandler struct {
	db *db.DatabaseService
}

func NewApiKeyHandler(database *db.DatabaseService) *ApiKeyHandler {
	return &ApiKeyHandler{db: database}
}

// GetApiKeys — daftar semua server + API key (untuk dashboard)
func (h *ApiKeyHandler) GetApiKeys(c *fiber.Ctx) error {
	rows, err := h.db.DB.Query(`
		SELECT s.id, s.hostname, k.api_key, s.created_at
		FROM servers s
		JOIN agent_api_keys k ON k.server_id = s.id
		ORDER BY s.created_at DESC`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database error"})
	}
	defer rows.Close()

	type apiKeyRow struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Key       string `json:"key"`
		CreatedAt string `json:"created_at"`
	}

	var keys []apiKeyRow
	for rows.Next() {
		var r apiKeyRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Key, &r.CreatedAt); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Scan error"})
		}
		keys = append(keys, r)
	}

	return c.JSON(fiber.Map{"api_keys": keys, "total": len(keys)})
}

// DeleteApiKey — menghapus server dan API key miliknya
func (h *ApiKeyHandler) DeleteApiKey(c *fiber.Ctx) error {
	serverID := c.Params("id")
	if serverID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Server ID is required"})
	}

	var hostname string
	_ = h.db.DB.QueryRow("SELECT hostname FROM servers WHERE id = $1", serverID).Scan(&hostname)

	// Hapus server (ON DELETE CASCADE akan otomatis menghapus agent_api_keys dan metriks)
	_, err := h.db.DB.Exec("DELETE FROM servers WHERE id = $1", serverID)
	if err != nil {
		// Fallback: hapus dari agent_api_keys jika ada issue pada foreign keys
		_, err2 := h.db.DB.Exec("DELETE FROM agent_api_keys WHERE server_id = $1", serverID)
		if err2 != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to delete API key", "detail": err.Error()})
		}
	}

	return c.JSON(fiber.Map{
		"message":   "API key / server deleted successfully",
		"server_id": serverID,
		"hostname":  hostname,
	})
}
