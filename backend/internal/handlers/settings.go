package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hermes/backend/internal/notify"
)

type SettingsHandler struct {
	db              *sql.DB
	listenerContext context.Context
	listenerCancel  context.CancelFunc
}

func NewSettingsHandler(db *sql.DB) *SettingsHandler {
	return &SettingsHandler{db: db}
}

func (h *SettingsHandler) TestTelegram(c *fiber.Ctx) error {
	var payload struct {
		Token  string `json:"token"`
		ChatID string `json:"chat_id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	// Step 1: verify token via getMe
	getMeURL := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", payload.Token)
	resp, err := http.Get(getMeURL)
	if err != nil || resp.StatusCode != 200 {
		return c.Status(400).JSON(fiber.Map{"error": "token tidak valid: Unauthorized"})
	}
	defer resp.Body.Close()

	var me struct {
		Ok      bool `json:"ok"`
		Result  struct {
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil || !me.Ok {
		return c.Status(400).JSON(fiber.Map{"error": "token tidak valid: API response"})
	}

	// Step 2: send actual test message
	sendURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", payload.Token)
	msgBody, _ := json.Marshal(map[string]string{
		"chat_id": payload.ChatID,
		"text":    "✅ Test koneksi berhasil! Hermes Storage Monitor siap kirim alert ke chat ini.",
	})
	sendResp, err := http.Post(sendURL, "application/json", bytes.NewReader(msgBody))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("gagal kirim pesan test: %v", err)})
	}
	defer sendResp.Body.Close()

	sendBody, _ := io.ReadAll(sendResp.Body)
	var sendResult struct {
		Ok          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(sendBody, &sendResult); err != nil || !sendResult.Ok {
		return c.Status(400).JSON(fiber.Map{"error": fmt.Sprintf("gagal kirim pesan test: %s", sendResult.Description)})
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"bot_name":     me.Result.FirstName,
		"bot_username": me.Result.Username,
		"message":      fmt.Sprintf("✓ Bot %s (@%s) terverifikasi dan test message berhasil dikirim ke chat %s", me.Result.FirstName, me.Result.Username, payload.ChatID),
	})
}

func (h *SettingsHandler) StartTelegramConnection(c *fiber.Ctx) error {
	var payload struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
	}

	// Cancel previous listener if exists
	if h.listenerCancel != nil {
		h.listenerCancel()
	}

	// Start new listener
	h.listenerContext, h.listenerCancel = context.WithTimeout(context.Background(), 5*time.Minute)
	go notify.StartTelegramListener(h.listenerContext, payload.Token, h.db)

	return c.JSON(fiber.Map{"success": true, "message": "Listening untuk pesan /hubungkan"})
}

func (h *SettingsHandler) GetTelegramConnectionStatus(c *fiber.Ctx) error {
	var value string
	row := h.db.QueryRowContext(c.UserContext(), "SELECT value FROM settings WHERE key = $1", "telegram_config")
	if err := row.Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(fiber.Map{"connected": false})
		}
		return c.Status(500).JSON(fiber.Map{"error": "gagal cek status"})
	}

	var config map[string]string
	if err := json.Unmarshal([]byte(value), &config); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal parse config"})
	}

	if config["chat_id"] != "" {
		return c.JSON(fiber.Map{
			"connected":  true,
			"chat_id":    config["chat_id"],
			"first_name": config["first_name"],
		})
	}

	return c.JSON(fiber.Map{"connected": false})
}

func (h *SettingsHandler) SaveNotificationSettings(c *fiber.Ctx) error {
	var payload struct {
		TelegramBotToken string `json:"telegram_bot_token"`
		TelegramChatID   string `json:"telegram_chat_id"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "payload tidak valid"})
	}
	// ponytail: encryption skipped - store plaintext (TODO: encrypt at rest)
	sqlQuery := `INSERT INTO settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP`
	_, err := h.db.ExecContext(c.UserContext(), sqlQuery, "telegram_config", fmt.Sprintf(`{"token":"%s","chat_id":"%s"}`, payload.TelegramBotToken, payload.TelegramChatID))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal simpan pengaturan telegram"})
	}
	return c.JSON(fiber.Map{"success": true, "message": "Pengaturan tersimpan"})
}

func (h *SettingsHandler) GetNotificationSettings(c *fiber.Ctx) error {
	var value string
	row := h.db.QueryRowContext(c.UserContext(), "SELECT value FROM settings WHERE key = $1", "telegram_config")
	if err := row.Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(fiber.Map{"telegram_config": nil})
		}
		return c.Status(500).JSON(fiber.Map{"error": "gagal load pengaturan"})
	}
	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "gagal parse pengaturan"})
	}
	return c.JSON(fiber.Map{"telegram_config": settings})
}
