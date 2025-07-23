#!/bin/bash
set -e

echo "Starting Salt Infrastructure..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
MAX_WAIT_TIME=180
POLL_INTERVAL=5

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to wait for containers to be healthy
wait_for_containers() {
    print_status "Waiting for containers to start..."
    
    local start_time=$(date +%s)
    
    while true; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -gt $MAX_WAIT_TIME ]; then
            print_error "Timeout waiting for containers to start"
            return 1
        fi
        
        # Check if master is healthy
        if docker-compose ps salt-master | grep -q "healthy"; then
            print_status "Salt Master is healthy"
            break
        else
            print_warning "Waiting for Salt Master to be healthy... (${elapsed}s)"
            sleep $POLL_INTERVAL
        fi
    done
    
    # Wait a bit more for minions to connect
    print_status "Waiting for minions to connect..."
    sleep 20
}

# Function to check minion status
check_minion_status() {
    print_status "Checking minion status..."
    
    local expected_minions=("minion-1" "minion-2" "minion-3")
    local connected_minions=0
    
    for minion in "${expected_minions[@]}"; do
        if docker exec salt-master salt-key -l accepted | grep -q "^$minion$"; then
            print_status "✓ $minion is accepted"
            ((connected_minions++))
        else
            print_warning "✗ $minion is not accepted"
        fi
    done
    
    print_status "Connected minions: $connected_minions/${#expected_minions[@]}"
}

# Function to apply initial configuration
apply_initial_config() {
    print_status "Applying initial Salt configuration..."
    
    # Apply highstate to all minions
    if docker exec salt-master salt '*' state.apply --timeout=120; then
        print_status "✓ Initial configuration applied successfully"
    else
        print_warning "Initial configuration application had issues"
    fi
}

# Main execution
main() {
    print_status "Starting Salt Infrastructure with Docker Compose..."
    
    # Build and start containers
    docker-compose build
    docker-compose up -d
    
    # Wait for containers to be ready
    wait_for_containers
    
    # Check status
    check_minion_status
    
    # Apply initial configuration
    apply_initial_config
    
    print_status "Salt infrastructure is ready!"
    print_status "You can now run tests with: ./run-tests-full.sh"
    print_status "Master logs: docker-compose logs -f salt-master"
    print_status "Minion logs: docker-compose logs -f salt-minion-1"
    
    # Show running containers
    echo ""
    print_status "Running containers:"
    docker-compose ps
}

# Handle script interruption
trap 'print_error "Script interrupted"; exit 1' INT TERM

# Execute main function
main "$@"
