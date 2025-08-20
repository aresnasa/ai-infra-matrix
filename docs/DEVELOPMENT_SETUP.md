# AI Infrastructure Matrix - 开发环境搭建指南

## 🎯 开发环境概述

本指南帮助开发者快速搭建AI Infrastructure Matrix的本地开发环境。

## 📋 前置要求

### 必需软件

| 软件 | 版本要求 | 安装方式 |
|------|----------|----------|
| **Docker** | 20.10+ | [官方安装指南](https://docs.docker.com/get-docker/) |
| **Docker Compose** | 2.0+ | 随Docker Desktop安装 |
| **Git** | 2.30+ | [官方下载](https://git-scm.com/) |
| **Node.js** | 18+ | [官方下载](https://nodejs.org/) |
| **Python** | 3.11+ | [官方下载](https://python.org/) |

### 系统要求

- **内存**: 最少4GB，推荐8GB+
- **磁盘**: 最少10GB可用空间
- **操作系统**: macOS 10.15+, Ubuntu 20.04+, Windows 10+

## 🚀 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/aresnasa/ai-infra-matrix.git
cd ai-infra-matrix
```

### 2. 环境配置

```bash
# 复制环境配置文件
cp .env.example .env

# 编辑配置文件
vi .env
```

### 3. 一键启动开发环境

```bash
# 构建并启动所有服务
./scripts/build.sh dev --up --test

# 或者分步执行
./scripts/build.sh dev              # 构建镜像
docker compose up -d                # 启动服务
./scripts/test-health.sh           # 健康检查
```

### 4. 验证安装

访问以下地址确认服务正常：

- 🌐 主页: <http://localhost:8080>
- 🔐 管理后台: <http://localhost:8080/admin>
- 📊 JupyterHub: <http://localhost:8080/jupyter>
- 🗃️ Gitea: <http://localhost:8080/gitea>

## 🛠️ 开发工具配置

### 前端开发

```bash
# 进入前端目录
cd src/frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 代码检查
npm run lint

# 运行测试
npm test
```

### 后端开发

```bash
# 进入后端目录
cd src/backend

# 创建虚拟环境
python -m venv venv
source venv/bin/activate  # Linux/macOS
# venv\Scripts\activate   # Windows

# 安装依赖
pip install -r requirements.txt
pip install -r requirements-dev.txt

# 启动开发服务器
uvicorn main:app --reload --host 0.0.0.0 --port 8000

# 运行测试
pytest

# 代码格式化
black .
isort .
```

### 数据库开发

```bash
# 连接PostgreSQL
docker exec -it ai-infra-postgres psql -U ai_infra_user -d ai_infra_db

# 数据库迁移
cd src/backend
alembic upgrade head

# 创建新迁移
alembic revision --autogenerate -m "描述信息"
```

## 🔧 环境变量配置

### 开发环境 (.env)

```bash
# 数据库配置
POSTGRES_DB=ai_infra_db
POSTGRES_USER=ai_infra_user
POSTGRES_PASSWORD=ai_infra_dev_pass

# Redis配置
REDIS_PASSWORD=redis_dev_pass

# JWT配置
JWT_SECRET_KEY=your_jwt_secret_key_here
JWT_ALGORITHM=HS256
JWT_ACCESS_TOKEN_EXPIRE_MINUTES=30

# 管理员账号
ADMIN_USER=admin
ADMIN_PASSWORD=admin123

# 调试模式
DEBUG_MODE=true
BUILD_ENV=development

# 前端配置
REACT_APP_API_URL=http://localhost:8000/api
REACT_APP_JUPYTERHUB_URL=http://localhost:8080/jupyter
```

### 生产环境 (.env.prod)

```bash
# 数据库配置（强密码）
POSTGRES_DB=ai_infra_db
POSTGRES_USER=ai_infra_user
POSTGRES_PASSWORD=强密码请修改

# Redis配置（强密码）
REDIS_PASSWORD=强密码请修改

# JWT配置（生产密钥）
JWT_SECRET_KEY=生产环境密钥请修改
JWT_ALGORITHM=HS256
JWT_ACCESS_TOKEN_EXPIRE_MINUTES=1440

# 生产配置
DEBUG_MODE=false
BUILD_ENV=production

# 域名配置
DOMAIN=your-domain.com
```

## 🐳 Docker开发

### 构建脚本使用

```bash
# 开发模式构建
./scripts/build.sh dev

# 生产模式构建
./scripts/build.sh prod --version v0.0.3.3

# 多架构构建
./scripts/build.sh prod --multi-arch --registry docker.io/username --push

# 仅构建特定组件
./scripts/build.sh dev --nginx-only

# 无缓存构建
./scripts/build.sh dev --no-cache
```

### Docker Compose命令

```bash
# 启动所有服务
docker compose up -d

# 重新构建并启动
docker compose up -d --build

# 查看日志
docker compose logs -f [服务名]

# 停止服务
docker compose down

# 完全清理
docker compose down -v --remove-orphans
```

### 服务调试

```bash
# 进入容器调试
docker exec -it ai-infra-backend bash
docker exec -it ai-infra-frontend sh
docker exec -it ai-infra-postgres psql -U ai_infra_user -d ai_infra_db

# 查看容器日志
docker logs ai-infra-backend -f
docker logs ai-infra-nginx -f
```

## 🧪 测试与质量保证

### 运行测试套件

```bash
# 健康检查
./scripts/test-health.sh

# 完整集成测试
./scripts/test-integration-full.sh

# 前端测试
cd src/frontend && npm test

# 后端测试
cd src/backend && pytest

# 端到端测试
./scripts/test-e2e.sh
```

### 代码质量检查

```bash
# Python代码检查
cd src/backend
black --check .
isort --check-only .
flake8 .
mypy .

# JavaScript代码检查
cd src/frontend
npm run lint
npm run type-check
```

### 性能测试

```bash
# API性能测试
cd tests/performance
python load_test.py

# 前端性能分析
cd src/frontend
npm run build
npm run analyze
```

## 🔍 调试技巧

### 后端调试

```python
# 在代码中添加断点
import pdb; pdb.set_trace()

# 或使用ipdb
import ipdb; ipdb.set_trace()

# 使用VSCode调试
# 配置 .vscode/launch.json
{
    "name": "Python: FastAPI",
    "type": "python",
    "request": "launch",
    "program": "${workspaceFolder}/src/backend/main.py",
    "console": "integratedTerminal"
}
```

### 前端调试

```javascript
// 浏览器开发者工具
console.log('调试信息');
debugger; // 断点

// React DevTools
// 安装浏览器扩展
```

### 数据库调试

```sql
-- 查看活动连接
SELECT * FROM pg_stat_activity;

-- 查看表结构
\d table_name

-- 查看慢查询
SELECT query, mean_time FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;
```

## 📁 项目结构详解

```
ai-infra-matrix/
├── src/                        # 源代码目录
│   ├── backend/               # 后端API服务
│   │   ├── main.py           # FastAPI应用入口
│   │   ├── models/           # 数据模型
│   │   ├── routes/           # API路由
│   │   ├── services/         # 业务逻辑
│   │   └── utils/            # 工具函数
│   ├── frontend/             # 前端React应用
│   │   ├── src/
│   │   │   ├── components/   # React组件
│   │   │   ├── services/     # API服务
│   │   │   ├── utils/        # 工具函数
│   │   │   └── hooks/        # 自定义Hooks
│   │   └── public/           # 静态资源
│   ├── nginx/                # Nginx配置
│   └── jupyterhub/           # JupyterHub配置
├── scripts/                   # 构建和部署脚本
├── docs/                     # 项目文档
├── tests/                    # 测试文件
└── docker-compose.yml        # 容器编排配置
```

## 🎨 开发规范

### Git工作流

```bash
# 创建功能分支
git checkout -b feature/new-feature

# 提交代码
git add .
git commit -m "feat: 添加新功能"

# 推送分支
git push origin feature/new-feature

# 创建Pull Request
```

### 提交信息规范

```
feat: 新功能
fix: 修复问题
docs: 文档更新
style: 代码格式
refactor: 重构
test: 测试相关
chore: 构建/工具相关
```

### 代码风格

- **Python**: 遵循PEP 8，使用black格式化
- **JavaScript**: 遵循ESLint规则，使用Prettier格式化
- **注释**: 重要逻辑必须添加注释
- **函数**: 单一职责，合理命名

## 🚨 常见问题

### 服务启动失败

```bash
# 检查端口占用
lsof -i :8080
lsof -i :5432

# 清理Docker资源
docker system prune -a

# 重置数据库
docker compose down -v
docker compose up postgres -d
```

### 权限问题

```bash
# 修复文件权限
sudo chown -R $USER:$USER .

# Docker权限
sudo usermod -aG docker $USER
```

### 依赖问题

```bash
# 更新Node.js依赖
cd src/frontend
rm -rf node_modules package-lock.json
npm install

# 更新Python依赖
cd src/backend
pip install --upgrade -r requirements.txt
```

## 💡 最佳实践

### 开发流程

1. **功能开发前**: 创建分支，编写测试
2. **开发过程中**: 频繁提交，及时测试
3. **开发完成后**: 代码审查，集成测试
4. **部署前**: 完整测试，性能检查

### 性能优化

1. **数据库**: 合理使用索引，避免N+1查询
2. **前端**: 代码分割，懒加载，缓存优化
3. **Docker**: 多阶段构建，镜像优化
4. **网络**: 启用gzip，CDN加速

### 安全考虑

1. **认证**: JWT令牌，强密码策略
2. **授权**: RBAC权限模型
3. **传输**: HTTPS加密，安全头设置
4. **存储**: 密码加密，敏感数据保护

## 📞 获取帮助

- 📧 技术支持: <tech-support@example.com>
- 💬 开发者群: [加入讨论](https://github.com/aresnasa/ai-infra-matrix/discussions)
- 🐛 问题报告: [GitHub Issues](https://github.com/aresnasa/ai-infra-matrix/issues)
- 📖 在线文档: [项目Wiki](https://github.com/aresnasa/ai-infra-matrix/wiki)

---

**最后更新**: 2025年8月20日  
**维护者**: AI Infrastructure Team
