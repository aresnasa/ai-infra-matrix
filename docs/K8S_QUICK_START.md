# Kubernetes 多版本支持 - 快速使用指南

## 🚀 快速开始

### 1. 构建项目

```bash
# 进入项目目录
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix

# 使用 build.sh 构建所有服务
./build.sh build-all

# 或者只构建受影响的服务
./build.sh build-backend
./build.sh build-frontend
```

### 2. 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 或者重启特定服务
docker-compose restart backend frontend
```

### 3. 访问新功能

打开浏览器访问：

**增强的 Kubernetes 资源管理**:
```
http://localhost:8080/kubernetes/resources
```

**传统 Kubernetes 集群管理**:
```
http://localhost:8080/kubernetes
```

## 📋 功能说明

### 新增功能

1. **多版本 Kubernetes 兼容**
   - 自动适配 Kubernetes 1.16+ 到 1.33+
   - 完全支持 1.27.5 及其他常见版本
   - 无需为不同版本配置不同客户端

2. **集群版本检测**
   - 自动显示集群版本信息
   - 显示 Git Version、Platform、Build Date

3. **完整的资源发现**
   - 列出所有标准 k8s 资源
   - 自动发现所有 CRD（自定义资源定义）
   - 按 API 组/版本分类展示

4. **资源树形视图**
   - 集群版本信息
   - 内置资源（按 API 组分组）
   - 自定义资源/CRD（按 Group 分组）
   - 支持搜索过滤

5. **资源管理**
   - 查看资源实例列表
   - 命名空间过滤
   - 查看资源详情（元数据、Spec、Status）
   - YAML 编辑器（Monaco Editor）
   - 更新和删除资源
   - 下载 YAML

## 🔧 使用示例

### 查看集群版本

```bash
curl http://localhost:8080/api/kubernetes/clusters/1/version
```

响应示例：
```json
{
  "major": "1",
  "minor": "27",
  "gitVersion": "v1.27.5",
  "platform": "linux/amd64",
  "buildDate": "2023-08-15T10:20:30Z"
}
```

### 获取增强的资源发现

```bash
curl http://localhost:8080/api/kubernetes/clusters/1/enhanced-discovery
```

响应包含：
- 集群版本
- 所有 API 组
- 按 GroupVersion 分组的资源
- 所有 CRD 列表
- 资源统计信息

### 前端操作流程

1. **选择集群**: 在顶部下拉框中选择要管理的集群
2. **查看版本**: 确认显示的集群版本正确
3. **浏览资源树**: 
   - 展开"内置资源"查看标准 k8s 资源
   - 展开"自定义资源 (CRD)"查看 CRD
4. **查看资源列表**: 点击资源类型（如 pods、deployments）
5. **查看详情**: 点击"查看"按钮
6. **编辑资源**: 
   - 点击"编辑"按钮
   - 在 YAML 编辑器中修改
   - 点击"保存"应用更改
7. **删除资源**: 点击"删除"按钮并确认

## 📦 依赖说明

### 后端依赖（自动安装）
- `k8s.io/client-go v0.33.1` - Kubernetes 客户端
- `k8s.io/api v0.33.1` - Kubernetes API 定义
- `k8s.io/apimachinery v0.33.1` - API 机制

### 前端依赖（已包含在 package.json）
- `@monaco-editor/react ^4.6.0` - YAML 编辑器
- `js-yaml ^4.1.0` - YAML 解析

这些依赖在使用 `build.sh` 构建时会自动安装。

## 🎯 API 端点

### 新增端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/kubernetes/clusters/:id/version` | 获取集群版本 |
| GET | `/api/kubernetes/clusters/:id/enhanced-discovery` | 增强的资源发现 |

### 现有端点（继续可用）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/kubernetes/clusters/:id/discovery` | 基础资源发现 |
| GET | `/api/kubernetes/clusters/:id/namespaces` | 命名空间列表 |
| GET | `/api/kubernetes/clusters/:id/namespaces/:ns/resources/:type` | 资源列表 |
| GET | `/api/kubernetes/clusters/:id/namespaces/:ns/resources/:type/:name` | 资源详情 |
| PUT | `/api/kubernetes/clusters/:id/namespaces/:ns/resources/:type/:name` | 更新资源 |
| DELETE | `/api/kubernetes/clusters/:id/namespaces/:ns/resources/:type/:name` | 删除资源 |

## 🐛 故障排查

### 构建失败

**问题**: Docker 构建被取消
```bash
ERROR: failed to build: failed to solve: Canceled: context canceled
```

**解决方案**:
```bash
# 清理并重新构建
docker system prune -f
./build.sh build-all --force
```

### 前端无法显示

**问题**: Monaco Editor 或资源树不显示

**解决方案**:
```bash
# 检查依赖是否安装
cd src/frontend
npm list @monaco-editor/react js-yaml

# 重新构建前端
cd ../..
./build.sh build-frontend --force
```

### API 返回 404

**问题**: 新的 API 端点返回 404

**解决方案**:
```bash
# 重新构建后端
./build.sh build-backend

# 重启服务
docker-compose restart backend
```

### CRD 列表为空

**问题**: 集群中有 CRD 但不显示

**解决方案**:
1. 检查 kubeconfig 权限
2. 确认集群支持 `apiextensions.k8s.io/v1`
3. 查看后端日志：`docker-compose logs backend | grep CRD`

## ✅ 验证清单

- [ ] `./build.sh build-all` 执行成功
- [ ] 所有服务正常运行（`docker-compose ps`）
- [ ] 能访问 http://localhost:8080/kubernetes/resources
- [ ] 集群版本正确显示
- [ ] 资源树显示内置资源和 CRD
- [ ] 能查看资源列表
- [ ] 能打开资源详情
- [ ] YAML 编辑器正常工作
- [ ] 能更新和删除资源

## 📚 相关文档

- 完整实施文档: `docs/KUBERNETES_MULTI_VERSION_IMPLEMENTATION.md`
- 原始需求: `dev-md.md` 第 30 条
- Build 脚本: `build.sh`

## 🎓 最佳实践

1. **首次使用**: 建议使用 `./build.sh build-all` 完整构建
2. **开发调试**: 只构建修改的服务（backend 或 frontend）
3. **生产部署**: 使用 `--force` 标志确保完全重建
4. **版本兼容**: client-go v0.33.1 兼容大多数 k8s 版本，无需担心
5. **权限管理**: 确保 kubeconfig 有足够权限读取 CRD

---

**更新日期**: 2025-10-10  
**版本**: v0.3.7
