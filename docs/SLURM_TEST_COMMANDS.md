# SLURM测试和验证命令备忘清单

## 快速参考

### 1. 运行Playwright测试

```bash
# 运行快速API测试（推荐）
BASE_URL=http://192.168.0.200:8080 \
  npx playwright test test/e2e/specs/slurm-quick-test.spec.js

# 运行完整测试
BASE_URL=http://192.168.0.200:8080 \
  npx playwright test test/e2e/specs/slurm-async-loading-test.spec.js

# 生成HTML报告
BASE_URL=http://192.168.0.200:8080 \
  npx playwright test test/e2e/specs/slurm-quick-test.spec.js \
  --reporter=html

# 查看HTML报告
npx playwright show-report playwright-report
```

### 2. 手动API测试

```bash
# 登录获取Token
TOKEN=$(curl -s -X POST http://192.168.0.200:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

# 测试集群摘要
curl -H "Authorization: Bearer $TOKEN" \
  http://192.168.0.200:8080/api/slurm/summary | jq

# 测试节点列表
curl -H "Authorization: Bearer $TOKEN" \
  http://192.168.0.200:8080/api/slurm/nodes | jq

# 测试作业队列
curl -H "Authorization: Bearer $TOKEN" \
  http://192.168.0.200:8080/api/slurm/jobs | jq

# 执行SLURM命令
curl -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -X POST http://192.168.0.200:8080/api/slurm/exec \
  -d '{"command":"sinfo"}' | jq

# 获取诊断信息
curl -H "Authorization: Bearer $TOKEN" \
  http://192.168.0.200:8080/api/slurm/diagnostics | jq
```

### 3. 检查服务状态

```bash
# 检查Docker容器
docker-compose -f docker-compose.test.yml ps | grep -E "(frontend|backend)"

# 检查前端服务
curl -s http://192.168.0.200:8080/ | head -10

# 检查Backend健康状态
curl http://192.168.0.200:8082/health
```

### 4. 修复节点down*状态

```bash
# 在计算节点上执行
sudo yum install -y slurm-slurmd munge

# 复制munge密钥
sudo scp slurm-master:/etc/munge/munge.key /etc/munge/
sudo chown munge:munge /etc/munge/munge.key
sudo chmod 400 /etc/munge/munge.key

# 启动服务
sudo systemctl enable --now munge
sudo systemctl enable --now slurmd

# 在master上验证
sinfo -Nel
```

### 5. 查看测试结果

```bash
# 查看测试截图
ls -lh test-screenshots/

# 查看测试报告
cat docs/SLURM_TEST_EXECUTION_SUMMARY.md
cat docs/SLURM_ASYNC_LOADING_TEST_REPORT.md
```

## 测试文件位置

```
test/e2e/specs/
├── slurm-quick-test.spec.js              # 快速API测试 ✅
└── slurm-async-loading-test.spec.js      # 完整异步加载测试 ⏳

docs/
├── SLURM_TEST_EXECUTION_SUMMARY.md       # 测试执行总结 ✅
├── SLURM_ASYNC_LOADING_TEST_REPORT.md    # 完整测试报告 ✅
├── SLURM_FRONTEND_OPTIMIZATION.md        # 前端优化文档 ✅
├── SLURM_SYNC_AND_OPS_INTEGRATION_SUMMARY.md  # Backend优化总结 ✅
└── SLURM_NODE_DOWN_FIX_GUIDE.md          # 节点修复指南 ✅
```

## 预期测试结果

```
测试通过率:
✅ API测试: 100% (3/3)
✅ 功能测试: 100% (5/5核心API)
⏳ 前端测试: 待优化 (登录问题)

性能提升:
✅ API响应: 30秒+ → <1秒 (99.7%提升)
✅ 首屏渲染: 3秒 → <10ms (99.7%提升)
✅ 完整加载: 5秒+ → <1秒 (80%提升)
```
