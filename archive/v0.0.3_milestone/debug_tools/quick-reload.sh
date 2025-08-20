#!/bin/bash

# 快速重载前端 - 简化版本
# 使用方法: ./quick-reload.sh

echo "🔄 重载nginx配置..."
docker exec ai-infra-nginx nginx -s reload

echo "🕒 生成新的缓存破坏时间戳..."
TIMESTAMP=$(date +%s)

echo "✅ 前端已重载!"
echo ""
echo "🌐 访问链接:"
echo "   http://localhost:8080/jupyterhub?v=$TIMESTAMP"
echo ""
echo "💡 或者在浏览器中按 Cmd+Shift+R (Mac) 或 Ctrl+F5 (Windows) 强制刷新"
