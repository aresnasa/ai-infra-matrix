# AI Infrastructure Matrix - Project Structure

## 📁 Core Project Structure (Post v0.0.3 Cleanup)

```
ai-infra-matrix/
├── 📊 Configuration Files
│   ├── docker-compose.yml          # 主要容器编排配置
│   ├── .env.example                # 环境变量模板
│   └── .gitignore                  # Git忽略规则
│
├── 🐳 Container Sources
│   ├── src/
│   │   ├── nginx/                  # Nginx代理配置
│   │   ├── backend/                # 后端API服务
│   │   ├── frontend/               # 前端React应用
│   │   ├── jupyterhub/             # JupyterHub配置
│   │   └── shared/                 # 共享静态资源
│   └── docker/                     # Docker构建文件
│
├── 📚 Documentation
│   ├── docs/                       # 项目文档
│   └── dev_doc/                    # 开发文档
│
├── 🗂️ Data & Storage
│   ├── data/                       # 持久化数据
│   ├── shared/                     # 共享文件
│   └── notebooks/                  # Jupyter notebooks
│
├── 🚀 Deployment & Scripts
│   ├── deploy.sh                   # 部署脚本
│   ├── start-services.sh           # 服务启动脚本
│   ├── test-deployment.sh          # 部署测试脚本
│   └── verify-system.sh            # 系统验证脚本
│
├── 🧪 Legacy/Testing
│   ├── tests/                      # 单元测试
│   ├── scripts/                    # 辅助脚本
│   └── jupyterhub/                 # 遗留JupyterHub配置
│
└── 📦 Archive
    └── archive/
        ├── v0.0.3_milestone/       # v0.0.3里程碑归档
        │   ├── test_scripts/       # 开发测试脚本
        │   ├── debug_tools/        # 调试工具
        │   ├── reports/            # 项目报告
        │   ├── MILESTONE_SUMMARY.md
        │   └── ARCHIVE_INVENTORY.md
        └── [previous versions]/    # 历史版本归档
```

## 🎯 Key Changes in v0.0.3

### ✅ Removed from Root
- 31 test scripts and debugging tools → `archive/v0.0.3_milestone/`
- 5 project reports and analyses → `archive/v0.0.3_milestone/reports/`
- Temporary files and debug HTML pages → `archive/v0.0.3_milestone/debug_tools/`

### ✅ Preserved in Root
- **Core configuration**: docker-compose.yml, .env files
- **Source code**: src/ directory with all services
- **Documentation**: docs/ and dev_doc/
- **Deployment scripts**: Essential deployment and verification scripts
- **Data directories**: Persistent data and shared resources

## 🏗️ Architecture Overview

### Service Architecture
```
Nginx (Entry Point)
├── Frontend (React App)
├── Backend (API Server)
├── JupyterHub (ML Platform)
└── Databases (PostgreSQL, Redis)
```

### Authentication Flow
```
User → Nginx → Auth Bridge → JWT Validation → JupyterHub
```

## 📋 Development Guidelines

### File Organization
- **Source code**: Keep in `src/` with service-specific subdirectories
- **Documentation**: Use `docs/` for user docs, `dev_doc/` for development
- **Testing**: Archive test scripts after milestones, keep only essential tests
- **Configuration**: Main configs in root, service configs in respective src/ folders

### Archive Strategy
- Archive development files after each milestone
- Preserve important test scripts for regression testing
- Document development process in milestone summaries
- Keep project root clean and focused on production files

---

**Last Updated**: 2025年8月10日 - v0.0.3 Milestone Cleanup
