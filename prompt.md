# Detailed Cursor AI Prompts for Each File/Module in Your Go Agent

---

### 1. `cmd/agent/main.go`

**Prompt:**
Create the main entry point for the Go deployment agent. This should:

* Load configuration from environment variables and config files.
* Initialize logging and audit systems with appropriate levels.
* Authenticate the agent securely with the backend API using stored API tokens.
* Register the agent on startup and handle token refresh if necessary.
* Start the main event loop including heartbeat sending, deployment command watchers, and API polling or websocket listeners.
* Handle graceful shutdown on system signals (SIGINT, SIGTERM) with cleanup of running containers and resources.
* Include comprehensive error handling and panic recovery.
* Avoid logging any sensitive information like tokens or secrets.

---

### 2. `config/config.go` & `config.yaml`

**Prompt:**
Implement configuration management for the Go deployment agent.

* Support loading config from environment variables and YAML config files.
* Define configuration structs covering API endpoints, authentication tokens, server metadata (ID, location), resource limits, and Traefik integration.
* Include validation for required fields and acceptable value ranges.
* Support dynamic config reload on file change if possible.
* Securely handle any encrypted config values.

Provide a sample `config.yaml` illustrating all configurable parameters.

---

### 3. `internal/api/client.go`, `auth.go`, `requests.go`

**Prompt:**
Develop a secure API client package for the agent to communicate with the backend server.

* Authenticate every request using API tokens managed by the `auth` package.
* Implement retry logic with exponential backoff for transient errors.
* Provide functions to register the agent, report status and resource usage, fetch deployment jobs, and stream logs.
* Support both HTTP polling and websocket (or long-polling) for real-time command updates.
* Validate TLS certificates and securely store tokens in memory only.
* Handle token expiration and refresh seamlessly.
* Parse backend JSON responses into appropriate Go structs with error checking.

---

### 4. `internal/auth/token_manager.go`, `token_store.go`

**Prompt:**
Build token lifecycle management for API authentication.

* Load and decrypt API tokens securely from disk at startup.
* Provide thread-safe access to tokens for API client use.
* Support token refresh, rotation, and revocation triggered by backend commands or admin actions.
* Store tokens encrypted on disk using strong encryption (e.g., AES).
* Log all token lifecycle events to audit logs without exposing secrets.
* Fail safe on missing or invalid tokens, prompting re-authentication or admin intervention.

---

### 5. `internal/deploy/deploy.go`, `git.go`, `docker.go`, `lifecycle.go`, `resource_limits.go`

**Prompt:**
Create the deployment engine for the agent.

* `git.go`: Clone private GitHub repos securely using SSH keys or OAuth tokens, checkout specific commits, branches, or tags as instructed.
* `docker.go`: Manage Docker images and containers for deployments — pulling images, building containers, injecting environment variables securely.
* `deploy.go`: Orchestrate deployment workflows — from code fetch to build and run.
* `lifecycle.go`: Support app lifecycle operations — start, stop, restart, update, rollback with zero downtime (use blue-green or canary strategies).
* `resource_limits.go`: Enforce CPU, memory, disk, and network limits on containers using Docker APIs.
* Periodically clean up unused images and stopped containers.
* Provide detailed logs and error handling for all operations.
* Sanitize and validate all inputs to prevent injection or security risks.

---

### 6. `internal/logging/logger.go`, `audit.go`

**Prompt:**
Implement structured logging and audit trail functionality.

* Support multiple log levels: debug, info, warn, error.
* Write logs to local files with rotation and retention policies.
* Stream critical logs and audit records securely to backend API.
* Create tamper-resistant audit logs capturing all deployment and admin-related events.
* Ensure logs never contain sensitive information such as secrets or tokens.
* Provide utilities for consistent log formatting and timestamps.

---

### 7. `internal/monitoring/metrics.go`, `health_check.go`

**Prompt:**
Develop monitoring components for server and container health.

* Collect metrics: CPU usage, memory consumption, disk I/O, network stats at container and host level.
* Implement health checks for containers (e.g., responsiveness, uptime) and agent process.
* Periodically send collected metrics securely to backend APIs.
* Support alerting or retry logic on health degradation.
* Optionally expose Prometheus-compatible metrics endpoint for external monitoring.
* Log monitoring events and failures appropriately.

---

### 8. `internal/proxy/traefik_client.go`, `routing.go`

**Prompt:**
Integrate with Traefik reverse proxy for dynamic routing of deployed apps.

* Implement functionality to add, update, and remove Traefik routes dynamically via its API or configuration files.
* Assign subdomains to apps as directed by backend, supporting multiple domains and wildcards.
* Automate SSL certificate issuance and renewal through Traefik (Let's Encrypt).
* Ensure routing updates are transactional to avoid downtime.
* Handle rollback of routing changes on deployment failure.
* Log all routing changes with audit trail.

---

### 9. `internal/server/info.go`, `heartbeat.go`

**Prompt:**
Manage server metadata and heartbeat reporting.

* Collect static server info: hostname, server ID, physical or cloud location, hardware specs.
* Send heartbeat signals to backend at configurable intervals reporting availability and resource status.
* Support graceful shutdown notifications to backend.
* Monitor own agent health and report anomalies.
* Log heartbeat status and failures.

---

### 10. `internal/storage/store.go`, `encryption.go`

**Prompt:**
Provide secure local storage for sensitive agent data.

* Implement encrypted storage for API tokens, logs, and secrets using strong symmetric encryption.
* Provide thread-safe read/write APIs.
* Handle secure storage file permissions and access control.
* Include corruption detection and recovery mechanisms.
* Ensure minimal latency and robust error handling.
* Avoid exposing raw secrets in logs or API responses.

---

### 11. `internal/utils/helpers.go`

**Prompt:**
Create utility helper functions to support the agent.

* Implement retry wrappers with backoff for network and deployment commands.
* Provide safe shell command execution helpers with output capturing.
* File and directory manipulation utilities (create, delete, check existence).
* JSON/XML marshal/unmarshal helpers with error handling.
* Time formatting and parsing utilities.
* Any other reusable helpers needed by core modules.

---

### 12. `internal/watcher/watcher.go`

**Prompt:**
Build a watcher to detect and act on deployment or config changes.

* Poll backend API or listen to websocket/webhook for new deployment jobs.
* Watch local config files for changes and trigger reloads.
* Debounce rapid changes to avoid duplicate processing.
* Trigger deployment workflows on received commands.
* Handle error retries and notify backend on failure.
* Log watcher activity and failures.

---

### 13. `scripts/install.sh`

**Prompt:**
Write an installation script for the agent.

* Check for required dependencies (Docker, systemd, network connectivity).
* Download and install agent binary and configuration files.
* Set correct file permissions and ownership.
* Setup systemd service for auto-start and auto-restart on failure.
* Open necessary firewall ports if required.
* Optionally perform initial registration with backend API.
* Provide clear logs and error messages.

---