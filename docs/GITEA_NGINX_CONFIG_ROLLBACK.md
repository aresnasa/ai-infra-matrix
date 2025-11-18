# Gitea Nginx 配置回滚报告

## 日期
2025年10月11日

## 问题背景

访问 Gitea 时出现资产文件加载失败的错误：
```
Failed to load asset files from http://192.168.18.114:8080/assets/assets/js/index.js
```

路径出现了重复的 `assets/assets/` 问题。

## 根本原因分析

### 问题1: Nginx Rewrite 规则被移除
在尝试修复时，错误地移除了关键的 `rewrite` 指令：
```nginx
# 错误的修改：
location ^~ /gitea/ {
    # 不做 rewrite，直接代理保持 /gitea/ 路径
    proxy_pass http://gitea:3000;
}

# 正确的配置：
location ^~ /gitea/ {
    rewrite ^/gitea(/.*)$ $1 break;
    proxy_pass http://gitea:3000;
}
```

### 问题2: 静态资源 Location 块冲突
添加了额外的静态资源 location 块，与主 location 块产生了冲突：
```nginx
# 不需要的额外配置：
location ~ ^/gitea/(assets|css|js|...)/ {
    proxy_pass http://gitea:3000;
}
```

### 问题3: STATIC_URL_PREFIX 配置误解
`.env` 文件中的 `STATIC_URL_PREFIX=/assets` 是正确的配置，不应该改为 `/gitea/assets`。

**配置逻辑**：
- Gitea 的 `ROOT_URL=http://192.168.18.114:8080/gitea/`
- Nginx rewrite 将 `/gitea/xxx` 重写为 `/xxx`
- Gitea 接收到的请求路径是 `/xxx`（不带 `/gitea/` 前缀）
- Gitea 使用 `STATIC_URL_PREFIX=/assets` 生成静态资源路径
- Nginx 将请求转发时，Gitea 返回的 HTML 中包含 `/assets/...` 路径
- 浏览器请求 `/assets/...` → Nginx 重写 → `/gitea/assets/...` → Gitea 处理

## 修复方案

### 1. 恢复 Rewrite 指令
```diff
  location ^~ /gitea/ {
      access_log /var/log/nginx/gitea_access.log authdebug;
-     # 不做 rewrite，直接代理保持 /gitea/ 路径
-     # Gitea 的 ROOT_URL 配置为 /gitea/，需要保持完整路径
+     rewrite ^/gitea(/.*)$ $1 break;
      proxy_pass http://gitea:3000;
```

### 2. 移除静态资源 Location 块
删除了额外添加的静态资源 location 块，保持原有的简洁配置。

### 3. 保持 .env 配置不变
```bash
STATIC_URL_PREFIX=/assets  # 正确配置，不需要修改
```

## 验证步骤

1. **检查模板文件**：
   ```bash
   git diff src/nginx/templates/conf.d/includes/gitea.conf.tpl
   ```

2. **验证渲染后的配置**：
   ```bash
   ./build.sh render-templates nginx
   cat src/nginx/conf.d/includes/gitea.conf | grep -A 5 "location ^~ /gitea/"
   ```

3. **重新构建和部署**：
   ```bash
   ./build.sh build nginx --force
   docker-compose restart nginx
   ```

4. **测试访问**：
   - 访问 http://192.168.18.114:8080/gitea/
   - 检查浏览器控制台，确认资产文件正确加载
   - 验证路径不再出现 `/assets/assets/` 重复

## 模板变量

当前 Gitea 配置模板使用了以下环境变量（已在 `.env` 中定义）：

| 变量名 | 默认值 | 用途 |
|--------|--------|------|
| `GITEA_ALIAS_ADMIN_TO` | `admin` | SSO 管理员用户映射 |
| `GITEA_ADMIN_EMAIL` | `admin@example.com` | 管理员邮箱 |

这些变量通过 `scripts/render_template.py` 正确渲染到最终的 Nginx 配置中。

## 经验教训

1. **不要随意修改已工作的 Nginx Rewrite 规则**
   - Rewrite 规则与应用的 ROOT_URL 配置密切相关
   - 修改前需要完整理解路径转换逻辑

2. **静态资源路径配置的完整链路**
   - 浏览器请求 → Nginx 代理 → 应用处理 → HTML 响应 → 浏览器再次请求静态资源
   - 每个环节的路径转换都需要考虑

3. **使用 Git 历史作为参考**
   - 遇到问题时，先检查 `git diff` 和 `git log`
   - 对比历史版本找出正确的配置

4. **渲染模板系统的重要性**
   - 模板变量替换由 `scripts/render_template.py` 处理
   - 保留 Nginx 变量（如 `$http_host`），替换环境变量（如 `${GITEA_ADMIN_EMAIL}`）

## 相关文件

- `/src/nginx/templates/conf.d/includes/gitea.conf.tpl` - Nginx 配置模板
- `/.env` - 环境变量配置
- `/scripts/render_template.py` - 模板渲染脚本
- `/build.sh` - 构建脚本（render-templates 命令）

## 状态

✅ **已修复** - Gitea Nginx 配置已恢复到正确状态
✅ **已验证** - 模板渲染成功，包含新增的两个变量
🔄 **待测试** - 需要在浏览器中验证 Gitea 资产加载是否正常

## 下一步

1. 在浏览器中访问 http://192.168.18.114:8080/gitea/ 验证修复效果
2. 检查浏览器控制台确认不再有 `/assets/assets/` 错误
3. 如果问题仍存在，检查 Gitea 容器的 `app.ini` 配置
