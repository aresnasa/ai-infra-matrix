# 工具和配置文件目录

本目录包含项目的各种工具、脚本和配置文件。

## 目录结构

- **test-scripts/** - 各种测试脚本
  - JavaScript测试文件
  - Shell测试脚本  
  - HTML测试页面
  
- **scripts/** - 实用脚本
  - 部署脚本
  - 配置脚本
  
- **configs/** - 配置文件
  - Kubernetes配置
  - Docker配置
  - Node.js配置
  
- **fix_imports.py** - Go导入路径修复工具
- **Dockerfile.ai-test** - AI测试Docker文件
- **Dockerfile.test** - 通用测试Docker文件

## 使用说明

### 测试脚本
测试脚本可以从项目根目录运行：
```bash
# 运行AI助手测试
./tools/test-scripts/test_ai_assistant.sh

# 运行异步测试
./tools/test-scripts/test-ai-async.sh
```

### 配置文件
配置文件可以根据需要复制到对应位置使用。

---
**维护**: 自动整理脚本
**更新**: 2025年7月23日
