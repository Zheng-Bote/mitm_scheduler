/**
 * SPDX-FileComment: HTTP handlers for Storage Keys
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: Antigravity
 * SPDX-FileCopyrightText: 2026 Antigravity
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file server_keys.go
 * @brief HTTP endpoint for returning all active wrapped keys for Envelope Encryption
 * @version 1.0.0
 * @date 2026-09-08
 *
 * @author Antigravity
 * @copyright Copyright (c) 2026 Antigravity
 * @/home/zb_bamboo/DEV/__NEW__/Go/mitm-2/LICENSE Apache-2.0
 */

package http

import (
	"encoding/json"
	"log"
	"net/http"
)

// handleGetStorageKeys returns a list of all currently active wrapped keys
func (s *Server) handleGetStorageKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		writeJSONError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	keys, err := s.Repo.GetAllActiveWrappedKeys(r.Context())
	if err != nil {
		log.Printf("Failed to get active wrapped keys: %v", err)
		writeJSONError(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Ensure keys is an empty array rather than null if empty
	if keys == nil {
		keys = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(keys); err != nil {
		log.Printf("Failed to encode keys response: %v", err)
	}
}
