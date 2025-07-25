# 开发文档编号系统

## 编号规则

### 分类编号
- **01-基础架构** - 基础系统、架构设计
- **02-构建部署** - 构建、部署相关文档  
- **03-数据库** - 数据库设计、迁移
- **04-AI服务** - AI相关功能和集成
- **05-测试** - 测试策略、报告
- **06-修复记录** - 问题修复和解决方案
- **07-集成** - 系统集成和配置
- **08-运维** - 运维、监控相关
- **09-项目管理** - 项目状态、组织

### 文件命名格式
```
{分类编号}-{顺序编号}-{简短描述}.md
```

示例：
- `01-01-architecture-overview.md` - 架构概览
- `03-01-database-schema.md` - 数据库设计
- `05-01-testing-guide.md` - 测试指南

## 当前文档重新编号计划

### 01-基础架构
- `01-01-ai-middleware-architecture.md` - AI中间件架构
- `01-02-kubernetes-status.md` - Kubernetes状态说明

### 02-构建部署  
- `02-01-ai-async-deployment-guide.md` - AI异步部署指南
- `02-02-ai-async-docker-test-guide.md` - Docker测试指南

### 03-数据库
- `03-01-database-init-report.md` - 数据库初始化报告

### 04-AI服务
- `04-01-ai-assistant-implementation.md` - AI助手实现报告
- `04-02-ai-middleware-implementation.md` - AI中间件实现总结
- `04-03-ai-async-test-completion.md` - AI异步测试完成报告

### 05-测试
- `05-01-test-ai-assistant.md` - AI助手测试指南

### 06-修复记录
- `06-01-go-import-fix-report.md` - Go导入路径修复报告
- `06-02-ssl-fix-success.md` - SSL修复成功记录
- `06-03-fix-summary.md` - 修复总结

### 07-集成
- `07-01-integration-summary.md` - 集成总结
- `07-02-login-ai-verification.md` - 登录和AI验证报告

### 09-项目管理
- `09-01-project-organization-complete.md` - 项目组织完成报告

## 模型学习优化

### 文档结构标准
每个文档应包含：
1. **标题** - 清晰的文档标题
2. **概述** - 简要说明文档内容
3. **详细内容** - 分段落的详细说明
4. **代码示例** - 相关代码片段（如适用）
5. **相关链接** - 关联文档引用

### 标签系统
在文档开头添加标签便于分类：
```markdown
---
tags: [architecture, ai, middleware]
category: 基础架构
difficulty: 中级
last-updated: 2025-07-23
---
```

### 交叉引用
使用标准化的文档引用格式：
```markdown
参考文档：[01-01-架构概览](01-01-architecture-overview.md)
```
