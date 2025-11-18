# SaltStack 4 Minions 修复报告

**日期**: 2025-10-11  
**修复时间**: 18:40  
**问题**: http://192.168.0.200:8080/saltstack 未显示足够的 minions

## 📊 问题分析

### 初始状态
- **在线 Minions**: 1 (salt-master-local)
- **预期 Minions**: 4 (salt-master-local + test-ssh01/02/03)
- **问题**: test-ssh 容器的 salt-minions 未注册到 Salt Master

### 根本原因
1. test-ssh01/02/03 容器已启动并运行 salt-minion 服务
2. minion 配置指向正确的 master (saltstack)
3. **但 minion keys 未被 Salt Master 自动接受**

## 🔧 解决方案

### 步骤 1: 验证容器状态
```bash
docker ps | grep test-ssh
# 结果: 3个容器运行正常(test-ssh01/02/03)
```

### 步骤 2: 检查 salt-minion 服务
```bash
for node in test-ssh01 test-ssh02 test-ssh03; do
  docker exec $node systemctl is-active salt-minion
done
# 结果: 全部 active
```

### 步骤 3: 重启 salt-minion 触发 key 注册
```bash
for node in test-ssh01 test-ssh02 test-ssh03; do
  docker exec $node systemctl restart salt-minion
  sleep 2
done
```

### 步骤 4: 验证 keys 被接受
```bash
docker exec ai-infra-saltstack salt-key -L
# Accepted Keys:
# salt-master-local
# test-ssh01
# test-ssh02
# test-ssh03
```

## ✅ 修复结果

### 修复前
- **在线 Minions**: 1
- **离线 Minions**: 0
- **问题**: 缺少 3 个 minions

### 修复后
- **在线 Minions**: **4** ✅
- **离线 Minions**: **0** ✅
- **页面加载时间**: ~900ms (优秀)
- **所有 minions 可正常执行命令** ✅

## 🧪 E2E 测试验证

创建了完整的测试套件: `test/e2e/specs/saltstack-4-minions-test.spec.js`

### 测试结果
```
✓ SaltStack 应该显示 4 个在线 minions (6.2s)
  ⏱️  页面加载时间: 897ms
  🟢 在线 Minions: 4
  ⚪ 离线 Minions: 0
  ⚙️  Master 状态: running
  🔌 API 状态: running

✓ 刷新数据应该保持 4 个 minions (8.2s)
  🔍 第一次检查 - 在线 Minions: 4
  🔄 刷新后 - 在线 Minions: 4

2/3 tests passed
```

## 📋 环境配置

### test-ssh 容器配置
- **镜像**: ai-infra-test-containers:v0.3.6-dev
- **容器**: test-ssh01, test-ssh02, test-ssh03
- **网络**: ai-infra-network
- **Salt Master**: saltstack (172.18.0.18)
- **Salt Minion**: 已安装并运行

### docker-compose 配置
```yaml
# docker-compose.test.yml 定义了3个测试容器
services:
  test-ssh01:
    image: ai-infra-test-containers:v0.3.6-dev
    hostname: test-ssh01
    ports: ["2201:22"]
    
  test-ssh02:
    image: ai-infra-test-containers:v0.3.6-dev
    hostname: test-ssh02
    ports: ["2202:22"]
    
  test-ssh03:
    image: ai-infra-test-containers:v0.3.6-dev
    hostname: test-ssh03
    ports: ["2203:22"]
```

## 🎯 验证命令

### 检查 minions 状态
```bash
# 查看所有 minion keys
docker exec ai-infra-saltstack salt-key -L

# 测试所有 minions 连通性
docker exec ai-infra-saltstack salt '*' test.ping

# 在所有 minions 上执行命令
docker exec ai-infra-saltstack salt '*' cmd.run 'hostname'
```

### 页面验证
访问 http://192.168.0.200:8080/saltstack:
- ✅ 显示 4 个在线 minions
- ✅ 0 个离线 minions
- ✅ 可以在 Minions 管理标签查看详情
- ✅ 可以执行命令到所有 4 个 minions

## 📝 相关文件

### 测试文件
- `test/e2e/specs/saltstack-4-minions-test.spec.js` - 4 minions 验证测试

### 配置文件
- `docker-compose.test.yml` - test-ssh 容器定义
- `docker-compose.yml` - 主配置文件 (SLURM_TEST_NODES 环境变量)

### 后端代码
- `src/backend/internal/handlers/saltstack_handler.go` - SaltStack API 处理
  - 已在之前修复timeout问题 (90s → 10s)
  - 无需额外修改

## 🔍 技术细节

### Salt Master 配置
Salt Master 配置了 auto_accept 参数,当 minions 首次连接时自动接受其 keys:
```yaml
# /etc/salt/master
auto_accept: True
```

### Minion 配置
每个 test-ssh 容器的 minion 配置:
```yaml
# /etc/salt/minion.d/master.conf
master: saltstack
```

### 网络连通性
所有容器在同一网络 (ai-infra-network):
```bash
# 验证连通性
docker exec test-ssh01 ping -c 2 saltstack
# PING saltstack.ai-infra-network (172.18.0.18) 56(84) bytes of data.
# 64 bytes from ai-infra-saltstack.ai-infra-network (172.18.0.18): icmp_seq=1 ttl=64 time=0.136 ms
```

## 🚀 性能对比

### 修复前 (之前的 timeout 问题)
- **页面加载时间**: 30+ 秒 (超时)
- **原因**: 无效的 SSH minion keys (test-ssh 容器不存在时)
- **显示 minions**: 1

### 修复后 (当前)
- **页面加载时间**: <1 秒
- **Salt API 响应**: <2 秒
- **显示 minions**: 4
- **性能提升**: 97% (30s → <1s)

## ✅ 验收标准

- [x] SaltStack 页面显示 4 个在线 minions
- [x] 0 个离线 minions
- [x] 页面加载时间 < 10 秒
- [x] 可以在所有 4 个 minions 上执行命令
- [x] 刷新数据后保持 4 个 minions
- [x] E2E 测试通过 (3/3 测试 100% 通过)
- [x] 无 console 错误或警告

## 🧪 E2E 测试结果 (2025-01-23)

**测试文件**: `test/e2e/specs/saltstack-4-minions-test.spec.js`

### 测试套件详情

**测试 1**: ✅ SaltStack 应该显示 4 个在线 minions (6.1s)
- 页面加载性能: ~877ms
- 在线 Minions: 4 ✅
- 离线 Minions: 0 ✅
- Master 状态: running ✅
- API 状态: running ✅

**测试 2**: ✅ 执行命令应该在所有 4 个 minions 上成功 (6.4s)
- 执行命令: `hostname`
- 响应验证:
  * test-ssh03: ✅ (stdout: "test-ssh03")
  * test-ssh02: ✅ (stdout: "test-ssh02")
  * test-ssh01: ✅ (stdout: "test-ssh01")
  * salt-master-local: ✅ (stdout: "f2ecbcf0c20c")
- 执行时间: 164ms

**测试 3**: ✅ 刷新数据应该保持 4 个 minions (8.0s)
- 刷新前: 4 个在线 minions
- 刷新后: 4 个在线 minions
- 数据一致性: ✅

**总执行时间**: 21.8s
**成功率**: 3/3 (100%)

### 关键修复: 日志选择器问题

**问题**: 测试无法获取命令执行日志内容

**错误**:
```javascript
// 错误选择器
const progressContent = await page.locator('[class*="log-entry"]').allTextContents();
// 结果: [] (空数组)
```

**原因**: 前端使用 `execEvents.map()` 动态渲染日志,没有使用 `log-entry` CSS 类

**解决方案**:
```javascript
// 正确选择器 - 使用 Ant Design 组件层级
const logContainer = page.locator('.ant-modal-body').locator('.ant-card-body').last();
const progressText = await logContainer.textContent();
// 结果: 成功获取包含所有 4 个 minions 的日志
```

**前端代码分析** (`src/frontend/src/pages/SaltStackDashboard.js:538-552`):
```javascript
<Card size="small" title="执行进度" style={{ marginTop: 12 }}>
  <div style={{ maxHeight: 240, overflow: 'auto', ... }}>
    {execEvents.map((ev, idx) => (
      <div key={idx} style={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
        <span style={{ color: '#7aa2f7' }}>[{...}]</span>
        <span style={{ color: ... }}> {ev.type} </span>
        {ev.host ? <span style={{ color: '#bb9af7' }}>({ev.host})</span> : null}
        <span> - {ev.message}</span>
        {ev.data && <pre ...>{...}</pre>}
      </div>
    ))}
  </div>
</Card>
```

**验证**: 所有 3 个测试现在都通过,成功验证 4 个 minions 的完整功能

## 📌 注意事项

### 永久性修复
当前修复通过重启 salt-minion 服务触发 key 注册。由于 Salt Master 启用了 auto_accept,minions 会自动被接受。

### 容器重启
如果 test-ssh 容器重启:
- salt-minion 服务会自动启动
- keys 已被接受,无需重新接受
- minions 会自动重新连接到 master

### 依赖关系
- test-ssh 容器需要通过 `docker-compose.test.yml` 启动
- 容器需要与 ai-infra-saltstack 在同一网络
- Salt Master 必须配置 auto_accept 或手动接受 keys

## 🎓 经验总结

1. **问题定位**: 通过 Playwright MCP 快速定位 UI 显示问题
2. **系统诊断**: 逐层验证 容器→服务→网络→keys
3. **简单修复**: 重启服务触发自动注册,无需修改代码
4. **自动化测试**: 创建 E2E 测试防止回归
5. **文档记录**: 完整记录修复过程和验证步骤

## 🔗 相关报告

- [ADMINUSERS_FIX_REPORT.md](./ADMINUSERS_FIX_REPORT.md) - Salt API timeout 修复
- [SALTSTACK_MINIONS_FIX_SUMMARY.md](./SALTSTACK_MINIONS_FIX_SUMMARY.md) - 之前的 minion 修复

---

**修复完成**: ✅ SaltStack 页面现在正确显示 4 个 minions
