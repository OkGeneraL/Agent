#!/bin/bash

set -euo pipefail

# SuperAgent Uninstallation Script
# This script removes SuperAgent and optionally removes installed dependencies

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
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
FORCE_REMOVE=false
KEEP_DATA=false
REMOVE_PACKAGES=false

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

info() {
    echo -e "${CYAN}[INFO]${NC} $1"
}

# Interactive confirmation
ask_confirmation() {
    local message="$1"
    local default="${2:-n}"
    
    if [[ "$INTERACTIVE" == "false" ]]; then
        if [[ "$default" == "y" ]]; then
            return 0
        else
            return 1
        fi
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
        PKG_REMOVE="apt-get remove -y"
        PKG_PURGE="apt-get purge -y"
        PKG_AUTOREMOVE="apt-get autoremove -y"
    elif command -v yum &> /dev/null; then
        PKG_MANAGER="yum"
        PKG_REMOVE="yum remove -y"
        PKG_PURGE="yum remove -y"
        PKG_AUTOREMOVE="yum autoremove -y"
    elif command -v dnf &> /dev/null; then
        PKG_MANAGER="dnf"
        PKG_REMOVE="dnf remove -y"
        PKG_PURGE="dnf remove -y"
        PKG_AUTOREMOVE="dnf autoremove -y"
    elif command -v zypper &> /dev/null; then
        PKG_MANAGER="zypper"
        PKG_REMOVE="zypper remove -y"
        PKG_PURGE="zypper remove -y"
        PKG_AUTOREMOVE="zypper remove -y"
    else
        warn "Package manager not detected. Manual package removal may be required."
        PKG_MANAGER="unknown"
    fi
    
    info "Detected OS: $OS $OS_VERSION"
    info "Package manager: $PKG_MANAGER"
}

# Stop and remove SuperAgent service
stop_and_remove_service() {
    log "Stopping and removing SuperAgent service..."

    # Stop service if running
    if systemctl is-active superagent &> /dev/null; then
        info "Stopping SuperAgent service..."
        systemctl stop superagent
        log "SuperAgent service stopped"
    else
        log "SuperAgent service is not running"
    fi

    # Disable service if enabled
    if systemctl is-enabled superagent &> /dev/null; then
        info "Disabling SuperAgent service..."
        systemctl disable superagent
        log "SuperAgent service disabled"
    else
        log "SuperAgent service is not enabled"
    fi

    # Remove service file
    if [[ -f "$SYSTEMD_DIR/superagent.service" ]]; then
        rm -f "$SYSTEMD_DIR/superagent.service"
        systemctl daemon-reload
        log "SuperAgent systemd service file removed"
    else
        log "SuperAgent systemd service file not found"
    fi
}

# Remove Docker network
remove_docker_network() {
    log "Checking Docker network..."

    if command -v docker &> /dev/null && docker network ls | grep -q superagent; then
        if ask_confirmation "Do you want to remove the SuperAgent Docker network?"; then
            docker network rm superagent || warn "Failed to remove Docker network"
            log "SuperAgent Docker network removed"
        else
            warn "Keeping SuperAgent Docker network"
        fi
    else
        log "SuperAgent Docker network not found or Docker not available"
    fi
}

# Remove SuperAgent containers
remove_containers() {
    log "Checking for SuperAgent containers..."

    if command -v docker &> /dev/null; then
        # Get SuperAgent containers
        containers=$(docker ps -a --filter "label=managed-by=superagent" --format "{{.ID}} {{.Names}}" 2>/dev/null || echo "")
        
        if [[ -n "$containers" ]]; then
            echo "Found SuperAgent containers:"
            echo "$containers"
            echo ""
            
            if ask_confirmation "Do you want to remove all SuperAgent containers?"; then
                # Stop and remove containers
                docker ps -a --filter "label=managed-by=superagent" -q | xargs -r docker stop
                docker ps -a --filter "label=managed-by=superagent" -q | xargs -r docker rm -f
                log "SuperAgent containers removed"
            else
                warn "Keeping SuperAgent containers (they may not work without SuperAgent)"
            fi
        else
            log "No SuperAgent containers found"
        fi

        # Check for SuperAgent images
        images=$(docker images --filter "reference=superagent/*" --format "{{.Repository}}:{{.Tag}}" 2>/dev/null || echo "")
        
        if [[ -n "$images" ]]; then
            echo "Found SuperAgent images:"
            echo "$images"
            echo ""
            
            if ask_confirmation "Do you want to remove SuperAgent Docker images?"; then
                docker images --filter "reference=superagent/*" -q | xargs -r docker rmi -f
                log "SuperAgent images removed"
            else
                warn "Keeping SuperAgent images"
            fi
        else
            log "No SuperAgent images found"
        fi
    else
        log "Docker not available, skipping container cleanup"
    fi
}

# Remove files and directories
remove_files() {
    log "Removing SuperAgent files and directories..."

    # Remove installation directory
    if [[ -d "$INSTALL_DIR" ]]; then
        if ask_confirmation "Do you want to remove the SuperAgent installation directory ($INSTALL_DIR)?"; then
            rm -rf "$INSTALL_DIR"
            log "Installation directory removed: $INSTALL_DIR"
        else
            warn "Keeping installation directory: $INSTALL_DIR"
        fi
    else
        log "Installation directory not found: $INSTALL_DIR"
    fi

    # Remove configuration directory
    if [[ -d "$CONFIG_DIR" ]]; then
        if ask_confirmation "Do you want to remove the SuperAgent configuration directory ($CONFIG_DIR)?" "n"; then
            rm -rf "$CONFIG_DIR"
            log "Configuration directory removed: $CONFIG_DIR"
        else
            warn "Keeping configuration directory: $CONFIG_DIR"
        fi
    else
        log "Configuration directory not found: $CONFIG_DIR"
    fi

    # Remove or keep data directory
    if [[ -d "$DATA_DIR" ]]; then
        if [[ "$KEEP_DATA" == "true" ]]; then
            warn "Keeping data directory as requested: $DATA_DIR"
        else
            if ask_confirmation "Do you want to remove the SuperAgent data directory ($DATA_DIR)? This will delete all deployment data." "n"; then
                rm -rf "$DATA_DIR"
                log "Data directory removed: $DATA_DIR"
            else
                warn "Keeping data directory: $DATA_DIR"
            fi
        fi
    else
        log "Data directory not found: $DATA_DIR"
    fi

    # Remove log directory
    if [[ -d "$LOG_DIR" ]]; then
        if ask_confirmation "Do you want to remove the SuperAgent log directory ($LOG_DIR)?"; then
            rm -rf "$LOG_DIR"
            log "Log directory removed: $LOG_DIR"
        else
            warn "Keeping log directory: $LOG_DIR"
        fi
    else
        log "Log directory not found: $LOG_DIR"
    fi

    # Remove cache directory
    if [[ -d "/var/cache/superagent" ]]; then
        if ask_confirmation "Do you want to remove the SuperAgent cache directory (/var/cache/superagent)?"; then
            rm -rf "/var/cache/superagent"
            log "Cache directory removed: /var/cache/superagent"
        else
            warn "Keeping cache directory: /var/cache/superagent"
        fi
    else
        log "Cache directory not found: /var/cache/superagent"
    fi

    # Remove log rotation configuration
    if [[ -f "/etc/logrotate.d/superagent" ]]; then
        rm -f "/etc/logrotate.d/superagent"
        log "Log rotation configuration removed"
    else
        log "Log rotation configuration not found"
    fi

    # Remove from /usr/local/bin if symlinked
    if [[ -L "/usr/local/bin/superagent" ]]; then
        rm -f "/usr/local/bin/superagent"
        log "SuperAgent symlink removed from /usr/local/bin"
    fi
}

# Remove user and group
remove_user() {
    log "Checking SuperAgent user and group..."

    # Remove user
    if getent passwd "$AGENT_USER" &> /dev/null; then
        if ask_confirmation "Do you want to remove the SuperAgent system user ($AGENT_USER)?"; then
            # Check if user has any running processes
            if pgrep -u "$AGENT_USER" &> /dev/null; then
                warn "User $AGENT_USER has running processes. Attempting to terminate..."
                pkill -u "$AGENT_USER" || true
                sleep 2
            fi
            
            userdel "$AGENT_USER" 2>/dev/null || warn "Failed to remove user $AGENT_USER"
            log "User removed: $AGENT_USER"
        else
            warn "Keeping user: $AGENT_USER"
        fi
    else
        log "User not found: $AGENT_USER"
    fi

    # Remove group
    if getent group "$AGENT_GROUP" &> /dev/null; then
        if ask_confirmation "Do you want to remove the SuperAgent system group ($AGENT_GROUP)?"; then
            groupdel "$AGENT_GROUP" 2>/dev/null || warn "Failed to remove group $AGENT_GROUP"
            log "Group removed: $AGENT_GROUP"
        else
            warn "Keeping group: $AGENT_GROUP"
        fi
    else
        log "Group not found: $AGENT_GROUP"
    fi
}

# Remove package with confirmation
remove_package_with_confirmation() {
    local package_name="$1"
    local command_name="${2:-$1}"
    local description="$3"
    
    if ! command -v "$command_name" &> /dev/null; then
        log "$description is not installed"
        return 0
    fi
    
    info "$description is installed: $(which $command_name)"
    
    if ask_confirmation "Do you want to remove $description ($package_name)? This may affect other applications."; then
        info "Removing $package_name..."
        
        case "$PKG_MANAGER" in
            apt)
                $PKG_REMOVE "$package_name" || warn "Failed to remove $package_name"
                ;;
            yum|dnf|zypper)
                $PKG_REMOVE "$package_name" || warn "Failed to remove $package_name"
                ;;
            *)
                warn "Cannot remove $package_name automatically. Please remove manually if desired."
                ;;
        esac
        
        if ! command -v "$command_name" &> /dev/null; then
            log "$description removed successfully"
        else
            warn "$description may not have been completely removed"
        fi
    else
        log "Keeping $description"
    fi
}

# Remove Go installation
remove_go() {
    if [[ -d "/usr/local/go" ]]; then
        if ask_confirmation "Do you want to remove the Go installation (/usr/local/go)? This may affect other applications."; then
            rm -rf "/usr/local/go"
            
            # Remove from PATH in /etc/profile
            if grep -q "/usr/local/go/bin" /etc/profile; then
                sed -i '/\/usr\/local\/go\/bin/d' /etc/profile
            fi
            
            log "Go installation removed"
        else
            warn "Keeping Go installation"
        fi
    else
        log "Go installation not found in /usr/local/go"
    fi
}

# Remove installed packages
remove_packages() {
    if [[ "$REMOVE_PACKAGES" == "false" ]]; then
        log "Skipping package removal (use --remove-packages to enable)"
        return 0
    fi

    log "Checking installed packages..."
    
    if [[ "$PKG_MANAGER" == "unknown" ]]; then
        warn "Package manager not detected. Cannot remove packages automatically."
        return 0
    fi

    echo ""
    warn "⚠️  WARNING: Removing these packages may affect other applications on your system!"
    echo ""

    # Ask about Docker
    remove_package_with_confirmation "docker.io" "docker" "Docker"

    # Ask about Git
    remove_package_with_confirmation "git" "git" "Git"

    # Ask about curl
    remove_package_with_confirmation "curl" "curl" "curl"

    # Ask about wget
    remove_package_with_confirmation "wget" "wget" "wget"

    # Ask about OpenSSL
    remove_package_with_confirmation "openssl" "openssl" "OpenSSL"

    # Ask about Go (special handling)
    remove_go

    # Run autoremove to clean up orphaned packages
    if ask_confirmation "Do you want to run autoremove to clean up orphaned packages?"; then
        case "$PKG_MANAGER" in
            apt)
                apt-get autoremove -y || warn "Failed to run autoremove"
                apt-get autoclean -y || warn "Failed to run autoclean"
                ;;
            yum)
                yum autoremove -y || warn "Failed to run autoremove"
                ;;
            dnf)
                dnf autoremove -y || warn "Failed to run autoremove"
                ;;
            zypper)
                zypper remove -y --clean-deps || warn "Failed to clean dependencies"
                ;;
        esac
        log "Package cleanup completed"
    fi
}

# Main uninstallation function
uninstall_agent() {
    info "Starting SuperAgent uninstallation..."
    info "This will remove SuperAgent and optionally remove installed dependencies."
    info ""
    
    # Show what will be removed
    echo -e "${YELLOW}The following will be checked for removal:${NC}"
    echo "- SuperAgent service and systemd configuration"
    echo "- SuperAgent installation directory: $INSTALL_DIR"
    echo "- SuperAgent configuration directory: $CONFIG_DIR"
    echo "- SuperAgent data directory: $DATA_DIR"
    echo "- SuperAgent log directory: $LOG_DIR"
    echo "- SuperAgent cache directory: /var/cache/superagent"
    echo "- SuperAgent Docker network and containers"
    echo "- SuperAgent system user and group"
    
    if [[ "$REMOVE_PACKAGES" == "true" ]]; then
        echo "- Installed packages (Docker, Git, Go, etc.)"
    fi
    
    echo ""
    
    if [[ "$INTERACTIVE" == "true" ]]; then
        if ! ask_confirmation "Do you want to continue with the uninstallation?"; then
            info "Uninstallation cancelled by user"
            exit 0
        fi
    fi

    echo ""
    detect_os
    stop_and_remove_service
    remove_containers
    remove_docker_network
    remove_files
    remove_user
    remove_packages

    log "SuperAgent uninstallation completed!"

    echo ""
    echo -e "${GREEN}=== Uninstallation Summary ===${NC}"
    echo -e "${GREEN}✓${NC} SuperAgent service stopped and removed"
    echo -e "${GREEN}✓${NC} SuperAgent files and directories processed"
    echo -e "${GREEN}✓${NC} SuperAgent Docker resources processed"
    echo -e "${GREEN}✓${NC} SuperAgent user and group processed"
    
    if [[ "$REMOVE_PACKAGES" == "true" ]]; then
        echo -e "${GREEN}✓${NC} Package removal options presented"
    fi
    
    echo ""
    echo -e "${CYAN}SuperAgent has been uninstalled from your system.${NC}"
    
    # Check for remaining files
    remaining_files=""
    for dir in "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR" "/var/cache/superagent"; do
        if [[ -d "$dir" ]]; then
            remaining_files="$remaining_files\n  - $dir"
        fi
    done
    
    if [[ -n "$remaining_files" ]]; then
        echo ""
        echo -e "${YELLOW}Note: Some directories were kept at your request:${NC}"
        echo -e "$remaining_files"
        echo ""
        echo "You can manually remove these directories later if needed."
    fi
    
    echo ""
    echo -e "${CYAN}Thank you for using SuperAgent!${NC}"
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
                FORCE_REMOVE=true
                INTERACTIVE=false
                shift
                ;;
            --keep-data)
                KEEP_DATA=true
                shift
                ;;
            --remove-packages)
                REMOVE_PACKAGES=true
                shift
                ;;
            --help|-h)
                echo "SuperAgent Uninstallation Script"
                echo ""
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --non-interactive, -n    Run in non-interactive mode (keep everything)"
                echo "  --force, -f              Force removal without prompts (removes everything)"
                echo "  --keep-data              Keep data directory even in non-interactive mode"
                echo "  --remove-packages        Ask about removing installed packages"
                echo "  --help, -h               Show this help message"
                echo ""
                echo "Environment Variables:"
                echo "  AGENT_USER               System user for SuperAgent (default: superagent)"
                echo "  INSTALL_DIR              Installation directory (default: /opt/superagent)"
                echo "  CONFIG_DIR               Configuration directory (default: /etc/superagent)"
                echo "  LOG_DIR                  Log directory (default: /var/log/superagent)"
                echo "  DATA_DIR                 Data directory (default: /var/lib/superagent)"
                echo ""
                echo "Examples:"
                echo "  $0                       # Interactive uninstallation"
                echo "  $0 --remove-packages     # Also ask about removing packages"
                echo "  $0 --force               # Remove everything without prompts"
                echo "  $0 --keep-data           # Keep data directory"
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
    check_root
    uninstall_agent
}

# Run main function
main "$@"