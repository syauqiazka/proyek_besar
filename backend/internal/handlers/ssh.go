package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/archon/backend/internal/db"
	"github.com/archon/backend/internal/ssh"
)

type SSHHandler struct {
	db          *db.DatabaseService
	rateLimiter *ssh.RateLimiter
}

func NewSSHHandler(database *db.DatabaseService) *SSHHandler {
	return &SSHHandler{
		db:          database,
		rateLimiter: ssh.NewRateLimiter(10*time.Second, 2), // 10s interval, max 2 concurrent
	}
}

// ExecuteCommand executes an allowlisted SSH command on a server
func (h *SSHHandler) ExecuteCommand(c *fiber.Ctx) error {
	serverID := c.Params("id")

	var req struct {
		CommandTemplate string                 `json:"command_template"`
		Parameters      map[string]interface{} `json:"parameters"`
		TriggeredBy     string                 `json:"triggered_by"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Check rate limit
	release, err := h.rateLimiter.AllowWithRelease(serverID)
	if err != nil {
		return c.Status(429).JSON(fiber.Map{"error": err.Error()})
	}
	defer release()

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
		PrivateKey:      []byte(privateKeyEncrypted), // TODO: decrypt first
		HostFingerprint: hostFingerprint,
		Timeout:         30 * time.Second,
		MaxOutputBytes:  1024 * 1024,
	}

	executor, err := ssh.NewExecutor(config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create SSH executor"})
	}

	// Connect
	if err := executor.Connect(); err != nil {
		return c.Status(503).JSON(fiber.Map{"error": "SSH connection failed", "detail": err.Error()})
	}
	defer executor.Close()

	// Execute with audit logging
	auditLogger := ssh.NewAuditLogger(h.db.DB)
	result, err := ssh.ExecuteWithAudit(
		context.Background(),
		executor,
		auditLogger,
		serverID,
		req.CommandTemplate,
		req.Parameters,
		req.TriggeredBy,
	)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":      "Command execution failed",
			"detail":     err.Error(),
			"exit_code":  result.ExitCode,
			"stderr":     result.Stderr,
			"stdout":     result.Stdout,
			"duration_ms": result.DurationMs,
		})
	}

	return c.JSON(fiber.Map{
		"server_id":    serverID,
		"command":      result.Command,
		"exit_code":    result.ExitCode,
		"stdout":       result.Stdout,
		"stderr":       result.Stderr,
		"duration_ms":  result.DurationMs,
	})
}

// ScanDiskUsage scans disk usage and stores snapshot
func (h *SSHHandler) ScanDiskUsage(c *fiber.Ctx) error {
	serverID := c.Params("id")

	// Rate limit check
	release, err := h.rateLimiter.AllowWithRelease(serverID)
	if err != nil {
		return c.Status(429).JSON(fiber.Map{"error": err.Error()})
	}
	defer release()

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

	// Create executor
	config := &ssh.ConnectionConfig{
		Host:            host,
		Port:            port,
		Username:        username,
		PrivateKey:      []byte(privateKeyEncrypted),
		HostFingerprint: hostFingerprint,
		Timeout:         60 * time.Second, // longer timeout for disk scan
		MaxOutputBytes:  1024 * 1024,
	}

	executor, err := ssh.NewExecutor(config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create executor"})
	}

	if err := executor.Connect(); err != nil {
		return c.Status(503).JSON(fiber.Map{"error": "SSH connection failed", "detail": err.Error()})
	}
	defer executor.Close()

	// Execute df command to get disk usage
	auditLogger := ssh.NewAuditLogger(h.db.DB)
	result, err := ssh.ExecuteWithAudit(
		context.Background(),
		executor,
		auditLogger,
		serverID,
		"disk_usage",
		map[string]interface{}{"mount_path": "/"},
		"scan_job",
	)

	if err != nil || result.ExitCode != 0 {
		return c.Status(500).JSON(fiber.Map{"error": "Disk scan failed", "detail": result.Stderr})
	}

	// Parse df output (simplified - production should parse properly)
	// Example: Filesystem      Size  Used Avail Use% Mounted on
	//          /dev/sda1        50G   30G   18G  63% /
	
	// For now, store raw output and return success
	// TODO: Parse df output and insert into storage_snapshots

	// Update server last_seen_at
	h.db.DB.Exec(`UPDATE servers SET last_seen_at = NOW() WHERE id = $1`, serverID)

	return c.JSON(fiber.Map{
		"server_id":   serverID,
		"message":     "Disk scan completed",
		"raw_output":  result.Stdout,
		"duration_ms": result.DurationMs,
	})
}
