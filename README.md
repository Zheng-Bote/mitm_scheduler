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
- [Administrative Tools](#administrative-tools)
  - [1. GUI Admin Tool (Fyne)](#1-gui-admin-tool-fyne)
  - [2. Remote REST API](#2-remote-rest-api)
    - [Update or Create Jobs:](#update-or-create-jobs)
    - [Download Database Logs with Date Filtering:](#download-database-logs-with-date-filtering)
- [Injected Environment Variables](#injected-environment-variables)
- [IPC & Job Communication](#ipc-job-communication)
- [Docker](#docker)

</details>

<!-- END doctoc generated TOC please keep comment here to allow auto update -->
---

## Features

- **Cron Scheduling**: Jobs are scheduled via standard cron expressions.
- **Encrypted Config**: Database credentials and admin tokens are stored in an encrypted JSON file (AES-256-GCM + Argon2id).
- **Dynamic Reloading**: Jobs can be updated via the API and reloaded without restarting the service.
- **IPC over Unix Sockets**: Jobs report status events back to the scheduler via JSON-Lines.
- **Admin API**: Remote job management with authentication.
- **Enhanced Logging**:
  - `system_logs`: Core scheduler lifecycle events.
  - `job_status_events`: Real-time job progress tracking.
  - `job_audit_logs`: Custom audit messages from jobs.
  - `admin_audit_logs`: Audit trail for administrative actions via API.
- **HTTP API**: Health check (`/health`) and info (`/info`) endpoints.

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
    "sslmode": false
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

```bash
./scheduler config.json.enc
```

Decryption password can be provided via prompt or `SCHEDULER_PASSWORD` environment variable.

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

#### Download Database Logs with Date Filtering:

```bash
# Download system logs between June 1st and June 2nd, 2026
curl -u "admin1:your_secure_token" -X GET \
     "http://localhost:8080/admin/logs/system?from=2026-06-01&to=2026-06-02" \
     -o system_logs.json
```

## Injected Environment Variables

When the scheduler starts a job, it securely passes configuration and context via environment variables:

- `RUN_ID`: The unique ID of the current job execution.
- `SCHEDULER_SOCKET_PATH`: The path to the Unix domain socket for IPC communication.
- `MITM_DB_CONFIG_JSON`: The complete, raw database configuration JSON string containing the nested `"db"` structure.
- `MITM_DB_HOST`: Target MitM database hostname.
- `MITM_DB_PORT`: Target MitM database port.
- `MITM_DB_USER`: Target MitM database username.
- `MITM_DB_PASSWORD`: Target MitM database password.
- `MITM_DB_NAME`: Target MitM database name.
- `MITM_DB_SSLMODE`: Derived from the `sslmode` attribute in `config.json` (`true` or `false`). Instructs child processes whether to enforce SSL connections.

These variables allow data collectors to connect to the central MitM database or communicate with the scheduler without relying on CLI arguments or local configuration files.

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
