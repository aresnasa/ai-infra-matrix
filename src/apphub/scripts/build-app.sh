#!/bin/bash
# =============================================================================
# Generic App Builder for AppHub
# 通用应用构建辅助脚本，用于在 Dockerfile 中调用各个应用的构建脚本
# =============================================================================

set -e

APP_NAME=${1:-""}
if [ -z "$APP_NAME" ]; then
    echo "Error: APP_NAME is required"
    echo "Usage: $0 <app-name>"
    exit 1
fi

SCRIPT_DIR="/scripts/${APP_NAME}"
BUILD_SCRIPT="${SCRIPT_DIR}/${APP_NAME}-build.sh"

if [ ! -f "$BUILD_SCRIPT" ]; then
    echo "Error: Build script not found: $BUILD_SCRIPT"
    exit 1
fi

echo "==================================================================="
echo "  Building: ${APP_NAME}"
echo "  Script: ${BUILD_SCRIPT}"
echo "==================================================================="
echo ""

# 设置通用环境变量
export SCRIPT_DIR
export BUILD_DIR=${BUILD_DIR:-"/build"}
export OUTPUT_DIR=${OUTPUT_DIR:-"/out"}

# 执行应用特定的构建脚本
bash "$BUILD_SCRIPT"

echo ""
echo "==================================================================="
echo "  ✓ Build completed: ${APP_NAME}"
echo "==================================================================="
