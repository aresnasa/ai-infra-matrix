#!/bin/bash
# =============================================================================
# 主机文件解析 API 调试脚本
# =============================================================================
# 用法:
#   ./test-host-parser-api.sh [API_URL]
#
# 参数:
#   API_URL - API 服务地址，默认 http://localhost:8080
#
# 示例:
#   ./test-host-parser-api.sh
#   ./test-host-parser-api.sh http://192.168.18.127:8080
# =============================================================================

set -e

API_URL="${1:-http://localhost:8080}"
DEBUG_ENDPOINT="${API_URL}/api/saltstack/hosts/parse/debug"
NORMAL_ENDPOINT="${API_URL}/api/saltstack/hosts/parse"

echo "=============================================="
echo " 主机文件解析 API 调试工具"
echo "=============================================="
echo "API URL: ${API_URL}"
echo "调试接口: ${DEBUG_ENDPOINT}"
echo "普通接口: ${NORMAL_ENDPOINT}"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 辅助函数
print_header() {
    echo ""
    echo -e "${BLUE}=== $1 ===${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# JSON 格式化输出
format_json() {
    if command -v jq &> /dev/null; then
        jq '.'
    elif command -v python3 &> /dev/null; then
        python3 -m json.tool
    else
        cat
    fi
}

# 测试 1: CSV 格式
test_csv() {
    print_header "测试 1: CSV 格式解析"
    
    CSV_CONTENT='host,port,username,password,use_sudo,minion_id,group
192.168.1.10,22,root,password123,false,minion-01,webservers
192.168.1.11,22,admin,password456,true,minion-02,databases
192.168.1.12,2222,deploy,password789,true,minion-03,webservers'

    echo "CSV 内容:"
    echo "$CSV_CONTENT"
    echo ""
    
    echo "调用调试接口..."
    RESPONSE=$(curl -s -X POST "${DEBUG_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d "{\"content\": $(echo "$CSV_CONTENT" | jq -Rs '.'), \"filename\": \"test.csv\"}")
    
    echo "响应结果:"
    echo "$RESPONSE" | format_json
    
    if echo "$RESPONSE" | grep -q '"success":true'; then
        print_success "CSV 解析成功"
    else
        print_error "CSV 解析失败"
    fi
}

# 测试 2: JSON 格式
test_json() {
    print_header "测试 2: JSON 格式解析"
    
    JSON_CONTENT='[
  {"host": "192.168.2.10", "port": 22, "username": "root", "password": "pass1", "use_sudo": false, "minion_id": "json-01", "group": "app"},
  {"host": "192.168.2.11", "port": 22, "username": "admin", "password": "pass2", "use_sudo": true, "minion_id": "json-02", "group": "db"}
]'

    echo "JSON 内容:"
    echo "$JSON_CONTENT" | format_json
    echo ""
    
    echo "调用调试接口..."
    RESPONSE=$(curl -s -X POST "${DEBUG_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d "{\"content\": $(echo "$JSON_CONTENT" | jq -Rs '.'), \"filename\": \"test.json\"}")
    
    echo "响应结果:"
    echo "$RESPONSE" | format_json
    
    if echo "$RESPONSE" | grep -q '"success":true'; then
        print_success "JSON 解析成功"
    else
        print_error "JSON 解析失败"
    fi
}

# 测试 3: YAML 格式
test_yaml() {
    print_header "测试 3: YAML 格式解析"
    
    YAML_CONTENT='hosts:
  - host: 192.168.3.10
    port: 22
    username: root
    password: yaml_pass1
    use_sudo: false
    minion_id: yaml-01
    group: frontend
  - host: 192.168.3.11
    port: 22
    username: deploy
    password: yaml_pass2
    use_sudo: true
    minion_id: yaml-02
    group: backend'

    echo "YAML 内容:"
    echo "$YAML_CONTENT"
    echo ""
    
    echo "调用调试接口..."
    RESPONSE=$(curl -s -X POST "${DEBUG_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d "{\"content\": $(echo "$YAML_CONTENT" | jq -Rs '.'), \"filename\": \"test.yaml\"}")
    
    echo "响应结果:"
    echo "$RESPONSE" | format_json
    
    if echo "$RESPONSE" | grep -q '"success":true'; then
        print_success "YAML 解析成功"
    else
        print_error "YAML 解析失败"
    fi
}

# 测试 4: Ansible INI 格式
test_ansible_ini() {
    print_header "测试 4: Ansible INI 格式解析"
    
    INI_CONTENT='[webservers]
web1 ansible_host=192.168.4.10 ansible_port=22 ansible_user=root ansible_password=ini_pass1 ansible_become=false
web2 ansible_host=192.168.4.11 ansible_port=22 ansible_user=admin ansible_password=ini_pass2 ansible_become=true

[databases]
db1 ansible_host=192.168.4.20 ansible_port=22 ansible_user=dba ansible_password=ini_pass3 ansible_become=true'

    echo "Ansible INI 内容:"
    echo "$INI_CONTENT"
    echo ""
    
    echo "调用调试接口..."
    RESPONSE=$(curl -s -X POST "${DEBUG_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d "{\"content\": $(echo "$INI_CONTENT" | jq -Rs '.'), \"filename\": \"inventory.ini\"}")
    
    echo "响应结果:"
    echo "$RESPONSE" | format_json
    
    if echo "$RESPONSE" | grep -q '"success":true'; then
        print_success "Ansible INI 解析成功"
    else
        print_error "Ansible INI 解析失败"
    fi
}

# 测试 5: 自动检测格式
test_auto_detect() {
    print_header "测试 5: 自动检测格式"
    
    CSV_CONTENT='host,port,username,password,use_sudo
192.168.5.10,22,root,auto_pass,false'

    echo "内容（无扩展名）:"
    echo "$CSV_CONTENT"
    echo ""
    
    echo "调用调试接口..."
    RESPONSE=$(curl -s -X POST "${DEBUG_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d "{\"content\": $(echo "$CSV_CONTENT" | jq -Rs '.'), \"filename\": \"hostfile\"}")
    
    echo "响应结果:"
    echo "$RESPONSE" | format_json
    
    if echo "$RESPONSE" | grep -q '"success":true'; then
        print_success "自动检测格式成功"
    else
        print_error "自动检测格式失败"
    fi
}

# 测试 6: 错误格式处理
test_error_handling() {
    print_header "测试 6: 错误格式处理"
    
    INVALID_CONTENT='this is not a valid format
just some random text
no structure at all'

    echo "无效内容:"
    echo "$INVALID_CONTENT"
    echo ""
    
    echo "调用调试接口..."
    RESPONSE=$(curl -s -X POST "${DEBUG_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d "{\"content\": $(echo "$INVALID_CONTENT" | jq -Rs '.'), \"filename\": \"invalid.txt\"}")
    
    echo "响应结果:"
    echo "$RESPONSE" | format_json
    
    if echo "$RESPONSE" | grep -q '"success":false'; then
        print_success "错误格式处理正确（返回失败）"
    else
        print_warning "预期返回失败，但返回了成功"
    fi
}

# 测试 7: 空文件处理
test_empty_file() {
    print_header "测试 7: 空文件处理"
    
    echo "调用调试接口（空内容）..."
    RESPONSE=$(curl -s -X POST "${DEBUG_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d '{"content": "", "filename": "empty.csv"}')
    
    echo "响应结果:"
    echo "$RESPONSE" | format_json
    
    if echo "$RESPONSE" | grep -q '"success":false'; then
        print_success "空文件处理正确（返回失败）"
    else
        print_warning "预期返回失败，但返回了成功"
    fi
}

# 测试 8: 安全检查（危险内容）
test_security_check() {
    print_header "测试 8: 安全检查（危险内容）"
    
    # 注意：这个测试用于验证安全检查是否工作
    DANGEROUS_CONTENT='host,port,username,password
$(rm -rf /),22,root,password'

    echo "危险内容示例:"
    echo "$DANGEROUS_CONTENT"
    echo ""
    
    echo "调用调试接口..."
    RESPONSE=$(curl -s -X POST "${DEBUG_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d "{\"content\": $(echo "$DANGEROUS_CONTENT" | jq -Rs '.'), \"filename\": \"dangerous.csv\"}")
    
    echo "响应结果:"
    echo "$RESPONSE" | format_json
    
    if echo "$RESPONSE" | grep -q '"success":false'; then
        print_success "安全检查生效（阻止了危险内容）"
    else
        print_error "安全检查未生效，需要检查！"
    fi
}

# 测试 9: 普通接口对比
test_normal_endpoint() {
    print_header "测试 9: 普通接口对比"
    
    CSV_CONTENT='host,port,username,password,use_sudo
192.168.9.10,22,root,normal_pass,false
192.168.9.11,22,admin,normal_pass2,true'

    echo "CSV 内容:"
    echo "$CSV_CONTENT"
    echo ""
    
    echo "调用普通接口..."
    RESPONSE=$(curl -s -X POST "${NORMAL_ENDPOINT}" \
        -H "Content-Type: application/json" \
        -d "{\"content\": $(echo "$CSV_CONTENT" | jq -Rs '.'), \"filename\": \"test.csv\"}")
    
    echo "响应结果:"
    echo "$RESPONSE" | format_json
    
    if echo "$RESPONSE" | grep -q '"success":true'; then
        print_success "普通接口调用成功"
    else
        print_error "普通接口调用失败"
    fi
}

# 测试 10: 交互式测试
test_interactive() {
    print_header "测试 10: 交互式测试"
    
    echo "请输入要测试的文件路径（或按 Enter 跳过）:"
    read -r FILE_PATH
    
    if [ -n "$FILE_PATH" ] && [ -f "$FILE_PATH" ]; then
        CONTENT=$(cat "$FILE_PATH")
        FILENAME=$(basename "$FILE_PATH")
        
        echo "文件内容预览（前500字符）:"
        echo "${CONTENT:0:500}"
        echo ""
        
        echo "调用调试接口..."
        RESPONSE=$(curl -s -X POST "${DEBUG_ENDPOINT}" \
            -H "Content-Type: application/json" \
            -d "{\"content\": $(echo "$CONTENT" | jq -Rs '.'), \"filename\": \"$FILENAME\"}")
        
        echo "响应结果:"
        echo "$RESPONSE" | format_json
    else
        echo "跳过交互式测试"
    fi
}

# 运行所有测试
run_all_tests() {
    test_csv
    test_json
    test_yaml
    test_ansible_ini
    test_auto_detect
    test_error_handling
    test_empty_file
    test_security_check
    test_normal_endpoint
}

# 主程序
main() {
    echo ""
    echo "选择测试模式:"
    echo "  1) 运行所有测试"
    echo "  2) 仅测试 CSV"
    echo "  3) 仅测试 JSON"
    echo "  4) 仅测试 YAML"
    echo "  5) 仅测试 Ansible INI"
    echo "  6) 测试自动检测"
    echo "  7) 测试错误处理"
    echo "  8) 测试安全检查"
    echo "  9) 交互式测试（自定义文件）"
    echo "  0) 退出"
    echo ""
    echo "请输入选项 [默认: 1]:"
    read -r CHOICE
    
    case "${CHOICE:-1}" in
        1) run_all_tests ;;
        2) test_csv ;;
        3) test_json ;;
        4) test_yaml ;;
        5) test_ansible_ini ;;
        6) test_auto_detect ;;
        7) test_error_handling ;;
        8) test_security_check ;;
        9) test_interactive ;;
        0) echo "退出"; exit 0 ;;
        *) echo "无效选项"; exit 1 ;;
    esac
    
    echo ""
    echo "=============================================="
    echo " 测试完成"
    echo "=============================================="
}

# 检查 jq 是否安装
check_dependencies() {
    if ! command -v jq &> /dev/null; then
        print_warning "jq 未安装，JSON 输出将不会格式化"
        print_warning "安装方法: brew install jq (macOS) 或 apt install jq (Ubuntu)"
    fi
    
    if ! command -v curl &> /dev/null; then
        print_error "curl 未安装，无法进行测试"
        exit 1
    fi
}

check_dependencies
main
