#!/bin/bash

# Remote SaltStack Installation Script
# This script installs SaltStack Minion on remote hosts via SSH

set -e

# Default values
DEFAULT_SALT_MASTER="salt-master"
DEFAULT_SALT_MINION_ID=""
DEFAULT_USER="root"
DEFAULT_PORT=22
DEFAULT_KEY_PATH=""
DEFAULT_PASSWORD=""
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print usage information
usage() {
    echo "Usage: $0 [OPTIONS] <host>"
    echo ""
    echo "Options:"
    echo "  -h, --help                  Show this help message"
    echo "  -u, --user USER             SSH user (default: root)"
    echo "  -p, --port PORT             SSH port (default: 22)"
    echo "  -i, --identity KEY_PATH     SSH private key path"
    echo "  -P, --password PASSWORD     SSH password (not recommended for production)"
    echo "  --master MASTER             SaltStack Master address (default: salt-master)"
    echo "  --minion-id MINION_ID       SaltStack Minion ID (default: hostname)"
    echo ""
    echo "Examples:"
    echo "  $0 192.168.1.100"
    echo "  $0 -u ubuntu -i ~/.ssh/id_rsa --master 192.168.1.10 192.168.1.100"
    echo "  $0 -u centos -P password123 --minion-id worker01 192.168.1.101"
}

# Print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            -u|--user)
                SSH_USER="$2"
                shift 2
                ;;
            -p|--port)
                SSH_PORT="$2"
                shift 2
                ;;
            -i|--identity)
                SSH_KEY_PATH="$2"
                shift 2
                ;;
            -P|--password)
                SSH_PASSWORD="$2"
                shift 2
                ;;
            --master)
                SALT_MASTER="$2"
                shift 2
                ;;
            --minion-id)
                SALT_MINION_ID="$2"
                shift 2
                ;;
            -*)
                print_error "Unknown option $1"
                usage
                exit 1
                ;;
            *)
                TARGET_HOST="$1"
                shift
                ;;
        esac
    done
}

# Validate required arguments
validate_args() {
    if [[ -z "$TARGET_HOST" ]]; then
        print_error "Target host is required"
        usage
        exit 1
    fi

    if [[ -n "$SSH_PASSWORD" ]] && [[ -n "$SSH_KEY_PATH" ]]; then
        print_warning "Both password and key provided. Using key for authentication."
    fi

    if [[ -z "$SSH_PASSWORD" ]] && [[ -z "$SSH_KEY_PATH" ]]; then
        # Try to use default SSH key
        if [[ -f "$HOME/.ssh/id_rsa" ]]; then
            SSH_KEY_PATH="$HOME/.ssh/id_rsa"
            print_info "Using default SSH key: $SSH_KEY_PATH"
        else
            print_error "No authentication method provided. Please specify either password or SSH key."
            exit 1
        fi
    fi

    # Set defaults if not provided
    SSH_USER="${SSH_USER:-$DEFAULT_USER}"
    SSH_PORT="${SSH_PORT:-$DEFAULT_PORT}"
    SALT_MASTER="${SALT_MASTER:-$DEFAULT_SALT_MASTER}"
    SALT_MINION_ID="${SALT_MINION_ID:-$DEFAULT_SALT_MINION_ID}"
}

# Check SSH connectivity
check_ssh() {
    print_info "Checking SSH connectivity to $TARGET_HOST:$SSH_PORT..."
    
    if [[ -n "$SSH_KEY_PATH" ]]; then
        if ! ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -i "$SSH_KEY_PATH" -p "$SSH_PORT" "$SSH_USER@$TARGET_HOST" "echo 'SSH connection successful'" 2>/dev/null; then
            print_error "Cannot establish SSH connection to $TARGET_HOST:$SSH_PORT with key authentication"
            exit 1
        fi
    elif [[ -n "$SSH_PASSWORD" ]]; then
        if ! sshpass -e ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -p "$SSH_PORT" "$SSH_USER@$TARGET_HOST" "echo 'SSH connection successful'" 2>/dev/null; then
            print_error "Cannot establish SSH connection to $TARGET_HOST:$SSH_PORT with password authentication"
            exit 1
        fi
    fi
    
    print_success "SSH connection established successfully"
}

# Detect OS distribution
detect_os() {
    print_info "Detecting OS distribution..."
    
    local os_info
    if [[ -n "$SSH_KEY_PATH" ]]; then
        os_info=$(ssh -o StrictHostKeyChecking=no -i "$SSH_KEY_PATH" -p "$SSH_PORT" "$SSH_USER@$TARGET_HOST" '
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                echo "$ID|$VERSION_ID|$ARCHITECTURE"
            elif [ -f /etc/redhat-release ]; then
                if grep -q "CentOS" /etc/redhat-release; then
                    echo "centos|$(rpm -q --queryformat "%{VERSION}" centos-release)|$(uname -m)"
                elif grep -q "Red Hat" /etc/redhat-release; then
                    echo "rhel|$(rpm -q --queryformat "%{VERSION}" redhat-release)|$(uname -m)"
                else
                    echo "unknown||$(uname -m)"
                fi
            else
                echo "unknown||$(uname -m)"
            fi
        ' 2>/dev/null)
    elif [[ -n "$SSH_PASSWORD" ]]; then
        export SSHPASS="$SSH_PASSWORD"
        os_info=$(sshpass -e ssh -o StrictHostKeyChecking=no -p "$SSH_PORT" "$SSH_USER@$TARGET_HOST" '
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                echo "$ID|$VERSION_ID|$ARCHITECTURE"
            elif [ -f /etc/redhat-release ]; then
                if grep -q "CentOS" /etc/redhat-release; then
                    echo "centos|$(rpm -q --queryformat "%{VERSION}" centos-release)|$(uname -m)"
                elif grep -q "Red Hat" /etc/redhat-release; then
                    echo "rhel|$(rpm -q --queryformat "%{VERSION}" redhat-release)|$(uname -m)"
                else
                    echo "unknown||$(uname -m)"
                fi
            else
                echo "unknown||$(uname -m)"
            fi
        ' 2>/dev/null)
    fi
    
    OS_NAME=$(echo "$os_info" | cut -d'|' -f1)
    OS_VERSION=$(echo "$os_info" | cut -d'|' -f2)
    OS_ARCH=$(echo "$os_info" | cut -d'|' -f3)
    
    if [[ "$OS_NAME" == "unknown" ]]; then
        print_error "Unsupported or unknown OS distribution"
        exit 1
    fi
    
    print_success "Detected OS: $OS_NAME $OS_VERSION ($OS_ARCH)"
}

# Install SaltStack based on OS
install_saltstack() {
    print_info "Installing SaltStack Minion..."
    
    local install_cmd
    
    case "$OS_NAME" in
        ubuntu|debian)
            install_cmd="
                set -e
                export DEBIAN_FRONTEND=noninteractive
                apt-get update
                apt-get install -y curl wget gnupg2
                curl -fsSL https://packages.broadcom.com/artifactory/api/security/keypair/SaltProjectKey/public | gpg --dearmor > /etc/apt/trusted.gpg.d/saltproject.gpg
                echo 'deb [signed-by=/etc/apt/trusted.gpg.d/saltproject.gpg] https://packages.broadcom.com/artifactory/saltproject-deb/ stable main' > /etc/apt/sources.list.d/saltstack.list
                apt-get update
                apt-get install -y salt-minion
            "
            ;;
        centos|rhel)
            if [[ "${OS_VERSION%.*}" -ge 8 ]]; then
                install_cmd="
                    set -e
                    yum install -y https://repo.saltproject.io/salt/py3/redhat/8/x86_64/latest/salt-repo-latest.el8.noarch.rpm
                    yum clean expire-cache
                    yum install -y salt-minion
                "
            else
                print_error "Unsupported CentOS/RHEL version: $OS_VERSION"
                exit 1
            fi
            ;;
        *)
            print_error "Unsupported OS: $OS_NAME"
            exit 1
            ;;
    esac
    
    if [[ -n "$SSH_KEY_PATH" ]]; then
        if ! ssh -o StrictHostKeyChecking=no -i "$SSH_KEY_PATH" -p "$SSH_PORT" "$SSH_USER@$TARGET_HOST" "$install_cmd"; then
            print_error "Failed to install SaltStack"
            exit 1
        fi
    elif [[ -n "$SSH_PASSWORD" ]]; then
        export SSHPASS="$SSH_PASSWORD"
        if ! sshpass -e ssh -o StrictHostKeyChecking=no -p "$SSH_PORT" "$SSH_USER@$TARGET_HOST" "$install_cmd"; then
            print_error "Failed to install SaltStack"
            exit 1
        fi
    fi
    
    print_success "SaltStack Minion installed successfully"
}

# Configure SaltStack Minion
configure_saltstack() {
    print_info "Configuring SaltStack Minion..."
    
    local minion_id_config=""
    if [[ -n "$SALT_MINION_ID" ]]; then
        minion_id_config="id: $SALT_MINION_ID"
    fi
    
    local config_cmd="
        set -e
        cat > /etc/salt/minion << EOF
master: $SALT_MASTER
$minion_id_config
log_level: info
EOF
        systemctl enable salt-minion
    "
    
    if [[ -n "$SSH_KEY_PATH" ]]; then
        if ! ssh -o StrictHostKeyChecking=no -i "$SSH_KEY_PATH" -p "$SSH_PORT" "$SSH_USER@$TARGET_HOST" "$config_cmd"; then
            print_error "Failed to configure SaltStack Minion"
            exit 1
        fi
    elif [[ -n "$SSH_PASSWORD" ]]; then
        export SSHPASS="$SSH_PASSWORD"
        if ! sshpass -e ssh -o StrictHostKeyChecking=no -p "$SSH_PORT" "$SSH_USER@$TARGET_HOST" "$config_cmd"; then
            print_error "Failed to configure SaltStack Minion"
            exit 1
        fi
    fi
    
    print_success "SaltStack Minion configured successfully"
}

# Start SaltStack Minion service
start_saltstack() {
    print_info "Starting SaltStack Minion service..."
    
    local start_cmd="systemctl start salt-minion"
    
    if [[ -n "$SSH_KEY_PATH" ]]; then
        if ! ssh -o StrictHostKeyChecking=no -i "$SSH_KEY_PATH" -p "$SSH_PORT" "$SSH_USER@$TARGET_HOST" "$start_cmd"; then
            print_error "Failed to start SaltStack Minion service"
            exit 1
        fi
    elif [[ -n "$SSH_PASSWORD" ]]; then
        export SSHPASS="$SSH_PASSWORD"
        if ! sshpass -e ssh -o StrictHostKeyChecking=no -p "$SSH_PORT" "$SSH_USER@$TARGET_HOST" "$start_cmd"; then
            print_error "Failed to start SaltStack Minion service"
            exit 1
        fi
    fi
    
    print_success "SaltStack Minion service started successfully"
}

# Main function
main() {
    parse_args "$@"
    validate_args
    check_ssh
    detect_os
    install_saltstack
    configure_saltstack
    start_saltstack
    
    print_success "SaltStack Minion installation and configuration completed on $TARGET_HOST"
    echo "Please remember to accept the minion key on your SaltStack Master:"
    echo "  salt-key -a ${SALT_MINION_ID:-$TARGET_HOST}"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi