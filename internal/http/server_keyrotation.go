package http

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"go-scheduler/internal/crypto"
)

// handleKeyRotation performs native key rotation.
func (s *Server) handleKeyRotation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username, ok := s.authenticate(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"Admin API\"")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var payload struct {
		Nonce      string `json:"nonce"`
		Ciphertext string `json:"ciphertext"`
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	nonceBytes, err1 := base64.StdEncoding.DecodeString(payload.Nonce)
	cipherBytes, err2 := base64.StdEncoding.DecodeString(payload.Ciphertext)

	if err1 != nil || err2 != nil {
		http.Error(w, "Invalid base64 encoding", http.StatusBadRequest)
		return
	}

	// The frontend encrypts the new KEK with the current KEK directly using AES-GCM
	kekBlock, err := aes.NewCipher(s.KEK)
	if err != nil {
		http.Error(w, "Internal crypto error", http.StatusInternalServerError)
		return
	}
	kekGCM, err := cipher.NewGCM(kekBlock)
	if err != nil {
		http.Error(w, "Internal crypto error", http.StatusInternalServerError)
		return
	}
	
	newMasterKey, err := kekGCM.Open(nil, nonceBytes, cipherBytes, nil)
	if err != nil {
		s.Repo.LogAdminAction(r.Context(), username, "key_rotation_fail", map[string]string{"error": "decryption failed"})
		http.Error(w, "Failed to decrypt payload", http.StatusBadRequest)
		return
	}

	if len(newMasterKey) != 32 {
		http.Error(w, "New master key must be 32 bytes", http.StatusBadRequest)
		return
	}

	// 4. Pause/lock job execution (wait for active jobs to finish, prevent new ones from starting).
	pauseCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	s.Scheduler.Pause(pauseCtx)

	// Ensure we resume later
	defer func() {
		s.Scheduler.Resume(context.Background())
	}()

	// 5. Execute the key rotation in the database.
	ctx := r.Context()
	rows, err := s.Repo.Pool.Query(ctx, "SELECT id, wrapped_key FROM storage_keys")
	if err != nil {
		s.Repo.LogAdminAction(ctx, username, "key_rotation_fail", err.Error())
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	type record struct {
		ID         string
		WrappedKey []byte
	}
	var records []record

	for rows.Next() {
		var rec record
		if err := rows.Scan(&rec.ID, &rec.WrappedKey); err != nil {
			rows.Close()
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		records = append(records, rec)
	}
	rows.Close()

	for _, rec := range records {
		_, errNew := crypto.UnwrapDEK(rec.WrappedKey, newMasterKey)
		if errNew == nil {
			continue // Already rotated
		}

		dek, err := crypto.UnwrapDEK(rec.WrappedKey, s.KEK)
		if err != nil {
			s.Repo.LogAdminAction(ctx, username, "key_rotation_fail", map[string]string{"error": "failed to unwrap DEK for ID " + rec.ID})
			http.Error(w, "Failed to unwrap DEK", http.StatusInternalServerError)
			return
		}

		newWrappedKey, err := crypto.WrapDEK(dek, newMasterKey)
		if err != nil {
			s.Repo.LogAdminAction(ctx, username, "key_rotation_fail", map[string]string{"error": "failed to re-wrap DEK for ID " + rec.ID})
			http.Error(w, "Failed to wrap DEK", http.StatusInternalServerError)
			return
		}

		_, err = s.Repo.Pool.Exec(ctx, "UPDATE storage_keys SET wrapped_key =  WHERE id = ", newWrappedKey, rec.ID)
		if err != nil {
			s.Repo.LogAdminAction(ctx, username, "key_rotation_fail", map[string]string{"error": "failed to update DEK for ID " + rec.ID})
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// 6. Update the Scheduler's MASTER_KEY in memory
	s.KEK = newMasterKey
	s.Repo.LogAdminAction(ctx, username, "key_rotation_success", map[string]interface{}{"count": len(records)})
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Key rotation successful"))
}
