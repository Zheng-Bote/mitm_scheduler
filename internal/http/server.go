/**
 * SPDX-FileComment: HTTP Server
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file server.go
 * @brief HTTP API server with health, info, and admin endpoints
 * @version 1.0.0
 * @date 2026-06-02
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

// Package http provides an HTTP API server for the scheduler. It exposes
// health-check, service-info, and authenticated admin endpoints for managing
// scheduled jobs.
package http

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"go-scheduler/internal/config"
	"go-scheduler/internal/crypto"
	"go-scheduler/internal/db"
)

// SchedulerInterface allows the HTTP server to trigger reloads
type SchedulerInterface interface {
	Reload(ctx context.Context) error
	RunImmediateJob(ctx context.Context, command string, args []byte)
	GetActivePIDs() map[string]int
	StopJobByName(name string) error
}

type Server struct {
	Repo           *db.Repository
	Port           int
	UseHTTPS       bool
	SSLCert        string
	SSLKey         string
	Admins         []config.AdminUser
	KEK            []byte
	UploadDir      string
	Scheduler      SchedulerInterface
	AppName        string
	AppDescription string
	AppVersion     string
}

// Start runs the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			s.handleInfo(w, r)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/info", s.handleInfo)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/time", s.handleTime)
	mux.HandleFunc("/admin/update-jobs", s.handleUpdateJobs)
	mux.HandleFunc("/admin/jobs", s.handleGetJobs)
	mux.HandleFunc("/admin/delete-job", s.handleDeleteJob)
	mux.HandleFunc("/admin/stop-job", s.handleStopJob)
	mux.HandleFunc("/admin/upload/source_file", s.handleUploadFile)
	mux.HandleFunc("/admin/logs/system", s.handleDownloadSystemLogs)
	mux.HandleFunc("/admin/logs/system_bin", s.handleDownloadSystemLogsBin)
	mux.HandleFunc("/admin/logs/job-audit", s.handleDownloadJobAuditLogs)
	mux.HandleFunc("/admin/logs/job-audit_bin", s.handleDownloadJobAuditLogsBin)
	mux.HandleFunc("/admin/logs/admin-audit", s.handleDownloadAdminAuditLogs)
	mux.HandleFunc("/admin/logs/admin-audit_bin", s.handleDownloadAdminAuditLogsBin)
	mux.HandleFunc("/admin/credentials", s.handleCredentials)
	mux.HandleFunc("/admin/delivery_targets", s.handleDeliveryTargets)
	mux.HandleFunc("/admin/dlq", s.handleDLQ)
	mux.HandleFunc("/admin/dlq_bin", s.handleDLQBin)
	mux.HandleFunc("/admin/backup", s.handleBackup)
	mux.HandleFunc("/admin/restore", s.handleRestore)

	// RBAC routes
	mux.HandleFunc("/admin/rbac/roles", s.handleGetRoles)
	mux.HandleFunc("/admin/rbac/users", s.handleGetUsers)
	mux.HandleFunc("/admin/rbac/user/create", s.handleCreateUser)
	mux.HandleFunc("/admin/rbac/user/delete", s.handleDeleteUser)
	mux.HandleFunc("/admin/rbac/assign", s.handleAssignRoles)
	mux.HandleFunc("/admin/rbac/user_roles", s.handleGetUserRoles)
	mux.HandleFunc("/admin/rbac/os_user_roles", s.handleGetOsUserRoles)

	// Transformation layer routes
	mux.HandleFunc("/admin/transformation/sources", s.handleMappingSources)
	mux.HandleFunc("/admin/transformation/targets", s.handleMappingTargets)
	mux.HandleFunc("/admin/transformation/rules", s.handleMappingRules)
	mux.HandleFunc("/admin/transformation/transformations", s.handleMappingTransformations)
	mux.HandleFunc("/admin/transformation/validations", s.handleMappingValidations)
	mux.HandleFunc("/admin/transformation/auto-map", s.handleAutoMap)
	mux.HandleFunc("/admin/transformation/errors", s.handleTransformationErrors)
	mux.HandleFunc("/admin/transformation/errors_bin", s.handleTransformationErrorsBin)
	mux.HandleFunc("/admin/transformation/topic-dependencies", s.handleTopicDependencies)
	mux.HandleFunc("/admin/action", s.handleAdminAction)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.Port),
		Handler: mux,
	}

	go func() {
		if s.UseHTTPS {
			if s.SSLCert == "" {
				s.SSLCert = "server.crt"
			}
			if s.SSLKey == "" {
				s.SSLKey = "server.key"
			}
			if err := srv.ListenAndServeTLS(s.SSLCert, s.SSLKey); err != nil && err != http.ErrServerClosed {
				fmt.Printf("ERROR: Failed to start HTTPS server (cert: %s, key: %s): %v\n", s.SSLCert, s.SSLKey, err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("ERROR: Failed to start HTTP server: %v\n", err)
			}
		}
	}()

	return nil
}

// authenticate validates the HTTP Basic Auth credentials against the
// configured admin users. It returns the authenticated username and true
// on success, or an empty string and false on failure.
func (s *Server) authenticate(r *http.Request) (string, bool) {
	user, token, ok := r.BasicAuth()
	if !ok {
		return "", false
	}

	for _, admin := range s.Admins {
		if admin.Username == user && admin.Token == token {
			return user, true
		}
	}
	// Fallback to DB check
	var hashStr string
	err := s.Repo.Pool.QueryRow(r.Context(), "SELECT password_hash FROM admin_users WHERE username = $1 AND is_active = true", user).Scan(&hashStr)
	if err == nil {
		// hashStr is base64(salt):base64(hash)
		parts := strings.Split(hashStr, ":")
		if len(parts) == 2 {
			salt, _ := base64.StdEncoding.DecodeString(parts[0])
			expectedHash, _ := base64.StdEncoding.DecodeString(parts[1])

			// derive hash from provided token/password
			actualHash := crypto.DeriveKey([]byte(token), salt)
			if bytes.Equal(actualHash, expectedHash) {
				return user, true
			}
		}
	}

	return "", false
}

// handleAdminAction allows frontend to explicitly log an action
func (s *Server) handleAdminAction(w http.ResponseWriter, r *http.Request) {
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

	var payload struct {
		Action  string      `json:"action"`
		Details interface{} `json:"details"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.Repo.LogAdminAction(r.Context(), username, payload.Action, payload.Details)
	w.WriteHeader(http.StatusOK)
}

// handleUpdateJobs accepts a POST request with a JSON array of scheduled
// programs, upserts them into the database, and triggers a scheduler reload.
// Requires valid HTTP Basic Auth credentials.
func (s *Server) handleUpdateJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var jobs []db.ScheduledProgram
	if err := json.NewDecoder(r.Body).Decode(&jobs); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if s.Repo == nil {
		http.Error(w, "Service Unavailable: Database connection missing", http.StatusServiceUnavailable)
		return
	}

	ctx := context.Background()
	for _, job := range jobs {
		if err := s.Repo.UpsertScheduledProgram(ctx, job); err != nil {
			s.Repo.LogAdminAction(ctx, username, "update_job_fail", map[string]interface{}{"job": job.Name, "error": err.Error()})
			http.Error(w, fmt.Sprintf("Failed to update job %s: %v", job.Name, err), http.StatusInternalServerError)
			return
		}
	}

	if err := s.Scheduler.Reload(ctx); err != nil {
		s.Repo.LogAdminAction(ctx, username, "reload_fail", err.Error())
		http.Error(w, "Failed to reload scheduler", http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(ctx, username, "update_jobs_success", map[string]interface{}{"count": len(jobs)})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Jobs updated and scheduler reloaded"))
}

// handleGetJobs returns all scheduled programs as a JSON array. Requires
// valid HTTP Basic Auth credentials.
func (s *Server) handleGetJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if s.Repo == nil {
		http.Error(w, "Service Unavailable: Database connection missing", http.StatusServiceUnavailable)
		return
	}

	jobs, err := s.Repo.GetAllPrograms(context.Background())
	if err != nil {
		http.Error(w, "Failed to fetch jobs", http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(context.Background(), username, "get_jobs", nil)

	type JobResponse struct {
		db.ScheduledProgram
		NextRun   string `json:"next_run"`
		IsRunning bool   `json:"is_running"`
		ActivePID int    `json:"active_pid,omitempty"`
	}
	var res []JobResponse
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	now := time.Now()
	activePIDs := s.Scheduler.GetActivePIDs()
	for _, j := range jobs {
		nextRun := ""
		if j.Enabled {
			schedule, err := parser.Parse(j.CronExpr)
			if err == nil {
				nextRun = schedule.Next(now).Format(time.RFC3339)
			}
		}
		isRunning := false
		activePID := 0
		if pid, ok := activePIDs[j.Name]; ok {
			isRunning = true
			activePID = pid
		}
		res = append(res, JobResponse{
			ScheduledProgram: j,
			NextRun:          nextRun,
			IsRunning:        isRunning,
			ActivePID:        activePID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// handleDeleteJob deletes a scheduled program by name (query param "name").
// Requires valid HTTP Basic Auth credentials.
func (s *Server) handleDeleteJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing job name", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	_, err := s.Repo.Pool.Exec(ctx, "DELETE FROM scheduled_programs WHERE name = $1", name)
	if err != nil {
		http.Error(w, "Failed to delete job", http.StatusInternalServerError)
		return
	}

	_ = s.Scheduler.Reload(ctx)
	s.Repo.LogAdminAction(ctx, username, "delete_job", map[string]string{"name": name})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Job deleted"))
}

// handleStopJob stops a currently running scheduled program by name (query param "name").
// Requires valid HTTP Basic Auth credentials.
func (s *Server) handleStopJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify user has ADMIN role (either via config or DB RBAC roles)
	isAdmin := false
	for _, admin := range s.Admins {
		if admin.Username == username {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		roleNames, err := s.Repo.GetUserRolesByUsername(r.Context(), username, s.KEK)
		if err == nil {
			for _, role := range roleNames {
				if role == "ADMIN" {
					isAdmin = true
					break
				}
			}
		}
	}
	if !isAdmin {
		s.Repo.LogAdminAction(r.Context(), username, "stop_job_forbidden", map[string]string{"name": r.URL.Query().Get("name")})
		http.Error(w, "Forbidden: ADMIN role required to stop jobs", http.StatusForbidden)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Missing job name", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	if err := s.Scheduler.StopJobByName(name); err != nil {
		s.Repo.LogAdminAction(ctx, username, "stop_job_fail", map[string]string{"name": name, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.Repo.LogAdminAction(ctx, username, "stop_job", map[string]string{"name": name})
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Stop signal sent to job"))
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]string{
		"name":        s.AppName,
		"description": s.AppDescription,
		"version":     s.AppVersion,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleTime returns the server's local time as JSON.
// This endpoint is unauthenticated.
func (s *Server) handleTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	res := map[string]interface{}{
		"local_time": now.Format(time.RFC3339),
		"timestamp":  now.Unix(),
		"timezone":   now.Location().String(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// handleHealth performs a database ping and returns HTTP 200 if the
// database is reachable, or HTTP 500 with the error message otherwise.
// This endpoint is unauthenticated.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if s.Repo == nil || s.Repo.Pool == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "DB Connecting...")
		return
	}

	if err := s.Repo.Pool.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "DB Error: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// parseDateParam parses a date string from RFC3339 or YYYY-MM-DD format.
// Returns nil, nil if the input string is empty.
func parseDateParam(val string) (*time.Time, error) {
	if val == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, val)
	if err == nil {
		return &t, nil
	}
	t, err = time.Parse("2006-01-02", val)
	if err == nil {
		return &t, nil
	}
	return nil, err
}

// handleDownloadSystemLogs retrieves system logs within an optional date range
// and streams them as a JSON file download.
func (s *Server) handleDownloadSystemLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	from, err := parseDateParam(r.URL.Query().Get("from"))
	if err != nil {
		http.Error(w, "Invalid 'from' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	to, err := parseDateParam(r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, "Invalid 'to' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	if to != nil && len(r.URL.Query().Get("to")) == 10 {
		*to = to.Add(24*time.Hour - time.Second)
	}

	logs, err := s.Repo.GetSystemLogs(r.Context(), from, to)
	if err != nil {
		s.Repo.LogAdminAction(r.Context(), username, "download_system_logs_fail", err.Error())
		http.Error(w, "Failed to retrieve system logs", http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), username, "download_system_logs_success", map[string]interface{}{
		"from":  from,
		"to":    to,
		"count": len(logs),
	})

	w.Header().Set("Content-Disposition", "attachment; filename=system_logs.json")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// handleDownloadJobAuditLogs retrieves job audit logs within an optional date range
// and streams them as a JSON file download.
func (s *Server) handleDownloadJobAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	from, err := parseDateParam(r.URL.Query().Get("from"))
	if err != nil {
		http.Error(w, "Invalid 'from' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	to, err := parseDateParam(r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, "Invalid 'to' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	if to != nil && len(r.URL.Query().Get("to")) == 10 {
		*to = to.Add(24*time.Hour - time.Second)
	}

	logs, err := s.Repo.GetJobAuditLogs(r.Context(), from, to)
	if err != nil {
		s.Repo.LogAdminAction(r.Context(), username, "download_job_audit_logs_fail", err.Error())
		http.Error(w, "Failed to retrieve job audit logs", http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), username, "download_job_audit_logs_success", map[string]interface{}{
		"from":  from,
		"to":    to,
		"count": len(logs),
	})

	w.Header().Set("Content-Disposition", "attachment; filename=job_audit_logs.json")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// handleDownloadAdminAuditLogs retrieves administrative audit logs within an optional date range
// and streams them as a JSON file download.
func (s *Server) handleDownloadAdminAuditLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	from, err := parseDateParam(r.URL.Query().Get("from"))
	if err != nil {
		http.Error(w, "Invalid 'from' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	to, err := parseDateParam(r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, "Invalid 'to' parameter. Use RFC3339 or YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	if to != nil && len(r.URL.Query().Get("to")) == 10 {
		*to = to.Add(24*time.Hour - time.Second)
	}

	logs, err := s.Repo.GetAdminAuditLogs(r.Context(), from, to)
	if err != nil {
		s.Repo.LogAdminAction(r.Context(), username, "download_admin_audit_logs_fail", err.Error())
		http.Error(w, "Failed to retrieve admin audit logs", http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), username, "download_admin_audit_logs_success", map[string]interface{}{
		"from":  from,
		"to":    to,
		"count": len(logs),
	})

	w.Header().Set("Content-Disposition", "attachment; filename=admin_audit_logs.json")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// handleCredentials returns or updates source credentials. Requires valid HTTP Basic Auth.
func (s *Server) handleCredentials(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		creds, err := s.Repo.GetSourceCredentials(r.Context())
		if err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "get_credentials_fail", err.Error())
			http.Error(w, "Failed to fetch source credentials", http.StatusInternalServerError)
			return
		}

		s.Repo.LogAdminAction(r.Context(), username, "get_credentials_success", map[string]interface{}{
			"count": len(creds),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(creds)
		return
	}

	if r.Method == http.MethodPost {
		var cred db.SourceCredential
		if err := json.NewDecoder(r.Body).Decode(&cred); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := s.Repo.UpsertSourceCredential(r.Context(), cred); err != nil {
			fmt.Printf("[HTTP ERROR] Failed to upsert credential '%s': %v\n", cred.SourceName, err)
			s.Repo.LogAdminAction(r.Context(), username, "upsert_credential_fail", map[string]interface{}{"source": cred.SourceName, "error": err.Error()})
			http.Error(w, fmt.Sprintf("Failed to update credential %s: %v", cred.SourceName, err), http.StatusInternalServerError)
			return
		}

		s.Repo.LogAdminAction(r.Context(), username, "upsert_credential_success", map[string]interface{}{"source": cred.SourceName})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Credential updated"))
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleDeliveryTargets returns or updates delivery targets. Requires valid HTTP Basic Auth.
func (s *Server) handleDeliveryTargets(w http.ResponseWriter, r *http.Request) {
	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Admin API"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet {
		targets, err := s.Repo.GetDeliveryTargets(r.Context())
		if err != nil {
			s.Repo.LogAdminAction(r.Context(), username, "get_delivery_targets_fail", err.Error())
			http.Error(w, "Failed to fetch delivery targets", http.StatusInternalServerError)
			return
		}

		s.Repo.LogAdminAction(r.Context(), username, "get_delivery_targets_success", map[string]interface{}{
			"count": len(targets),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(targets)
		return
	}

	if r.Method == http.MethodPost {
		var target db.DeliveryTarget
		if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := s.Repo.UpsertDeliveryTarget(r.Context(), target); err != nil {
			fmt.Printf("[HTTP ERROR] Failed to upsert delivery target '%s': %v\n", target.Topic, err)
			s.Repo.LogAdminAction(r.Context(), username, "upsert_delivery_target_fail", map[string]interface{}{"topic": target.Topic, "error": err.Error()})
			http.Error(w, fmt.Sprintf("Failed to update delivery target %s: %v", target.Topic, err), http.StatusInternalServerError)
			return
		}

		s.Repo.LogAdminAction(r.Context(), username, "upsert_delivery_target_success", map[string]interface{}{"topic": target.Topic})
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Delivery target updated"))
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing id", http.StatusBadRequest)
			return
		}

		if err := s.Repo.DeleteDeliveryTarget(r.Context(), id); err != nil {
			http.Error(w, "Failed to delete target", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Target deleted"))
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handleDLQ handles GET requests for dead letter queue entries
func (s *Server) handleDLQ(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := s.Repo.GetDLQEntries(r.Context(), 100)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to fetch DLQ entries: %v", err)))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Failed to encode DLQ entries: %v", err)))
	}
}
