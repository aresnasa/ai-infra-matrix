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
# Pull Functions - 镜像拉取功能
# ==============================================================================

# Default retry settings
DEFAULT_MAX_RETRIES=3
DEFAULT_RETRY_DELAY=5

# Log file for tracking failures
FAILURE_LOG="${SCRIPT_DIR}/.build-failures.log"

# Log failure to file
log_failure() {
    local operation="$1"
    local target="$2"
    local error_msg="$3"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] $operation FAILED: $target - $error_msg" >> "$FAILURE_LOG"
    log_error "[$timestamp] $operation FAILED: $target - $error_msg"
}

# Pull single image with retry mechanism
# Args: $1 = image name, $2 = max retries (default 3), $3 = retry delay (default 5)
pull_image_with_retry() {
    local image="$1"
    local max_retries="${2:-$DEFAULT_MAX_RETRIES}"
    local retry_delay="${3:-$DEFAULT_RETRY_DELAY}"
    local retry_count=0
    local last_error=""
    
    # Check if image already exists locally
    if docker image inspect "$image" >/dev/null 2>&1; then
        log_info "  ✓ Image exists: $image"
        return 0
    fi
    
    while [[ $retry_count -lt $max_retries ]]; do
        retry_count=$((retry_count + 1))
        
        if [[ $retry_count -gt 1 ]]; then
            log_warn "  🔄 Retry $retry_count/$max_retries: $image (waiting ${retry_delay}s...)"
            sleep $retry_delay
        else
            log_info "  ⬇ Pulling: $image"
        fi
        
        # Capture both stdout and stderr
        local output
        if output=$(docker pull "$image" 2>&1); then
            log_info "  ✓ Pulled: $image"
            return 0
        else
            last_error="$output"
            log_warn "  ⚠ Attempt $retry_count failed: $(echo "$last_error" | head -1)"
        fi
    done
    
    # All retries exhausted - log failure
    log_failure "PULL" "$image" "Failed after $max_retries attempts. Last error: $(echo "$last_error" | head -1)"
    return 1
}

# Extract base images from Dockerfile
# Args: $1 = Dockerfile path
extract_base_images() {
    local dockerfile="$1"
    
    if [[ ! -f "$dockerfile" ]]; then
        return 1
    fi
    
    # Extract FROM statements
    # Pattern: FROM image:tag [AS alias]
    # Skip: ARG variables (${...}), local build stages, empty images
    grep -E "^FROM\s+" "$dockerfile" 2>/dev/null | \
        awk '{
            img=$2
            # Skip ARG variables (contains ${...})
            if (img ~ /\$\{/) next
            # Skip platform flags
            if (img ~ /^--/) next
            # Skip if no colon and no slash (likely a build stage alias like "builder")
            if (img !~ /[:\/]/) next
            print img
        }' | \
        sort -u
}

# Prefetch base images from Dockerfiles
# Args: $1 = service name (optional, if empty prefetch all)
prefetch_base_images() {
    local service_name="$1"
    local max_retries="${2:-3}"
    
    log_info "📦 Prefetching base images..."
    
    local dockerfiles=()
    
    if [[ -n "$service_name" ]]; then
        local dockerfile="$SRC_DIR/$service_name/Dockerfile"
        if [[ -f "$dockerfile" ]]; then
            dockerfiles+=("$dockerfile")
        fi
    else
        # Find all Dockerfiles
        while IFS= read -r df; do
            dockerfiles+=("$df")
        done < <(find "$SRC_DIR" -name "Dockerfile" -type f 2>/dev/null)
    fi
    
    local all_images=()
    local pull_count=0
    local skip_count=0
    local fail_count=0
    
    # Extract all base images
    for dockerfile in "${dockerfiles[@]}"; do
        local images
        images=$(extract_base_images "$dockerfile")
        while IFS= read -r img; do
            [[ -z "$img" ]] && continue
            [[ "$img" =~ ^[a-z_-]+$ ]] && continue  # Skip internal build stages
            all_images+=("$img")
        done <<< "$images"
    done
    
    # Remove duplicates
    local unique_images=($(printf '%s\n' "${all_images[@]}" | sort -u))
    
    log_info "Found ${#unique_images[@]} unique base images to check"
    
    for image in "${unique_images[@]}"; do
        if docker image inspect "$image" >/dev/null 2>&1; then
            log_info "  ✓ Exists: $image"
            ((skip_count++))
        else
            log_info "  ⬇ Pulling: $image"
            if pull_image_with_retry "$image" "$max_retries"; then
                ((pull_count++))
            else
                ((fail_count++))
            fi
        fi
    done
    
    log_info "📊 Prefetch summary: pulled=$pull_count, skipped=$skip_count, failed=$fail_count"
    return 0
}

# Pull all project images from registry
# Args: $1 = registry, $2 = tag
pull_all_services() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required for pull-all"
        log_info "Usage: $0 pull-all <registry> [tag]"
        return 1
    fi
    
    log_info "=========================================="
    log_info "Pulling all services from registry"
    log_info "=========================================="
    log_info "Registry: $registry"
    log_info "Tag: $tag"
    log_info "Max retries: $max_retries"
    echo
    
    discover_services
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # Pull all services
    for service in "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; do
        total_count=$((total_count + 1))
        local image_name="ai-infra-${service}:${tag}"
        local remote_image="$registry/$image_name"
        
        log_info "Pulling: $remote_image"
        
        if pull_image_with_retry "$remote_image" "$max_retries"; then
            # Tag as local image
            if docker tag "$remote_image" "$image_name"; then
                log_info "  ✓ Tagged: $image_name"
                success_count=$((success_count + 1))
            else
                log_failure "TAG" "$image_name" "Failed to tag from $remote_image"
                failed_services+=("$service")
            fi
        else
            failed_services+=("$service")
        fi
    done
    
    echo
    log_info "=========================================="
    log_info "Pull completed: $success_count/$total_count successful"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        log_warn "Failed services: ${failed_services[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🎉 All services pulled successfully!"
    return 0
}

# ==============================================================================
# Push Functions - 镜像推送功能
# ==============================================================================

# Push single image with retry mechanism
# Args: $1 = image, $2 = max retries (default 3), $3 = retry delay (default 5)
push_image_with_retry() {
    local image="$1"
    local max_retries="${2:-$DEFAULT_MAX_RETRIES}"
    local retry_delay="${3:-$DEFAULT_RETRY_DELAY}"
    local retry_count=0
    local last_error=""
    
    while [[ $retry_count -lt $max_retries ]]; do
        retry_count=$((retry_count + 1))
        
        if [[ $retry_count -gt 1 ]]; then
            log_warn "  🔄 Retry $retry_count/$max_retries: $image (waiting ${retry_delay}s...)"
            sleep $retry_delay
        fi
        
        # Capture both stdout and stderr
        local output
        if output=$(docker push "$image" 2>&1); then
            log_info "  ✓ Pushed: $image"
            return 0
        else
            last_error="$output"
            log_warn "  ⚠ Attempt $retry_count failed: $(echo "$last_error" | head -1)"
        fi
    done
    
    # All retries exhausted - log failure
    log_failure "PUSH" "$image" "Failed after $max_retries attempts. Last error: $(echo "$last_error" | head -1)"
    return 1
}

# Push single service image
# Args: $1 = service, $2 = tag, $3 = registry
push_service() {
    local service="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local registry="$3"
    local max_retries="${4:-$DEFAULT_MAX_RETRIES}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required for push"
        return 1
    fi
    
    local base_image="ai-infra-${service}:${tag}"
    local target_image="$registry/ai-infra-${service}:${tag}"
    
    log_info "Pushing service: $service"
    log_info "  Source: $base_image"
    log_info "  Target: $target_image"
    
    # Check if source image exists
    if ! docker image inspect "$base_image" >/dev/null 2>&1; then
        log_warn "Local image not found: $base_image"
        log_info "Building image first..."
        if ! build_component "$service"; then
            log_failure "BUILD" "$base_image" "Build failed before push"
            return 1
        fi
    fi
    
    # Tag for registry with retry
    if [[ "$base_image" != "$target_image" ]]; then
        log_info "  Tagging: $base_image -> $target_image"
        if ! docker tag "$base_image" "$target_image"; then
            log_failure "TAG" "$target_image" "Failed to tag image"
            return 1
        fi
    fi
    
    # Push to registry with retry
    log_info "  Pushing: $target_image"
    if push_image_with_retry "$target_image" "$max_retries"; then
        return 0
    else
        return 1
    fi
}

# Push all service images
# Args: $1 = registry, $2 = tag
push_all_services() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required for push-all"
        log_info "Usage: $0 push-all <registry> [tag]"
        return 1
    fi
    
    log_info "=========================================="
    log_info "Pushing all AI-Infra services"
    log_info "=========================================="
    log_info "Registry: $registry"
    log_info "Tag: $tag"
    echo
    
    discover_services
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # Push all services
    for service in "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; do
        total_count=$((total_count + 1))
        
        if push_service "$service" "$tag" "$registry"; then
            success_count=$((success_count + 1))
        else
            failed_services+=("$service")
        fi
        echo
    done
    
    log_info "=========================================="
    log_info "Push completed: $success_count/$total_count successful"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        log_warn "Failed services: ${failed_services[*]}"
        return 1
    fi
    
    log_info "🚀 All services pushed successfully!"
    return 0
}

# Get dependency image mappings
get_dependency_mappings() {
    local mappings=(
        "confluentinc/cp-kafka:${KAFKA_VERSION:-7.5.0}|cp-kafka"
        "provectuslabs/kafka-ui:${KAFKAUI_VERSION:-latest}|kafka-ui"
        "postgres:${POSTGRES_VERSION:-15-alpine}|postgres"
        "redis:${REDIS_VERSION:-7-alpine}|redis"
        "minio/minio:${MINIO_VERSION:-latest}|minio"
        "osixia/openldap:${OPENLDAP_VERSION:-stable}|openldap"
        "osixia/phpldapadmin:${PHPLDAPADMIN_VERSION:-stable}|phpldapadmin"
        "mysql:${MYSQL_VERSION:-8.0}|mysql"
    )
    echo "${mappings[@]}"
}

# Push all dependency images
# Args: $1 = registry, $2 = tag
push_all_dependencies() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required for push-dep"
        log_info "Usage: $0 push-dep <registry> [tag]"
        return 1
    fi
    
    # Ensure registry ends without trailing slash for consistent handling
    registry="${registry%/}"
    
    log_info "=========================================="
    log_info "Pushing all dependency images"
    log_info "=========================================="
    log_info "Registry: $registry"
    log_info "Tag: $tag"
    log_info "Max retries: $max_retries"
    echo
    
    local dependencies=($(get_dependency_mappings))
    local success_count=0
    local total_count=${#dependencies[@]}
    local failed_images=()
    
    for mapping in "${dependencies[@]}"; do
        local source_image="${mapping%%|*}"
        local short_name="${mapping##*|}"
        local target_image="${registry}/${short_name}:${tag}"
        
        log_info "Processing: $source_image"
        log_info "  → Target: $target_image"
        
        # 1. Pull or check source image (with retry)
        log_info "  [1/3] Checking source image..."
        if docker image inspect "$source_image" >/dev/null 2>&1; then
            log_info "  ✓ Image exists locally"
        else
            if ! pull_image_with_retry "$source_image" "$max_retries"; then
                failed_images+=("$source_image")
                echo
                continue
            fi
        fi
        
        # 2. Tag for registry
        log_info "  [2/3] Tagging image..."
        if ! docker tag "$source_image" "$target_image"; then
            log_failure "TAG" "$target_image" "Failed to tag from $source_image"
            failed_images+=("$source_image")
            echo
            continue
        fi
        log_info "  ✓ Tagged"
        
        # 3. Push to registry (with retry)
        log_info "  [3/3] Pushing image..."
        if push_image_with_retry "$target_image" "$max_retries"; then
            success_count=$((success_count + 1))
        else
            failed_images+=("$source_image")
        fi
        echo
    done
    
    log_info "=========================================="
    log_info "Dependency push completed: $success_count/$total_count successful"
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        log_warn "Failed images: ${failed_images[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🚀 All dependency images pushed successfully!"
    return 0
}

# Pull and tag dependencies from registry
# Args: $1 = registry, $2 = tag
pull_and_tag_dependencies() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required"
        log_info "Usage: $0 deps-pull <registry> [tag]"
        return 1
    fi
    
    registry="${registry%/}"
    
    log_info "=========================================="
    log_info "Pulling dependencies from: $registry"
    log_info "=========================================="
    log_info "Tag: $tag"
    log_info "Max retries: $max_retries"
    echo
    
    local dependencies=($(get_dependency_mappings))
    local success_count=0
    local total_count=${#dependencies[@]}
    local failed_deps=()
    
    for mapping in "${dependencies[@]}"; do
        local source_image="${mapping%%|*}"
        local short_name="${mapping##*|}"
        local remote_image="${registry}/${short_name}:${tag}"
        
        log_info "Pulling: $remote_image"
        
        if pull_image_with_retry "$remote_image" "$max_retries"; then
            # Tag as original image name
            if docker tag "$remote_image" "$source_image"; then
                log_info "  ✓ Tagged: $source_image"
                success_count=$((success_count + 1))
            else
                log_failure "TAG" "$source_image" "Failed to tag from $remote_image"
                failed_deps+=("$short_name")
            fi
        else
            failed_deps+=("$short_name")
        fi
    done
    
    echo
    log_info "=========================================="
    log_info "Dependencies pull completed: $success_count/$total_count"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        log_warn "Failed: ${failed_deps[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🎉 All dependencies pulled successfully!"
    return 0
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
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Build Commands:"
    echo "  build-all, all      Build all components in the correct order (AppHub first)"
    echo "  [component]         Build a specific component (e.g., backend, frontend)"
    echo ""
    echo "Service Commands:"
    echo "  start-all           Start all services using docker-compose"
    echo ""
    echo "Pull Commands:"
    echo "  prefetch            Prefetch all base images from Dockerfiles"
    echo "  pull-all <registry> [tag]   Pull all service images from registry"
    echo "  deps-pull <registry> [tag]  Pull dependency images from registry"
    echo ""
    echo "Push Commands:"
    echo "  push <service> <registry> [tag]  Push single service to registry"
    echo "  push-all <registry> [tag]        Push all services to registry"
    echo "  push-dep <registry> [tag]        Push dependency images to registry"
    echo ""
    echo "Examples:"
    echo "  $0 build-all"
    echo "  $0 start-all"
    echo "  $0 backend"
    echo "  $0 prefetch"
    echo "  $0 push-all harbor.example.com/ai-infra v0.3.8"
    echo "  $0 push-dep harbor.example.com/ai-infra v0.3.8"
    echo "  $0 pull-all harbor.example.com/ai-infra v0.3.8"
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
    prefetch)
        prefetch_base_images "$2"
        ;;
    pull-all)
        if [[ -z "$2" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 pull-all <registry> [tag]"
            exit 1
        fi
        pull_all_services "$2" "${3:-${IMAGE_TAG:-latest}}"
        ;;
    deps-pull)
        if [[ -z "$2" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 deps-pull <registry> [tag]"
            exit 1
        fi
        pull_and_tag_dependencies "$2" "${3:-${IMAGE_TAG:-latest}}"
        ;;
    push)
        if [[ -z "$2" ]]; then
            log_error "Service name is required"
            log_info "Usage: $0 push <service> <registry> [tag]"
            exit 1
        fi
        if [[ -z "$3" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 push <service> <registry> [tag]"
            exit 1
        fi
        push_service "$2" "${4:-${IMAGE_TAG:-latest}}" "$3"
        ;;
    push-all)
        if [[ -z "$2" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 push-all <registry> [tag]"
            exit 1
        fi
        push_all_services "$2" "${3:-${IMAGE_TAG:-latest}}"
        ;;
    push-dep|push-dependencies)
        if [[ -z "$2" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 push-dep <registry> [tag]"
            exit 1
        fi
        push_all_dependencies "$2" "${3:-${IMAGE_TAG:-latest}}"
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
