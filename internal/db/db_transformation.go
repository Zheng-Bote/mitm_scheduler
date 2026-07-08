package db

import (
	"context"
	"encoding/json"
	"time"
)

type MappingSource struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Topic     string    `json:"topic"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *Repository) GetMappingSources(ctx context.Context) ([]MappingSource, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id::text, name, type, topic, version, created_at FROM mapping_source ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []MappingSource
	for rows.Next() {
		var m MappingSource
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.Topic, &m.Version, &m.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, m)
	}
	return res, nil
}

func (r *Repository) UpsertMappingSource(ctx context.Context, m MappingSource) error {
	query := `
		INSERT INTO mapping_source (id, name, type, topic, version, created_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			topic = EXCLUDED.topic,
			version = EXCLUDED.version
	`
	_, err := r.Pool.Exec(ctx, query, m.ID, m.Name, m.Type, m.Topic, m.Version)
	return err
}

func (r *Repository) DeleteMappingSource(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM mapping_source WHERE id = $1", id)
	return err
}

type MappingTargetField struct {
	ID         string `json:"id"`
	Topic      string `json:"topic"`
	FieldName  string `json:"field_name"`
	DataType   string `json:"data_type"`
	IsRequired bool   `json:"is_required"`
	Encrypted  bool   `json:"encrypted"`
	Version    int    `json:"version"`
}

func (r *Repository) GetMappingTargetFields(ctx context.Context) ([]MappingTargetField, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id::text, topic, field_name, data_type, is_required, encrypted, version FROM mapping_target_field ORDER BY topic ASC, field_name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []MappingTargetField
	for rows.Next() {
		var m MappingTargetField
		if err := rows.Scan(&m.ID, &m.Topic, &m.FieldName, &m.DataType, &m.IsRequired, &m.Encrypted, &m.Version); err != nil {
			return nil, err
		}
		res = append(res, m)
	}
	return res, nil
}

func (r *Repository) UpsertMappingTargetField(ctx context.Context, m MappingTargetField) error {
	query := `
		INSERT INTO mapping_target_field (id, topic, field_name, data_type, is_required, encrypted, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			topic = EXCLUDED.topic,
			field_name = EXCLUDED.field_name,
			data_type = EXCLUDED.data_type,
			is_required = EXCLUDED.is_required,
			encrypted = EXCLUDED.encrypted,
			version = EXCLUDED.version
	`
	_, err := r.Pool.Exec(ctx, query, m.ID, m.Topic, m.FieldName, m.DataType, m.IsRequired, m.Encrypted, m.Version)
	return err
}

func (r *Repository) DeleteMappingTargetField(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM mapping_target_field WHERE id = $1", id)
	return err
}

type MappingRule struct {
	ID                  string          `json:"id"`
	SourceID            string          `json:"source_id"`
	TargetFieldID       string          `json:"target_field_id"`
	SourceField         string          `json:"source_field"`
	Priority            int             `json:"priority"`
	TransformationChain json.RawMessage `json:"transformation_chain"`
	ValidationChain     json.RawMessage `json:"validation_chain"`
	Version             int             `json:"version"`
}

func (r *Repository) GetMappingRules(ctx context.Context) ([]MappingRule, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id::text, source_id::text, target_field_id::text, source_field, priority, transformation_chain, validation_chain, version FROM mapping_rule ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []MappingRule
	for rows.Next() {
		var m MappingRule
		if err := rows.Scan(&m.ID, &m.SourceID, &m.TargetFieldID, &m.SourceField, &m.Priority, &m.TransformationChain, &m.ValidationChain, &m.Version); err != nil {
			return nil, err
		}
		res = append(res, m)
	}
	return res, nil
}

func (r *Repository) UpsertMappingRule(ctx context.Context, m MappingRule) error {
	query := `
		INSERT INTO mapping_rule (id, source_id, target_field_id, source_field, priority, transformation_chain, validation_chain, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			source_id = EXCLUDED.source_id,
			target_field_id = EXCLUDED.target_field_id,
			source_field = EXCLUDED.source_field,
			priority = EXCLUDED.priority,
			transformation_chain = EXCLUDED.transformation_chain,
			validation_chain = EXCLUDED.validation_chain,
			version = EXCLUDED.version
	`
	_, err := r.Pool.Exec(ctx, query, m.ID, m.SourceID, m.TargetFieldID, m.SourceField, m.Priority, m.TransformationChain, m.ValidationChain, m.Version)
	return err
}

func (r *Repository) DeleteMappingRule(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM mapping_rule WHERE id = $1", id)
	return err
}

type MappingTransformation struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Version     int             `json:"version"`
}

func (r *Repository) GetMappingTransformations(ctx context.Context) ([]MappingTransformation, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id::text, name, description, parameters, version FROM mapping_transformation ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []MappingTransformation
	for rows.Next() {
		var m MappingTransformation
		var desc *string
		if err := rows.Scan(&m.ID, &m.Name, &desc, &m.Parameters, &m.Version); err != nil {
			return nil, err
		}
		if desc != nil {
			m.Description = *desc
		}
		res = append(res, m)
	}
	return res, nil
}

func (r *Repository) UpsertMappingTransformation(ctx context.Context, m MappingTransformation) error {
	query := `
		INSERT INTO mapping_transformation (id, name, description, parameters, version)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			parameters = EXCLUDED.parameters,
			version = EXCLUDED.version
	`
	_, err := r.Pool.Exec(ctx, query, m.ID, m.Name, m.Description, m.Parameters, m.Version)
	return err
}

func (r *Repository) DeleteMappingTransformation(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM mapping_transformation WHERE id = $1", id)
	return err
}

type MappingValidation struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Version     int             `json:"version"`
}

func (r *Repository) GetMappingValidations(ctx context.Context) ([]MappingValidation, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id::text, name, description, parameters, version FROM mapping_validation ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []MappingValidation
	for rows.Next() {
		var m MappingValidation
		var desc *string
		if err := rows.Scan(&m.ID, &m.Name, &desc, &m.Parameters, &m.Version); err != nil {
			return nil, err
		}
		if desc != nil {
			m.Description = *desc
		}
		res = append(res, m)
	}
	return res, nil
}

func (r *Repository) UpsertMappingValidation(ctx context.Context, m MappingValidation) error {
	query := `
		INSERT INTO mapping_validation (id, name, description, parameters, version)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			parameters = EXCLUDED.parameters,
			version = EXCLUDED.version
	`
	_, err := r.Pool.Exec(ctx, query, m.ID, m.Name, m.Description, m.Parameters, m.Version)
	return err
}

func (r *Repository) DeleteMappingValidation(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM mapping_validation WHERE id = $1", id)
	return err
}

type TransformationError struct {
	ID            string    `json:"id"`
	CorrelationID string    `json:"correlation_id"`
	Topic         string    `json:"topic"`
	FailedField   string    `json:"failed_field"`
	RuleName      string    `json:"rule_name"`
	ErrorMessage  string    `json:"error_message"`
	CreatedAt     time.Time `json:"created_at"`
}

func (r *Repository) GetTransformationErrors(ctx context.Context, limit int) ([]TransformationError, error) {
	query := `
		SELECT 
			te.id::text, 
			te.correlation_id::text, 
			'' AS topic, 
			te.failed_field, 
			te.rule_name, 
			te.error_message, 
			te.created_at
		FROM transformation_errors te
		ORDER BY te.created_at DESC
		LIMIT $1
	`
	rows, err := r.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []TransformationError
	for rows.Next() {
		var e TransformationError
		if err := rows.Scan(&e.ID, &e.CorrelationID, &e.Topic, &e.FailedField, &e.RuleName, &e.ErrorMessage, &e.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, e)
	}
	if res == nil {
		res = []TransformationError{}
	}
	return res, nil
}

type TopicDependency struct {
	Topic           string   `json:"topic"`
	RequiredSources []string `json:"required_sources"`
}

func (r *Repository) GetTopicDependencies(ctx context.Context) ([]TopicDependency, error) {
	rows, err := r.Pool.Query(ctx, "SELECT topic, required_sources FROM topic_dependencies ORDER BY topic ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []TopicDependency
	for rows.Next() {
		var td TopicDependency
		if err := rows.Scan(&td.Topic, &td.RequiredSources); err != nil {
			return nil, err
		}
		res = append(res, td)
	}
	return res, nil
}

func (r *Repository) UpsertTopicDependency(ctx context.Context, t TopicDependency) error {
	query := `
		INSERT INTO topic_dependencies (topic, required_sources)
		VALUES ($1, $2)
		ON CONFLICT (topic) DO UPDATE SET
			required_sources = EXCLUDED.required_sources
	`
	_, err := r.Pool.Exec(ctx, query, t.Topic, t.RequiredSources)
	return err
}

func (r *Repository) DeleteTopicDependency(ctx context.Context, topic string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM topic_dependencies WHERE topic = $1", topic)
	return err
}

