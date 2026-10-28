package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type TelegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int `json:"message_id"`
		Chat      struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Username  string `json:"username"`
		} `json:"chat"`
		Text string `json:"text"`
		Date int64  `json:"date"`
	} `json:"message"`
}

type TelegramUpdatesResp struct {
	Ok     bool            `json:"ok"`
	Result []TelegramUpdate `json:"result"`
}

// StartTelegramListener polls getUpdates for /hubungkan, auto-saves chat_id
func StartTelegramListener(ctx context.Context, token string, db *sql.DB) error {
	offset := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=5", token, offset)
			resp, err := http.Get(url)
			if err != nil || resp.StatusCode != 200 {
				continue
			}

			var updates TelegramUpdatesResp
			if err := json.NewDecoder(resp.Body).Decode(&updates); err != nil {
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			if !updates.Ok || len(updates.Result) == 0 {
				continue
			}

			for _, upd := range updates.Result {
				offset = upd.UpdateID + 1

				if upd.Message.Text != "" && strings.Contains(upd.Message.Text, "/hubungkan") {
					chatID := fmt.Sprintf("%d", upd.Message.Chat.ID)
					firstName := upd.Message.Chat.FirstName

					// Auto-save to settings
					config := map[string]string{
						"token":      token,
						"chat_id":    chatID,
						"first_name": firstName,
					}
					configJSON, _ := json.Marshal(config)
					_, _ = db.ExecContext(context.Background(),
						`INSERT INTO settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = CURRENT_TIMESTAMP`,
						"telegram_config", string(configJSON))

					return nil // Connection established
				}
			}
		}
	}
}
