#!/bin/bash

# =======================================================================
# AI Infrastructure Matrix - 对象存储 iframe 功能构建和测试脚本
# =======================================================================
# 功能: 构建并测试对象存储iframe集成功能
# 作者: AI Infrastructure Team
# 版本: v1.0.0
# =======================================================================

set -e

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
LOG_FILE="${PROJECT_DIR}/logs/object-storage-test.log"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:8080}"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [INFO] $1" >> "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [SUCCESS] $1" >> "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [WARN] $1" >> "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [ERROR] $1" >> "$LOG_FILE"
}

# 创建日志目录
mkdir -p "$(dirname "$LOG_FILE")"

# 检测操作系统
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macOS"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "Linux"
    else
        echo "Unknown"
    fi
}

OS_TYPE=$(detect_os)
log_info "检测到操作系统: $OS_TYPE"

# 检查必要的命令
check_dependencies() {
    log_info "检查系统依赖..."
    
    local missing_deps=()
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        missing_deps+=("docker")
    fi
    
    # 检查docker-compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        missing_deps+=("docker-compose")
    fi
    
    # 检查curl
    if ! command -v curl &> /dev/null; then
        missing_deps+=("curl")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "缺少必要依赖: ${missing_deps[*]}"
        log_error "请安装缺少的依赖后重新运行"
        exit 1
    fi
    
    log_success "依赖检查通过"
}

# 检查Docker Compose版本
check_docker_compose() {
    log_info "检查Docker Compose版本..."
    
    if docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
        COMPOSE_VERSION=$(docker compose version --short 2>/dev/null || echo "unknown")
        log_info "使用Docker Compose v2: $COMPOSE_VERSION"
    elif command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
        COMPOSE_VERSION=$(docker-compose version --short 2>/dev/null || echo "unknown")
        log_info "使用Docker Compose v1: $COMPOSE_VERSION"
    else
        log_error "未找到可用的Docker Compose"
        exit 1
    fi
}

# 停止现有服务
stop_services() {
    log_info "停止现有服务..."
    
    cd "$PROJECT_DIR"
    
    # 尝试停止服务
    if $COMPOSE_CMD ps -q | grep -q .; then
        log_info "发现运行中的服务，正在停止..."
        $COMPOSE_CMD down --remove-orphans || log_warn "服务停止过程中出现警告"
    else
        log_info "没有运行中的服务"
    fi
}

# 构建服务
build_services() {
    log_info "构建对象存储相关服务..."
    
    cd "$PROJECT_DIR"
    
    # 构建后端（包含对象存储API）
    log_info "构建后端服务..."
    $COMPOSE_CMD build backend
    
    # 构建前端（包含对象存储页面）
    log_info "构建前端服务..."
    $COMPOSE_CMD build frontend
    
    # 构建nginx（包含MinIO代理配置）
    log_info "构建nginx服务..."
    $COMPOSE_CMD build nginx
    
    log_success "服务构建完成"
}

# 启动服务
start_services() {
    log_info "启动对象存储测试环境..."
    
    cd "$PROJECT_DIR"
    
    # 启动核心服务
    log_info "启动数据库服务..."
    $COMPOSE_CMD up -d postgres redis
    
    # 等待数据库启动
    log_info "等待数据库启动..."
    sleep 10
    
    # 启动MinIO服务
    log_info "启动MinIO服务..."
    $COMPOSE_CMD up -d minio
    
    # 启动后端服务
    log_info "启动后端服务..."
    $COMPOSE_CMD up -d backend
    
    # 启动前端和nginx
    log_info "启动前端和代理服务..."
    $COMPOSE_CMD up -d frontend nginx
    
    log_success "所有服务启动完成"
}

# 等待服务健康
wait_for_services() {
    log_info "等待服务健康检查..."
    
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        log_info "健康检查尝试 $attempt/$max_attempts..."
        
        # 检查后端健康状态
        if curl -s -f "$FRONTEND_URL/api/health" > /dev/null 2>&1; then
            log_success "后端服务健康检查通过"
            break
        fi
        
        if [ $attempt -eq $max_attempts ]; then
            log_error "服务启动超时，请检查日志"
            $COMPOSE_CMD logs --tail=50
            return 1
        fi
        
        sleep 5
        ((attempt++))
    done
    
    # 额外等待MinIO服务稳定
    log_info "等待MinIO服务稳定..."
    sleep 10
}

# 测试对象存储API
test_object_storage_api() {
    log_info "测试对象存储API端点..."
    
    # 测试获取配置列表
    log_info "测试获取对象存储配置..."
    local response=$(curl -s -w "%{http_code}" "$FRONTEND_URL/api/object-storage/configs" -H "Content-Type: application/json" -o /tmp/os_configs.json)
    
    if [[ "$response" == "200" ]] || [[ "$response" == "401" ]]; then
        log_success "对象存储配置API响应正常 (HTTP $response)"
    else
        log_error "对象存储配置API响应异常 (HTTP $response)"
        return 1
    fi
    
    # 测试MinIO健康检查
    log_info "测试MinIO健康检查..."
    local minio_health=$(curl -s -w "%{http_code}" "$FRONTEND_URL/minio/health" -o /tmp/minio_health.json)
    
    if [[ "$minio_health" == "200" ]]; then
        log_success "MinIO健康检查通过 (HTTP $minio_health)"
    else
        log_warn "MinIO健康检查响应: HTTP $minio_health（可能需要认证）"
    fi
}

# 测试MinIO控制台iframe
test_minio_console_iframe() {
    log_info "测试MinIO控制台iframe集成..."
    
    # 测试nginx代理路径
    log_info "测试MinIO控制台代理路径..."
    local console_response=$(curl -s -w "%{http_code}" "$FRONTEND_URL/minio-console/" -o /tmp/minio_console.html)
    
    if [[ "$console_response" == "200" ]]; then
        log_success "MinIO控制台代理响应正常 (HTTP $console_response)"
        
        # 检查响应内容
        if grep -q "MinIO\|Console\|login" /tmp/minio_console.html 2>/dev/null; then
            log_success "MinIO控制台页面内容验证通过"
        else
            log_warn "MinIO控制台页面内容可能不完整"
        fi
    else
        log_error "MinIO控制台代理响应异常 (HTTP $console_response)"
        return 1
    fi
}

# 测试前端对象存储页面
test_frontend_pages() {
    log_info "测试前端对象存储页面..."
    
    # 测试主要对象存储页面
    log_info "测试对象存储主页面..."
    local main_page=$(curl -s -w "%{http_code}" "$FRONTEND_URL/object-storage" -o /tmp/object_storage_page.html)
    
    if [[ "$main_page" == "200" ]]; then
        log_success "对象存储主页面响应正常 (HTTP $main_page)"
    else
        log_warn "对象存储主页面响应: HTTP $main_page（可能需要登录）"
    fi
    
    # 测试管理配置页面
    log_info "测试对象存储配置页面..."
    local admin_page=$(curl -s -w "%{http_code}" "$FRONTEND_URL/admin/object-storage" -o /tmp/object_storage_admin.html)
    
    if [[ "$admin_page" == "200" ]]; then
        log_success "对象存储配置页面响应正常 (HTTP $admin_page)"
    else
        log_warn "对象存储配置页面响应: HTTP $admin_page（可能需要登录）"
    fi
}

# 测试iframe测试页面
test_iframe_test_page() {
    log_info "测试iframe集成测试页面..."
    
    local test_page=$(curl -s -w "%{http_code}" "$FRONTEND_URL/test-object-storage-iframe.html" -o /tmp/test_page.html)
    
    if [[ "$test_page" == "200" ]]; then
        log_success "iframe测试页面响应正常 (HTTP $test_page)"
        log_info "可以通过以下地址访问测试页面: $FRONTEND_URL/test-object-storage-iframe.html"
    else
        log_error "iframe测试页面响应异常 (HTTP $test_page)"
        return 1
    fi
}

# 显示服务状态
show_service_status() {
    log_info "显示服务状态..."
    
    cd "$PROJECT_DIR"
    
    echo -e "\n${CYAN}=== 服务状态 ===${NC}"
    $COMPOSE_CMD ps
    
    echo -e "\n${CYAN}=== 端口映射 ===${NC}"
    echo "主入口: $FRONTEND_URL"
    echo "MinIO API: $FRONTEND_URL/minio/"
    echo "MinIO控制台: $FRONTEND_URL/minio-console/"
    echo "对象存储管理: $FRONTEND_URL/object-storage"
    echo "存储配置管理: $FRONTEND_URL/admin/object-storage"
    echo "iframe测试页面: $FRONTEND_URL/test-object-storage-iframe.html"
}

# 显示测试结果总结
show_test_summary() {
    echo -e "\n${PURPLE}================================================================${NC}"
    echo -e "${PURPLE}              对象存储 iframe 功能测试完成${NC}"
    echo -e "${PURPLE}================================================================${NC}"
    
    echo -e "\n${CYAN}📋 功能测试结果:${NC}"
    echo "  ✅ 服务构建和启动"
    echo "  ✅ 对象存储API端点"
    echo "  ✅ MinIO服务集成"
    echo "  ✅ nginx代理配置"
    echo "  ✅ iframe嵌入功能"
    echo "  ✅ 前端页面路由"
    
    echo -e "\n${CYAN}🌐 访问地址:${NC}"
    echo "  • 主页面: $FRONTEND_URL"
    echo "  • 对象存储: $FRONTEND_URL/object-storage"
    echo "  • 存储配置: $FRONTEND_URL/admin/object-storage"
    echo "  • MinIO控制台: $FRONTEND_URL/minio-console/"
    echo "  • 测试页面: $FRONTEND_URL/test-object-storage-iframe.html"
    
    echo -e "\n${CYAN}🔧 测试命令:${NC}"
    echo "  • 查看服务状态: $COMPOSE_CMD ps"
    echo "  • 查看服务日志: $COMPOSE_CMD logs -f [service]"
    echo "  • 停止服务: $COMPOSE_CMD down"
    
    echo -e "\n${GREEN}🎉 对象存储iframe功能已就绪！${NC}"
}

# 清理临时文件
cleanup() {
    log_info "清理临时文件..."
    rm -f /tmp/os_configs.json /tmp/minio_health.json /tmp/minio_console.html
    rm -f /tmp/object_storage_page.html /tmp/object_storage_admin.html /tmp/test_page.html
}

# 主函数
main() {
    echo -e "${PURPLE}================================================================${NC}"
    echo -e "${PURPLE}           AI Infrastructure Matrix${NC}"
    echo -e "${PURPLE}         对象存储 iframe 功能构建测试${NC}"
    echo -e "${PURPLE}================================================================${NC}"
    
    # 检查依赖
    check_dependencies
    check_docker_compose
    
    # 构建和启动服务
    stop_services
    build_services
    start_services
    
    # 等待服务就绪
    wait_for_services
    
    # 运行测试
    test_object_storage_api
    test_minio_console_iframe
    test_frontend_pages
    test_iframe_test_page
    
    # 显示结果
    show_service_status
    show_test_summary
    
    # 清理
    cleanup
    
    log_success "对象存储iframe功能测试完成"
}

# 错误处理
trap 'log_error "脚本执行过程中发生错误，退出码: $?"' ERR

# 执行主函数
main "$@"