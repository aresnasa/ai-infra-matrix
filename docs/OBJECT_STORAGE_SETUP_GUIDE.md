# 对象存储配置指南

## 问题说明

访问 `http://192.168.0.200:8080/object-storage` 显示"尚未配置对象存储"，无法查看MinIO状态。

## 解决方案

### 方法一：通过Web界面配置

1. **登录系统**
   - 使用管理员账号登录: `admin / admin123`

2. **进入配置页面**
   - 访问: `http://192.168.0.200:8080/admin/object-storage`
   - 或点击页面上的"立即配置"按钮

3. **添加MinIO配置**
   点击"添加存储"按钮，填写以下信息：
   
   ```
   配置名称: MinIO主存储
   存储类型: MinIO
   Endpoint: minio:9000 (Docker内部) 或 192.168.0.200:9000 (外部访问)
   Access Key: minioadmin
   Secret Key: minioadmin
   Web控制台URL: /minio-console/ (使用nginx代理) 或 http://192.168.0.200:9001 (直接访问)
   启用HTTPS: 否 (开发环境)
   设为默认: 是
   ```

4. **测试连接**
   - 点击"测试连接"按钮验证配置
   - 确保显示"连接成功"

5. **保存配置**
   - 点击"保存"按钮
   - 返回对象存储页面查看状态

### 方法二：通过API配置

使用curl命令配置：

```bash
# 1. 登录获取token
TOKEN=$(curl -s -X POST http://192.168.0.200:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | jq -r '.token')

# 2. 添加MinIO配置
curl -X POST http://192.168.0.200:8080/api/object-storage/configs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MinIO主存储",
    "type": "minio",
    "endpoint": "minio:9000",
    "access_key": "minioadmin",
    "secret_key": "minioadmin",
    "web_url": "/minio-console/",
    "use_ssl": false,
    "is_active": true
  }'
```

### 方法三：检查MinIO服务状态

如果配置后仍无法连接，检查MinIO服务：

```bash
# 检查MinIO容器状态
docker ps | grep minio

# 查看MinIO日志
docker logs ai-infra-minio

# 测试MinIO连接
curl http://192.168.0.200:9000/minio/health/live
```

## 预期结果

配置成功后，访问 `http://192.168.0.200:8080/object-storage` 应该能看到：

1. **存储服务列表**
   - MinIO配置卡片
   - 显示"已连接"状态
   - 显示"当前激活"标签

2. **存储统计**
   - 存储桶数量
   - 对象数量
   - 已用存储空间

3. **快速操作**
   - "访问MinIO控制台"按钮
   - 点击后跳转到 `/object-storage/minio/:id`
   - 显示MinIO Web控制台（iframe嵌入）

## 相关文件

- 前端页面: `src/frontend/src/pages/ObjectStoragePage.js`
- MinIO控制台页面: `src/frontend/src/pages/MinIOConsolePage.js`
- 配置页面: `src/frontend/src/pages/admin/ObjectStorageConfigPage.js`
- 后端API: `src/backend/internal/controllers/object_storage.go`

## 常见问题

### Q1: 点击"访问"按钮无反应
- 检查MinIO服务是否运行
- 检查Web控制台URL配置是否正确
- 建议使用nginx代理路径: `/minio-console/`

### Q2: iframe无法加载MinIO控制台
- 检查浏览器控制台是否有跨域错误
- 使用同源代理路径避免跨域问题
- 检查nginx配置是否正确代理MinIO

### Q3: 显示"连接失败"
- 检查Endpoint配置（Docker内部使用 `minio:9000`）
- 检查Access Key / Secret Key是否正确
- 检查MinIO服务是否正常运行

## 测试验证

运行自动化测试验证配置：

```bash
BASE_URL=http://192.168.0.200:8080 \
  npx playwright test test/e2e/specs/object-storage-full.spec.js \
  --config=test/e2e/playwright.config.js
```

成功后应该看到：
- ✓ 登录成功
- ✓ 页面加载完成
- ✓ 找到配置卡片
- ✓ 访问按钮可用
- ✓ 跳转到MinIO控制台
- ✓ iframe加载完成
