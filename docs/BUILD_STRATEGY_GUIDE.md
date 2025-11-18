# AI-Infra-Matrix 构建指南

## 构建策略决策

### 2025年9月15日 - 构建策略优化

#### 决策：统一使用 build.sh 脚本进行构建，降低直接使用 npm/docker/go 的权重

### 背景

- 项目包含多个组件：前端(React)、后端(Go)、基础设施(Docker)
- 之前存在多种构建方式，造成维护复杂性
- 用户反馈需要统一的构建体验

### 决策内容

#### ✅ 推荐构建方式

```bash
# 完整构建流程（推荐）
./build.sh build-all

# 单独构建前端
./build.sh build frontend

# 生产环境部署
./build.sh prod-up --force
```

#### ❌ 不推荐的构建方式

```bash
# 直接使用npm（不推荐）
cd src/frontend && npm run build

# 直接使用docker（不推荐）
docker build -t frontend src/frontend

# 直接使用go（不推荐）
cd src/backend && go build
```

### 原因分析

#### 1. 统一性

- `build.sh` 脚本集成了所有组件的构建逻辑
- 确保构建环境的一致性
- 自动处理依赖关系和构建顺序

#### 2. 环境管理

- 脚本自动配置中国镜像源
- 处理网络问题和构建优化
- 统一的版本管理和标签策略

#### 3. 错误处理

- 脚本提供详细的构建日志
- 自动重试机制
- 构建失败时的清晰错误信息

#### 4. 维护性

- 单一入口点便于维护
- 集中的配置管理
- 自动化的依赖检查

### 技术实现

#### 前端构建集成

```bash
# build.sh 中的前端构建逻辑
build_service() {
    # ... 其他服务构建逻辑
    if [[ "$service" == "frontend" ]]; then
        # 使用Docker容器构建，确保环境一致性
        docker build -f src/frontend/Dockerfile -t ai-infra-frontend:$tag src/frontend
    fi
}
```

#### 镜像源配置

- Alpine镜像源：阿里云 → 清华 → 中科大 → 官方
- npm镜像源：自动配置为 npmmirror.com
- Go模块代理：自动配置国内代理

### 使用指南

#### 开发环境

```bash
# 快速构建所有服务
./build.sh build-all

# 只构建前端
./build.sh build frontend

# 查看构建状态
./build.sh list
```

#### 生产环境

```bash
# 完整生产部署
./build.sh prod-up --force

# 查看服务状态
./build.sh prod-status

# 查看服务日志
./build.sh prod-logs frontend
```

### 迁移计划

#### 短期 (已完成)

- ✅ 标记直接工具使用为不推荐
- ✅ 在文档中明确推荐使用 build.sh
- ✅ 添加构建脚本的详细说明

#### 长期计划

- [ ] 在CI/CD中移除直接工具调用
- [ ] 添加构建脚本的健康检查
- [ ] 完善构建缓存机制

### 风险评估

#### 低风险

- 构建脚本已经稳定运行
- 所有组件都有Dockerfile
- 现有功能不受影响

#### 潜在问题

- 学习曲线：开发者需要熟悉 build.sh 参数
- 调试复杂性：多层封装可能增加调试难度

### 监控和改进

#### 监控指标

- 构建成功率
- 构建时间
- 错误类型统计

#### 改进计划

- 添加构建性能优化
- 完善错误提示信息
- 支持并行构建

---

**文档版本**: 1.0
**最后更新**: 2025年9月15日
**维护者**: AI-Infra Team
