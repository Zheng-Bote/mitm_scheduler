/**
 * SPDX-FileComment: Config
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file config.go
 * @brief Configuration data types and ENV config loading
 * @version 2.0.0
 * @date 2026-09-14
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	SSLMode        bool   `json:"sslmode,omitempty"`
	MaxConns       int    `json:"max_conns,omitempty"`
}

type DBConfig struct {
	DB        DBConnectionConfig `json:"db"`
	LogLevel  string             `json:"log_level"`
	UploadDir string             `json:"upload_dir"`
	Admins    []AdminUser        `json:"admins"`
	HTTPPort  int                `json:"http_port,omitempty"`
	UseHTTPS  bool               `json:"use_https,omitempty"`
	SSLCert   string             `json:"ssl_cert,omitempty"`
	SSLKey    string             `json:"ssl_key,omitempty"`
}

func getEnvStr(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, exists := os.LookupEnv(key); exists {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val, exists := os.LookupEnv(key); exists {
		lower := strings.ToLower(val)
		return lower == "true" || lower == "1" || lower == "yes"
	}
	return defaultVal
}

// LoadConfig reads configuration from standard environment variables
func LoadConfig() (*DBConfig, error) {
	exePath, err := os.Executable()
	var exeDir string
	if err == nil {
		exeDir = filepath.Dir(exePath)
	} else {
		exeDir = "."
	}

	var dbConfig DBConfig
	
	dbConfig.DB.Host = getEnvStr("MITM_DB_HOST", "")
	dbConfig.DB.Port = getEnvInt("MITM_DB_PORT", 5432)
	dbConfig.DB.User = getEnvStr("MITM_DB_USER", "")
	dbConfig.DB.Password = getEnvStr("MITM_DB_PASSWORD", "")
	dbConfig.DB.Database = getEnvStr("MITM_DB_NAME", "")
	dbConfig.DB.DBConnectDelay = getEnvInt("MITM_DB_CONNECT_DELAY", 5)
	
	sslModeStr := strings.ToLower(getEnvStr("MITM_DB_SSLMODE", ""))
	if sslModeStr == "require" || sslModeStr == "true" || sslModeStr == "1" || sslModeStr == "yes" {
		dbConfig.DB.SSLMode = true
	} else {
		dbConfig.DB.SSLMode = getEnvBool("MITM_DB_SSL", false)
	}
	
	dbConfig.LogLevel = getEnvStr("MITM_LOG_LEVEL", "INFO")
	dbConfig.UploadDir = getEnvStr("MITM_UPLOAD_DIR", filepath.Join(exeDir, "mitm_uploads"))
	dbConfig.HTTPPort = getEnvInt("MITM_HTTP_PORT", 8080)
	dbConfig.UseHTTPS = getEnvBool("MITM_USE_HTTPS", false)
	dbConfig.SSLCert = getEnvStr("MITM_SSL_CERT", filepath.Join(exeDir, "certs", "server.crt"))
	dbConfig.SSLKey = getEnvStr("MITM_SSL_KEY", filepath.Join(exeDir, "certs", "server.key"))

	adminsStr := getEnvStr("MITM_ADMINS", "")
	if adminsStr != "" {
		for _, admin := range strings.Split(adminsStr, ",") {
			admin = strings.TrimSpace(admin)
			if admin != "" {
				dbConfig.Admins = append(dbConfig.Admins, AdminUser{
					Username: admin,
					Token:    "cority", // "Blender" token
				})
			}
		}
	}

	if dbConfig.DB.Host == "" || dbConfig.DB.User == "" || dbConfig.DB.Password == "" {
		return nil, fmt.Errorf("MITM_DB_HOST, MITM_DB_USER, and MITM_DB_PASSWORD are required")
	}

	return &dbConfig, nil
}

// GetDSN returns the PostgreSQL connection string
func (c *DBConfig) GetDSN() string {
	sslMode := "disable"
	if c.DB.SSLMode {
		sslMode = "require"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DB.User, c.DB.Password, c.DB.Host, c.DB.Port, c.DB.Database, sslMode)
}
