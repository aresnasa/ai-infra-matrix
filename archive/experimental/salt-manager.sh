#!/bin/bash

# Salt Infrastructure Management Script
# Provides easy management of the Salt Docker infrastructure

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}=====================================
$1
=====================================${NC}"
}

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_usage() {
    print_header "Salt Infrastructure Management"
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  start           - Start the infrastructure"
    echo "  stop            - Stop the infrastructure"
    echo "  restart         - Restart the infrastructure"
    echo "  status          - Show status of all services"
    echo "  logs [service]  - Show logs (optionally for specific service)"
    echo "  test            - Run complete test suite"
    echo "  shell <minion>  - Open shell in minion container"
    echo "  exec <cmd>      - Execute Salt command on master"
    echo "  keys            - Show accepted/pending keys"
    echo "  apply           - Apply Salt states to all minions"
    echo "  clean           - Stop and clean up everything"
    echo "  rebuild         - Rebuild and restart infrastructure"
    echo ""
    echo "Examples:"
    echo "  $0 start"
    echo "  $0 logs salt-master"
    echo "  $0 shell minion-1"
    echo "  $0 exec \"salt '*' test.ping\""
}

cmd_start() {
    print_status "Starting Salt infrastructure..."
    ./start.sh
}

cmd_stop() {
    print_status "Stopping Salt infrastructure..."
    ./stop.sh
}

cmd_restart() {
    print_status "Restarting Salt infrastructure..."
    ./stop.sh
    sleep 3
    ./start.sh
}

cmd_status() {
    print_header "Infrastructure Status"
    docker-compose ps
    echo ""
    
    print_status "Container Health:"
    docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
    echo ""
    
    if docker-compose ps | grep -q "Up"; then
        print_status "Checking Salt connectivity..."
        docker exec salt-master salt-run manage.status 2>/dev/null || print_warning "Salt master not responding"
    fi
}

cmd_logs() {
    local service=${1:-}
    if [ -n "$service" ]; then
        print_status "Showing logs for $service..."
        docker-compose logs -f "$service"
    else
        print_status "Showing logs for all services..."
        docker-compose logs -f
    fi
}

cmd_test() {
    print_status "Running complete test suite..."
    ./run-tests-full.sh
}

cmd_shell() {
    local minion=${1:-minion-1}
    local container="salt-$minion"
    
    print_status "Opening shell in $container..."
    docker exec -it "$container" /bin/bash || docker exec -it "$container" /bin/sh
}

cmd_exec() {
    local command="$1"
    if [ -z "$command" ]; then
        print_error "No command provided"
        return 1
    fi
    
    print_status "Executing: $command"
    docker exec salt-master $command
}

cmd_keys() {
    print_header "Salt Key Management"
    print_status "Accepted keys:"
    docker exec salt-master salt-key -l accepted
    echo ""
    
    print_status "Pending keys:"
    docker exec salt-master salt-key -l pending || echo "No pending keys"
    echo ""
    
    print_status "Rejected keys:"
    docker exec salt-master salt-key -l rejected || echo "No rejected keys"
}

cmd_apply() {
    print_status "Applying Salt states to all minions..."
    docker exec salt-master salt '*' state.apply --timeout=120
}

cmd_clean() {
    print_warning "This will stop and remove all containers and volumes"
    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        ./stop.sh force
        print_status "Cleanup completed"
    else
        print_status "Cleanup cancelled"
    fi
}

cmd_rebuild() {
    print_status "Rebuilding infrastructure..."
    docker-compose down
    docker-compose build --no-cache
    ./start.sh
}

# Main execution
main() {
    local command=${1:-}
    
    if [ -z "$command" ]; then
        show_usage
        exit 1
    fi
    
    case "$command" in
        "start")
            cmd_start
            ;;
        "stop")
            cmd_stop
            ;;
        "restart")
            cmd_restart
            ;;
        "status")
            cmd_status
            ;;
        "logs")
            cmd_logs "$2"
            ;;
        "test")
            cmd_test
            ;;
        "shell")
            cmd_shell "$2"
            ;;
        "exec")
            cmd_exec "$2"
            ;;
        "keys")
            cmd_keys
            ;;
        "apply")
            cmd_apply
            ;;
        "clean")
            cmd_clean
            ;;
        "rebuild")
            cmd_rebuild
            ;;
        "help"|"-h"|"--help")
            show_usage
            ;;
        *)
            print_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"
