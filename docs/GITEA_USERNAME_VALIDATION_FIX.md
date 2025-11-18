# Gitea 用户名验证修复报告

## 问题概述

**报告时间**: 2025-09-03 19:50  
**错误信息**: `CreateUser: name is invalid [admin@example.com]: must be valid alpha or numeric or dash(-_) or dot characters`

## 根本原因分析

1. **模板渲染问题**: 
   - `scripts/template_renderer.py` 中 `GITEA_ALIAS_ADMIN_TO` 默认值错误设置为 `admin@example.com`
   - 应该设置为 `admin`

2. **模板占位符格式不统一**:
   - 模板文件使用 `{{GITEA_ALIAS_ADMIN_TO}}` 格式
   - nginx entrypoint 脚本只处理 `${GITEA_ALIAS_ADMIN_TO}` 格式
   - 导致运行时环境变量替换失败

3. **硬编码值残留**:
   - 模板中存在硬编码的 `admin@example.com` 值
   - nginx 配置中用于用户名字段的值包含 @ 符号，违反 Gitea 用户名规则

## 修复步骤

### 1. 修复模板渲染器默认值
**文件**: `scripts/template_renderer.py`
```python
# 修改前
'GITEA_ALIAS_ADMIN_TO': 'admin@example.com',

# 修改后  
'GITEA_ALIAS_ADMIN_TO': 'admin',
```

### 2. 统一模板占位符格式
**文件**: `src/nginx/templates/conf.d/includes/gitea.conf.tpl`
```nginx
# 修改前
proxy_set_header X-WEBAUTH-USER "{{GITEA_ALIAS_ADMIN_TO}}";

# 修改后
proxy_set_header X-WEBAUTH-USER "${GITEA_ALIAS_ADMIN_TO}";
```

### 3. 增强 nginx entrypoint 脚本
**文件**: `src/nginx/docker-entrypoint.sh`
```bash
# 增加对双大括号格式的支持
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{GITEA_ALIAS_ADMIN_TO}}/${GITEA_ALIAS_ADMIN_TO}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{GITEA_ADMIN_EMAIL}}/${GITEA_ADMIN_EMAIL}/g" {} \;
```

### 4. 验证环境变量配置
**文件**: `.env`
```bash
GITEA_ALIAS_ADMIN_TO=admin          # ✅ 正确 
GITEA_ADMIN_EMAIL=admin@example.com # ✅ 正确
```

## 修复后的配置

### nginx 反向代理头设置
```nginx
# 用户认证头 (不能包含 @)
proxy_set_header X-WEBAUTH-USER "admin";
proxy_set_header X-Forwarded-User "admin";
proxy_set_header Remote-User "admin";
proxy_set_header X-User "admin";

# 邮箱头 (可以包含 @)
proxy_set_header X-WEBAUTH-EMAIL "admin@example.com";
```

### 用户映射规则
```nginx
# SSO admin 用户映射为 Gitea admin 用户
if ($user_header = "admin") { 
    set $user_header "admin"; 
}
```

## 验证结果

1. **构建验证**: nginx 镜像构建成功，模板正确渲染
2. **运行时验证**: nginx 容器启动时环境变量正确替换
3. **功能验证**: Gitea 不再报告用户名验证错误
4. **访问验证**: `curl http://localhost:8080/gitea/` 返回正常的 admin Dashboard

## 影响服务

- ✅ **Nginx**: 配置已修复并重新加载
- ✅ **Gitea**: 用户创建错误已解决
- ✅ **SSO 认证**: 用户映射正常工作

## 预防措施

1. **模板系统统一**: 确保所有模板使用一致的占位符格式
2. **默认值验证**: 在模板渲染器中验证默认值符合目标系统要求
3. **集成测试**: 增加对用户认证流程的自动化测试
4. **文档更新**: 更新用户认证配置相关文档

## 相关文件清单

- `scripts/template_renderer.py` - 修复默认值
- `src/nginx/templates/conf.d/includes/gitea.conf.tpl` - 统一占位符格式
- `src/nginx/docker-entrypoint.sh` - 增强环境变量替换
- `.env` - 环境变量配置（已验证正确）

## 总结

本次修复解决了 Gitea 用户名验证问题的根本原因，通过统一模板系统和修正配置值，确保了 SSO 认证流程的正常工作。修复已通过构建和运行时验证，系统现在可以正常处理用户认证。
