package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go-scheduler/internal/db"
)

func (s *Server) handleMappingSources(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		res, err := s.Repo.GetMappingSources(r.Context())
		if err != nil {
			http.Error(w, "Failed to fetch mapping sources", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
		return
	}

	if r.Method == http.MethodPost {
		var m db.MappingSource
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if m.ID == "" {
			m.ID = uuid.New().String()
		}
		if m.Version == 0 {
			m.Version = 1
		}
		if err := s.Repo.UpsertMappingSource(r.Context(), m); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "upsert_mapping_source_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to update mapping source: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "upsert_mapping_source_success", map[string]interface{}{"id": m.ID})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Mapping source updated"))
		return
	}
	
	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing ID", http.StatusBadRequest)
			return
		}
		if err := s.Repo.DeleteMappingSource(r.Context(), id); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "delete_mapping_source_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to delete: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "delete_mapping_source_success", map[string]interface{}{"id": id})
		w.WriteHeader(http.StatusOK)
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleMappingTargets(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		res, err := s.Repo.GetMappingTargetFields(r.Context())
		if err != nil {
			http.Error(w, "Failed to fetch target fields", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
		return
	}

	if r.Method == http.MethodPost {
		var m db.MappingTargetField
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if m.ID == "" {
			m.ID = uuid.New().String()
		}
		if m.Version == 0 {
			m.Version = 1
		}
		if err := s.Repo.UpsertMappingTargetField(r.Context(), m); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "upsert_target_field_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to update target field: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "upsert_target_field_success", map[string]interface{}{"id": m.ID})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Target field updated"))
		return
	}
	
	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing ID", http.StatusBadRequest)
			return
		}
		if err := s.Repo.DeleteMappingTargetField(r.Context(), id); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "delete_target_field_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to delete: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "delete_target_field_success", map[string]interface{}{"id": id})
		w.WriteHeader(http.StatusOK)
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleMappingRules(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		res, err := s.Repo.GetMappingRules(r.Context())
		if err != nil {
			http.Error(w, "Failed to fetch rules", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
		return
	}

	if r.Method == http.MethodPost {
		var m db.MappingRule
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if m.ID == "" {
			m.ID = uuid.New().String()
		}
		if m.Version == 0 {
			m.Version = 1
		}
		if err := s.Repo.UpsertMappingRule(r.Context(), m); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "upsert_rule_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to update rule: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "upsert_rule_success", map[string]interface{}{"id": m.ID})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Rule updated"))
		return
	}
	
	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing ID", http.StatusBadRequest)
			return
		}
		if err := s.Repo.DeleteMappingRule(r.Context(), id); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "delete_rule_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to delete: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "delete_rule_success", map[string]interface{}{"id": id})
		w.WriteHeader(http.StatusOK)
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleMappingTransformations(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		res, err := s.Repo.GetMappingTransformations(r.Context())
		if err != nil {
			http.Error(w, "Failed to fetch transformations", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
		return
	}

	if r.Method == http.MethodPost {
		var m db.MappingTransformation
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if m.ID == "" {
			m.ID = uuid.New().String()
		}
		if m.Version == 0 {
			m.Version = 1
		}
		if err := s.Repo.UpsertMappingTransformation(r.Context(), m); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "upsert_transformation_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to update transformation: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "upsert_transformation_success", map[string]interface{}{"id": m.ID})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Transformation updated"))
		return
	}
	
	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing ID", http.StatusBadRequest)
			return
		}
		if err := s.Repo.DeleteMappingTransformation(r.Context(), id); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "delete_transformation_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to delete: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "delete_transformation_success", map[string]interface{}{"id": id})
		w.WriteHeader(http.StatusOK)
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleMappingValidations(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		res, err := s.Repo.GetMappingValidations(r.Context())
		if err != nil {
			http.Error(w, "Failed to fetch validations", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
		return
	}

	if r.Method == http.MethodPost {
		var m db.MappingValidation
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if m.ID == "" {
			m.ID = uuid.New().String()
		}
		if m.Version == 0 {
			m.Version = 1
		}
		if err := s.Repo.UpsertMappingValidation(r.Context(), m); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "upsert_validation_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to update validation: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "upsert_validation_success", map[string]interface{}{"id": m.ID})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Validation updated"))
		return
	}
	
	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing ID", http.StatusBadRequest)
			return
		}
		if err := s.Repo.DeleteMappingValidation(r.Context(), id); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "delete_validation_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to delete: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "delete_validation_success", map[string]interface{}{"id": id})
		w.WriteHeader(http.StatusOK)
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// levenshtein computes the Levenshtein distance between two strings
func levenshtein(a, b string) int {
	la := len(a)
	lb := len(b)
	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			min := d[i-1][j] + 1
			if d[i][j-1]+1 < min {
				min = d[i][j-1] + 1
			}
			if d[i-1][j-1]+cost < min {
				min = d[i-1][j-1] + cost
			}
			d[i][j] = min
		}
	}
	return d[la][lb]
}

func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func (s *Server) handleAutoMap(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SourceID     string   `json:"source_id"`
		SourceFields []string `json:"source_fields"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.SourceID == "" || len(req.SourceFields) == 0 {
		http.Error(w, "Missing source_id or source_fields", http.StatusBadRequest)
		return
	}

	targets, err := s.Repo.GetMappingTargetFields(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch targets", http.StatusInternalServerError)
		return
	}

	var createdRules []db.MappingRule

	for _, sf := range req.SourceFields {
		normSf := normalizeForMatch(sf)
		var bestTarget *db.MappingTargetField
		bestDist := 9999
		for i, tgt := range targets {
			normTgt := normalizeForMatch(tgt.FieldName)
			dist := levenshtein(normSf, normTgt)
			
			// Threshold: up to 3 edits allowed for string matching
			if dist < bestDist && dist <= 3 {
				bestDist = dist
				bestTarget = &targets[i]
			}
		}

		if bestTarget != nil {
			rule := db.MappingRule{
				ID:                  uuid.New().String(),
				SourceID:            req.SourceID,
				TargetFieldID:       bestTarget.ID,
				SourceField:         sf,
				Priority:            1,
				TransformationChain: []byte("[]"),
				ValidationChain:     []byte("[]"),
				Version:             1,
			}
			if err := s.Repo.UpsertMappingRule(r.Context(), rule); err == nil {
				createdRules = append(createdRules, rule)
			}
		}
	}

	s.Repo.LogAdminAction(r.Context(), username, "auto_map", map[string]interface{}{
		"source_id": req.SourceID,
		"fields_provided": len(req.SourceFields),
		"rules_created": len(createdRules),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"created": len(createdRules),
	})
}

func (s *Server) handleTransformationErrors(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	res, err := s.Repo.GetTransformationErrors(r.Context(), 500)
	if err != nil {
		http.Error(w, "Failed to fetch transformation errors", http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), username, "get_transformation_errors", map[string]interface{}{
		"count": len(res),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (s *Server) handleTopicDependencies(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		res, err := s.Repo.GetTopicDependencies(r.Context())
		if err != nil {
			http.Error(w, "Failed to fetch topic dependencies", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)
		return
	}

	if r.Method == http.MethodPost {
		var td db.TopicDependency
		if err := json.NewDecoder(r.Body).Decode(&td); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if td.Topic == "" {
			http.Error(w, "Missing Topic", http.StatusBadRequest)
			return
		}
		if td.RequiredSources == nil {
			td.RequiredSources = []string{}
		}
		if err := s.Repo.UpsertTopicDependency(r.Context(), td); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "upsert_topic_dependency_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to update topic dependency: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "upsert_topic_dependency_success", map[string]interface{}{"topic": td.Topic})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Topic dependency updated"))
		return
	}
	
	if r.Method == http.MethodDelete {
		topic := r.URL.Query().Get("topic")
		if topic == "" {
			http.Error(w, "Missing Topic", http.StatusBadRequest)
			return
		}
		if err := s.Repo.DeleteTopicDependency(r.Context(), topic); err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "delete_topic_dependency_fail", err.Error())
			http.Error(w, fmt.Sprintf("Failed to delete topic dependency: %v", err), http.StatusInternalServerError)
			return
		}
		s.Repo.LogAdminAction(r.Context(), username, "delete_topic_dependency_success", map[string]interface{}{"topic": topic})
		w.WriteHeader(http.StatusOK)
		return
	}
	
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

