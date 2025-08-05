#!/bin/bash

# Ansible Playbook Generator Web-v2 构建和测试脚本
# 此脚本用于构建、运行和测试整个应用程序

set -e  # 遇到错误时立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT=$(dirname "$(realpath "$0")")
cd "$PROJECT_ROOT"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查必要的工具
check_dependencies() {
    log_info "检查依赖工具..."
    
    local tools=("docker" "docker-compose" "curl")
    for tool in "${tools[@]}"; do
        if ! command -v $tool &> /dev/null; then
            log_error "$tool 未安装，请先安装"
            exit 1
        fi
    done
    
    log_success "所有依赖工具已安装"
}

# 清理旧容器和镜像
cleanup() {
    log_info "清理旧容器和镜像..."
    
    # 停止并删除所有相关容器
    docker-compose down --remove-orphans 2>/dev/null || true
    
    # 删除悬空镜像
    docker image prune -f 2>/dev/null || true
    
    log_success "清理完成"
}

# 构建所有服务
build_services() {
    log_info "构建所有服务..."
    
    # 构建后端
    log_info "构建后端服务..."
    docker-compose build backend
    
    # 构建前端
    log_info "构建前端服务..."
    docker-compose build frontend
    
    log_success "所有服务构建完成"
}

# 启动所有服务
start_services() {
    log_info "启动所有服务..."
    
    # 启动依赖服务
    log_info "启动数据库和缓存服务..."
    docker-compose up -d postgres redis openldap phpldapadmin
    
    # 等待数据库就绪
    log_info "等待数据库服务就绪..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if docker-compose exec -T postgres pg_isready -U postgres -d ansible_playbook_generator >/dev/null 2>&1; then
            log_success "PostgreSQL 数据库已就绪"
            break
        fi
        
        if [ $attempt -eq $max_attempts ]; then
            log_error "数据库启动超时"
            exit 1
        fi
        
        log_info "等待数据库启动... (尝试 $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    # 启动应用服务
    log_info "启动应用服务..."
    docker-compose up -d backend frontend
    
    log_success "所有服务启动完成"
}

# 等待服务就绪
wait_for_services() {
    log_info "等待服务就绪..."
    
    # 等待后端服务
    local backend_url="http://localhost:8082/api/health"
    local frontend_url="http://localhost:3001"
    local max_attempts=60
    
    log_info "等待后端服务 ($backend_url)..."
    local attempt=1
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$backend_url" >/dev/null 2>&1; then
            log_success "后端服务已就绪"
            break
        fi
        
        if [ $attempt -eq $max_attempts ]; then
            log_error "后端服务启动超时"
            return 1
        fi
        
        log_info "等待后端服务启动... (尝试 $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    # 等待前端服务
    log_info "等待前端服务 ($frontend_url)..."
    attempt=1
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$frontend_url" >/dev/null 2>&1; then
            log_success "前端服务已就绪"
            break
        fi
        
        if [ $attempt -eq $max_attempts ]; then
            log_error "前端服务启动超时"
            return 1
        fi
        
        log_info "等待前端服务启动... (尝试 $attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    log_success "所有服务已就绪"
}

# 运行健康检查
health_check() {
    log_info "运行健康检查..."
    
    # 检查容器状态
    log_info "检查容器状态..."
    if ! docker-compose ps | grep -q "Up"; then
        log_error "部分容器未正常运行"
        docker-compose ps
        return 1
    fi
    
    # 检查后端API
    log_info "检查后端API..."
    local response=$(curl -s -w "%{http_code}" -o /dev/null "http://localhost:8082/api/health")
    if [ "$response" != "200" ]; then
        log_error "后端健康检查失败 (HTTP $response)"
        return 1
    fi
    
    # 检查前端页面
    log_info "检查前端页面..."
    local response=$(curl -s -w "%{http_code}" -o /dev/null "http://localhost:3001")
    if [ "$response" != "200" ]; then
        log_error "前端页面检查失败 (HTTP $response)"
        return 1
    fi
    
    # 检查数据库连接
    log_info "检查数据库连接..."
    if ! docker-compose exec -T postgres psql -U postgres -d ansible_playbook_generator -c "SELECT 1;" >/dev/null 2>&1; then
        log_error "数据库连接失败"
        return 1
    fi
    
    # 检查Redis连接
    log_info "检查Redis连接..."
    if ! docker-compose exec -T redis redis-cli -a ansible-redis-password ping >/dev/null 2>&1; then
        log_error "Redis连接失败"
        return 1
    fi
    
    log_success "所有健康检查通过"
}

# 运行功能测试
run_functional_tests() {
    log_info "运行功能测试..."
    
    # 测试登录功能
    log_info "测试用户登录..."
    local login_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' \
        -w "%{http_code}" \
        -o /tmp/login_response.json \
        "http://localhost:8082/api/auth/login")
    
    if [ "$login_response" = "200" ]; then
        log_success "登录测试通过"
        
        # 提取token用于后续测试
        local token=$(cat /tmp/login_response.json | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$token" ]; then
            log_success "获取到认证token"
            
            # 测试受保护的API
            log_info "测试受保护的API..."
            local api_response=$(curl -s -w "%{http_code}" \
                -H "Authorization: Bearer $token" \
                -o /dev/null \
                "http://localhost:8082/api/auth/profile")
            
            if [ "$api_response" = "200" ]; then
                log_success "受保护API测试通过"
            else
                log_warning "受保护API测试失败 (HTTP $api_response)"
            fi
        fi
    else
        log_warning "登录测试失败 (HTTP $login_response)"
    fi
    
    # 测试管理中心导航功能
    log_info "测试前端管理中心导航..."
    local admin_page_response=$(curl -s -w "%{http_code}" -o /dev/null "http://localhost:3001/admin")
    if [ "$admin_page_response" = "200" ]; then
        log_success "管理中心页面访问正常"
    else
        log_warning "管理中心页面访问异常 (HTTP $admin_page_response)"
    fi
    
    # 清理临时文件
    rm -f /tmp/login_response.json
    
    log_success "功能测试完成"
}

# 显示服务信息
show_service_info() {
    log_info "服务信息:"
    echo ""
    echo "🌐 前端页面:"
    echo "   URL: http://localhost:3001"
    echo "   管理中心: http://localhost:3001/admin"
    echo ""
    echo "🔧 后端API:"
    echo "   URL: http://localhost:8082"
    echo "   健康检查: http://localhost:8082/health"
    echo "   API文档: http://localhost:8082/swagger/index.html"
    echo ""
    echo "🗄️ 数据库管理:"
    echo "   PostgreSQL: localhost:5433"
    echo "   用户名: postgres"
    echo "   密码: postgres"
    echo "   数据库: ansible_playbook_generator"
    echo ""
    echo "📊 LDAP管理:"
    echo "   phpLDAPadmin: http://localhost:8081"
    echo "   LDAP服务器: localhost:389"
    echo ""
    echo "📈 Redis:"
    echo "   地址: localhost:6379"
    echo "   密码: ansible-redis-password"
    echo ""
    echo "📋 默认登录信息:"
    echo "   用户名: admin"
    echo "   密码: admin123"
    echo ""
}

# 显示日志
show_logs() {
    local service=${1:-""}
    
    if [ -n "$service" ]; then
        log_info "显示 $service 服务日志..."
        docker-compose logs -f "$service"
    else
        log_info "显示所有服务日志..."
        docker-compose logs -f
    fi
}

# 停止服务
stop_services() {
    log_info "停止所有服务..."
    docker-compose down
    log_success "所有服务已停止"
}

# 完全清理
full_cleanup() {
    log_info "执行完全清理..."
    docker-compose down -v --remove-orphans
    docker system prune -f
    log_success "完全清理完成"
}

# 测试管理中心功能
test_admin_center() {
    log_info "开始测试管理中心导航功能..."
    
    local base_url="http://localhost:3001"
    local backend_url="http://localhost:8082"
    
    # 1. 检查前端服务是否可用
    log_info "检查前端服务状态..."
    local frontend_status=$(curl -s -w "%{http_code}" -o /dev/null "$base_url")
    if [ "$frontend_status" != "200" ]; then
        log_error "前端服务不可用 (HTTP $frontend_status)"
        log_info "请先运行: $0 start"
        exit 1
    fi
    log_success "前端服务正常"
    
    # 2. 检查后端服务是否可用
    log_info "检查后端服务状态..."
    local backend_status=$(curl -s -w "%{http_code}" -o /dev/null "$backend_url/api/health")
    if [ "$backend_status" != "200" ]; then
        log_error "后端服务不可用 (HTTP $backend_status)"
        log_info "请先运行: $0 start"
        exit 1
    fi
    log_success "后端服务正常"
    
    # 3. 测试管理中心主页面
    log_info "测试管理中心主页面访问..."
    local admin_status=$(curl -s -w "%{http_code}" -o /dev/null "$base_url/admin")
    if [ "$admin_status" = "200" ]; then
        log_success "管理中心主页面访问正常"
    else
        log_warning "管理中心主页面访问异常 (HTTP $admin_status)"
    fi
    
    # 4. 测试各个管理子页面
    local admin_pages=(
        "/admin/users"
        "/admin/roles" 
        "/admin/permissions"
        "/admin/system"
        "/admin/logs"
    )
    
    log_info "测试管理中心子页面..."
    for page in "${admin_pages[@]}"; do
        local page_status=$(curl -s -w "%{http_code}" -o /dev/null "$base_url$page")
        if [ "$page_status" = "200" ]; then
            log_success "页面 $page 访问正常"
        else
            log_warning "页面 $page 访问异常 (HTTP $page_status)"
        fi
    done
    
    # 5. 模拟用户登录并测试受保护的管理功能
    log_info "测试管理员登录和权限..."
    local login_response=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}' \
        -w "%{http_code}" \
        -o /tmp/admin_login.json \
        "$backend_url/api/auth/login")
    
    if [ "$login_response" = "200" ]; then
        log_success "管理员登录成功"
        
        # 提取token
        local token=$(cat /tmp/admin_login.json | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$token" ]; then
            log_success "获取到管理员认证token"
            
            # 测试管理API
            local admin_apis=(
                "/api/admin/users"
                "/api/admin/system/info"
                "/api/auth/profile"
            )
            
            for api in "${admin_apis[@]}"; do
                local api_status=$(curl -s -w "%{http_code}" \
                    -H "Authorization: Bearer $token" \
                    -o /dev/null \
                    "$backend_url$api")
                
                if [ "$api_status" = "200" ]; then
                    log_success "管理API $api 访问正常"
                else
                    log_warning "管理API $api 访问异常 (HTTP $api_status)"
                fi
            done
        fi
    else
        log_warning "管理员登录失败 (HTTP $login_response)"
    fi
    
    # 清理临时文件
    rm -f /tmp/admin_login.json
    
    log_success "管理中心功能测试完成"
}

# 显示前端组件测试说明
show_admin_test_instructions() {
    log_info "前端管理中心导航测试说明:"
    echo ""
    echo "🖱️ 手动测试步骤:"
    echo "1. 在浏览器中打开: http://localhost:3001"
    echo "2. 使用管理员账号登录:"
    echo "   - 用户名: admin"
    echo "   - 密码: admin123"
    echo "3. 登录后，观察顶部导航栏的\"管理中心\"按钮"
    echo ""
    echo "🎯 测试要点:"
    echo "✅ 点击\"管理中心\"按钮应该导航到 /admin 页面"
    echo "✅ 鼠标悬停在\"管理中心\"按钮上应该显示下拉菜单"
    echo "✅ 下拉菜单应该包含以下选项:"
    echo "   - 用户管理"
    echo "   - 角色管理" 
    echo "   - 权限管理"
    echo "   - 系统设置"
    echo "   - 系统日志"
    echo "✅ 点击下拉菜单中的任意选项应该导航到对应页面"
    echo "✅ 当前在管理页面时，\"管理中心\"按钮应该显示激活状态"
    echo ""
    echo "🎨 视觉验证:"
    echo "- 管理中心按钮应该有设置图标"
    echo "- 按钮应该有下拉箭头图标"
    echo "- 在管理页面时按钮背景应该为蓝色 (#1890ff)"
    echo "- 下拉菜单样式应该与主题一致"
    echo ""
}

# 启动浏览器测试管理中心
open_admin_browser_test() {
    log_info "启动浏览器进行管理中心手动测试..."
    
    local base_url="http://localhost:3001"
    
    # 检查服务是否运行
    if ! curl -s "$base_url" >/dev/null 2>&1; then
        log_error "前端服务未启动，请先运行: $0 start"
        exit 1
    fi
    
    # 在macOS上打开浏览器
    if command -v open >/dev/null 2>&1; then
        log_info "在默认浏览器中打开应用..."
        open "$base_url"
        sleep 2
        open "$base_url/admin"
    else
        log_info "请手动在浏览器中打开: $base_url"
    fi
    
    show_admin_test_instructions
}

# 主函数
main() {
    local command=${1:-"help"}
    
    case $command in
        "build")
            check_dependencies
            cleanup
            build_services
            ;;
        "start")
            check_dependencies
            start_services
            wait_for_services
            show_service_info
            ;;
        "test")
            check_dependencies
            health_check
            run_functional_tests
            ;;
        "full")
            check_dependencies
            cleanup
            build_services
            start_services
            wait_for_services
            health_check
            run_functional_tests
            show_service_info
            ;;
        "stop")
            stop_services
            ;;
        "logs")
            show_logs "${2:-}"
            ;;
        "clean")
            full_cleanup
            ;;
        "info")
            show_service_info
            ;;
        "restart")
            stop_services
            start_services
            wait_for_services
            show_service_info
            ;;
        "admin-test")
            test_admin_center
            ;;
        "admin-browser")
            open_admin_browser_test
            ;;
        "admin-full")
            test_admin_center
            open_admin_browser_test
            ;;
        "help"|*)
            echo "Ansible Playbook Generator Web-v2 测试脚本"
            echo ""
            echo "用法: $0 <command>"
            echo ""
            echo "命令:"
            echo "  build     - 构建所有服务"
            echo "  start     - 启动所有服务"
            echo "  test      - 运行健康检查和功能测试"
            echo "  full      - 完整流程：清理->构建->启动->测试"
            echo "  stop      - 停止所有服务"
            echo "  restart   - 重启所有服务"
            echo "  logs      - 显示服务日志 (可选择服务名)"
            echo "  clean     - 完全清理所有容器和数据"
            echo "  info      - 显示服务信息"
            echo "  admin-test    - 运行管理中心导航功能测试"
            echo "  admin-browser - 打开浏览器进行管理中心手动测试"
            echo "  admin-full    - 完整管理中心测试（API + 浏览器）"
            echo "  help      - 显示此帮助信息"
            echo ""
            echo "示例:"
            echo "  $0 full                    # 完整测试流程"
            echo "  $0 start                   # 仅启动服务"
            echo "  $0 logs frontend          # 查看前端日志"
            echo "  $0 test                    # 运行测试"
            echo "  $0 admin-full             # 完整管理中心测试"
            echo ""
            ;;
    esac
}

# 脚本入口
main "$@"