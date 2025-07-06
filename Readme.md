

---

# Cursor AI Prompt: Build a Secure, Enterprise-Grade Go Deployment Agent for PaaS

**Project Overview:**
Build a robust Go-based agent to run on multiple servers as part of a PaaS platform. The agent’s job is to securely manage app deployments (Next.js, Docker containers) from GitHub private repos or Docker images, handle lifecycle operations, enforce resource limits, manage routing with Traefik, report monitoring data, and support multi-server failover. The agent must be enterprise-grade, highly secure, and production-ready.

---

## 1. Authentication & Security

* Use **secure, unique API tokens** for each agent instance to authenticate with the backend API. Tokens must be revocable and rotatable.
* All communications between agent and backend must use **HTTPS with TLS encryption**.
* Implement **token refresh and revocation** mechanism.
* Optionally support **mutual TLS (mTLS)** for stronger security.
* Store all sensitive data (tokens, secrets) encrypted on disk.
* Run agent process with **least privileges**; avoid running as root.
* Use Docker security best practices (user namespaces, seccomp, etc.).
* Implement detailed **audit logs** for all agent actions.

---

## 2. Backend Connectivity & API

* Agent must **register itself** with the backend on startup using API token.
* Periodically poll or receive push notifications from backend for deployment instructions.
* Support webhook or long-polling mechanism to receive commands efficiently.
* Provide API endpoints for:

  * Status reporting (health, resource usage)
  * Log streaming
  * Deployment lifecycle event reporting

---

## 3. Deployment Support

### Sources

* Deploy from **GitHub private repos**:

  * Clone repos securely using SSH keys or OAuth tokens.
  * Support checking out specific commits, branches, or tags.
* Deploy from **Docker images**:

  * Pull images from private or public registries.
  * Support tagging and version pinning.

### Build & Run

* For repo-based apps:

  * Run build commands (`npm install`, `next build`, etc.) as defined by backend config.
* Run apps inside Docker containers with specified environment variables.
* Assign each deployment a unique container name/ID.
* Support mounting volumes if required.

---

## 4. App Lifecycle Operations

* Support **start**, **stop**, **restart**, **update**, **delete** operations on containers.
* Support **zero-downtime updates** (spin up new container, switch traffic via Traefik).
* Support **rollback** to specific versions (git commit/tag).
* Clean up unused containers and images regularly.

---

## 5. Resource Management

* Enforce **CPU**, **memory**, **disk space**, and **network bandwidth** limits per container.
* Monitor resource usage continuously.
* Reject deployments that exceed server or user quota.
* Support live adjustment of resource limits on running containers.
* Report usage metrics back to backend in real-time.

---

## 6. Reverse Proxy & Routing

* Integrate with **Traefik**:

  * Automatically update Traefik config or use Traefik API to route deployed apps.
  * Assign and manage subdomains dynamically as instructed by backend.
  * Support SSL certificate issuance and renewal via Traefik.
* Support wildcard DNS usage.
* Handle routing updates during zero-downtime deployments.

---

## 7. Multi-Server & Multi-Location Support

* Agent must identify its **location and server ID** on startup.
* Support receiving deployment jobs targeting this specific agent/server.
* Support **failover deployments**:

  * Detect server or agent offline.
  * Allow backend to redeploy apps to alternate servers.
* Report health and availability status continuously.

---

## 8. Monitoring & Logging

* Collect container and server-level metrics (CPU, memory, disk, network).
* Collect and stream container logs (stdout/stderr) to backend.
* Provide health checks for running apps.
* Support configurable log retention and rotation.
* Push metrics and logs securely to backend APIs.

---

## 9. Configuration & Extensibility

* Configurable via secure config files or environment variables.
* Support dynamic configuration reload without downtime.
* Modular codebase to allow adding new deployment types or integrations.

---

## 10. Operational Considerations

* Provide a clean CLI interface for installation, startup, status, and logs.
* Support automatic updates or admin-triggered upgrades.
* Detailed error handling and retries for network failures or build issues.
* Documentation on setup, config, and operation.

---

## Deliverables

* Fully working Go agent source code, ready for production use.
* Unit and integration tests covering critical components.
* Deployment scripts or Dockerfile to run agent easily.
* Documentation for installation, configuration, API usage, and security best practices.

---

**Additional notes:**
The agent will be paired with a Next.js-based admin and user dashboard backend that handles license management, deployment policies, user access, billing, and version control. The agent must strictly follow backend instructions and report detailed status.
