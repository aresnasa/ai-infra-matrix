#!/bin/bash

# =======================================================================
# AI Infrastructure Matrix - 对象存储快速测试脚本
# =======================================================================
# 功能: 快速测试对象存储相关页面和API
# =======================================================================

set -e

# 配置
FRONTEND_URL="${FRONTEND_URL:-http://localhost:8080}"
LOG_FILE="./test-object-storage.log"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [INFO] $1" >> "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [SUCCESS] $1" >> "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [WARN] $1" >> "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [ERROR] $1" >> "$LOG_FILE"
}

# 测试HTTP响应
test_url() {
    local url="$1"
    local description="$2"
    local expected_status="${3:-200}"
    
    log_info "测试: $description ($url)"
    
    local response=$(curl -s -w "%{http_code}" -o /dev/null "$url" --connect-timeout 10 --max-time 30)
    
    if [[ "$response" == "$expected_status" ]]; then
        log_success "$description - HTTP $response ✅"
        return 0
    elif [[ "$response" == "401" ]] || [[ "$response" == "403" ]]; then
        log_warn "$description - 需要认证 (HTTP $response) ⚠️"
        return 0
    else
        log_error "$description - HTTP $response ❌"
        return 1
    fi
}

# 测试页面内容
test_page_content() {
    local url="$1"
    local description="$2"
    local expected_content="$3"
    
    log_info "测试页面内容: $description"
    
    local content=$(curl -s "$url" --connect-timeout 10 --max-time 30)
    local status_code=$?
    
    if [[ $status_code -eq 0 ]] && [[ "$content" == *"$expected_content"* ]]; then
        log_success "$description - 内容验证通过 ✅"
        return 0
    else
        log_error "$description - 内容验证失败 ❌"
        return 1
    fi
}

# 主测试函数
main() {
    echo -e "${BLUE}================================================================${NC}"
    echo -e "${BLUE}           AI Infrastructure Matrix${NC}"
    echo -e "${BLUE}         对象存储功能快速测试${NC}"
    echo -e "${BLUE}================================================================${NC}"
    
    log_info "开始测试对象存储功能..."
    log_info "测试目标: $FRONTEND_URL"
    
    local total_tests=0
    local passed_tests=0
    
    # 测试基础服务
    echo -e "\n${YELLOW}📡 基础服务测试${NC}"
    
    if test_url "$FRONTEND_URL/health" "系统健康检查"; then
        ((passed_tests++))
    fi
    ((total_tests++))
    
    if test_url "$FRONTEND_URL/api/health" "后端API健康检查"; then
        ((passed_tests++))
    fi
    ((total_tests++))
    
    # 测试MinIO服务
    echo -e "\n${YELLOW}🗄️ MinIO服务测试${NC}"
    
    if test_url "$FRONTEND_URL/minio/health" "MinIO健康检查"; then
        ((passed_tests++))
    fi
    ((total_tests++))
    
    if test_url "$FRONTEND_URL/minio-console/" "MinIO控制台代理"; then
        ((passed_tests++))
    fi
    ((total_tests++))
    
    # 测试对象存储API
    echo -e "\n${YELLOW}🔌 对象存储API测试${NC}"
    
    if test_url "$FRONTEND_URL/api/object-storage/configs" "对象存储配置API" "200"; then
        ((passed_tests++))
    elif test_url "$FRONTEND_URL/api/object-storage/configs" "对象存储配置API" "401"; then
        ((passed_tests++))
    fi
    ((total_tests++))
    
    # 测试前端页面
    echo -e "\n${YELLOW}🌐 前端页面测试${NC}"
    
    if test_page_content "$FRONTEND_URL/" "主页面" "<!DOCTYPE html>"; then
        ((passed_tests++))
    fi
    ((total_tests++))
    
    if test_page_content "$FRONTEND_URL/object-storage" "对象存储主页面" "<!DOCTYPE html>"; then
        ((passed_tests++))
    fi
    ((total_tests++))
    
    if test_page_content "$FRONTEND_URL/admin/object-storage" "对象存储管理页面" "<!DOCTYPE html>"; then
        ((passed_tests++))
    fi
    ((total_tests++))
    
    # 测试iframe测试页面
    if test_page_content "$FRONTEND_URL/test-object-storage-iframe.html" "iframe测试页面" "对象存储 iframe"; then
        ((passed_tests++))
    fi
    ((total_tests++))
    
    # 生成报告
    echo -e "\n${BLUE}================================================================${NC}"
    echo -e "${BLUE}                    测试结果报告${NC}"
    echo -e "${BLUE}================================================================${NC}"
    
    local pass_rate=$(echo "scale=2; $passed_tests * 100 / $total_tests" | bc -l 2>/dev/null || echo "N/A")
    
    echo -e "${BLUE}📊 测试统计:${NC}"
    echo "  总计: $total_tests"
    echo "  通过: $passed_tests ✅"
    echo "  失败: $((total_tests - passed_tests)) ❌"
    echo "  通过率: ${pass_rate}%"
    
    echo -e "\n${BLUE}🌐 访问地址:${NC}"
    echo "  • 主页面: $FRONTEND_URL"
    echo "  • 对象存储: $FRONTEND_URL/object-storage"
    echo "  • 存储管理: $FRONTEND_URL/admin/object-storage"
    echo "  • MinIO控制台: $FRONTEND_URL/minio-console/"
    echo "  • 测试页面: $FRONTEND_URL/test-object-storage-iframe.html"
    
    echo -e "\n${BLUE}📋 问题排查:${NC}"
    if [[ $passed_tests -lt $total_tests ]]; then
        echo "  • 检查服务是否启动: docker compose ps"
        echo "  • 检查服务日志: docker compose logs [service]"
        echo "  • 验证nginx配置: docker compose exec nginx nginx -t"
        echo "  • 检查MinIO状态: docker compose logs minio"
    else
        echo "  🎉 所有测试通过，对象存储功能正常！"
    fi
    
    echo -e "\n${BLUE}📄 详细日志: $LOG_FILE${NC}"
    
    # 返回合适的退出码
    if [[ $passed_tests -eq $total_tests ]]; then
        log_success "所有测试通过"
        return 0
    else
        log_error "存在测试失败"
        return 1
    fi
}

# 清理函数
cleanup() {
    log_info "清理临时文件..."
}

# 错误处理
trap cleanup EXIT

# 执行主函数
main "$@"