# build.sh 脚本优化总结

## 目标
修改 build.sh 脚本，避免在本地镜像已存在时重复下载和构建，提升构建效率。

## 实现的功能

### 1. 全局 --force 选项
- **位置**: 命令行参数的第一个位置
- **用法**: `./build.sh --force <命令> [参数...]`
- **功能**: 强制重新构建/下载，忽略本地存在的镜像

### 2. 镜像存在检查
- **实现方式**: 使用 `docker image inspect <镜像名> >/dev/null 2>&1` 检查镜像是否存在
- **覆盖范围**: 
  - 源码服务构建 (`build_service()`)
  - 依赖镜像拉取 (`pull_and_tag_dependencies()`)

### 3. 智能跳过逻辑
- **正常模式**: 如果镜像存在，显示跳过信息并继续下一个任务
- **强制模式**: 忽略镜像存在检查，强制执行构建/拉取操作

## 代码修改详情

### 1. 全局变量添加
```bash
# 在脚本开头添加
FORCE_REBUILD=false
```

### 2. 主函数参数处理
```bash
main() {
    # 预处理命令行参数，检查 --force 标志
    local args=()
    for arg in "$@"; do
        if [[ "$arg" == "--force" ]]; then
            FORCE_REBUILD=true
            print_info "启用强制重新构建模式"
        else
            args+=("$arg")
        fi
    done
    
    # 重新设置位置参数
    set -- "${args[@]}"
    # ... 其余代码
}
```

### 3. build_service() 函数优化
```bash
build_service() {
    # ... 原有代码
    
    # 检查镜像是否已存在 (除非强制重建)
    if [[ "$FORCE_REBUILD" != "true" ]] && docker image inspect "$local_image_name" >/dev/null 2>&1; then
        print_success "  ✓ 镜像已存在，跳过构建: $local_image_name"
        return 0
    fi
    
    # ... 原有构建逻辑
}
```

### 4. pull_and_tag_dependencies() 函数优化
```bash
pull_and_tag_dependencies() {
    # ... 循环处理每个依赖镜像
    
    # 检查目标镜像是否已存在
    if [[ "$FORCE_REBUILD" != "true" ]] && docker image inspect "$target_image" >/dev/null 2>&1; then
        print_success "  ✓ 镜像已存在，跳过: $target_image"
        ((success_count++))
        continue
    fi
    
    # ... 原有拉取和标记逻辑
}
```

### 5. 帮助文档更新
- 添加全局选项说明
- 增加 `--force` 选项的使用示例
- 更新用法格式为 `./build.sh [--force] <命令> [参数...]`

## 使用示例

### 正常模式（镜像存在检查）
```bash
# 构建单个服务 - 跳过已存在的镜像
./build.sh build backend

# 构建所有服务 - 跳过已存在的镜像
./build.sh build-all v0.3.5

# 拉取依赖镜像 - 跳过已存在的镜像
./build.sh deps-pull aiharbor.msxf.local/aihpc
```

### 强制模式（忽略镜像存在）
```bash
# 强制重新构建单个服务
./build.sh --force build backend

# 强制重新构建所有服务
./build.sh --force build-all v0.3.5

# 强制重新拉取依赖镜像
./build.sh --force deps-pull aiharbor.msxf.local/aihpc
```

## 测试结果

✅ **正常构建模式**: 成功跳过已存在镜像
```
[SUCCESS]   ✓ 镜像已存在，跳过构建: ai-infra-nginx:v0.3.5
```

✅ **强制构建模式**: 成功强制重新构建
```
[INFO] 启用强制重新构建模式
[SUCCESS] ✓ 构建成功: ai-infra-nginx:v0.3.5
```

✅ **依赖镜像优化**: 成功跳过已存在的依赖镜像
```
[SUCCESS]   ✓ 镜像已存在，跳过: aiharbor.msxf.local/library/postgres:v0.3.5
```

## 优化效果

1. **节省时间**: 避免重复构建已存在的镜像，显著减少构建时间
2. **节省带宽**: 跳过已下载的依赖镜像，减少网络流量
3. **灵活控制**: 通过 `--force` 选项提供强制重建的能力
4. **用户体验**: 清晰的输出信息，用户可以看到哪些镜像被跳过
5. **向后兼容**: 所有现有命令使用方式保持不变

## 技术要点

- 使用 `docker image inspect` 检查镜像存在性，比其他方法更可靠
- 全局 `FORCE_REBUILD` 变量控制所有相关功能
- 命令行参数预处理，确保 `--force` 可以在任何位置使用
- 保持原有函数结构，最小化代码修改
- 错误处理和输出格式保持一致

## 兼容性

- ✅ 与现有所有命令兼容
- ✅ 支持所有镜像标签和注册表格式
- ✅ 保持原有错误处理机制
- ✅ Docker Compose 兼容性检查不受影响
