package ssh

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// CommandLog represents an SSH command execution for audit trail
type CommandLog struct {
	ID               string
	ServerID         string
	CommandTemplate  string
	Parameters       map[string]interface{}
	ExecutedCommand  string
	ExitCode         int
	DurationMs       int64
	TriggeredBy      string // "cron", "user:{uuid}", "agent"
	ExecutedAt       time.Time
}

// AuditLogger logs SSH command executions to database
type AuditLogger struct {
	db *sql.DB
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *sql.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// LogExecution records a command execution in ssh_command_logs table
func (a *AuditLogger) LogExecution(ctx context.Context, log *CommandLog) error {
	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(log.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Insert log entry
	query := `
		INSERT INTO ssh_command_logs 
			(server_id, command_template, parameters, executed_command, exit_code, duration_ms, triggered_by, executed_at)
		VALUES 
			($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`

	err = a.db.QueryRowContext(ctx, query,
		log.ServerID,
		log.CommandTemplate,
		paramsJSON,
		log.ExecutedCommand,
		log.ExitCode,
		log.DurationMs,
		log.TriggeredBy,
		log.ExecutedAt,
	).Scan(&log.ID)

	if err != nil {
		return fmt.Errorf("failed to insert command log: %w", err)
	}

	return nil
}

// GetRecentLogs retrieves recent command logs for a server
func (a *AuditLogger) GetRecentLogs(ctx context.Context, serverID string, limit int) ([]*CommandLog, error) {
	query := `
		SELECT id, server_id, command_template, parameters, executed_command, 
		       exit_code, duration_ms, triggered_by, executed_at
		FROM ssh_command_logs
		WHERE server_id = $1
		ORDER BY executed_at DESC
		LIMIT $2`

	rows, err := a.db.QueryContext(ctx, query, serverID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query command logs: %w", err)
	}
	defer rows.Close()

	var logs []*CommandLog
	for rows.Next() {
		log := &CommandLog{}
		var paramsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.ServerID,
			&log.CommandTemplate,
			&paramsJSON,
			&log.ExecutedCommand,
			&log.ExitCode,
			&log.DurationMs,
			&log.TriggeredBy,
			&log.ExecutedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log row: %w", err)
		}

		// Unmarshal parameters
		if len(paramsJSON) > 0 {
			if err := json.Unmarshal(paramsJSON, &log.Parameters); err != nil {
				return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
			}
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// ExecuteWithAudit executes a command and logs it to audit trail
func ExecuteWithAudit(
	ctx context.Context,
	executor *SSHExecutor,
	auditLogger *AuditLogger,
	serverID string,
	templateName string,
	params map[string]interface{},
	triggeredBy string,
) (*ExecuteResult, error) {
	// Execute command
	result, err := executor.ExecuteTemplate(templateName, params)

	// Log execution (even if failed)
	log := &CommandLog{
		ServerID:        serverID,
		CommandTemplate: templateName,
		Parameters:      params,
		ExecutedCommand: result.Command,
		ExitCode:        result.ExitCode,
		DurationMs:      result.DurationMs,
		TriggeredBy:     triggeredBy,
		ExecutedAt:      time.Now(),
	}

	if logErr := auditLogger.LogExecution(ctx, log); logErr != nil {
		// Log error but don't fail the operation
		fmt.Printf("WARNING: failed to log SSH command: %v\n", logErr)
	}

	return result, err
}
