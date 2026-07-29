package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	adminUser, ok := s.authenticate(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify user has UPLOADER or ADMIN role
	roleNames, err := s.Repo.GetUserRolesByUsername(r.Context(), adminUser, s.KEK)
	if err != nil {
		http.Error(w, "Error fetching user roles", http.StatusInternalServerError)
		return
	}

	isUploader := false
	for _, r := range roleNames {
		if r == "UPLOADER" || r == "ADMIN" {
			isUploader = true
			break
		}
	}

	if !isUploader {
		http.Error(w, "Forbidden: UPLOADER role required", http.StatusForbidden)
		return
	}

	// Parse multipart form data
	// 32 MB max memory
	err = r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Error parsing form: "+err.Error(), http.StatusBadRequest)
		return
	}

	topic := r.FormValue("topic")
	if topic == "" {
		http.Error(w, "Missing 'topic' form field", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	if s.UploadDir == "" {
		http.Error(w, "Upload directory not configured on server", http.StatusInternalServerError)
		return
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(s.UploadDir, os.ModePerm); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Save file with timestamp to avoid collisions
	destPath := filepath.Join(s.UploadDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename))
	destFile, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Failed to create file on disk", http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, file); err != nil {
		http.Error(w, "Failed to save file content", http.StatusInternalServerError)
		return
	}

	// Log the audit
	s.Repo.LogAdminAction(r.Context(), adminUser, "UPLOAD_FILE", map[string]string{"file": header.Filename, "topic": topic})

	// Trigger the file collector immediately
	go s.triggerFileCollector(destPath, topic)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "File uploaded and collector triggered"})
}

func (s *Server) triggerFileCollector(filePath, topic string) {
	// The binary should be available in the same directory as the scheduler.
	executablePath, err := os.Executable()
	var collectorBin string
	if err == nil {
		collectorBin = filepath.Join(filepath.Dir(executablePath), "mitm-collector-csv-xls")
	} else {
		collectorBin = "./bin/mitm-collector-csv-xls"
	}

	if _, err := os.Stat(collectorBin); os.IsNotExist(err) {
		collectorBin = "mitm-collector-csv-xls" // fallback to PATH if all else fails
	}

	argsMap := map[string]string{
		"file":  filePath,
		"topic": topic,
	}
	argsJSON, _ := json.Marshal(argsMap)

	// Delegate to the scheduler engine which handles DB injection, IPC, and RUN_ID
	s.Scheduler.RunImmediateJob(context.Background(), collectorBin, argsJSON)
}
