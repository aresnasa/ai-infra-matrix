---
description: 'Describe what this custom agent does and when to use it.'
tools: [playwright/*,github/*,oraios/serena/check_onboarding_performed]
---
1. 每次执行相关命令先读取.env 文件中的配置，然后再执行

2. 使用 build.sh 进行构建。

3. 这是两个 ci/cd 的函数，基于他们进行构建和启动
function ai_ci {
    local project_dir="/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix"

    cd "$project_dir" || {
        echo "错误: 无法进入目录 $project_dir"
        return 1
    }

    echo "开始构建..."
    ./build.sh all --parallel || return 1

    #echo "拉取镜像..."
    ./build.sh pull-all || return 1

    echo "启动服务..."
    docker-compose down || return 1
    ./build.sh start-all || return 1

    echo "AI Infra CI 完成"
}

function ai_cd {
    local project_dir="/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix"
    local custom_msg="$1"

    cd "$project_dir" || {
        echo "错误: 无法进入目录 $project_dir"
        return 1
    }

    # 检查是否有变更
    if git diff-index --quiet HEAD -- 2>/dev/null; then
        echo "没有文件变更，跳过提交"
        return 0
    fi

    # 生成变更描述
    local changed_files=$(git diff --name-only HEAD 2>/dev/null)
    local desc=""

    # 根据变更文件路径生成描述
    echo "$changed_files" | grep -q "src/frontend" && desc="${desc}frontend,"
    echo "$changed_files" | grep -q "src/backend" && desc="${desc}backend,"
    echo "$changed_files" | grep -q "src/saltstack" && desc="${desc}saltstack,"
    echo "$changed_files" | grep -q "src/apphub" && desc="${desc}apphub,"
    echo "$changed_files" | grep -q "src/nginx" && desc="${desc}nginx,"
    echo "$changed_files" | grep -q "src/nightingale" && desc="${desc}nightingale,"
    echo "$changed_files" | grep -q "src/jupyterhub" && desc="${desc}jupyterhub,"
    echo "$changed_files" | grep -q "build.sh" && desc="${desc}build,"
    echo "$changed_files" | grep -q "docker-compose" && desc="${desc}compose,"
    echo "$changed_files" | grep -q ".env" && desc="${desc}env,"
    echo "$changed_files" | grep -q "helm/" && desc="${desc}helm,"
    echo "$changed_files" | grep -q "scripts/" && desc="${desc}scripts,"
    echo "$changed_files" | grep -q "docs" && desc="${desc}docs,"
    echo "$changed_files" | grep -q "test/" && desc="${desc}test,"

    # 移除末尾逗号
    desc="${desc%,}"

    # 统计变更数量
    local file_count=$(echo "$changed_files" | wc -l | tr -d ' ')

    # 构建 commit 信息
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local commit_msg=""

    if [[ -n "$custom_msg" ]]; then
        # 用户提供了自定义信息
        commit_msg="$timestamp [$desc] $custom_msg"
    elif [[ -n "$desc" ]]; then
        # 自动生成描述
        commit_msg="$timestamp [$desc] ${file_count} files changed"
    else
        commit_msg="$timestamp update"
    fi

    echo "📝 变更文件 ($file_count):"
    echo "$changed_files" | head -10
    [[ $file_count -gt 10 ]] && echo "  ... 及其他 $((file_count - 10)) 个文件"
    echo ""
    echo "📦 Commit: $commit_msg"
    echo ""

    git add . || return 1
    git commit -m "$commit_msg" || return 1

    echo "推送到 origin..."
    git push -u origin || return 1

    echo "推送到 gitee..."
    git push -u gitee || return 1

    echo "✅ AI Infra CD 完成: $commit_msg"
}

4. 所有的组件都需要支持中英文，需要引用翻译组件。

5. 所有的 TASK 任务生成的 markdown 文件都要放在private-docs/docs-all 中

8. 除了 README.md 和README_zh_CN.md 外的 markdown 文档，都要放入private-docs/docs-all/中

9. 我本地的 docker 使用了代理http://127.0.0.1：7890去访问 dockerhub，这里需要一个检查机制保证 docker 能够获取元数据，同时本地的 docker 已经使用了如下镜像加速配置配置：
{
  "builder": {
    "gc": {
      "defaultKeepStorage": "60GB",
      "enabled": true
    }
  },
  "experimental": false,
  "insecure-registries": [
    "d9qvoql50lvykf.xuanyuan.run",
    "d9qvoql50lvykf-ghcr.xuanyuan.run",
    "d9qvoql50lvykf-k8s.xuanyuan.run",
    "nexus-docker.zs.shaipower.online"
  ],
  "registry-mirrors": [
    "https://d9qvoql50lvykf.xuanyuan.run",
    "https://d9qvoql50lvykf-ghcr.xuanyuan.run",
    "https://d9qvoql50lvykf-k8s.xuanyuan.run"
  ]
}

10. 除了build.sh 外的 shell 脚本，都要放入scripts/中