//go:build linux
// +build linux

package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"
	"github.com/hermes/agent/internal/config"
)

type StorageScanner struct {
	config *config.Config
}

type DiskMetrics struct {
	ServerID       string    `json:"server_id"`
	Path           string    `json:"path"`
	TotalBytes     uint64    `json:"total_bytes"`
	UsedBytes      uint64    `json:"used_bytes"`
	AvailableBytes uint64    `json:"available_bytes"`
	UsedPercent    float64   `json:"used_percent"`
	Timestamp      time.Time `json:"timestamp"`
}

type FileCandidate struct {
	ServerID     string    `json:"server_id"`
	Path         string    `json:"path"`
	SizeBytes    int64     `json:"size_bytes"`
	LastAccess   time.Time `json:"last_access"`
	LastModified time.Time `json:"last_modified"`
	Category     string    `json:"category"` // "log", "temp", "cache", "old_backup", etc
	Confidence   float64   `json:"confidence"` // 0.0-1.0, seberapa yakin file aman dihapus
}

func NewStorageScanner(cfg *config.Config) *StorageScanner {
	return &StorageScanner{config: cfg}
}

func (s *StorageScanner) ScanDiskUsage() ([]DiskMetrics, error) {
	var metrics []DiskMetrics

	for _, mountPath := range s.config.MountPaths {
	var stat unix.Statfs_t
	err := unix.Statfs(mountPath, &stat)
	if err != nil {
		return nil, fmt.Errorf("failed to stat mount %s: %w", mountPath, err)
	}

		totalBytes := stat.Blocks * uint64(stat.Bsize)
		availableBytes := stat.Bavail * uint64(stat.Bsize)
		usedBytes := totalBytes - availableBytes
		usedPercent := float64(usedBytes) / float64(totalBytes) * 100

		metrics = append(metrics, DiskMetrics{
			ServerID:       s.config.ServerID,
			Path:           mountPath,
			TotalBytes:     totalBytes,
			UsedBytes:      usedBytes,
			AvailableBytes: availableBytes,
			UsedPercent:    usedPercent,
			Timestamp:      time.Now(),
		})
	}

	return metrics, nil
}

// ScanFileCandidates mencari file yang potensial untuk dibersihkan
func (s *StorageScanner) ScanFileCandidates(targetPath string) ([]FileCandidate, error) {
	var candidates []FileCandidate

	// Daftar path yang aman untuk discan (hindari system-critical paths)
	safePaths := []string{
		"/var/log",
		"/tmp",
		"/var/tmp",
		"/var/cache",
	}

	now := time.Now()
	oldThreshold := now.AddDate(0, 0, -90) // File tidak diakses 90 hari

	for _, basePath := range safePaths {
		err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// Skip files/dirs that can't be accessed
				return nil
			}

			if info.IsDir() {
				return nil
			}

			// Skip files kecil (< 10MB)
			if info.Size() < 10*1024*1024 {
				return nil
			}

			// Gunakan ModTime sebagai proxy untuk LastAccess. Waktu akses (atime) adalah platform spesifik
			// dan sulit didapatkan di Linux, ModTime cukup untuk pipeline pembersihan Fase 1.
			lastAccess := info.ModTime()

			// Kategorisasi berdasarkan path dan ekstensi
			category := categorizeFile(path)
			confidence := calculateConfidence(path, info, lastAccess, oldThreshold)

			// Hanya masukkan kandidat dengan confidence > 0.5
			if confidence > 0.5 {
				candidates = append(candidates, FileCandidate{
					ServerID:     s.config.ServerID,
					Path:         path,
					SizeBytes:    info.Size(),
					LastAccess:   lastAccess,
					LastModified: info.ModTime(),
					Category:     category,
					Confidence:   confidence,
				})
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("error walking path %s: %w", basePath, err)
		}
	}

	return candidates, nil
}

func categorizeFile(path string) string {
	ext := filepath.Ext(path)

	if filepath.HasPrefix(path, "/var/log") {
		return "log"
	}
	if filepath.HasPrefix(path, "/tmp") || filepath.HasPrefix(path, "/var/tmp") {
		return "temp"
	}
	if filepath.HasPrefix(path, "/var/cache") {
		return "cache"
	}
	if ext == ".old" || ext == ".bak" {
		return "old_backup"
	}

	return "other"
}

func calculateConfidence(path string, info os.FileInfo, lastAccess time.Time, oldThreshold time.Time) float64 {
	confidence := 0.0

	// Base confidence berdasarkan kategori
	if filepath.HasPrefix(path, "/var/log") && lastAccess.Before(oldThreshold) {
		confidence = 0.8 // Log lama relatif aman
	} else if filepath.HasPrefix(path, "/tmp") {
		confidence = 0.9 // Temp files sangat aman
	} else if filepath.HasPrefix(path, "/var/cache") {
		confidence = 0.7 // Cache bisa di-regenerate
	} else {
		confidence = 0.3 // Default rendah untuk unknown
	}

	// Adjust berdasarkan umur file
	daysSinceAccess := time.Since(lastAccess).Hours() / 24
	if daysSinceAccess > 180 {
		confidence += 0.1
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}
