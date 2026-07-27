/**
 * SPDX-FileComment: Scheduler
 * SPDX-FileType: SOURCE
 * SPDX-FileContributor: ZHENG Robert
 * SPDX-FileCopyrightText: 2026 ZHENG Robert
 * SPDX-License-Identifier: Apache-2.0
 *
 * @file scheduler.go
 * @brief Cron-based job scheduler managing program lifecycle
 * @version 1.0.0
 * @date 2026-06-02
 *
 * @author ZHENG Robert (robert@hase-zheng.net)
 * @copyright Copyright (c) 2026 ZHENG Robert
 * @LICENSE Apache-2.0
 */

// Package scheduler manages the lifecycle of cron-driven jobs. It loads
// enabled programs from the database, schedules them via the robfig/cron
// library, launches child processes with IPC environment variables, and
// tracks execution runs and exit codes.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"go-scheduler/internal/db"
)

// ActiveRun tracks an executing job instance in memory
type ActiveRun struct {
	RunID     int
	ProgramID int
	Name      string
	PID       int
	Cmd       *exec.Cmd
	Cancelled bool
}

// Scheduler manages the lifecycle of jobs
type Scheduler struct {
	Repo         *db.Repository
	Cron         *cron.Cron
	SocketPath   string
	DBConfigJSON string
	mu           sync.Mutex
	running      map[int]bool // Tracks active runs
	activeCmds   map[int]*exec.Cmd
	activeRuns   map[int]*ActiveRun
	wg           sync.WaitGroup
}

// New creates a new scheduler instance
func New(repo *db.Repository, socketPath string, dbConfigJSON string) *Scheduler {
	return &Scheduler{
		Repo:         repo,
		Cron:         cron.New(),
		SocketPath:   socketPath,
		DBConfigJSON: dbConfigJSON,
		running:      make(map[int]bool),
		activeCmds:   make(map[int]*exec.Cmd),
		activeRuns:   make(map[int]*ActiveRun),
	}
}

// Reload stops the current cron scheduler and reloads jobs from the DB
func (s *Scheduler) Reload(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Cron.Stop()
	s.Cron = cron.New() // Fresh instance

	return s.Start(ctx)
}

// Start loads enabled jobs and starts the cron scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	programs, err := s.Repo.GetEnabledPrograms(ctx)
	if err != nil {
		return fmt.Errorf("failed to load programs: %w", err)
	}

	for _, p := range programs {
		p := p // capture loop variable
		_, err := s.Cron.AddFunc(p.CronExpr, func() {
			s.RunProgram(p)
		})
		if err != nil {
			log.Printf("Error scheduling job %s: %v", p.Name, err)
			continue
		}
		log.Printf("Scheduled job %s with cron %s", p.Name, p.CronExpr)
	}

	s.Cron.Start()
	return nil
}

// RunProgram executes a single job
func (s *Scheduler) RunProgram(p db.ScheduledProgram) {
	ctx := context.Background()

	s.Repo.LogSystem(ctx, "INFO", "Scheduler", fmt.Sprintf("Starting job %s", p.Name))

	// 1. Create run entry in DB
	runID, err := s.Repo.CreateProgramRun(ctx, p.ID, 0)
	if err != nil {
		s.Repo.LogSystem(ctx, "ERROR", "Scheduler", fmt.Sprintf("Failed to create program run for %s: %v", p.Name, err))
		log.Printf("Failed to create program run for %s: %v", p.Name, err)
		return
	}

	// 2. Prepare command
	argsJSON := "{}"
	if len(p.Args) > 0 {
		argsJSON = string(p.Args)
	}
	cmd := exec.Command(p.Command, argsJSON)

	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RUN_ID=%d", runID),
		fmt.Sprintf("SCHEDULER_SOCKET_PATH=%s", s.SocketPath),
		fmt.Sprintf("MITM_DB_CONFIG_JSON=%s", s.DBConfigJSON),
	)

	var dbCfg struct {
		DB struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			User     string `json:"user"`
			Password string `json:"password"`
			Database string `json:"database"`
			SSLMode  bool   `json:"sslmode"`
		} `json:"db"`
	}
	if err := json.Unmarshal([]byte(s.DBConfigJSON), &dbCfg); err == nil {
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("MITM_DB_HOST=%s", dbCfg.DB.Host),
			fmt.Sprintf("MITM_DB_PORT=%d", dbCfg.DB.Port),
			fmt.Sprintf("MITM_DB_USER=%s", dbCfg.DB.User),
			fmt.Sprintf("MITM_DB_PASSWORD=%s", dbCfg.DB.Password),
			fmt.Sprintf("MITM_DB_NAME=%s", dbCfg.DB.Database),
			fmt.Sprintf("MITM_DB_SSLMODE=%t", dbCfg.DB.SSLMode),
		)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 3. Start process
	if err := cmd.Start(); err != nil {
		s.Repo.LogSystem(ctx, "ERROR", "Scheduler", fmt.Sprintf("Failed to start job %s (RunID %d): %v", p.Name, runID, err))
		log.Printf("Failed to start job %s (RunID %d): %v", p.Name, runID, err)
		s.Repo.UpdateProgramRun(ctx, runID, -1, false)
		return
	}

	run := &ActiveRun{
		RunID:     runID,
		ProgramID: p.ID,
		Name:      p.Name,
		PID:       cmd.Process.Pid,
		Cmd:       cmd,
	}
	s.wg.Add(1)
	s.mu.Lock()
	s.activeCmds[runID] = cmd
	s.activeRuns[runID] = run
	s.mu.Unlock()

	pid := cmd.Process.Pid
	s.Repo.LogSystem(ctx, "DEBUG", "Scheduler", fmt.Sprintf("Job %s started (RunID %d, PID %d)", p.Name, runID, pid))
	log.Printf("Started job %s (RunID %d, PID %d)", p.Name, runID, pid)

	_, _ = s.Repo.Pool.Exec(ctx, "UPDATE program_runs SET pid = $1 WHERE id = $2", pid, runID)

	// 4. Wait for completion
	go func() {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			delete(s.activeCmds, runID)
			delete(s.activeRuns, runID)
			s.mu.Unlock()
		}()

		err := cmd.Wait()
		exitCode := 0
		success := true

		if err != nil {
			success = false
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			} else {
				exitCode = -1
			}
		}

		s.Repo.LogSystem(ctx, "INFO", "Scheduler", fmt.Sprintf("Job %s (RunID %d) finished with exit code %d", p.Name, runID, exitCode))
		log.Printf("Job %s (RunID %d) finished with exit code %d", p.Name, runID, exitCode)
		if err := s.Repo.UpdateProgramRun(ctx, runID, exitCode, success); err != nil {
			log.Printf("Failed to update final status for RunID %d: %v", runID, err)
		}

		s.mu.Lock()
		cancelled := false
		if r, ok := s.activeRuns[runID]; ok {
			cancelled = r.Cancelled
		}
		s.mu.Unlock()

		// Restart if configured and not explicitly cancelled
		if p.RestartOnExit && !success && !cancelled {
			s.Repo.LogSystem(ctx, "INFO", "Scheduler", fmt.Sprintf("Restarting job %s due to failure", p.Name))
			log.Printf("Restarting job %s due to failure...", p.Name)
			s.RunProgram(p)
		}
	}()
}

// RunImmediateJob executes a job immediately without saving it to the enabled programs list
func (s *Scheduler) RunImmediateJob(ctx context.Context, command string, args []byte) {
	p := db.ScheduledProgram{
		ID:            -1, // Temporary ID
		Name:          "Immediate_Trigger",
		Command:       command,
		Args:          args,
		RestartOnExit: false,
	}
	s.RunProgram(p)
}

// Stop halts the cron scheduler and gracefully shuts down running jobs
func (s *Scheduler) Stop(ctx context.Context) {
	s.Cron.Stop()

	s.mu.Lock()
	for runID, cmd := range s.activeCmds {
		s.Repo.LogSystem(context.Background(), "INFO", "Scheduler", fmt.Sprintf("Sending SIGTERM to job RunID %d", runID))
		log.Printf("Sending SIGTERM to job RunID %d", runID)
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
	}
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All jobs shut down gracefully.")
	case <-ctx.Done():
		log.Println("Graceful shutdown timeout exceeded. Killing remaining jobs.")
		s.mu.Lock()
		for runID, cmd := range s.activeCmds {
			log.Printf("Killing job RunID %d", runID)
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
		s.mu.Unlock()
	}
}

// GetActivePIDs returns a map of currently running job names to their process IDs
func (s *Scheduler) GetActivePIDs() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	res := make(map[string]int)
	for _, run := range s.activeRuns {
		res[run.Name] = run.PID
	}
	return res
}

// StopJobByName sends a termination signal to an active job identified by its name
func (s *Scheduler) StopJobByName(name string) error {
	s.mu.Lock()
	var targetRun *ActiveRun
	for _, run := range s.activeRuns {
		if run.Name == name {
			targetRun = run
			break
		}
	}
	if targetRun == nil {
		s.mu.Unlock()
		return fmt.Errorf("job %q is not currently running", name)
	}
	targetRun.Cancelled = true
	cmd := targetRun.Cmd
	runID := targetRun.RunID
	s.mu.Unlock()

	s.Repo.LogSystem(context.Background(), "INFO", "Scheduler", fmt.Sprintf("Sending SIGTERM to job %q (RunID %d)", name, runID))
	log.Printf("Sending SIGTERM to job %q (RunID %d)", name, runID)
	if cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}

	go func() {
		time.Sleep(5 * time.Second)
		s.mu.Lock()
		_, stillActive := s.activeRuns[runID]
		s.mu.Unlock()
		if stillActive && cmd.Process != nil {
			log.Printf("Job %q (RunID %d) did not terminate after 5s, sending SIGKILL", name, runID)
			_ = cmd.Process.Kill()
		}
	}()

	return nil
}
