#!/bin/bash

# 测试结果报告生成器
# 生成详细的端到端测试报告

REPORT_DIR="../reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$REPORT_DIR/e2e-test-report-$TIMESTAMP.md"

# 创建报告目录
mkdir -p "$REPORT_DIR"

# 生成测试报告
generate_test_report() {
    local total_tests=$1
    local passed_tests=$2
    local failed_tests=$((total_tests - passed_tests))
    local success_rate=$((passed_tests * 100 / total_tests))
    
    cat > "$REPORT_FILE" << EOF
# Ansible Playbook Generator - 端到端测试报告

## 测试概览

- **测试时间**: $(date)
- **测试环境**: Docker Compose
- **总测试数**: $total_tests
- **通过测试**: $passed_tests
- **失败测试**: $failed_tests
- **成功率**: $success_rate%

## 测试详情

### 1. 服务启动和连接测试
- **目标**: 验证所有Docker服务正常启动
- **测试内容**: 
  - 后端服务健康检查
  - 前端服务可访问性
  - 服务间连接测试

### 2. 用户认证测试
- **目标**: 验证用户认证系统正常工作
- **测试内容**:
  - 管理员登录
  - Token获取和验证
  - 认证状态持久性

### 3. 时区配置测试
- **目标**: 验证容器时区配置正确
- **测试内容**:
  - 后端容器时区 (期望: Asia/Shanghai CST)
  - 前端容器时区 (期望: Asia/Shanghai CST)
  - 时间同步检查

### 4. 项目管理测试
- **目标**: 验证项目CRUD操作正常
- **测试内容**:
  - 创建测试项目
  - 项目配置验证
  - 项目权限检查

### 5. 前端可访问性测试
- **目标**: 验证前端应用正常访问
- **测试内容**:
  - HTTP响应状态检查
  - 页面加载验证

### 6. 增强健康检查测试
- **目标**: 验证系统各组件健康状态
- **测试内容**:
  - 基础健康检查 (/health)
  - 数据库连接检查 (/health/db)
  - Redis连接检查 (/health/redis)
  - API文档可访问性 (/swagger/index.html)

### 7. 用户管理功能测试
- **目标**: 验证用户管理系统完整性
- **测试内容**:
  - 用户注册功能
  - 用户登录验证
  - 用户资料管理
  - 权限控制验证
  - 管理员功能测试

### 8. 垃圾箱功能测试
- **目标**: 验证垃圾箱/回收站功能
- **测试内容**:
  - 软删除功能 (PATCH /projects/{id}/soft-delete)
  - 垃圾箱列表查看 (GET /projects/trash)
  - 项目恢复功能 (PATCH /projects/{id}/restore)
  - 永久删除功能 (DELETE /projects/{id}/force)
  - 垃圾箱状态验证

### 9. Playbook预览测试
- **目标**: 验证Playbook预览功能
- **测试内容**:
  - 预览API调用 (POST /playbook/preview)
  - 验证分数检查
  - 预览内容格式验证

### 10. 包生成测试
- **目标**: 验证Playbook包生成功能
- **测试内容**:
  - 包生成API调用 (POST /playbook/package)
  - ZIP文件创建验证
  - 文件大小检查

### 11. ZIP下载测试
- **目标**: 验证ZIP包下载功能
- **测试内容**:
  - 下载API调用 (GET /playbook/download-zip/{path})
  - 文件完整性验证
  - 下载速度检查

### 12. Playbook生成测试
- **目标**: 验证单个Playbook生成功能
- **测试内容**:
  - 生成API调用 (POST /playbook/generate)
  - 生成ID返回验证
  - 文件名格式检查

### 13. 单文件下载测试
- **目标**: 验证单个Playbook文件下载功能
- **测试内容**:
  - 下载API调用 (GET /playbook/download/{id})
  - 文件内容验证
  - YAML格式检查

## 测试环境信息

### Docker 容器状态
\`\`\`
$(docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "Docker信息获取失败")
\`\`\`

### 系统资源使用
\`\`\`
$(docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" 2>/dev/null || echo "资源信息获取失败")
\`\`\`

## 测试建议

### 成功率评估
- **优秀** (90-100%): 所有核心功能正常，系统可投入生产
- **良好** (80-89%): 大部分功能正常，需要修复少量问题
- **一般** (70-79%): 存在一些功能问题，需要进一步开发
- **较差** (60-69%): 存在较多问题，不建议投入使用
- **失败** (<60%): 系统存在严重问题，需要重新开发

### 当前状态: $(if [ $success_rate -ge 90 ]; then echo "优秀"; elif [ $success_rate -ge 80 ]; then echo "良好"; elif [ $success_rate -ge 70 ]; then echo "一般"; elif [ $success_rate -ge 60 ]; then echo "较差"; else echo "失败"; fi)

## 问题诊断

### 常见问题及解决方案

1. **服务启动失败**
   - 检查Docker服务状态
   - 检查端口冲突
   - 查看容器日志

2. **认证失败**
   - 验证默认管理员账户
   - 检查JWT配置
   - 确认数据库连接

3. **API调用失败**
   - 检查网络连接
   - 验证API端点
   - 查看后端服务日志

4. **文件下载失败**
   - 检查文件路径权限
   - 验证存储空间
   - 确认临时目录配置

## 下一步操作

$(if [ $success_rate -ge 90 ]; then
cat << 'NEXT_STEPS_SUCCESS'
✅ **系统状态良好**
- 可以进行生产环境部署
- 建议进行性能测试
- 考虑添加监控和告警

NEXT_STEPS_SUCCESS
elif [ $success_rate -ge 70 ]; then
cat << 'NEXT_STEPS_PARTIAL'
⚠️ **需要改进**
- 修复失败的测试用例
- 补充缺失的功能实现
- 重新运行测试验证

NEXT_STEPS_PARTIAL
else
cat << 'NEXT_STEPS_FAILED'
❌ **需要修复**
- 检查系统架构设计
- 修复核心功能问题
- 完善错误处理机制
- 重新进行开发和测试

NEXT_STEPS_FAILED
fi)

---
*报告生成时间: $(date)*
*测试环境: Docker Compose*
*生成工具: E2E Test Suite v1.0*
EOF

    echo "测试报告已生成: $REPORT_FILE"
}

# 导出函数供其他脚本使用
export -f generate_test_report
