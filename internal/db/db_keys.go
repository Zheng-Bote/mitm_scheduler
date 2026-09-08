/**
 * SPDX-FileComment: Database functions for Storage Keys
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: Antigravity
 * SPDX-FileCopyrightText: 2026 Antigravity
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file db_keys.go
 * @brief Repository methods for fetching wrapped keys for the admin frontend
 * @version 1.0.0
 * @date 2026-09-08
 *
 * @author Antigravity
 * @copyright Copyright (c) 2026 Antigravity
 * @/home/zb_bamboo/DEV/__NEW__/Go/mitm-2/LICENSE Apache-2.0
 */

package db

import (
	"context"
	"fmt"
)

// GetAllActiveWrappedKeys retrieves all unique wrapped_keys from storage_keys 
// and wrapped_dek from user_roles_encrypted.
func (r *Repository) GetAllActiveWrappedKeys(ctx context.Context) ([]string, error) {
	var keys []string

	// 1. Fetch from storage_keys
	queryStorage := `SELECT wrapped_key FROM storage_keys WHERE is_active = true`
	rowsStorage, err := r.Pool.Query(ctx, queryStorage)
	if err != nil {
		return nil, fmt.Errorf("failed to query storage_keys: %w", err)
	}
	defer rowsStorage.Close()

	for rowsStorage.Next() {
		var key string
		if err := rowsStorage.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan storage_keys row: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rowsStorage.Err(); err != nil {
		return nil, fmt.Errorf("storage_keys rows error: %w", err)
	}

	// 2. Fetch from user_roles_encrypted
	queryUsers := `SELECT wrapped_dek FROM user_roles_encrypted`
	rowsUsers, err := r.Pool.Query(ctx, queryUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to query user_roles_encrypted: %w", err)
	}
	defer rowsUsers.Close()

	for rowsUsers.Next() {
		var key string
		if err := rowsUsers.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan user_roles_encrypted row: %w", err)
		}
		// Avoid duplicates if needed, though they shouldn't be duplicated usually,
		// or just append them all and the client will iterate over all keys.
		// For safety and efficiency on client side, we can deduplicate here.
		keys = append(keys, key)
	}

	if err := rowsUsers.Err(); err != nil {
		return nil, fmt.Errorf("user_roles_encrypted rows error: %w", err)
	}

	// Deduplicate keys
	uniqueKeys := make([]string, 0, len(keys))
	seen := make(map[string]bool)
	for _, key := range keys {
		if !seen[key] {
			seen[key] = true
			uniqueKeys = append(uniqueKeys, key)
		}
	}

	return uniqueKeys, nil
}
