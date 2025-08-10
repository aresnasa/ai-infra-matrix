#!/bin/bash

# 浏览器缓存和Cookie清理脚本
# AI基础设施矩阵 - 浏览器清理工具

echo "🧹 AI基础设施矩阵 - 浏览器缓存Cookie清理工具"
echo "============================================================"

# 检测操作系统
OS="$(uname -s)"
case "${OS}" in
    Linux*)     MACHINE=Linux;;
    Darwin*)    MACHINE=Mac;;
    CYGWIN*)    MACHINE=Cygwin;;
    MINGW*)     MACHINE=MinGw;;
    *)          MACHINE="UNKNOWN:${OS}"
esac

echo "🖥️  检测到操作系统: $MACHINE"
echo ""

# 显示手动清理说明
echo "📋 手动清理浏览器缓存和Cookie步骤:"
echo "============================================================"
echo ""

echo "🔧 Chrome/Edge 清理步骤:"
echo "1. 按 Ctrl+Shift+Delete (Mac: Cmd+Shift+Delete)"
echo "2. 选择 '所有时间' 时间范围"
echo "3. 勾选:"
echo "   ✅ 浏览记录"
echo "   ✅ Cookie及其他网站数据"
echo "   ✅ 缓存的图片和文件"
echo "   ✅ 托管应用数据"
echo "4. 点击 '清除数据'"
echo ""

echo "🦊 Firefox 清理步骤:"
echo "1. 按 Ctrl+Shift+Delete (Mac: Cmd+Shift+Delete)"
echo "2. 选择 '全部' 时间范围"
echo "3. 勾选:"
echo "   ✅ 浏览记录和下载记录"
echo "   ✅ Cookie"
echo "   ✅ 缓存"
echo "   ✅ 站点设置"
echo "4. 点击 '确定'"
echo ""

echo "🍎 Safari 清理步骤:"
echo "1. Safari > 偏好设置 > 隐私"
echo "2. 点击 '管理网站数据'"
echo "3. 点击 '移除全部'"
echo "4. 或者: 开发 > 清空缓存"
echo ""

# 显示localhost特定清理
echo "🎯 针对 localhost:8080 的特定清理:"
echo "============================================================"
echo "如果只想清理本项目相关的数据:"
echo ""
echo "Chrome/Edge:"
echo "1. 访问 chrome://settings/content/cookies"
echo "2. 搜索 'localhost'"
echo "3. 删除所有 localhost:8080 相关项目"
echo ""
echo "Firefox:"
echo "1. 按 F12 打开开发者工具"
echo "2. 存储标签 > Cookie"
echo "3. 删除 localhost:8080 下的所有cookie"
echo ""

# 自动清理功能（如果可能）
echo "🤖 自动清理尝试 (需要浏览器支持):"
echo "============================================================"

if [[ "$MACHINE" == "Mac" ]]; then
    echo "📱 macOS Safari 缓存清理..."
    if command -v osascript &> /dev/null; then
        osascript -e 'tell application "Safari" to activate' 2>/dev/null || true
        echo "✅ Safari 已激活，请手动清理或重启Safari"
    fi
fi

echo ""
echo "🔄 重启浏览器建议:"
echo "============================================================"
echo "1. 完全关闭浏览器（所有窗口和标签）"
echo "2. 等待 5-10 秒"
echo "3. 重新启动浏览器"
echo "4. 访问 http://localhost:8080"
echo ""

# 验证清理效果
echo "✅ 验证清理效果:"
echo "============================================================"
echo "清理后请测试以下步骤:"
echo "1. 访问 http://localhost:8080"
echo "2. 执行正常登录流程"
echo "3. 检查是否可以正常访问 http://localhost:8080/jupyter/hub/"
echo ""

# 测试连接
echo "🔍 测试服务连接状态..."
echo "============================================================"

# 检查服务是否运行
if curl -s http://localhost:8080/health >/dev/null 2>&1; then
    echo "✅ 主服务 (http://localhost:8080) - 正常"
else
    echo "❌ 主服务 (http://localhost:8080) - 连接失败"
    echo "   请确保服务正在运行: docker compose up -d"
fi

if curl -s http://localhost:8080/api/health >/dev/null 2>&1; then
    echo "✅ 后端API (http://localhost:8080/api) - 正常"
else
    echo "❌ 后端API (http://localhost:8080/api) - 连接失败"
fi

if curl -s http://localhost:8080/jupyter/hub/api >/dev/null 2>&1; then
    echo "✅ JupyterHub (http://localhost:8080/jupyter) - 正常"
else
    echo "❌ JupyterHub (http://localhost:8080/jupyter) - 连接失败"
fi

echo ""
echo "📞 如果问题仍然存在:"
echo "============================================================"
echo "1. 尝试无痕/隐私浏览模式"
echo "2. 尝试不同的浏览器"
echo "3. 检查浏览器控制台错误信息"
echo "4. 运行 SSO 测试: python test_sso_complete.py"
echo ""

echo "✨ 清理完成！请重启浏览器并重新尝试登录。"
