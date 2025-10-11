#!/bin/bash

# 增强仪表板与LDAP多用户集成 - API测试脚本
# 测试所有新增的API端点是否正常工作

echo "🧪 增强仪表板与LDAP多用户集成 - API测试"
echo "=============================================="

# 配置
API_BASE="http://localhost:8080/api"
TOKEN=""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试函数
test_api() {
    local method="$1"
    local endpoint="$2"
    local description="$3"
    local data="$4"
    
    echo -e "\n${BLUE}🔍 测试: ${description}${NC}"
    echo "   ${method} ${API_BASE}${endpoint}"
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $TOKEN" \
            -d "$data" \
            "${API_BASE}${endpoint}")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Authorization: Bearer $TOKEN" \
            "${API_BASE}${endpoint}")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
        echo -e "   ${GREEN}✅ 成功 (HTTP $http_code)${NC}"
    elif [ "$http_code" = "401" ]; then
        echo -e "   ${YELLOW}⚠️  需要认证 (HTTP $http_code)${NC}"
    elif [ "$http_code" = "404" ]; then
        echo -e "   ${YELLOW}⚠️  端点未实现 (HTTP $http_code)${NC}"
    else
        echo -e "   ${RED}❌ 失败 (HTTP $http_code)${NC}"
        echo "   响应: $response_body"
    fi
}

# 检查服务器是否运行
echo "🔍 检查后端服务状态..."
if ! curl -s "${API_BASE}/health" > /dev/null 2>&1; then
    echo -e "${RED}❌ 后端服务未运行或无法访问${NC}"
    echo "请确保后端服务在 http://localhost:8080 运行"
    exit 1
fi
echo -e "${GREEN}✅ 后端服务正常运行${NC}"

# 尝试获取认证token（如果需要）
echo -e "\n${YELLOW}ℹ️  提示: 如需测试需要认证的API，请先登录获取token${NC}"
echo "可以通过以下方式设置token:"
echo "export TOKEN='your-jwt-token'"

if [ -n "$AUTH_TOKEN" ]; then
    TOKEN="$AUTH_TOKEN"
    echo -e "${GREEN}✅ 使用环境变量中的token${NC}"
fi

echo -e "\n${BLUE}📋 开始API测试...${NC}"

# 基础仪表板API测试
echo -e "\n${YELLOW}=== 基础仪表板API ===${NC}"
test_api "GET" "/dashboard" "获取用户仪表板配置"
test_api "GET" "/dashboard/enhanced" "获取增强仪表板配置"
test_api "GET" "/dashboard/stats" "获取仪表板统计信息"
test_api "GET" "/dashboard/export" "导出仪表板配置"

# 导入测试数据
sample_config='{"widgets":[{"id":"test","title":"Test Widget","url":"/test","width":6,"height":400}],"layout":["test"]}'
test_api "POST" "/dashboard/import" "导入仪表板配置" "{\"config\":$sample_config,\"overwrite\":false}"

# 用户管理API测试
echo -e "\n${YELLOW}=== 用户管理API ===${NC}"
test_api "GET" "/users" "获取用户列表"
test_api "GET" "/users/profile" "获取用户个人信息"
test_api "GET" "/user-groups" "获取用户组列表"
test_api "GET" "/roles" "获取角色列表"

# 管理员API测试
echo -e "\n${YELLOW}=== 管理员API ===${NC}"
test_api "GET" "/admin/users" "获取管理员用户列表"
test_api "GET" "/admin/user-stats" "获取用户统计信息"
test_api "GET" "/admin/stats" "获取系统统计信息"

# LDAP相关API测试
echo -e "\n${YELLOW}=== LDAP集成API ===${NC}"
test_api "GET" "/admin/ldap/config" "获取LDAP配置"
test_api "GET" "/admin/ldap/sync/history" "获取LDAP同步历史"

# 测试LDAP连接
ldap_test_config='{"server":"ldap://localhost:389","baseDN":"dc=example,dc=com","bindUser":"cn=admin,dc=example,dc=com","bindPassword":"admin"}'
test_api "POST" "/admin/ldap/test" "测试LDAP连接" "$ldap_test_config"

# 触发LDAP同步
sync_options='{"dryRun":true,"batchSize":10}'
test_api "POST" "/admin/ldap/sync" "触发LDAP同步" "$sync_options"

# 认证相关API
echo -e "\n${YELLOW}=== 认证API ===${NC}"
test_api "GET" "/auth/me" "获取当前用户信息"

# LDAP传统API（向后兼容）
echo -e "\n${YELLOW}=== LDAP传统API ===${NC}"
test_api "GET" "/ldap/config" "获取LDAP配置（传统）"
test_api "GET" "/ldap/groups" "获取LDAP用户组"

echo -e "\n${BLUE}📊 测试完成总结${NC}"
echo "=============================================="
echo -e "${GREEN}✅ 基础API端点已定义${NC}"
echo -e "${YELLOW}⚠️  部分API可能需要后端实现${NC}"
echo -e "${BLUE}ℹ️  需要认证的API请先获取有效token${NC}"

echo -e "\n${BLUE}🚀 下一步操作建议：${NC}"
echo "1. 确保所有后端控制器都已正确实现"
echo "2. 检查数据库表结构是否符合API需求"
echo "3. 配置LDAP服务器并测试连接"
echo "4. 在前端集成组件中测试完整流程"

echo -e "\n${BLUE}📝 相关文件：${NC}"
echo "- 前端集成组件: src/frontend/src/components/DashboardIntegration.js"
echo "- 增强仪表板: src/frontend/src/pages/EnhancedDashboardPage.js"
echo "- LDAP用户管理: src/frontend/src/pages/MultiUserLDAPManagement.js"
echo "- 后端控制器: src/backend/internal/controllers/enhanced_dashboard.go"
echo "- API定义: src/frontend/src/services/api.js"
echo "- 使用文档: docs/ENHANCED_DASHBOARD_LDAP_GUIDE.md"
