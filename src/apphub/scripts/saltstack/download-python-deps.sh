#!/bin/bash
# =============================================================================
# SaltStack Python Dependencies Download Script for AppHub
# 下载 SaltStack 所需的 Python 依赖包到 AppHub
# 
# 这些包用于解决新版 Python (3.12+) 中缺失的模块问题：
# - looseversion: Python 3.12+ 移除了 distutils.version.LooseVersion
# - packaging: looseversion 的依赖 (通常已预装)
#
# 环境变量:
#   PYPI_INDEX_URL - PyPI 镜像地址 (默认: https://mirrors.aliyun.com/pypi/simple/)
#   OUTPUT_DIR     - 输出目录 (默认: /usr/share/nginx/html/pkgs/python-deps)
# =============================================================================

set -e

# 配置
OUTPUT_DIR="${OUTPUT_DIR:-/usr/share/nginx/html/pkgs/python-deps}"

# PyPI 镜像配置
PYPI_INDEX_URL="${PYPI_INDEX_URL:-https://mirrors.aliyun.com/pypi/simple/}"

echo "📦 Downloading SaltStack Python dependencies..."
echo "  PyPI Index URL: ${PYPI_INDEX_URL}"
echo "  Output Dir: ${OUTPUT_DIR}"

# 创建输出目录
mkdir -p "${OUTPUT_DIR}"

# 定义要下载的包
PACKAGES="looseversion packaging"

# 使用 pip download 下载包
download_with_pip() {
    echo "  📥 Downloading packages using pip..."
    
    if command -v pip3 >/dev/null 2>&1; then
        PIP_CMD="pip3"
    elif command -v pip >/dev/null 2>&1; then
        PIP_CMD="pip"
    else
        echo "  ❌ pip not found"
        return 1
    fi
    
    # 下载到输出目录
    $PIP_CMD download \
        --index-url "${PYPI_INDEX_URL}" \
        --dest "${OUTPUT_DIR}" \
        --no-deps \
        looseversion packaging 2>&1 || {
            echo "  ⚠️  pip download from ${PYPI_INDEX_URL} failed, trying default PyPI..."
            $PIP_CMD download \
                --dest "${OUTPUT_DIR}" \
                --no-deps \
                looseversion packaging 2>&1 || return 1
        }
    
    return 0
}

# 执行下载
if download_with_pip; then
    echo "  ✓ Packages downloaded successfully"
else
    echo "  ❌ Failed to download packages"
    exit 1
fi

# 生成包清单文件
cat > "${OUTPUT_DIR}/packages.json" << EOF
{
    "description": "Python dependencies for SaltStack on Python 3.12+",
    "packages": ["looseversion", "packaging"],
    "updated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

# 生成简单安装脚本
cat > "${OUTPUT_DIR}/install-deps.sh" << 'INSTALL_EOF'
#!/bin/bash
# =============================================================================
# 安装 SaltStack Python 依赖
# 使用方法: curl -fsSL http://apphub/pkgs/python-deps/install-deps.sh | bash
# =============================================================================

set -e

APPHUB_URL="${APPHUB_URL:-http://localhost:8081}"
BASE_URL="${APPHUB_URL}/pkgs/python-deps"

echo "📦 Installing SaltStack Python dependencies from AppHub..."

# 检测 Python 版本
PYTHON_CMD=""
for cmd in python3 python; do
    if command -v $cmd >/dev/null 2>&1; then
        PYTHON_CMD=$cmd
        break
    fi
done

if [ -z "$PYTHON_CMD" ]; then
    echo "❌ Python not found"
    exit 1
fi

PY_VERSION=$($PYTHON_CMD -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')")
echo "  Python version: ${PY_VERSION}"

# 创建临时目录
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT
cd "$TMP_DIR"

# 下载并安装 wheel 包
for pkg in looseversion packaging; do
    echo "  📥 Downloading ${pkg}..."
    # 列出目录获取实际文件名
    whl_file=$(curl -fsSL "${BASE_URL}/" 2>/dev/null | grep -oE "${pkg}[^\"<>]+\.whl" | head -1)
    if [ -n "$whl_file" ]; then
        curl -fsSL -O "${BASE_URL}/${whl_file}" 2>/dev/null || \
        wget -q "${BASE_URL}/${whl_file}" 2>/dev/null || true
    fi
done

# 安装下载的包
if ls *.whl >/dev/null 2>&1; then
    echo "  📦 Installing wheel packages..."
    pip3 install *.whl --break-system-packages 2>/dev/null || \
    pip3 install *.whl 2>/dev/null || \
    $PYTHON_CMD -m pip install *.whl --break-system-packages 2>/dev/null || \
    $PYTHON_CMD -m pip install *.whl 2>/dev/null || true
fi

# 验证安装
if $PYTHON_CMD -c "import looseversion" 2>/dev/null; then
    echo "✓ looseversion installed successfully"
else
    echo "⚠️  looseversion installation may have failed"
fi

echo "✓ Done"
INSTALL_EOF
chmod +x "${OUTPUT_DIR}/install-deps.sh"

# 统计结果
total_downloaded=$(ls -1 "${OUTPUT_DIR}"/*.whl 2>/dev/null | wc -l || echo 0)

echo ""
echo "✓ SaltStack Python dependencies downloaded: ${total_downloaded} packages"
echo "  Location: ${OUTPUT_DIR}"
ls -la "${OUTPUT_DIR}"
