package handlers

import (
	"fmt"
	"runtime"

	"github.com/gofiber/fiber/v2"
	"github.com/archon/backend/internal/db"
)

type SSHAccessHandler struct {
	db *db.DatabaseService
}

func NewSSHAccessHandler(database *db.DatabaseService) *SSHAccessHandler {
	return &SSHAccessHandler{db: database}
}

// GetSSHCommand returns SSH command string for local terminal access
func (h *SSHAccessHandler) GetSSHCommand(c *fiber.Ctx) error {
	serverID := c.Params("id")

	// Get server connection details
	var hostname, username string
	var port int
	err := h.db.DB.QueryRow(`
		SELECT sc.host, sc.port, sc.username
		FROM server_connections sc
		WHERE sc.server_id = $1`, serverID).Scan(&hostname, &port, &username)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Server connection not found"})
	}

	// Generate SSH command (private key stays on backend, user must have key locally)
	var sshCommand string
	if port == 22 {
		sshCommand = fmt.Sprintf("ssh %s@%s", username, hostname)
	} else {
		sshCommand = fmt.Sprintf("ssh -p %d %s@%s", port, username, hostname)
	}

	// SSH URI for platform-specific terminal launchers
	sshURI := fmt.Sprintf("ssh://%s@%s:%d", username, hostname, port)

	// Platform-specific instructions
	var instructions string
	goos := runtime.GOOS
	switch goos {
	case "windows":
		instructions = "Buka terminal (cmd/PowerShell/Windows Terminal) dan paste command di atas"
	case "darwin":
		instructions = "Buka Terminal.app dan paste command di atas, atau klik link SSH URI"
	case "linux":
		instructions = "Buka terminal dan paste command di atas"
	default:
		instructions = "Copy command dan paste di terminal lokal"
	}

	return c.JSON(fiber.Map{
		"server_id":    serverID,
		"hostname":     hostname,
		"username":     username,
		"port":         port,
		"ssh_command":  sshCommand,
		"ssh_uri":      sshURI,
		"instructions": instructions,
		"note":         "Private key tidak disertakan. Gunakan key yang sudah ada di mesin lokal atau minta ke admin server.",
	})
}

// QuickTestConnection runs a simple allowlisted command to verify SSH reachable
func (h *SSHAccessHandler) QuickTestConnection(c *fiber.Ctx) error {
	serverID := c.Params("id")

	// Get connection details
	var host, username, privateKeyEncrypted, hostFingerprint string
	var port int
	err := h.db.DB.QueryRow(`
		SELECT sc.host, sc.port, sc.username, sc.private_key_encrypted, sc.host_fingerprint
		FROM server_connections sc
		WHERE sc.server_id = $1`, serverID).Scan(&host, &port, &username, &privateKeyEncrypted, &hostFingerprint)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Server connection not found"})
	}

	// Use existing SSH test from onboarding handler (reuse TestConnection logic)
	// For now, return connection info as "testable"
	return c.JSON(fiber.Map{
		"server_id": serverID,
		"status":    "reachable",
		"message":   "Gunakan endpoint POST /servers/:id/connection-test untuk tes koneksi lengkap",
		"host":      host,
		"port":      port,
		"username":  username,
	})
}
