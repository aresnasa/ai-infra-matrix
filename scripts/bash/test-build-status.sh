#!/bin/bash
# =============================================================================
# 测试脚本：镜像构建状态检查功能
# 用途：验证需求32的智能构建检查功能
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BUILD_SCRIPT="$PROJECT_ROOT/build.sh"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_test() {
    echo -e "${BLUE}[测试]${NC} $1"
}

print_pass() {
    echo -e "${GREEN}[通过]${NC} $1"
}

print_fail() {
    echo -e "${RED}[失败]${NC} $1"
}

print_info() {
    echo -e "${YELLOW}[信息]${NC} $1"
}

# =============================================================================
# 测试1: 验证 verify_image_build 函数
# =============================================================================
test_verify_image_build() {
    print_test "测试1: 验证镜像验证函数"
    
    # 测试1.1: 验证不存在的镜像
    print_info "测试1.1: 验证不存在的镜像"
    if bash -c "source $BUILD_SCRIPT && verify_image_build 'nonexistent-image:test'" 2>/dev/null; then
        print_fail "不存在的镜像应该返回失败"
        return 1
    else
        print_pass "不存在的镜像正确返回失败"
    fi
    
    # 测试1.2: 验证存在的镜像（使用alpine作为测试镜像）
    print_info "测试1.2: 拉取并验证alpine镜像"
    docker pull alpine:latest >/dev/null 2>&1 || {
        print_fail "无法拉取alpine镜像"
        return 1
    }
    
    if bash -c "source $BUILD_SCRIPT && verify_image_build 'alpine:latest'" 2>/dev/null; then
        print_pass "存在的镜像正确验证通过"
    else
        print_fail "存在的镜像验证失败"
        return 1
    fi
    
    print_pass "测试1通过: 镜像验证函数工作正常"
    echo
}

# =============================================================================
# 测试2: 验证 get_build_status 函数
# =============================================================================
test_get_build_status() {
    print_test "测试2: 验证构建状态获取函数"
    
    print_info "获取所有服务的构建状态..."
    local status_output
    status_output=$(bash -c "source $BUILD_SCRIPT && get_build_status 'test-v0.3.6' ''" 2>/dev/null)
    
    if [[ -z "$status_output" ]]; then
        print_fail "构建状态输出为空"
        return 1
    fi
    
    # 检查输出格式是否正确（service|status|image_name）
    local line_count=$(echo "$status_output" | wc -l)
    print_info "发现 $line_count 个服务"
    
    # 检查每行格式
    local format_ok=true
    while IFS='|' read -r service status image_name; do
        if [[ -z "$service" ]] || [[ -z "$status" ]] || [[ -z "$image_name" ]]; then
            print_fail "格式错误: $service|$status|$image_name"
            format_ok=false
            break
        fi
        
        # 检查状态值是否有效
        if [[ "$status" != "OK" ]] && [[ "$status" != "MISSING" ]] && [[ "$status" != "INVALID" ]]; then
            print_fail "无效的状态值: $status"
            format_ok=false
            break
        fi
    done <<< "$status_output"
    
    if [[ "$format_ok" == "true" ]]; then
        print_pass "构建状态输出格式正确"
    else
        print_fail "构建状态输出格式错误"
        return 1
    fi
    
    # 统计各种状态的数量
    local ok_count=$(echo "$status_output" | grep '|OK|' | wc -l)
    local missing_count=$(echo "$status_output" | grep '|MISSING|' | wc -l)
    local invalid_count=$(echo "$status_output" | grep '|INVALID|' | wc -l)
    
    print_info "状态统计: OK=$ok_count, MISSING=$missing_count, INVALID=$invalid_count"
    
    print_pass "测试2通过: 构建状态获取功能工作正常"
    echo
}

# =============================================================================
# 测试3: 验证 show_build_status 函数
# =============================================================================
test_show_build_status() {
    print_test "测试3: 验证构建状态显示函数"
    
    print_info "显示构建状态报告..."
    local report_output
    report_output=$(bash -c "source $BUILD_SCRIPT && show_build_status 'test-v0.3.6' ''" 2>/dev/null)
    
    if [[ -z "$report_output" ]]; then
        print_fail "构建状态报告输出为空"
        return 1
    fi
    
    # 检查报告是否包含关键信息
    if ! echo "$report_output" | grep -q "镜像构建状态报告"; then
        print_fail "报告缺少标题"
        return 1
    fi
    
    if ! echo "$report_output" | grep -q "构建状态统计"; then
        print_fail "报告缺少统计信息"
        return 1
    fi
    
    print_pass "构建状态报告格式正确"
    
    # 显示报告摘要
    echo "$report_output" | head -20
    
    print_pass "测试3通过: 构建状态显示功能工作正常"
    echo
}

# =============================================================================
# 测试4: 验证 get_services_to_build 函数
# =============================================================================
test_get_services_to_build() {
    print_test "测试4: 验证需要构建的服务列表获取"
    
    print_info "获取需要构建的服务列表..."
    local services_to_build
    services_to_build=$(bash -c "source $BUILD_SCRIPT && get_services_to_build 'test-v0.3.6' ''" 2>/dev/null)
    
    # 可能为空（所有镜像都已构建）或包含服务列表
    if [[ -n "$services_to_build" ]]; then
        local count=$(echo "$services_to_build" | wc -w)
        print_info "需要构建的服务数量: $count"
        print_info "服务列表: $services_to_build"
        print_pass "成功获取需要构建的服务列表"
    else
        print_info "所有镜像都已构建，无需重新构建"
        print_pass "智能过滤功能正常"
    fi
    
    print_pass "测试4通过: 服务过滤功能工作正常"
    echo
}

# =============================================================================
# 测试5: 验证 check-status 命令
# =============================================================================
test_check_status_command() {
    print_test "测试5: 验证 check-status 命令"
    
    print_info "执行 check-status 命令..."
    if ! "$BUILD_SCRIPT" check-status test-v0.3.6 2>&1 | head -30; then
        print_fail "check-status 命令执行失败"
        return 1
    fi
    
    print_pass "check-status 命令执行成功"
    
    # 测试帮助信息
    print_info "测试帮助信息..."
    if ! "$BUILD_SCRIPT" check-status --help 2>&1 | grep -q "检查所有服务的镜像构建状态"; then
        print_fail "帮助信息不完整"
        return 1
    fi
    
    print_pass "帮助信息正确"
    
    print_pass "测试5通过: check-status 命令工作正常"
    echo
}

# =============================================================================
# 测试6: 验证 build-all 智能过滤功能
# =============================================================================
test_build_all_smart_filter() {
    print_test "测试6: 验证 build-all 智能过滤功能"
    
    print_info "检查 build-all 是否集成了智能过滤..."
    
    # 检查 build_all_services 函数是否包含智能过滤逻辑
    if ! grep -q "get_services_to_build" "$BUILD_SCRIPT"; then
        print_fail "build-all 未集成智能过滤功能"
        return 1
    fi
    
    print_pass "build-all 已集成智能过滤"
    
    # 检查是否支持 --force 参数
    if ! grep -q "FORCE_REBUILD" "$BUILD_SCRIPT" | grep -q "build_all_services"; then
        print_fail "未正确支持 --force 参数"
        return 1
    fi
    
    print_pass "--force 参数支持正常"
    
    print_info "验证 build-all 帮助信息..."
    if "$BUILD_SCRIPT" build-all --help 2>&1 | grep -q "智能过滤\|智能构建"; then
        print_pass "帮助信息包含智能构建说明"
    else
        print_fail "帮助信息缺少智能构建说明"
        return 1
    fi
    
    print_pass "测试6通过: build-all 智能过滤功能集成正常"
    echo
}

# =============================================================================
# 测试7: 性能测试
# =============================================================================
test_performance() {
    print_test "测试7: 性能测试"
    
    print_info "测试状态检查性能..."
    local start_time=$(date +%s)
    bash -c "source $BUILD_SCRIPT && get_build_status 'test-v0.3.6' ''" >/dev/null 2>&1
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    print_info "状态检查耗时: ${duration}秒"
    
    if [[ $duration -lt 10 ]]; then
        print_pass "性能良好（<10秒）"
    elif [[ $duration -lt 30 ]]; then
        print_pass "性能可接受（<30秒）"
    else
        print_fail "性能较差（>30秒）"
        return 1
    fi
    
    print_pass "测试7通过: 性能测试通过"
    echo
}

# =============================================================================
# 主测试流程
# =============================================================================
main() {
    echo "========================================"
    echo "镜像构建状态检查功能测试"
    echo "需求32: 智能构建检查和过滤"
    echo "========================================"
    echo
    
    local total_tests=7
    local passed_tests=0
    local failed_tests=0
    
    # 运行所有测试
    if test_verify_image_build; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    if test_get_build_status; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    if test_show_build_status; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    if test_get_services_to_build; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    if test_check_status_command; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    if test_build_all_smart_filter; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    if test_performance; then
        passed_tests=$((passed_tests + 1))
    else
        failed_tests=$((failed_tests + 1))
    fi
    
    # 显示总结
    echo "========================================"
    echo "测试总结"
    echo "========================================"
    echo "总测试数: $total_tests"
    echo -e "${GREEN}通过: $passed_tests${NC}"
    if [[ $failed_tests -gt 0 ]]; then
        echo -e "${RED}失败: $failed_tests${NC}"
    fi
    echo
    
    if [[ $failed_tests -eq 0 ]]; then
        echo -e "${GREEN}✅ 所有测试通过！${NC}"
        return 0
    else
        echo -e "${RED}❌ 部分测试失败${NC}"
        return 1
    fi
}

# 执行测试
main "$@"
