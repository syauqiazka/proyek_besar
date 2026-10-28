package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
	"github.com/hermes/agent/internal/config"
	"github.com/hermes/agent/internal/reporter"
	"github.com/hermes/agent/internal/scanner"
	"github.com/robfig/cron/v3"
)

func main() {
	fmt.Println("Hermes Agent starting...")

	// Load .env file jika ada (di folder yang sama dengan binary)
	_ = godotenv.Load()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize scanner and reporter
	storageScanner := scanner.NewStorageScanner(cfg)
	apiReporter := reporter.NewReporter(cfg)

	// Setup cron scheduler for periodic scanning
	c := cron.New()

	// Scan storage every hour (configurable)
	_, err = c.AddFunc(cfg.ScanInterval, func() {
		log.Println("Running storage scan...")
		
		metrics, err := storageScanner.ScanDiskUsage()
		if err != nil {
			log.Printf("Error scanning disk: %v", err)
			return
		}

		err = apiReporter.SendMetrics(metrics)
		if err != nil {
			log.Printf("Error sending metrics: %v", err)
			return
		}

		log.Println("Storage metrics sent successfully")
	})

	if err != nil {
		log.Fatalf("Failed to schedule scan job: %v", err)
	}

	// Run initial scan immediately
	log.Println("Running initial scan...")
	metrics, err := storageScanner.ScanDiskUsage()
	if err != nil {
		log.Printf("Initial scan error: %v", err)
	} else {
		err = apiReporter.SendMetrics(metrics)
		if err != nil {
			log.Printf("Error sending initial metrics: %v", err)
		}
	}

	// Start cron scheduler
	c.Start()

	log.Printf("Agent running with scan interval: %s", cfg.ScanInterval)
	log.Println("Press Ctrl+C to stop")

	// Keep the agent running
	select {}
}
