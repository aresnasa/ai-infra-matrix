#!/bin/bash
set -e

# ==============================================================================
# AI Infrastructure Matrix - Refactored Build Script
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"
ENV_EXAMPLE="$SCRIPT_DIR/.env.example"
SRC_DIR="$SCRIPT_DIR/src"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ==============================================================================
# 1. Configuration & Environment
# ==============================================================================

if [ ! -f "$ENV_FILE" ]; then
    log_warn ".env file not found. Creating from .env.example..."
    if [ -f "$ENV_EXAMPLE" ]; then
        cp "$ENV_EXAMPLE" "$ENV_FILE"
    else
        log_error ".env.example not found! Cannot initialize configuration."
        exit 1
    fi
fi

# Load .env variables
set -a
source "$ENV_FILE"
set +a

# Ensure SSH Keys
SSH_KEY_DIR="$SCRIPT_DIR/ssh-key"
if [ ! -f "$SSH_KEY_DIR/id_rsa" ]; then
    log_info "Generating SSH keys..."
    mkdir -p "$SSH_KEY_DIR"
    ssh-keygen -t rsa -b 4096 -f "$SSH_KEY_DIR/id_rsa" -N "" -C "ai-infra-system@shared"
fi

# Ensure Third Party Directory
mkdir -p "$SCRIPT_DIR/third_party"

# ==============================================================================
# 2. Helper Functions
# ==============================================================================

detect_compose_command() {
    if command -v docker-compose >/dev/null 2>&1; then
        echo "docker-compose"
    elif docker compose version >/dev/null 2>&1; then
        echo "docker compose"
    else
        return 1
    fi
}

detect_external_host() {
    # Simplified host detection
    local ip=""
    if command -v ip >/dev/null 2>&1; then
        ip=$(ip route get 1 | awk '{print $7;exit}')
    elif command -v ifconfig >/dev/null 2>&1; then
        ip=$(ifconfig | grep -E "inet " | grep -v 127.0.0.1 | awk '{print $2}' | head -n 1)
    fi
    echo "${ip:-localhost}"
}

wait_for_apphub_ready() {
    local timeout="${1:-300}"
    local container_name="ai-infra-apphub"
    local check_interval=5
    local elapsed=0
    
    local apphub_port="${APPHUB_PORT:-53434}"
    local external_host="${EXTERNAL_HOST:-$(detect_external_host)}"
    local apphub_url="http://${external_host}:${apphub_port}"
    
    log_info "Waiting for AppHub at $apphub_url (Timeout: ${timeout}s)..."
    
    while [[ $elapsed -lt $timeout ]]; do
        # Check if container is running
        if ! docker ps --filter "name=$container_name" --filter "status=running" | grep -q "$container_name"; then
            log_warn "[${elapsed}s] Container not running..."
            sleep $check_interval
            elapsed=$((elapsed + check_interval))
            continue
        fi
        
        # Check if packages are accessible
        if curl -sf --connect-timeout 2 --max-time 5 "${apphub_url}/pkgs/slurm-deb/Packages" >/dev/null 2>&1; then
            log_info "✅ AppHub is ready!"
            return 0
        fi
        
        log_warn "[${elapsed}s] AppHub not ready yet..."
        sleep $check_interval
        elapsed=$((elapsed + check_interval))
    done
    
    log_error "❌ AppHub failed to become ready."
    return 1
}

# ==============================================================================
# 3. Build Logic
# ==============================================================================

# Prepare base build args
BASE_BUILD_ARGS=()
if [ -f "$ENV_EXAMPLE" ]; then
    while IFS='=' read -r key value; do
        [[ "$key" =~ ^#.*$ ]] && continue
        [[ -z "$key" ]] && continue
        curr_val="${!key}"
        if [ -n "$curr_val" ]; then
            BASE_BUILD_ARGS+=("--build-arg" "$key=$curr_val")
        fi
    done < <(grep -v '^#' "$ENV_EXAMPLE")
fi
BASE_BUILD_ARGS+=("--build-arg" "BUILD_ENV=${BUILD_ENV:-production}")

build_component() {
    local component="$1"
    local extra_args=("${@:2}") # Capture all remaining arguments
    local component_dir="$SRC_DIR/$component"
    
    if [ ! -d "$component_dir" ]; then
        log_error "Component directory not found: $component_dir"
        return 1
    fi

    # Check for dependency configuration (External Image)
    local dep_conf="$component_dir/dependency.conf"
    if [ -f "$dep_conf" ]; then
        local upstream_image=$(grep -v '^#' "$dep_conf" | head -n 1 | tr -d '[:space:]')
        if [ -z "$upstream_image" ]; then
            log_error "Empty dependency config for $component"
            return 1
        fi
        
        local target_image="ai-infra-$component:${IMAGE_TAG:-latest}"
        if [ -n "$PRIVATE_REGISTRY" ]; then
            target_image="$PRIVATE_REGISTRY/$target_image"
        fi
        
        log_info "Processing dependency $component: $upstream_image -> $target_image"
        
        if docker pull "$upstream_image"; then
            if docker tag "$upstream_image" "$target_image"; then
                log_info "✓ Dependency ready: $target_image"
                return 0
            else
                log_error "✗ Failed to tag $upstream_image"
                return 1
            fi
        else
            log_error "✗ Failed to pull $upstream_image"
            return 1
        fi
    fi
    
    if [ ! -f "$component_dir/Dockerfile" ]; then
        log_warn "No Dockerfile or dependency.conf in $component, skipping..."
        return 0
    fi

    # Check for build-targets.conf
    local targets_file="$component_dir/build-targets.conf"
    local targets=()
    local images=()
    
    if [ -f "$targets_file" ]; then
        while read -r target image_suffix || [ -n "$target" ]; do
            [[ "$target" =~ ^#.*$ ]] && continue
            [[ -z "$target" ]] && continue
            targets+=("$target")
            images+=("$image_suffix")
        done < "$targets_file"
    else
        targets+=("default")
        images+=("ai-infra-$component")
    fi

    for i in "${!targets[@]}"; do
        local target="${targets[$i]}"
        local image_name="${images[$i]}"
        local full_image_name="${image_name}:${IMAGE_TAG:-latest}"
        
        if [ -n "$PRIVATE_REGISTRY" ]; then
            full_image_name="$PRIVATE_REGISTRY/$full_image_name"
        fi
        
        log_info "Building $component [$target] -> $full_image_name"
        
        local cmd=("docker" "build" "${BASE_BUILD_ARGS[@]}" "${extra_args[@]}" "-t" "$full_image_name" "-f" "$component_dir/Dockerfile")
        
        if [ "$target" != "default" ]; then
            cmd+=("--target" "$target")
        fi
        
        # Add build context (project root)
        cmd+=("$SCRIPT_DIR")
        
        if "${cmd[@]}"; then
            log_info "✓ Build success: $full_image_name"
        else
            log_error "✗ Build failed: $full_image_name"
            return 1
        fi
    done
}

discover_services() {
    log_info "Discovering components in $SRC_DIR..."
    DEPENDENCY_SERVICES=()
    FOUNDATION_SERVICES=()
    DEPENDENT_SERVICES=()

    # Use find to avoid issues if directory is empty and sort for deterministic order
    while IFS= read -r dir; do
        local component=$(basename "$dir")
        
        # 1. Check for dependency.conf (External Image)
        if [ -f "$dir/dependency.conf" ]; then
            DEPENDENCY_SERVICES+=("$component")
            continue
        fi
        
        # 2. Check for Dockerfile (Buildable Component)
        if [ -f "$dir/Dockerfile" ]; then
            local phase="dependent" # Default phase
            
            # Check for build.conf override
            if [ -f "$dir/build.conf" ]; then
                local conf_phase=$(grep "^BUILD_PHASE=" "$dir/build.conf" | cut -d= -f2 | tr -d '[:space:]')
                if [ -n "$conf_phase" ]; then
                    phase="$conf_phase"
                fi
            fi
            
            if [ "$phase" == "foundation" ]; then
                FOUNDATION_SERVICES+=("$component")
            else
                DEPENDENT_SERVICES+=("$component")
            fi
        fi
    done < <(find "$SRC_DIR" -mindepth 1 -maxdepth 1 -type d | sort)
    
    log_info "Found ${#DEPENDENCY_SERVICES[@]} dependency services: ${DEPENDENCY_SERVICES[*]}"
    log_info "Found ${#FOUNDATION_SERVICES[@]} foundation services: ${FOUNDATION_SERVICES[*]}"
    log_info "Found ${#DEPENDENT_SERVICES[@]} dependent services: ${DEPENDENT_SERVICES[*]}"
}

build_all() {
    log_info "Starting coordinated build process..."
    
    # Discover services dynamically
    discover_services
    
    # 1. Pull & Tag Dependency Services
    log_info "=== Phase 0: Processing Dependency Services ==="
    for service in "${DEPENDENCY_SERVICES[@]}"; do
        build_component "$service"
    done
    
    # 2. Build Foundation Services
    log_info "=== Phase 1: Building Foundation Services ==="
    for service in "${FOUNDATION_SERVICES[@]}"; do
        build_component "$service"
    done
    
    # 2. Start AppHub Service
    log_info "=== Phase 2: Starting AppHub Service ==="
    local compose_cmd=$(detect_compose_command)
    if [ -z "$compose_cmd" ]; then
        log_error "docker-compose not found! Cannot start AppHub."
        exit 1
    fi
    
    log_info "Starting AppHub container..."
    $compose_cmd up -d apphub
    
    if ! wait_for_apphub_ready 300; then
        log_error "AppHub failed to start. Aborting build."
        exit 1
    fi
    
    # 3. Build Dependent Services
    log_info "=== Phase 3: Building Dependent Services ==="
    
    # Determine AppHub URL for build args
    local apphub_port="${APPHUB_PORT:-53434}"
    local external_host="${EXTERNAL_HOST:-$(detect_external_host)}"
    local apphub_url="http://${external_host}:${apphub_port}"
    
    log_info "Using AppHub URL for builds: $apphub_url"
    
    for service in "${DEPENDENT_SERVICES[@]}"; do
        # Pass APPHUB_URL to dependent services
        build_component "$service" "--build-arg" "APPHUB_URL=$apphub_url"
    done
    
    log_info "=== Build Process Completed Successfully ==="
}

start_all() {
    log_info "Starting all services..."
    local compose_cmd=$(detect_compose_command)
    if [ -z "$compose_cmd" ]; then
        log_error "docker-compose not found!"
        exit 1
    fi
    
    $compose_cmd up -d
    log_info "All services started."
}

print_help() {
    echo "Usage: $0 [command] [component]"
    echo ""
    echo "Commands:"
    echo "  build-all, all      Build all components in the correct order (AppHub first)"
    echo "  start-all           Start all services using docker-compose"
    echo "  [component]         Build a specific component (e.g., backend, frontend)"
    echo ""
    echo "Examples:"
    echo "  $0 build-all"
    echo "  $0 start-all"
    echo "  $0 backend"
}

# ==============================================================================
# 4. Main Execution
# ==============================================================================

if [ $# -eq 0 ]; then
    print_help
    exit 0
fi

case "$1" in
    build-all|all)
        build_all
        ;;
    start-all)
        start_all
        ;;
    help|--help|-h)
        print_help
        ;;
    *)
        # Single component build
        for component in "$@"; do
            build_component "$component"
        done
        ;;
esac
