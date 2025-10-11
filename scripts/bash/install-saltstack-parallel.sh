#!/bin/bash

# Parallel SaltStack Installation Script
# This script installs SaltStack Minion on multiple remote hosts via SSH in parallel

set -e

# Default values
DEFAULT_SALT_MASTER="salt-master"
DEFAULT_USER="root"
DEFAULT_PORT=22
DEFAULT_CONCURRENT=5
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
    echo "Usage: $0 [OPTIONS] <hosts_file>"
    echo ""
    echo "Options:"
    echo "  -h, --help                  Show this help message"
    echo "  -u, --user USER             SSH user (default: root)"
    echo "  -p, --port PORT             SSH port (default: 22)"
    echo "  -i, --identity KEY_PATH     SSH private key path"
    echo "  -P, --password PASSWORD     SSH password (not recommended for production)"
    echo "  -c, --concurrent COUNT      Number of concurrent installations (default: 5)"
    echo "  --master MASTER             SaltStack Master address (default: salt-master)"
    echo ""
    echo "Hosts file format:"
    echo "  # Lines starting with # are comments"
    echo "  # Format: hostname_or_ip[:port] [minion_id]"
    echo "  192.168.1.100"
    echo "  192.168.1.101 worker01"
    echo "  192.168.1.102:2222 worker02"
    echo ""
    echo "Examples:"
    echo "  $0 hosts.txt"
    echo "  $0 -u ubuntu -i ~/.ssh/id_rsa --master 192.168.1.10 hosts.txt"
    echo "  $0 -c 10 -P password123 hosts.txt"
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
            -c|--concurrent)
                CONCURRENT_COUNT="$2"
                shift 2
                ;;
            --master)
                SALT_MASTER="$2"
                shift 2
                ;;
            -*)
                print_error "Unknown option $1"
                usage
                exit 1
                ;;
            *)
                HOSTS_FILE="$1"
                shift
                ;;
        esac
    done
}

# Validate required arguments
validate_args() {
    if [[ -z "$HOSTS_FILE" ]]; then
        print_error "Hosts file is required"
        usage
        exit 1
    fi

    if [[ ! -f "$HOSTS_FILE" ]]; then
        print_error "Hosts file not found: $HOSTS_FILE"
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
    CONCURRENT_COUNT="${CONCURRENT_COUNT:-$DEFAULT_CONCURRENT}"
    SALT_MASTER="${SALT_MASTER:-$DEFAULT_SALT_MASTER}"
}

# Read hosts from file
read_hosts() {
    print_info "Reading hosts from $HOSTS_FILE..."
    
    HOSTS=()
    while IFS= read -r line || [[ -n "$line" ]]; do
        # Skip empty lines and comments
        [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
        
        # Remove leading/trailing whitespace
        line=$(echo "$line" | xargs)
        
        # Extract host and optional minion ID
        if [[ "$line" =~ ^([^[:space:]]+)[[:space:]]+(.+)$ ]]; then
            HOSTS+=("${BASH_REMATCH[1]}|${BASH_REMATCH[2]}")
        else
            HOSTS+=("${line}|")
        fi
    done < "$HOSTS_FILE"
    
    if [[ ${#HOSTS[@]} -eq 0 ]]; then
        print_error "No hosts found in $HOSTS_FILE"
        exit 1
    fi
    
    print_success "Found ${#HOSTS[@]} hosts to process"
}

# Install SaltStack on a single host
install_on_host() {
    local host_entry="$1"
    local host_port="${host_entry%|*}"
    local minion_id="${host_entry#*|}"
    
    local host="${host_port%:*}"
    local port="${host_port#*:}"
    
    # If port is the same as host, it means no port was specified
    if [[ "$port" == "$host" ]]; then
        port="$SSH_PORT"
    fi
    
    local log_file="/tmp/saltstack-install-${host//./-}.log"
    
    {
        echo "=== Installing SaltStack on $host:$port ==="
        echo "Minion ID: ${minion_id:-<default>}"
        echo ""
        
        # Use the single host installation script
        local cmd_args=("-u" "$SSH_USER" "-p" "$port" "--master" "$SALT_MASTER")
        
        if [[ -n "$minion_id" ]]; then
            cmd_args+=("--minion-id" "$minion_id")
        fi
        
        if [[ -n "$SSH_KEY_PATH" ]]; then
            cmd_args+=("-i" "$SSH_KEY_PATH")
        elif [[ -n "$SSH_PASSWORD" ]]; then
            cmd_args+=("-P" "$SSH_PASSWORD")
        fi
        
        cmd_args+=("$host")
        
        if "$SCRIPT_DIR/install-saltstack-remote.sh" "${cmd_args[@]}"; then
            echo "SUCCESS: SaltStack installed on $host:$port"
            return 0
        else
            echo "FAILED: SaltStack installation failed on $host:$port"
            return 1
        fi
    } 2>&1 | tee "$log_file"
    
    # Return the exit code of the installation command
    tail -n 1 "$log_file" | grep -q "SUCCESS" && return 0 || return 1
}

# Install SaltStack on all hosts in parallel
install_on_all_hosts() {
    print_info "Starting parallel SaltStack installation on ${#HOSTS[@]} hosts (concurrency: $CONCURRENT_COUNT)..."
    
    local success_count=0
    local fail_count=0
    local active_jobs=()
    local job_hosts=()
    
    # Process hosts in batches
    for (( i=0; i<${#HOSTS[@]}; i++ )); do
        local host_entry="${HOSTS[$i]}"
        local host_port="${host_entry%|*}"
        local host="${host_port%:*}"
        
        # Wait if we've reached the concurrency limit
        while [[ ${#active_jobs[@]} -ge $CONCURRENT_COUNT ]]; do
            for (( j=0; j<${#active_jobs[@]}; j++ )); do
                local job="${active_jobs[$j]}"
                local job_host="${job_hosts[$j]}"
                
                if ! kill -0 "$job" 2>/dev/null; then
                    # Job finished, check result
                    wait "$job"
                    local exit_code=$?
                    
                    if [[ $exit_code -eq 0 ]]; then
                        ((success_count++))
                        print_success "Installation completed successfully on $job_host"
                    else
                        ((fail_count++))
                        print_error "Installation failed on $job_host"
                    fi
                    
                    # Remove job from active list
                    unset 'active_jobs[$j]'
                    unset 'job_hosts[$j]'
                    active_jobs=("${active_jobs[@]}")
                    job_hosts=("${job_hosts[@]}")
                    break
                fi
            done
            
            # Small delay to avoid busy waiting
            sleep 0.1
        done
        
        # Start new job
        install_on_host "$host_entry" &
        local job_pid=$!
        active_jobs+=("$job_pid")
        job_hosts+=("$host")
        print_info "Started installation on $host (PID: $job_pid)"
    done
    
    # Wait for all remaining jobs to complete
    for (( i=0; i<${#active_jobs[@]}; i++ )); do
        local job="${active_jobs[$i]}"
        local job_host="${job_hosts[$i]}"
        
        wait "$job"
        local exit_code=$?
        
        if [[ $exit_code -eq 0 ]]; then
            ((success_count++))
            print_success "Installation completed successfully on $job_host"
        else
            ((fail_count++))
            print_error "Installation failed on $job_host"
        fi
    done
    
    # Print summary
    echo ""
    print_info "=== Installation Summary ==="
    print_success "Successful installations: $success_count"
    if [[ $fail_count -gt 0 ]]; then
        print_error "Failed installations: $fail_count"
    else
        print_success "Failed installations: $fail_count"
    fi
    
    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    
    return 0
}

# Main function
main() {
    parse_args "$@"
    validate_args
    read_hosts
    install_on_all_hosts
    
    if [[ $? -eq 0 ]]; then
        print_success "All SaltStack installations completed"
    else
        print_error "Some installations failed"
        exit 1
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi