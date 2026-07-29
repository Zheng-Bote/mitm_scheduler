package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"

	"go-scheduler/internal/config"
	"go-scheduler/internal/crypto"
	"go-scheduler/internal/db"
)

func main() {
	var configPath = flag.String("config", "", "Path to encrypted config.json")
	var password = flag.String("password", "", "Decryption password for config")
	var newAdminUser = flag.String("user", "admin", "Username of the new admin")
	var newAdminPass = flag.String("pass", "admin123", "Password for the new admin")
	flag.Parse()

	if *configPath == "" || *password == "" {
		log.Fatal("Usage: create-admin -config <config.json> -password <password> [-user admin] [-pass admin123]")
	}

	dbCfg, err := config.LoadEncryptedConfig(*configPath, *password)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	repo, err := db.NewRepository(ctx, dbCfg.GetDSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Pool.Close()

	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		log.Fatalf("Failed to generate salt: %v", err)
	}

	// Use our crypto package's Argon2 derive to get the hash
	hash := crypto.DeriveKey([]byte(*newAdminPass), salt)

	// Format as a simple string to store in DB: base64(salt):base64(hash)
	hashStr := fmt.Sprintf("%s:%s", base64.StdEncoding.EncodeToString(salt), base64.StdEncoding.EncodeToString(hash))

	var userID int
	err = repo.Pool.QueryRow(ctx, `
		INSERT INTO admin_users (username, password_hash, is_active)
		VALUES ($1, $2, true)
		ON CONFLICT (username) DO UPDATE SET password_hash = EXCLUDED.password_hash
		RETURNING id
	`, *newAdminUser, hashStr).Scan(&userID)

	if err != nil {
		log.Fatalf("Failed to insert user: %v", err)
	}

	// Also assign the ADMIN role
	// KEK is the password used to decrypt config
	kek := []byte(*password)

	// Get ADMIN role ID
	var roleID int
	err = repo.Pool.QueryRow(ctx, "SELECT id FROM roles WHERE name = 'ADMIN'").Scan(&roleID)
	if err != nil {
		log.Fatalf("Failed to get ADMIN role: %v", err)
	}

	err = repo.AssignRoles(ctx, userID, []int{roleID}, kek)
	if err != nil {
		log.Fatalf("Failed to assign ADMIN role: %v", err)
	}

	fmt.Printf("Successfully created or updated admin user '%s' (ID: %d) with ADMIN role.\n", *newAdminUser, userID)
}
