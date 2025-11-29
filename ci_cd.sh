# AI Infra CI 函数
function ai_ci {
    local project_dir="/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix"
    
    cd "$project_dir" || {
        echo "错误: 无法进入目录 $project_dir"
        return 1
    }
    
    echo "开始构建..."
    ./build.sh all || return 1
    
    echo "拉取镜像..."
    ./build.sh pull-all || return 1
    
    echo "启动服务..."
    ./build.sh start-all || return 1
    
    echo "AI Infra CI 完成"
}

# AI Infra CD 函数
function ai_cd {
    local project_dir="/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix"
    local commit_msg="${1:-$(date '+%Y-%m-%d %H:%M:%S') update}"
    
    cd "$project_dir" || {
        echo "错误: 无法进入目录 $project_dir"
        return 1
    }
    
    # 检查是否有变更
    if git diff-index --quiet HEAD -- 2>/dev/null; then
        echo "没有文件变更，跳过提交"
        return 0
    fi
    
    git add . || return 1
    git commit -m "$commit_msg" || return 1
    
    echo "推送到 origin..."
    git push -u origin || return 1
    
    echo "推送到 gitee..."
    git push -u gitee || return 1
    
    echo "AI Infra CD 完成: $commit_msg"
}