package db

import (
	"context"
	"encoding/json"
	"fmt"
)

// ExportConfigTable exports all rows from a given table as a slice of JSON objects.
func (r *Repository) ExportConfigTable(ctx context.Context, tableName string) ([]json.RawMessage, error) {
	// Restrict to known config tables to prevent SQL injection
	validTables := map[string]bool{
		"scheduled_programs":     true,
		"source_credentials":     true,
		"delivery_targets":       true,
		"mapping_transformation": true,
		"mapping_rule":           true,
		"mapping_validation":     true,
		"mapping_source":         true,
		"mapping_target_field":   true,
		"topic_dependencies":     true,
	}

	if !validTables[tableName] {
		return nil, fmt.Errorf("invalid table for backup: %s", tableName)
	}

	query := fmt.Sprintf("SELECT row_to_json(t) FROM %s t", tableName)
	rows, err := r.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s: %w", tableName, err)
	}
	defer rows.Close()

	var results []json.RawMessage
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, fmt.Errorf("failed to scan row from %s: %w", tableName, err)
		}
		results = append(results, json.RawMessage(b))
	}
	return results, rows.Err()
}

// ImportConfigTable imports a slice of JSON objects into a given table.
func (r *Repository) ImportConfigTable(ctx context.Context, tableName string, rowsData []json.RawMessage) error {
	validTables := map[string]bool{
		"scheduled_programs":     true,
		"source_credentials":     true,
		"delivery_targets":       true,
		"mapping_transformation": true,
		"mapping_rule":           true,
		"mapping_validation":     true,
		"mapping_source":         true,
		"mapping_target_field":   true,
		"topic_dependencies":     true,
	}

	if !validTables[tableName] {
		return fmt.Errorf("invalid table for restore: %s", tableName)
	}

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Clean table first
	_, err = tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s", tableName))
	if err != nil {
		return fmt.Errorf("failed to clear table %s: %w", tableName, err)
	}

	for _, row := range rowsData {
		// Insert JSON via json_populate_record
		query := fmt.Sprintf("INSERT INTO %s SELECT * FROM json_populate_record(null::%s, $1::json)", tableName, tableName)
		if _, err := tx.Exec(ctx, query, string(row)); err != nil {
			return fmt.Errorf("failed to insert row into %s: %w", tableName, err)
		}
	}

	return tx.Commit(ctx)
}
