#!/bin/bash

# ==================================================================================
# Backend 环境变量配置迁移脚本
# ==================================================================================
# 用途：将 src/backend/.env 的配置迁移到项目根目录 .env
# 作者：AI Infrastructure Team
# 版本：v0.3.8
# ==================================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_ENV="$PROJECT_ROOT/src/backend/.env"
ROOT_ENV="$PROJECT_ROOT/.env"
ROOT_ENV_EXAMPLE="$PROJECT_ROOT/.env.example"
BACKUP_DIR="$PROJECT_ROOT/backup/env-migration-$(date +%Y%m%d-%H%M%S)"

echo -e "${BLUE}=======================================${NC}"
echo -e "${BLUE}Backend 环境变量配置迁移工具${NC}"
echo -e "${BLUE}=======================================${NC}"
echo

# 检查是否存在旧配置文件
if [ ! -f "$BACKEND_ENV" ]; then
    echo -e "${YELLOW}⚠️  未找到 src/backend/.env 文件${NC}"
    echo "   如果你是新用户，请直接使用根目录的 .env 文件："
    echo "   cp .env.example .env"
    echo
    exit 0
fi

echo -e "${GREEN}✓ 找到旧配置文件：$BACKEND_ENV${NC}"
echo

# 创建备份目录
mkdir -p "$BACKUP_DIR"
echo -e "${BLUE}📦 创建备份目录：$BACKUP_DIR${NC}"

# 备份旧配置
cp "$BACKEND_ENV" "$BACKUP_DIR/backend.env.backup"
echo -e "${GREEN}✓ 已备份旧配置到：$BACKUP_DIR/backend.env.backup${NC}"

# 如果根目录的 .env 存在，也备份
if [ -f "$ROOT_ENV" ]; then
    cp "$ROOT_ENV" "$BACKUP_DIR/root.env.backup"
    echo -e "${GREEN}✓ 已备份根目录 .env 到：$BACKUP_DIR/root.env.backup${NC}"
fi

echo

# 分析旧配置
echo -e "${BLUE}🔍 分析配置差异...${NC}"
echo

# 提取旧配置中的非注释行
OLD_VARS=$(grep -v "^#" "$BACKEND_ENV" | grep -v "^$" | cut -d'=' -f1 || true)

if [ -z "$OLD_VARS" ]; then
    echo -e "${YELLOW}⚠️  旧配置文件为空或只包含注释${NC}"
    exit 0
fi

# 检查根目录 .env 是否存在
if [ ! -f "$ROOT_ENV" ]; then
    echo -e "${YELLOW}⚠️  根目录 .env 不存在，将从 .env.example 创建${NC}"
    if [ -f "$ROOT_ENV_EXAMPLE" ]; then
        cp "$ROOT_ENV_EXAMPLE" "$ROOT_ENV"
        echo -e "${GREEN}✓ 已创建 .env 文件${NC}"
    else
        echo -e "${RED}✗ 错误：.env.example 也不存在！${NC}"
        exit 1
    fi
fi

echo

# 询问是否继续
read -p "是否继续迁移配置？(y/n): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${YELLOW}取消迁移${NC}"
    exit 0
fi

echo

# 迁移配置
echo -e "${BLUE}🚀 开始迁移配置...${NC}"
echo

MIGRATED_COUNT=0
SKIPPED_COUNT=0

while IFS= read -r var_name; do
    # 跳过空行
    [ -z "$var_name" ] && continue
    
    # 从旧配置中获取值
    old_value=$(grep "^${var_name}=" "$BACKEND_ENV" | head -1 | cut -d'=' -f2-)
    
    # 检查根目录 .env 中是否已存在该变量
    if grep -q "^${var_name}=" "$ROOT_ENV"; then
        # 获取现有值
        current_value=$(grep "^${var_name}=" "$ROOT_ENV" | head -1 | cut -d'=' -f2-)
        
        # 如果值不同，询问是否覆盖
        if [ "$old_value" != "$current_value" ]; then
            echo -e "${YELLOW}变量冲突：$var_name${NC}"
            echo "  旧值: $old_value"
            echo "  现值: $current_value"
            read -p "  是否覆盖为旧值？(y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                # 覆盖值
                sed -i.bak "s|^${var_name}=.*|${var_name}=${old_value}|" "$ROOT_ENV"
                echo -e "${GREEN}  ✓ 已覆盖${NC}"
                MIGRATED_COUNT=$((MIGRATED_COUNT + 1))
            else
                echo -e "${BLUE}  - 保持现值${NC}"
                SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
            fi
        else
            echo -e "${BLUE}- $var_name (已存在且相同)${NC}"
            SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
        fi
    else
        # 变量不存在，直接添加到文件末尾
        echo "${var_name}=${old_value}" >> "$ROOT_ENV"
        echo -e "${GREEN}+ $var_name (新增)${NC}"
        MIGRATED_COUNT=$((MIGRATED_COUNT + 1))
    fi
done <<< "$OLD_VARS"

echo
echo -e "${GREEN}=======================================${NC}"
echo -e "${GREEN}迁移完成！${NC}"
echo -e "${GREEN}=======================================${NC}"
echo
echo "统计信息："
echo "  - 迁移/更新: $MIGRATED_COUNT 个变量"
echo "  - 跳过: $SKIPPED_COUNT 个变量"
echo
echo "备份位置："
echo "  $BACKUP_DIR"
echo

# 询问是否删除旧配置
read -p "是否删除旧配置文件 src/backend/.env？(y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm "$BACKEND_ENV"
    echo -e "${GREEN}✓ 已删除 $BACKEND_ENV${NC}"
    echo -e "${BLUE}ℹ️  注意：src/backend/.env.example 已标记为废弃，但保留作为参考${NC}"
else
    echo -e "${YELLOW}⚠️  保留旧配置文件（建议删除）${NC}"
fi

echo
echo -e "${BLUE}=======================================${NC}"
echo -e "${BLUE}下一步操作：${NC}"
echo -e "${BLUE}=======================================${NC}"
echo
echo "1. 检查迁移后的配置："
echo "   vim .env"
echo
echo "2. 重启 backend 服务："
echo "   docker-compose restart backend"
echo
echo "3. 查看服务日志："
echo "   docker-compose logs -f backend"
echo
echo "4. 验证环境变量："
echo "   docker exec ai-infra-backend env | grep -E 'SALT|DB_|REDIS'"
echo
echo -e "${GREEN}✅ 迁移完成！${NC}"
echo
