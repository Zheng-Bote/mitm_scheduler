/**
 * SPDX-FileComment: Database
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file db.go
 * @brief PostgreSQL repository for job and audit log persistence
 * @version 1.0.0
 * @date 2026-06-02
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

// Package db implements the PostgreSQL data-access layer for the scheduler.
// It manages job configurations, execution runs, system logs, audit trails,
// and IPC status events through a connection pool.
package db

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go-scheduler/internal/crypto"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ScheduledProgram represents a job configuration from the DB
type ScheduledProgram struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"`
	Command       string          `json:"command"`
	Args          json.RawMessage `json:"args"`
	CronExpr      string          `json:"cron_expr"`
	Enabled       bool            `json:"enabled"`
	RestartOnExit bool            `json:"restart_on_exit"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// ProgramRun tracks an execution instance of a job
type ProgramRun struct {
	ID         int
	ProgramID  int
	PID        int
	StartedAt  time.Time
	FinishedAt *time.Time
	ExitCode   *int
	Success    *bool
}

// SchedulerConfig holds runtime settings for the scheduler
type SchedulerConfig struct {
	HTTPPort   int
	SocketPath string
	LogLevel   string
}

// Repository handles database operations
type Repository struct {
	Pool *pgxpool.Pool
}

// NewRepository creates a new repository with a connection pool
func NewRepository(ctx context.Context, dsn string) (*Repository, error) {
	config_pool, err := pgxpool.ParseConfig(dsn)
	if err == nil {
		config_pool.MaxConns = 20
		config_pool.MaxConnIdleTime = 5 * time.Minute
		config_pool.MaxConnLifetime = 1 * time.Hour
	}
	var pool *pgxpool.Pool
	if err == nil {
		pool, err = pgxpool.NewWithConfig(ctx, config_pool)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	return &Repository{Pool: pool}, nil
}

// GetSchedulerConfig fetches the scheduler configuration
func (r *Repository) GetSchedulerConfig(ctx context.Context) (*SchedulerConfig, error) {
	var cfg SchedulerConfig
	err := r.Pool.QueryRow(ctx, "SELECT http_port, socket_path, log_level FROM scheduler_config LIMIT 1").Scan(&cfg.HTTPPort, &cfg.SocketPath, &cfg.LogLevel)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LogSystem records a system-wide log entry
func (r *Repository) LogSystem(ctx context.Context, level, component, message string) {
	if r.Pool == nil {
		fmt.Printf("[DB-LOG-SKIPPED] %s: [%s] %s: %s\n", time.Now().Format(time.RFC3339), level, component, message)
		return
	}
	_, err := r.Pool.Exec(ctx, "INSERT INTO system_logs (level, component, message) VALUES ($1, $2, $3)", level, component, message)
	if err != nil {
		fmt.Printf("[DB-LOG-ERROR] Failed to write to system_logs: %v (Original Msg: [%s] %s: %s)\n", err, level, component, message)
	}
}

// CreateAuditLog records a job audit entry
func (r *Repository) CreateAuditLog(ctx context.Context, runID int, component, message string) error {
	if r.Pool == nil {
		fmt.Printf("[DB-AUDIT-SKIPPED] RunID: %d, Component: %s, Message: %s\n", runID, component, message)
		return nil
	}
	_, err := r.Pool.Exec(ctx, "INSERT INTO job_audit_logs (run_id, component, message) VALUES ($1, $2, $3)", runID, component, message)
	if err != nil {
		fmt.Printf("[DB-AUDIT-ERROR] Failed to write to job_audit_logs: %v\n", err)
	}
	return err
}

// LogAdminAction records an administrative action
func (r *Repository) LogAdminAction(ctx context.Context, username, action string, details interface{}) {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		fmt.Printf("[DB-ADMIN-LOG-ERROR] Failed to marshal details for User: %s, Action: %s: %v\n", username, action, err)
		detailsJSON = []byte(`{"error":"failed to marshal details"}`)
	}
	if r.Pool == nil {
		fmt.Printf("[DB-ADMIN-LOG-SKIPPED] User: %s, Action: %s, Details: %s\n", username, action, string(detailsJSON))
		return
	}
	_, err = r.Pool.Exec(ctx, "INSERT INTO admin_audit_logs (username, action, details) VALUES ($1, $2, $3)", username, action, detailsJSON)
	if err != nil {
		fmt.Printf("[DB-ADMIN-LOG-ERROR] %v\n", err)
	}
}

// UpsertScheduledProgram inserts or updates a program configuration
func (r *Repository) UpsertScheduledProgram(ctx context.Context, p ScheduledProgram) error {
	query := `
		INSERT INTO scheduled_programs (name, command, args, cron_expr, enabled, restart_on_exit, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		ON CONFLICT (name) DO UPDATE SET
			command = EXCLUDED.command,
			args = EXCLUDED.args,
			cron_expr = EXCLUDED.cron_expr,
			enabled = EXCLUDED.enabled,
			restart_on_exit = EXCLUDED.restart_on_exit,
			updated_at = CURRENT_TIMESTAMP`

	// Note: You need a UNIQUE constraint on 'name' for ON CONFLICT to work.
	// We'll assume the migration adds it or we use a manual check.
	_, err := r.Pool.Exec(ctx, query, p.Name, p.Command, p.Args, p.CronExpr, p.Enabled, p.RestartOnExit)
	return err
}

// GetAllPrograms fetches all scheduled programs
func (r *Repository) GetAllPrograms(ctx context.Context) ([]ScheduledProgram, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id, name, command, args, cron_expr, enabled, restart_on_exit FROM scheduled_programs ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var programs []ScheduledProgram
	for rows.Next() {
		var p ScheduledProgram
		if err := rows.Scan(&p.ID, &p.Name, &p.Command, &p.Args, &p.CronExpr, &p.Enabled, &p.RestartOnExit); err != nil {
			return nil, err
		}
		programs = append(programs, p)
	}
	return programs, nil
}

// GetEnabledPrograms fetches all enabled scheduled programs
func (r *Repository) GetEnabledPrograms(ctx context.Context) ([]ScheduledProgram, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id, name, command, args, cron_expr, enabled, restart_on_exit FROM scheduled_programs WHERE enabled = true")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var programs []ScheduledProgram
	for rows.Next() {
		var p ScheduledProgram
		if err := rows.Scan(&p.ID, &p.Name, &p.Command, &p.Args, &p.CronExpr, &p.Enabled, &p.RestartOnExit); err != nil {
			return nil, err
		}
		programs = append(programs, p)
	}
	return programs, nil
}

// CreateProgramRun initializes a new program run entry
func (r *Repository) CreateProgramRun(ctx context.Context, programID int, pid int) (int, error) {
	var id int
	var err error
	if programID <= 0 {
		err = r.Pool.QueryRow(ctx, "INSERT INTO program_runs (pid, started_at) VALUES ($1, CURRENT_TIMESTAMP) RETURNING id", pid).Scan(&id)
	} else {
		err = r.Pool.QueryRow(ctx, "INSERT INTO program_runs (program_id, pid, started_at) VALUES ($1, $2, CURRENT_TIMESTAMP) RETURNING id", programID, pid).Scan(&id)
	}
	return id, err
}

// UpdateProgramRun updates a run entry upon completion
func (r *Repository) UpdateProgramRun(ctx context.Context, runID int, exitCode int, success bool) error {
	_, err := r.Pool.Exec(ctx, "UPDATE program_runs SET finished_at = CURRENT_TIMESTAMP, exit_code = $1, success = $2 WHERE id = $3", exitCode, success, runID)
	return err
}

// CreateStatusEvent logs an IPC status event
func (r *Repository) CreateStatusEvent(ctx context.Context, runID int, status string, message string, progress int) error {
	_, err := r.Pool.Exec(ctx, "INSERT INTO job_status_events (run_id, status, message, progress) VALUES ($1, $2, $3, $4)", runID, status, message, progress)
	return err
}

// SystemLog represents a log entry in the system_logs table
type SystemLog struct {
	ID        int       `json:"id"`
	Level     string    `json:"level"`
	Component string    `json:"component"`
	Message   string    `json:"message"`
	TS        time.Time `json:"ts"`
}

// JobAuditLog represents a log entry in the job_audit_logs table
type JobAuditLog struct {
	ID        int       `json:"id"`
	RunID     int       `json:"run_id"`
	Component string    `json:"component"`
	Message   string    `json:"message"`
	TS        time.Time `json:"ts"`
}

// AdminAuditLog represents a log entry in the admin_audit_logs table
type AdminAuditLog struct {
	ID       int             `json:"id"`
	Username string          `json:"username"`
	Action   string          `json:"action"`
	Details  json.RawMessage `json:"details"`
	TS       time.Time       `json:"ts"`
}

// GetSystemLogs retrieves system logs, optionally filtered by a date range
func (r *Repository) GetSystemLogs(ctx context.Context, from, to *time.Time) ([]SystemLog, error) {
	query := "SELECT id, level, component, message, ts FROM system_logs WHERE 1=1"
	var args []interface{}
	placeholderIdx := 1

	if from != nil {
		query += fmt.Sprintf(" AND ts >= $%d", placeholderIdx)
		args = append(args, *from)
		placeholderIdx++
	}
	if to != nil {
		query += fmt.Sprintf(" AND ts <= $%d", placeholderIdx)
		args = append(args, *to)
		placeholderIdx++
	}
	query += " ORDER BY ts DESC"

	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []SystemLog
	for rows.Next() {
		var l SystemLog
		if err := rows.Scan(&l.ID, &l.Level, &l.Component, &l.Message, &l.TS); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// GetJobAuditLogs retrieves job audit logs, optionally filtered by a date range
func (r *Repository) GetJobAuditLogs(ctx context.Context, from, to *time.Time) ([]JobAuditLog, error) {
	query := "SELECT id, run_id, component, message, ts FROM job_audit_logs WHERE 1=1"
	var args []interface{}
	placeholderIdx := 1

	if from != nil {
		query += fmt.Sprintf(" AND ts >= $%d", placeholderIdx)
		args = append(args, *from)
		placeholderIdx++
	}
	if to != nil {
		query += fmt.Sprintf(" AND ts <= $%d", placeholderIdx)
		args = append(args, *to)
		placeholderIdx++
	}
	query += " ORDER BY ts DESC"

	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []JobAuditLog
	for rows.Next() {
		var l JobAuditLog
		if err := rows.Scan(&l.ID, &l.RunID, &l.Component, &l.Message, &l.TS); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// GetAdminAuditLogs retrieves administrative audit logs, optionally filtered by a date range
func (r *Repository) GetAdminAuditLogs(ctx context.Context, from, to *time.Time) ([]AdminAuditLog, error) {
	query := "SELECT id, username, action, details, ts FROM admin_audit_logs WHERE 1=1"
	var args []interface{}
	placeholderIdx := 1

	if from != nil {
		query += fmt.Sprintf(" AND ts >= $%d", placeholderIdx)
		args = append(args, *from)
		placeholderIdx++
	}
	if to != nil {
		query += fmt.Sprintf(" AND ts <= $%d", placeholderIdx)
		args = append(args, *to)
		placeholderIdx++
	}
	query += " ORDER BY ts DESC"

	rows, err := r.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AdminAuditLog
	for rows.Next() {
		var l AdminAuditLog
		if err := rows.Scan(&l.ID, &l.Username, &l.Action, &l.Details, &l.TS); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// SourceCredential represents a source connection
type SourceCredential struct {
	ID            string    `json:"id"`
	SourceName    string    `json:"source_name"`
	ConnectorType string    `json:"connector_type"`
	Topic         string    `json:"topic"`
	ConfigPayload string    `json:"config_payload"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GetSourceCredentials fetches all source credentials and decrypts their config_payload
func (r *Repository) GetSourceCredentials(ctx context.Context) ([]SourceCredential, error) {
	query := `
		SELECT 
			sc.id::text, sc.source_name, sc.connector_type, sc.topic, sc.is_active, sc.created_at, sc.updated_at,
			sc.config_payload, sc.nonce, sk.wrapped_key
		FROM source_credentials sc
		JOIN storage_keys sk ON sc.dek_id = sk.id
		ORDER BY sc.source_name ASC
	`
	rows, err := r.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	masterKeyStr := os.Getenv("MASTER_KEY")
	var kek []byte
	if decoded, err := base64.StdEncoding.DecodeString(masterKeyStr); err == nil {
		kek = decoded
	} else {
		kek = []byte(masterKeyStr)
	}

	var creds []SourceCredential
	creds = make([]SourceCredential, 0)
	for rows.Next() {
		var c SourceCredential
		var payload, nonce, wrappedKey []byte
		var isActive *bool
		var createdAt, updatedAt *time.Time

		if err := rows.Scan(&c.ID, &c.SourceName, &c.ConnectorType, &c.Topic, &isActive, &createdAt, &updatedAt, &payload, &nonce, &wrappedKey); err != nil {
			fmt.Printf("[DB ERROR] GetSourceCredentials Scan failed: %v\n", err)
			return nil, err
		}

		if isActive != nil {
			c.IsActive = *isActive
		}
		if createdAt != nil {
			c.CreatedAt = *createdAt
		}
		if updatedAt != nil {
			c.UpdatedAt = *updatedAt
		}

		decrypted, err := crypto.EnvelopeDecrypt(kek, wrappedKey, nonce, payload)
		if err == nil {
			c.ConfigPayload = string(decrypted)
		} else {
			c.ConfigPayload = fmt.Sprintf(`{"error": "decryption failed: %v"}`, err)
		}

		creds = append(creds, c)
	}
	return creds, nil
}

// UpsertSourceCredential inserts or updates a source credential
func (r *Repository) UpsertSourceCredential(ctx context.Context, c SourceCredential) error {
	masterKeyStr := os.Getenv("MASTER_KEY")
	var kek []byte
	if decoded, err := base64.StdEncoding.DecodeString(masterKeyStr); err == nil {
		kek = decoded
	} else {
		kek = []byte(masterKeyStr)
	}

	// Get active DEK for encryption
	var dekID string
	var wrappedKey []byte
	err := r.Pool.QueryRow(ctx, "SELECT id::text, wrapped_key FROM storage_keys WHERE is_active = true LIMIT 1").Scan(&dekID, &wrappedKey)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			wrappedKey, err = crypto.GenerateWrappedDEK(kek)
			if err != nil {
				return fmt.Errorf("failed to generate new DEK: %v", err)
			}
			err = r.Pool.QueryRow(ctx, "INSERT INTO storage_keys (wrapped_key, is_active) VALUES ($1, true) RETURNING id::text", wrappedKey).Scan(&dekID)
			if err != nil {
				return fmt.Errorf("failed to save new storage key: %v", err)
			}
		} else {
			return fmt.Errorf("failed to fetch active storage key: %v", err)
		}
	}

	encryptedPayload, nonce, err := crypto.EnvelopeEncrypt(kek, wrappedKey, []byte(c.ConfigPayload))
	if err != nil {
		return fmt.Errorf("encryption failed: %v", err)
	}

	query := `
		INSERT INTO source_credentials (source_name, connector_type, topic, config_payload, nonce, dek_id, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
		ON CONFLICT (source_name) DO UPDATE SET
			connector_type = EXCLUDED.connector_type,
			topic = EXCLUDED.topic,
			config_payload = EXCLUDED.config_payload,
			nonce = EXCLUDED.nonce,
			dek_id = EXCLUDED.dek_id,
			is_active = EXCLUDED.is_active,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err = r.Pool.Exec(ctx, query, c.SourceName, c.ConnectorType, c.Topic, encryptedPayload, nonce, dekID, c.IsActive)
	return err
}

// DLQEntry represents a record in the dead_letter_queue table
type DLQEntry struct {
	ID           string     `json:"id"`
	PackageID    *string    `json:"package_id"`
	Payload      string     `json:"payload"`
	ErrorCode    *string    `json:"error_code"`
	ErrorMessage *string    `json:"error_message"`
	FailedAt     time.Time  `json:"failed_at"`
	Resolved     bool       `json:"resolved"`
	ResolvedAt   *time.Time `json:"resolved_at"`
}

// GetDLQEntries fetches the latest unresolved DLQ entries
func (r *Repository) GetDLQEntries(ctx context.Context, limit int) ([]DLQEntry, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id::text, package_id::text, payload::text, error_code, error_message, failed_at, resolved, resolved_at FROM dead_letter_queue WHERE resolved = false ORDER BY failed_at DESC LIMIT $1", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []DLQEntry
	for rows.Next() {
		var e DLQEntry
		if err := rows.Scan(&e.ID, &e.PackageID, &e.Payload, &e.ErrorCode, &e.ErrorMessage, &e.FailedAt, &e.Resolved, &e.ResolvedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// RequeueDLQEntries requeues specified DLQ entries by resetting their package status or creating new ones, then deleting them from DLQ.
func (r *Repository) RequeueDLQEntries(ctx context.Context, dlqIDs []string) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, dlqID := range dlqIDs {
		var pkgID *string
		var payload string
		err = tx.QueryRow(ctx, "SELECT package_id::text, payload::text FROM dead_letter_queue WHERE id = $1", dlqID).Scan(&pkgID, &payload)
		if err != nil {
			return fmt.Errorf("failed to fetch dlq entry %s: %v", dlqID, err)
		}

		if pkgID != nil {
			// Update existing package
			_, err = tx.Exec(ctx, "UPDATE packages SET status = 'pending', retry_count = 0, next_retry_at = NOW(), error_message = NULL WHERE id = $1", *pkgID)
			if err != nil {
				return fmt.Errorf("failed to update package for dlq %s: %v", dlqID, err)
			}
		} else {
			// Insert a new package if original package doesn't exist anymore
			_, err = tx.Exec(ctx, "INSERT INTO packages (payload, idempotency_key, status) VALUES ($1, gen_random_uuid(), 'pending')", payload)
			if err != nil {
				return fmt.Errorf("failed to recreate package for dlq %s: %v", dlqID, err)
			}
		}

		// Remove from DLQ
		_, err = tx.Exec(ctx, "DELETE FROM dead_letter_queue WHERE id = $1", dlqID)
		if err != nil {
			return fmt.Errorf("failed to delete dlq entry %s: %v", dlqID, err)
		}
	}

	return tx.Commit(ctx)
}
