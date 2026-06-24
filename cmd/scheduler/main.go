/**
 * SPDX-FileComment: Go Scheduler
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file main.go
 * @brief Main entry point for the scheduler service
 * @version 1.1.0
 * @date 2026-06-08
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go-scheduler/internal/config"
	"go-scheduler/internal/db"
	"go-scheduler/internal/http"
	"go-scheduler/internal/ipc"
	"go-scheduler/internal/scheduler"
)

var (
	appName        = "MitM Scheduler"
	appDescription = "Backend scheduler for the MitM project"
	version        = "1.0.0"
)

func bootstrapAdmins(ctx context.Context, repo *db.Repository, admins []config.AdminUser, kek []byte) {
	roles, err := repo.GetRoles(ctx)
	if err != nil {
		log.Printf("Failed to get roles for bootstrap: %v", err)
		return
	}
	var adminRoleID int
	for _, r := range roles {
		if r.Name == "ADMIN" {
			adminRoleID = r.ID
			break
		}
	}
	if adminRoleID == 0 {
		log.Printf("ADMIN role not found in database for bootstrap")
		return
	}

	for _, adminCfg := range admins {
		users, err := repo.GetUsers(ctx)
		if err != nil {
			log.Printf("Failed to get users for bootstrap: %v", err)
			continue
		}
		
		var user *db.User
		for _, u := range users {
			if u.Username == adminCfg.Username {
				uCopy := u
				user = &uCopy
				break
			}
		}

		created := false
		if user == nil {
			err = repo.CreateUser(ctx, adminCfg.Username, adminCfg.Token)
			if err != nil {
				log.Printf("Failed to create admin user %s: %v", adminCfg.Username, err)
				continue
			}
			created = true
			
			// Refresh users to get the ID
			users, _ = repo.GetUsers(ctx)
			for _, u := range users {
				if u.Username == adminCfg.Username {
					uCopy := u
					user = &uCopy
					break
				}
			}
			if user == nil {
				log.Printf("Failed to fetch newly created admin user %s", adminCfg.Username)
				continue
			}
		}

		roleIDs, err := repo.GetUserRoles(ctx, user.ID, kek)
		if err != nil {
			// If no roles exist yet, ignore the error and initialize empty
			roleIDs = []int{}
		}

		hasAdminRole := false
		for _, rid := range roleIDs {
			if rid == adminRoleID {
				hasAdminRole = true
				break
			}
		}

		roleAssigned := false
		if !hasAdminRole {
			roleIDs = append(roleIDs, adminRoleID)
			err = repo.AssignRoles(ctx, user.ID, roleIDs, kek)
			if err != nil {
				log.Printf("Failed to assign ADMIN role to %s: %v", adminCfg.Username, err)
				continue
			}
			roleAssigned = true
		}

		if created || roleAssigned {
			details := map[string]interface{}{
				"user_created":  created,
				"role_assigned": roleAssigned,
			}
			repo.LogAdminAction(ctx, "SYSTEM", fmt.Sprintf("Bootstrap Admin: %s", adminCfg.Username), details)
			log.Printf("Bootstrapped admin %s (created: %v, role_assigned: %v)", adminCfg.Username, created, roleAssigned)
		}
	}
}

func main() {
	var configPath string
	if len(os.Args) < 2 {
		execPath, err := os.Executable()
		if err == nil {
			execDir := filepath.Dir(execPath)
			if _, err := os.Stat(filepath.Join(execDir, "config.json")); err == nil {
				configPath = filepath.Join(execDir, "config.json")
			} else if _, err := os.Stat(filepath.Join(execDir, "config.enc")); err == nil {
				configPath = filepath.Join(execDir, "config.enc")
			}
		}
		if configPath == "" {
			fmt.Println("Usage: scheduler <path/to/encrypted/config.json>")
			fmt.Println("Alternatively, place config.json or config.enc in the same directory as the executable.")
			os.Exit(1)
		}
	} else {
		configPath = os.Args[1]
	}

	if os.Getenv("MASTER_KEY") == "" {
		log.Println("DEBUG: Using default MASTER_KEY")
		os.Setenv("MASTER_KEY", "6mkdHpNHfF5bdCMj/+MeYAM4wVMy3nJ9FRxpSibhumE=")
	}


	// 1. Get Password
	password := os.Getenv("SCHEDULER_PASSWORD")
	if password == "" {
		log.Println("DEBUG: Using default SCHEDULER_PASSWORD")
		password = "Sidolin1"
	}

	// 2. Load Config
	dbCfg, err := config.LoadEncryptedConfig(configPath, password)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 3. Prepare early startup configuration
	var schedCfg *db.SchedulerConfig = &db.SchedulerConfig{
		SocketPath: "/tmp/mitm_debug.sock",
		HTTPPort:   8080,
	}

	dbConfigBytes, err := json.Marshal(dbCfg)
	if err != nil {
		log.Fatalf("Failed to marshal DB config: %v", err)
	}
	sched := scheduler.New(nil, schedCfg.SocketPath, string(dbConfigBytes))

	// 4. Start HTTP Server Early
	httpServer := &http.Server{
		Repo:           nil, // DB not connected yet
		Port:           schedCfg.HTTPPort,
		Admins:         dbCfg.Admins,
		KEK:            []byte(password),
		UploadDir:      dbCfg.UploadDir,
		Scheduler:      sched,
		AppName:        appName,
		AppDescription: appDescription,
		AppVersion:     version,
	}
	if err := httpServer.Start(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
	log.Printf("HTTP Server listening on port %d (pre-DB connect)", schedCfg.HTTPPort)

	// 5. Initial DB Connect Delay
	if dbCfg.DBConnectDelay > 0 {
		log.Printf("Delaying initial DB connection by %d seconds...", dbCfg.DBConnectDelay)
		time.Sleep(time.Duration(dbCfg.DBConnectDelay) * time.Second)
	}

	// 6. Connect to DB with retries
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var repo *db.Repository
	for {
		repo, err = db.NewRepository(ctx, dbCfg.GetDSN())
		if err == nil {
			log.Println("Successfully connected to database.")
			break
		}
		
		delay := dbCfg.DBConnectDelay
		if delay <= 0 {
			delay = 10 // Fallback
		}
		log.Printf("Failed to connect to database: %v. Retrying in %d seconds...", err, delay)
		time.Sleep(time.Duration(delay) * time.Second)
	}
	defer repo.Pool.Close()

	// Supply the connected repository to the running servers
	httpServer.Repo = repo
	sched.Repo = repo

	// Bootstrap admins from config
	bootstrapAdmins(ctx, repo, dbCfg.Admins, []byte(password))

	// Load Scheduler Config from DB
	dbSchedCfg, err := repo.GetSchedulerConfig(ctx)
	if err == nil {
		schedCfg = dbSchedCfg
	} else {
		log.Printf("Failed to load scheduler config from DB: %v", err)
	}

	// 7. Start IPC Server
	ipcServer := &ipc.Server{
		SocketPath: schedCfg.SocketPath,
		OnEvent: func(event ipc.StatusEvent) {
			if event.Type == "audit" {
				log.Printf("AUDIT: RunID=%d, Component=%s, Message=%s", event.RunID, event.Component, event.Message)
				_ = repo.CreateAuditLog(context.Background(), event.RunID, event.Component, event.Message)
				return
			}

			log.Printf("IPC Event: RunID=%d, Status=%s, Message=%s, Progress=%d%%",
				event.RunID, event.Status, event.Message, event.Progress)

			err := repo.CreateStatusEvent(context.Background(), event.RunID, event.Status, event.Message, event.Progress)
			if err != nil {
				log.Printf("Failed to save IPC event to DB: %v", err)
			}
		},
	}
	if err := ipcServer.Start(); err != nil {
		repo.LogSystem(ctx, "ERROR", "IPC", fmt.Sprintf("Failed to start IPC server: %v", err))
		log.Fatalf("Failed to start IPC server: %v", err)
	}
	repo.LogSystem(ctx, "INFO", "IPC", fmt.Sprintf("IPC Server listening on %s", schedCfg.SocketPath))
	log.Printf("IPC Server listening on %s", schedCfg.SocketPath)

	// Update scheduler's socket path if it changed from the DB config
	sched.SocketPath = schedCfg.SocketPath
	// 8. Start Scheduler
	if err := sched.Start(ctx); err != nil {
		repo.LogSystem(ctx, "ERROR", "Scheduler", fmt.Sprintf("Failed to start scheduler: %v", err))
		log.Fatalf("Failed to start scheduler: %v", err)
	}
	successMsg := fmt.Sprintf("%s (%s) started successfully", appName, version)
	repo.LogSystem(ctx, "INFO", "Scheduler", successMsg)
	log.Println(successMsg)

	// 8. Wait for termination
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	repo.LogSystem(ctx, "INFO", "Scheduler", "Shutting down...")
	log.Println("Shutting down...")
	
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	sched.Stop(shutdownCtx)
}
