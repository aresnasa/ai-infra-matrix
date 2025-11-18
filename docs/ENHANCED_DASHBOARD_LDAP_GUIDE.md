# 增强仪表板与LDAP多用户集成使用说明

## 功能概述

这是一个完整的多用户LDAP集成仪表板系统，支持可拖拽的iframe部件、基于角色的权限管理和LDAP用户同步。

## 主要特性

### 🎯 增强仪表板
- **可拖拽部件**：基于react-beautiful-dnd实现iframe部件的自由拖拽重排
- **角色模板**：预设开发者、管理员、研究员等角色的默认配置
- **个性化配置**：每个用户可保存自己的仪表板布局偏好
- **导入导出**：支持仪表板配置的导入导出，便于团队协作
- **自动保存**：配置更改实时保存，无需手动确认

### 👥 LDAP多用户管理
- **自动同步**：支持从OpenLDAP自动同步用户和组织架构
- **权限管理**：基于LDAP组的细粒度权限控制
- **用户统计**：实时用户活跃度分析和统计信息
- **同步监控**：LDAP同步状态监控和历史记录查看
- **连接测试**：LDAP配置验证和连接测试

## 技术架构

### 前端技术栈
- **React 18** + **Ant Design 5** - 现代化UI框架
- **react-beautiful-dnd** - 拖拽功能实现
- **React Router v6** - 路由管理
- **Axios** - API请求封装

### 后端技术栈
- **Go/Gin** - 高性能Web框架
- **JWT认证** + **RBAC权限模型**
- **LDAP集成** - OpenLDAP用户同步
- **PostgreSQL/MySQL** - 用户配置持久化

## 使用方法

### 1. 访问增强仪表板

在导航菜单中点击"增强仪表板"，或直接访问 `/enhanced-dashboard` 路径。

### 2. 仪表板配置

#### 2.1 部件管理
- 查看当前配置的iframe部件列表
- 拖拽部件重新排序（按住部件标题栏拖拽）
- 添加新的iframe部件
- 编辑或删除现有部件

#### 2.2 权限控制
- 基于用户角色显示可访问的部件
- 管理员可查看所有部件
- 普通用户只能查看有权限的部件

#### 2.3 模板功能
- 选择预设角色模板（开发者/管理员/研究员）
- 应用模板会重置当前配置
- 支持从其他用户克隆配置

#### 2.4 导入导出
- 导出当前仪表板配置为JSON文件
- 导入配置文件快速恢复设置
- 支持覆盖或合并模式

### 3. LDAP用户管理（仅管理员）

#### 3.1 用户列表管理
- 查看所有系统用户
- 区分本地用户和LDAP用户
- 用户状态指示（在线/离线）
- 批量用户操作

#### 3.2 LDAP同步
- 手动触发用户同步
- 查看同步进度和状态
- 同步历史记录查看
- 同步失败原因分析

#### 3.3 统计信息
- 总用户数统计
- 在线用户实时监控
- LDAP用户占比
- 用户活跃度分析

#### 3.4 LDAP配置
- LDAP服务器连接配置
- 用户映射规则设置
- 组权限映射配置
- 连接测试和验证

## 配置示例

### 仪表板配置结构
```json
{
  "widgets": [
    {
      "id": "jupyter",
      "title": "JupyterHub",
      "url": "/jupyterhub",
      "width": 6,
      "height": 400,
      "permissions": ["user", "admin"]
    },
    {
      "id": "kubernetes",
      "title": "Kubernetes Dashboard",
      "url": "/kubernetes",
      "width": 6,
      "height": 400,
      "permissions": ["admin"]
    }
  ],
  "layout": ["jupyter", "kubernetes"],
  "theme": "light",
  "autoSave": true
}
```

### LDAP配置示例
```json
{
  "server": "ldap://openldap.example.com:389",
  "baseDN": "dc=example,dc=com",
  "userDN": "ou=users,dc=example,dc=com",
  "groupDN": "ou=groups,dc=example,dc=com",
  "bindUser": "cn=admin,dc=example,dc=com",
  "syncInterval": 3600,
  "enabled": true
}
```

## API接口

### 仪表板相关API
- `GET /dashboard/enhanced` - 获取增强仪表板配置
- `POST /dashboard/import` - 导入仪表板配置
- `GET /dashboard/export` - 导出仪表板配置
- `GET /dashboard/stats` - 获取仪表板统计信息

### 用户管理API
- `GET /admin/users` - 获取用户列表
- `GET /admin/user-stats` - 获取用户统计
- `POST /admin/ldap/sync` - 触发LDAP同步
- `GET /admin/ldap/sync/history` - 获取同步历史

## 部署说明

### 1. 环境要求
- Node.js 16+ (前端构建)
- Go 1.19+ (后端服务)
- PostgreSQL 12+ 或 MySQL 8.0+
- OpenLDAP (可选)

### 2. 配置文件
确保后端配置文件包含以下配置：
```yaml
# config.yaml
server:
  port: 8080
  
database:
  type: postgres
  host: localhost
  port: 5432
  name: ai_infra_matrix
  
ldap:
  enabled: true
  server: ldap://openldap:389
  base_dn: dc=example,dc=com
  
jwt:
  secret: your-secret-key
  expires: 24h
```

### 3. 启动服务
```bash
# 前端
cd src/frontend
npm install
npm run build

# 后端
cd src/backend
go mod tidy
go run cmd/main.go
```

## 故障排除

### 常见问题

#### 1. LDAP连接失败
- 检查LDAP服务器地址和端口
- 验证bind用户的认证信息
- 确认网络连通性

#### 2. 仪表板部件无法显示
- 检查iframe URL的有效性
- 验证用户对该部件的访问权限
- 查看浏览器控制台错误信息

#### 3. 用户同步失败
- 检查LDAP用户DN配置
- 验证用户映射规则
- 查看同步日志获取详细错误信息

### 日志查看
```bash
# 后端日志
tail -f logs/app.log

# LDAP同步日志
tail -f logs/ldap-sync.log

# 前端调试
# 浏览器开发者工具 -> Console
```

## 更新记录

### v1.0.0 (2024-01-15)
- ✅ 基础仪表板功能实现
- ✅ react-beautiful-dnd拖拽集成
- ✅ LDAP用户同步功能
- ✅ 基于角色的权限控制
- ✅ 用户统计和监控
- ✅ 配置导入导出功能

### 待开发功能
- 🔄 仪表板主题自定义
- 🔄 更多iframe部件预设
- 🔄 用户行为分析
- 🔄 移动端适配优化

## 技术支持

如有问题请查看：
1. 项目文档: `/docs/README.md`
2. 技术架构: `/docs/PROJECT_STRUCTURE.md`
3. 部署指南: `/docs/DEVELOPMENT_SETUP.md`

---

**注意**: 此功能需要管理员权限进行LDAP配置和用户管理。普通用户只能使用仪表板个性化功能。
