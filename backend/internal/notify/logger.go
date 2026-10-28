package notify

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type NotificationLog struct {
	ID             string
	Channel        string // "telegram", "whatsapp"
	NotificationType string // "storage_alert", "backup_failed", "daily_summary"
	ServerID       *string
	Payload        string
	SentAt         time.Time
	DeliveryStatus string // "sent", "failed"
	Error          *string
}

type Logger struct {
	db *sql.DB
}

func NewLogger(db *sql.DB) *Logger {
	return &Logger{db: db}
}

// LogNotification mencatat notifikasi ke database
func (l *Logger) LogNotification(ctx context.Context, channel, notificationType string, serverID *string, payload, status string, err error) error {
	var errMsg *string
	if err != nil {
		e := err.Error()
		errMsg = &e
	}

	query := `
		INSERT INTO notifications (channel, notification_type, server_id, payload, sent_at, delivery_status, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, execErr := l.db.ExecContext(ctx, query,
		channel, notificationType, serverID, payload, time.Now().UTC(), status, errMsg,
	)

	if execErr != nil {
		return fmt.Errorf("gagal log notifikasi: %w", execErr)
	}

	return nil
}

// GetNotificationLog mengambil riwayat notifikasi
func (l *Logger) GetNotificationLog(ctx context.Context, limit int) ([]NotificationLog, error) {
	query := `
		SELECT id, channel, notification_type, server_id, payload, sent_at, delivery_status, error
		FROM notifications
		ORDER BY sent_at DESC
		LIMIT $1
	`

	rows, err := l.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("gagal query notifications: %w", err)
	}
	defer rows.Close()

	var logs []NotificationLog
	for rows.Next() {
		var log NotificationLog
		if err := rows.Scan(&log.ID, &log.Channel, &log.NotificationType, &log.ServerID, &log.Payload, &log.SentAt, &log.DeliveryStatus, &log.Error); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}
