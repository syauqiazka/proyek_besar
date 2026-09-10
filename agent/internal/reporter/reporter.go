package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hermes/agent/internal/config"
	"github.com/hermes/agent/internal/scanner"
)

type Reporter struct {
	config     *config.Config
	httpClient *http.Client
}

func NewReporter(cfg *config.Config) *Reporter {
	return &Reporter{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *Reporter) SendMetrics(metrics []scanner.DiskMetrics) error {
	url := fmt.Sprintf("%s/agent/%s/metrics", r.config.BackendURL, r.config.ServerID)

	payload := map[string]interface{}{
		"metrics": metrics,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", r.config.APIKey)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	return nil
}

func (r *Reporter) SendFileCandidates(candidates []scanner.FileCandidate) error {
	url := fmt.Sprintf("%s/agent/%s/file-candidates", r.config.BackendURL, r.config.ServerID)

	payload := map[string]interface{}{
		"candidates": candidates,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal candidates: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", r.config.APIKey)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	return nil
}

func (r *Reporter) SendBackupStatus(backupName string, status string, sizeBytes int64) error {
	url := fmt.Sprintf("%s/agent/%s/backup-status", r.config.BackendURL, r.config.ServerID)

	payload := map[string]interface{}{
		"backup_name": backupName,
		"status":      status,
		"size_bytes":  sizeBytes,
		"checked_at":  time.Now(),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal backup status: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", r.config.APIKey)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	return nil
}
