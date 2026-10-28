package prediction

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"
)

// PredictionResult holds prediction range (min/max/best_estimate days until full)
type PredictionResult struct {
	ServerID      string
	MinDays       *int    // pessimistic (fast growth)
	MaxDays       *int    // optimistic (slow growth)
	BestEstimate  *int    // most likely
	Confidence    float64 // 0-1, low if data < 7 days
	Status        string  // "ok", "insufficient_data", "no_growth", "full"
	Method        string  // "linear_regression", "insufficient_data"
	CalculatedAt  time.Time
	DataPointsDays int
}

// Engine calculates storage predictions
type Engine struct {
	db *sql.DB
}

func NewEngine(db *sql.DB) *Engine {
	return &Engine{db: db}
}

// CalculatePrediction computes days until disk full for a server
func (e *Engine) CalculatePrediction(ctx context.Context, serverID string) (*PredictionResult, error) {
	// Get recent snapshots (last 30 days max)
	rows, err := e.db.QueryContext(ctx, `
		SELECT captured_at, used_bytes, total_bytes
		FROM storage_snapshots
		WHERE server_id = $1
		  AND captured_at > NOW() - INTERVAL '30 days'
		ORDER BY captured_at ASC`, serverID)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer rows.Close()

	type snapshot struct {
		timestamp  time.Time
		usedBytes  int64
		totalBytes int64
	}
	var snapshots []snapshot

	for rows.Next() {
		var s snapshot
		if err := rows.Scan(&s.timestamp, &s.usedBytes, &s.totalBytes); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, s)
	}

	if len(snapshots) < 2 {
		// Insufficient data
		return &PredictionResult{
			ServerID:      serverID,
			Status:        "insufficient_data",
			Method:        "insufficient_data",
			Confidence:    0.0,
			CalculatedAt:  time.Now(),
			DataPointsDays: 0,
		}, nil
	}

	// Calculate data span in days
	firstTime := snapshots[0].timestamp
	lastTime := snapshots[len(snapshots)-1].timestamp
	spanDays := int(lastTime.Sub(firstTime).Hours() / 24)

	// Confidence based on data span (prinsip #3: < 7 hari = low confidence)
	confidence := math.Min(float64(spanDays)/7.0, 1.0)

	// Linear regression: y = mx + b (y=usedBytes, x=days since first snapshot)
	var sumX, sumY, sumXY, sumX2 float64
	n := float64(len(snapshots))

	for i, s := range snapshots {
		x := float64(i) // days index
		y := float64(s.usedBytes)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Slope (bytes per day)
	denominator := n*sumX2 - sumX*sumX
	if math.Abs(denominator) < 1e-10 {
		// No variance in X (all same time) - shouldn't happen
		return &PredictionResult{
			ServerID:      serverID,
			Status:        "insufficient_data",
			Method:        "insufficient_data",
			Confidence:    0.0,
			CalculatedAt:  time.Now(),
			DataPointsDays: spanDays,
		}, nil
	}

	slope := (n*sumXY - sumX*sumY) / denominator
	// intercept := (sumY - slope*sumX) / n // not needed for prediction

	// Check if disk is growing
	if slope <= 0 {
		// No growth or shrinking
		return &PredictionResult{
			ServerID:      serverID,
			Status:        "no_growth",
			Method:        "linear_regression",
			Confidence:    confidence,
			CalculatedAt:  time.Now(),
			DataPointsDays: spanDays,
		}, nil
	}

	// Current state (last snapshot)
	lastSnapshot := snapshots[len(snapshots)-1]
	currentUsed := float64(lastSnapshot.usedBytes)
	totalBytes := float64(lastSnapshot.totalBytes)

	// Check if already full
	if currentUsed >= totalBytes*0.98 {
		return &PredictionResult{
			ServerID:      serverID,
			Status:        "full",
			Method:        "linear_regression",
			Confidence:    confidence,
			CalculatedAt:  time.Now(),
			DataPointsDays: spanDays,
		}, nil
	}

	// Calculate days until full: (totalBytes - currentUsed) / slope
	remainingBytes := totalBytes - currentUsed
	daysUntilFull := remainingBytes / slope

	// Range calculation (prinsip #3: min/max/best_estimate)
	// Pessimistic: +20% growth rate
	// Optimistic: -20% growth rate
	pessimisticSlope := slope * 1.2
	optimisticSlope := slope * 0.8

	minDays := int(remainingBytes / pessimisticSlope)
	maxDays := int(remainingBytes / optimisticSlope)
	bestEstimate := int(daysUntilFull)

	// Sanity bounds
	if minDays < 0 {
		minDays = 0
	}
	if maxDays > 3650 { // cap at 10 years
		maxDays = 3650
	}
	if bestEstimate < minDays {
		bestEstimate = minDays
	}
	if bestEstimate > maxDays {
		bestEstimate = maxDays
	}

	return &PredictionResult{
		ServerID:      serverID,
		MinDays:       &minDays,
		MaxDays:       &maxDays,
		BestEstimate:  &bestEstimate,
		Status:        "ok",
		Method:        "linear_regression",
		Confidence:    confidence,
		CalculatedAt:  time.Now(),
		DataPointsDays: spanDays,
	}, nil
}

// SavePrediction stores prediction to database
func (e *Engine) SavePrediction(ctx context.Context, pred *PredictionResult) error {
	_, err := e.db.ExecContext(ctx, `
		INSERT INTO predictions (server_id, min_days, max_days, best_estimate_days, confidence, status, method, calculated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		pred.ServerID, pred.MinDays, pred.MaxDays, pred.BestEstimate,
		pred.Confidence, pred.Status, pred.Method, pred.CalculatedAt)
	return err
}

// RunForAllServers calculates and saves predictions for all active servers
func (e *Engine) RunForAllServers(ctx context.Context) error {
	rows, err := e.db.QueryContext(ctx, `
		SELECT id FROM servers WHERE status IN ('active', 'connected')`)
	if err != nil {
		return fmt.Errorf("failed to query servers: %w", err)
	}
	defer rows.Close()

	var serverIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		serverIDs = append(serverIDs, id)
	}

	successCount := 0
	for _, serverID := range serverIDs {
		pred, err := e.CalculatePrediction(ctx, serverID)
		if err != nil {
			fmt.Printf("Prediction failed for server %s: %v\n", serverID, err)
			continue
		}

		if err := e.SavePrediction(ctx, pred); err != nil {
			fmt.Printf("Failed to save prediction for server %s: %v\n", serverID, err)
			continue
		}
		successCount++
	}

	fmt.Printf("Prediction engine: %d/%d servers processed\n", successCount, len(serverIDs))
	return nil
}
