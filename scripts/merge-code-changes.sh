#!/bin/bash
# 智能合并 commit 1354863 的代码更改（排除dev-md.md）
# 作者: AI Infrastructure Team  
# 日期: 2025-10-20

set -e

COMMIT="1354863"

echo "============================================"
echo "智能合并 commit ${COMMIT} 的代码更改"
echo "============================================"

# 获取该commit修改的所有文件（排除dev-md.md）
echo ""
echo "步骤 1: 分析需要合并的文件..."
FILES=$(git diff-tree --no-commit-id --name-only -r ${COMMIT} | grep -v "^dev-md.md$")

echo "需要合并的文件列表:"
echo "${FILES}" | head -20
echo "..."
echo "总计: $(echo "${FILES}" | wc -l) 个文件"

# 关键文件列表
KEY_FILES=(
    ".env.example"
    "src/backend/cmd/init/main.go"
    "src/backend/internal/services/ai_providers/factory.go"
    "src/backend/internal/services/ai_service.go"
    "docs/DEEPSEEK_V3.2_UPDATE_SUMMARY.md"
)

echo ""
echo "步骤 2: 检查关键文件差异..."
for file in "${KEY_FILES[@]}"; do
    if git diff-tree --no-commit-id --name-only -r ${COMMIT} | grep -q "^${file}$"; then
        echo "  ✅ ${file} - 有更新"
    else
        echo "  ⚠️  ${file} - 无更新"
    fi
done

echo ""
echo "步骤 3: 创建合并补丁（排除dev-md.md）..."
git format-patch -1 ${COMMIT} --stdout | \
    filterdiff -x 'dev-md.md' > /tmp/merge-${COMMIT}.patch

echo "补丁已创建: /tmp/merge-${COMMIT}.patch"
PATCH_SIZE=$(wc -l < /tmp/merge-${COMMIT}.patch)
echo "补丁大小: ${PATCH_SIZE} 行"

if [ ${PATCH_SIZE} -lt 10 ]; then
    echo "⚠️  警告: 补丁文件太小，可能没有实际更改"
    exit 1
fi

echo ""
echo "步骤 4: 应用补丁..."
if git apply --check /tmp/merge-${COMMIT}.patch 2>/dev/null; then
    echo "✅ 补丁可以干净地应用"
    git apply /tmp/merge-${COMMIT}.patch
    echo "✅ 补丁已成功应用"
else
    echo "⚠️  补丁应用有冲突，尝试三路合并..."
    if git apply --3way /tmp/merge-${COMMIT}.patch; then
        echo "✅ 三路合并成功"
    else
        echo "❌ 合并失败，需要手动处理冲突"
        echo ""
        echo "冲突文件:"
        git status --short
        echo ""
        echo "请手动解决冲突后执行:"
        echo "  git add <冲突文件>"
        echo "  git commit -m 'Merge code changes from ${COMMIT}'"
        exit 1
    fi
fi

echo ""
echo "步骤 5: 检查合并结果..."
git status --short

echo ""
echo "============================================"
echo "✅ 代码合并完成！"
echo "============================================"
echo ""
echo "已合并的更改:"
git diff --stat HEAD

echo ""
echo "下一步操作:"
echo "  1. 检查更改: git diff"
echo "  2. 暂存更改: git add ."
echo "  3. 提交更改: git commit -m 'Merge DeepSeek v3.2 updates from ${COMMIT}'"
echo "  4. 或者放弃更改: git reset --hard HEAD"
echo ""
