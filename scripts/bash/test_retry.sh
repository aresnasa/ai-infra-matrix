#!/bin/bash
source build.sh
echo "Testing batch_download_base_images function with retry..."
# 只测试前3个镜像来验证重试机制
batch_download_base_images 2>&1 | grep -E "(下载|重试|成功|失败)" | head -20
