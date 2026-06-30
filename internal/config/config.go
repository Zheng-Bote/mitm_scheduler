/**
 * SPDX-FileComment: Config
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file config.go
 * @brief Configuration data types and encrypted config loading
 * @version 1.0.0
 * @date 2026-06-02
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

// Package config provides data types and functions for loading and managing
// the scheduler's encrypted configuration file. It handles decryption via the
// crypto package and exposes PostgreSQL connection parameters and admin user
// definitions.
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"go-scheduler/internal/crypto"
)

// AdminUser defines a simple administrative user for API access
type AdminUser struct {
	Username string `json:"username"`
	Token    string `json:"token"` // This will be used for the HELO auth
}

type DBConnectionConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	Password       string `json:"password"`
	Database       string `json:"database"`
	DBConnectDelay int    `json:"db_connect_delay,omitempty"`
}

type DBConfig struct {
	DB             DBConnectionConfig `json:"db"`
	LogLevel       string             `json:"log_level"`
	UploadDir      string             `json:"upload_dir"`
	Admins         []AdminUser        `json:"admins"`
	HTTPPort       int                `json:"http_port,omitempty"`
	UseHTTPS       bool               `json:"use_https,omitempty"`
	SSLCert        string             `json:"ssl_cert,omitempty"`
	SSLKey         string             `json:"ssl_key,omitempty"`
}

// LoadEncryptedConfig reads an encrypted JSON file and decrypts it into DBConfig
func LoadEncryptedConfig(filePath string, password string) (*DBConfig, error) {
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted config file: %w", err)
	}

	decryptedData, err := crypto.Decrypt(encryptedData, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt config: %w", err)
	}

	var dbConfig DBConfig
	if err := json.Unmarshal(decryptedData, &dbConfig); err != nil {
		return nil, fmt.Errorf("failed to parse decrypted config JSON: %w", err)
	}

	return &dbConfig, nil
}

// GetDSN returns the PostgreSQL connection string
func (c *DBConfig) GetDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.DB.User, c.DB.Password, c.DB.Host, c.DB.Port, c.DB.Database)
}
