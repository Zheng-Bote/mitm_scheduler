package http

import (
	"encoding/json"
	"net/http"
	"strconv"
)


func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (string, bool) {
	username, ok := s.authenticate(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}
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
		http.Error(w, "Forbidden: ADMIN role required", http.StatusForbidden)
		return "", false
	}
	return username, true
}

func (s *Server) handleGetRoles(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	roles, err := s.Repo.GetRoles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	users, err := s.Repo.GetUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (s *Server) handleAssignRoles(w http.ResponseWriter, r *http.Request) {
	adminUser, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID  int   `json:"user_id"`
		RoleIDs []int `json:"role_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.Repo.AssignRoles(r.Context(), req.UserID, req.RoleIDs, s.KEK); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), adminUser, "ASSIGN_ROLES", req)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetUserRoles(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user_id", http.StatusBadRequest)
		return
	}

	roleIDs, err := s.Repo.GetUserRoles(r.Context(), userID, s.KEK)
	if err != nil {
		// Just return empty if no roles or decrypt error
		roleIDs = []int{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roleIDs)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	adminUser, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	if err := s.Repo.CreateUser(r.Context(), req.Username, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), adminUser, "CREATE_USER", map[string]string{"username": req.Username})

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	adminUser, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.URL.Query().Get("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := s.Repo.DeleteUser(r.Context(), userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.Repo.LogAdminAction(r.Context(), adminUser, "DELETE_USER", map[string]int{"user_id": userID})

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetOsUserRoles(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.authenticate(r); !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("os_user")
	if username == "" {
		http.Error(w, "os_user required", http.StatusBadRequest)
		return
	}

	roleNames, err := s.Repo.GetUserRolesByUsername(r.Context(), username, s.KEK)
	if err != nil {
		roleNames = []string{} // return empty list if user not found or error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roleNames)
}
