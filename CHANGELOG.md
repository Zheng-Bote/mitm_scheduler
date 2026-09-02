# Changelog

All notable changes to the MitM Scheduler will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.30.2] - 2026-09-02

### Changed

- **API**: Improved formatting of the `db_version` field in the `/admin/dashboard/stats` endpoint. It now returns a shortened, readable version string (e.g., "PostgreSQL 18.2") instead of the verbose compiler output.

## [v0.30.1] - 2026-09-02

### Fixed

- **API**: Fixed an issue where the `/info` endpoint occasionally returned an empty version string. Corrected the variable name in the build script and implemented a graceful fallback to the default version in the code (Issue #2).

## [v0.30.0] - 2026-09-02

### Added

- **API**: Added `/admin/dashboard/stats` endpoint to serve PostgreSQL database statistics (name, version, database size) and Dead Letter Queue (DLQ) entry counts to the frontend Dashboard (Issue #4).

## [v0.29.0] - 2026-08-31

### Added

- **IPC Socket as Credential Broker**: The scheduler now acts as a credential broker for the collector, transformation, delivery, and cleanup components. It serves database credentials and the master key over its Unix Domain Socket in response to `get_credentials` requests (identified via `RUN_ID`).

### Changed

- **Administrative RBAC Authorization**: Hardened `handleDLQ` and `handleDLQBin` to require authentication and upgraded `handleGetOsUserRoles` to enforce `requireAdmin`, blocking unauthenticated or unprivileged access to administrative functionality.

## [v0.28.0] - 2026-08-29

### Changed/Added

- Configured `pgxpool` connection limits (`MaxConns=20`, `MaxConnIdleTime=5m`, `MaxConnLifetime=1h`).
- Implemented graceful shutdown with context cancellation on `SIGINT`/`SIGTERM`.
- Added/updated DLQ and error tracking mechanisms.
- Improved security using `subtle.ConstantTimeCompare` for sensitive comparisons.

## [v0.27.0] - 2026-08-17

### Fixed

- **Job Concurrency Control**: Implemented an in-memory job control map (`s.running`) in the scheduler to strictly prevent overlapping concurrent executions of the _same_ scheduled job program, while allowing different jobs to run in parallel.
- **Orphaned Runs Cleanup**: Added an automated database cleanup step during scheduler startup. Any previous `program_runs` left incomplete (`finished_at IS NULL`) due to daemon crashes are now correctly marked as failed (`exit_code = -1`, `success = false`), preventing them from permanently appearing active in the DB.

## [v0.26.0] - 2026-08-15

### Security & Hardening

- **HTTP Server**: Upgraded the internal web server to production-readiness by implementing comprehensive safeguards:
  - **Timeouts**: Added explicit `ReadTimeout`, `ReadHeaderTimeout`, `WriteTimeout`, and `IdleTimeout` to mitigate Slowloris and connection starvation attacks.
  - **Graceful Shutdown**: The HTTP server now listens to termination contexts and shuts down cleanly alongside the job scheduler.
  - **Panic Recovery Middleware**: Added `Recover` middleware to intercept unhandled panics and return HTTP 500 without crashing the process.
  - **Request Size Limits**: Implemented `LimitBody` middleware to cap incoming JSON requests at 1 MB, and explicitly raised it to 100 MB for `/admin/upload/source_file` to support CSV/XLSX uploads.
  - **Context Propagation**: Updated all HTTP handlers to correctly pass `r.Context()` down to the database layer, ensuring background cancellation.
  - **Correlation IDs**: `WithRequestID` middleware now assigns and tracks a UUID (`X-Request-ID`) for every request.
  - **Concurrency Throttling**: Added a `Throttle` middleware to cap active connections (max 100) and return HTTP 503 during extreme traffic spikes.

## [v0.25.0] - 2026-08-10

### Added

- **API**: Implemented `/admin/execute-job?name=<JobName>` endpoint to manually trigger the immediate execution of a scheduled job outside of its regular cron schedule.
- **API**: Implemented `/admin/dlq/requeue?id=...` endpoint to requeue specified Dead Letter Queue (DLQ) entries back into the delivery packages queue and remove them from the DLQ.

## [v0.24.0] - 2026-08-09

### Added

- **Backup & Restore API**: Implemented `/admin/backup` and `/admin/restore` endpoints to export and import the complete system configuration (jobs, sources, targets, rules) as JSON.
- **RBAC**: Introduced the `BACKUP-RESTORE` role to control access to the configuration backup and restore functionality.

## [v0.23.0] - 2026-07-29

### Added

- **Delivery Layer**: Implemented configurable `slowdown` and `timeout` parameters for the `CORITY_SAAS` delivery adapter.

### Changed

- **Database**: Synced PostgreSQL database schema IST-Zustand across all layer `.sql` migrations (`setup.sql`, `transformation-layer`, `delivery-layer`, `scheduler`).
- **Components Logging**: Refactored component version logging mechanism across all layers (Collectors, Transformation, Delivery, Scheduler) to consistently output a clean `Major.Minor.Patch` version format.

### Fixed

- **Scheduler**: Resolved an HTTP 500 error on the `/admin/transformation/errors_bin` API endpoint by updating the query to correctly reference the `raw_ingestion_id` column and gracefully handle null values.

## [v0.22.0] - 2026-07-27

### Added

- **Job Cancellation / Stopping**: Added `StopJobByName(name string)` in the scheduler to cleanly terminate active job processes via `SIGTERM` (with a 5-second `SIGKILL` fallback).
- **Admin API Job Termination**: Implemented `POST/DELETE /admin/stop-job?name=<job_name>` endpoint to stop running jobs. Restricted execution strictly to users with the `ADMIN` role.
- **Active State Tracking**: Extended `/admin/jobs` endpoint to report real-time execution status (`is_running` boolean and `active_pid` integer) for scheduled programs.
- **Audit Logging**: Added `stop_job` and `stop_job_forbidden` audit events to log job termination attempts and RBAC violations.

## [v0.21.0] - 2026-07-26

### Added

- **FlatBuffers Binary API Endpoints**: Implemented new high-performance FlatBuffers binary endpoints (`/admin/dlq_bin`, `/admin/logs/system_bin`, `/admin/logs/job-audit_bin`, `/admin/logs/admin-audit_bin`, and `/admin/transformation/errors_bin`).
- **FlatBuffers Schemas**: Added comprehensive FlatBuffers schemas (`dlq.fbs`, `system_logs.fbs`, `job_audit_logs.fbs`, `admin_audit_logs.fbs`, and `transformation_errors.fbs`) in `schematas/` with dual timestamp representations (RFC3339 string and Unix epoch milliseconds) and generated Go bindings.
- **Documentation**: Updated API documentation and README to cover the new FlatBuffers binary endpoints.

## [v0.20.0] - 2026-07-21

### Fixed

- **HTTPS Server Startup Logging**: Enhanced the HTTP server to properly log startup errors when `UseHTTPS` is true (e.g. if the SSL certificate or key cannot be found on the filesystem), instead of silently failing and swallowing the `ListenAndServeTLS` error.

## [v0.19.0] - 2026-07-08

### Fixed

- **HTTP Server Startup**: Fixed a critical bug causing nil pointer dereferences in the HTTP server by strictly delaying the server start until the database connection logic (including `db_connect_delay` and retries) has successfully completed.
- **Runtime Nil Checks**: Added safeguard checks (guard clauses) to API endpoints in `server.go` to return an HTTP 503 instead of panicking if the database connection object is nil.

## [v0.18.0] - 2026-07-08

### Added

- **Server Time API**: Added a new unauthenticated `/time` JSON endpoint to expose the server's local time and timezone for frontend synchronization.
- **Scheduler Next Run**: Extended the `/admin/jobs` API response to automatically compute and attach the `next_run` timestamp for enabled scheduled programs using `github.com/robfig/cron/v3`.

### Fixed

- **Transformation Errors Schema Updates**: Fixed the `GetTransformationErrors` SQL query to align with the latest database schema. Dropped the outdated join on `raw_ingestion` and now fetches `correlation_id` directly from the `transformation_errors` table. Additionally, fixed an untyped literal issue (`''::text AS topic`) that caused the PostgreSQL driver (`pgx`) to fail during `rows.Scan`, resolving HTTP 500 errors on the `/admin/transformation/errors` endpoint.

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
