package prediction

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler runs prediction engine on schedule
type Scheduler struct {
	engine *Engine
	cron   *cron.Cron
}

// NewScheduler creates prediction scheduler
func NewScheduler(db *sql.DB) *Scheduler {
	return &Scheduler{
		engine: NewEngine(db),
		cron:   cron.New(),
	}
}

// Start begins scheduled prediction runs
func (s *Scheduler) Start() error {
	// Run every 6 hours (agent.md: frequent enough untuk prediction akurat)
	_, err := s.cron.AddFunc("0 */6 * * *", func() {
		fmt.Printf("[%s] Running prediction engine...\n", time.Now().Format(time.RFC3339))
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := s.engine.RunForAllServers(ctx); err != nil {
			fmt.Printf("[Prediction] Error: %v\n", err)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule prediction job: %w", err)
	}

	s.cron.Start()
	fmt.Println("Prediction scheduler started (runs every 6 hours)")
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// RunNow triggers immediate prediction run (for manual trigger)
func (s *Scheduler) RunNow(ctx context.Context) error {
	return s.engine.RunForAllServers(ctx)
}
