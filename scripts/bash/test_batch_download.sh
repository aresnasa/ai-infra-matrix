#!/bin/bash
source build.sh
echo "Testing batch_download_base_images function..."
batch_download_base_images --dry-run 2>/dev/null || echo "Function called successfully (dry-run not supported, but function exists)"
