#!/bin/bash

echo "Stopping Salt Infrastructure..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to gracefully stop containers
graceful_stop() {
    print_status "Stopping containers gracefully..."
    docker-compose stop
    
    print_status "Removing containers..."
    docker-compose down
    
    print_status "Salt infrastructure stopped"
}

# Function to force stop and cleanup
force_cleanup() {
    print_warning "Force stopping and cleaning up..."
    docker-compose down -v --remove-orphans
    
    print_status "Cleaning up unused volumes..."
    docker volume prune -f
    
    print_status "Force cleanup completed"
}

# Main execution
main() {
    case "${1:-graceful}" in
        "force")
            force_cleanup
            ;;
        "graceful"|*)
            graceful_stop
            ;;
    esac
}

print_status "Usage: $0 [graceful|force]"
print_status "  graceful: Stop containers gracefully (default)"
print_status "  force: Force stop and cleanup volumes"
echo ""

main "$@"
