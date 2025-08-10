#!/bin/bash

# 前端开发热重载脚本 - AI基础设施矩阵
# 用于开发过程中快速重载静态文件

set -e

echo "🔄 AI基础设施矩阵 - 前端开发热重载"
echo "================================="

# 检查Docker服务状态
if ! docker-compose ps nginx | grep -q "Up"; then
    echo "❌ nginx服务未运行，请先启动服务"
    echo "   运行: docker-compose up -d nginx"
    exit 1
fi

# 强制清理nginx缓存
echo "🧹 清理nginx缓存..."
docker exec ai-infra-nginx nginx -s reload

# 添加时间戳到文件以强制浏览器重新加载
TIMESTAMP=$(date +%s)
echo "⏰ 添加缓存破坏时间戳: $TIMESTAMP"

# 更新主要的HTML文件，添加版本参数
for file in src/shared/jupyterhub/*.html; do
    if [ -f "$file" ]; then
        filename=$(basename "$file")
        echo "📝 处理文件: $filename"
        
        # 在HTML中添加一个隐藏的版本标记
        if ! grep -q "data-version" "$file"; then
            sed -i.bak "s|<head>|<head><meta name=\"cache-version\" content=\"$TIMESTAMP\" data-version=\"dev\">|" "$file"
            rm -f "$file.bak"
        else
            sed -i.bak "s|content=\"[0-9]*\"|content=\"$TIMESTAMP\"|" "$file"
            rm -f "$file.bak"
        fi
    fi
done

echo "✅ 前端文件已更新"
echo ""
echo "🌐 测试链接:"
echo "   主页面: http://localhost:8080/jupyterhub?v=$TIMESTAMP"
echo "   测试页: http://localhost:8080/jupyterhub/iframe_test.html?v=$TIMESTAMP"
echo ""
echo "💡 提示: URL中的?v=$TIMESTAMP参数会强制浏览器重新加载"
echo "💡 提示: 可以按 Ctrl+F5 强制刷新浏览器缓存"
echo ""
echo "🔍 查看nginx访问日志:"
echo "   docker logs ai-infra-nginx --tail 10"
echo ""
echo "🐛 调试模式 - 查看浏览器控制台："
echo "   F12 -> Console 查看JavaScript错误"
echo "   F12 -> Network 查看网络请求"
