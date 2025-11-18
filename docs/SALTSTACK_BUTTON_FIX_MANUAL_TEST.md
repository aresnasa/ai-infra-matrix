# SaltStack 执行按钮修复 - 手动测试指南

## 问题描述

执行命令后，按钮显示"命令执行完成"，但按钮还在转圈（loading 状态），无法正确退出。

## 已实施的修复

在 `src/frontend/src/pages/SaltStackDashboard.js` 中添加了详细的 SSE 事件日志：

```javascript
const startSSE = (opId) => {
  closeSSE();
  const url = saltStackAPI.streamProgressUrl(opId);
  const es = new EventSource(url, { withCredentials: false });
  sseRef.current = es;
  es.onmessage = (evt) => {
    try {
      const data = JSON.parse(evt.data);
      console.log('[SSE事件]', data.type, data);  // ← 新增日志
      setExecEvents((prev) => [...prev, data]);
      
      if (data.type === 'complete' || data.type === 'error') {
        console.log('[SSE] 收到完成事件，准备停止');  // ← 新增日志
        setTimeout(() => {
          console.log('[SSE] 设置 execRunning = false');  // ← 新增日志
          setExecRunning(false);
          closeSSE();
        }, 300);
      }
    } catch (err) {
      console.error('[SSE] 解析消息失败:', err);
    }
  };
  es.onerror = (err) => {
    console.error('[SSE] 连接错误:', err);
    closeSSE();
    setExecRunning(false);
  };
};
```

## 手动测试步骤

### 前提条件

1. **构建前端代码**（必须执行）：
   ```bash
   cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/src/frontend
   npm run build
   ```

2. **重启服务**（如果需要）：
   ```bash
   docker-compose restart frontend nginx
   ```

### 测试步骤

1. **打开浏览器开发者工具**
   - 访问 http://192.168.0.200:8080/saltstack
   - 按 F12 或右键 → 检查
   - 切换到 **Console** 标签页

2. **执行测试命令**
   - 点击【执行命令】按钮
   - 在代码框中输入：`hostname`
   - 目标节点保持：`*`
   - 点击【执 行】按钮

3. **观察控制台日志**

   **期望看到的日志流**：
   ```
   [SSE事件] - {type: "-", ...}
   [SSE事件] step-log {type: "step-log", minion: "salt-master-local", ...}
   [SSE事件] step-log {type: "step-log", minion: "test-ssh01", ...}
   [SSE事件] step-log {type: "step-log", minion: "test-ssh02", ...}
   [SSE事件] step-log {type: "step-log", minion: "test-ssh03", ...}
   [SSE事件] step-done {type: "step-done", ...}
   [SSE事件] complete {type: "complete", ...}
   [SSE] 收到完成事件，准备停止
   [SSE] 设置 execRunning = false
   ```

4. **验证按钮状态**

   ✅ **修复成功的表现**：
   - 执行时：按钮显示"loading 执 行"（禁用状态，有转圈图标）
   - 完成后：按钮变为"执 行"（可点击状态，无转圈图标）
   - 控制台显示：`[SSE] 设置 execRunning = false`
   - 可以再次点击执行按钮

   ❌ **修复失败的表现**：
   - 看到"命令执行完成"
   - 但按钮还显示"loading 执 行"（一直转圈）
   - 控制台没有显示：`[SSE] 设置 execRunning = false`
   - 无法再次点击执行按钮（需要刷新页面）

## 调试指南

### 如果看不到日志

**可能原因**：前端代码没有重新构建

**解决方法**：
```bash
# 1. 清理旧的构建
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/src/frontend
rm -rf build/

# 2. 重新构建
npm run build

# 3. 确认构建产物包含修改
grep -r "SSE事件" build/static/js/*.js
# 应该能找到字符串

# 4. 重启服务
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix
docker-compose restart frontend nginx
```

### 如果按钮还是转圈

**检查点 1：浏览器缓存**
- 硬刷新：Ctrl+Shift+R (Windows/Linux) 或 Cmd+Shift+R (Mac)
- 或者打开隐私/无痕窗口测试

**检查点 2：SSE 事件流**
- 控制台看到 `[SSE事件] complete` 吗？
  - ✅ 看到 → 说明后端正常，检查前端代码
  - ❌ 没看到 → 检查后端 SSE 实现

**检查点 3：错误日志**
- 控制台有 `[SSE] 解析消息失败` 或 `[SSE] 连接错误` 吗？
  - 如果有，检查 SSE 返回的数据格式

## 预期的 SSE 事件流

根据你提供的日志，正常的事件流应该是：

```
事件1: type="-"           (开始标记)
事件2: type="step-log"    (minion: salt-master-local)
事件3: type="step-log"    (minion: test-ssh01)
事件4: type="step-log"    (minion: test-ssh02)
事件5: type="step-log"    (minion: test-ssh03)
事件6: type="step-done"   (所有 minion 执行完成)
事件7: type="complete"    (整体完成) ← 这里应该停止 loading
```

## 验证成功标准

- [ ] 控制台能看到所有 `[SSE事件]` 日志
- [ ] 最后一个事件是 `type: "complete"`
- [ ] 看到 `[SSE] 收到完成事件，准备停止`
- [ ] 看到 `[SSE] 设置 execRunning = false`
- [ ] 按钮从 "loading 执 行" 变为 "执 行"
- [ ] 可以再次点击执行按钮
- [ ] 无需刷新页面即可继续使用

## 如需帮助

如果按照以上步骤测试仍有问题，请提供：

1. **控制台完整日志**（包含所有 `[SSE` 开头的日志）
2. **Network 标签页的 SSE 请求详情**
   - 找到 `/api/saltstack/stream/` 开头的请求
   - 查看 EventStream 标签页的所有事件
3. **按钮的最终 HTML 状态**
   - 右键按钮 → 检查元素
   - 复制完整的 button 标签

## 相关文件

- 修改的源文件：`src/frontend/src/pages/SaltStackDashboard.js` (第 147-172 行)
- E2E 测试：`test/e2e/specs/saltstack-exec.spec.js` (第 277 行)
- 详细文档：`docs/SALTSTACK_UI_FIXES.md`

---

**最后更新**：2025年10月11日  
**修复版本**：v0.3.7  
**测试状态**：待验证（需先构建前端）
