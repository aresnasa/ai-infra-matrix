# Gitea 静态资源路径修复报告

## 修复日期
2025年10月12日

## 问题描述

访问 Gitea 页面时出现静态资源加载失败错误：

```
Failed to load asset files from http://192.168.0.200:8080/assets/assets/js/index.js?v=1.24.6
Please make sure the asset files can be accessed.
```

**问题分析**：
- URL 路径中出现重复的 `/assets/` → `/assets/assets/js/index.js`
- 正确的路径应该是 `/gitea/assets/js/index.js`

## 根本原因

在 `.env` 文件中，`STATIC_URL_PREFIX` 被错误地设置为 `/assets`：

```bash
# 错误配置
SUBURL=/gitea
STATIC_URL_PREFIX=/assets  # ❌ 错误
```

**导致的问题**：
- Gitea 在渲染静态资源路径时，会将 `STATIC_URL_PREFIX` 与 `/assets/` 拼接
- 结果变成：`/assets` + `/assets/js/index.js` = `/assets/assets/js/index.js` ❌

## Gitea 静态资源路径机制

Gitea 在子路径部署时的静态资源路径构建逻辑：

1. **正确配置**（STATIC_URL_PREFIX = SUBURL）：
   ```bash
   SUBURL=/gitea
   STATIC_URL_PREFIX=/gitea
   ```
   - 生成的路径：`/gitea/assets/js/index.js` ✅
   - Gitea 模板会自动在 STATIC_URL_PREFIX 后添加 `/assets/`

2. **错误配置**（STATIC_URL_PREFIX = /assets）：
   ```bash
   SUBURL=/gitea
   STATIC_URL_PREFIX=/assets  # ❌
   ```
   - 生成的路径：`/assets/assets/js/index.js` ❌
   - 导致路径重复，资源404

## 修复方案

### 1. 修复 `.env` 文件

修改 `.env` 文件第278行：

```bash
# 修复前
STATIC_URL_PREFIX=/assets

# 修复后
STATIC_URL_PREFIX=/gitea
```

### 2. 更新 `build.sh` 自动配置逻辑

在 `generate_or_update_env_file()` 函数中添加 `STATIC_URL_PREFIX` 的自动设置：

```bash
# Gitea 配置
update_env_variable "ROOT_URL" "${base_url}/gitea/"
update_env_variable "STATIC_URL_PREFIX" "/gitea"  # 新增
```

这样在执行 `bash build.sh build-all` 或其他命令时，会自动设置正确的 `STATIC_URL_PREFIX`。

### 3. gitea-entrypoint.sh 配置验证

`src/gitea/gitea-entrypoint.sh` 中已经有正确的逻辑：

```bash
# Ensure STATIC_URL_PREFIX aligns with proxy subpath 
# (use '/gitea' so templates adding '/assets' don't double it)
if grep -q '^STATIC_URL_PREFIX *=.*' "$APP_INI"; then
  sed -i "s#^STATIC_URL_PREFIX *=.*#STATIC_URL_PREFIX = ${STATIC_URL_PREFIX:-/gitea}#" "$APP_INI"
fi
```

注释说得很清楚：**use '/gitea' so templates adding '/assets' don't double it**

## 验证方法

### 方法1: 使用验证脚本

```bash
bash test-gitea-assets.sh
```

验证项目：
- ✅ `.env` 文件中 `STATIC_URL_PREFIX` 配置
- ✅ `STATIC_URL_PREFIX` 与 `SUBURL` 是否一致
- ✅ Gitea 容器内 `app.ini` 配置

### 方法2: 手动验证

```bash
# 检查 .env 配置
grep "STATIC_URL_PREFIX" .env

# 检查容器内配置
docker compose exec gitea grep "STATIC_URL_PREFIX" /data/gitea/conf/app.ini

# 预期输出
STATIC_URL_PREFIX = /gitea
```

## 修复结果

### 修复前
```
URL: http://192.168.0.200:8080/assets/assets/js/index.js ❌
错误: 404 Not Found
```

### 修复后
```
URL: http://192.168.0.200:8080/gitea/assets/js/index.js ✅
状态: 200 OK
```

## 验证测试结果

```
==================================
验证总结
==================================
通过: 3
失败: 0

✓ Gitea 静态资源配置正确！
```

## 应用修复

```bash
# 方式1: 已经修改了 .env，直接重启容器
docker compose restart gitea

# 方式2: 如果 .env 未修改，先修改再重启
sed -i 's|^STATIC_URL_PREFIX=.*|STATIC_URL_PREFIX=/gitea|' .env
docker compose restart gitea

# 方式3: 使用 build.sh 重新生成配置
bash build.sh render-templates all
docker compose restart gitea
```

## 浏览器缓存清理

修复后仍需清理浏览器缓存：

1. **Chrome/Edge**:
   - `Ctrl+Shift+Delete` (Windows/Linux)
   - `Cmd+Shift+Delete` (Mac)
   - 选择"缓存的图片和文件"

2. **Firefox**:
   - `Ctrl+Shift+Delete`
   - 选择"缓存"

3. **硬刷新**（推荐）:
   - `Ctrl+F5` (Windows/Linux)
   - `Cmd+Shift+R` (Mac)

## 相关文件

- 环境配置：`.env` (第278行)
- 构建脚本：`build.sh` (`generate_or_update_env_file` 函数)
- Gitea启动脚本：`src/gitea/gitea-entrypoint.sh` (第174-179行)
- 验证脚本：`test-gitea-assets.sh`

## 配置规则总结

**黄金法则**：当 Gitea 部署在子路径时，`STATIC_URL_PREFIX` 应该等于 `SUBURL`

| 部署方式 | SUBURL | STATIC_URL_PREFIX | 静态资源路径 |
|---------|--------|-------------------|-------------|
| 根路径 | / | / | /assets/js/index.js |
| 子路径 /gitea | /gitea | /gitea | /gitea/assets/js/index.js ✅ |
| ❌ 错误配置 | /gitea | /assets | /assets/assets/js/index.js ❌ |

## 未来改进建议

1. **在 `.env.example` 中添加注释**：
   ```bash
   # Gitea 静态资源前缀 (应与 SUBURL 保持一致)
   STATIC_URL_PREFIX=/gitea
   ```

2. **在 `build.sh` 中添加配置校验**：
   ```bash
   if [ "$STATIC_URL_PREFIX" != "$SUBURL" ]; then
       print_warning "STATIC_URL_PREFIX 与 SUBURL 不一致，可能导致静态资源加载失败"
   fi
   ```

3. **在文档中明确说明**：
   - 更新 README.md 添加 Gitea 配置说明
   - 在故障排除部分添加此问题的解决方案

## 总结

✅ **修复完成**：Gitea 静态资源路径配置已修复

🔧 **修复方式**：
- 修改 `.env` 中 `STATIC_URL_PREFIX=/assets` → `STATIC_URL_PREFIX=/gitea`
- 更新 `build.sh` 自动配置逻辑
- 重启 Gitea 容器应用配置

📝 **重要提醒**：
- `STATIC_URL_PREFIX` 必须与 `SUBURL` 保持一致
- 修复后需要清理浏览器缓存
- 使用 `test-gitea-assets.sh` 验证配置正确性
