package notify

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type NotificationService struct {
	APIKey       string
	APIURL       string
	HTTPClient   *http.Client
}

func NewNotificationService(apiKey string, apiURL string) *NotificationService {
	return &NotificationService{
		APIKey:     apiKey,
		APIURL:     apiURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SendWhatsAppMessage mengirim pesan WhatsApp menggunakan Wablas/Fonnte API
func (s *NotificationService) SendWhatsAppMessage(phoneNumber string, message string) error {
	if s.APIKey == "" {
		return errors.New("WhatsApp API key not configured")
	}

	payload := map[string]interface{}{
		"token":   s.APIKey,
		"to":      phoneNumber,
		"message": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", s.APIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}
