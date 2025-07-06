# Deployment Agent

A secure, enterprise-grade Go-based deployment agent for PaaS platforms. This agent runs on multiple servers and manages containerized application deployments from GitHub repositories or Docker images with advanced security, monitoring, and resource management capabilities.

## Features

### 🔒 **Security First**
- **Secure API Authentication**: Unique, revocable API tokens with automatic rotation
- **TLS/mTLS Support**: End-to-end encryption with mutual TLS option
- **Encrypted Storage**: All sensitive data encrypted at rest using AES-256
- **Audit Logging**: Comprehensive audit trails for all operations
- **Container Security**: SELinux/AppArmor, seccomp profiles, and no-root execution
- **Network Security**: Firewall rules and network policies

### 🚀 **Deployment Capabilities**
- **Multi-Source Deployments**: Deploy from GitHub repositories or Docker images
- **Zero-Downtime Updates**: Blue-green deployments with health checks
- **Resource Management**: CPU, memory, storage, and network quotas
- **Auto-Scaling**: Container scaling based on resource usage
- **Rollback Support**: Quick rollback to previous versions

### 📊 **Monitoring & Observability**
- **Real-time Metrics**: Prometheus-compatible metrics endpoint
- **Health Checks**: Application and system health monitoring
- **Log Streaming**: Centralized log aggregation
- **Resource Monitoring**: CPU, memory, disk, and network usage tracking
- **Alerting**: Configurable alerts for resource thresholds

### 🌐 **High Availability**
- **Multi-Server Support**: Deploy across multiple servers/locations
- **Failover Capabilities**: Automatic failover to alternate servers
- **Load Balancing**: Traefik integration for traffic management
- **Horizontal Scaling**: Scale across multiple agent instances

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    PaaS Backend                              │
│              (Next.js Dashboard)                             │
└─────────────────────┬───────────────────────────────────────┘
                      │ HTTPS/WebSocket
                      │ (Secure API)
┌─────────────────────▼───────────────────────────────────────┐
│                Deployment Agent                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ │
│  │   Backend   │ │    Docker   │ │        Git              │ │
│  │   Client    │ │   Manager   │ │      Manager            │ │
│  └─────────────┘ └─────────────┘ └─────────────────────────┘ │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────────┐ │
│  │  Monitoring │ │   Traefik   │ │      Resource           │ │
│  │   System    │ │Integration  │ │      Manager            │ │
│  └─────────────┘ └─────────────┘ └─────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────┐
│                  Docker Engine                              │
│  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ │
│  │   Container 1   │ │   Container 2   │ │   Container N   │ │
│  │   (Your App)    │ │   (Your App)    │ │   (Your App)    │ │
│  └─────────────────┘ └─────────────────┘ └─────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- **Linux Server** with systemd support
- **Docker** (version 20.10+)
- **Git** for repository operations
- **Root access** for installation
- **Minimum 1GB** available disk space
- **Network connectivity** to your backend

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/your-org/deployment-agent.git
   cd deployment-agent
   ```

2. **Run the installation script**:
   ```bash
   sudo ./install.sh
   ```

3. **Configure the agent**:
   ```bash
   sudo nano /etc/deployment-agent/config.yaml
   ```

   Update the following required settings:
   ```yaml
   backend:
     base_url: "https://your-backend.com/api"
     api_token: "your-secure-api-token"
   
   traefik:
     base_domain: "yourdomain.com"
   ```

4. **Start the agent**:
   ```bash
   sudo systemctl start deployment-agent
   sudo systemctl enable deployment-agent
   ```

5. **Verify installation**:
   ```bash
   sudo systemctl status deployment-agent
   /opt/deployment-agent/scripts/health-check.sh
   ```

## Configuration

The agent is configured via `/etc/deployment-agent/config.yaml`. Key sections include:

### Backend Configuration
```yaml
backend:
  base_url: "https://your-backend.com/api"
  api_token: "your-secure-token"
  timeout: "30s"
  retry_attempts: 3
```

### Security Configuration
```yaml
security:
  encryption_key_file: "/etc/deployment-agent/encryption.key"
  audit_log_enabled: true
  run_as_non_root: true
  token_rotation_interval: "24h"
```

### Resource Limits
```yaml
resources:
  cpu_quota: "80%"
  memory_quota: "80%"
  max_containers: 50
  reserved_cpu: "0.5"
  reserved_memory: "1G"
```

### Docker Configuration
```yaml
docker:
  host: "unix:///var/run/docker.sock"
  network_name: "deployment-agent"
  cleanup_interval: "1h"
  default_cpu_limit: "1"
  default_memory_limit: "1G"
```

## Usage

### Basic Commands

```bash
# Check agent status
sudo systemctl status deployment-agent

# View real-time logs
sudo journalctl -u deployment-agent -f

# Health check
/opt/deployment-agent/scripts/health-check.sh

# Complete status report
/opt/deployment-agent/scripts/status.sh

# Restart agent
sudo systemctl restart deployment-agent
```

### CLI Interface

The agent provides a comprehensive CLI:

```bash
# Start the agent
deployment-agent start

# Check version
deployment-agent version

# Validate configuration
deployment-agent config validate

# Install as system service
deployment-agent install

# Show help
deployment-agent --help
```

## API Integration

The agent communicates with your backend via secure HTTPS and WebSocket connections:

### Registration
When started, the agent automatically registers with the backend:
```json
{
  "id": "agent-server01-1234567890",
  "server_id": "server-server01",
  "location": "us-east-1",
  "capabilities": ["docker", "git", "traefik", "monitoring"],
  "status": "online"
}
```

### Command Processing
The agent polls for and processes deployment commands:
```json
{
  "id": "cmd-123",
  "type": "deployment",
  "action": "deploy",
  "spec": {
    "name": "my-app",
    "image": "nginx:latest",
    "ports": [{"host_port": 8080, "container_port": 80}]
  }
}
```

### Status Reporting
Regular status reports include:
- Container health and resource usage
- System resource availability
- Active deployments
- Agent health status

## Security Features

### Authentication & Authorization
- **API Token Authentication**: Secure token-based authentication
- **Token Rotation**: Automatic token refresh every 24 hours
- **mTLS Support**: Optional mutual TLS for enhanced security

### Container Security
- **Non-root Execution**: All containers run as non-root users
- **Read-only Root Filesystem**: Prevents runtime modifications
- **Security Profiles**: SELinux/AppArmor and seccomp profiles
- **Network Policies**: Restrict container network access

### Data Protection
- **Encryption at Rest**: All sensitive data encrypted using AES-256
- **Secure Key Management**: Hardware security module support
- **Audit Logging**: Comprehensive audit trails for compliance

## Monitoring

### Metrics Endpoint
Access Prometheus metrics at `http://localhost:9090/metrics`:
- Container resource usage
- Deployment success/failure rates
- Agent performance metrics
- System resource utilization

### Health Checks
Health endpoint at `http://localhost:8080/health` provides:
- Agent status
- Docker daemon connectivity
- Backend connectivity
- Resource availability

### Log Management
- **Structured Logging**: JSON-formatted logs for easy parsing
- **Log Rotation**: Automatic log rotation and compression
- **Centralized Logging**: Stream logs to centralized systems
- **Audit Trails**: Separate audit logs for security events

## Troubleshooting

### Common Issues

1. **Agent won't start**:
   ```bash
   # Check service status
   sudo systemctl status deployment-agent
   
   # Check logs
   sudo journalctl -u deployment-agent --no-pager
   
   # Validate configuration
   deployment-agent config validate
   ```

2. **Docker connection issues**:
   ```bash
   # Verify Docker is running
   sudo systemctl status docker
   
   # Check Docker socket permissions
   ls -la /var/run/docker.sock
   
   # Test Docker connectivity
   sudo -u deployment-agent docker ps
   ```

3. **Backend connectivity issues**:
   ```bash
   # Test API connectivity
   curl -H "Authorization: Bearer YOUR_TOKEN" \
        https://your-backend.com/api/health
   
   # Check network connectivity
   ping your-backend.com
   
   # Verify TLS certificate
   openssl s_client -connect your-backend.com:443
   ```

### Log Analysis

Key log patterns to monitor:
```bash
# Successful deployments
sudo journalctl -u deployment-agent | grep "DEPLOY_CONTAINER_SUCCESS"

# Failed operations
sudo journalctl -u deployment-agent | grep "ERROR"

# Security events
sudo tail -f /var/log/deployment-agent/audit.log
```

## Development

### Building from Source

```bash
# Install dependencies
go mod download

# Build the agent
go build -o deployment-agent ./cmd/agent

# Run tests
go test ./...

# Build with optimizations
go build -ldflags "-s -w" -o deployment-agent ./cmd/agent
```

### Contributing

1. Fork the repository
2. Create a feature branch
3. Implement your changes with tests
4. Submit a pull request

### Project Structure

```
deployment-agent/
├── cmd/agent/              # Main application entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── api/               # Backend API client
│   ├── docker/            # Docker operations
│   ├── git/               # Git operations
│   ├── logging/           # Logging and audit
│   └── agent/             # Main agent orchestrator
├── install.sh             # Installation script
├── go.mod                 # Go module definition
└── README.md              # This file
```

## Support

- **Documentation**: Check this README and inline code comments
- **Issues**: Report bugs via GitHub Issues
- **Security**: Report security vulnerabilities privately
- **Community**: Join our discussion forums

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Docker for containerization platform
- Traefik for reverse proxy capabilities
- Prometheus for monitoring infrastructure
- Go community for excellent libraries

---

**Built with ❤️ for secure, scalable deployments**