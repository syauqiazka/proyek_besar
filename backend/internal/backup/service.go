package backup

import (
	"errors"
	"fmt"
	"time"
)

type BackupService struct {
	// TODO: Add database connection, storage service
}

func NewBackupService() *BackupService {
	return &BackupService{}
}

// CheckBackupStatus memeriksa status job backup
func (s *BackupService) CheckBackupStatus(serverID string, backupName string) (*BackupStatus, error) {
	// TODO: fetch backup from database
	// TODO: run actual backup status check (not just status check)
	
	// Mock implementation untuk Fase 1 MVP
	if backupName == "nonexistent" {
		return nil, errors.New("backup not found")
	}

	return &BackupStatus{
		BackupID:   generateBackupID(),
		BackupName: backupName,
		ServerID:   serverID,
		Status:     "success", // success, failed, running, pending
		SizeBytes:  1024 * 1024 * 100, // 100MB
		StartedAt:  time.Now().Add(-2 * time.Hour), // 2 hours ago
		CompletedAt: time.Now().Add(-1 * time.Hour), // 1 hour ago
		LastChecked: time.Now(),
		Type:        "full",
	}, nil
}

// StartBackupJob memulai backup job baru (Fase 2 fitur)
func (s *BackupService) StartBackupJob(serverID string, backupType string) (*BackupJob, error) {
	// TODO: implement full backup job di Fase 2
	
	return nil, errors.New("backup jobs not implemented in Fase 1 MVP")
}

// Struktur data

type BackupStatus struct {
	BackupID    string
	BackupName  string
	ServerID    string
	Status      string // success, failed, running, pending
	SizeBytes   int64
	StartedAt   time.Time
	CompletedAt time.Time
	LastChecked time.Time
	Type        string // full, incremental, snapshot
}

type BackupJob struct {
	JobID      string
	BackupID   string
	ServerID   string
	Status     string // pending, running, completed, failed
	StartedAt  time.Time
	EndedAt    time.Time
	Progress   float64 // 0.0 - 1.0
}

// Helper functions
func generateBackupID() string {
	return fmt.Sprintf("backup_%d", time.Now().UnixNano())
}
