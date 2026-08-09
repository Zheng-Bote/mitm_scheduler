package http

import (
	"encoding/json"
	"net/http"
)

// handleBackup exports configuration as JSON
func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	roles, err := s.Repo.GetUserRolesByUsername(r.Context(), username, s.KEK)
	hasRole := false
	for _, role := range roles {
		if role == "BACKUP-RESTORE" || role == "ADMIN" {
			hasRole = true
			break
		}
	}
	if err != nil || !hasRole {
		http.Error(w, "Forbidden: Missing BACKUP-RESTORE role", http.StatusForbidden)
		return
	}

	tables := []string{
		"scheduled_programs",
		"source_credentials",
		"delivery_targets",
		"mapping_transformation",
		"mapping_rule",
		"mapping_validation",
		"mapping_source",
		"mapping_target_field",
		"topic_dependencies",
	}

	backupData := make(map[string][]json.RawMessage)
	for _, t := range tables {
		data, err := s.Repo.ExportConfigTable(r.Context(), t)
		if err != nil {
			http.Error(w, "Failed to export table "+t+": "+err.Error(), http.StatusInternalServerError)
			return
		}
		backupData[t] = data
	}

	s.Repo.LogAdminAction(r.Context(), username, "BACKUP_CONFIG", nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(backupData)
}

// handleRestore imports configuration from JSON
func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	roles, err := s.Repo.GetUserRolesByUsername(r.Context(), username, s.KEK)
	hasRole := false
	for _, role := range roles {
		if role == "BACKUP-RESTORE" || role == "ADMIN" {
			hasRole = true
			break
		}
	}
	if err != nil || !hasRole {
		http.Error(w, "Forbidden: Missing BACKUP-RESTORE role", http.StatusForbidden)
		return
	}

	var backupData map[string][]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&backupData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Important: Order of insertion must respect foreign key constraints
	tables := []string{
		"scheduled_programs",
		"source_credentials",
		"delivery_targets",
		"mapping_transformation",
		"mapping_rule",
		"mapping_validation",
		"mapping_source",
		"mapping_target_field",
		"topic_dependencies",
	}

	for _, t := range tables {
		if data, ok := backupData[t]; ok {
			if err := s.Repo.ImportConfigTable(r.Context(), t, data); err != nil {
				http.Error(w, "Failed to restore table "+t+": "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// Trigger a scheduler reload if applicable
	s.Scheduler.Reload(r.Context())

	s.Repo.LogAdminAction(r.Context(), username, "RESTORE_CONFIG", nil)
	w.WriteHeader(http.StatusOK)
}
