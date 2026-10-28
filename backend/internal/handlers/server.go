package handlers

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/archon/backend/internal/db"
)

type ServerHandler struct {
	db *db.DatabaseService
}

func NewServerHandler(database *db.DatabaseService) *ServerHandler {
	return &ServerHandler{db: database}
}

type PredictionResponse struct {
	ServerID      string  `json:"server_id"`
	CurrentUsed   float64 `json:"current_used_percent"`
	DaysUntilFull struct {
		Min          int `json:"min"`
		Max          int `json:"max"`
		BestEstimate int `json:"best_estimate"`
	} `json:"days_until_full"`
	Confidence   float64   `json:"confidence"`
	CalculatedAt time.Time `json:"calculated_at"`
}

type CleanupCandidate struct {
	FileID     string    `json:"file_id"`
	Path       string    `json:"path"`
	SizeBytes  int64     `json:"size_bytes"`
	LastAccess time.Time `json:"last_access"`
	Category   string    `json:"category"`
	Confidence float64   `json:"confidence"`
	SizeMB     float64   `json:"size_mb"`
}

type CleanupRequest struct {
	FileIDs   []string `json:"file_ids"`
	Confirmed bool     `json:"confirmed"`
	Reason    string   `json:"reason"`
}

// GetServers — daftar semua server + snapshot terakhir (untuk dashboard)
func (h *ServerHandler) GetServers(c *fiber.Ctx) error {
	rows, err := h.db.DB.Query(`
		SELECT s.id, s.name, s.hostname,
		       CASE
		           WHEN s.last_seen_at IS NULL THEN 'pending'
		           WHEN s.last_seen_at < NOW() - INTERVAL '5 minutes' THEN 'error'
		           ELSE s.status
		       END AS status,
		       s.last_seen_at,
		       COALESCE(ss.usage_percent, 0) AS used_percent,
		       COALESCE(ss.total_bytes, 0) AS total_bytes,
		       COALESCE(ss.used_bytes, 0) AS used_bytes,
		       COALESCE(ss.available_bytes, 0) AS available_bytes
		FROM servers s
		LEFT JOIN LATERAL (
			SELECT usage_percent, total_bytes, used_bytes, available_bytes
			FROM storage_snapshots
			WHERE server_id = s.id
			ORDER BY captured_at DESC LIMIT 1
		) ss ON TRUE
		ORDER BY s.created_at DESC`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database error", "detail": err.Error()})
	}
	defer rows.Close()

	type serverRow struct {
		ID             string     `json:"id"`
		Name           string     `json:"name"`
		Hostname       string     `json:"hostname"`
		Status         string     `json:"status"`
		LastSeenAt     *time.Time `json:"last_seen_at"`
		UsedPercent    float64    `json:"used_percent"`
		TotalBytes     int64      `json:"total_bytes"`
		UsedBytes      int64      `json:"used_bytes"`
		AvailableBytes int64      `json:"available_bytes"`
	}

	var servers []serverRow
	for rows.Next() {
		var s serverRow
		var lastSeen sql.NullTime
		if err := rows.Scan(&s.ID, &s.Name, &s.Hostname, &s.Status, &lastSeen,
			&s.UsedPercent, &s.TotalBytes, &s.UsedBytes, &s.AvailableBytes); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Scan error"})
		}
		if lastSeen.Valid {
			s.LastSeenAt = &lastSeen.Time
		}
		servers = append(servers, s)
	}

	return c.JSON(fiber.Map{"servers": servers, "total": len(servers)})
}

// GetPrediction — hitung sisa hari ke penuh dari snapshot terakhir
func (h *ServerHandler) GetPrediction(c *fiber.Ctx) error {
	serverID := c.Params("id")

	// Ambil 2 snapshot terakhir untuk hitung growth rate
	rows, err := h.db.DB.Query(`
		SELECT total_bytes, used_bytes, captured_at FROM storage_snapshots
		WHERE server_id = $1 ORDER BY captured_at DESC LIMIT 2`, serverID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database error"})
	}
	defer rows.Close()

	type snap struct {
		total, used uint64
		ts          time.Time
	}
	var snaps []snap
	for rows.Next() {
		var s snap
		if err := rows.Scan(&s.total, &s.used, &s.ts); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Scan error"})
		}
		snaps = append(snaps, s)
	}
	if len(snaps) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "No metrics for this server"})
	}

	latest := snaps[0]
	currentUsedPct := float64(latest.used) / float64(latest.total) * 100

	daysUntil := 999
	confidence := 0.0
	if len(snaps) == 2 {
		prev := snaps[1]
		daysDiff := latest.ts.Sub(prev.ts).Hours() / 24
		if daysDiff > 0 {
			growthPerDay := float64(latest.used-prev.used) / daysDiff
			usedPctPast := float64(prev.used) / float64(prev.total) * 100
			if growthPerDay > 0 {
				available := float64(latest.total - latest.used)
				daysUntil = int(available / growthPerDay)
			}
			// Confidence: naik jika selisih usage antar snapshot masuk akal (0-100 poin)
			diff := currentUsedPct - usedPctPast
			if diff >= 0 && diff <= 50 {
				confidence = 0.8
			} else {
				confidence = 0.4
			}
		}
	}

	// Range: ±10% dari best estimate
	spread := daysUntil / 10
	if spread < 1 {
		spread = 1
	}
	minDays := daysUntil - spread
	if minDays < 0 {
		minDays = 0
	}

	return c.JSON(PredictionResponse{
		ServerID:      serverID,
		CurrentUsed:   currentUsedPct,
		DaysUntilFull: struct {
			Min          int `json:"min"`
			Max          int `json:"max"`
			BestEstimate int `json:"best_estimate"`
		}{Min: minDays, Max: daysUntil + spread, BestEstimate: daysUntil},
		Confidence:   confidence,
		CalculatedAt: time.Now(),
	})
}

// GetCleanupCandidates — ambil file_candidates dari DB, urut ukuran besar dulu
func (h *ServerHandler) GetCleanupCandidates(c *fiber.Ctx) error {
	serverID := c.Params("id")

	rows, err := h.db.DB.Query(`
		SELECT id, file_path, size_bytes, category, confidence, detected_at
		FROM file_candidates
		WHERE server_id = $1 AND status = 'pending'
		ORDER BY size_bytes DESC LIMIT 50`, serverID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database error"})
	}
	defer rows.Close()

	var candidates []CleanupCandidate
	for rows.Next() {
		var f CleanupCandidate
		var detectedAt time.Time
		if err := rows.Scan(&f.FileID, &f.Path, &f.SizeBytes, &f.Category, &f.Confidence, &detectedAt); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Scan error"})
		}
		f.LastAccess = detectedAt // use detected_at as placeholder
		f.SizeMB = float64(f.SizeBytes) / (1024 * 1024)
		candidates = append(candidates, f)
	}

	return c.JSON(fiber.Map{
		"server_id":  serverID,
		"candidates": candidates,
		"total":      len(candidates),
	})
}

// TriggerCleanup — wajib confirmed:true; simpan cleanup_action (audit trail)
func (h *ServerHandler) TriggerCleanup(c *fiber.Ctx) error {
	serverID := c.Params("id")

	var req CleanupRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// CRITICAL SAFETY CHECK (prinsip desain #1)
	if !req.Confirmed {
		return c.Status(400).JSON(fiber.Map{
			"error": "Cleanup must be explicitly confirmed with 'confirmed: true'",
		})
	}
	if len(req.FileIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "No files specified for cleanup"})
	}

	// Check if cleanup enabled for this server (prinsip desain #9)
	var cleanupEnabled bool
	err := h.db.DB.QueryRow(`SELECT cleanup_enabled FROM servers WHERE id = $1`, serverID).Scan(&cleanupEnabled)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Server not found"})
	}
	if !cleanupEnabled {
		return c.Status(403).JSON(fiber.Map{
			"error": "Cleanup not enabled for this server. Enable it in server settings first.",
		})
	}

	// Pastikan setiap file milik server ini
	var totalBytes int64
	validFiles := []string{}
	for _, fid := range req.FileIDs {
		var path string
		var size int64
		err := h.db.DB.QueryRow(`
			SELECT file_path, size_bytes FROM file_candidates 
			WHERE id = $1 AND server_id = $2 AND status = 'pending'`,
			fid, serverID).Scan(&path, &size)
		if err == nil {
			validFiles = append(validFiles, fid)
			totalBytes += size
		}
	}
	if len(validFiles) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "No valid file candidates for this server"})
	}

	// Buat cleanup_action (kategori A: file_candidate)
	var actionID string
	err = h.db.DB.QueryRow(`
		INSERT INTO cleanup_actions 
			(server_id, category, files_deleted, bytes_freed, confirmed, dry_run, details)
		VALUES ($1, 'file_candidate', $2, $3, TRUE, FALSE, $4)
		RETURNING id`, 
		serverID, len(validFiles), totalBytes, 
		map[string]interface{}{"file_ids": validFiles, "reason": req.Reason}).Scan(&actionID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create cleanup action", "detail": err.Error()})
	}

	// Update file_candidates status to approved
	_, err = h.db.DB.Exec(`
		UPDATE file_candidates 
		SET status = 'approved' 
		WHERE id = ANY($1) AND server_id = $2`, 
		validFiles, serverID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update candidates"})
	}

	return c.Status(202).JSON(fiber.Map{
		"message":           "Cleanup request accepted",
		"server_id":         serverID,
		"cleanup_action_id": actionID,
		"file_count":        len(validFiles),
		"bytes_to_free":     totalBytes,
		"status":            "pending",
		"note":              "Files will be moved to snapshot, not permanently deleted (48h restore window)",
	})
}