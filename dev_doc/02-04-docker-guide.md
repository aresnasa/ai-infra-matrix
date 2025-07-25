# Docker文件组织结构

本目录按环境和用途组织了项目中的所有Docker相关文件。

## 目录结构

```
docker/
├── production/          # 生产环境Docker文件
│   ├── backend/        # 后端生产环境
│   └── frontend/       # 前端生产环境
├── development/        # 开发环境Docker文件
│   └── docker-compose.dev.yml
├── testing/           # 测试环境Docker文件
│   ├── backend/       # 后端测试环境
│   ├── frontend/      # 前端测试环境
│   └── docker-compose.test.yml
└── README.md          # 本文件
```

## 文件用途说明

### 生产环境 (production/)

- **backend/Dockerfile** - 生产环境后端镜像
  - 基于Go 1.24
  - 多阶段构建优化
  - 安全配置和最小化镜像
  
- **frontend/Dockerfile** - 生产环境前端镜像
  - 基于Node.js构建
  - Nginx静态文件服务
  - 生产环境优化

### 开发环境 (development/)

- **docker-compose.dev.yml** - 开发环境编排
  - 热重载支持
  - 开发工具集成
  - 调试端口开放

### 测试环境 (testing/)

- **backend/Dockerfile.test** - 测试环境后端镜像
  - 包含测试工具
  - 测试数据库配置
  - Coverage报告生成

- **frontend/Dockerfile.frontend.test** - 前端测试镜像
  - 端到端测试环境
  - 测试浏览器支持
  - 测试报告生成

- **docker-compose.test.yml** - 测试环境编排
  - 独立测试网络
  - 测试数据库实例
  - 自动化测试流程

## 使用指南

### 生产环境部署

```bash
# 使用生产环境Dockerfile构建
docker build -f docker/production/backend/Dockerfile -t ansible-backend:prod .
docker build -f docker/production/frontend/Dockerfile -t ansible-frontend:prod .
```

### 开发环境

```bash
# 启动开发环境
docker-compose -f docker/development/docker-compose.dev.yml up -d
```

### 测试环境

```bash
# 运行测试
docker-compose -f docker/testing/docker-compose.test.yml up --abort-on-container-exit
```

## 维护说明

1. **生产环境文件**：保持最新稳定版本，注重安全和性能优化
2. **开发环境文件**：支持快速迭代，包含开发工具
3. **测试环境文件**：专注于自动化测试，包含测试工具和数据

## 版本管理

- 所有Dockerfile使用语义化版本标签
- 定期更新基础镜像版本
- 记录重要变更在各自目录的CHANGELOG中
