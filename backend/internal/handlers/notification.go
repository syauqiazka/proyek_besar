package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

type NotificationHandler struct {
	// TODO: Tambahkan WhatsApp service (Wablas/Fonnte) + DB
}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

type WhatsAppWebhookPayload struct {
	From    string `json:"from"`
	Message string `json:"message"`
	// Struktur bisa berbeda tergantung provider (Wablas/Fonnte)
}

func (h *NotificationHandler) WhatsAppWebhook(c *fiber.Ctx) error {
	var payload WhatsAppWebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid webhook payload"})
	}

	// Parse command dari message
	command := strings.TrimSpace(strings.ToLower(payload.Message))

	switch command {
	case "/cek-storage", "/storage":
		return h.handleStorageCheck(c, payload.From)
	case "/cek-backup", "/backup":
		return h.handleBackupCheck(c, payload.From)
	case "/help":
		return h.handleHelp(c, payload.From)
	default:
		return h.handleUnknownCommand(c, payload.From)
	}
}

func (h *NotificationHandler) handleStorageCheck(c *fiber.Ctx, phoneNumber string) error {
	// TODO: lookup tenant by phone, fetch metrics, kirim via Wablas/Fonnte
	responseMessage := `📊 *Status Storage*

Server: web-01
Kapasitas: 78.5% terpakai
Prediksi penuh: 15 hari lagi

Server: db-01
Kapasitas: 45.2% terpakai
Prediksi penuh: 120+ hari

✅ Semua server normal`

	return c.JSON(fiber.Map{
		"message":  "Storage check processed",
		"response": responseMessage,
	})
}

func (h *NotificationHandler) handleBackupCheck(c *fiber.Ctx, phoneNumber string) error {
	responseMessage := `💾 *Status Backup*

Last backup: 2 jam yang lalu
Status: ✅ Success
Size: 2.4 GB

Backup terakhir berjalan normal.`

	return c.JSON(fiber.Map{
		"message":  "Backup check processed",
		"response": responseMessage,
	})
}

func (h *NotificationHandler) handleHelp(c *fiber.Ctx, phoneNumber string) error {
	responseMessage := `🤖 *Hermes Bot Commands*

/cek-storage - Cek kapasitas storage
/cek-backup - Cek status backup
/help - Tampilkan menu ini`

	return c.JSON(fiber.Map{
		"message":  "Help sent",
		"response": responseMessage,
	})
}

func (h *NotificationHandler) handleUnknownCommand(c *fiber.Ctx, phoneNumber string) error {
	responseMessage := `❌ Command tidak dikenal.
Ketik /help untuk melihat daftar command.`

	return c.JSON(fiber.Map{
		"message":  "Unknown command",
		"response": responseMessage,
	})
}