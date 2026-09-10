package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hermes/backend/internal/db"
)

type AgentHandler struct {
	db *db.DatabaseService
}

func NewAgentHandler(database *db.DatabaseService) *AgentHandler {
	return &AgentHandler{db: database}
}

type RegisterRequest struct {
	Hostname  string `json:"hostname"`
	IPAddress string `json:"ip_address"`
	TenantID  string `json:"tenant_id"`
}

type MetricsRequest struct {
	Metrics []struct {
		Path           string    `json:"path"`
		TotalBytes     uint64    `json:"total_bytes"`
		UsedBytes      uint64    `json:"used_bytes"`
		AvailableBytes uint64    `json:"available_bytes"`
		UsedPercent    float64   `json:"used_percent"`
		Timestamp      time.Time `json:"timestamp"`
	} `json:"metrics"`
}

type FileCandidatesRequest struct {
	Candidates []struct {
		Path         string    `json:"path"`
		SizeBytes    int64     `json:"size_bytes"`
		LastAccess   time.Time `json:"last_access"`
		LastModified time.Time `json:"last_modified"`
		Category     string    `json:"category"`
		Confidence   float64   `json:"confidence"`
	} `json:"candidates"`
}

type BackupStatusRequest struct {
	BackupName string    `json:"backup_name"`
	Status     string    `json:"status"`
	SizeBytes  int64     `json:"size_bytes"`
	CheckedAt  time.Time `json:"checked_at"`
}

// Register — buat server + API key baru di Supabase
func (h *AgentHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.Hostname == "" || req.TenantID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "hostname and tenant_id are required"})
	}

	// Validasi tenant ada
	var tenantExists bool
	err := h.db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)", req.TenantID).Scan(&tenantExists)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database error"})
	}
	if !tenantExists {
		return c.Status(404).JSON(fiber.Map{"error": "tenant_id not found"})
	}

	// Insert server — pakai ON CONFLICT supaya idempotent (hostname unik per tenant).
	// ip_address boleh kosong (agent belum tentu kirim) — maka NULL, bukan ''::inet.
	var serverID string
	if req.IPAddress != "" {
		err = h.db.DB.QueryRow(`
			INSERT INTO servers (tenant_id, hostname, ip_address, os_type, status, last_seen_at)
			VALUES ($1, $2, $3::inet, 'linux', 'active', NOW())
			ON CONFLICT (tenant_id, hostname) DO UPDATE SET last_seen_at = NOW()
			RETURNING id`, req.TenantID, req.Hostname, req.IPAddress).Scan(&serverID)
	} else {
		err = h.db.DB.QueryRow(`
			INSERT INTO servers (tenant_id, hostname, os_type, status, last_seen_at)
			VALUES ($1, $2, 'linux', 'active', NOW())
			ON CONFLICT (tenant_id, hostname) DO UPDATE SET last_seen_at = NOW()
			RETURNING id`, req.TenantID, req.Hostname).Scan(&serverID)
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to register server", "detail": err.Error()})
	}

	// Generate API key random 32 hex
	keyBytes := make([]byte, 32)
	_, _ = rand.Read(keyBytes)
	apiKey := "hermes_" + hex.EncodeToString(keyBytes)

	_, err = h.db.DB.Exec(`
		INSERT INTO agent_api_keys (server_id, api_key)
		VALUES ($1, $2)`, serverID, apiKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create API key", "detail": err.Error()})
	}

	return c.Status(201).JSON(fiber.Map{
		"server_id": serverID,
		"api_key":   apiKey,
		"message":   "Agent registered successfully",
	})
}

// authServer — validasi X-API-Key milik server_id
func (h *AgentHandler) authServer(c *fiber.Ctx) (string, error) {
	serverID := c.Params("server_id")
	apiKey := c.Get("X-API-Key")
	if apiKey == "" {
		return "", fiber.ErrUnauthorized
	}
	var count int
	err := h.db.DB.QueryRow(`
		SELECT COUNT(*) FROM agent_api_keys
		WHERE server_id = $1 AND api_key = $2 AND revoked_at IS NULL`,
		serverID, apiKey).Scan(&count)
	if err != nil || count == 0 {
		return "", fiber.ErrUnauthorized
	}
	return serverID, nil
}

// ReceiveMetrics — simpan snapshot per mount ke storage_snapshots
func (h *AgentHandler) ReceiveMetrics(c *fiber.Ctx) error {
	serverID, err := h.authServer(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req MetricsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	for _, m := range req.Metrics {
		_, err := h.db.DB.Exec(`
			INSERT INTO storage_snapshots (server_id, mount_path, total_bytes, used_bytes, available_bytes, used_percent, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			serverID, m.Path, m.TotalBytes, m.UsedBytes, m.AvailableBytes, m.UsedPercent, m.Timestamp)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to store metrics", "detail": err.Error()})
		}
	}

	// Update last_seen_at
	_, _ = h.db.DB.Exec(`UPDATE servers SET last_seen_at = NOW() WHERE id = $1`, serverID)

	return c.JSON(fiber.Map{
		"message":     "Metrics received",
		"server_id":   serverID,
		"count":       len(req.Metrics),
		"received_at": time.Now(),
	})
}

// ReceiveFileCandidates — simpan kandidat cleanup ke file_candidates
func (h *AgentHandler) ReceiveFileCandidates(c *fiber.Ctx) error {
	serverID, err := h.authServer(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req FileCandidatesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	for _, f := range req.Candidates {
		_, err := h.db.DB.Exec(`
			INSERT INTO file_candidates (server_id, path, size_bytes, last_access, last_modified, category, confidence)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			serverID, f.Path, f.SizeBytes, f.LastAccess, f.LastModified, f.Category, f.Confidence)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to store candidates", "detail": err.Error()})
		}
	}

	return c.JSON(fiber.Map{
		"message":     "File candidates received",
		"server_id":   serverID,
		"count":       len(req.Candidates),
		"received_at": time.Now(),
	})
}

// ReceiveBackupStatus — upsert backup + tambah verifikasi (Fase 1: status check)
func (h *AgentHandler) ReceiveBackupStatus(c *fiber.Ctx) error {
	serverID, err := h.authServer(c)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req BackupStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	var backupID string
	err = h.db.DB.QueryRow(`
		INSERT INTO backups (server_id, backup_name, backup_type, size_bytes, status, started_at, completed_at)
		VALUES ($1, $2, 'full', $3, $4, $5, $5)
		ON CONFLICT DO NOTHING
		RETURNING id`, serverID, req.BackupName, req.SizeBytes, req.Status, req.CheckedAt).Scan(&backupID)
	if err == sql.ErrNoRows {
		// Ambil backup yang sudah ada
		err = h.db.DB.QueryRow(`
			SELECT id FROM backups WHERE server_id = $1 AND backup_name = $2 LIMIT 1`,
			serverID, req.BackupName).Scan(&backupID)
	}
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to store backup", "detail": err.Error()})
	}

	// Backup verification (status check)
	verifyStatus := "success"
	if req.Status != "success" {
		verifyStatus = "failed"
	}
	_, err = h.db.DB.Exec(`
		INSERT INTO backup_verifications (backup_id, verification_type, status, notes, checked_at)
		VALUES ($1, 'status_check', $2, $3, $4)`,
		backupID, verifyStatus, req.Status, req.CheckedAt)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to store verification", "detail": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message":     "Backup status received",
		"server_id":   serverID,
		"backup_name": req.BackupName,
		"status":      req.Status,
	})
}