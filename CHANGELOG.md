# Changelog

All notable changes to the MitM Scheduler will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [v0.11.0] - 2026-06-15

### Added
- **Centralized App Info**: Added `AppName`, `AppDescription`, and `AppVersion` centrally in `main.go` and exposed them via the HTTP server.
- **Dynamic Versioning**: Added support for overriding the application version at compile time via `go build -ldflags "-X main.version=$MITM_SERVER"`.
- **Startup Logging**: The scheduler now logs its name and version upon successful startup.

## [v0.10.0] - 2026-06-14

### Added
- **RBAC API**: Added full REST API for Role-Based Access Control (`/admin/rbac/*`).
- **Database User Management**: Added support for creating, deleting, and fetching users from the `admin_users` table with Argon2 password hashing.
- **Envelope Encryption for Roles**: Role assignments are securely AES-GCM encrypted and stored in the database (`user_roles_encrypted`).
- **Audit Logging for RBAC**: All RBAC modifications (create user, delete user, assign roles) are natively logged to the `admin_audit_logs` table.
- **CLI Utility**: New `create-admin` Go command-line tool added to initialize the first DB admin user.

## [v0.9.0] - 2026-06-14

### Added
- **File Upload API**: Added `/admin/upload/source_file` REST endpoint to allow uploading CSV/XLSX files via the frontend.
- **Dynamic File Collector Trigger**: The scheduler dynamically resolves its own path and directly executes `mitm_collector_csv-xls` for immediate processing of uploaded files.

## [v0.8.0] - 2026-06-10

### Added
- **Dead Letter Queue API**: Added `GET /admin/dlq` REST endpoint for fetching Dead Letter Queue (DLQ) entries.
- **Audit Logs Component**: Added `component` column to `job_audit_logs` and updated IPC `StatusEvent` to parse the `component` field for better log attribution.

## [v0.7.0] - 2026-06-09

### Added
- **Auto-Map API**: Added `/admin/transformation/auto-map` REST endpoint that accepts comma-separated source headers and returns Levenshtein-distance matched mappings against target fields.

## [v0.6.0] - 2026-06-06

### Added
- Scheduler now automatically injects `MITM_DB_*` environment variables (e.g., `MITM_DB_HOST`, `MITM_DB_PORT`, `MITM_DB_USER`, `MITM_DB_PASSWORD`, `MITM_DB_NAME`, `MITM_DB_CONFIG_JSON`) into child processes, providing target database connection details without requiring CLI arguments.

## [v0.5.0] - 2026-06-05

### Added
- Migration `005_change_args_to_jsonb.sql` converting job arguments column in `scheduled_programs` from `TEXT[]` to `JSONB`.
- Integrated JSON validation in `scheduler-admin` arguments input box (Fyne GUI client).
- Support in scheduler daemon to deserialize `JSONB` args and forward them to executed collectors as a serialized JSON string in `os.Args[2]`.

### Changed
- Updated database schema documentation to reflect JSONB representation.
- Updated README documentation with instructions on JSONB overrides.

## [v0.4.0] - 2026-06-04

### Added
- HTTP REST API endpoints to download `system`, `job_status_events`, `job_audit_logs`, and `admin_audit_logs` filtered by date ranges.
- Integrated downloading and local file saving of logs directly from the `scheduler-admin` Fyne GUI.
- Added HELO authentication handshake for remote API connections.

## [v0.1.0] - 2026-06-03

### Added
- Core Scheduler engine supporting Linux cron schedules.
- AES-256-GCM encrypted database connection configurations.
- Unix Domain Socket IPC listener for receiving runtime status/audit notifications from running collectors.
- Database auditing schema including `system_logs`, `job_status_events`, and `job_audit_logs`.
- Multi-platform `scheduler-admin` client built with Fyne.
