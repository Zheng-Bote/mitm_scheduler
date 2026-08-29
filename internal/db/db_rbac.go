package db

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"go-scheduler/internal/crypto"
)

type Role struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

func (r *Repository) GetRoles(ctx context.Context) ([]Role, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id, name, description FROM roles ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *Repository) GetUsers(ctx context.Context) ([]User, error) {
	rows, err := r.Pool.Query(ctx, "SELECT id, username, is_active FROM admin_users ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Username, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *Repository) AssignRoles(ctx context.Context, userID int, roleIDs []int, kek []byte) error {
	rolesJSON, err := json.Marshal(roleIDs)
	if err != nil {
		return err
	}

	wrappedDEK, err := crypto.GenerateWrappedDEK(kek)
	if err != nil {
		return err
	}

	ciphertext, nonce, err := crypto.EnvelopeEncrypt(kek, wrappedDEK, rolesJSON)
	if err != nil {
		return err
	}

	_, err = r.Pool.Exec(ctx, `
		INSERT INTO user_roles_encrypted (user_id, wrapped_dek, nonce, encrypted_roles)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			wrapped_dek = EXCLUDED.wrapped_dek,
			nonce = EXCLUDED.nonce,
			encrypted_roles = EXCLUDED.encrypted_roles,
			updated_at = CURRENT_TIMESTAMP
	`, userID, wrappedDEK, nonce, ciphertext)
	return err
}

func (r *Repository) GetUserRoles(ctx context.Context, userID int, kek []byte) ([]int, error) {
	var wrappedDEK, nonce, ciphertext []byte
	err := r.Pool.QueryRow(ctx, "SELECT wrapped_dek, nonce, encrypted_roles FROM user_roles_encrypted WHERE user_id = $1", userID).Scan(&wrappedDEK, &nonce, &ciphertext)
	if err != nil {
		return nil, err
	}

	plaintext, err := crypto.EnvelopeDecrypt(kek, wrappedDEK, nonce, ciphertext)
	if err != nil {
		return nil, errors.New("failed to decrypt roles")
	}

	var roleIDs []int
	if err := json.Unmarshal(plaintext, &roleIDs); err != nil {
		return nil, err
	}

	return roleIDs, nil
}

func (r *Repository) CreateUser(ctx context.Context, username, password string) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	hash := crypto.DeriveKey([]byte(password), salt)
	hashStr := fmt.Sprintf("%s:%s", base64.StdEncoding.EncodeToString(salt), base64.StdEncoding.EncodeToString(hash))

	_, err := r.Pool.Exec(ctx, "INSERT INTO admin_users (username, password_hash, is_active) VALUES ($1, $2, true)", username, hashStr)
	return err
}

func (r *Repository) DeleteUser(ctx context.Context, userID int) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM admin_users WHERE id = $1", userID)
	return err
}
func (r *Repository) GetUserRolesByUsername(ctx context.Context, username string, kek []byte) ([]string, error) {
	var userID int
	err := r.Pool.QueryRow(ctx, "SELECT id FROM admin_users WHERE username = $1 AND is_active = true", username).Scan(&userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	roleIDs, err := r.GetUserRoles(ctx, userID, kek)
	if err != nil {
		return nil, err
	}

	if len(roleIDs) == 0 {
		return []string{}, nil
	}

	// Fetch role names
	var roleNames []string
	rows, err := r.Pool.Query(ctx, "SELECT name FROM roles WHERE id = ANY($1)", roleIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			roleNames = append(roleNames, name)
		}
	}

	return roleNames, nil
}
