# SLURM + SaltStack 集成测试指南

## 系统架构概览

### 容器组成
1. **ai-infra-backend** - Go 后端 API 服务
2. **ai-infra-saltstack** - Salt Master 容器
3. **ai-infra-slurm-master** - SLURM Master 容器
4. **ai-infra-slurmd-01~03** - SLURM 计算节点（如果配置）
5. **Salt Minions** - 7个 minions（test-rocky01-03, test-ssh01-03, salt-master-local）

### 集成方式
- **Salt API**: 用于在 Salt minions 上执行命令（管道、shell 操作等）
- **SLURM SSH**: 后端通过 SSH 连接到 SLURM master 执行 SLURM 命令
- **混合使用**: Salt 管理节点，SLURM 管理作业

---

## 1. Salt API 测试

### 1.1 基础命令执行
```bash
TOKEN=$(curl -s -X POST http://192.168.3.91:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

# 测试简单命令
curl -s -X POST http://192.168.3.91:8080/api/slurm/saltstack/execute \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"test.ping","target":"*"}' | jq
```

**预期输出：**
```json
{
  "data": {
    "result": {
      "return": [{
        "salt-master-local": true,
        "test-rocky01": true,
        "test-rocky02": true,
        "test-rocky03": true,
        "test-ssh01": true,
        "test-ssh02": true,
        "test-ssh03": true
      }]
    },
    "success": true
  }
}
```

### 1.2 Shell 管道和命令组合
```bash
# 管道命令
curl -s -X POST http://192.168.3.91:8080/api/slurm/saltstack/execute \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"*","arguments":"ps aux | grep python | wc -l"}' \
  | jq -r '.data.result.return[0]'

# 重定向和命令链
curl -s -X POST http://192.168.3.91:8080/api/slurm/saltstack/execute \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"test-ssh01","arguments":"echo hello > /tmp/test.txt && cat /tmp/test.txt"}' \
  | jq -r '.data.result.return[0]["test-ssh01"]'

# 多管道
curl -s -X POST http://192.168.3.91:8080/api/slurm/saltstack/execute \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"*","arguments":"ls -la / | head -5"}' \
  | jq -r '.data.result.return[0] | to_entries[] | "\(.key): \(.value)"'
```

### 1.3 自动检测 Shell 特性
后端会自动检测以下字符并启用 `python_shell=True`：
- 管道: `|`
- 重定向: `>`, `>>`, `<`, `2>`
- 逻辑运算符: `&&`, `||`, `;`
- 变量: `$VAR`, `$()`
- 命令替换: `` ` ``
- 子 shell: `()`
- 通配符: `*`, `?`, `[]`

**后端日志确认：**
```
[SaltStack] Detected shell metacharacters, enabling python_shell=True
[SaltStack] Request JSON: [{"arg":["ps aux | grep python"],"client":"local","fun":"cmd.run","kwarg":{"python_shell":true},"tgt":"*"}]
```

---

## 2. SLURM 命令测试

### 2.1 查看集群状态
```bash
# sinfo - 节点信息
curl -s -X POST http://192.168.3.91:8080/api/slurm/exec \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"command":"sinfo"}' | jq -r '.output'

# sinfo -Nel - 详细节点列表
curl -s -X POST http://192.168.3.91:8080/api/slurm/exec \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"command":"sinfo -Nel"}' | jq -r '.output'
```

**预期输出：**
```
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      6  idle* test-rocky[01-03],test-ssh[01-03]
```

### 2.2 查看作业队列
```bash
# squeue - 当前队列
curl -s -X POST http://192.168.3.91:8080/api/slurm/exec \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"command":"squeue"}' | jq -r '.output'

# 格式化输出
curl -s -X POST http://192.168.3.91:8080/api/slurm/exec \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"command":"squeue -o \"%.18i %.9P %.20j %.8u %.8T %.10M %.6D %R\""}' \
  | jq -r '.output'
```

### 2.3 查看作业详情
```bash
# scontrol show job
curl -s -X POST http://192.168.3.91:8080/api/slurm/exec \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"command":"scontrol show job 1"}' | jq -r '.output'

# 查看节点详情
curl -s -X POST http://192.168.3.91:8080/api/slurm/exec \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"command":"scontrol show nodes test-rocky01"}' | jq -r '.output'
```

### 2.4 提交测试作业
```bash
# 在 SLURM master 上创建测试脚本
docker exec ai-infra-slurm-master bash -c 'cat > /tmp/test_job.sh << "EOF"
#!/bin/bash
#SBATCH --job-name=test_job
#SBATCH --output=/tmp/test_job_%j.out
#SBATCH --error=/tmp/test_job_%j.err
#SBATCH --time=00:01:00
#SBATCH --nodes=1
#SBATCH --ntasks=1

echo "Job started at: $(date)"
hostname
sleep 5
echo "Job completed at: $(date)"
EOF
chmod +x /tmp/test_job.sh'

# 提交作业
docker exec ai-infra-slurm-master sbatch /tmp/test_job.sh

# 查看作业状态
docker exec ai-infra-slurm-master squeue

# 查看作业输出（作业完成后）
docker exec ai-infra-slurm-master cat /tmp/test_job_*.out
```

---

## 3. 系统诊断

### 3.1 SLURM 诊断
```bash
curl -s -X GET "http://192.168.3.91:8080/api/slurm/diagnostics" \
  -H "Authorization: Bearer $TOKEN" | jq
```

### 3.2 Salt API 诊断
```bash
# 检查 Salt minion 状态
curl -s -X POST http://192.168.3.91:8080/api/slurm/saltstack/execute \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"test.version","target":"*"}' | jq

# 检查 minion 磁盘使用
curl -s -X POST http://192.168.3.91:8080/api/slurm/saltstack/execute \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"*","arguments":"df -h | head -5"}' | jq
```

### 3.3 后端日志查看
```bash
# 查看 Salt 执行日志
docker logs ai-infra-backend --tail=50 | grep "SaltStack"

# 查看 SLURM SSH 连接日志
docker logs ai-infra-backend --tail=50 | grep "SSH"

# 实时查看日志
docker logs -f ai-infra-backend | grep -E "(SaltStack|SSH|SLURM)"
```

---

## 4. 常见问题和解决方案

### 4.1 节点显示 NOT_RESPONDING
**问题：** `sinfo` 显示节点状态为 `idle*`（NOT_RESPONDING）

**原因：** 计算节点上没有运行 slurmd 守护进程

**解决方案（选项1 - 测试环境）：**
在 SLURM master 上本地测试：
```bash
# 配置 master 节点也作为计算节点
docker exec ai-infra-slurm-master bash -c '
echo "NodeName=$(hostname) CPUs=1 State=UNKNOWN" >> /etc/slurm/slurm.conf
echo "PartitionName=local Nodes=$(hostname) Default=YES MaxTime=INFINITE State=UP" >> /etc/slurm/slurm.conf
scontrol reconfigure
'
```

**解决方案（选项2 - 生产环境）：**
在计算节点上安装并启动 slurmd：
```bash
# 通过 Salt 在所有节点上安装 slurmd
curl -s -X POST http://192.168.3.91:8080/api/slurm/saltstack/execute \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"test-*","arguments":"yum install -y slurm-slurmd && systemctl start slurmd"}' | jq
```

### 4.2 Salt API 返回空结果
**问题：** `{"return": [{}]}`

**原因：** 
1. Target 参数使用了列表模式但传入了通配符 `["*"]`
2. 请求格式不是数组

**已修复：**
- ✅ 自动检测 `["*"]` 并使用 glob 模式
- ✅ 请求体自动包装为数组格式 `[{...}]`
- ✅ 自动检测 shell 元字符并启用 `python_shell=True`

### 4.3 管道命令失败
**问题：** `error: unsupported option (BSD syntax)`

**原因：** Salt `cmd.run` 默认不通过 shell 执行

**已修复：**
后端自动检测管道、重定向等字符，并设置 `python_shell=True`

**验证：**
```bash
# 后端日志应该显示：
[SaltStack] Detected shell metacharacters, enabling python_shell=True
```

---

## 5. 性能测试

### 5.1 并发执行测试
```bash
# 在所有 minions 上并发执行
curl -s -X POST http://192.168.3.91:8080/api/slurm/saltstack/execute \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"*","arguments":"sleep 3 && echo done"}' | jq

# 验证响应时间（应该约 3 秒，不是 3*7=21 秒）
```

### 5.2 大规模节点测试
```bash
# 测试大量输出
curl -s -X POST http://192.168.3.91:8080/api/slurm/saltstack/execute \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"*","arguments":"ps aux"}' \
  | jq '.data.result.return[0] | to_entries | length'
```

---

## 6. E2E 自动化测试

### 6.1 Playwright 测试
```bash
cd test/e2e
BASE_URL=http://192.168.3.91:8080 npx playwright test \
  specs/saltstack-quick-test.spec.js \
  --config=playwright.config.js \
  --workers=1 \
  --timeout=90000
```

**测试覆盖：**
- ✅ 登录功能
- ✅ 导航到 SLURM 页面
- ✅ SaltStack 集成 tab
- ✅ 命令执行（test.ping）
- ✅ 输出格式验证
- ✅ 复制功能

---

## 7. 监控和告警

### 7.1 实时监控脚本
创建 `monitor.sh`：
```bash
#!/bin/bash
TOKEN=$(curl -s -X POST http://192.168.3.91:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

while true; do
  echo "=== $(date) ==="
  
  # SLURM 队列
  echo "SLURM Queue:"
  curl -s -X POST http://192.168.3.91:8080/api/slurm/exec \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"command":"squeue"}' | jq -r '.output'
  
  # Salt Minions
  echo -e "\nSalt Minions:"
  curl -s -X POST http://192.168.3.91:8080/api/slurm/saltstack/execute \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"function":"test.ping","target":"*"}' \
    | jq -r '.data.result.return[0] | to_entries[] | "\(.key): \(.value)"'
  
  sleep 30
done
```

---

## 8. 总结

### 已实现功能
- ✅ Salt API 集成（支持所有 Salt 函数）
- ✅ Shell 命令组合自动支持（管道、重定向等）
- ✅ SLURM 命令执行（通过 SSH）
- ✅ 详细的调试日志
- ✅ 错误处理和重试机制
- ✅ E2E 自动化测试

### API 端点总结
| 功能 | 端点 | 方法 | 说明 |
|------|------|------|------|
| Salt 命令执行 | `/api/slurm/saltstack/execute` | POST | 在 Salt minions 上执行命令 |
| SLURM 命令执行 | `/api/slurm/exec` | POST | 在 SLURM master 上执行命令 |
| 系统诊断 | `/api/slurm/diagnostics` | GET | 获取系统诊断信息 |
| Salt 集成状态 | `/api/slurm/saltstack/integration` | GET | 获取 Salt 集成状态 |

### 下一步优化建议
1. 添加作业提交 API（sbatch）
2. 添加作业取消 API（scancel）
3. 实现作业状态实时监控（WebSocket）
4. 添加节点资源使用率图表
5. 集成作业历史和统计分析
