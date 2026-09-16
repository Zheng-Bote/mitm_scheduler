# System Components - Go Scheduler

This document details the software components, packages, and CLI utilities that make up the Go Scheduler project.

---

## 1. Directory & Package Overview

The project is structured following clean Go conventions, separating the executable entry points (`cmd/`) from internal reusable library code (`internal/`).

```
go_scheduler/
├── cmd/                          # Command entry points
│   ├── scheduler/               # Main scheduler application
│   ├── create-admin/            # CLI utility to bootstrap admin usersclient
│   └── job1/, job2/             # Sample jobs for testing IPC
├── internal/                     # Private application packages
│   ├── config/                  # Configuration structure & loader
│   ├── crypto/                  # Cryptographic helper functions (AES-256-GCM + Argon2id)
│   ├── db/                      # PostgreSQL database repository layer (pgx/v5)
│   ├── http/                    # REST API server and handlers
│   ├── ipc/                     # Unix Domain Socket IPC (JSON-Lines)
│   └── scheduler/               # Process manager & cron runner (robfig/cron/v3)
└── windows/                      # Windows-specific API/bindings for Windows Hello
```

---

## 2. Component Deep-Dive

### 2.1 Core Scheduler Daemon (`cmd/scheduler`)
*   **Location**: [cmd/scheduler/main.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/cmd/scheduler/main.go)
*   **Purpose**: The central background process (daemon) running on the Linux host.
*   **Startup Sequence**:
    1.  Prompts for decryption password (if the `SCHEDULER_PASSWORD` environment variable is not set).
    2.  Invokes [config.LoadEncryptedConfig](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/config/config.go) to read database credentials.
    3.  Initializes the PostgreSQL database connection pool.
    4.  Fetches global parameters from the DB (port, socket path, log level).
    5.  Starts the IPC UNIX Domain Socket Server in the background.
    6.  Starts the REST API Server in the background.
    7.  Loads enabled jobs from the DB, schedules them in the Cron Engine, and enters a blocking state waiting for termination signals (`SIGINT`, `SIGTERM`).

### 2.2 Security & Configuration (`internal/config`, `internal/crypto`)

*   **Envelope Encryption**: All sensitive Personal Identifiable Information (PII) is encrypted at rest using AES-GCM (256-bit).
*   **Key Hierarchy**: 
    *   **Master Key (KEK)**: The Key Encryption Key. It is *never* stored in the database or on disk. It is injected into the Scheduler via the `MASTER_KEY` environment variable.
    *   **Data Encryption Keys (DEKs)**: Unique DEKs are generated for each data topic, wrapped (encrypted) with the KEK, and stored in the PostgreSQL database. When agents run, they receive the wrapped DEK, which they unwrap in memory using the KEK (provided via IPC).
*   **Configuration**: Handled strictly via standard Environment Variables (`MITM_DB_HOST`, `MITM_DB_PASSWORD`, etc.) enabling secure AWS deployment via Secrets Manager.

### 2.3 Database Access Layer (`internal/db`)
*   **Location**: [internal/db/db.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/db/db.go)
*   **Driver**: `github.com/jackc/pgx/v5/pgxpool` (PostgreSQL connection pool).
*   **Repository Pattern**: Encapsulates all SQL statements inside the [Repository](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/db/db.go#L64) struct. Responsible for:
    *   Reading and writing job configurations.
    *   Recording system logs ([LogSystem](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/db/db.go#L93)).
    *   Tracking individual execution instances ([CreateProgramRun](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/db/db.go#L176) & [UpdateProgramRun](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/db/db.go#L183)).
    *   Persisting IPC job status events ([CreateStatusEvent](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/db/db.go#L189)) and job audit records ([CreateAuditLog](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/db/db.go#L101)).

### 2.4 Scheduling & Process Execution (`internal/scheduler`)
*   **Location**: [internal/scheduler/scheduler.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/scheduler/scheduler.go)
*   **Engine**: `github.com/robfig/cron/v3`
*   **Process Spawner**: When a cron trigger fires, the scheduler:
    1.  Records a start entry in the DB (`program_runs`) to get a unique `RUN_ID`.
    2.  Prepares the job command using Go's `os/exec` package.
    3.  Injects environment variables into the process:
        *   `RUN_ID`: Unique integer mapping to `program_runs(id)`.
        *   `SCHEDULER_SOCKET_PATH`: The path to the IPC Unix Domain Socket.
    4.  Spawns the child process and immediately records its `PID` in the DB.
    5.  Waits asynchronously (in a goroutine) for the process to exit. Once exited, the scheduler updates `program_runs` with the `exit_code` and `success` flag.
    6.  Triggers automatic process restarts on failures if the job config has `restart_on_exit` set to `true`.

### 2.5 Unix Socket IPC (`internal/ipc`)
*   **Location**: [internal/ipc/ipc.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/ipc/ipc.go)
*   **Socket Protocol**: Unix Domain Socket streaming JSON-Lines (newline-delimited JSON payloads).
*   **IPC Server**: Runs as part of the core scheduler. It parses incoming [StatusEvent](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/ipc/ipc.go#L32) payloads from jobs and calls its `OnEvent` handler to persist events to the database.
*   **IPC Client**: Imported by individual jobs (like [cmd/job1/main.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/cmd/job1/main.go) and [cmd/job2/main.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/cmd/job2/main.go)). Jobs dial the socket path and invoke [SendEvent](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/ipc/ipc.go#L95) to stream status changes (`started`, `processing`, `finished`) or custom audit messages to the daemon.

### 2.6 Admin REST API (`internal/http`)
*   **Location**: [internal/http/server.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/internal/http/server.go)
*   **Port**: Configured dynamically in database table `scheduler_config`.
*   **Endpoints**:
    *   `/health` (Public GET): Pings the database to verify connectivity.
    *   `/info` (Public GET): Returns a JSON map of service properties (name, version, description).
    *   `/admin/jobs` (Authenticated GET): Retrieves all scheduled jobs from the database.
    *   `/admin/update-jobs` (Authenticated POST): Upserts a list of jobs into the DB and reloads the cron configuration.
    *   `/admin/delete-job?name=<job_name>` (Authenticated DELETE): Removes a job configuration and reloads the cron configuration.
    *   `/admin/logs/system` (Authenticated GET): Downloads system logs as JSON (accepts optional `from` and `to` query parameters in RFC3339 or YYYY-MM-DD format).
    *   `/admin/logs/job-audit` (Authenticated GET): Downloads job audit logs as JSON (accepts optional `from` and `to` query parameters).
    *   `/admin/logs/admin-audit` (Authenticated GET): Downloads administrative audit logs as JSON (accepts optional `from` and `to` query parameters).
*   **Security**: Uses Basic Authentication. It verifies the credentials against the admin users configured in the encrypted `config.json` loaded at startup.

### 2.7 Cross-Platform GUI Admin Client (`cmd/scheduler-admin`)
*   **Location**: [cmd/scheduler-admin/main.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/cmd/scheduler-admin/main.go)
*   **UI Framework**: `fyne.io/fyne/v2`
*   **Features**:
    *   Provides forms to connect to the REST API server using an authentication token.
    *   Lists scheduled programs, allows editing, creating, and deleting jobs.
    *   Automatically infers the admin user from the current system username.
*   **Windows Hello integration**:
    *   On Windows platforms, [cmd/scheduler-admin/hello_windows.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/cmd/scheduler-admin/hello_windows.go) uses WinRT COM bindings located in [windows/security/credentials/ui/userconsentverifier.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/windows/security/credentials/ui/userconsentverifier.go) to trigger biometric authentication (fingerprint/face recognition) before any API call is allowed to go through.
    *   On non-Windows platforms, [cmd/scheduler-admin/hello_other.go](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/cmd/scheduler-admin/hello_other.go) provides a no-op fallback stub that instantly yields verification success.
