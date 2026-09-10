package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type DatabaseService struct {
	DB *sql.DB
}

func NewDatabaseService(dsn string) (*DatabaseService, error) {
	dbConn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err = dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DatabaseService{DB: dbConn}, nil
}

func (s *DatabaseService) GetDB() *sql.DB {
	return s.DB
}

func (s *DatabaseService) Close() error {
	return s.DB.Close()
}