# 空目录Git跟踪问题修复

## 问题描述

在构建过程中，某些空目录（如 `src/shared/debug`）是必需的，但Git默认不跟踪空目录，导致在克隆仓库后这些目录缺失，影响构建过程。

## 解决方案

### 方案1：添加.gitkeep文件

为以下必需的空目录添加了`.gitkeep`文件：

- `src/shared/debug/.gitkeep` - 共享调试目录
- `src/frontend/debug/.gitkeep` - 前端调试目录  
- `src/backend/uploads/.gitkeep` - 后端上传目录
- `src/backend/outputs/.gitkeep` - 后端输出目录

### 方案2：build.sh自动创建

在`build.sh`脚本中添加了`ensure_build_directories()`函数：

```bash
# 创建构建所需的目录
ensure_build_directories() {
    local required_dirs=(
        "$SCRIPT_DIR/src/shared/debug"
        "$SCRIPT_DIR/src/frontend/debug" 
        "$SCRIPT_DIR/src/backend/uploads"
        "$SCRIPT_DIR/src/backend/outputs"
    )
    
    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            print_info "创建构建目录: $dir"
            mkdir -p "$dir"
        fi
    done
}
```

该函数在`build_all_images()`开始时自动调用，确保构建所需的目录存在。

## 验证

1. **Git跟踪验证**：
   ```bash
   find src -name ".gitkeep" -type f
   ```

2. **自动创建验证**：
   ```bash
   rm -rf src/shared/debug
   ./build.sh build v1.0.0
   # 会看到: [INFO] 创建构建目录: .../src/shared/debug
   ```

3. **空目录检查**：
   ```bash
   find src -type d -empty
   # 应该没有输出，表示所有目录都有内容
   ```

## 效果

- ✅ Git能正确跟踪所有必需的目录
- ✅ 克隆仓库后构建不会因缺失目录而失败
- ✅ build.sh具备自动修复能力
- ✅ 双重保障确保构建环境完整性

现在无论是新克隆的仓库还是缺失目录的环境，构建过程都能正常进行。
