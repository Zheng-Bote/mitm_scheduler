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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

// LoadConfig resolves configuration following precedence rules:
// 1. Commandline Parameter (if provided)
// 2. ENV variables (if present)
// 3. Default config files (<binary>/config.enc or <binary>/cfg/config.enc)
// 4. Internal defaults
func LoadConfig(cliParam string, password string) (*DBConfig, error) {
	exePath, err := os.Executable()
	var exeDir string
	if err == nil {
		exeDir = filepath.Dir(exePath)
	} else {
		exeDir = "."
	}

	var cfg *DBConfig

	// 1. Try Commandline parameter
	if cliParam != "" {
		cfg, err = LoadEncryptedConfig(cliParam, password)
		if err == nil {
			log.Printf("Loaded config from parameter: %s", cliParam)
			return applyCertificateFallback(cfg, exeDir), nil
		}
		log.Printf("Warning: Failed to load config from parameter %s: %v. Falling back to ENVs.", cliParam, err)
	}

	// 2. Try ENVs if required ENV (MITM_DB_HOST) is present
	if cfg == nil && getEnvStr("MITM_DB_HOST", "") != "" {
		cfg = loadFromEnv(exeDir)
		log.Println("Loaded config from Environment Variables.")
		return applyCertificateFallback(cfg, exeDir), nil
	}

	// 3. Try Default files
	if cfg == nil {
		defaultPath := filepath.Join(exeDir, "config.enc")
		cfg, err = LoadEncryptedConfig(defaultPath, password)
		if err == nil {
			log.Printf("Loaded config from default path: %s", defaultPath)
			return applyCertificateFallback(cfg, exeDir), nil
		}

		fallbackPath := filepath.Join(exeDir, "cfg", "config.enc")
		cfg, err = LoadEncryptedConfig(fallbackPath, password)
		if err == nil {
			log.Printf("Loaded config from fallback path: %s", fallbackPath)
			return applyCertificateFallback(cfg, exeDir), nil
		}
	}

	// 4. Fallback to internal defaults
	if cfg == nil {
		log.Println("Warning: No config file or ENVs found. Falling back to internal defaults.")
		cfg = applyInternalDefaults(exeDir)
	}

	return applyCertificateFallback(cfg, exeDir), nil
}

func loadFromEnv(exeDir string) *DBConfig {
	var dbConfig DBConfig

	dbConfig.DB.Host = getEnvStr("MITM_DB_HOST", "")
	dbConfig.DB.Port = getEnvInt("MITM_DB_PORT", 5432)
	dbConfig.DB.User = getEnvStr("MITM_DB_USER", "")
	dbConfig.DB.Password = getEnvStr("MITM_DB_PASSWORD", "")
	dbConfig.DB.Database = getEnvStr("MITM_DB_NAME", "")
	dbConfig.DB.DBConnectDelay = getEnvInt("MITM_DB_CONNECT_DELAY", 5)
	dbConfig.DB.MaxConns = getEnvInt("MITM_DB_MAX_CONNS", 50)

	sslModeStr := strings.ToLower(getEnvStr("MITM_DB_SSLMODE", ""))
	if sslModeStr == "disable" || sslModeStr == "false" || sslModeStr == "0" || sslModeStr == "no" {
		dbConfig.DB.SSLMode = false
	} else if sslModeStr == "require" || sslModeStr == "true" || sslModeStr == "1" || sslModeStr == "yes" {
		dbConfig.DB.SSLMode = true
	} else {
		dbConfig.DB.SSLMode = getEnvBool("MITM_DB_SSL", true)
	}

	dbConfig.LogLevel = getEnvStr("MITM_LOG_LEVEL", "INFO")
	dbConfig.UploadDir = getEnvStr("MITM_UPLOAD_DIR", filepath.Join(exeDir, "mitm_uploads"))
	dbConfig.HTTPPort = getEnvInt("MITM_HTTP_PORT", 8443)
	dbConfig.UseHTTPS = getEnvBool("MITM_USE_HTTPS", true)
	dbConfig.SSLCert = getEnvStr("MITM_SSL_CERT", getEnvStr("MITM_SSL_CRT", filepath.Join(exeDir, "certs", "server.crt")))
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

	return &dbConfig
}

func applyInternalDefaults(exeDir string) *DBConfig {
	return &DBConfig{
		DB: DBConnectionConfig{
			Host:           "localhost",
			Port:           5432,
			User:           "mitm_user",
			Password:       "",
			Database:       "mitm",
			DBConnectDelay: 5,
			MaxConns:       50,
			SSLMode:        true,
		},
		LogLevel:  "INFO",
		UploadDir: filepath.Join(exeDir, "mitm_uploads"),
		HTTPPort:  8443,
		UseHTTPS:  true,
		SSLCert:   filepath.Join(exeDir, "certs", "server.crt"),
		SSLKey:    filepath.Join(exeDir, "certs", "server.key"),
	}
}

// applyCertificateFallback verifies if SSLCert and SSLKey files exist.
// If not, searches fallback directories <binary_dir>/. and <binary_dir>/certs/.
func applyCertificateFallback(cfg *DBConfig, exeDir string) *DBConfig {
	if cfg == nil {
		return cfg
	}

	checkFallback := func(currentPath string, filename string) string {
		if _, err := os.Stat(currentPath); err == nil {
			return currentPath
		}
		fallbackOpts := []string{
			filepath.Join(exeDir, filename),
			filepath.Join(exeDir, "certs", filename),
		}
		for _, opt := range fallbackOpts {
			if _, err := os.Stat(opt); err == nil {
				return opt
			}
		}
		return currentPath
	}

	certName := filepath.Base(cfg.SSLCert)
	if certName == "." || certName == "/" || certName == "" {
		certName = "server.crt"
	}
	cfg.SSLCert = checkFallback(cfg.SSLCert, certName)

	keyName := filepath.Base(cfg.SSLKey)
	if keyName == "." || keyName == "/" || keyName == "" {
		keyName = "server.key"
	}
	cfg.SSLKey = checkFallback(cfg.SSLKey, keyName)

	return cfg
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
