#!/bin/bash

set -euo pipefail

# SuperAgent Installation Script
# This script installs and configures the SuperAgent deployment system

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
AGENT_VERSION="${AGENT_VERSION:-1.0.0}"
AGENT_USER="${AGENT_USER:-superagent}"
AGENT_GROUP="${AGENT_GROUP:-superagent}"
INSTALL_DIR="${INSTALL_DIR:-/opt/superagent}"
CONFIG_DIR="${CONFIG_DIR:-/etc/superagent}"
LOG_DIR="${LOG_DIR:-/var/log/superagent}"
DATA_DIR="${DATA_DIR:-/var/lib/superagent}"
SYSTEMD_DIR="/etc/systemd/system"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Global flags
INTERACTIVE=true
FORCE_INSTALL=false
SKIP_DEPENDENCIES=false

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

info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

# Interactive confirmation
ask_confirmation() {
    local message="$1"
    local default="${2:-n}"
    
    if [[ "$INTERACTIVE" == "false" ]]; then
        return 0
    fi
    
    while true; do
        if [[ "$default" == "y" ]]; then
            read -p "$message [Y/n]: " yn
            yn=${yn:-y}
        else
            read -p "$message [y/N]: " yn
            yn=${yn:-n}
        fi
        
        case $yn in
            [Yy]* ) return 0;;
            [Nn]* ) return 1;;
            * ) echo "Please answer yes or no.";;
        esac
    done
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        error "This script must be run as root. Please use: sudo $0"
    fi
}

# Detect OS and package manager
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$ID
        OS_VERSION=$VERSION_ID
    else
        error "Cannot detect OS. /etc/os-release not found."
    fi
    
    # Detect package manager
    if command -v apt-get &> /dev/null; then
        PKG_MANAGER="apt"
        PKG_INSTALL="apt-get install -y"
        PKG_UPDATE="apt-get update"
    elif command -v yum &> /dev/null; then
        PKG_MANAGER="yum"
        PKG_INSTALL="yum install -y"
        PKG_UPDATE="yum update"
    elif command -v dnf &> /dev/null; then
        PKG_MANAGER="dnf"
        PKG_INSTALL="dnf install -y"
        PKG_UPDATE="dnf update"
    elif command -v zypper &> /dev/null; then
        PKG_MANAGER="zypper"
        PKG_INSTALL="zypper install -y"
        PKG_UPDATE="zypper refresh"
    else
        error "Unsupported package manager. Please install dependencies manually."
    fi
    
    info "Detected OS: $OS $OS_VERSION"
    info "Package manager: $PKG_MANAGER"
}

# Check and install package
check_and_install_package() {
    local package_name="$1"
    local command_name="${2:-$1}"
    local description="$3"
    
    if command -v "$command_name" &> /dev/null; then
        log "$description is already installed: $(which $command_name)"
        return 0
    fi
    
    warn "$description is not installed."
    
    if ask_confirmation "Do you want to install $description ($package_name)?"; then
        info "Installing $package_name..."
        
        # Update package list first
        $PKG_UPDATE || warn "Failed to update package list"
        
        if $PKG_INSTALL "$package_name"; then
            log "$description installed successfully"
            return 0
        else
            error "Failed to install $package_name"
        fi
    else
        error "$description is required for SuperAgent to function properly"
    fi
}

# Install Go if not present
install_go() {
    local go_version="1.21.5"
    local go_archive="go${go_version}.linux-amd64.tar.gz"
    
    if command -v go &> /dev/null; then
        local current_version=$(go version | awk '{print $3}' | sed 's/go//')
        log "Go is already installed: $current_version"
        return 0
    fi
    
    warn "Go is not installed."
    
    if ask_confirmation "Do you want to install Go $go_version?"; then
        info "Installing Go $go_version..."
        
        cd /tmp
        if ! wget "https://golang.org/dl/${go_archive}"; then
            error "Failed to download Go"
        fi
        
        tar -C /usr/local -xzf "$go_archive"
        
        # Add Go to PATH
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
        export PATH=$PATH:/usr/local/go/bin
        
        rm "$go_archive"
        log "Go installed successfully"
    else
        error "Go is required to build SuperAgent"
    fi
}

# Check system requirements
check_requirements() {
    log "Checking system requirements..."

    if [[ "$SKIP_DEPENDENCIES" == "true" ]]; then
        warn "Skipping dependency checks as requested"
        return 0
    fi

    # Check OS compatibility
    case "$OS" in
        ubuntu|debian)
            log "Supported OS detected: $OS"
            ;;
        centos|rhel|fedora|opensuse*)
            log "Supported OS detected: $OS"
            ;;
        *)
            warn "Untested OS: $OS. Installation may not work correctly."
            if ! ask_confirmation "Do you want to continue anyway?"; then
                exit 1
            fi
            ;;
    esac

    # Check systemd
    if ! command -v systemctl &> /dev/null; then
        error "systemd is required but not found"
    fi
    log "systemd is available"

    # Check and install Docker
    check_and_install_package "docker.io" "docker" "Docker"
    
    # Start Docker if not running
    if ! docker info &> /dev/null; then
        info "Starting Docker service..."
        systemctl start docker
        systemctl enable docker
    fi
    log "Docker is running"

    # Check and install Git
    check_and_install_package "git" "git" "Git"

    # Check and install other dependencies
    case "$PKG_MANAGER" in
        apt)
            check_and_install_package "curl" "curl" "curl"
            check_and_install_package "wget" "wget" "wget"
            check_and_install_package "openssl" "openssl" "OpenSSL"
            ;;
        yum|dnf)
            check_and_install_package "curl" "curl" "curl"
            check_and_install_package "wget" "wget" "wget"
            check_and_install_package "openssl" "openssl" "OpenSSL"
            ;;
        zypper)
            check_and_install_package "curl" "curl" "curl"
            check_and_install_package "wget" "wget" "wget"
            check_and_install_package "openssl" "openssl" "OpenSSL"
            ;;
    esac

    # Install Go
    install_go

    # Check available disk space (minimum 2GB)
    available_space=$(df / | awk 'NR==2 {print $4}')
    required_space=2097152  # 2GB in KB
    if [[ $available_space -lt $required_space ]]; then
        error "Insufficient disk space. At least 2GB is required, but only $(($available_space / 1024))MB available."
    fi
    log "Sufficient disk space available: $(($available_space / 1024))MB"

    log "System requirements check completed successfully"
}

# Create system user and group
create_user() {
    log "Creating system user and group..."

    if ! getent group "$AGENT_GROUP" &> /dev/null; then
        groupadd --system "$AGENT_GROUP"
        log "Created group: $AGENT_GROUP"
    else
        log "Group already exists: $AGENT_GROUP"
    fi

    if ! getent passwd "$AGENT_USER" &> /dev/null; then
        useradd --system --gid "$AGENT_GROUP" --home-dir "$DATA_DIR" \
                --shell /bin/false --create-home "$AGENT_USER"
        log "Created user: $AGENT_USER"
    else
        log "User already exists: $AGENT_USER"
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
    mkdir -p "/var/cache/superagent/git"

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
    log "Building SuperAgent..."

    cd "$SCRIPT_DIR"
    
    # Set Go environment
    export GOPATH="/tmp/go-build"
    export PATH=$PATH:/usr/local/go/bin
    
    # Clean any previous builds
    rm -f "$INSTALL_DIR/superagent"
    
    # Build the agent
    info "Downloading Go dependencies..."
    go mod download
    
    info "Compiling SuperAgent..."
    go build -o "$INSTALL_DIR/superagent" \
             -ldflags "-X main.version=$AGENT_VERSION -X main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
             ./cmd/agent

    # Set permissions
    chmod 755 "$INSTALL_DIR/superagent"
    chown root:root "$INSTALL_DIR/superagent"

    # Verify build
    if [[ -x "$INSTALL_DIR/superagent" ]]; then
        log "SuperAgent built successfully"
        "$INSTALL_DIR/superagent" version
    else
        error "Failed to build SuperAgent"
    fi
}

# Generate encryption key
generate_encryption_key() {
    log "Generating encryption key..."

    if [[ -f "$CONFIG_DIR/encryption.key" ]]; then
        warn "Encryption key already exists"
        if ! ask_confirmation "Do you want to regenerate the encryption key? (This will invalidate existing encrypted data)"; then
            log "Using existing encryption key"
            return 0
        fi
    fi

    openssl rand -base64 32 > "$CONFIG_DIR/encryption.key"
    chmod 600 "$CONFIG_DIR/encryption.key"
    chown root:root "$CONFIG_DIR/encryption.key"

    log "Encryption key generated"
}

# Create configuration file
create_config() {
    log "Creating configuration file..."

    if [[ -f "$CONFIG_DIR/config.yaml" ]]; then
        warn "Configuration file already exists"
        if ! ask_confirmation "Do you want to overwrite the existing configuration?"; then
            log "Using existing configuration"
            return 0
        fi
        cp "$CONFIG_DIR/config.yaml" "$CONFIG_DIR/config.yaml.backup.$(date +%Y%m%d-%H%M%S)"
        log "Backup created for existing configuration"
    fi

    cat > "$CONFIG_DIR/config.yaml" << 'EOF'
# SuperAgent Configuration

agent:
  id: ""                    # Auto-generated if empty
  location: "default"       # Server location identifier
  server_id: ""            # Auto-generated if empty
  work_dir: "/var/lib/superagent"
  data_dir: "/var/lib/superagent/data"
  temp_dir: "/tmp/superagent"
  pid_file: "/var/run/superagent.pid"
  user: "superagent"
  group: "superagent"
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
  network_name: "superagent"
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
  cache_dir: "/var/cache/superagent/git"
  cache_retention: "24h"

traefik:
  enabled: true
  provider: "file"
  config_file: "/etc/traefik/dynamic.yml"
  base_domain: "yourdomain.com"
  cert_resolver: "letsencrypt"
  enable_tls: true

security:
  encryption_key_file: "/etc/superagent/encryption.key"
  token_rotation_interval: "24h"
  audit_log_enabled: true
  audit_log_path: "/var/log/superagent/audit.log"
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
  log_file: "/var/log/superagent/agent.log"
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

    cat > "$SYSTEMD_DIR/superagent.service" << EOF
[Unit]
Description=SuperAgent - Enterprise Deployment Agent
Documentation=https://github.com/your-org/superagent
After=network.target docker.service
Wants=docker.service
Requires=docker.service

[Service]
Type=simple
User=$AGENT_USER
Group=$AGENT_GROUP
ExecStart=$INSTALL_DIR/superagent start --config $CONFIG_DIR/config.yaml
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
Environment="SUPERAGENT_CONFIG=$CONFIG_DIR/config.yaml"
Environment="SUPERAGENT_LOG_LEVEL=info"

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    log "Systemd service created"
}

# Create Docker network
create_docker_network() {
    log "Creating Docker network..."

    if ! docker network ls | grep -q superagent; then
        docker network create \
            --driver bridge \
            --subnet=172.20.0.0/16 \
            --gateway=172.20.0.1 \
            superagent
        log "Docker network 'superagent' created"
    else
        log "Docker network 'superagent' already exists"
    fi
}

# Setup firewall rules (if ufw is available)
setup_firewall() {
    if command -v ufw &> /dev/null; then
        log "Configuring firewall rules..."

        if ask_confirmation "Do you want to configure firewall rules with ufw?"; then
            # Allow SSH (be careful not to lock yourself out)
            ufw allow ssh

            # Allow HTTP and HTTPS
            ufw allow 80/tcp
            ufw allow 443/tcp

            # Allow agent monitoring ports
            ufw allow 8080/tcp  # Health check
            ufw allow 9090/tcp  # Metrics

            log "Firewall rules configured"
        else
            warn "Skipping firewall configuration"
        fi
    else
        warn "UFW not found, skipping firewall configuration"
    fi
}

# Create log rotation configuration
setup_log_rotation() {
    log "Setting up log rotation..."

    cat > "/etc/logrotate.d/superagent" << 'EOF'
/var/log/superagent/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 superagent superagent
    postrotate
        systemctl reload superagent 2>/dev/null || true
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
# Health check script for SuperAgent

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
# Status script for SuperAgent

set -euo pipefail

echo "=== SuperAgent Status ==="
echo "Service Status:"
systemctl status superagent --no-pager

echo ""
echo "Recent Logs:"
journalctl -u superagent --no-pager -n 10

echo ""
echo "Health Check:"
if /opt/superagent/scripts/health-check.sh; then
    echo "Agent is healthy"
else
    echo "Agent is unhealthy"
fi

echo ""
echo "Resource Usage:"
if command -v docker &> /dev/null; then
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" 2>/dev/null || echo "No containers running"
fi
EOF

    chmod +x "$INSTALL_DIR/scripts"/*.sh
    chown root:root "$INSTALL_DIR/scripts"/*.sh

    log "Monitoring scripts created"
}

# Copy uninstall script
copy_uninstall_script() {
    log "Copying uninstall script..."

    if [[ -f "$SCRIPT_DIR/uninstall.sh" ]]; then
        cp "$SCRIPT_DIR/uninstall.sh" "$INSTALL_DIR/uninstall.sh"
        chmod +x "$INSTALL_DIR/uninstall.sh"
        chown root:root "$INSTALL_DIR/uninstall.sh"
        log "Uninstall script copied to $INSTALL_DIR/uninstall.sh"
    else
        warn "Uninstall script not found at $SCRIPT_DIR/uninstall.sh"
    fi
}

# Main installation function
install_agent() {
    info "Starting SuperAgent installation..."
    info "This will install SuperAgent as a system service with enterprise security features."
    info ""
    
    if [[ "$INTERACTIVE" == "true" ]]; then
        if ! ask_confirmation "Do you want to continue with the installation?"; then
            info "Installation cancelled by user"
            exit 0
        fi
    fi

    detect_os
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
    copy_uninstall_script

    log "Installation completed successfully!"

    echo ""
    echo -e "${GREEN}=== Installation Summary ===${NC}"
    echo -e "SuperAgent installed to: ${BLUE}$INSTALL_DIR${NC}"
    echo -e "Configuration: ${BLUE}$CONFIG_DIR/config.yaml${NC}"
    echo -e "Logs: ${BLUE}$LOG_DIR${NC}"
    echo -e "Data: ${BLUE}$DATA_DIR${NC}"
    echo -e "User: ${BLUE}$AGENT_USER${NC}"
    echo -e "Service: ${BLUE}superagent.service${NC}"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo "1. Edit the configuration file: $CONFIG_DIR/config.yaml"
    echo "2. Set your backend URL and API token"
    echo "3. Configure SSH keys for Git access (if needed)"
    echo "4. Start the service: systemctl start superagent"
    echo "5. Enable auto-start: systemctl enable superagent"
    echo ""
    echo -e "${YELLOW}Useful commands:${NC}"
    echo "- Check status: systemctl status superagent"
    echo "- View logs: journalctl -u superagent -f"
    echo "- Health check: $INSTALL_DIR/scripts/health-check.sh"
    echo "- Full status: $INSTALL_DIR/scripts/status.sh"
    echo "- SuperAgent CLI: $INSTALL_DIR/superagent --help"
    echo ""
    echo -e "${CYAN}To uninstall SuperAgent, run: /opt/superagent/uninstall.sh${NC}"
    
    # Ask if user wants to start the service
    if ask_confirmation "Do you want to start SuperAgent now?" "y"; then
        systemctl enable superagent
        systemctl start superagent
        log "SuperAgent service started and enabled"
        
        # Show status
        sleep 2
        systemctl status superagent --no-pager || true
    fi
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --non-interactive|-n)
                INTERACTIVE=false
                shift
                ;;
            --force|-f)
                FORCE_INSTALL=true
                shift
                ;;
            --skip-deps)
                SKIP_DEPENDENCIES=true
                shift
                ;;
            --help|-h)
                echo "SuperAgent Installation Script"
                echo ""
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --non-interactive, -n    Run in non-interactive mode"
                echo "  --force, -f              Force installation even if already installed"
                echo "  --skip-deps              Skip dependency installation"
                echo "  --help, -h               Show this help message"
                echo ""
                echo "Environment Variables:"
                echo "  AGENT_VERSION            SuperAgent version (default: 1.0.0)"
                echo "  AGENT_USER               System user for SuperAgent (default: superagent)"
                echo "  INSTALL_DIR              Installation directory (default: /opt/superagent)"
                echo "  CONFIG_DIR               Configuration directory (default: /etc/superagent)"
                echo "  LOG_DIR                  Log directory (default: /var/log/superagent)"
                echo "  DATA_DIR                 Data directory (default: /var/lib/superagent)"
                echo ""
                exit 0
                ;;
            *)
                error "Unknown option: $1. Use --help for usage information."
                ;;
        esac
    done
}

# Main execution
main() {
    parse_args "$@"
    
    # Check if already installed
    if [[ -f "$INSTALL_DIR/superagent" && "$FORCE_INSTALL" == "false" ]]; then
        warn "SuperAgent appears to be already installed at $INSTALL_DIR"
        if ask_confirmation "Do you want to reinstall?"; then
            FORCE_INSTALL=true
        else
            info "Installation cancelled"
            exit 0
        fi
    fi
    
    install_agent
}

# Run main function
main "$@"