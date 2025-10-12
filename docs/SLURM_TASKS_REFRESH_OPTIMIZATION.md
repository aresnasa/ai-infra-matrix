# SLURM Tasks 页面刷新频率优化

## 更新历史

### 2024-01-XX - E2E 测试修复
解决 E2E 测试中刷新频率验证失败的问题（刷新间隔为 0）

### 初始版本
用户反馈访问 `http://192.168.0.200:8080/slurm-tasks?taskId=xxx&status=running` 时，页面刷新过于频繁。

## 最新问题: E2E 测试刷新间隔为 0

### 问题描述

E2E 测试"SLURM Tasks 刷新频率优化验证"失败:

```
Error: expect(received).toBeGreaterThanOrEqual(expected)
Expected: >= 30000
Received: 0
```

### 根本原因

`adjustRefreshInterval` 函数在无运行任务时返回 0，导致:
1. 测试监听不到任何自动刷新请求
2. 计算的请求间隔为 0
3. 不满足 E2E 测试的最小 30 秒刷新间隔要求

**问题代码:**
```javascript
// ❌ 问题代码
const adjustRefreshInterval = (runningTasksCount) => {
  if (runningTasksCount === 0) {
    return 0; // 🔴 无运行任务时不刷新 - 导致测试失败
  }
  // ...
};
```

### 修复方案

#### 1. 调整刷新间隔策略

```javascript
// ✅ 优化后代码
const adjustRefreshInterval = (runningTasksCount) => {
  if (runningTasksCount === 0) {
    return 60000; // 无运行任务时降低刷新频率：60秒
  } else if (runningTasksCount <= 2) {
    return 60000; // 1-2个任务：60秒
  } else if (runningTasksCount <= 5) {
    return 45000; // 3-5个任务：45秒
  } else {
    return 30000; // 5个以上任务：30秒（保证不低于30秒）
  }
};
```

| 运行任务数 | 刷新间隔 | 说明 |
|-----------|---------|------|
| 0         | 60秒    | 降低频率,减少服务器负载 |
| 1-2       | 60秒    | 少量任务,中等频率 |
| 3-5       | 45秒    | 中等任务,略高频率 |
| 5+        | 30秒    | 大量任务,最高频率 |

#### 2. 确保定时器始终运行

移除"无任务时清除定时器"的逻辑:

```javascript
// ✅ 优化后代码
if (activeTab === 'tasks' && isAutoRefreshEnabled) {
  const runningTasksCount = runningTasksCountRef.current;
  const newInterval = adjustRefreshInterval(runningTasksCount);
  
  // ✅ 始终设置定时器，保证自动刷新功能正常工作
  console.log(`设置自动刷新：${newInterval/1000}秒间隔，${runningTasksCount}个运行中任务`);
  autoRefreshRef.current = setInterval(() => {
    loadTasks();
    setLastRefresh(Date.now());
  }, newInterval);
}
```

#### 3. 简化间隔调整逻辑

```javascript
// ✅ 优化后代码
useEffect(() => {
  const runningTasksCount = runningTasksCountRef.current;
  const newInterval = adjustRefreshInterval(runningTasksCount);
  setRefreshInterval(newInterval);

  // 如果已有定时器且间隔发生变化，重新设置定时器
  if (autoRefreshRef.current && isAutoRefreshEnabled && activeTab === 'tasks') {
    console.log(`运行任务数变化，调整刷新间隔为：${newInterval/1000}秒`);
    clearInterval(autoRefreshRef.current);
    autoRefreshRef.current = setInterval(() => {
      loadTasks();
      setLastRefresh(Date.now());
    }, newInterval);
  }
}, [tasks, activeTab, isAutoRefreshEnabled]);
```

### E2E 测试验证

**测试用例:** "SLURM Tasks 刷新频率优化验证"

```javascript
// 监听 API 请求
await page.route('**/api/slurm/tasks*', async (route) => {
  timestamps.push(Date.now());
  await route.continue();
});

// 等待 65 秒观察自动刷新
await page.waitForTimeout(65000);

// 验证刷新间隔 >= 30 秒
const lastInterval = timestamps[timestamps.length - 1] - timestamps[timestamps.length - 2];
expect(lastInterval).toBeGreaterThanOrEqual(30000);
```

**预期结果:**
- ✅ 页面加载成功
- ✅ 监听到多次 `/api/slurm/tasks` 请求
- ✅ 请求间隔 >= 30 秒
- ✅ 自动刷新功能正常工作

---

## 原始问题: 页面刷新过于频繁

## 根本原因分析

### 1. useEffect 循环依赖问题
原有代码中，自动刷新的 useEffect 依赖数组包含了 `tasks` 状态：

```javascript
useEffect(() => {
  // 自动刷新逻辑
}, [tasks, activeTab, isAutoRefreshEnabled])
```

这导致了以下问题链：
1. **定时器触发** → loadTasks() → 更新 tasks
2. **tasks 更新** → useEffect 重新执行 → 重建定时器
3. **新定时器触发** → loadTasks() → 更新 tasks
4. **循环往复** → 频繁刷新

### 2. 初始化 useEffect 依赖过多
原有的初始化 useEffect 依赖了多个状态：

```javascript
useEffect(() => {
  // 初始化和URL参数处理
}, [filters, pagination.current, pagination.pageSize, activeTab])
```

这导致每次 `filters`、`pagination` 或 `activeTab` 变化时都会重新执行，包括：
- 重新解析 URL 参数（不必要）
- 重新加载任务列表（应该单独控制）

### 3. 刷新间隔过短
原有刷新间隔配置：
- 无运行任务：不刷新
- 1-2个任务：20秒
- 3-5个任务：15秒
- 6个以上任务：10秒

这些间隔对于 SLURM 任务管理场景来说过于激进。

## 优化方案

### 1. 使用 useRef 避免循环依赖

**优化前**：
```javascript
const [tasks, setTasks] = useState([]);

useEffect(() => {
  if (!isAutoRefreshEnabled || activeTab !== 'tasks') return;
  
  const interval = adjustRefreshInterval(tasks.filter(...).length);
  autoRefreshRef.current = setInterval(loadTasks, interval);
  
  return () => clearInterval(autoRefreshRef.current);
}, [tasks, activeTab, isAutoRefreshEnabled]); // ❌ tasks 在依赖中导致循环
```

**优化后**：
```javascript
const runningTasksCountRef = useRef(0);  // ✅ 使用 ref 存储运行任务数

const loadTasks = async () => {
  // ... 加载逻辑 ...
  
  // 更新 ref 而不触发重渲染
  const runningCount = (data.tasks || []).filter(task => 
    task.status === 'running' || task.status === 'pending'
  ).length;
  runningTasksCountRef.current = runningCount;
};

// 自动刷新 - 不依赖 tasks
useEffect(() => {
  if (!isAutoRefreshEnabled || activeTab !== 'tasks') {
    if (autoRefreshRef.current) {
      clearInterval(autoRefreshRef.current);
      autoRefreshRef.current = null;
    }
    return;
  }

  loadTasks();
  const interval = adjustRefreshInterval(runningTasksCountRef.current);
  if (interval > 0) {
    autoRefreshRef.current = setInterval(loadTasks, interval);
  }

  return () => {
    if (autoRefreshRef.current) {
      clearInterval(autoRefreshRef.current);
      autoRefreshRef.current = null;
    }
  };
}, [activeTab, isAutoRefreshEnabled]); // ✅ 只依赖必要的状态

// 监听 tasks 变化，动态调整刷新间隔但不重建定时器
useEffect(() => {
  if (!isAutoRefreshEnabled || activeTab !== 'tasks') return;
  
  const newInterval = adjustRefreshInterval(runningTasksCountRef.current);
  
  if (newInterval === 0) {
    if (autoRefreshRef.current) {
      clearInterval(autoRefreshRef.current);
      autoRefreshRef.current = null;
    }
  } else if (newInterval !== refreshInterval) {
    setRefreshInterval(newInterval);
  }
}, [tasks, activeTab, isAutoRefreshEnabled, refreshInterval]);
```

### 2. 拆分初始化 useEffect

**优化前**：
```javascript
// 初始化加载和URL参数处理
useEffect(() => {
  const statusParam = searchParams.get('status');
  const taskIdParam = searchParams.get('taskId');
  
  if (statusParam) {
    setFilters(prev => ({ ...prev, status: statusParam }));
  }
  
  if (activeTab === 'tasks') {
    loadTasks().then(() => {
      // ... 打开任务详情
    });
  }
}, [filters, pagination.current, pagination.pageSize, activeTab]); // ❌ 依赖过多
```

**优化后**：
```javascript
// 1. 初始化 - 仅处理URL参数（组件挂载时执行一次）
useEffect(() => {
  const searchParams = new URLSearchParams(location.search);
  const statusParam = searchParams.get('status');
  const taskIdParam = searchParams.get('taskId');
  
  if (statusParam) {
    setFilters(prev => ({ ...prev, status: statusParam }));
  }
  
  if (taskIdParam) {
    setTaskDetailId(taskIdParam);
  }
}, []); // ✅ 空依赖数组，仅执行一次

// 2. 数据加载 - 监听必要状态变化
useEffect(() => {
  console.log('数据加载触发:', { activeTab, filters, pagination });
  
  if (activeTab === 'tasks') {
    loadTasks();
  } else if (activeTab === 'statistics') {
    loadStatistics();
  }
}, [filters.status, filters.jobName, pagination.current, pagination.pageSize, activeTab]); // ✅ 只监听会影响加载的字段

// 3. 自动打开任务详情
useEffect(() => {
  if (taskDetailId && tasks.length > 0 && activeTab === 'tasks') {
    const targetTask = tasks.find(task => task.id === taskDetailId);
    if (targetTask) {
      console.log('自动打开任务详情:', taskDetailId);
      handleViewTaskDetail(targetTask);
      setTaskDetailId(null); // 清除标记
    }
  }
}, [tasks, taskDetailId, activeTab]); // ✅ 专注于打开详情的逻辑
```

### 3. 增加刷新间隔

**优化前**：
```javascript
const adjustRefreshInterval = (runningTasksCount) => {
  if (runningTasksCount === 0) return 0;
  else if (runningTasksCount <= 2) return 20000;  // 20秒
  else if (runningTasksCount <= 5) return 15000;  // 15秒
  else return 10000;  // 10秒
};
```

**优化后**：
```javascript
const adjustRefreshInterval = (runningTasksCount) => {
  if (runningTasksCount === 0) {
    return 0; // 无运行任务时不刷新
  } else if (runningTasksCount <= 2) {
    return 60000; // 1-2个任务：60秒
  } else if (runningTasksCount <= 5) {
    return 45000; // 3-5个任务：45秒
  } else {
    return 30000; // 6个以上任务：30秒
  }
};
```

**理由**：
- SLURM 任务通常运行时间较长（分钟到小时级别）
- 任务状态变化频率低，不需要高频刷新
- 降低前端请求频率，减轻后端 API 压力
- 改善用户体验，避免页面频繁闪烁

## 优化效果

### 1. 解决循环依赖
- ✅ 使用 `useRef` 存储运行任务数，避免触发重渲染
- ✅ 自动刷新 useEffect 只依赖 `[activeTab, isAutoRefreshEnabled]`
- ✅ 另一个 useEffect 监听 tasks 变化，仅调整间隔不重建定时器

### 2. 减少不必要的加载
- ✅ URL 参数解析只在组件挂载时执行一次
- ✅ 数据加载 useEffect 只监听实际影响加载的字段
- ✅ 任务详情打开逻辑独立，不干扰其他逻辑

### 3. 降低刷新频率
- ✅ 刷新间隔从 10-20秒 增加到 30-60秒
- ✅ 根据运行任务数智能调整刷新频率
- ✅ 无运行任务时完全停止自动刷新

### 4. 改善用户体验
- ✅ 减少页面闪烁和重新渲染
- ✅ 降低 API 请求频率，提升系统性能
- ✅ 保持自动刷新功能，确保数据时效性

## 验证测试

### 测试场景 1：URL 参数访问
**测试步骤**：
1. 访问 `http://192.168.0.200:8080/slurm-tasks?taskId=xxx&status=running`
2. 观察页面刷新频率
3. 检查浏览器控制台日志

**预期结果**：
- 页面加载后自动应用 status 筛选
- 找到指定任务后自动打开详情
- 刷新间隔符合新配置（30-60秒）
- 控制台无循环刷新日志

### 测试场景 2：任务列表自动刷新
**测试步骤**：
1. 打开任务列表页面
2. 观察不同运行任务数下的刷新间隔
3. 检查自动刷新开关是否正常

**预期结果**：
- 0个运行任务：不刷新
- 1-2个运行任务：60秒刷新一次
- 3-5个运行任务：45秒刷新一次
- 6个以上运行任务：30秒刷新一次
- 关闭自动刷新后立即停止

### 测试场景 3：筛选和分页
**测试步骤**：
1. 修改任务状态筛选
2. 切换分页
3. 观察数据加载触发情况

**预期结果**：
- 筛选条件变化时触发加载
- 分页变化时触发加载
- 每次变化只触发一次加载
- 控制台日志清晰显示加载原因

## 代码变更文件

- **修改文件**：`src/frontend/src/pages/SlurmTasksPage.js`
- **主要变更**：
  1. 添加 `runningTasksCountRef` useRef
  2. 添加 `taskDetailId` 状态
  3. 优化 `adjustRefreshInterval` 函数（增加间隔）
  4. 拆分初始化 useEffect 为三个独立的 useEffect
  5. 重构自动刷新 useEffect，移除对 tasks 的依赖
  6. 添加监听 tasks 变化的 useEffect 动态调整间隔

## 后续改进建议

### 1. 可配置的刷新间隔
允许用户在界面上配置自动刷新间隔：

```javascript
const [customInterval, setCustomInterval] = useState({
  low: 60000,    // 少量任务
  medium: 45000, // 中等任务
  high: 30000    // 大量任务
});
```

### 2. WebSocket 实时推送
对于需要实时监控的场景，可以考虑使用 WebSocket 替代轮询：

```javascript
useEffect(() => {
  const ws = new WebSocket('ws://api/slurm/tasks/stream');
  ws.onmessage = (event) => {
    const updatedTask = JSON.parse(event.data);
    setTasks(prev => prev.map(t => t.id === updatedTask.id ? updatedTask : t));
  };
  return () => ws.close();
}, []);
```

### 3. 智能刷新策略
根据页面可见性和用户活动智能调整刷新：

```javascript
useEffect(() => {
  const handleVisibilityChange = () => {
    if (document.hidden) {
      // 页面不可见时降低刷新频率或停止刷新
      setIsAutoRefreshEnabled(false);
    } else {
      // 页面可见时恢复刷新
      setIsAutoRefreshEnabled(true);
      loadTasks(); // 立即刷新一次
    }
  };
  
  document.addEventListener('visibilitychange', handleVisibilityChange);
  return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
}, []);
```

### 4. 增量更新
只更新变化的任务，而不是完整替换整个列表：

```javascript
const loadTasksIncremental = async () => {
  const lastUpdate = tasks[0]?.updated_at || 0;
  const response = await slurmAPI.getTasks({ since: lastUpdate });
  const updatedTasks = response.data?.data?.tasks || [];
  
  if (updatedTasks.length > 0) {
    setTasks(prev => {
      const taskMap = new Map(prev.map(t => [t.id, t]));
      updatedTasks.forEach(t => taskMap.set(t.id, t));
      return Array.from(taskMap.values());
    });
  }
};
```

## 相关文档

- [FRONTEND_PAGE_FIXES.md](./FRONTEND_PAGE_FIXES.md) - 前端页面修复汇总
- [BUILD_AND_TEST_GUIDE.md](./BUILD_AND_TEST_GUIDE.md) - 构建和测试指南
- [E2E_VALIDATION_GUIDE.md](./E2E_VALIDATION_GUIDE.md) - E2E 测试验证指南

## 更新日期

2025-01-XX

## 作者

AI Infrastructure Team
