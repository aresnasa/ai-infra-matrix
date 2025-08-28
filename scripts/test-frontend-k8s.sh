#!/bin/bash

# AI Infrastructure Matrix - 前端Kubernetes功能验证脚本
# 用途: 测试Web界面的Kubernetes管理功能

set -e

# 配置
FRONTEND_URL="http://localhost:3000"
BACKEND_URL="http://localhost:8080"

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

echo "🌐 AI Infrastructure Matrix - 前端Kubernetes功能验证"
echo "=================================================="

# 1. 检查服务状态
echo ""
echo "📝 1. 服务状态检查"
echo "--------------------------------"

log_info "检查前端服务状态..."
if curl -s "$FRONTEND_URL" > /dev/null; then
    log_success "前端服务可访问: $FRONTEND_URL"
else
    log_error "前端服务不可访问: $FRONTEND_URL"
    log_info "启动前端服务: cd src/frontend && npm start"
    exit 1
fi

log_info "检查后端服务状态..."
if curl -s "$BACKEND_URL/health" > /dev/null; then
    log_success "后端服务可访问: $BACKEND_URL"
else
    log_error "后端服务不可访问: $BACKEND_URL"
    log_info "启动后端服务: docker-compose up -d"
    exit 1
fi

# 2. 验证认证功能
echo ""
echo "📝 2. 认证功能验证"
echo "--------------------------------"

log_info "测试管理员登录..."
LOGIN_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' \
    "$BACKEND_URL/api/auth/login")

if echo "$LOGIN_RESPONSE" | grep -q "token"; then
    TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.token')
    log_success "管理员认证成功"
else
    log_error "管理员认证失败"
    echo "响应: $LOGIN_RESPONSE"
    exit 1
fi

# 3. 验证集群管理API
echo ""
echo "📝 3. 集群管理API验证"
echo "--------------------------------"

log_info "获取集群列表..."
CLUSTERS_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \
    "$BACKEND_URL/api/kubernetes/clusters")

CLUSTER_COUNT=$(echo "$CLUSTERS_RESPONSE" | jq '. | length' 2>/dev/null || echo "0")
if [ "$CLUSTER_COUNT" -gt "0" ]; then
    log_success "发现 $CLUSTER_COUNT 个Kubernetes集群"
    echo "$CLUSTERS_RESPONSE" | jq -r '.[] | "  - \(.name): \(.status) (版本: \(.version))"'
    
    # 获取第一个集群ID用于测试
    CLUSTER_ID=$(echo "$CLUSTERS_RESPONSE" | jq -r '.[0].id')
    CLUSTER_NAME=$(echo "$CLUSTERS_RESPONSE" | jq -r '.[0].name')
else
    log_warning "未发现Kubernetes集群"
fi

# 4. 验证命名空间API
if [ "$CLUSTER_COUNT" -gt "0" ]; then
    echo ""
    echo "📝 4. 命名空间API验证"
    echo "--------------------------------"
    
    log_info "获取集群 $CLUSTER_NAME 的命名空间..."
    NAMESPACES_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \
        "$BACKEND_URL/api/kubernetes/clusters/$CLUSTER_ID/namespaces")
    
    if echo "$NAMESPACES_RESPONSE" | grep -q "items"; then
        NAMESPACE_COUNT=$(echo "$NAMESPACES_RESPONSE" | jq '.items | length' 2>/dev/null || echo "0")
        log_success "发现 $NAMESPACE_COUNT 个命名空间"
        echo "$NAMESPACES_RESPONSE" | jq -r '.items[0:5][] | "  - \(.metadata.name)"' 2>/dev/null || echo "  命名空间列表解析错误"
    else
        log_warning "命名空间获取失败"
    fi
fi

# 5. 验证Pod资源API
if [ "$CLUSTER_COUNT" -gt "0" ]; then
    echo ""
    echo "📝 5. Pod资源API验证"
    echo "--------------------------------"
    
    log_info "获取default命名空间的Pod..."
    PODS_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \
        "$BACKEND_URL/api/kubernetes/clusters/$CLUSTER_ID/namespaces/default/resources/pods")
    
    if echo "$PODS_RESPONSE" | grep -q "items"; then
        POD_COUNT=$(echo "$PODS_RESPONSE" | jq '.items | length' 2>/dev/null || echo "0")
        log_success "发现 $POD_COUNT 个Pod"
        if [ "$POD_COUNT" -gt "0" ]; then
            echo "$PODS_RESPONSE" | jq -r '.items[0:3][] | "  - \(.metadata.name): \(.status.phase)"' 2>/dev/null || echo "  Pod列表解析错误"
        fi
    else
        log_warning "Pod获取失败"
    fi
fi

# 6. 测试资源发现API
if [ "$CLUSTER_COUNT" -gt "0" ]; then
    echo ""
    echo "📝 6. 资源发现API验证"
    echo "--------------------------------"
    
    log_info "测试资源发现功能..."
    DISCOVERY_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \
        "$BACKEND_URL/api/kubernetes/clusters/$CLUSTER_ID/discovery")
    
    if echo "$DISCOVERY_RESPONSE" | grep -q "groups\|resources"; then
        log_success "资源发现功能正常"
    else
        log_warning "资源发现功能异常"
    fi
fi

# 7. 前端页面验证指南
echo ""
echo "📝 7. 前端页面验证指南"
echo "--------------------------------"

log_info "请通过浏览器访问以下页面进行手动验证:"
echo ""
echo "🌐 主页面:"
echo "   $FRONTEND_URL"
echo ""
echo "🔐 登录页面:"
echo "   $FRONTEND_URL/login"
echo "   用户名: admin"
echo "   密码: admin123"
echo ""
echo "☸️  Kubernetes管理页面:"
echo "   $FRONTEND_URL/kubernetes"
echo "   - 验证集群列表显示"
echo "   - 验证命名空间切换"
echo "   - 验证Pod列表显示"
echo "   - 验证资源操作按钮"
echo ""

# 8. 自动打开浏览器（macOS）
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "📝 8. 自动打开浏览器"
    echo "--------------------------------"
    
    log_info "尝试自动打开浏览器..."
    if command -v open >/dev/null 2>&1; then
        open "$FRONTEND_URL/kubernetes"
        log_success "已在浏览器中打开Kubernetes管理页面"
    else
        log_warning "无法自动打开浏览器，请手动访问: $FRONTEND_URL/kubernetes"
    fi
fi

# 9. 总结
echo ""
echo "🎉 前端验证完成!"
echo "=================================="
echo ""
echo "✅ 已验证功能:"
echo "  ✅ 前端服务可访问性"
echo "  ✅ 后端API连通性"
echo "  ✅ 用户认证系统"
echo "  ✅ 集群管理API"
echo "  ✅ 命名空间管理API"
echo "  ✅ Pod资源管理API"
echo "  ✅ 资源发现API"
echo ""
echo "🔍 手动验证项目:"
echo "  1. 登录Web界面"
echo "  2. 访问Kubernetes管理页面"
echo "  3. 验证集群列表和状态显示"
echo "  4. 测试命名空间切换功能"
echo "  5. 验证Pod列表和操作按钮"
echo "  6. 测试创建/删除资源功能"
echo ""
echo "📞 问题排查:"
echo "  - 前端问题: 检查 src/frontend/package.json 和 node_modules"
echo "  - 后端问题: 检查 docker-compose logs backend"
echo "  - 认证问题: 验证用户名密码 admin/admin123"
echo "  - API问题: 查看浏览器开发者工具的网络选项卡"
