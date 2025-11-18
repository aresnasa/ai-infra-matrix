#!/bin/bash
# 合并 commit 1354863 的更改到当前分支
# 作者: AI Infrastructure Team
# 日期: 2025-10-20

set -e

COMMIT="1354863"
BRANCH_NAME="merge-${COMMIT}-$(date +%Y%m%d-%H%M%S)"

echo "============================================"
echo "合并 commit ${COMMIT} 到当前分支"
echo "============================================"

# 获取当前分支名
CURRENT_BRANCH=$(git branch --show-current)
echo "当前分支: ${CURRENT_BRANCH}"

# 创建新分支进行合并
echo ""
echo "步骤 1: 创建新分支 ${BRANCH_NAME}"
git checkout -b "${BRANCH_NAME}"

echo ""
echo "步骤 2: Cherry-pick commit ${COMMIT}"
if git cherry-pick "${COMMIT}" 2>/dev/null; then
    echo "✅ Cherry-pick 成功！"
else
    echo "⚠️  Cherry-pick 有冲突，需要手动解决"
    echo ""
    echo "冲突文件列表:"
    git status --short | grep "^UU\|^AA\|^DD"
    echo ""
    echo "请手动解决冲突后执行:"
    echo "  git add <冲突文件>"
    echo "  git cherry-pick --continue"
    echo ""
    echo "或者放弃合并:"
    echo "  git cherry-pick --abort"
    echo "  git checkout ${CURRENT_BRANCH}"
    echo "  git branch -D ${BRANCH_NAME}"
    exit 1
fi

echo ""
echo "步骤 3: 检查合并结果"
echo ""
echo "新增文件:"
git diff --name-status "${CURRENT_BRANCH}" "${BRANCH_NAME}" | grep "^A" | wc -l
echo ""
echo "修改文件:"
git diff --name-status "${CURRENT_BRANCH}" "${BRANCH_NAME}" | grep "^M" | wc -l
echo ""
echo "删除文件:"
git diff --name-status "${CURRENT_BRANCH}" "${BRANCH_NAME}" | grep "^D" | wc -l

echo ""
echo "============================================"
echo "✅ 合并完成！"
echo "============================================"
echo ""
echo "新分支: ${BRANCH_NAME}"
echo ""
echo "下一步操作:"
echo "  1. 检查代码: git diff ${CURRENT_BRANCH}"
echo "  2. 运行测试: ./build.sh test"
echo "  3. 如果满意，合并到原分支:"
echo "     git checkout ${CURRENT_BRANCH}"
echo "     git merge ${BRANCH_NAME}"
echo "  4. 或者放弃合并:"
echo "     git checkout ${CURRENT_BRANCH}"
echo "     git branch -D ${BRANCH_NAME}"
echo ""
