package db

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"go-scheduler/internal/crypto"
)

// DeliveryTarget represents a target connection config for the delivery layer
type DeliveryTarget struct {
	ID            string    `json:"id"`
	Topic         string    `json:"topic"`
	AdapterType   string    `json:"adapter_type"`
	EndpointURL   string    `json:"endpoint_url"`
	ConfigPayload string    `json:"config_payload"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GetDeliveryTargets fetches all delivery targets and decrypts their config_payload
func (r *Repository) GetDeliveryTargets(ctx context.Context) ([]DeliveryTarget, error) {
	query := `
		SELECT 
			dt.id::text, dt.topic, dt.adapter_type, dt.endpoint_url, dt.is_active, dt.created_at, dt.updated_at,
			dt.config_payload, dt.nonce, sk.wrapped_key
		FROM delivery_targets dt
		JOIN storage_keys sk ON dt.dek_id = sk.id
		ORDER BY dt.topic ASC
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

	var targets []DeliveryTarget
	targets = make([]DeliveryTarget, 0)
	for rows.Next() {
		var t DeliveryTarget
		var payload, nonce, wrappedKey []byte
		var isActive *bool
		var createdAt, updatedAt *time.Time

		if err := rows.Scan(&t.ID, &t.Topic, &t.AdapterType, &t.EndpointURL, &isActive, &createdAt, &updatedAt, &payload, &nonce, &wrappedKey); err != nil {
			fmt.Printf("[DB ERROR] GetDeliveryTargets Scan failed: %v\n", err)
			return nil, err
		}

		if isActive != nil {
			t.IsActive = *isActive
		}
		if createdAt != nil {
			t.CreatedAt = *createdAt
		}
		if updatedAt != nil {
			t.UpdatedAt = *updatedAt
		}

		decrypted, err := crypto.EnvelopeDecrypt(kek, wrappedKey, nonce, payload)
		if err == nil {
			t.ConfigPayload = string(decrypted)
		} else {
			t.ConfigPayload = fmt.Sprintf(`{"error": "decryption failed: %v"}`, err)
		}

		targets = append(targets, t)
	}
	return targets, nil
}

// UpsertDeliveryTarget inserts or updates a delivery target
func (r *Repository) UpsertDeliveryTarget(ctx context.Context, t DeliveryTarget) error {
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

	encryptedPayload, nonce, err := crypto.EnvelopeEncrypt(kek, wrappedKey, []byte(t.ConfigPayload))
	if err != nil {
		return fmt.Errorf("encryption failed: %v", err)
	}

	query := `
		INSERT INTO delivery_targets (topic, adapter_type, endpoint_url, config_payload, nonce, dek_id, is_active, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
		ON CONFLICT (topic) DO UPDATE SET
			adapter_type = EXCLUDED.adapter_type,
			endpoint_url = EXCLUDED.endpoint_url,
			config_payload = EXCLUDED.config_payload,
			nonce = EXCLUDED.nonce,
			dek_id = EXCLUDED.dek_id,
			is_active = EXCLUDED.is_active,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err = r.Pool.Exec(ctx, query, t.Topic, t.AdapterType, t.EndpointURL, encryptedPayload, nonce, dekID, t.IsActive)
	return err
}

// DeleteDeliveryTarget deletes a delivery target by ID
func (r *Repository) DeleteDeliveryTarget(ctx context.Context, id string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM delivery_targets WHERE id = $1", id)
	return err
}
