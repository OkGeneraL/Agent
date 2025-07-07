#!/bin/bash

set -euo pipefail

# Deployment Agent Installation Script
# This script installs and configures the secure deployment agent

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_VERSION="${AGENT_VERSION:-1.0.0}"
AGENT_USER="${AGENT_USER:-deployment-agent}"
AGENT_GROUP="${AGENT_GROUP:-deployment-agent}"
INSTALL_DIR="${INSTALL_DIR:-/opt/deployment-agent}"
CONFIG_DIR="${CONFIG_DIR:-/etc/deployment-agent}"
LOG_DIR="${LOG_DIR:-/var/log/deployment-agent}"
DATA_DIR="${DATA_DIR:-/var/lib/deployment-agent}"
SYSTEMD_DIR="/etc/systemd/system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

debug() {
    if [[ "${DEBUG:-}" == "1" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1"
    fi
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root"
    fi
}

# Check system requirements
check_requirements() {
    log "Checking system requirements..."

    # Check OS
    if ! command -v systemctl &> /dev/null; then
        error "systemd is required but not found"
    fi

    # Check Docker
    if ! command -v docker &> /dev/null; then
        error "Docker is required but not found. Please install Docker first."
    fi

    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        error "Docker daemon is not running. Please start Docker first."
    fi

    # Check Go (for building)
    if ! command -v go &> /dev/null; then
        warn "Go is not installed. Will attempt to install Go..."
        install_go
    fi

    # Check Git
    if ! command -v git &> /dev/null; then
        error "Git is required but not found. Please install Git first."
    fi

    # Check available disk space (minimum 1GB)
    available_space=$(df / | awk 'NR==2 {print $4}')
    if [[ $available_space -lt 1048576 ]]; then
        error "Insufficient disk space. At least 1GB is required."
    fi

    log "System requirements check passed"
}

# Install Go if not present
install_go() {
    log "Installing Go..."
    
    GO_VERSION="1.21.5"
    GO_ARCHIVE="go${GO_VERSION}.linux-amd64.tar.gz"
    
    cd /tmp
    wget "https://golang.org/dl/${GO_ARCHIVE}"
    tar -C /usr/local -xzf "$GO_ARCHIVE"
    
    # Add Go to PATH
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    export PATH=$PATH:/usr/local/go/bin
    
    rm "$GO_ARCHIVE"
    log "Go installed successfully"
}

# Create system user and group
create_user() {
    log "Creating system user and group..."

    if ! getent group "$AGENT_GROUP" &> /dev/null; then
        groupadd --system "$AGENT_GROUP"
        log "Created group: $AGENT_GROUP"
    fi

    if ! getent passwd "$AGENT_USER" &> /dev/null; then
        useradd --system --gid "$AGENT_GROUP" --home-dir "$DATA_DIR" \
                --shell /bin/false --create-home "$AGENT_USER"
        log "Created user: $AGENT_USER"
    fi

    # Add user to docker group
    usermod -aG docker "$AGENT_USER"
    log "Added $AGENT_USER to docker group"
}

# Create directories
create_directories() {
    log "Creating directories..."

    mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
    mkdir -p "$DATA_DIR/data" "$DATA_DIR/cache" "$DATA_DIR/git"
    mkdir -p "/var/cache/deployment-agent/git"

    # Set ownership and permissions
    chown -R "$AGENT_USER:$AGENT_GROUP" "$DATA_DIR" "$LOG_DIR"
    chown root:root "$CONFIG_DIR"
    chmod 755 "$INSTALL_DIR" "$CONFIG_DIR"
    chmod 750 "$DATA_DIR" "$LOG_DIR"
    chmod 700 "$CONFIG_DIR"

    log "Directories created and configured"
}

# Build the agent
build_agent() {
    log "Building deployment agent..."

    cd "$SCRIPT_DIR"
    
    # Set Go environment
    export GOPATH="/tmp/go-build"
    export PATH=$PATH:/usr/local/go/bin
    
    # Build the agent
    go mod download
    go build -o "$INSTALL_DIR/deployment-agent" \
             -ldflags "-X main.version=$AGENT_VERSION -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
             ./cmd/agent

    # Set permissions
    chmod 755 "$INSTALL_DIR/deployment-agent"
    chown root:root "$INSTALL_DIR/deployment-agent"

    log "Agent built successfully"
}

# Generate encryption key
generate_encryption_key() {
    log "Generating encryption key..."

    openssl rand -base64 32 > "$CONFIG_DIR/encryption.key"
    chmod 600 "$CONFIG_DIR/encryption.key"
    chown root:root "$CONFIG_DIR/encryption.key"

    log "Encryption key generated"
}

# Create configuration file
create_config() {
    log "Creating configuration file..."

    cat > "$CONFIG_DIR/config.yaml" << 'EOF'
# Deployment Agent Configuration

agent:
  id: ""                    # Auto-generated if empty
  location: "default"       # Server location identifier
  server_id: ""            # Auto-generated if empty
  work_dir: "/var/lib/deployment-agent"
  data_dir: "/var/lib/deployment-agent/data"
  temp_dir: "/tmp/deployment-agent"
  pid_file: "/var/run/deployment-agent.pid"
  user: "deployment-agent"
  group: "deployment-agent"
  max_concurrent_ops: 5
  heartbeat_interval: "30s"

backend:
  base_url: "https://your-backend.com/api"
  api_token: ""            # Required - set via environment variable
  token_file: ""           # Optional - path to token file
  refresh_interval: "30s"
  timeout: "30s"
  retry_attempts: 3
  retry_delay: "5s"
  insecure_skip_tls: false
  webhook_endpoint: "/webhook"
  webhook_secret: ""       # Required for webhook validation

docker:
  host: "unix:///var/run/docker.sock"
  version: "1.41"
  network_name: "deployment-agent"
  log_driver: "json-file"
  cleanup_interval: "1h"
  cleanup_retention: "24h"
  default_cpu_limit: "1"
  default_memory_limit: "1G"
  default_storage_limit: "10G"

git:
  ssh_key_path: ""         # Path to SSH private key
  timeout: "30s"
  max_depth: 50
  cache_dir: "/var/cache/deployment-agent/git"
  cache_retention: "24h"

traefik:
  enabled: true
  provider: "file"
  config_file: "/etc/traefik/dynamic.yml"
  base_domain: "yourdomain.com"
  cert_resolver: "letsencrypt"
  enable_tls: true

security:
  encryption_key_file: "/etc/deployment-agent/encryption.key"
  token_rotation_interval: "24h"
  audit_log_enabled: true
  audit_log_path: "/var/log/deployment-agent/audit.log"
  run_as_non_root: true
  read_only_root_fs: true
  no_new_privileges: true

monitoring:
  enabled: true
  metrics_port: 9090
  metrics_path: "/metrics"
  health_check_port: 8080
  health_check_path: "/health"
  prometheus_enabled: true
  metrics_interval: "15s"

logging:
  level: "info"
  format: "json"
  output: "file"
  log_file: "/var/log/deployment-agent/agent.log"
  max_size: 100
  max_backups: 10
  max_age: 30
  compress: true

resources:
  cpu_quota: "80%"
  memory_quota: "80%"
  storage_quota: "80%"
  network_quota: "1Gbps"
  max_containers: 50
  max_volumes: 100
  max_networks: 10
  reserved_cpu: "0.5"
  reserved_memory: "1G"
  reserved_storage: "10G"

networking:
  allowed_ports: [80, 443, 8080, 8443]
  blocked_ports: [22, 23, 135, 139, 445]
  dns_servers: ["8.8.8.8", "8.8.4.4"]
  firewall_enabled: true
EOF

    chmod 600 "$CONFIG_DIR/config.yaml"
    chown root:root "$CONFIG_DIR/config.yaml"

    log "Configuration file created"
}

# Create systemd service
create_systemd_service() {
    log "Creating systemd service..."

    cat > "$SYSTEMD_DIR/deployment-agent.service" << EOF
[Unit]
Description=Deployment Agent - Secure PaaS deployment agent
Documentation=https://github.com/your-org/deployment-agent
After=network.target docker.service
Wants=docker.service
Requires=docker.service

[Service]
Type=simple
User=$AGENT_USER
Group=$AGENT_GROUP
ExecStart=$INSTALL_DIR/deployment-agent start --config $CONFIG_DIR/config.yaml
ExecReload=/bin/kill -HUP \$MAINPID
KillMode=mixed
KillSignal=SIGTERM
TimeoutStopSec=30
Restart=always
RestartSec=5
StartLimitInterval=60
StartLimitBurst=3

# Security settings
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=$DATA_DIR $LOG_DIR /tmp
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes
RestrictRealtime=yes
RestrictSUIDSGID=yes
LockPersonality=yes
MemoryDenyWriteExecute=yes
RestrictNamespaces=yes
SystemCallFilter=@system-service
SystemCallErrorNumber=EPERM

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Environment
Environment="DEPLOYMENT_AGENT_CONFIG=$CONFIG_DIR/config.yaml"
Environment="DEPLOYMENT_AGENT_LOG_LEVEL=info"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log "Systemd service created"
}

# Create Docker network
create_docker_network() {
    log "Creating Docker network..."

    if ! docker network ls | grep -q deployment-agent; then
        docker network create \
            --driver bridge \
            --subnet=172.20.0.0/16 \
            --gateway=172.20.0.1 \
            deployment-agent
        log "Docker network 'deployment-agent' created"
    else
        log "Docker network 'deployment-agent' already exists"
    fi
}

# Setup firewall rules (if ufw is available)
setup_firewall() {
    if command -v ufw &> /dev/null; then
        log "Configuring firewall rules..."

        # Allow SSH (be careful not to lock yourself out)
        ufw allow ssh

        # Allow HTTP and HTTPS
        ufw allow 80/tcp
        ufw allow 443/tcp

        # Allow agent monitoring ports
        ufw allow 8080/tcp  # Health check
        ufw allow 9090/tcp  # Metrics

        # Block dangerous ports
        ufw deny 22/tcp from any to any port 22  # SSH (adjust as needed)
        ufw deny 23/tcp    # Telnet
        ufw deny 135/tcp   # RPC
        ufw deny 139/tcp   # NetBIOS
        ufw deny 445/tcp   # SMB

        log "Firewall rules configured"
    else
        warn "UFW not found, skipping firewall configuration"
    fi
}

# Create log rotation configuration
setup_log_rotation() {
    log "Setting up log rotation..."

    cat > "/etc/logrotate.d/deployment-agent" << 'EOF'
/var/log/deployment-agent/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 deployment-agent deployment-agent
    postrotate
        systemctl reload deployment-agent 2>/dev/null || true
    endscript
}
EOF

    log "Log rotation configured"
}

# Create monitoring scripts
create_monitoring_scripts() {
    log "Creating monitoring scripts..."

    mkdir -p "$INSTALL_DIR/scripts"

    # Health check script
    cat > "$INSTALL_DIR/scripts/health-check.sh" << 'EOF'
#!/bin/bash
# Health check script for deployment agent

set -euo pipefail

HEALTH_URL="http://localhost:8080/health"
TIMEOUT=10

if curl -sf --max-time "$TIMEOUT" "$HEALTH_URL" > /dev/null; then
    echo "healthy"
    exit 0
else
    echo "unhealthy"
    exit 1
fi
EOF

    # Status script
    cat > "$INSTALL_DIR/scripts/status.sh" << 'EOF'
#!/bin/bash
# Status script for deployment agent

set -euo pipefail

echo "=== Deployment Agent Status ==="
echo "Service Status:"
systemctl status deployment-agent --no-pager

echo ""
echo "Recent Logs:"
journalctl -u deployment-agent --no-pager -n 10

echo ""
echo "Health Check:"
if /opt/deployment-agent/scripts/health-check.sh; then
    echo "Agent is healthy"
else
    echo "Agent is unhealthy"
fi

echo ""
echo "Resource Usage:"
docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}"
EOF

    chmod +x "$INSTALL_DIR/scripts"/*.sh
    chown root:root "$INSTALL_DIR/scripts"/*.sh

    log "Monitoring scripts created"
}

# Cleanup function
cleanup() {
    log "Cleaning up temporary files..."
    rm -rf /tmp/go-build
}

# Main installation function
install_agent() {
    log "Starting deployment agent installation..."

    check_root
    check_requirements
    create_user
    create_directories
    build_agent
    generate_encryption_key
    create_config
    create_systemd_service
    create_docker_network
    setup_firewall
    setup_log_rotation
    create_monitoring_scripts

    log "Installation completed successfully!"

    echo ""
    echo -e "${GREEN}=== Installation Summary ===${NC}"
    echo -e "Agent installed to: ${BLUE}$INSTALL_DIR${NC}"
    echo -e "Configuration: ${BLUE}$CONFIG_DIR/config.yaml${NC}"
    echo -e "Logs: ${BLUE}$LOG_DIR${NC}"
    echo -e "Data: ${BLUE}$DATA_DIR${NC}"
    echo -e "User: ${BLUE}$AGENT_USER${NC}"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "1. Edit the configuration file: $CONFIG_DIR/config.yaml"
    echo "2. Set your backend URL and API token"
    echo "3. Configure SSH keys for Git access (if needed)"
    echo "4. Start the service: systemctl start deployment-agent"
    echo "5. Enable auto-start: systemctl enable deployment-agent"
    echo ""
    echo -e "${YELLOW}Useful commands:${NC}"
    echo "- Check status: systemctl status deployment-agent"
    echo "- View logs: journalctl -u deployment-agent -f"
    echo "- Health check: $INSTALL_DIR/scripts/health-check.sh"
    echo "- Full status: $INSTALL_DIR/scripts/status.sh"
}

# Uninstall function
uninstall_agent() {
    log "Uninstalling deployment agent..."

    # Stop and disable service
    if systemctl is-active deployment-agent &> /dev/null; then
        systemctl stop deployment-agent
    fi
    if systemctl is-enabled deployment-agent &> /dev/null; then
        systemctl disable deployment-agent
    fi

    # Remove service file
    rm -f "$SYSTEMD_DIR/deployment-agent.service"
    systemctl daemon-reload

    # Remove Docker network
    if docker network ls | grep -q deployment-agent; then
        docker network rm deployment-agent || true
    fi

    # Remove files and directories
    rm -rf "$INSTALL_DIR"
    rm -rf "$CONFIG_DIR"
    rm -rf "$LOG_DIR"
    rm -rf "$DATA_DIR"
    rm -f "/etc/logrotate.d/deployment-agent"

    # Remove user and group
    if getent passwd "$AGENT_USER" &> /dev/null; then
        userdel "$AGENT_USER"
    fi
    if getent group "$AGENT_GROUP" &> /dev/null; then
        groupdel "$AGENT_GROUP"
    fi

    log "Uninstallation completed"
}

# Parse command line arguments
case "${1:-install}" in
    install)
        install_agent
        ;;
    uninstall)
        uninstall_agent
        ;;
    cleanup)
        cleanup
        ;;
    *)
        echo "Usage: $0 {install|uninstall|cleanup}"
        echo ""
        echo "Commands:"
        echo "  install   - Install the deployment agent"
        echo "  uninstall - Remove the deployment agent"
        echo "  cleanup   - Clean up temporary files"
        exit 1
        ;;
esac

# Set trap for cleanup
trap cleanup EXIT