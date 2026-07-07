# Changelog

All notable changes to the MitM Scheduler will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.17.0] - 2026-07-07

### Added
- **SSL Support via Environment Propagation**: The scheduler now parses the `sslmode` boolean from the `config.json` database block and injects it as the `MITM_DB_SSLMODE` environment variable (`true` or `false`) into all spawned child processes. This centralizes SSL configuration for the entire MitM ecosystem.

## [v0.16.0] - 2026-07-02

### Added
- **Database SSL Mode**: Added `sslmode` boolean attribute to the nested `db` configuration in `config.json`. If set to `true`, `?sslmode=require` is appended to the PostgreSQL connection string; otherwise, `?sslmode=disable` is used.

## [v0.15.0] - 2026-06-30

### Added
- **HTTPS/TLS Support**: Added support for dual HTTP and HTTPS bindings. The scheduler can now automatically generate SAN-based self-signed certificates for local HTTPS if configured, or use provided certificates.

### Changed
- **Config Restructuring**: Refactored `config.json` to nest database connection parameters inside a `db` object (e.g. `db_connect_delay`, `host`, `port`, `user`, `password`, `database`).
- **Database Connection**: Updated all layers (Collector, Transformation, Delivery, Maintenance) to read and parse the new nested JSON configuration provided via `MITM_DB_CONFIG_JSON` with fallback to direct environment variables. All connection initializations are now properly audited.
- **Migrations Consolidation**: Consolidated individual `.sql` migration files into a single `setup.sql` script for simpler database initialization. Removed redundant migration folders.

## [v0.14.0] - 2026-06-24

### Changed
- **Component Updates**: Updated bundled `mitm_delivery` to v0.7.0 and `mitm_transformation` to v0.8.0.
- **Envelope Encryption Rollout**: The bundled Transformation and Delivery engines now fully utilize end-to-end Envelope Encryption (AES-GCM) for processing sensitive PII targets, securely isolating Key Encryption Keys (KEK) and Data Encryption Keys (DEK).

### Fixed
- **DLQ mapping fix**
- **updated DLQ querry**

## [v0.13.1] - 2026-06-22

### Fixed
- **Transformation Errors (DLQ)**: Fixed a SQL error in `GetTransformationErrors` causing an `Internal Server Error` (HTTP 500). The query now correctly joins the `raw_ingestion` table via `raw_ingestion_id` to retrieve the `correlation_id` and `topic`, and properly uses a limit.

## [v0.13.0] - 2026-06-21

### Added
- **Topic Dependencies API**: Added `GET`, `POST`, and `DELETE` REST endpoints under `/admin/transformation/topic-dependencies` to manage the required source topics for Stateful Aggregation.
- **Database CRUD**: Extended `db_transformation.go` to support operations on the `topic_dependencies` PostgreSQL table.

### Changed
- **Transformation Errors (DLQ)**: Updated the `GET /admin/transformation/errors` SQL query and JSON response mapping to use `correlation_id` instead of the deprecated `raw_ingestion_id`, aligning with the N:1 Stateful Aggregation logic.

## [v0.12.0] - 2026-06-16

### Added
- **Transformation Errors API**: Added `GET /admin/transformation/errors` REST endpoint for fetching transformation errors (DLQ) along with the associated topic from the raw ingestion table.
- **API Documentation**: Fully documented all transformation, configuration, log, and RBAC endpoints in `docs/api/README.md`.

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
