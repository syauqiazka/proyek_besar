package prediction

import (
	"fmt"
	"math"
	"time"
)

type StorageDataPoint struct {
	Timestamp   time.Time
	UsedBytes   uint64
	TotalBytes  uint64
	UsedPercent float64
}

type PredictionResult struct {
	DaysUntilFull struct {
		Min          int
		Max          int
		BestEstimate int
	}
	Confidence       float64
	DailyGrowthRate  float64 // bytes per day
	CurrentUsedBytes uint64
	TotalBytes       uint64
}

type PredictionEngine struct{}

func NewPredictionEngine() *PredictionEngine {
	return &PredictionEngine{}
}

// Predict menggunakan linear regression sederhana dari historical data
func (p *PredictionEngine) Predict(dataPoints []StorageDataPoint) (*PredictionResult, error) {
	if len(dataPoints) < 3 {
		return nil, fmt.Errorf("need at least 3 data points for prediction")
	}

	// Sort by timestamp ascending
	// TODO: implement proper sorting

	// Calculate daily growth rate using linear regression
	growthRate, confidence := p.calculateGrowthRate(dataPoints)

	if growthRate <= 0 {
		// Storage is shrinking or stable, no prediction needed
		return &PredictionResult{
			DaysUntilFull: struct {
				Min          int
				Max          int
				BestEstimate int
			}{
				Min:          999,
				Max:          999,
				BestEstimate: 999,
			},
			Confidence:       0.0,
			DailyGrowthRate:  growthRate,
			CurrentUsedBytes: dataPoints[len(dataPoints)-1].UsedBytes,
			TotalBytes:       dataPoints[len(dataPoints)-1].TotalBytes,
		}, nil
	}

	latest := dataPoints[len(dataPoints)-1]
	availableBytes := latest.TotalBytes - latest.UsedBytes

	// Best estimate
	daysUntilFull := int(float64(availableBytes) / growthRate)

	// Calculate min/max range based on confidence
	// Lower confidence = wider range
	uncertaintyFactor := 1.0 - confidence
	rangeSpread := float64(daysUntilFull) * uncertaintyFactor * 0.3

	minDays := int(math.Max(0, float64(daysUntilFull)-rangeSpread))
	maxDays := int(float64(daysUntilFull) + rangeSpread)

	return &PredictionResult{
		DaysUntilFull: struct {
			Min          int
			Max          int
			BestEstimate int
		}{
			Min:          minDays,
			Max:          maxDays,
			BestEstimate: daysUntilFull,
		},
		Confidence:       confidence,
		DailyGrowthRate:  growthRate,
		CurrentUsedBytes: latest.UsedBytes,
		TotalBytes:       latest.TotalBytes,
	}, nil
}

// calculateGrowthRate menggunakan simple linear regression
func (p *PredictionEngine) calculateGrowthRate(dataPoints []StorageDataPoint) (float64, float64) {
	n := len(dataPoints)
	if n < 2 {
		return 0, 0
	}

	// Convert timestamps to days since first data point
	firstTime := dataPoints[0].Timestamp
	
	var sumX, sumY, sumXY, sumX2 float64

	for _, point := range dataPoints {
		x := point.Timestamp.Sub(firstTime).Hours() / 24.0 // days
		y := float64(point.UsedBytes)

		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Linear regression: y = mx + b
	// slope (m) is the daily growth rate
	slope := (float64(n)*sumXY - sumX*sumY) / (float64(n)*sumX2 - sumX*sumX)

	// Calculate R² (coefficient of determination) for confidence
	meanY := sumY / float64(n)
	var ssTotal, ssResidual float64

	for _, point := range dataPoints {
		x := point.Timestamp.Sub(firstTime).Hours() / 24.0
		y := float64(point.UsedBytes)
		yPredicted := slope*x + (meanY - slope*(sumX/float64(n)))

		ssTotal += math.Pow(y-meanY, 2)
		ssResidual += math.Pow(y-yPredicted, 2)
	}

	rSquared := 1.0 - (ssResidual / ssTotal)
	
	// Confidence based on R² (clamped 0-1)
	confidence := math.Max(0, math.Min(1, rSquared))

	return slope, confidence
}

// DetectAnomalies mencari pola tidak normal pada storage growth
func (p *PredictionEngine) DetectAnomalies(dataPoints []StorageDataPoint) []string {
	var anomalies []string

	if len(dataPoints) < 5 {
		return anomalies
	}

	// Check for sudden spike (> 20% increase in single interval)
	for i := 1; i < len(dataPoints); i++ {
		prev := dataPoints[i-1]
		curr := dataPoints[i]

		if prev.TotalBytes > 0 {
			increase := curr.UsedBytes - prev.UsedBytes
			percentIncrease := (float64(increase) / float64(prev.TotalBytes)) * 100

			if percentIncrease > 20 {
				anomalies = append(anomalies, fmt.Sprintf(
					"Sudden spike detected: %.1f%% increase between %s and %s",
					percentIncrease,
					prev.Timestamp.Format("2006-01-02 15:04"),
					curr.Timestamp.Format("2006-01-02 15:04"),
				))
			}
		}
	}

	return anomalies
}
