#!/usr/bin/env bash
# 测试版本参数生成功能
# 这个脚本不会影响终端，只是测试函数功能

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"

echo "=========================================="
echo "测试版本管理系统"
echo "=========================================="
echo ""

# 加载环境变量的函数（从 build.sh 复制）
load_env_file() {
    local env_file="${ENV_FILE}"
    
    # 如果 .env 不存在，尝试使用 .env.example
    if [[ ! -f "$env_file" ]]; then
        env_file="${SCRIPT_DIR}/.env.example"
        echo "⚠️  .env 文件不存在，使用 .env.example"
    fi
    
    if [[ ! -f "$env_file" ]]; then
        echo "❌ 找不到环境文件: $env_file"
        return 1
    fi
    
    echo "📂 加载环境文件: $env_file"
    
    # 读取环境文件（跳过注释和空行）
    while IFS='=' read -r key value; do
        # 跳过注释和空行
        [[ "$key" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$key" ]] && continue
        
        # 移除前后空格
        key=$(echo "$key" | xargs)
        value=$(echo "$value" | xargs)
        
        # 移除值两边的引号（如果有）
        value="${value%\"}"
        value="${value#\"}"
        value="${value%\'}"
        value="${value#\'}"
        
        # 导出环境变量（不覆盖已有的）
        if [[ -z "${!key:-}" ]]; then
            export "$key=$value"
        fi
    done < <(grep -v '^[[:space:]]*#' "$env_file" | grep -v '^[[:space:]]*$')
    
    echo "✓ 环境变量加载完成"
    echo ""
}

# 获取版本构建参数的函数（从 build.sh 复制）
get_version_build_args() {
    local service="$1"
    local build_args=""
    
    # 基础镜像版本参数（所有服务通用）
    [[ -n "${GOLANG_VERSION:-}" ]] && build_args+=" --build-arg GOLANG_VERSION=${GOLANG_VERSION}"
    [[ -n "${GOLANG_ALPINE_VERSION:-}" ]] && build_args+=" --build-arg GOLANG_ALPINE_VERSION=${GOLANG_ALPINE_VERSION}"
    [[ -n "${NODE_VERSION:-}" ]] && build_args+=" --build-arg NODE_VERSION=${NODE_VERSION}"
    [[ -n "${NODE_ALPINE_VERSION:-}" ]] && build_args+=" --build-arg NODE_ALPINE_VERSION=${NODE_ALPINE_VERSION}"
    [[ -n "${PYTHON_VERSION:-}" ]] && build_args+=" --build-arg PYTHON_VERSION=${PYTHON_VERSION}"
    [[ -n "${PYTHON_ALPINE_VERSION:-}" ]] && build_args+=" --build-arg PYTHON_ALPINE_VERSION=${PYTHON_ALPINE_VERSION}"
    [[ -n "${UBUNTU_VERSION:-}" ]] && build_args+=" --build-arg UBUNTU_VERSION=${UBUNTU_VERSION}"
    [[ -n "${ROCKYLINUX_VERSION:-}" ]] && build_args+=" --build-arg ROCKYLINUX_VERSION=${ROCKYLINUX_VERSION}"
    [[ -n "${NGINX_VERSION:-}" ]] && build_args+=" --build-arg NGINX_VERSION=${NGINX_VERSION}"
    [[ -n "${NGINX_ALPINE_VERSION:-}" ]] && build_args+=" --build-arg NGINX_ALPINE_VERSION=${NGINX_ALPINE_VERSION}"
    [[ -n "${HAPROXY_VERSION:-}" ]] && build_args+=" --build-arg HAPROXY_VERSION=${HAPROXY_VERSION}"
    [[ -n "${JUPYTER_BASE_NOTEBOOK_VERSION:-}" ]] && build_args+=" --build-arg JUPYTER_BASE_NOTEBOOK_VERSION=${JUPYTER_BASE_NOTEBOOK_VERSION}"
    
    # 应用组件版本参数
    [[ -n "${GITEA_VERSION:-}" ]] && build_args+=" --build-arg GITEA_VERSION=${GITEA_VERSION}"
    [[ -n "${SALTSTACK_VERSION:-}" ]] && build_args+=" --build-arg SALTSTACK_VERSION=${SALTSTACK_VERSION}"
    [[ -n "${SLURM_VERSION:-}" ]] && build_args+=" --build-arg SLURM_VERSION=${SLURM_VERSION}"
    [[ -n "${CATEGRAF_VERSION:-}" ]] && build_args+=" --build-arg CATEGRAF_VERSION=${CATEGRAF_VERSION}"
    [[ -n "${SINGULARITY_VERSION:-}" ]] && build_args+=" --build-arg SINGULARITY_VERSION=${SINGULARITY_VERSION}"
    
    # 依赖工具版本参数
    [[ -n "${PIP_VERSION:-}" ]] && build_args+=" --build-arg PIP_VERSION=${PIP_VERSION}"
    [[ -n "${JUPYTERHUB_VERSION:-}" ]] && build_args+=" --build-arg JUPYTERHUB_VERSION=${JUPYTERHUB_VERSION}"
    [[ -n "${GO_PROXY:-}" ]] && build_args+=" --build-arg GO_PROXY=${GO_PROXY}"
    [[ -n "${PYPI_INDEX_URL:-}" ]] && build_args+=" --build-arg PYPI_INDEX_URL=${PYPI_INDEX_URL}"
    [[ -n "${NPM_REGISTRY:-}" ]] && build_args+=" --build-arg NPM_REGISTRY=${NPM_REGISTRY}"
    
    # 服务特定的版本参数
    case "$service" in
        gitea)
            [[ -n "${GITEA_VERSION:-}" ]] && build_args+=" --build-arg GITEA_IMAGE=gitea/gitea:${GITEA_VERSION}"
            ;;
        saltstack)
            [[ -n "${SALTSTACK_VERSION:-}" ]] && build_args+=" --build-arg SALT_VERSION=${SALTSTACK_VERSION}"
            ;;
        slurm-master)
            [[ -n "${SLURM_VERSION:-}" ]] && build_args+=" --build-arg SLURM_PKG_VERSION=${SLURM_VERSION}"
            ;;
        apphub)
            [[ -n "${SLURM_VERSION:-}" ]] && build_args+=" --build-arg SLURM_VERSION=${SLURM_VERSION}"
            [[ -n "${CATEGRAF_VERSION:-}" ]] && build_args+=" --build-arg CATEGRAF_VERSION=${CATEGRAF_VERSION}"
            [[ -n "${SINGULARITY_VERSION:-}" ]] && build_args+=" --build-arg SINGULARITY_VERSION=${SINGULARITY_VERSION}"
            ;;
    esac
    
    echo "$build_args"
}

# 执行测试
load_env_file

echo "=========================================="
echo "测试各服务的版本参数生成"
echo "=========================================="
echo ""

# 测试所有主要服务
services=("backend" "frontend" "gitea" "jupyterhub" "saltstack" "slurm-master" "nginx" "singleuser" "proxy" "apphub")

for service in "${services[@]}"; do
    echo "📦 服务: $service"
    echo "---"
    args=$(get_version_build_args "$service")
    if [[ -n "$args" ]]; then
        # 格式化输出，每个参数一行
        echo "$args" | tr ' ' '\n' | grep -v '^$' | sed 's/^/   /'
    else
        echo "   (无版本参数)"
    fi
    echo ""
done

echo "=========================================="
echo "测试完成！"
echo "=========================================="
echo ""
echo "💡 提示：要测试实际构建，运行："
echo "   ./build.sh build backend --force"
