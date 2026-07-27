# Go Scheduler - REST API Documentation

This document describes the REST API endpoints provided by the Go Scheduler server.

The server's HTTP engine is implemented in the `internal/http` package. The port is dynamically loaded from the database configuration.

---

## 1. API Endpoints Table

### General & Health
| URL | Method | Description | Options / Parameters | Authentication / Role |
| :--- | :--- | :--- | :--- | :--- |
| `/health` | `GET` | Health check (verifies database ping) | None | Public (Unauthenticated) |
| `/info` | `GET` | Service descriptor, name, version | None | Public (Unauthenticated) |

### Job Management
| URL | Method | Description | Options / Parameters | Authentication / Role |
| :--- | :--- | :--- | :--- | :--- |
| `/admin/jobs` | `GET` | List all scheduled job configs (includes `next_run`, `is_running`, and `active_pid`) | None | Admin (HTTP Basic Auth) |
| `/admin/update-jobs`| `POST` | Create or update one or more job configurations | Request Body: JSON array of job configs | Admin (HTTP Basic Auth) |
| `/admin/delete-job` | `DELETE`| Remove a job configuration and reload scheduler | `name` (Query parameter) | Admin (HTTP Basic Auth) |
| `/admin/stop-job` | `POST`, `DELETE` | Send termination signal (SIGTERM/SIGKILL) to stop an active job | `name` (Query parameter) | Admin (HTTP Basic Auth + RBAC `ADMIN` role) |

### Uploads & DLQ
| URL | Method | Description | Options / Parameters | Authentication / Role |
| :--- | :--- | :--- | :--- | :--- |
| `/admin/upload/source_file` | `POST` | Upload a data source file | Request Body: multipart/form-data | Admin (HTTP Basic Auth) |
| `/admin/dlq` | `GET` | Get Dead Letter Queue entries | None | Public (Unauthenticated) |
| `/admin/dlq_bin` | `GET` | Get Dead Letter Queue entries as FlatBuffers binary | None | Public (Unauthenticated) |

### Configuration (Credentials & Delivery Targets)
| URL | Method | Description | Options / Parameters | Authentication / Role |
| :--- | :--- | :--- | :--- | :--- |
| `/admin/credentials` | `GET`, `POST` | List or update source credentials | Request Body: JSON | Admin (HTTP Basic Auth) |
| `/admin/delivery_targets`| `GET`, `POST`, `DELETE` | List, update, or delete delivery targets | Request Body: JSON, or `id` (Query parameter) | Admin (HTTP Basic Auth) |

### Logs & Auditing
| URL | Method | Description | Options / Parameters | Authentication / Role |
| :--- | :--- | :--- | :--- | :--- |
| `/admin/logs/system`| `GET` | Download system logs as a JSON file | `from`, `to` (Query parameters, optional) | Admin (HTTP Basic Auth) |
| `/admin/logs/system_bin`| `GET` | Download system logs as a FlatBuffers binary file | `from`, `to` (Query parameters, optional) | Admin (HTTP Basic Auth) |
| `/admin/logs/job-audit`| `GET` | Download job audit logs as a JSON file | `from`, `to` (Query parameters, optional) | Admin (HTTP Basic Auth) |
| `/admin/logs/job-audit_bin`| `GET` | Download job audit logs as a FlatBuffers binary file | `from`, `to` (Query parameters, optional) | Admin (HTTP Basic Auth) |
| `/admin/logs/admin-audit`| `GET`| Download administrative audit logs as JSON | `from`, `to` (Query parameters, optional) | Admin (HTTP Basic Auth) |
| `/admin/logs/admin-audit_bin`| `GET`| Download administrative audit logs as FlatBuffers binary | `from`, `to` (Query parameters, optional) | Admin (HTTP Basic Auth) |
| `/admin/action` | `POST` | Explicitly log an administrative action | Request Body: JSON | Admin (HTTP Basic Auth) |

### Role-Based Access Control (RBAC)
| URL | Method | Description | Options / Parameters | Authentication / Role |
| :--- | :--- | :--- | :--- | :--- |
| `/admin/rbac/roles` | `GET` | List all roles | None | Admin (HTTP Basic Auth) |
| `/admin/rbac/users` | `GET` | List all users | None | Admin (HTTP Basic Auth) |
| `/admin/rbac/user/create` | `POST` | Create a new user | Request Body: JSON | Admin (HTTP Basic Auth) |
| `/admin/rbac/user/delete` | `DELETE` | Delete a user | `id` (Query parameter) | Admin (HTTP Basic Auth) |
| `/admin/rbac/assign` | `POST` | Assign roles to a user | Request Body: JSON | Admin (HTTP Basic Auth) |
| `/admin/rbac/user_roles` | `GET` | Get roles assigned to a user | `user_id` (Query parameter) | Admin (HTTP Basic Auth) |
| `/admin/rbac/os_user_roles` | `GET` | Get roles for an OS user | `os_user` (Query parameter) | Public (Unauthenticated) |

### Transformation Layer
| URL | Method | Description | Options / Parameters | Authentication / Role |
| :--- | :--- | :--- | :--- | :--- |
| `/admin/transformation/sources` | `GET`, `POST`, `DELETE` | Manage data mapping sources | Request Body: JSON, or `id` (Query parameter) | Admin (HTTP Basic Auth) |
| `/admin/transformation/targets` | `GET`, `POST`, `DELETE` | Manage data mapping targets | Request Body: JSON, or `id` (Query parameter) | Admin (HTTP Basic Auth) |
| `/admin/transformation/rules` | `GET`, `POST`, `DELETE` | Manage data mapping rules | Request Body: JSON, or `id` (Query parameter) | Admin (HTTP Basic Auth) |
| `/admin/transformation/transformations`| `GET`, `POST` | Manage data mapping transformations | Request Body: JSON | Admin (HTTP Basic Auth) |
| `/admin/transformation/validations` | `GET`, `POST`, `DELETE`| Manage data mapping validations | Request Body: JSON, or `id` (Query parameter) | Admin (HTTP Basic Auth) |
| `/admin/transformation/auto-map` | `POST` | Auto-map source to target | Request Body: JSON | Admin (HTTP Basic Auth) |
| `/admin/transformation/errors` | `GET` | Get transformation errors (DLQ) with topic | None | Admin (HTTP Basic Auth) |
| `/admin/transformation/errors_bin` | `GET` | Get transformation errors as FlatBuffers binary | None | Admin (HTTP Basic Auth) |

---

## 2. Endpoints Detail (Core Examples)

*Note: The following examples illustrate typical responses. All endpoints in the `admin/` namespace (except `/admin/dlq` and `/admin/rbac/os_user_roles`) require HTTP Basic Authentication using the configured admin credentials.*

### 2.1 Health Check
*   **Path**: `/health`
*   **Method**: `GET`
*   **Response**: `200 OK` (Body: `OK`) or `500 Internal Server Error`

### 2.2 Info
*   **Path**: `/info`
*   **Method**: `GET`
*   **Response**: `200 OK` (Content-Type: `application/json`)
    ```json
    {
      "name": "Go Scheduler",
      "description": "A Linux commandline scheduler",
      "version": "1.0.0"
    }
    ```

### 2.3 List Jobs
*   **Path**: `/admin/jobs`
*   **Method**: `GET`
*   **Response**:
    ```json
    [
      {
        "id": 1,
        "name": "Job1",
        "command": "./bin/job1",
        "args": [],
        "cron_expr": "*/5 * * * *",
        "enabled": true,
        "restart_on_exit": false,
        "next_run": "2026-07-27T14:05:00+02:00",
        "is_running": true,
        "active_pid": 12345
      }
    ]
    ```

### 2.4 Create / Update Jobs
*   **Path**: `/admin/update-jobs`
*   **Method**: `POST`
*   **Request Body**: JSON array of job structures.
    ```json
    [
      {
        "name": "RemoteJob",
        "command": "./job1",
        "args": [],
        "cron_expr": "*/2 * * * *",
        "enabled": true,
        "restart_on_exit": false
      }
    ]
    ```
*   **Response**: `200 OK` (Body: `Jobs updated and scheduler reloaded`)

### 2.5 Delete Job
*   **Path**: `/admin/delete-job`
*   **Method**: `DELETE`
*   **Parameters**: `name` (Query parameter) - exact unique name of the program configuration.
*   **Response**: `200 OK` (Body: `Job deleted`)

### 2.6 Stop Job
*   **Path**: `/admin/stop-job`
*   **Method**: `POST`, `DELETE`
*   **Parameters**: `name` (Query parameter) - exact unique name of the currently running program configuration.
*   **Authentication / Role**: Requires valid HTTP Basic Auth credentials and the user must have the `ADMIN` role (via config or database RBAC).
*   **Response**: `200 OK` (Body: `Stop signal sent to job`) or `403 Forbidden` if user lacks `ADMIN` role.


### 2.7 Download Logs
*   **Paths**: `/admin/logs/system`, `/admin/logs/job-audit`, `/admin/logs/admin-audit`
*   **Method**: `GET`
*   **Parameters** (Optional): `from`, `to` (RFC3339 timestamp or YYYY-MM-DD date)
*   **Response**: Returns an `application/json` stream with a `Content-Disposition` attachment header.

### 2.8 FlatBuffers Binary API Endpoints
*   **Paths**: `/admin/dlq_bin`, `/admin/logs/system_bin`, `/admin/logs/job-audit_bin`, `/admin/logs/admin-audit_bin`, `/admin/transformation/errors_bin`
*   **Method**: `GET`
*   **Parameters** (for log download endpoints): `from`, `to` (RFC3339 timestamp or YYYY-MM-DD date, optional)
*   **Response**: Returns an `application/octet-stream` binary stream encoded using Google FlatBuffers schemas (defined in `scheduler/mitm_scheduler/schematas/`). For log download endpoints (`/admin/logs/*_bin`), a `Content-Disposition` attachment header is also included (e.g. `filename=system_logs.bin`).
