<div id="top" align="center">
<h1>MitM Scheduler</h1>

<p>A Linux command-line scheduler that executes external Go programs based on a PostgreSQL configuration.</p>


![License](https://img.shields.io/badge/license-Apache_2.0-green)
![Platform](https://img.shields.io/badge/platform-Linux-lightgrey.svg)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/Zheng-Bote/mitm_scheduler?logo=GitHub)](https://github.com/Zheng-Bote/mitm_scheduler/releases)
<br/>
[Report Issue](https://github.com/Zheng-Bote/mitm_scheduler/issues) · [Request Feature](https://github.com/Zheng-Bote/mitm_scheduler/pulls)
</div>

---
<!-- START doctoc generated TOC please keep comment here to allow auto update -->

<details>
<summary>Table of Contents</summary>

- [Features](#features)
  - [Status](#status)
- [Project Structure](#project-structure)
- [Setup & Build](#setup-build)
  - [1. Prerequisites](#1-prerequisites)
  - [2. Build](#2-build)
  - [3. Database Setup](#3-database-setup)
  - [4. Configuration](#4-configuration)
- [Running the Scheduler](#running-the-scheduler)
  - [Generating a MASTER_KEY](#generating-a-master_key)
- [Administrative Tools](#administrative-tools)
  - [1. GUI Admin Tool (Fyne)](#1-gui-admin-tool-fyne)
  - [2. Remote REST API](#2-remote-rest-api)
    - [Update or Create Jobs:](#update-or-create-jobs)
    - [Stop a Running Job (Requires ADMIN Role):](#stop-a-running-job-requires-admin-role)
    - [Download Database Logs with Date Filtering:](#download-database-logs-with-date-filtering)
    - [Download Logs as FlatBuffers Binary:](#download-logs-as-flatbuffers-binary)
- [Injected Environment Variables](#injected-environment-variables)
- [IPC & Job Communication](#ipc-job-communication)
- [Docker](#docker)
- [Key Rotation](#key-rotation)

</details>

<!-- END doctoc generated TOC please keep comment here to allow auto update -->
---

## Features

- **Cron Scheduling**: Jobs are scheduled via standard cron expressions.
- **Encrypted Config**: Database credentials and admin tokens are stored in an encrypted JSON file (AES-256-GCM + Argon2id).
- **Dynamic Reloading**: Jobs can be updated via the API and reloaded without restarting the service.
- **IPC over Unix Sockets**: Jobs report status events back to the scheduler via JSON-Lines.
- **Admin API**: Remote job management with authentication and RBAC, including automatic `next_run` cron calculations, active job termination (`/admin/stop-job` for `ADMIN` role), and high-performance FlatBuffers binary endpoints (`/admin/*_bin`).
- **Configuration Backup & Restore**: Export and import the complete system configuration (jobs, sources, targets, rules) as JSON via API (requires `BACKUP-RESTORE` role).
- **Enhanced Logging**:
  - `system_logs`: Core scheduler lifecycle events.
  - `job_status_events`: Real-time job progress tracking.
  - `job_audit_logs`: Custom audit messages from jobs.
  - `admin_audit_logs`: Audit trail for administrative actions via API.
- **HTTP API**: Health check (`/health`), info (`/info`), and server local time (`/time`) endpoints.

### Status

![GitHub Created At](https://img.shields.io/github/created-at/Zheng-Bote/mitm_scheduler?logo=GitHub)
![GitHub Release Date](https://img.shields.io/github/release-date/Zheng-Bote/mitm_scheduler?logo=GitHub)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/Zheng-Bote/mitm_scheduler?logo=GitHub)](https://github.com/Zheng-Bote/mitm_scheduler/releases)
![GitHub repo size](https://img.shields.io/github/repo-size/zheng-bote/mitm_scheduler)

![Status](https://img.shields.io/badge/Status-stable-green)

## Project Structure

- `cmd/scheduler`: Main application entry point.
- `cmd/encrypt-config`: Utility to encrypt a JSON configuration file.
- `cmd/job1`, `cmd/job2`: Example jobs demonstrating IPC and Audit features.
- `internal/`: Core logic (crypto, db, ipc, http, scheduler).
- `schematas/`: FlatBuffers schemas (`*.fbs`) and generated Go bindings for high-performance binary APIs.
- `migrations/`: SQL files for database setup.

## Setup & Build

### 1. Prerequisites

- Go 1.25+
- PostgreSQL Server

### 2. Build

```bash
go build -o ./bin/mitm-server ./cmd/scheduler
go build -o ./bin/encrypt-config ./cmd/encrypt-config
go build -o ./bin/job1 ./cmd/job1
go build -o ./bin/job2 ./cmd/job2
# Linux
go build -o ./bin/scheduler-admin ./cmd/scheduler-admin
# Windows
go build -v -ldflags="-H=windowsgui" -o ./bin/scheduler-admin.exe ./cmd/scheduler-admin/main.go ./cmd/scheduler-admin/hello_windows.go
```

### 3. Database Setup

Apply the migrations in order:

```bash
psql -h <host> -U <user> -d <db> -f migrations/001_init.sql
psql -h <host> -U <user> -d <db> -f migrations/002_logging_and_audit.sql
psql -h <host> -U <user> -d <db> -f migrations/003_admin_and_api.sql
psql -h <host> -U <user> -d <db> -f migrations/004_add_name_unique.sql
psql -h <host> -U <user> -d <db> -f migrations/005_change_args_to_jsonb.sql
psql -h <host> -U <user> -d <db> -f migrations/006_rbac.sql
```

### 4. Configuration

Create a `config.json` (see `example_config.json` for a template):

```json
{
  "db": {
    "host": "your-db-host",
    "port": 5432,
    "user": "your-user",
    "password": "your-password",
    "database": "your-dbname",
    "db_connect_delay": 30,
    "sslmode": false,
    "max_conns": 50
  },
  "http_port": 8080,
  "use_https": false,
  "ssl_cert": "server.crt",
  "ssl_key": "server.key",
  "log_level": "DEBUG",
  "upload_dir": "/tmp/mitm_uploads",
  "admins": [
    {
      "username": "admin1",
      "token": "your_secure_token"
    }
  ]
}
```

Encrypt it:

```bash
./encrypt-config config.json config.json.enc
```

## Running the Scheduler

The Scheduler requires two critical environment variables to start:
1. `SCHEDULER_PASSWORD`: The password used to decrypt the `config.json.enc` file.
2. `MASTER_KEY`: The base64-encoded Key Encryption Key (KEK) used for Envelope Encryption.

```bash
export SCHEDULER_PASSWORD="your_secure_password"
export MASTER_KEY="your_base64_master_key"
./scheduler config.json.enc
```

### Generating a MASTER_KEY

The `MASTER_KEY` must be a cryptographically secure 32-byte key, encoded in Base64. You can generate a new one using either `openssl` or Go:

**Using OpenSSL:**
```bash
openssl rand -base64 32
```

**Using Go:**
```bash
go run -e 'import ("crypto/rand"; "encoding/base64"; "fmt"); b := make([]byte, 32); rand.Read(b); fmt.Println(base64.StdEncoding.EncodeToString(b))'
```

> **Warning:** Do not change the `MASTER_KEY` on an existing installation without first re-wrapping all existing Data Encryption Keys (DEKs) in the `storage_keys` database table. If you start the scheduler with a new `MASTER_KEY`, it will fail to decrypt existing payloads!

## Administrative Tools

### 1. GUI Admin Tool (Fyne)

A cross-platform GUI application is available in `cmd/scheduler-admin`. It allows you to:

- Connect to the scheduler via URL and HELO token.
- List all current jobs.
- Create, Edit, and Delete jobs.
- Automatically reload the scheduler after changes.

**Build & Run:**

```bash
cd cmd/scheduler-admin
go build -o scheduler-admin
./scheduler-admin
```

### 2. Remote REST API

For a detailed list of all endpoints, query parameters, and roles, see the [REST API Documentation](file:///home/zb_bamboo/DEV/__NEW__/Go/go_scheduler/docs/api/README.md).

#### Update or Create Jobs:

```bash
curl -u "admin1:your_secure_token" -X POST -H "Content-Type: application/json" \
     -d '[{
            "name": "RemoteJob",
            "command": "./bin/mitm-collector-pg-employee",
            "args": {
              "source_name": "PG_EMPLOYEE",
              "table": "employees"
            },
            "cron_expr": "*/2 * * * *",
            "enabled": true,
            "restart_on_exit": false
          }]' \
     http://localhost:8080/admin/update-jobs
```

This request will:

1. Authenticate the user.
2. Upsert the job into the database (by name).
3. Trigger a scheduler reload.
4. Log the action in `admin_audit_logs`.

#### Stop a Running Job (Requires ADMIN Role):

```bash
curl -u "admin1:your_secure_token" -X POST \
     "http://localhost:8080/admin/stop-job?name=RemoteJob"
```

#### Download Database Logs with Date Filtering:

```bash
# Download system logs between June 1st and June 2nd, 2026
curl -u "admin1:your_secure_token" -X GET \
     "http://localhost:8080/admin/logs/system?from=2026-06-01&to=2026-06-02" \
     -o system_logs.json
```

#### Download Logs as FlatBuffers Binary:

```bash
# Download system logs as FlatBuffers binary
curl -u "admin1:your_secure_token" -X GET \
     "http://localhost:8080/admin/logs/system_bin?from=2026-06-01&to=2026-06-02" \
     -o system_logs.bin
```

## Injected Environment Variables

When the scheduler starts a job, it securely passes configuration and context via environment variables:

- `RUN_ID`: The unique ID of the current job execution.
- `SCHEDULER_SOCKET_PATH`: The path to the Unix domain socket for IPC communication.
**Note on Security (Master Key & DB Config):**
For security reasons, sensitive data like the **MASTER_KEY** (KEK) and the database credentials (`MITM_DB_CONFIG_JSON`) are **NOT** injected as environment variables, to prevent accidental exposure via `ps` or crash dumps. 
Instead, child processes (jobs) must fetch these credentials securely at runtime by sending a `get_credentials` IPC request to the scheduler via the Unix domain socket.

## IPC & Job Communication

Jobs can send JSON messages to the Unix Domain Socket specified in `SCHEDULER_SOCKET_PATH`.

**Types of messages:**

- **Status (default)**: Updates `job_status_events`.
- **Audit**: Updates `job_audit_logs`.

Example (Go):

```go
client.SendEvent(ipc.StatusEvent{
    RunID:   runID,
    Type:    "audit",
    Message: "Sensitive operation performed",
})
```

## Docker

Build:

```bash
docker build -t go-scheduler .
```

Run:

```bash
docker run -p 8080:8080 -e SCHEDULER_PASSWORD=mypassword go-scheduler ./scheduler /app/config.json.enc
```
## Key Rotation

The MitM Data Aggregator supports native on-the-fly key rotation via the /admin/key-rotation API endpoint. The new Master Key must be encrypted with the current Master Key. During rotation, all active jobs are paused, the Data Encryption Keys (DEKs) in the database are re-encrypted with the new KEK, the in-memory KEK is updated, and normal operation resumes automatically.

