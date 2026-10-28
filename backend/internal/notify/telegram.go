package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

type TelegramResponse struct {
	Ok     bool   `json:"ok"`
	Result struct {
		MessageID int `json:"message_id"`
	} `json:"result"`
	Description string `json:"description"`
}

type TelegramBotInfo struct {
	Ok     bool   `json:"ok"`
	Result struct {
		ID       int64  `json:"id"`
		IsBot    bool   `json:"is_bot"`
		Username string `json:"username"`
		FirstName string `json:"first_name"`
	} `json:"result"`
	Description string `json:"description"`
}

type Telegram struct {
	Token string
}

func NewTelegram(token string) *Telegram {
	return &Telegram{Token: token}
}

// VerifyToken checks if the Telegram bot token is valid
func (t *Telegram) VerifyToken() (botName, botUsername string, err error) {
	if t.Token == "" {
		return "", "", fmt.Errorf("token kosong")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", t.Token)
	resp, err := http.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("gagal koneksi ke Telegram: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var botInfo TelegramBotInfo
	if err := json.Unmarshal(body, &botInfo); err != nil {
		return "", "", fmt.Errorf("invalid response: %w", err)
	}

	if !botInfo.Ok {
		return "", "", fmt.Errorf("token tidak valid: %s", botInfo.Description)
	}

	return botInfo.Result.FirstName, botInfo.Result.Username, nil
}

// SendMessage mengirim pesan ke chat ID
func (t *Telegram) SendMessage(chatID, text string) error {
	if t.Token == "" || chatID == "" {
		return fmt.Errorf("token atau chat_id kosong")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Token)
	msg := TelegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	payload, _ := json.Marshal(msg)
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("gagal send message: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result TelegramResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	if !result.Ok {
		return fmt.Errorf("telegram error: %s", result.Description)
	}

	return nil
}

// SendStorageAlert mengirim alert storage warning
func (t *Telegram) SendStorageAlert(serverName string, usedPercent float64, chatID string) error {
	text := fmt.Sprintf(
		"<b>⚠️ Storage Alert</b>\n\n"+
			"Server: <code>%s</code>\n"+
			"Pemakaian: <b>%.1f%%</b>\n"+
			"Waktu: %s",
		serverName, usedPercent, time.Now().Format("2006-01-02 15:04:05"),
	)
	return t.SendMessage(chatID, text)
}

// SendBackupAlert mengirim alert backup gagal
func (t *Telegram) SendBackupAlert(backupName string, chatID string) error {
	text := fmt.Sprintf(
		"<b>❌ Backup Failed</b>\n\n"+
			"Backup: <code>%s</code>\n"+
			"Status: Verification FAILED\n"+
			"Waktu: %s",
		backupName, time.Now().Format("2006-01-02 15:04:05"),
	)
	return t.SendMessage(chatID, text)
}

// SendDailySummary mengirim laporan harian
func (t *Telegram) SendDailySummary(summary string, chatID string) error {
	text := fmt.Sprintf(
		"<b>📊 Daily Storage Summary</b>\n\n%s\n\nWaktu: %s",
		summary, time.Now().Format("2006-01-02 15:04:05"),
	)
	return t.SendMessage(chatID, text)
}
