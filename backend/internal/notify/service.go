package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Service loads config from DB and sends via Telegram
type Service struct {
	db     *sql.DB
	logger *Logger
}

func NewService(db *sql.DB) *Service {
	return &Service{
		db:     db,
		logger: NewLogger(db),
	}
}

func (s *Service) LoadConfig() (token, chatID string, err error) {
	query := `SELECT value FROM settings WHERE key = 'notification_telegram' LIMIT 1`
	var valueJSON string
	err = s.db.QueryRow(query).Scan(&valueJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", nil
		}
		return "", "", err
	}

	var config map[string]string
	if err := json.Unmarshal([]byte(valueJSON), &config); err != nil {
		return "", "", err
	}
	return config["token"], config["chat_id"], nil
}

func (s *Service) VerifyToken() (name, username string, err error) {
	token, _, err := s.LoadConfig()
	if err != nil || token == "" {
		return "", "", fmt.Errorf("no telegram config")
	}

	tg := NewTelegram(token)
	return tg.VerifyToken()
}

func (s *Service) SendStorageAlert(serverName string, usedPercent float64) error {
	token, chatID, err := s.LoadConfig()
	if err != nil || token == "" || chatID == "" {
		return nil
	}

	tg := NewTelegram(token)
	text := fmt.Sprintf("<b>Storage Alert</b>\n\nServer: <code>%s</code>\nUsed: <b>%.1f%%</b>", serverName, usedPercent)
	if err := tg.SendMessage(chatID, text); err != nil {
		s.logger.LogNotification(context.Background(), "telegram", "storage_alert", nil, text, "failed", err)
		return err
	}
	s.logger.LogNotification(context.Background(), "telegram", "storage_alert", nil, text, "sent", nil)
	return nil
}

func (s *Service) SendBackupFailed(backupName string) error {
	token, chatID, err := s.LoadConfig()
	if err != nil || token == "" || chatID == "" {
		return nil
	}

	tg := NewTelegram(token)
	text := fmt.Sprintf("<b>Backup Failed</b>\n\nBackup: <code>%s</code>", backupName)
	if err := tg.SendMessage(chatID, text); err != nil {
		s.logger.LogNotification(context.Background(), "telegram", "backup_failed", nil, text, "failed", err)
		return err
	}
	s.logger.LogNotification(context.Background(), "telegram", "backup_failed", nil, text, "sent", nil)
	return nil
}

func (s *Service) SendDailySummary(summary string) error {
	token, chatID, err := s.LoadConfig()
	if err != nil || token == "" || chatID == "" {
		return nil
	}

	tg := NewTelegram(token)
	text := fmt.Sprintf("<b>Daily Storage Summary</b>\n\n%s", summary)
	if err := tg.SendMessage(chatID, text); err != nil {
		s.logger.LogNotification(context.Background(), "telegram", "daily_summary", nil, text, "failed", err)
		return err
	}
	s.logger.LogNotification(context.Background(), "telegram", "daily_summary", nil, text, "sent", nil)
	return nil
}

// VerifyTokenWithToken verifies a token directly (for settings test endpoint)
func VerifyTokenWithToken(token string) (name, username string, err error) {
	tg := NewTelegram(token)
	return tg.VerifyToken()
}

// SendMessageToChat sends a test message to a specific chat
func SendMessageToChat(token, chatID, text string) error {
	tg := NewTelegram(token)
	return tg.SendMessage(chatID, text)
}