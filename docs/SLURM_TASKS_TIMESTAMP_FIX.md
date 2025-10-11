# SLURM Tasks 时间戳显示修复报告

## 问题描述

在 http://192.168.18.114:8080/slurm-tasks 页面中，任务的创建时间显示为 `1970-01-21 16:56:30`，这是一个典型的时间戳格式错误。

## 问题分析

### 根本原因

后端 API (`/api/slurm/tasks`) 返回的时间戳格式为 **Unix 时间戳（秒）**：

```json
{
  "created_at": 1760190457,
  "started_at": 1760190457,
  "completed_at": 1760190457
}
```

但前端代码直接使用 `dayjs(timestamp)` 处理，dayjs 默认将数字当作**毫秒**处理。

### 计算验证

- **1760190457 秒** = 2025-10-11 21:47:37 ✅ 正确
- **1760190457 毫秒** = 1760190.457 秒 = 约 20.4 天 = 1970-01-21 ❌ 错误

时间戳需要乘以 1000 才能正确转换为毫秒。

### 受影响的位置

1. **任务列表表格** - `created_at` 列
2. **任务详情弹窗** - 创建时间、开始时间、完成时间
3. **任务事件时间线** - 事件时间戳

## 修复方案

### 文件修改

**文件**: `src/frontend/src/pages/SlurmTasksPage.js`

### 1. 任务列表 - 开始时间列 (Line 179-188)

**修改前**:
```javascript
{
  title: '开始时间',
  dataIndex: 'created_at',
  key: 'created_at',
  render: (timestamp) => timestamp ? dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss') : '-',
},
```

**修改后**:
```javascript
{
  title: '开始时间',
  dataIndex: 'created_at',
  key: 'created_at',
  render: (timestamp) => {
    if (!timestamp) return '-';
    // 如果是 Unix 时间戳（秒），转换为毫秒
    const ts = timestamp < 10000000000 ? timestamp * 1000 : timestamp;
    return dayjs(ts).format('YYYY-MM-DD HH:mm:ss');
  },
},
```

### 2. 任务列表 - 持续时间列 (Line 189-203)

**修改前**:
```javascript
{
  title: '持续时间',
  dataIndex: 'duration',
  key: 'duration',
  render: (_, record) => {
    if (record.created_at) {
      const start = dayjs(record.created_at);
      const end = record.completed_at ? dayjs(record.completed_at) : dayjs();
      const duration = end.diff(start, 'second');
      return formatDuration(duration);
    }
    return '-';
  },
},
```

**修改后**:
```javascript
{
  title: '持续时间',
  dataIndex: 'duration',
  key: 'duration',
  render: (_, record) => {
    if (record.created_at) {
      // 如果是 Unix 时间戳（秒），转换为毫秒
      const startTs = record.created_at < 10000000000 ? record.created_at * 1000 : record.created_at;
      const endTs = record.completed_at 
        ? (record.completed_at < 10000000000 ? record.completed_at * 1000 : record.completed_at)
        : Date.now();
      const start = dayjs(startTs);
      const end = dayjs(endTs);
      const duration = end.diff(start, 'second');
      return formatDuration(duration);
    }
    return '-';
  },
},
```

### 3. 任务详情 - 时间显示 (Line 928-951)

**修改前**:
```javascript
<Descriptions.Item label="创建时间">
  {selectedTask.created_at ? dayjs(selectedTask.created_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
</Descriptions.Item>
<Descriptions.Item label="开始时间">
  {selectedTask.started_at ? dayjs(selectedTask.started_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
</Descriptions.Item>
<Descriptions.Item label="完成时间">
  {selectedTask.completed_at ? dayjs(selectedTask.completed_at).format('YYYY-MM-DD HH:mm:ss') : '-'}
</Descriptions.Item>
<Descriptions.Item label="执行时长">
  {selectedTask.created_at && selectedTask.completed_at ? 
    formatDuration(dayjs(selectedTask.completed_at).diff(dayjs(selectedTask.started_at || selectedTask.created_at), 'second')) :
    selectedTask.started_at ? formatDuration(dayjs().diff(dayjs(selectedTask.started_at), 'second')) : '-'
  }
</Descriptions.Item>
```

**修改后**:
```javascript
<Descriptions.Item label="创建时间">
  {selectedTask.created_at ? (() => {
    const ts = selectedTask.created_at < 10000000000 ? selectedTask.created_at * 1000 : selectedTask.created_at;
    return dayjs(ts).format('YYYY-MM-DD HH:mm:ss');
  })() : '-'}
</Descriptions.Item>
<Descriptions.Item label="开始时间">
  {selectedTask.started_at ? (() => {
    const ts = selectedTask.started_at < 10000000000 ? selectedTask.started_at * 1000 : selectedTask.started_at;
    return dayjs(ts).format('YYYY-MM-DD HH:mm:ss');
  })() : '-'}
</Descriptions.Item>
<Descriptions.Item label="完成时间">
  {selectedTask.completed_at ? (() => {
    const ts = selectedTask.completed_at < 10000000000 ? selectedTask.completed_at * 1000 : selectedTask.completed_at;
    return dayjs(ts).format('YYYY-MM-DD HH:mm:ss');
  })() : '-'}
</Descriptions.Item>
<Descriptions.Item label="执行时长">
  {selectedTask.created_at && selectedTask.completed_at ? (() => {
    const startTs = (selectedTask.started_at || selectedTask.created_at);
    const start = startTs < 10000000000 ? startTs * 1000 : startTs;
    const end = selectedTask.completed_at < 10000000000 ? selectedTask.completed_at * 1000 : selectedTask.completed_at;
    return formatDuration(dayjs(end).diff(dayjs(start), 'second'));
  })() : selectedTask.started_at ? (() => {
    const ts = selectedTask.started_at < 10000000000 ? selectedTask.started_at * 1000 : selectedTask.started_at;
    return formatDuration(dayjs().diff(dayjs(ts), 'second'));
  })() : '-'
  }
</Descriptions.Item>
```

### 4. 事件时间线 - 事件时间 (Line 1034)

**修改前**:
```javascript
<Text type="secondary" style={{ fontSize: '12px' }}>
  {event.created_at ? dayjs(event.created_at).format('YYYY-MM-DD HH:mm:ss') : ''}
</Text>
```

**修改后**:
```javascript
<Text type="secondary" style={{ fontSize: '12px' }}>
  {event.created_at ? (() => {
    const ts = event.created_at < 10000000000 ? event.created_at * 1000 : event.created_at;
    return dayjs(ts).format('YYYY-MM-DD HH:mm:ss');
  })() : ''}
</Text>
```

### 5. 任务进度事件 - ts字段 (Line 1060)

**修改前**:
```javascript
<Text type="secondary">
  {new Date(event.ts).toLocaleString()}
</Text>
```

**修改后**:
```javascript
<Text type="secondary">
  {(() => {
    const ts = event.ts < 10000000000 ? event.ts * 1000 : event.ts;
    return new Date(ts).toLocaleString();
  })()}
</Text>
```

## 技术细节

### 时间戳判断逻辑

```javascript
const ts = timestamp < 10000000000 ? timestamp * 1000 : timestamp;
```

**说明**:
- **10000000000** = 2286-11-20 17:46:40（作为秒）
- 如果时间戳小于这个值，说明是秒，需要乘以 1000
- 如果时间戳大于等于这个值，说明已经是毫秒，直接使用

这个判断可以兼容两种格式的时间戳。

### 为什么不修改后端

1. **后端返回秒级时间戳是标准做法** - Unix 时间戳通常以秒为单位
2. **兼容性考虑** - 其他客户端可能依赖当前格式
3. **前端更灵活** - 可以处理多种时间戳格式

### 参考实现

在 `SlurmTaskBar.jsx` 中已经使用了正确的方法：

```javascript
startedAt: t.started_at ? dayjs.unix(t.started_at) : null,
completedAt: t.completed_at && t.completed_at > 0 ? dayjs.unix(t.completed_at) : null,
```

`dayjs.unix()` 专门用于处理 Unix 时间戳（秒）。

## 构建和部署

### 1. 重新构建前端

```bash
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix
./build.sh build frontend --force
```

### 2. 重启前端容器

```bash
docker-compose restart frontend
```

### 3. 清除浏览器缓存

由于前端文件可能被浏览器缓存，需要硬刷新：
- Chrome/Edge: `Ctrl+Shift+R` (Windows/Linux) 或 `Cmd+Shift+R` (Mac)
- Firefox: `Ctrl+F5` (Windows/Linux) 或 `Cmd+Shift+R` (Mac)
- Safari: `Cmd+Option+R`

或者使用无痕/隐身模式访问。

## 验证方法

### 1. 检查 API 响应

```bash
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://192.168.18.114:8080/api/slurm/tasks?page=1&limit=10 | jq '.data.tasks[0]'
```

应该看到类似：
```json
{
  "created_at": 1760190457,  // Unix 时间戳（秒）
  "started_at": 1760190457,
  "completed_at": 1760190457
}
```

### 2. 访问页面验证

访问 http://192.168.18.114:8080/slurm-tasks

**正确显示示例**:
- 创建时间: `2025-10-11 21:47:37` ✅
- 完成时间: `2025-10-11 21:47:37` ✅

**错误显示示例**:
- 创建时间: `1970-01-21 16:56:30` ❌

### 3. 浏览器控制台验证

打开浏览器开发者工具，在 Console 中执行：

```javascript
// 测试转换逻辑
const timestamp = 1760190457;
const ts = timestamp < 10000000000 ? timestamp * 1000 : timestamp;
console.log(new Date(ts).toLocaleString());
// 应该输出: "2025/10/11 21:47:37" 或类似格式
```

## 相关文件

- **修改**: `src/frontend/src/pages/SlurmTasksPage.js` (5处修改)
- **参考**: `src/frontend/src/components/SlurmTaskBar.jsx` (正确实现)
- **API**: `/api/slurm/tasks` (返回Unix时间戳秒)

## 后续建议

1. **统一时间处理工具函数**
   
   创建一个工具函数统一处理时间戳转换：
   
   ```javascript
   // src/utils/datetime.js
   export const parseTimestamp = (timestamp) => {
     if (!timestamp) return null;
     // Unix时间戳秒转毫秒
     const ts = timestamp < 10000000000 ? timestamp * 1000 : timestamp;
     return dayjs(ts);
   };
   ```

2. **添加类型检查**
   
   在接收API数据时进行类型转换和验证

3. **后端文档化**
   
   在 API 文档中明确说明时间戳字段的格式（Unix秒）

## 测试结果

- ✅ 任务列表时间显示正常
- ✅ 任务详情时间显示正常  
- ✅ 事件时间线时间显示正常
- ✅ 持续时间计算正确

---

**状态**: ✅ 修复已完成
**修复时间**: 2025-10-11
**版本**: v0.3.7
