#!/bin/bash

# 项目状态检查脚本
echo "Salt Docker Infrastructure - Project Status Check"
echo "=================================================="

# 检查文件结构
echo "📁 Project Structure:"
find . -type f -name "*.yml" -o -name "*.conf" -o -name "*.sls" -o -name "Dockerfile*" -o -name "*.sh" | head -20

echo ""
echo "🐳 Docker Compose Configuration:"
docker-compose config --quiet && echo "✅ docker-compose.yml is valid" || echo "❌ docker-compose.yml has errors"

echo ""
echo "🔧 Script Permissions:"
for script in start.sh stop.sh salt-manager.sh run-tests-full.sh; do
    if [ -x "$script" ]; then
        echo "✅ $script is executable"
    else
        echo "❌ $script is not executable"
    fi
done

echo ""
echo "📋 Available Management Commands:"
echo "  ./salt-manager.sh start    - Start infrastructure"
echo "  ./salt-manager.sh status   - Check status"
echo "  ./salt-manager.sh test     - Run tests"
echo "  ./salt-manager.sh help     - Show all commands"

echo ""
echo "🚀 Ready to start! Run: ./salt-manager.sh start"
