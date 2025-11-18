#!/bin/bash
source build.sh
echo "Testing batch_download_base_images with retry mechanism..."
# 只测试一个镜像来快速验证重试
echo "postgres:15-alpine redis:7-alpine" | while read image; do
    if [[ -n "$image" ]]; then
        echo "Testing retry for: $image"
        retry_pull_image "$image" 3
        echo "Result: $?"
        break
    fi
done
