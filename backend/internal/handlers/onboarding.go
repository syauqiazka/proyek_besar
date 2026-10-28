package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/archon/backend/internal/db"
	"github.com/archon/backend/internal/onboarding"
	"github.com/archon/backend/internal/ssh"
)

type OnboardingHandler struct {
	db *db.DatabaseService
}

func NewOnboardingHandler(database *db.DatabaseService) *OnboardingHandler {
	return &OnboardingHandler{db: database}
}

type CreateServerRequest struct {
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

type CreateServerResponse struct {
	ServerID       string `json:"server_id"`
	Name           string `json:"name"`
	Hostname       string `json:"hostname"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	PublicKey      string `json:"public_key"`
	SetupScript    string `json:"setup_script"`
	NextSteps      string `json:"next_steps"`
}

// CreateServer generates keypair and returns setup script
func (h *OnboardingHandler) CreateServer(c *fiber.Ctx) error {
	var req CreateServerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Validate required fields
	if req.Name == "" || req.Hostname == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Name and hostname are required"})
	}
	if req.Port == 0 {
		req.Port = 22
	}
	if req.Username == "" {
		req.Username = "archon"
	}

	// Generate SSH keypair
	keyPair, err := onboarding.GenerateKeyPair(2048)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate keypair", "detail": err.Error()})
	}

	// Create server record
	var serverID string
	err = h.db.DB.QueryRow(`
		INSERT INTO servers (tenant_id, name, hostname, connection_mode, status)
		VALUES ($1, $2, $3, 'ssh', 'pending')
		RETURNING id`,
		req.TenantID, req.Name, req.Hostname).Scan(&serverID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create server", "detail": err.Error()})
	}

	// TODO: Encrypt private key before storing (prinsip keamanan #3)
	// For now, store as-is (DEVELOPMENT ONLY - MUST ENCRYPT IN PRODUCTION)
	privateKeyEncrypted := string(keyPair.PrivateKeyPEM)

	// Create server connection record
	_, err = h.db.DB.Exec(`
		INSERT INTO server_connections (server_id, host, port, username, private_key_encrypted, host_fingerprint)
		VALUES ($1, $2, $3, $4, $5, '')`,
		serverID, req.Hostname, req.Port, req.Username, privateKeyEncrypted)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to store connection", "detail": err.Error()})
	}

	// Generate setup script
	setupScript := onboarding.GenerateSetupScript(req.Name, string(keyPair.PublicKeySSH))

	return c.Status(201).JSON(CreateServerResponse{
		ServerID:    serverID,
		Name:        req.Name,
		Hostname:    req.Hostname,
		Port:        req.Port,
		Username:    req.Username,
		PublicKey:   string(keyPair.PublicKeySSH),
		SetupScript: setupScript,
		NextSteps:   "1. Copy setup script to server\n2. Run: bash setup.sh\n3. Click 'Test Connection' in dashboard",
	})
}

// TestConnection tests SSH connection and runs stack detection
func (h *OnboardingHandler) TestConnection(c *fiber.Ctx) error {
	serverID := c.Params("id")

	// Get connection details
	var host, username, privateKeyEncrypted string
	var port int
	err := h.db.DB.QueryRow(`
		SELECT sc.host, sc.port, sc.username, sc.private_key_encrypted
		FROM server_connections sc
		WHERE sc.server_id = $1`, serverID).Scan(&host, &port, &username, &privateKeyEncrypted)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Server connection not found"})
	}

	// Create SSH executor
	config := &ssh.ConnectionConfig{
		Host:            host,
		Port:            port,
		Username:        username,
		PrivateKey:      []byte(privateKeyEncrypted), // TODO: decrypt first
		Timeout:         30 * time.Second,
		MaxOutputBytes:  1024 * 1024,
	}

	executor, err := ssh.NewExecutor(config)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create SSH executor"})
	}

	// Connect
	if err := executor.Connect(); err != nil {
		// Update server status to error
		h.db.DB.Exec(`UPDATE servers SET status = 'error' WHERE id = $1`, serverID)
		h.db.DB.Exec(`UPDATE server_connections SET last_test_status = 'failed', last_test_error = $1, last_test_at = NOW() WHERE server_id = $2`, 
			err.Error(), serverID)
		
		return c.Status(503).JSON(fiber.Map{
			"error": "SSH connection failed",
			"detail": err.Error(),
		})
	}
	defer executor.Close()

	// Save host fingerprint
	fingerprint := executor.GetHostFingerprint()
	h.db.DB.Exec(`UPDATE server_connections SET host_fingerprint = $1 WHERE server_id = $2`, fingerprint, serverID)

	// Run stack detection
	detectionResult, err := executor.Execute(onboarding.DetectionScript())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Connection succeeded but detection script failed",
			"detail": err.Error(),
		})
	}

	// Parse detection output (simplified - real implementation should parse properly)
	stackDetected := map[string]interface{}{
		"raw_output": detectionResult.Stdout,
	}
	stackJSON, _ := json.Marshal(stackDetected)

	// Update server status to connected
	_, err = h.db.DB.Exec(`
		UPDATE servers 
		SET status = 'connected', last_seen_at = NOW(), stack_detected = $1
		WHERE id = $2`, stackJSON, serverID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update server status"})
	}

	// Update connection test result
	h.db.DB.Exec(`
		UPDATE server_connections 
		SET last_test_status = 'success', last_test_at = NOW(), last_test_error = NULL
		WHERE server_id = $1`, serverID)

	// Log SSH command (detection script)
	auditLogger := ssh.NewAuditLogger(h.db.DB)
	auditLogger.LogExecution(context.Background(), &ssh.CommandLog{
		ServerID:        serverID,
		CommandTemplate: "stack_detection",
		Parameters:      map[string]interface{}{},
		ExecutedCommand: "detection_script",
		ExitCode:        detectionResult.ExitCode,
		DurationMs:      detectionResult.DurationMs,
		TriggeredBy:     "onboarding",
		ExecutedAt:      time.Now(),
	})

	return c.JSON(fiber.Map{
		"message":         "Connection test successful",
		"server_id":       serverID,
		"status":          "connected",
		"host_fingerprint": fingerprint,
		"stack_detected":  stackDetected,
	})
}

// UpdateServerSettings enables/disables cleanup for a server
func (h *OnboardingHandler) UpdateServerSettings(c *fiber.Ctx) error {
	serverID := c.Params("id")

	var req struct {
		CleanupEnabled *bool `json:"cleanup_enabled"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if req.CleanupEnabled == nil {
		return c.Status(400).JSON(fiber.Map{"error": "cleanup_enabled is required"})
	}

	// Update cleanup_enabled (prinsip desain #9: default read-only)
	_, err := h.db.DB.Exec(`
		UPDATE servers 
		SET cleanup_enabled = $1, updated_at = NOW()
		WHERE id = $2`, *req.CleanupEnabled, serverID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to update settings"})
	}

	return c.JSON(fiber.Map{
		"message":         "Server settings updated",
		"server_id":       serverID,
		"cleanup_enabled": *req.CleanupEnabled,
	})
}

// GetCommandLogs returns SSH command audit trail for a server
func (h *OnboardingHandler) GetCommandLogs(c *fiber.Ctx) error {
	serverID := c.Params("id")
	limit := c.QueryInt("limit", 50)

	auditLogger := ssh.NewAuditLogger(h.db.DB)
	logs, err := auditLogger.GetRecentLogs(context.Background(), serverID, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch logs"})
	}

	return c.JSON(fiber.Map{
		"server_id": serverID,
		"logs":      logs,
		"total":     len(logs),
	})
}
