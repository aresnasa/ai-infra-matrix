# 前端服务重启指南

由于我们修复了前端API路径配置，需要重新启动前端服务以使更改生效。

## 🔄 重启前端服务

### 方法1: 使用Docker Compose（推荐）
```bash
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix
docker-compose restart frontend
```

### 方法2: 手动重启
```bash
# 停止前端服务
docker-compose stop frontend

# 重新构建并启动
docker-compose up -d frontend
```

### 方法3: 完整重启所有服务
```bash
docker-compose down
docker-compose up -d
```

## 🧪 验证修复效果

重启后，访问以下页面验证修复效果：

1. **登录**: http://192.168.0.200:3000/login
   - 用户名: admin
   - 密码: admin123

2. **Kubernetes管理**: http://192.168.0.200:3000/kubernetes
   - 选择集群: docker-desktop-local (ID: 2)
   - 切换命名空间: kube-node-lease
   - 查看Pod列表（之前会404错误，现在应该正常显示）

## 🔍 问题排查

如果仍有问题，检查：

```bash
# 查看前端服务日志
docker-compose logs frontend

# 查看后端服务日志
docker-compose logs backend

# 验证API直接访问
curl -H "Authorization: Bearer $TOKEN" \
  "http://192.168.0.200:8080/api/kubernetes/clusters/2/namespaces/kube-node-lease/resources/pods"
```

## ✅ 修复确认

修复完成后，以下功能应该正常工作：
- ✅ Pod列表显示
- ✅ Deployment列表显示
- ✅ Service列表显示  
- ✅ Events显示
- ✅ 集群级资源（如Nodes）显示
- ✅ 资源详情查看
- ✅ 日志查看功能
- ✅ 所有命名空间的资源访问

修复的核心问题是API路径格式：
- ❌ 错误: `/namespaces/{namespace}/pods`
- ✅ 正确: `/namespaces/{namespace}/resources/pods`
