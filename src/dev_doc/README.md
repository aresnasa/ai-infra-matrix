# AI Infrastructure Matrix - 开发文档索引

本目录包含项目的所有开发文档和技术报告。

## 📋 项目完成报告

### 核心修复和部署
- **[Go Import 修复完成报告](GO_IMPORT_FIX_COMPLETE_REPORT.md)** - Go导入路径修复和Docker Compose部署完成总结
- **[SSL修复成功报告](SSL_FIX_SUCCESS.md)** - SSL证书和HTTPS配置修复
- **[修复总结](FIX-SUMMARY.md)** - 项目主要问题修复汇总

### AI功能实现
- **[AI助手实现报告](AI_ASSISTANT_IMPLEMENTATION_REPORT.md)** - AI助手功能的完整实现记录
- **[AI异步测试完成报告](AI_ASYNC_TEST_COMPLETION_REPORT.md)** - AI异步功能测试和验证
- **[登录和AI验证报告](LOGIN_AND_AI_VERIFICATION_REPORT.md)** - 用户认证和AI功能验证

### Kubernetes相关
- **[Kubernetes状态说明](KUBERNETES_STATUS_EXPLANATION.md)** - K8s集群状态和配置说明

### 测试文档
- **[AI助手测试](test_ai_assistant.md)** - AI助手功能测试指南

## 🏗️ 架构和实现文档

### AI中间件架构
- **[AI中间件架构](ai-middleware-architecture.md)** - AI服务的整体架构设计
- **[AI中间件实现总结](ai-middleware-implementation-summary.md)** - 实现细节和技术选择
- **[集成总结](INTEGRATION_SUMMARY.md)** - 各组件集成方案

### 项目组织
- **[项目组织完整文档](PROJECT_ORGANIZATION_COMPLETE.md)** - 项目结构和组织方式

## � 部署和运行指南

### AI异步部署
- **[AI异步部署指南](ai-async-deployment-guide.md)** - AI异步服务的部署步骤
- **[AI异步Docker测试指南](ai-async-docker-test-guide.md)** - Docker环境下的AI测试

## 🧪 测试文档

### 测试系统
- **[完整测试系统](testing/complete-testing-system.md)** - 完整的测试框架和方案
- **[测试总览](testing/testing-overview.md)** - 测试策略概述
- **[测试指南](testing/TESTING.md)** - 详细测试指导
- **[测试脚本](testing/test-scripts.md)** - 自动化测试脚本

### E2E测试
- **[E2E测试报告](testing/e2e-test-report-20250608_032523.md)** - 端到端测试结果

### 代理配置
- **[代理指南](testing/PROXY_GUIDE.md)** - 网络代理配置说明

## 📚 参考文档

### 构建和部署
- **[构建部署文档](build-deploy/)** - 构建和部署相关文档

### 数据库
- **[数据库文档](database/)** - 数据库设计和管理

### 通用文档
- **[通用文档](general/)** - 其他通用技术文档
- **[原始README](general/original-README.md)** - 项目原始文档

- **[TESTING.md](testing/TESTING.md)** - 主要测试脚本使用说明
- **[complete-testing-system.md](testing/complete-testing-system.md)** - 完整测试系统概述
- **[testing-overview.md](testing/testing-overview.md)** - 测试概览
- **[test-scripts.md](testing/test-scripts.md)** - 测试脚本详细说明
- **[PROXY_GUIDE.md](testing/PROXY_GUIDE.md)** - 代理配置指南
- **[e2e-test-report-20250608_032523.md](testing/e2e-test-report-20250608_032523.md)** - 端到端测试报告

### 🚀 [build-deploy/](build-deploy/) - 构建部署文档

构建、部署和运维相关文档

- **[DEPLOYMENT.md](build-deploy/DEPLOYMENT.md)** - 部署指南
- **[docker-guide.md](build-deploy/docker-guide.md)** - Docker 配置和使用指南

### 🗄️ [database/](database/) - 数据库文档

数据库结构、迁移和管理相关文档

- **[migrations-guide.md](database/migrations-guide.md)** - 数据库迁移和备份指南

### 📋 [general/](general/) - 通用文档

其他通用技术文档和参考资料

## 🎯 快速开始

1. **环境准备**: 参考部署指南设置开发环境
2. **数据库初始化**: 使用 `docker exec ansible-backend ./init` 初始化数据库
3. **服务启动**: 使用 `docker-compose up -d` 启动所有服务
4. **功能测试**: 参考测试文档验证功能

## 📊 当前状态

### 服务状态
- ✅ 后端服务: `http://localhost:8082`
- ✅ 前端服务: `http://localhost:3001`
- ✅ API文档: `http://localhost:8082/swagger/index.html`
- ✅ 数据库: PostgreSQL (端口5433)
- ✅ 缓存: Redis (端口6379)
- ✅ LDAP: OpenLDAP (端口389/636)

### 默认账户
- **管理员账户**: admin / admin123
- **首次登录后请修改密码**

### 数据库初始化完成
- ✅ 数据库schema已创建
- ✅ RBAC权限系统已初始化
- ✅ 默认管理员用户已创建
- ✅ AI配置已初始化（需要配置API密钥）

---

*最后更新: 2025-07-23*
*维护者: AI Infrastructure Matrix Team*

功能集成、修复总结和其他通用文档

- **[original-README.md](general/original-README.md)** - 原始项目README备份
- **[LDAP_INTEGRATION_SUMMARY.md](general/LDAP_INTEGRATION_SUMMARY.md)** - LDAP集成功能总结
- **[FIX-SUMMARY.md](general/FIX-SUMMARY.md)** - 管理中心导航修复总结
- **[INTEGRATION_SUMMARY.md](general/INTEGRATION_SUMMARY.md)** - 系统集成总结
- **[PROJECT_ORGANIZATION_COMPLETE.md](general/PROJECT_ORGANIZATION_COMPLETE.md)** - 项目组织结构完整说明

## 🔍 文档导航

### 快速开始

1. 阅读 [主README](../README.md) 了解项目概况
2. 参考 [testing/TESTING.md](testing/TESTING.md) 运行测试
3. 查看 [build-deploy/](build-deploy/) 了解部署选项

### 开发人员

- **测试开发**: [testing/](testing/) 目录下的所有文档
- **数据库开发**: [database/migrations-guide.md](database/migrations-guide.md)
- **功能集成**: [general/](general/) 目录下的集成文档

### 运维人员

- **部署运维**: [build-deploy/](build-deploy/) 目录
- **数据库管理**: [database/](database/) 目录
- **问题排查**: [testing/PROXY_GUIDE.md](testing/PROXY_GUIDE.md)

### 项目管理

- **项目状态**: [general/PROJECT_ORGANIZATION_COMPLETE.md](general/PROJECT_ORGANIZATION_COMPLETE.md)
- **功能总结**: [general/](general/) 目录下的各类总结文档

## 📝 文档维护

### 文档更新原则

- 每个功能模块的文档放在对应分类目录下
- 保持文档的时效性和准确性
- 重要变更需要更新相关文档

### 分类说明

- **testing/**: 所有测试相关的文档，包括测试脚本、测试报告、测试指南
- **build-deploy/**: 构建、部署、Docker配置等运维相关文档
- **database/**: 数据库设计、迁移、备份等数据库相关文档  
- **general/**: 功能集成总结、修复记录、项目组织等通用文档

---

**文档结构**: 按功能分类，便于查找和维护  
**最后更新**: 2025年6月9日
