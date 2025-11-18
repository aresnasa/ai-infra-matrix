# 跨平台兼容性修复文档

## 问题背景

开发环境使用macOS (BSD工具链)，生产环境使用Linux (GNU工具链)，存在以下兼容性差异：

- **生产环境**: GNU bash 5.1.16(1)-release (x86_64-pc-linux-gnu)
- **开发环境**: macOS zsh + BSD工具链

## 主要修复

### 1. sed命令兼容性

**问题**: macOS的sed需要备份后缀，Linux的sed不需要

**解决方案**: 添加操作系统检测，根据不同系统使用不同的sed语法

```bash
# macOS
sed -i.bak "pattern" file

# Linux  
sed -i "pattern" file
```

### 2. 操作系统检测功能

添加了`detect_os()`函数自动检测操作系统：

```bash
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macOS"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "Linux"
    else
        echo "Other"
    fi
}
```

### 3. 备份文件清理

只在macOS上清理`.bak`备份文件：

```bash
if [[ "$OS_TYPE" == "macOS" ]]; then
    rm -f "$output_file.bak"
fi
```

## 修复的功能模块

### 1. 生产环境配置生成
- ✅ 镜像registry路径替换 (`s|image: ai-infra-|image: ${registry}/ai-infra-|g`)
- ✅ 镜像标签更新 (`s|\${IMAGE_TAG}|${tag}|g`)
- ✅ LDAP服务移除的sed操作
- ✅ 备份文件清理

### 2. 环境变量处理
- ✅ 正确处理`${IMAGE_TAG}`变量替换
- ✅ 处理带默认值的变量`${IMAGE_TAG:-v0.0.3.3}`

## 验证结果

### 开发环境 (macOS)
```bash
./build.sh prod-generate registry.test v1.1-test
# ✅ 成功生成，OS检测为macOS
# ✅ 正确更新registry路径
# ✅ 正确更新镜像标签
# ✅ docker-compose配置验证通过
```

### 生产环境 (Linux)
脚本现在兼容GNU bash 5.1.16，使用Linux版本的sed语法。

## 兼容的bash特性

脚本使用的所有特性都兼容GNU bash 5.1.16：

- ✅ `[[ ]]` 条件测试
- ✅ `$( )` 命令替换
- ✅ `${var}` 变量展开
- ✅ 数组操作 `"${array[@]}"`
- ✅ `local` 变量声明
- ✅ `set -e` 错误处理

## 使用方法

### 生产环境部署
```bash
# 生成生产环境配置
./build.sh prod-generate <内部registry地址> <版本标签>

# 示例
./build.sh prod-generate registry.company.com v1.0.0
```

### 兼容性检查
脚本会自动显示当前操作系统：
```
[INFO] 更新镜像registry路径... (OS: Linux)
```

## 注意事项

1. **sed版本差异**: 脚本自动处理macOS BSD sed和GNU sed的差异
2. **文件权限**: Linux环境确保脚本有执行权限 `chmod +x build.sh`
3. **依赖工具**: 确保生产环境有必要的工具：
   - docker
   - docker-compose
   - python3 (可选，用于YAML处理)
   - awk, grep, sed (标准工具)

## 测试验证

### 镜像路径验证
```bash
grep "registry.test" docker-compose.prod.yml
# 应显示所有AI服务镜像都使用了指定的registry
```

### 标签验证
```bash  
grep "v1.1-test" docker-compose.prod.yml
# 应显示所有镜像都使用了指定的标签
```

### 配置文件验证
```bash
docker-compose -f docker-compose.prod.yml config
# 应该成功验证配置而不报错
```
