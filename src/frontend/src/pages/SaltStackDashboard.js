import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Space, Alert, Spin, Button, Layout, Typography, List, Progress, Descriptions, Badge, Tabs, Modal, Form, Input, Select, message, Skeleton, InputNumber, Switch, Divider, Tooltip, Popconfirm, Upload, Dropdown, Menu, Checkbox, Empty, Timeline } from 'antd';
import { 
  CheckCircleOutlined, 
  ExclamationCircleOutlined, 
  ClockCircleOutlined, 
  DesktopOutlined,
  SettingOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
  DatabaseOutlined,
  ApiOutlined,
  CloudUploadOutlined,
  PlusOutlined,
  DeleteOutlined,
  HistoryOutlined,
  SyncOutlined,
  QuestionCircleOutlined,
  SafetyCertificateOutlined,
  WifiOutlined,
  KeyOutlined,
  LockOutlined,
  UploadOutlined,
  DownloadOutlined,
  FileTextOutlined,
  DashboardOutlined,
  CopyOutlined,
  TeamOutlined,
  EditOutlined,
  LoadingOutlined,
  CloseCircleOutlined,
  CodeOutlined,
  EyeOutlined,
  SearchOutlined,
  FilterOutlined,
  ExpandOutlined,
  CompressOutlined
} from '@ant-design/icons';
import { saltStackAPI, aiAPI } from '../services/api';
import MinionsTable from '../components/MinionsTable';
import ResizableMetricsPanel from '../components/ResizableMetricsPanel';
import { useI18n } from '../hooks/useI18n';
import { useTheme } from '../hooks/useTheme';

const { Content } = Layout;
const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;

const { TextArea } = Input;
const { Option } = Select;

// 骨架屏组件
const StatisticSkeleton = ({ title, icon }) => (
  <Card>
    <div style={{ display: 'flex', alignItems: 'center' }}>
      {icon}
      <div style={{ marginLeft: 8, flex: 1 }}>
        <div style={{ fontSize: '14px', color: '#999', marginBottom: 4 }}>{title}</div>
        <Skeleton.Input style={{ width: 60, height: 24 }} active />
      </div>
    </div>
  </Card>
);

const SaltStackDashboard = () => {
  const { t } = useI18n();
  const { isDark } = useTheme();
  
  // 页面状态管理
  const [pageLoaded, setPageLoaded] = useState(false);
  const [activeTabKey, setActiveTabKey] = useState('overview'); // Tab 切换状态
  
  // 数据状态 - 分别管理loading状态
  const [status, setStatus] = useState(null);
  const [minions, setMinions] = useState([]);
  const [jobs, setJobs] = useState([]);
  
  // 分组状态
  const [minionGroups, setMinionGroups] = useState([]);
  const [selectedGroup, setSelectedGroup] = useState('');
  const [groupsLoading, setGroupsLoading] = useState(false);
  
  // 系统概览分组筛选
  const [overviewGroupFilter, setOverviewGroupFilter] = useState('all'); // 'all' 或分组名
  
  // 加载状态 - 分别管理每个数据块的加载状态
  const [statusLoading, setStatusLoading] = useState(false);
  const [minionsLoading, setMinionsLoading] = useState(false);
  const [jobsLoading, setJobsLoading] = useState(false);
  
  // IB 端口告警状态
  const [ibAlerts, setIbAlerts] = useState([]);
  
  // 全局状态
  const [demo, setDemo] = useState(false);
  const [error, setError] = useState(null);
  
  // 自定义执行弹窗
  const [execVisible, setExecVisible] = useState(false);
  const [execForm] = Form.useForm();
  const [execRunning, setExecRunning] = useState(false);
  const [execOpId, setExecOpId] = useState('');
  const [execEvents, setExecEvents] = useState([]);
  const sseRef = useRef(null);
  
  // 配置管理弹窗
  const [configVisible, setConfigVisible] = useState(false);
  const [configForm] = Form.useForm();
  const [configTemplates] = useState([
    { id: 'nginx', name: 'Nginx', desc: 'Install and configure Nginx web server' },
    { id: 'mysql', name: 'MySQL', desc: 'Install and configure MySQL database' },
    { id: 'docker', name: 'Docker', desc: 'Install and configure Docker container engine' },
    { id: 'firewall', name: 'Firewall', desc: 'Configure system firewall rules' },
    { id: 'user', name: 'User Management', desc: 'Add, delete and manage system users' },
  ]);

  // 批量安装 Salt Minion 弹窗
  const [batchInstallVisible, setBatchInstallVisible] = useState(false);
  const [batchInstallForm] = Form.useForm();
  const [batchInstallRunning, setBatchInstallRunning] = useState(false);
  const [batchInstallTaskId, setBatchInstallTaskId] = useState('');
  const [batchInstallEvents, setBatchInstallEvents] = useState([]);
  const [batchInstallHosts, setBatchInstallHosts] = useState([
    { key: Date.now(), host: '', port: 22, username: 'root', password: '', use_sudo: false, group: '', install_categraf: false }
  ]);
  const batchSseRef = useRef(null);
  
  // 动态并行度信息
  const [parallelInfo, setParallelInfo] = useState({ parallel: 0, percentage: 0, is_auto_calculate: true });
  
  // 文件导入相关状态
  const [importLoading, setImportLoading] = useState(false);

  // 粘贴导入弹窗状态
  const [pasteImportVisible, setPasteImportVisible] = useState(false);
  const [pasteContent, setPasteContent] = useState('');
  const [pasteFormat, setPasteFormat] = useState('csv');
  const [pasteImportLoading, setPasteImportLoading] = useState(false);

  // SSH 测试弹窗
  const [sshTestVisible, setSSHTestVisible] = useState(false);
  const [sshTestForm] = Form.useForm();
  const [sshTestRunning, setSSHTestRunning] = useState(false);
  const [sshTestResults, setSSHTestResults] = useState([]);
  const [sshTestHosts, setSSHTestHosts] = useState([
    { key: Date.now(), host: '', port: 22, username: 'root', password: '' }
  ]);

  // 删除/卸载 Minion 状态（使用 Set 追踪多个删除中的 minion）
  const [deletingMinionIds, setDeletingMinionIds] = useState(new Set());
  const [uninstallModalVisible, setUninstallModalVisible] = useState(false);
  const [uninstallForm] = Form.useForm();
  const [uninstallMinionId, setUninstallMinionId] = useState('');

  // 分组管理状态
  const [groupModalVisible, setGroupModalVisible] = useState(false);
  const [groupForm] = Form.useForm();
  const [editingGroup, setEditingGroup] = useState(null);

  // 快速创建分组弹窗（在批量安装中使用）
  const [quickGroupModalVisible, setQuickGroupModalVisible] = useState(false);
  const [quickGroupForm] = Form.useForm();
  const [quickGroupCreating, setQuickGroupCreating] = useState(false);
  const [quickGroupName, setQuickGroupName] = useState('');

  // 批量安装 Categraf 弹窗
  const [batchCategrafVisible, setBatchCategrafVisible] = useState(false);
  const [batchCategrafForm] = Form.useForm();
  const [batchCategrafRunning, setBatchCategrafRunning] = useState(false);
  const [batchCategrafTaskId, setBatchCategrafTaskId] = useState('');
  const [batchCategrafHosts, setBatchCategrafHosts] = useState([
    { key: Date.now(), host: '', port: 22, username: 'root', password: '', use_sudo: false }
  ]);
  const [batchCategrafEvents, setBatchCategrafEvents] = useState([]);
  const batchCategrafSseRef = useRef(null);

  // 部署节点指标采集弹窗
  const [deployMetricsVisible, setDeployMetricsVisible] = useState(false);
  const [deployMetricsForm] = Form.useForm();
  const [deployMetricsLoading, setDeployMetricsLoading] = useState(false);

  // 安装任务历史状态
  const [installTasks, setInstallTasks] = useState([]);
  const [installTasksLoading, setInstallTasksLoading] = useState(false);
  const [installTasksTotal, setInstallTasksTotal] = useState(0);
  const [installTasksPage, setInstallTasksPage] = useState({ current: 1, pageSize: 10 });
  const [expandedTaskId, setExpandedTaskId] = useState(null);

  // 作业详情弹窗状态
  const [jobDetailVisible, setJobDetailVisible] = useState(false);
  const [jobDetailLoading, setJobDetailLoading] = useState(false);
  const [jobDetail, setJobDetail] = useState(null);
  const [jobDetailSearchVisible, setJobDetailSearchVisible] = useState(false);
  const [jobDetailSearchText, setJobDetailSearchText] = useState('');
  const [jobDetailSearchRegex, setJobDetailSearchRegex] = useState(false);

  // 删除任务历史状态
  const [deleteTasks, setDeleteTasks] = useState([]);
  const [deleteTasksLoading, setDeleteTasksLoading] = useState(false);

  // 作业持久化配置状态
  const [jobConfig, setJobConfig] = useState(null);
  const [jobStats, setJobStats] = useState(null);
  const [jobConfigLoading, setJobConfigLoading] = useState(false);
  const [jobConfigForm] = Form.useForm();
  const [cleanupLoading, setCleanupLoading] = useState(false);
  const [deleteTasksTotal, setDeleteTasksTotal] = useState(0);
  const [expandedDeleteTaskId, setExpandedDeleteTaskId] = useState(null);
  const [deleteTaskLogs, setDeleteTaskLogs] = useState({});

  // 自动刷新状态
  const [autoRefreshMinions, setAutoRefreshMinions] = useState(false);
  const [autoRefreshTasks, setAutoRefreshTasks] = useState(false);
  const [autoRefreshOverview, setAutoRefreshOverview] = useState(false);
  const [autoRefreshInterval, setAutoRefreshInterval] = useState(10); // 默认10秒
  const autoRefreshMinionsRef = useRef(null);
  const autoRefreshTasksRef = useRef(null);
  const autoRefreshOverviewRef = useRef(null);

  // 批量执行命令状态
  const [batchExecResults, setBatchExecResults] = useState([]);
  const [batchExecLoading, setBatchExecLoading] = useState(false);
  const [selectedScriptTemplate, setSelectedScriptTemplate] = useState(null);
  const [batchExecForm] = Form.useForm();
  const [batchExecResultSearchVisible, setBatchExecResultSearchVisible] = useState(false);
  const [batchExecResultSearchText, setBatchExecResultSearchText] = useState('');
  const [batchExecResultSearchRegex, setBatchExecResultSearchRegex] = useState(false);
  const [batchExecTaskId, setBatchExecTaskId] = useState(null); // 当前执行任务ID
  const [batchExecJid, setBatchExecJid] = useState(null); // Salt JID
  const batchExecPollRef = useRef(null); // 轮询定时器引用
  const [jobSearchTaskId, setJobSearchTaskId] = useState(''); // 作业历史任务ID搜索
  const [jobSearchText, setJobSearchText] = useState(''); // 作业历史通用搜索
  
  // JID到TaskID的映射 - 使用localStorage持久化
  const TASK_ID_MAP_KEY = 'saltstack_jid_taskid_map';
  const jidToTaskIdMapRef = useRef((() => {
    try {
      const stored = localStorage.getItem(TASK_ID_MAP_KEY);
      if (stored) {
        const parsed = JSON.parse(stored);
        return new Map(Object.entries(parsed));
      }
    } catch (e) {
      console.warn('加载JID-TaskID映射失败', e);
    }
    return new Map();
  })());
  
  // 保存JID-TaskID映射到localStorage
  const saveJidTaskIdMap = useCallback((map) => {
    try {
      const obj = Object.fromEntries(map);
      localStorage.setItem(TASK_ID_MAP_KEY, JSON.stringify(obj));
    } catch (e) {
      console.warn('保存JID-TaskID映射失败', e);
    }
  }, []);
  
  // 添加JID-TaskID映射
  const addJidTaskIdMapping = useCallback((jid, taskId) => {
    if (!jid || !taskId) return;
    const map = jidToTaskIdMapRef.current;
    map.set(jid, taskId);
    // 保持映射表不超过100条记录
    if (map.size > 100) {
      const firstKey = map.keys().next().value;
      map.delete(firstKey);
    }
    saveJidTaskIdMap(map);
  }, [saveJidTaskIdMap]);
  
  // 获取TaskID
  const getTaskIdByJid = useCallback((jid) => {
    if (!jid) return null;
    return jidToTaskIdMapRef.current.get(jid);
  }, []);
  
  // 简化雪花算法 - 生成时序唯一ID (前端版本，降低计算量)
  const snowflakeIdRef = useRef({
    epoch: 1700000000000, // 自定义起始时间 2023-11-14
    sequence: 0,
    lastTimestamp: -1,
  });
  
  const generateTaskId = useCallback(() => {
    const sf = snowflakeIdRef.current;
    let timestamp = Date.now();
    
    if (timestamp === sf.lastTimestamp) {
      sf.sequence = (sf.sequence + 1) & 0xFFF; // 12位序列号，最大4095
      if (sf.sequence === 0) {
        // 序列号溢出，等待下一毫秒
        while (timestamp <= sf.lastTimestamp) {
          timestamp = Date.now();
        }
      }
    } else {
      sf.sequence = 0;
    }
    sf.lastTimestamp = timestamp;
    
    // 简化的雪花ID: 41位时间戳 + 12位序列号 + 10位随机数
    const timePart = timestamp - sf.epoch;
    const randomPart = Math.floor(Math.random() * 1024); // 10位随机数
    
    // 转换为字符串格式: YYYYMMDD-HHmmss-序列号-随机数
    const date = new Date(timestamp);
    const dateStr = `${date.getFullYear()}${String(date.getMonth() + 1).padStart(2, '0')}${String(date.getDate()).padStart(2, '0')}`;
    const timeStr = `${String(date.getHours()).padStart(2, '0')}${String(date.getMinutes()).padStart(2, '0')}${String(date.getSeconds()).padStart(2, '0')}`;
    
    return `EXEC-${dateStr}-${timeStr}-${String(sf.sequence).padStart(4, '0')}-${String(randomPart).padStart(4, '0')}`;
  }, []);
  
  // 脚本模板定义 - 用于批量检查和诊断
  const scriptTemplates = useMemo(() => [
    {
      id: 'check_driver',
      name: t('saltstack.checkDriverVersion', '检查驱动版本'),
      desc: t('saltstack.checkDriverVersionDesc', '检查所有节点的 NVIDIA 驱动版本、CUDA 版本'),
      language: 'bash',
      code: `#!/bin/bash
echo "=== NVIDIA Driver & CUDA Version Check ==="
echo "Hostname: $(hostname)"
echo "Date: $(date)"
echo ""

# Check NVIDIA driver
if command -v nvidia-smi &> /dev/null; then
    echo "--- nvidia-smi output ---"
    nvidia-smi --query-gpu=driver_version,cuda_version,name,memory.total --format=csv
else
    echo "nvidia-smi not found - No NVIDIA GPU or driver not installed"
fi

# Check NPU (Ascend)
if command -v npu-smi &> /dev/null; then
    echo ""
    echo "--- npu-smi output ---"
    npu-smi info | head -20
fi

echo ""
echo "=== Check Complete ==="`,
    },
    {
      id: 'check_xid',
      name: t('saltstack.checkNvidiaXID', '检查 NVIDIA XID 错误'),
      desc: t('saltstack.checkNvidiaXIDDesc', '检查操作系统中的 NVIDIA XID 错误日志'),
      language: 'bash',
      code: `#!/bin/bash
echo "=== NVIDIA XID Error Check ==="
echo "Hostname: $(hostname)"
echo "Date: $(date)"
echo ""

# Check dmesg for NVIDIA XID errors (last 7 days)
echo "--- Recent NVIDIA XID Errors (dmesg) ---"
dmesg -T 2>/dev/null | grep -i "NVRM: Xid" | tail -50 || echo "No XID errors found in dmesg"

echo ""
# Check journalctl for XID errors
echo "--- Recent NVIDIA XID Errors (journalctl, 7 days) ---"
journalctl --since "7 days ago" 2>/dev/null | grep -i "NVRM: Xid" | tail -50 || echo "No XID errors found in journalctl"

echo ""
# Check kernel log
echo "--- /var/log/kern.log or messages ---"
if [ -f /var/log/kern.log ]; then
    grep -i "NVRM: Xid" /var/log/kern.log 2>/dev/null | tail -20 || echo "No XID in kern.log"
elif [ -f /var/log/messages ]; then
    grep -i "NVRM: Xid" /var/log/messages 2>/dev/null | tail -20 || echo "No XID in messages"
else
    echo "Kernel log not found"
fi

echo ""
echo "=== XID Check Complete ==="`,
    },
    {
      id: 'check_gpu_drop',
      name: t('saltstack.checkGPUStatus', '检查 GPU 掉卡'),
      desc: t('saltstack.checkGPUStatusDesc', '检查 GPU 是否存在掉卡情况'),
      language: 'bash',
      code: `#!/bin/bash
echo "=== GPU Status Check ==="
echo "Hostname: $(hostname)"
echo "Date: $(date)"
echo ""

# Check expected vs actual GPU count
if command -v nvidia-smi &> /dev/null; then
    EXPECTED_GPUS=\${EXPECTED_GPUS:-8}  # Default expected 8 GPUs
    ACTUAL_GPUS=$(nvidia-smi -L 2>/dev/null | wc -l)
    
    echo "Expected GPUs: $EXPECTED_GPUS"
    echo "Detected GPUs: $ACTUAL_GPUS"
    
    if [ "$ACTUAL_GPUS" -lt "$EXPECTED_GPUS" ]; then
        echo "⚠️ WARNING: GPU count mismatch! Missing GPUs detected!"
        echo ""
        echo "--- Detected GPUs ---"
        nvidia-smi -L
    else
        echo "✅ All expected GPUs are present"
        echo ""
        nvidia-smi -L
    fi
    
    echo ""
    echo "--- GPU Health Status ---"
    nvidia-smi --query-gpu=index,name,pstate,pcie.link.gen.current,temperature.gpu,utilization.gpu --format=csv
else
    echo "nvidia-smi not found"
fi

# Check for NPU drops
if command -v npu-smi &> /dev/null; then
    echo ""
    echo "--- NPU Status ---"
    npu-smi info 2>/dev/null | grep -E "NPU|Health|Status" || echo "No NPU detected"
fi

echo ""
echo "=== GPU Status Check Complete ==="`,
    },
    {
      id: 'check_dmesg',
      name: t('saltstack.checkDmesgErrors', '检查 dmesg 错误'),
      desc: t('saltstack.checkDmesgErrorsDesc', '检查 dmesg 中的重要错误'),
      language: 'bash',
      code: `#!/bin/bash
echo "=== Critical dmesg Errors Check ==="
echo "Hostname: $(hostname)"
echo "Date: $(date)"
echo ""

# Check for OOM (Out of Memory)
echo "--- OOM Killer Events ---"
dmesg -T 2>/dev/null | grep -i "out of memory\\|oom\\|killed process" | tail -20 || echo "No OOM events"

echo ""
# Check for kernel panics
echo "--- Kernel Panic Events ---"
dmesg -T 2>/dev/null | grep -i "panic\\|Oops\\|BUG:" | tail -20 || echo "No panic events"

echo ""
# Check for hardware errors
echo "--- Hardware Errors (MCE/AER/PCIe) ---"
dmesg -T 2>/dev/null | grep -iE "hardware error\\|mce:|aer:|pcie.*error" | tail -20 || echo "No hardware errors"

echo ""
# Check for storage/disk errors
echo "--- Storage/Disk Errors ---"
dmesg -T 2>/dev/null | grep -iE "i/o error\\|disk error\\|ext4.*error\\|xfs.*error\\|scsi.*error" | tail -20 || echo "No disk errors"

echo ""
# Check for memory errors
echo "--- Memory Errors ---"
dmesg -T 2>/dev/null | grep -iE "memory.*error\\|edac\\|ecc" | tail -20 || echo "No memory errors"

echo ""
# Check for network errors
echo "--- Network Errors ---"
dmesg -T 2>/dev/null | grep -iE "link.*down\\|carrier.*off\\|timeout\\|drop" | tail -10 || echo "No significant network errors"

echo ""
echo "=== dmesg Check Complete ==="`,
    },
    {
      id: 'check_ib',
      name: t('saltstack.checkIBStatus', '检查 IB 网卡状态'),
      desc: t('saltstack.checkIBStatusDesc', '检查 InfiniBand 网卡状态和连接'),
      language: 'bash',
      code: `#!/bin/bash
echo "=== InfiniBand Status Check ==="
echo "Hostname: $(hostname)"
echo "Date: $(date)"
echo ""

# Check if ibstat is available
if command -v ibstat &> /dev/null; then
    echo "--- IB Device Summary ---"
    ibstat 2>/dev/null | grep -E "^CA|State|Rate|Port" || echo "No IB devices"
    
    echo ""
    echo "--- IB Port Status (detailed) ---"
    for ca in $(ibstat -l 2>/dev/null); do
        echo "CA: $ca"
        ibstat $ca 2>/dev/null | grep -A 10 "Port" | head -15
        echo ""
    done
    
    # Check for down ports
    echo "--- Down/Inactive Ports ---"
    DOWN_PORTS=$(ibstat 2>/dev/null | grep -B5 "State: Down\\|State: Initializing" | grep -E "^CA|Port|State")
    if [ -n "$DOWN_PORTS" ]; then
        echo "⚠️ WARNING: Found inactive IB ports:"
        echo "$DOWN_PORTS"
    else
        echo "✅ All IB ports are Active"
    fi
else
    echo "ibstat not found - InfiniBand tools not installed or no IB hardware"
fi

echo ""
# Check RDMA devices
if command -v rdma &> /dev/null; then
    echo "--- RDMA Devices ---"
    rdma link show 2>/dev/null || echo "No RDMA devices"
fi

echo ""
echo "=== IB Check Complete ==="`,
    },
    {
      id: 'check_system',
      name: t('saltstack.checkSystemHealth', '系统健康检查'),
      desc: t('saltstack.checkSystemHealthDesc', '综合检查系统状态'),
      language: 'bash',
      code: `#!/bin/bash
echo "=== System Health Check ==="
echo "Hostname: $(hostname)"
echo "Date: $(date)"
echo "Uptime: $(uptime)"
echo ""

# CPU
echo "--- CPU Info ---"
echo "CPU Cores: $(nproc)"
echo "Load Average: $(cat /proc/loadavg)"
top -bn1 | head -5

echo ""
# Memory
echo "--- Memory Info ---"
free -h
echo ""

# Disk
echo "--- Disk Usage ---"
df -h | grep -v "tmpfs\\|loop\\|udev"
echo ""

# Check high usage
echo "--- High Resource Usage Warnings ---"
# CPU load check
LOAD=$(cat /proc/loadavg | awk '{print $1}')
CORES=$(nproc)
if (( $(echo "$LOAD > $CORES" | bc -l) )); then
    echo "⚠️ High CPU Load: $LOAD (Cores: $CORES)"
fi

# Memory check
MEM_USED=$(free | awk '/Mem/{printf("%.0f", $3/$2*100)}')
if [ "$MEM_USED" -gt 90 ]; then
    echo "⚠️ High Memory Usage: $MEM_USED%"
fi

# Disk check
df -h | awk '$5 ~ /[0-9]+%/ {gsub(/%/,"",$5); if($5 > 90) print "⚠️ High Disk Usage: " $6 " at " $5 "%"}'

echo ""
# Process count
echo "--- Process Count ---"
echo "Total: $(ps aux | wc -l)"
echo "Zombie: $(ps aux | awk '$8 ~ /Z/' | wc -l)"

echo ""
echo "=== System Health Check Complete ==="`,
    },
    {
      id: 'daily_inspection',
      name: t('saltstack.dailyInspection', '日常巡检（综合）'),
      desc: t('saltstack.dailyInspectionDesc', 'GPU 集群和物理机日常巡检脚本'),
      language: 'bash',
      code: `#!/bin/bash
# =============================================================================
# 日常巡检脚本 - GPU 集群和物理机
# =============================================================================
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║               日 常 巡 检 报 告                                ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo "主机名: $(hostname)"
echo "IP 地址: $(hostname -I 2>/dev/null | awk '{print $1}')"
echo "巡检时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "运行时长: $(uptime -p 2>/dev/null || uptime)"
echo ""

# ========== 1. 系统基础信息 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【1. 系统基础信息】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "操作系统: $(cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d'"' -f2)"
echo "内核版本: $(uname -r)"
echo "CPU 核心: $(nproc) 核"
echo "负载均值: $(cat /proc/loadavg | awk '{print $1, $2, $3}')"
echo ""

# ========== 2. 内存状态 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【2. 内存状态】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
free -h
MEM_USED_PCT=$(free | awk '/Mem/{printf "%.1f", $3/$2*100}')
if (( $(echo "$MEM_USED_PCT > 90" | bc -l) )); then
    echo "⚠️ 警告: 内存使用率 $MEM_USED_PCT% 超过 90%"
else
    echo "✅ 内存使用率: $MEM_USED_PCT%"
fi
echo ""

# ========== 3. 磁盘状态 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【3. 磁盘状态】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
df -h | grep -v "tmpfs\\|loop\\|udev\\|overlay"
echo ""
echo "--- 磁盘使用率告警检查 ---"
DISK_WARN=0
while read -r line; do
    usage=$(echo "$line" | awk '{print $5}' | tr -d '%')
    mount=$(echo "$line" | awk '{print $6}')
    if [ "$usage" -gt 85 ]; then
        echo "⚠️ 磁盘 $mount 使用率 $usage% (警告阈值: 85%)"
        DISK_WARN=1
    fi
done < <(df -h | grep -v "tmpfs\\|loop\\|udev\\|overlay\\|Filesystem")
[ "$DISK_WARN" -eq 0 ] && echo "✅ 所有磁盘使用率正常"
echo ""

# ========== 4. GPU 状态 (NVIDIA) ==========
if command -v nvidia-smi &> /dev/null; then
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "【4. GPU 状态 (NVIDIA)】"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    GPU_COUNT=$(nvidia-smi -L 2>/dev/null | wc -l)
    echo "检测到 GPU 数量: $GPU_COUNT"
    echo ""
    nvidia-smi --query-gpu=index,name,driver_version,temperature.gpu,utilization.gpu,memory.used,memory.total --format=csv
    echo ""
    # 检查 GPU 温度
    echo "--- GPU 温度检查 ---"
    nvidia-smi --query-gpu=index,temperature.gpu --format=csv,noheader | while read line; do
        idx=$(echo "$line" | cut -d',' -f1)
        temp=$(echo "$line" | cut -d',' -f2 | tr -d ' ')
        if [ "$temp" -gt 85 ]; then
            echo "⚠️ GPU $idx 温度 $temp°C 过高！"
        elif [ "$temp" -gt 75 ]; then
            echo "⚠️ GPU $idx 温度 $temp°C 偏高"
        else
            echo "✅ GPU $idx 温度 $temp°C 正常"
        fi
    done
    echo ""
fi

# ========== 5. NPU 状态 (华为昇腾) ==========
if command -v npu-smi &> /dev/null; then
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "【5. NPU 状态 (华为昇腾)】"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    npu-smi info 2>/dev/null | head -30
    echo ""
fi

# ========== 6. 网络状态 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【6. 网络状态】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
# 显示物理网卡状态
ip -br link show | grep -v "lo\\|docker\\|veth\\|br-"
echo ""

# ========== 7. InfiniBand 状态 ==========
if command -v ibstat &> /dev/null; then
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "【7. InfiniBand 状态】"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    IB_DOWN=$(ibstat 2>/dev/null | grep -c "State: Down")
    IB_ACTIVE=$(ibstat 2>/dev/null | grep -c "State: Active")
    echo "IB 端口状态: Active=$IB_ACTIVE, Down=$IB_DOWN"
    if [ "$IB_DOWN" -gt 0 ]; then
        echo "⚠️ 发现 $IB_DOWN 个 IB 端口处于 Down 状态"
        ibstat 2>/dev/null | grep -B5 "State: Down"
    else
        echo "✅ 所有 IB 端口正常"
    fi
    echo ""
fi

# ========== 8. 关键服务状态 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【8. 关键服务状态】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
for svc in docker containerd kubelet slurmd slurmctld salt-minion; do
    if systemctl is-active "$svc" &>/dev/null; then
        echo "✅ $svc: active"
    elif systemctl list-unit-files | grep -q "^$svc"; then
        echo "❌ $svc: inactive"
    fi
done
echo ""

# ========== 9. 最近错误日志 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【9. 最近错误日志 (最近1小时)】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
journalctl --since "1 hour ago" -p err --no-pager 2>/dev/null | tail -10 || echo "无法读取 journalctl"
echo ""

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                    巡 检 完 成                                 ║"
echo "╚════════════════════════════════════════════════════════════════╝"`,
    },
    {
      id: 'gpu_health_check',
      name: t('saltstack.gpuHealthCheck', 'GPU 健康检查'),
      desc: t('saltstack.gpuHealthCheckDesc', '深度检查 GPU 健康状态、ECC 错误、温度'),
      language: 'bash',
      code: `#!/bin/bash
# =============================================================================
# GPU 健康深度检查脚本
# =============================================================================
echo "=== GPU 健康深度检查 ==="
echo "主机: $(hostname)"
echo "时间: $(date)"
echo ""

if ! command -v nvidia-smi &> /dev/null; then
    echo "❌ 未检测到 nvidia-smi，可能未安装 NVIDIA 驱动"
    exit 1
fi

# 1. 驱动和 CUDA 版本
echo "【1. 驱动信息】"
nvidia-smi --query-gpu=driver_version,cuda_version --format=csv
echo ""

# 2. GPU 列表
echo "【2. GPU 列表】"
nvidia-smi -L
GPU_COUNT=$(nvidia-smi -L | wc -l)
echo "总计: $GPU_COUNT 块 GPU"
echo ""

# 3. GPU 温度和功耗
echo "【3. GPU 温度/功耗/风扇】"
nvidia-smi --query-gpu=index,name,temperature.gpu,power.draw,fan.speed --format=csv
echo ""

# 4. GPU 利用率和显存
echo "【4. GPU 利用率/显存】"
nvidia-smi --query-gpu=index,utilization.gpu,utilization.memory,memory.used,memory.total --format=csv
echo ""

# 5. ECC 错误检查
echo "【5. ECC 错误检查】"
nvidia-smi --query-gpu=index,ecc.errors.corrected.volatile.total,ecc.errors.uncorrected.volatile.total --format=csv 2>/dev/null || echo "ECC 信息不可用"
echo ""

# 检查 ECC 错误
ECC_ERRORS=$(nvidia-smi --query-gpu=ecc.errors.uncorrected.volatile.total --format=csv,noheader 2>/dev/null | awk '{sum+=$1}END{print sum}')
if [ -n "$ECC_ERRORS" ] && [ "$ECC_ERRORS" -gt 0 ]; then
    echo "⚠️ 警告: 检测到 $ECC_ERRORS 个不可纠正的 ECC 错误！"
fi

# 6. PCIe 带宽
echo "【6. PCIe 信息】"
nvidia-smi --query-gpu=index,pcie.link.gen.current,pcie.link.gen.max,pcie.link.width.current --format=csv
echo ""

# 7. 持久模式和计算模式
echo "【7. GPU 模式设置】"
nvidia-smi --query-gpu=index,persistence_mode,compute_mode --format=csv
echo ""

# 8. 运行中的进程
echo "【8. GPU 上运行的进程】"
nvidia-smi --query-compute-apps=pid,name,gpu_bus_id,used_memory --format=csv 2>/dev/null || echo "无运行中的 GPU 进程"
echo ""

# 9. XID 错误检查
echo "【9. XID 错误检查 (最近24小时)】"
XID_COUNT=$(dmesg -T 2>/dev/null | grep -c "NVRM: Xid" || echo "0")
if [ "$XID_COUNT" -gt 0 ]; then
    echo "⚠️ 发现 $XID_COUNT 条 XID 错误日志:"
    dmesg -T 2>/dev/null | grep "NVRM: Xid" | tail -10
else
    echo "✅ 无 XID 错误"
fi
echo ""

# 10. 健康评估
echo "【10. 健康评估总结】"
ISSUES=0
# 温度检查
HIGH_TEMP=$(nvidia-smi --query-gpu=temperature.gpu --format=csv,noheader | awk '$1>85{count++}END{print count+0}')
if [ "$HIGH_TEMP" -gt 0 ]; then
    echo "⚠️ $HIGH_TEMP 块 GPU 温度过高 (>85°C)"
    ISSUES=$((ISSUES+1))
fi
# ECC 检查
if [ -n "$ECC_ERRORS" ] && [ "$ECC_ERRORS" -gt 0 ]; then
    echo "⚠️ 存在不可纠正的 ECC 错误"
    ISSUES=$((ISSUES+1))
fi
# XID 检查
if [ "$XID_COUNT" -gt 0 ]; then
    echo "⚠️ 存在 XID 错误日志"
    ISSUES=$((ISSUES+1))
fi

if [ "$ISSUES" -eq 0 ]; then
    echo "✅ GPU 健康状态良好"
else
    echo "❌ 发现 $ISSUES 类问题，请检查"
fi

echo ""
echo "=== GPU 健康检查完成 ==="`,
    },
    {
      id: 'network_diagnosis',
      name: t('saltstack.networkDiagnosis', '网络诊断'),
      desc: t('saltstack.networkDiagnosisDesc', '检查网络连接、丢包、延迟'),
      language: 'bash',
      code: `#!/bin/bash
# =============================================================================
# 网络诊断脚本
# =============================================================================
echo "=== 网络诊断 ==="
echo "主机: $(hostname)"
echo "时间: $(date)"
echo ""

# 1. 网卡状态
echo "【1. 网卡状态】"
ip -br link show
echo ""

# 2. IP 地址
echo "【2. IP 地址配置】"
ip -br addr show | grep -v "lo"
echo ""

# 3. 路由表
echo "【3. 路由表】"
ip route | head -20
echo ""

# 4. DNS 配置
echo "【4. DNS 配置】"
cat /etc/resolv.conf | grep -v "^#" | head -5
echo ""

# 5. 网络连接统计
echo "【5. 网络连接统计】"
echo "ESTABLISHED: $(ss -t state established | wc -l)"
echo "TIME_WAIT: $(ss -t state time-wait | wc -l)"
echo "CLOSE_WAIT: $(ss -t state close-wait | wc -l)"
echo ""

# 6. 监听端口
echo "【6. 监听端口 (前20个)】"
ss -tlnp | head -20
echo ""

# 7. 网卡错误统计
echo "【7. 网卡错误统计】"
for iface in $(ip -br link show | awk '{print $1}' | grep -v "lo\\|docker\\|veth"); do
    echo "--- $iface ---"
    ethtool -S $iface 2>/dev/null | grep -E "error|drop|collision|crc" | grep -v ": 0$" | head -10 || echo "无错误"
done
echo ""

# 8. InfiniBand 诊断
if command -v ibstat &> /dev/null; then
    echo "【8. InfiniBand 诊断】"
    echo "IB 设备列表:"
    ibstat -l 2>/dev/null
    echo ""
    echo "IB 端口状态:"
    ibstat 2>/dev/null | grep -E "^CA|State:|Rate:|Physical state:"
    echo ""
    
    # IB 错误计数
    if command -v perfquery &> /dev/null; then
        echo "IB 端口错误统计:"
        for port in $(ibstat -l 2>/dev/null); do
            perfquery -x 2>/dev/null | grep -E "Error|Drop" | grep -v ": 0$" || echo "无错误"
        done
    fi
fi

# 9. 连通性测试
echo "【9. 连通性测试】"
GATEWAY=$(ip route | grep default | awk '{print $3}' | head -1)
if [ -n "$GATEWAY" ]; then
    echo "测试网关 $GATEWAY ..."
    ping -c 3 -W 2 $GATEWAY 2>/dev/null && echo "✅ 网关可达" || echo "❌ 网关不可达"
fi

# DNS 测试
echo ""
echo "测试 DNS 解析..."
if nslookup baidu.com &>/dev/null || host baidu.com &>/dev/null; then
    echo "✅ DNS 解析正常"
else
    echo "❌ DNS 解析失败"
fi

echo ""
echo "=== 网络诊断完成 ==="`,
    },
    {
      id: 'storage_check',
      name: t('saltstack.storageCheck', '存储检查'),
      desc: t('saltstack.storageCheckDesc', '检查磁盘、RAID、NFS、分布式存储'),
      language: 'bash',
      code: `#!/bin/bash
# =============================================================================
# 存储检查脚本
# =============================================================================
echo "=== 存储检查 ==="
echo "主机: $(hostname)"
echo "时间: $(date)"
echo ""

# 1. 磁盘分区
echo "【1. 磁盘分区】"
lsblk -o NAME,SIZE,TYPE,MOUNTPOINT,FSTYPE
echo ""

# 2. 磁盘使用率
echo "【2. 磁盘使用率】"
df -h | grep -v "tmpfs\\|loop\\|udev\\|overlay"
echo ""

# 3. inode 使用率
echo "【3. inode 使用率】"
df -i | grep -v "tmpfs\\|loop\\|udev\\|overlay"
echo ""

# 4. 磁盘 IO 统计
echo "【4. 磁盘 IO 统计】"
if command -v iostat &> /dev/null; then
    iostat -x 1 2 | tail -20
else
    echo "iostat 未安装，使用 vmstat:"
    vmstat 1 3
fi
echo ""

# 5. RAID 状态
echo "【5. RAID 状态检查】"
if [ -f /proc/mdstat ]; then
    echo "--- Software RAID (mdadm) ---"
    cat /proc/mdstat
fi

if command -v MegaCli64 &> /dev/null; then
    echo "--- MegaRAID 状态 ---"
    MegaCli64 -LDInfo -Lall -aALL 2>/dev/null | grep -E "Name|State|Size" || echo "无 MegaRAID"
elif command -v storcli64 &> /dev/null; then
    echo "--- StorCLI RAID 状态 ---"
    storcli64 /c0 /vall show 2>/dev/null || echo "无 StorCLI"
else
    echo "未检测到硬件 RAID 控制器或工具"
fi
echo ""

# 6. NVMe 健康
echo "【6. NVMe 健康状态】"
if command -v nvme &> /dev/null; then
    for dev in $(nvme list 2>/dev/null | awk 'NR>2{print $1}'); do
        echo "--- $dev ---"
        nvme smart-log $dev 2>/dev/null | grep -E "temperature|percentage_used|available_spare" || echo "无法读取"
    done
else
    echo "nvme-cli 未安装"
fi
echo ""

# 7. NFS 挂载检查
echo "【7. NFS 挂载】"
if mount | grep -q nfs; then
    echo "检测到 NFS 挂载:"
    mount | grep nfs
    echo ""
    echo "NFS 挂载点状态:"
    for nfs_mount in $(mount | grep nfs | awk '{print $3}'); do
        if timeout 5 ls "$nfs_mount" &>/dev/null; then
            echo "✅ $nfs_mount 可访问"
        else
            echo "❌ $nfs_mount 不可访问或超时"
        fi
    done
else
    echo "无 NFS 挂载"
fi
echo ""

# 8. 分布式存储检查
echo "【8. 分布式存储】"
# Ceph
if command -v ceph &> /dev/null; then
    echo "--- Ceph 状态 ---"
    ceph health 2>/dev/null || echo "无法连接 Ceph"
fi
# GlusterFS
if command -v gluster &> /dev/null; then
    echo "--- GlusterFS 状态 ---"
    gluster peer status 2>/dev/null || echo "无法连接 GlusterFS"
fi
# Lustre
if lsmod | grep -q lustre; then
    echo "--- Lustre 挂载 ---"
    mount | grep lustre || echo "无 Lustre 挂载"
fi
echo ""

# 9. 磁盘健康 (SMART)
echo "【9. 磁盘 SMART 健康】"
if command -v smartctl &> /dev/null; then
    for disk in $(lsblk -d -o NAME | grep -E "^sd|^nvme" | head -5); do
        echo "--- /dev/$disk ---"
        smartctl -H /dev/$disk 2>/dev/null | grep -E "result|PASSED|FAILED" || echo "无法读取"
    done
else
    echo "smartmontools 未安装"
fi

echo ""
echo "=== 存储检查完成 ==="`,
    },
    {
      id: 'process_check',
      name: t('saltstack.processCheck', '进程检查'),
      desc: t('saltstack.processCheckDesc', '检查高 CPU/内存进程、僵尸进程'),
      language: 'bash',
      code: `#!/bin/bash
# =============================================================================
# 进程检查脚本
# =============================================================================
echo "=== 进程检查 ==="
echo "主机: $(hostname)"
echo "时间: $(date)"
echo ""

# 1. 系统负载
echo "【1. 系统负载】"
uptime
echo ""

# 2. CPU 使用 Top 10
echo "【2. CPU 使用率 Top 10 进程】"
ps aux --sort=-%cpu | head -11 | awk '{printf "%-10s %-8s %-6s %-6s %s\\n", $1, $2, $3"%", $4"%", $11}'
echo ""

# 3. 内存使用 Top 10
echo "【3. 内存使用 Top 10 进程】"
ps aux --sort=-%mem | head -11 | awk '{printf "%-10s %-8s %-6s %-6s %s\\n", $1, $2, $3"%", $4"%", $11}'
echo ""

# 4. 僵尸进程
echo "【4. 僵尸进程检查】"
ZOMBIE=$(ps aux | awk '$8=="Z"' | wc -l)
if [ "$ZOMBIE" -gt 0 ]; then
    echo "⚠️ 发现 $ZOMBIE 个僵尸进程:"
    ps aux | awk '$8=="Z"' | head -10
else
    echo "✅ 无僵尸进程"
fi
echo ""

# 5. 高 CPU 进程告警
echo "【5. 高 CPU 进程告警 (>80%)】"
HIGH_CPU=$(ps aux | awk 'NR>1 && $3>80 {printf "PID: %-8s CPU: %-6s CMD: %s\\n", $2, $3"%", $11}')
if [ -n "$HIGH_CPU" ]; then
    echo "⚠️ 发现高 CPU 进程:"
    echo "$HIGH_CPU"
else
    echo "✅ 无高 CPU 进程"
fi
echo ""

# 6. 高内存进程告警
echo "【6. 高内存进程告警 (>50%)】"
HIGH_MEM=$(ps aux | awk 'NR>1 && $4>50 {printf "PID: %-8s MEM: %-6s CMD: %s\\n", $2, $4"%", $11}')
if [ -n "$HIGH_MEM" ]; then
    echo "⚠️ 发现高内存进程:"
    echo "$HIGH_MEM"
else
    echo "✅ 无高内存进程"
fi
echo ""

# 7. 进程数统计
echo "【7. 进程统计】"
echo "总进程数: $(ps aux | wc -l)"
echo "运行中: $(ps aux | awk '$8=="R"' | wc -l)"
echo "睡眠中: $(ps aux | awk '$8=="S"' | wc -l)"
echo "僵尸: $ZOMBIE"
echo ""

# 8. 用户进程统计
echo "【8. 用户进程统计 (Top 5)】"
ps aux | awk 'NR>1{a[$1]++}END{for(i in a)print a[i],i}' | sort -rn | head -5
echo ""

# 9. 长时间运行进程
echo "【9. 长时间运行进程 (>7天)】"
ps -eo pid,etime,cmd --sort=-etime | awk 'NR>1{
    time=$2;
    if(index(time,"-")>0){
        split(time,a,"-");
        days=a[1];
        if(days>=7) print $0
    }
}' | head -10 || echo "无超过7天的进程"

echo ""
echo "=== 进程检查完成 ==="`,
    },
    {
      id: 'security_check',
      name: t('saltstack.securityCheck', '安全检查'),
      desc: t('saltstack.securityCheckDesc', '检查登录日志、异常用户、开放端口'),
      language: 'bash',
      code: `#!/bin/bash
# =============================================================================
# 安全检查脚本
# =============================================================================
echo "=== 安全检查 ==="
echo "主机: $(hostname)"
echo "时间: $(date)"
echo ""

# 1. 最近登录
echo "【1. 最近登录 (last 10)】"
last -10 2>/dev/null || echo "无法读取登录日志"
echo ""

# 2. 登录失败
echo "【2. 最近登录失败 (last 10)】"
lastb -10 2>/dev/null || grep -i "failed\\|failure" /var/log/auth.log 2>/dev/null | tail -10 || echo "无法读取"
echo ""

# 3. 当前登录用户
echo "【3. 当前登录用户】"
who
echo ""

# 4. sudo 权限用户
echo "【4. sudo 权限用户】"
grep -E "^%sudo|^%wheel|^%admin" /etc/sudoers 2>/dev/null
grep -E "ALL=\\(ALL\\)" /etc/sudoers.d/* 2>/dev/null || echo "无额外 sudoers 配置"
echo ""

# 5. 空密码账户
echo "【5. 空密码账户检查】"
EMPTY_PWD=$(awk -F: '($2==""){print $1}' /etc/shadow 2>/dev/null)
if [ -n "$EMPTY_PWD" ]; then
    echo "⚠️ 发现空密码账户:"
    echo "$EMPTY_PWD"
else
    echo "✅ 无空密码账户"
fi
echo ""

# 6. UID=0 用户
echo "【6. UID=0 用户 (root 权限)】"
awk -F: '$3==0{print $1}' /etc/passwd
echo ""

# 7. 新增用户 (7天内)
echo "【7. 最近新增用户 (7天内)】"
find /home -maxdepth 1 -type d -mtime -7 2>/dev/null | grep -v "^/home$" || echo "无新增用户"
echo ""

# 8. 监听端口
echo "【8. 对外监听端口】"
ss -tlnp | grep -v "127.0.0.1\\|::1" | head -20
echo ""

# 9. SSH 配置检查
echo "【9. SSH 安全配置】"
if [ -f /etc/ssh/sshd_config ]; then
    echo "PermitRootLogin: $(grep -i "^PermitRootLogin" /etc/ssh/sshd_config 2>/dev/null || echo "默认")"
    echo "PasswordAuthentication: $(grep -i "^PasswordAuthentication" /etc/ssh/sshd_config 2>/dev/null || echo "默认")"
    echo "PermitEmptyPasswords: $(grep -i "^PermitEmptyPasswords" /etc/ssh/sshd_config 2>/dev/null || echo "默认")"
fi
echo ""

# 10. 防火墙状态
echo "【10. 防火墙状态】"
if command -v ufw &> /dev/null; then
    ufw status 2>/dev/null || echo "ufw 未运行"
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --state 2>/dev/null || echo "firewalld 未运行"
elif command -v iptables &> /dev/null; then
    echo "iptables 规则数: $(iptables -L -n 2>/dev/null | wc -l)"
fi
echo ""

# 11. SUID 文件检查
echo "【11. 异常 SUID 文件检查】"
find /usr -perm -4000 -type f 2>/dev/null | head -20 || echo "无法检查"

echo ""
echo "=== 安全检查完成 ==="`,
    },
    {
      id: 'collect_sysinfo',
      name: t('saltstack.collectSysinfo', '采集系统信息'),
      desc: t('saltstack.collectSysinfoDesc', '采集完整系统配置信息用于资产管理'),
      language: 'bash',
      code: `#!/bin/bash
# =============================================================================
# 系统信息采集脚本 - 用于资产管理
# =============================================================================
echo "{"
echo "  \\"hostname\\": \\"$(hostname)\\","
echo "  \\"collected_at\\": \\"$(date -Iseconds)\\","

# 操作系统信息
echo "  \\"os\\": {"
echo "    \\"name\\": \\"$(cat /etc/os-release 2>/dev/null | grep ^ID= | cut -d= -f2 | tr -d '\"')\\","
echo "    \\"version\\": \\"$(cat /etc/os-release 2>/dev/null | grep VERSION_ID | cut -d= -f2 | tr -d '\"')\\","
echo "    \\"kernel\\": \\"$(uname -r)\\","
echo "    \\"arch\\": \\"$(uname -m)\\""
echo "  },"

# CPU 信息
CPU_MODEL=$(cat /proc/cpuinfo | grep "model name" | head -1 | cut -d: -f2 | xargs)
CPU_CORES=$(nproc)
CPU_SOCKETS=$(cat /proc/cpuinfo | grep "physical id" | sort -u | wc -l)
[ "$CPU_SOCKETS" -eq 0 ] && CPU_SOCKETS=1
echo "  \\"cpu\\": {"
echo "    \\"model\\": \\"$CPU_MODEL\\","
echo "    \\"cores\\": $CPU_CORES,"
echo "    \\"sockets\\": $CPU_SOCKETS"
echo "  },"

# 内存信息
MEM_TOTAL=$(free -b | awk '/Mem:/{print $2}')
MEM_TOTAL_GB=$(echo "scale=2; $MEM_TOTAL/1024/1024/1024" | bc)
echo "  \\"memory\\": {"
echo "    \\"total_bytes\\": $MEM_TOTAL,"
echo "    \\"total_gb\\": $MEM_TOTAL_GB"
echo "  },"

# GPU 信息
if command -v nvidia-smi &> /dev/null; then
    GPU_COUNT=$(nvidia-smi -L 2>/dev/null | wc -l)
    GPU_MODEL=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
    GPU_DRIVER=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>/dev/null | head -1)
    GPU_MEM=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1)
    echo "  \\"gpu\\": {"
    echo "    \\"vendor\\": \\"nvidia\\","
    echo "    \\"count\\": $GPU_COUNT,"
    echo "    \\"model\\": \\"$GPU_MODEL\\","
    echo "    \\"driver_version\\": \\"$GPU_DRIVER\\","
    echo "    \\"memory\\": \\"$GPU_MEM\\""
    echo "  },"
else
    echo "  \\"gpu\\": null,"
fi

# NPU 信息
if command -v npu-smi &> /dev/null; then
    NPU_COUNT=$(npu-smi info -l 2>/dev/null | grep -c "NPU ID")
    echo "  \\"npu\\": {"
    echo "    \\"vendor\\": \\"huawei\\","
    echo "    \\"count\\": $NPU_COUNT"
    echo "  },"
else
    echo "  \\"npu\\": null,"
fi

# 磁盘信息
echo "  \\"disks\\": ["
FIRST_DISK=1
for disk in $(lsblk -d -o NAME,TYPE | awk '$2=="disk"{print $1}'); do
    SIZE=$(lsblk -d -b -o SIZE /dev/$disk 2>/dev/null | tail -1)
    SIZE_GB=$(echo "scale=2; $SIZE/1024/1024/1024" | bc 2>/dev/null || echo "0")
    MODEL=$(cat /sys/block/$disk/device/model 2>/dev/null | xargs || echo "unknown")
    [ $FIRST_DISK -eq 0 ] && echo ","
    echo "    {\\"name\\": \\"/dev/$disk\\", \\"size_gb\\": $SIZE_GB, \\"model\\": \\"$MODEL\\"}"
    FIRST_DISK=0
done
echo "  ],"

# 网卡信息
echo "  \\"network_interfaces\\": ["
FIRST_NIC=1
for nic in $(ip -br link show | awk '$1!="lo"{print $1}' | grep -v "docker\\|veth\\|br-"); do
    MAC=$(ip link show $nic 2>/dev/null | awk '/link\/ether/{print $2}')
    IP=$(ip -4 addr show $nic 2>/dev/null | awk '/inet /{print $2}' | cut -d/ -f1)
    SPEED=$(ethtool $nic 2>/dev/null | grep Speed | awk '{print $2}')
    [ $FIRST_NIC -eq 0 ] && echo ","
    echo "    {\\"name\\": \\"$nic\\", \\"mac\\": \\"$MAC\\", \\"ip\\": \\"$IP\\", \\"speed\\": \\"$SPEED\\"}"
    FIRST_NIC=0
done
echo "  ],"

# InfiniBand 信息
if command -v ibstat &> /dev/null; then
    IB_COUNT=$(ibstat -l 2>/dev/null | wc -l)
    echo "  \\"infiniband\\": {"
    echo "    \\"count\\": $IB_COUNT"
    echo "  },"
else
    echo "  \\"infiniband\\": null,"
fi

# 服务状态
echo "  \\"services\\": {"
for svc in docker kubelet slurmd salt-minion; do
    STATUS=$(systemctl is-active $svc 2>/dev/null || echo "not-found")
    echo "    \\"$svc\\": \\"$STATUS\\","
done
echo "    \\"_end\\": null"
echo "  }"

echo "}"`,
    },
  ], [t]);

  const loadStatus = async () => {
    setStatusLoading(true);
    try {
      const response = await saltStackAPI.getStatus();
      setStatus(response.data?.data);
      setDemo(Boolean(response.data?.data?.demo));
      setError(null);
    } catch (e) {
      console.error('加载SaltStack状态失败', e);
      setError(e);
    } finally {
      setStatusLoading(false);
    }
  };

  // 加载分组列表
  const loadMinionGroups = async () => {
    setGroupsLoading(true);
    try {
      const response = await saltStackAPI.listMinionGroups();
      setMinionGroups(response.data?.data || []);
    } catch (e) {
      console.error('加载Minion分组失败', e);
    } finally {
      setGroupsLoading(false);
    }
  };

  const loadMinions = async (forceRefresh = false) => {
    setMinionsLoading(true);
    try {
      // 并行获取 Minion 列表、待删除状态、节点指标和 IB 告警
      const [minionsRes, pendingDeletesRes, nodeMetricsRes, ibAlertsRes] = await Promise.all([
        saltStackAPI.getMinions(forceRefresh),
        saltStackAPI.getPendingDeleteMinions().catch(() => ({ data: { minion_ids: [] } })),
        saltStackAPI.getNodeMetrics().catch(() => ({ data: { data: [] } })),
        saltStackAPI.getIBPortAlerts().catch(() => ({ data: { data: [] } })),
      ]);
      
      const minionList = minionsRes.data?.data || [];
      const pendingDeleteIds = new Set(pendingDeletesRes.data?.minion_ids || []);
      const nodeMetricsList = nodeMetricsRes.data?.data || [];
      
      // 更新 IB 告警状态
      setIbAlerts(ibAlertsRes.data?.data || []);
      
      // 构建节点指标映射表
      const metricsMap = {};
      nodeMetricsList.forEach(m => {
        metricsMap[m.minion_id] = m;
      });
      
      // 标记待删除的 Minion 并合并节点指标
      const minionsWithDeleteStatus = minionList.map(minion => {
        const minionId = minion.id || minion.name;
        const metrics = metricsMap[minionId];
        return {
          ...minion,
          pending_delete: pendingDeleteIds.has(minionId),
          status: pendingDeleteIds.has(minionId) ? 'deleting' : minion.status,
          // 合并采集到的 GPU/IB 指标
          gpu_info: metrics?.gpu ? {
            gpu_count: metrics.gpu.count || 0,
            gpu_model: metrics.gpu.model || '',
            driver_version: metrics.gpu.driver_version || '',
            cuda_version: metrics.gpu.cuda_version || '',
            memory_total: metrics.gpu.memory_total || '',
            // 新增 GPU 利用率和显存信息
            utilization: metrics.gpu.utilization || 0,
            memory_used: metrics.gpu.memory_used || '',
            memory_free: metrics.gpu.memory_free || '',
            gpus: metrics.gpu.gpus || [],
          } : minion.gpu_info,
          // NPU 信息 (华为昇腾、寒武纪等)
          npu_info: metrics?.npu ? {
            vendor: metrics.npu.vendor || '',
            version: metrics.npu.version || '',
            npu_count: metrics.npu.count || 0,
            npu_model: metrics.npu.model || '',
            utilization: metrics.npu.avg_utilization || 0,
            memory_used_mb: metrics.npu.memory_used_mb || 0,
            memory_total_mb: metrics.npu.memory_total_mb || 0,
            npus: metrics.npu.npus || [],
          } : minion.npu_info,
          // TPU 信息
          tpu_info: metrics?.tpu ? {
            vendor: metrics.tpu.vendor || '',
            version: metrics.tpu.version || '',
            tpu_count: metrics.tpu.count || 0,
            tpu_model: metrics.tpu.model || '',
          } : minion.tpu_info,
          ib_info: metrics?.ib ? {
            active_count: metrics.ib.active_count || 0,
            ports: metrics.ib.ports || [],
          } : minion.ib_info,
          // 新增 CPU/内存/网络/RoCE 指标
          cpu_info: metrics?.cpu ? {
            model: metrics.cpu.model || '',
            cores: metrics.cpu.cores || 0,
            threads: metrics.cpu.threads || 0,
            frequency: metrics.cpu.frequency || '',
            usage: metrics.cpu.usage || 0,
          } : null,
          memory_info: metrics?.memory ? {
            total: metrics.memory.total || '',
            used: metrics.memory.used || '',
            free: metrics.memory.free || '',
            usage_percent: metrics.memory.usage_percent || 0,
          } : null,
          network_info: metrics?.network ? {
            interfaces: metrics.network.interfaces || [],
            total_rx_rate: metrics.network.total_rx_rate || '',
            total_tx_rate: metrics.network.total_tx_rate || '',
          } : null,
          roce_info: metrics?.roce ? {
            count: metrics.roce.count || 0,
            interfaces: metrics.roce.interfaces || [],
          } : null,
          metrics_collected_at: metrics?.collected_at,
        };
      });
      
      setMinions(minionsWithDeleteStatus);
      setDemo(prev => prev || Boolean(minionsRes.data?.demo));
    } catch (e) {
      console.error('加载SaltStack Minions失败', e);
    } finally {
      setMinionsLoading(false);
    }
  };

  const loadJobs = async () => {
    setJobsLoading(true);
    try {
      const response = await saltStackAPI.getJobs(10);
      const jobsData = response.data?.data || [];
      
      // 关联 TaskID：优先使用后端返回的 task_id，如果没有则从 localStorage 查找
      const jobsWithTaskId = jobsData.map(job => {
        // 后端返回的字段是 task_id (snake_case)
        const backendTaskId = job.task_id;
        // 从 localStorage 查找作为备用
        const localTaskId = getTaskIdByJid(job.jid);
        // 优先使用后端返回的，否则使用本地缓存
        const taskId = backendTaskId || localTaskId;
        
        // 如果后端有 task_id，同步到本地缓存（保持双向一致）
        if (backendTaskId && !localTaskId) {
          addJidTaskIdMapping(job.jid, backendTaskId);
        }
        
        return taskId ? { ...job, taskId } : job;
      });
      
      setJobs(jobsWithTaskId);
      setDemo(prev => prev || Boolean(response.data?.demo));
    } catch (e) {
      console.error('加载SaltStack Jobs失败', e);
    } finally {
      setJobsLoading(false);
    }
  };

  // 查看作业详情
  const viewJobDetail = async (jid) => {
    setJobDetailLoading(true);
    setJobDetailVisible(true);
    try {
      const response = await saltStackAPI.getJobDetail(jid);
      setJobDetail(response.data);
    } catch (e) {
      console.error('获取作业详情失败', e);
      message.error(t('saltstack.getJobDetailFailed', '获取作业详情失败'));
      setJobDetail(null);
    } finally {
      setJobDetailLoading(false);
    }
  };

  // 加载安装任务历史
  const loadInstallTasks = useCallback(async (page = installTasksPage.current, pageSize = installTasksPage.pageSize) => {
    setInstallTasksLoading(true);
    try {
      const offset = (page - 1) * pageSize;
      const response = await saltStackAPI.listBatchInstallTasks({ limit: pageSize, offset });
      const data = response.data?.data || {};
      setInstallTasks(data.tasks || []);
      setInstallTasksTotal(data.total || 0);
      setInstallTasksPage({ current: page, pageSize });
    } catch (e) {
      console.error('加载安装任务历史失败', e);
    } finally {
      setInstallTasksLoading(false);
    }
  }, [installTasksPage.current, installTasksPage.pageSize]);

  // 加载删除任务历史
  const loadDeleteTasks = useCallback(async (limit = 100) => {
    setDeleteTasksLoading(true);
    try {
      const response = await saltStackAPI.listDeleteTasks({ limit });
      const data = response.data?.data || [];
      setDeleteTasks(data);
      setDeleteTasksTotal(response.data?.count || data.length);
    } catch (e) {
      console.error('加载删除任务历史失败', e);
    } finally {
      setDeleteTasksLoading(false);
    }
  }, []);

  // 加载删除任务日志
  const loadDeleteTaskLogs = async (minionId) => {
    try {
      const response = await saltStackAPI.getDeleteTaskLogs(minionId);
      const logs = response.data?.data || [];
      setDeleteTaskLogs(prev => ({ ...prev, [minionId]: logs }));
    } catch (e) {
      console.error('加载删除任务日志失败', e);
    }
  };

  // 计算按分组筛选后的 minions
  const filteredMinions = useMemo(() => {
    if (overviewGroupFilter === 'all') {
      return minions;
    }
    if (overviewGroupFilter === 'ungrouped') {
      return minions.filter(m => !m.group || m.group === '');
    }
    return minions.filter(m => m.group === overviewGroupFilter);
  }, [minions, overviewGroupFilter]);

  // 计算分组聚合统计
  const groupStats = useMemo(() => {
    const stats = {
      total: minions.length,
      online: minions.filter(m => m.status?.toLowerCase() === 'up' || m.status?.toLowerCase() === 'accepted').length,
      offline: minions.filter(m => m.status?.toLowerCase() !== 'up' && m.status?.toLowerCase() !== 'accepted').length,
      byGroup: {},
      gpuInfo: { total: 0, withGpu: 0, models: {} },
      npuInfo: { total: 0, withNpu: 0, vendors: {} },
      ibInfo: { total: 0, active: 0, down: 0 },
    };

    // 按分组统计
    minions.forEach(m => {
      const groupName = m.group || '未分组';
      if (!stats.byGroup[groupName]) {
        stats.byGroup[groupName] = {
          total: 0,
          online: 0,
          offline: 0,
          gpuCount: 0,
          npuCount: 0,
          ibActive: 0,
        };
      }
      stats.byGroup[groupName].total++;
      const isOnline = m.status?.toLowerCase() === 'up' || m.status?.toLowerCase() === 'accepted';
      if (isOnline) {
        stats.byGroup[groupName].online++;
      } else {
        stats.byGroup[groupName].offline++;
      }

      // GPU 统计
      if (m.gpu_info?.gpu_count > 0 || m.gpu_model) {
        stats.gpuInfo.withGpu++;
        stats.gpuInfo.total += m.gpu_info?.gpu_count || 1;
        const model = m.gpu_info?.gpu_model || m.gpu_model || 'Unknown';
        stats.gpuInfo.models[model] = (stats.gpuInfo.models[model] || 0) + 1;
        stats.byGroup[groupName].gpuCount += m.gpu_info?.gpu_count || 1;
      }

      // NPU 统计 (华为昇腾、寒武纪等)
      if (m.npu_info?.npu_count > 0) {
        stats.npuInfo.withNpu++;
        stats.npuInfo.total += m.npu_info?.npu_count || 0;
        const vendor = m.npu_info?.vendor || 'Unknown';
        stats.npuInfo.vendors[vendor] = (stats.npuInfo.vendors[vendor] || 0) + (m.npu_info?.npu_count || 1);
        stats.byGroup[groupName].npuCount += m.npu_info?.npu_count || 0;
      }

      // IB 统计（优先使用采集到的 ib_info）
      if (m.ib_info?.active_count > 0) {
        stats.ibInfo.total++;
        stats.ibInfo.active++;
        stats.byGroup[groupName].ibActive += m.ib_info.active_count || 1;
      } else if (m.ib_status) {
        stats.ibInfo.total++;
        if (m.ib_status === 'Active' || m.ib_status === 'active') {
          stats.ibInfo.active++;
          stats.byGroup[groupName].ibActive++;
        } else {
          stats.ibInfo.down++;
        }
      }
    });

    return stats;
  }, [minions]);

  const loadAllData = async () => {
    // 先加载 master 状态，确保 SaltStack 服务可用
    await loadStatus();
    // 然后并行加载 minion 列表、jobs 和分组
    await Promise.all([loadMinions(), loadJobs(), loadMinionGroups()]);
  };

  // 加载作业配置
  const loadJobConfig = useCallback(async () => {
    setJobConfigLoading(true);
    try {
      const response = await saltStackAPI.getJobConfig();
      if (response.success) {
        setJobConfig(response.config);
        setJobStats(response.stats);
        // 设置表单值
        if (response.config) {
          jobConfigForm.setFieldsValue({
            retention_days: response.config.retention_days,
            auto_cleanup_enabled: response.config.auto_cleanup_enabled,
            cleanup_interval_hours: response.config.cleanup_interval_hours,
            max_jobs_count: response.config.max_jobs_count,
            redis_cache_days: response.config.redis_cache_days,
          });
        }
      }
    } catch (error) {
      console.error('加载作业配置失败:', error);
      message.error(t('saltstack.loadJobConfigFailed', '加载作业配置失败'));
    } finally {
      setJobConfigLoading(false);
    }
  }, [jobConfigForm, t]);

  // 保存作业配置
  const saveJobConfig = async (values) => {
    try {
      const response = await saltStackAPI.updateJobConfig(values);
      if (response.success) {
        message.success(t('saltstack.saveJobConfigSuccess', '配置保存成功'));
        setJobConfig(response.config);
      } else {
        message.error(response.error || t('saltstack.saveJobConfigFailed', '保存配置失败'));
      }
    } catch (error) {
      console.error('保存作业配置失败:', error);
      message.error(t('saltstack.saveJobConfigFailed', '保存配置失败'));
    }
  };

  // 手动触发清理
  const triggerCleanup = async () => {
    setCleanupLoading(true);
    try {
      const response = await saltStackAPI.triggerJobCleanup();
      if (response.success) {
        message.success(t('saltstack.cleanupSuccess', `已清理 ${response.cleaned_count} 条记录`));
        // 刷新统计信息
        loadJobConfig();
      } else {
        message.error(response.error || t('saltstack.cleanupFailed', '清理失败'));
      }
    } catch (error) {
      console.error('触发清理失败:', error);
      message.error(t('saltstack.cleanupFailed', '清理失败'));
    } finally {
      setCleanupLoading(false);
    }
  };

  // 仅加载 Minion 数据（不包含 Master 状态）
  const loadMinionData = async () => {
    await Promise.all([loadMinions(), loadJobs()]);
  };

  // 页面初始化效果 - 立即显示静态内容
  useEffect(() => {
    // 标记页面已加载，显示静态内容
    setPageLoaded(true);
    
    // 异步加载数据（非阻塞）
    setTimeout(() => {
      loadAllData();
    }, 100); // 延迟100ms让静态内容先渲染
    
    // 设置定时刷新
    // Master 状态检查：3分钟一次（180秒）
    const masterInterval = setInterval(loadStatus, 180000);
    // Minion 列表检查：1分钟一次（60秒）
    const minionInterval = setInterval(loadMinionData, 60000);
    
    return () => {
      clearInterval(masterInterval);
      clearInterval(minionInterval);
    };
  }, []);

  // 当主机列表变化时，计算动态并行度
  useEffect(() => {
    const validHosts = batchInstallHosts.filter(h => h.host && h.host.trim());
    const hostCount = validHosts.length;
    
    // 使用前端模拟的动态并行度计算（与后端逻辑一致）
    // 这样可以在用户输入时实时显示，无需调用API
    const calculateParallel = (count) => {
      if (count <= 0) return { parallel: 0, percentage: 0 };
      let parallel;
      if (count <= 20) parallel = count;
      else if (count <= 50) parallel = Math.ceil(count * 0.6);
      else if (count <= 100) parallel = Math.ceil(count * 0.5);
      else if (count <= 500) parallel = Math.ceil(count * 0.2);
      else if (count <= 1000) parallel = Math.ceil(count * 0.1);
      else if (count <= 5000) parallel = Math.ceil(count * 0.03);
      else if (count <= 10000) parallel = Math.ceil(count * 0.01);
      else parallel = Math.ceil(count * 0.001);
      
      parallel = Math.max(1, Math.min(parallel, 100)); // 最小1，最大100
      return {
        parallel,
        percentage: count > 0 ? (parallel / count * 100) : 0,
        host_count: count,
        is_auto_calculate: true
      };
    };
    
    setParallelInfo(calculateParallel(hostCount));
  }, [batchInstallHosts]);

  // 自动刷新 Minions 列表
  useEffect(() => {
    if (autoRefreshMinions) {
      autoRefreshMinionsRef.current = setInterval(() => {
        loadMinions(false); // 静默刷新，不显示 loading
      }, autoRefreshInterval * 1000);
    } else {
      if (autoRefreshMinionsRef.current) {
        clearInterval(autoRefreshMinionsRef.current);
        autoRefreshMinionsRef.current = null;
      }
    }
    return () => {
      if (autoRefreshMinionsRef.current) {
        clearInterval(autoRefreshMinionsRef.current);
        autoRefreshMinionsRef.current = null;
      }
    };
  }, [autoRefreshMinions, autoRefreshInterval]);

  // 自动刷新安装任务
  useEffect(() => {
    if (autoRefreshTasks) {
      autoRefreshTasksRef.current = setInterval(() => {
        loadInstallTasks();
      }, autoRefreshInterval * 1000);
    } else {
      if (autoRefreshTasksRef.current) {
        clearInterval(autoRefreshTasksRef.current);
        autoRefreshTasksRef.current = null;
      }
    }
    return () => {
      if (autoRefreshTasksRef.current) {
        clearInterval(autoRefreshTasksRef.current);
        autoRefreshTasksRef.current = null;
      }
    };
  }, [autoRefreshTasks, autoRefreshInterval, loadInstallTasks]);

  // 自动刷新系统概览
  useEffect(() => {
    if (autoRefreshOverview) {
      autoRefreshOverviewRef.current = setInterval(() => {
        loadStatus();
        loadMinions(false);
      }, autoRefreshInterval * 1000);
    } else {
      if (autoRefreshOverviewRef.current) {
        clearInterval(autoRefreshOverviewRef.current);
        autoRefreshOverviewRef.current = null;
      }
    }
    return () => {
      if (autoRefreshOverviewRef.current) {
        clearInterval(autoRefreshOverviewRef.current);
        autoRefreshOverviewRef.current = null;
      }
    };
  }, [autoRefreshOverview, autoRefreshInterval]);

  // 关闭SSE
  const closeSSE = () => {
    if (sseRef.current) {
      try { sseRef.current.close?.(); } catch {}
      sseRef.current = null;
    }
  };

  // 关闭批量安装SSE
  const closeBatchSSE = () => {
    if (batchSseRef.current) {
      try { batchSseRef.current.close?.(); } catch {}
      batchSseRef.current = null;
    }
  };

  // 添加主机行
  const addHostRow = () => {
    setBatchInstallHosts([
      ...batchInstallHosts,
      { key: Date.now(), host: '', port: 22, username: 'root', password: '', use_sudo: false, group: '', install_categraf: false }
    ]);
  };

  // 复制第一行配置到当前行（仅复制端口、用户名、密码、sudo、分组、Categraf配置，不复制 host）
  const copyFirstRowConfig = (targetKey) => {
    if (batchInstallHosts.length === 0) return;
    const firstRow = batchInstallHosts[0];
    setBatchInstallHosts(batchInstallHosts.map(h => 
      h.key === targetKey ? { 
        ...h, 
        port: firstRow.port, 
        username: firstRow.username, 
        password: firstRow.password, 
        use_sudo: firstRow.use_sudo,
        group: firstRow.group,
        install_categraf: firstRow.install_categraf,
      } : h
    ));
    message.success(t('saltstack.configCopied', '已复制第一行配置'));
  };

  // 删除主机行
  const removeHostRow = (key) => {
    if (batchInstallHosts.length <= 1) {
      message.warning(t('saltstack.atLeastOneHostRequired'));
      return;
    }
    setBatchInstallHosts(batchInstallHosts.filter(h => h.key !== key));
  };

  // IP 地址验证正则表达式
  const isValidIPOrHostname = (value) => {
    if (!value || !value.trim()) return true; // 空值允许
    const trimmed = value.trim();
    
    // IPv4 地址验证
    const ipv4Pattern = /^(\d{1,3}\.){3}\d{1,3}$/;
    if (ipv4Pattern.test(trimmed)) {
      const parts = trimmed.split('.');
      return parts.every(part => {
        const num = parseInt(part, 10);
        return num >= 0 && num <= 255;
      });
    }
    
    // IPv6 地址验证 (简化版)
    const ipv6Pattern = /^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}$/;
    if (ipv6Pattern.test(trimmed)) return true;
    
    // 主机名验证 (允许域名和简单主机名)
    const hostnamePattern = /^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/;
    return hostnamePattern.test(trimmed);
  };

  // 解析布尔值（支持字符串 "true"/"false"/"yes"/"no"/"1"/"0" 和布尔值）
  const parseBoolValue = (value) => {
    if (typeof value === 'boolean') return value;
    if (typeof value === 'string') {
      const lower = value.toLowerCase().trim();
      return lower === 'true' || lower === 'yes' || lower === '1';
    }
    if (typeof value === 'number') return value === 1;
    return false;
  };

  // 检查主机是否重复
  const isDuplicateHost = (host, currentKey) => {
    if (!host || !host.trim()) return false;
    const trimmed = host.trim().toLowerCase();
    return batchInstallHosts.some(h => 
      h.key !== currentKey && h.host && h.host.trim().toLowerCase() === trimmed
    );
  };

  // 更新主机行（带验证）
  const updateHostRow = (key, field, value) => {
    if (field === 'host' && value) {
      const trimmedValue = value.trim();
      // IP/主机名格式验证
      if (trimmedValue && !isValidIPOrHostname(trimmedValue)) {
        message.warning(t('saltstack.invalidIPOrHostname', '无效的 IP 地址或主机名格式'));
      }
      // 重复检测
      if (isDuplicateHost(trimmedValue, key)) {
        message.warning(t('saltstack.duplicateHost', '该主机地址已存在'));
      }
    }
    setBatchInstallHosts(batchInstallHosts.map(h => 
      h.key === key ? { ...h, [field]: value } : h
    ));
  };

  // 下载主机模板
  const downloadHostTemplate = async (format) => {
    try {
      const response = await fetch(`/api/saltstack/host-templates/download/${format}`);
      if (!response.ok) throw new Error(t('saltstack.downloadFailed'));
      
      const blob = await response.blob();
      const filename = `hosts_template.${format === 'ini' ? 'ini' : format}`;
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      message.success(t('saltstack.downloadedTemplate', { filename }));
    } catch (e) {
      message.error(t('saltstack.downloadTemplateFailed') + ': ' + e.message);
    }
  };

  // 同步导入配置中的分组到分组管理
  // 检查导入的主机中是否有新的分组名，如果有则自动创建
  const syncImportedGroups = async (hosts) => {
    // 提取所有非空的分组名
    const importedGroupNames = [...new Set(
      hosts
        .map(h => (h.group || '').trim())
        .filter(g => g !== '')
    )];
    
    if (importedGroupNames.length === 0) {
      return; // 没有分组需要同步
    }

    // 获取现有分组名列表
    const existingGroupNames = new Set(minionGroups.map(g => g.name));
    
    // 找出需要创建的新分组
    const newGroupNames = importedGroupNames.filter(name => !existingGroupNames.has(name));
    
    if (newGroupNames.length === 0) {
      return; // 所有分组都已存在
    }

    console.log('🔄 需要创建的新分组:', newGroupNames);

    // 预定义的颜色列表，用于自动分配
    const colors = ['blue', 'green', 'orange', 'purple', 'cyan', 'magenta', 'gold', 'lime', 'volcano', 'geekblue'];
    
    // 批量创建分组
    let createdCount = 0;
    for (let i = 0; i < newGroupNames.length; i++) {
      const groupName = newGroupNames[i];
      try {
        const resp = await saltStackAPI.createMinionGroup({
          name: groupName,
          description: t('saltstack.autoCreatedGroup', '通过导入配置自动创建'),
          color: colors[i % colors.length],
        });
        
        if (resp.data?.success) {
          createdCount++;
          console.log(`✓ 分组 "${groupName}" 创建成功`);
        } else {
          console.warn(`⚠️ 分组 "${groupName}" 创建失败:`, resp.data?.message);
        }
      } catch (e) {
        console.warn(`⚠️ 分组 "${groupName}" 创建失败:`, e.message);
      }
    }

    // 刷新分组列表
    if (createdCount > 0) {
      await loadMinionGroups();
      message.info(t('saltstack.autoCreatedGroups', { count: createdCount }));
    }
  };

  // 导入主机文件
  const handleFileImport = async (file) => {
    setImportLoading(true);
    
    // 调试日志
    console.group('🔍 [DEBUG] 主机文件导入');
    console.log('📄 文件名:', file.name);
    console.log('📦 文件大小:', file.size, 'bytes');
    console.log('📝 文件类型:', file.type);
    
    try {
      const content = await file.text();
      console.log('📜 文件内容长度:', content.length);
      console.log('📜 文件内容预览 (前500字符):', content.substring(0, 500));
      
      console.log('🌐 调用 API: parseHostFile');
      const response = await saltStackAPI.parseHostFile(content, file.name);
      
      console.log('✅ API 响应:', response);
      console.log('✅ 响应数据:', response.data);
      
      if (!response.data?.success) {
        console.error('❌ 解析失败:', response.data?.message || response.data?.error);
        throw new Error(response.data?.message || response.data?.error || t('saltstack.parseFailed'));
      }

      const hosts = response.data?.data?.hosts || [];
      console.log('📋 解析到的主机数:', hosts.length);
      console.log('📋 解析到的主机列表:', hosts);
      
      if (hosts.length === 0) {
        console.warn('⚠️ 没有有效的主机配置');
        message.warning(t('saltstack.noValidHostConfig'));
        console.groupEnd();
        return false;
      }

      // 验证并转换主机列表
      let validCount = 0;
      let invalidCount = 0;
      let duplicateCount = 0;
      const invalidHosts = [];

      // 获取现有的主机列表（用于去重检查）
      const existingHosts = new Set(
        batchInstallHosts
          .filter(h => h.host && h.host.trim())
          .map(h => h.host.trim().toLowerCase())
      );
      console.log('🔄 现有主机列表:', Array.from(existingHosts));

      // 用于跟踪本次导入中的重复
      const importedHosts = new Set();

      const newHosts = [];
      hosts.forEach((h, idx) => {
        const hostValue = (h.host || '').trim();
        const hostLower = hostValue.toLowerCase();

        // 验证 IP/主机名格式
        if (hostValue && !isValidIPOrHostname(hostValue)) {
          invalidCount++;
          invalidHosts.push(hostValue);
          console.warn(`⚠️ 主机 ${idx + 1}: ${hostValue} - IP/主机名格式无效`);
          return; // 跳过无效主机
        }

        // 检查与现有列表的重复
        if (hostValue && existingHosts.has(hostLower)) {
          duplicateCount++;
          console.warn(`⚠️ 主机 ${idx + 1}: ${hostValue} - 与现有列表重复`);
          return; // 跳过重复主机
        }

        // 检查本次导入中的重复
        if (hostValue && importedHosts.has(hostLower)) {
          duplicateCount++;
          console.warn(`⚠️ 主机 ${idx + 1}: ${hostValue} - 本次导入中重复`);
          return; // 跳过重复主机
        }

        // 添加到导入集合
        if (hostValue) {
          importedHosts.add(hostLower);
        }

        validCount++;
        const newHost = {
          key: Date.now() + idx + validCount, // 确保 key 唯一
          host: hostValue,
          port: h.port || 22,
          username: h.username || 'root',
          password: h.password || '',
          use_sudo: parseBoolValue(h.use_sudo),
          minion_id: h.minion_id || '',
          group: h.group || '',
          install_categraf: parseBoolValue(h.install_categraf)
        };
        console.log(`✓ 主机 ${idx + 1}: ${hostValue} - 有效`, newHost);
        newHosts.push(newHost);
      });

      console.log('📊 导入统计:', {
        总数: hosts.length,
        有效: validCount,
        无效: invalidCount,
        重复: duplicateCount,
        无效主机: invalidHosts
      });

      if (newHosts.length === 0) {
        if (duplicateCount > 0) {
          message.warning(t('saltstack.allHostsDuplicate', `所有 ${duplicateCount} 个主机已存在于列表中`));
        } else if (invalidCount > 0) {
          message.error(t('saltstack.allHostsInvalid', `所有 ${invalidCount} 个主机地址格式无效`));
        } else {
          message.warning(t('saltstack.noValidHostConfig'));
        }
        console.groupEnd();
        return false;
      }

      // 如果当前只有一个空行，则替换；否则追加
      if (batchInstallHosts.length === 1 && !batchInstallHosts[0].host) {
        console.log('🔄 替换现有空行');
        setBatchInstallHosts(newHosts);
      } else {
        console.log('🔄 追加到现有列表');
        setBatchInstallHosts([...batchInstallHosts, ...newHosts]);
      }

      // 构建导入结果消息
      let resultMsg = t('saltstack.importedHosts', { count: validCount });
      if (duplicateCount > 0) {
        resultMsg += `, ${t('saltstack.skippedDuplicates', { count: duplicateCount })}`;
      }
      if (invalidCount > 0) {
        resultMsg += `, ${t('saltstack.skippedInvalid', { count: invalidCount })}`;
      }
      
      if (duplicateCount > 0 || invalidCount > 0) {
        message.info(resultMsg);
      } else {
        message.success(resultMsg);
      }
      
      // 同步导入配置中的分组到分组管理
      await syncImportedGroups(newHosts);
      
      console.log('✅ 导入完成:', resultMsg);
      console.groupEnd();

    } catch (e) {
      console.error('❌ 文件导入失败:', e);
      console.error('❌ 错误详情:', e.response?.data);
      console.groupEnd();
      message.error(t('saltstack.importFailed') + ': ' + (e.response?.data?.error || e.message));
    } finally {
      setImportLoading(false);
    }
    return false; // 阻止默认上传行为
  };

  // 打开粘贴导入弹窗
  const openPasteImportModal = () => {
    setPasteImportVisible(true);
    setPasteContent('');
    setPasteFormat('csv');
  };

  // 处理粘贴导入
  const handlePasteImport = async () => {
    if (!pasteContent || !pasteContent.trim()) {
      message.warning(t('saltstack.pasteContentEmpty', '请输入配置内容'));
      return;
    }

    setPasteImportLoading(true);
    
    console.group('🔍 [DEBUG] 粘贴内容导入');
    console.log('📝 格式:', pasteFormat);
    console.log('📜 内容长度:', pasteContent.length);
    console.log('📜 内容预览:', pasteContent.substring(0, 300));
    
    try {
      // 构造虚拟文件名以便后端识别格式
      const filename = `paste.${pasteFormat}`;
      
      console.log('🌐 调用 API: parseHostFile');
      const response = await saltStackAPI.parseHostFile(pasteContent, filename);
      
      console.log('✅ API 响应:', response);
      
      if (!response.data?.success) {
        console.error('❌ 解析失败:', response.data?.message || response.data?.error);
        throw new Error(response.data?.message || response.data?.error || t('saltstack.parseFailed'));
      }

      const hosts = response.data?.data?.hosts || [];
      console.log('📋 解析到的主机数:', hosts.length);
      
      if (hosts.length === 0) {
        console.warn('⚠️ 没有有效的主机配置');
        message.warning(t('saltstack.noValidHostConfig'));
        console.groupEnd();
        return;
      }

      // 验证并转换主机列表（复用现有逻辑）
      let validCount = 0;
      let invalidCount = 0;
      let duplicateCount = 0;
      const invalidHosts = [];

      const existingHosts = new Set(
        batchInstallHosts
          .filter(h => h.host && h.host.trim())
          .map(h => h.host.trim().toLowerCase())
      );

      const importedHosts = new Set();
      const newHosts = [];
      
      hosts.forEach((h, idx) => {
        const hostValue = (h.host || '').trim();
        const hostLower = hostValue.toLowerCase();

        if (hostValue && !isValidIPOrHostname(hostValue)) {
          invalidCount++;
          invalidHosts.push(hostValue);
          return;
        }

        if (hostValue && existingHosts.has(hostLower)) {
          duplicateCount++;
          return;
        }

        if (hostValue && importedHosts.has(hostLower)) {
          duplicateCount++;
          return;
        }

        if (hostValue) {
          importedHosts.add(hostLower);
        }

        validCount++;
        // 调试: 打印原始值和解析后的值
        console.log(`🔍 主机 ${hostValue} install_categraf 原始值:`, h.install_categraf, `类型:`, typeof h.install_categraf, `=> 解析结果:`, parseBoolValue(h.install_categraf));
        newHosts.push({
          key: Date.now() + idx + validCount,
          host: hostValue,
          port: h.port || 22,
          username: h.username || 'root',
          password: h.password || '',
          use_sudo: parseBoolValue(h.use_sudo),
          minion_id: h.minion_id || '',
          group: h.group || '',
          install_categraf: parseBoolValue(h.install_categraf)
        });
      });

      console.log('📊 导入统计:', { 总数: hosts.length, 有效: validCount, 无效: invalidCount, 重复: duplicateCount });
      console.log('📋 最终 newHosts:', newHosts.map(h => ({ host: h.host, install_categraf: h.install_categraf })));

      if (newHosts.length === 0) {
        if (duplicateCount > 0) {
          message.warning(t('saltstack.allHostsDuplicate', `所有 ${duplicateCount} 个主机已存在于列表中`));
        } else if (invalidCount > 0) {
          message.error(t('saltstack.allHostsInvalid', `所有 ${invalidCount} 个主机地址格式无效`));
        } else {
          message.warning(t('saltstack.noValidHostConfig'));
        }
        console.groupEnd();
        return;
      }

      // 如果当前只有一个空行，则替换；否则追加
      if (batchInstallHosts.length === 1 && !batchInstallHosts[0].host) {
        setBatchInstallHosts(newHosts);
      } else {
        setBatchInstallHosts([...batchInstallHosts, ...newHosts]);
      }

      // 构建导入结果消息
      let resultMsg = t('saltstack.importedHosts', { count: validCount });
      if (duplicateCount > 0) {
        resultMsg += `, ${t('saltstack.skippedDuplicates', { count: duplicateCount })}`;
      }
      if (invalidCount > 0) {
        resultMsg += `, ${t('saltstack.skippedInvalid', { count: invalidCount })}`;
      }
      
      message.success(resultMsg);
      console.log('✅ 粘贴导入完成:', resultMsg);
      console.groupEnd();
      
      // 同步导入配置中的分组到分组管理
      await syncImportedGroups(newHosts);
      
      // 关闭弹窗
      setPasteImportVisible(false);
      setPasteContent('');

    } catch (e) {
      console.error('❌ 粘贴导入失败:', e);
      console.groupEnd();
      message.error(t('saltstack.importFailed') + ': ' + (e.response?.data?.error || e.message));
    } finally {
      setPasteImportLoading(false);
    }
  };

  // 获取粘贴格式的示例内容
  const getPasteFormatExample = (format) => {
    switch (format) {
      case 'csv':
        return `host,port,username,password,use_sudo,group,install_categraf
192.168.1.100,22,root,password123,false,web,true
192.168.1.101,22,admin,pass456,true,db,true
node1.example.com,2222,deploy,secretpwd,false,,false`;
      case 'json':
        return `[
  {"host": "192.168.1.100", "port": 22, "username": "root", "password": "password123", "use_sudo": false, "group": "web", "install_categraf": true},
  {"host": "192.168.1.101", "port": 22, "username": "admin", "password": "pass456", "use_sudo": true, "group": "db", "install_categraf": true},
  {"host": "node1.example.com", "port": 2222, "username": "deploy", "password": "secretpwd", "install_categraf": false}
]`;
      case 'yaml':
        return `hosts:
  - host: 192.168.1.100
    port: 22
    username: root
    password: password123
    use_sudo: false
    group: web
    install_categraf: true
  - host: 192.168.1.101
    port: 22
    username: admin
    password: pass456
    use_sudo: true
    group: db
    install_categraf: true
  - host: node1.example.com
    port: 2222
    username: deploy
    password: secretpwd
    install_categraf: false`;
      case 'ini':
        return `[web]
192.168.1.100 ansible_port=22 ansible_user=root ansible_password=password123 install_categraf=true

[db]
192.168.1.101 ansible_port=22 ansible_user=admin ansible_password=pass456 ansible_become=true install_categraf=true

[all]
node1.example.com ansible_port=2222 ansible_user=deploy ansible_password=secretpwd install_categraf=false`;
      default:
        return '';
    }
  };

  // 模板下载菜单
  const templateMenu = (
    <Menu onClick={({ key }) => downloadHostTemplate(key)}>
      <Menu.Item key="csv" icon={<FileTextOutlined />}>
        CSV 格式 (.csv)
      </Menu.Item>
      <Menu.Item key="json" icon={<FileTextOutlined />}>
        JSON 格式 (.json)
      </Menu.Item>
      <Menu.Item key="yaml" icon={<FileTextOutlined />}>
        YAML 格式 (.yaml)
      </Menu.Item>
      <Menu.Item key="ini" icon={<FileTextOutlined />}>
        Ansible INI 格式 (.ini)
      </Menu.Item>
    </Menu>
  );

  // 打开批量安装弹窗
  const openBatchInstallModal = () => {
    setBatchInstallVisible(true);
    setBatchInstallEvents([]);
    setBatchInstallTaskId('');
    setBatchInstallRunning(false);
    setBatchInstallHosts([
      { key: Date.now(), host: '', port: 22, username: 'root', password: '', use_sudo: false }
    ]);
    batchInstallForm.setFieldsValue({
      parallel: 3,
      master_host: 'salt',
      install_type: 'saltstack',
      auto_accept: true,
      global_use_sudo: false
    });
  };

  // 启动批量安装SSE
  const startBatchInstallSSE = (taskId) => {
    closeBatchSSE();
    const url = saltStackAPI.getBatchInstallStreamUrl(taskId);
    const es = new EventSource(url, { withCredentials: false });
    batchSseRef.current = es;
    
    es.onmessage = (evt) => {
      try {
        const data = JSON.parse(evt.data);
        console.log('[批量安装 SSE事件]', data.type, data);
        setBatchInstallEvents((prev) => [...prev, data]);
        
        // 实时更新任务列表中的进度（如果有进度数据）
        if (data.type === 'progress' && data.data) {
          const { completed, total, success, failed, progress, host_result } = data.data;
          setInstallTasks((prevTasks) => {
            return prevTasks.map(task => {
              if (task.taskName?.includes(taskId)) {
                return {
                  ...task,
                  // 更新所有统计字段
                  totalHosts: total || task.totalHosts,
                  successHosts: success ?? task.successHosts,
                  failedHosts: failed ?? task.failedHosts,
                  // 如果有 host_result，更新 hostResults
                  hostResults: host_result ? [
                    ...(task.hostResults || []),
                    host_result
                  ] : task.hostResults
                };
              }
              return task;
            });
          });
        }
        
        // 处理完成事件时也更新统计数据
        if (data.type === 'complete' && data.data) {
          const { total_hosts, success_hosts, failed_hosts, status } = data.data;
          setInstallTasks((prevTasks) => {
            return prevTasks.map(task => {
              if (task.taskName?.includes(taskId)) {
                return {
                  ...task,
                  totalHosts: total_hosts || task.totalHosts,
                  successHosts: success_hosts ?? task.successHosts,
                  failedHosts: failed_hosts ?? task.failedHosts,
                  status: status || task.status
                };
              }
              return task;
            });
          });
        }
        
        if (data.type === 'complete' || data.type === 'error' || data.type === 'closed') {
          setTimeout(() => {
            setBatchInstallRunning(false);
            closeBatchSSE();
            // 刷新 minions 列表和安装任务列表
            loadMinions();
            loadInstallTasks(1);
          }, 500);
        }
      } catch (err) {
        console.error('[批量安装 SSE] 解析消息失败:', err);
      }
    };
    
    es.onerror = (err) => {
      console.error('[批量安装 SSE] 连接错误:', err);
      closeBatchSSE();
      setBatchInstallRunning(false);
      // SSE 错误时也刷新任务列表以获取最新状态
      loadInstallTasks(1);
    };
  };

  // 执行批量安装
  const handleBatchInstall = async () => {
    try {
      const values = await batchInstallForm.validateFields();
      
      // 验证主机列表
      const validHosts = batchInstallHosts.filter(h => h.host && h.host.trim());
      if (validHosts.length === 0) {
        message.error(t('saltstack.atLeastOneHost'));
        return;
      }

      // 检查必填字段
      for (const h of validHosts) {
        if (!h.username || !h.password) {
          message.error(t('saltstack.missingCredentials', { host: h.host }));
          return;
        }
      }

      setBatchInstallRunning(true);
      setBatchInstallEvents([]);

      // 构建请求（Linux 中登录密码和 sudo 密码相同）
      // parallel 为 0 或未设置时，后端将自动计算动态并行度
      const payload = {
        hosts: validHosts.map(h => ({
          host: h.host.trim(),
          port: h.port || 22,
          username: h.username,
          password: h.password,
          use_sudo: values.global_use_sudo || h.use_sudo,
          sudo_pass: h.password,  // Linux 用户密码即 sudo 密码
          group: h.group || values.global_group || '',  // 单独设置的分组优先，否则使用全局分组
          install_categraf: h.install_categraf || false,  // 传递每个主机的 Categraf 安装设置
        })),
        parallel: values.parallel || 0, // 0 表示自动计算并行度
        master_host: values.master_host || 'salt',
        install_type: values.install_type || 'saltstack',
        auto_accept: values.auto_accept ?? true,
        // Categraf 监控代理安装选项（全局设置）
        install_categraf: values.install_categraf ?? false,
        n9e_host: values.n9e_host || '',
        n9e_port: values.n9e_port || '17000',
        categraf_version: values.categraf_version || '',
      };

      const resp = await saltStackAPI.batchInstallMinion(payload);
      
      if (!resp.data?.success) {
        message.error(resp.data?.message || t('saltstack.startInstallFailed'));
        setBatchInstallRunning(false);
        return;
      }

      const taskId = resp.data?.task_id;
      if (!taskId) {
        message.error(t('saltstack.noTaskIdReturned'));
        setBatchInstallRunning(false);
        return;
      }

      setBatchInstallTaskId(taskId);
      message.success(t('saltstack.installTaskCreated', { taskId }));
      
      // 立即添加一个临时任务到列表（避免等待后端返回时进度显示为0）
      const tempTask = {
        id: Date.now(),
        taskName: taskId,
        taskType: 'saltstack',
        status: 'running',
        totalHosts: validHosts.length,
        successHosts: 0,
        failedHosts: 0,
        startTime: new Date().toISOString(),
        hostResults: []
      };
      setInstallTasks(prev => [tempTask, ...prev.filter(t => !t.taskName?.includes(taskId))]);
      
      // 延迟刷新安装任务列表，让后端有时间创建记录
      setTimeout(() => loadInstallTasks(1), 2000);
      startBatchInstallSSE(taskId);
    } catch (e) {
      message.error(t('saltstack.submitInstallFailed') + ': ' + (e?.response?.data?.message || e.message));
      setBatchInstallRunning(false);
    }
  };

  // ========== SSH 测试相关函数 ==========
  
  // 打开 SSH 测试弹窗
  const openSSHTestModal = () => {
    setSSHTestVisible(true);
    setSSHTestResults([]);
    setSSHTestHosts([
      { key: Date.now(), host: '', port: 22, username: 'root', password: '' }
    ]);
  };

  // 添加 SSH 测试主机行
  const addSSHTestHostRow = () => {
    setSSHTestHosts([
      ...sshTestHosts,
      { key: Date.now(), host: '', port: 22, username: 'root', password: '' }
    ]);
  };

  // 删除 SSH 测试主机行
  const removeSSHTestHostRow = (key) => {
    if (sshTestHosts.length <= 1) {
      message.warning(t('saltstack.atLeastOneHostRequired'));
      return;
    }
    setSSHTestHosts(sshTestHosts.filter(h => h.key !== key));
  };

  // 更新 SSH 测试主机行
  const updateSSHTestHostRow = (key, field, value) => {
    setSSHTestHosts(sshTestHosts.map(h => 
      h.key === key ? { ...h, [field]: value } : h
    ));
  };

  // 执行 SSH 批量测试
  const handleSSHTest = async () => {
    const validHosts = sshTestHosts.filter(h => h.host && h.host.trim());
    if (validHosts.length === 0) {
      message.error(t('saltstack.atLeastOneHost'));
      return;
    }

    for (const h of validHosts) {
      if (!h.username || !h.password) {
        message.error(t('saltstack.missingCredentials', { host: h.host }));
        return;
      }
    }

    setSSHTestRunning(true);
    setSSHTestResults([]);

    try {
      // Linux 中登录密码和 sudo 密码相同
      const payload = {
        hosts: validHosts.map(h => ({
          host: h.host.trim(),
          port: h.port || 22,
          username: h.username,
          password: h.password,
          sudo_pass: h.password  // Linux 用户密码即 sudo 密码
        })),
        parallel: 5
      };

      const resp = await saltStackAPI.batchTestSSH(payload);
      
      if (resp.data?.success) {
        setSSHTestResults(resp.data.data?.results || []);
        message.success(t('saltstack.testCompleted', { 
          connected: resp.data.data?.connected_count, 
          total: resp.data.data?.total, 
          sudo: resp.data.data?.sudo_count 
        }));
      } else {
        message.error(resp.data?.error || t('saltstack.sshTestFailed'));
      }
    } catch (e) {
      message.error(t('saltstack.sshTestFailed') + ': ' + (e?.response?.data?.message || e.message));
    } finally {
      setSSHTestRunning(false);
    }
  };

  // ========== Minion 分组管理函数 ==========

  // 设置单个 Minion 的分组
  const handleSetMinionGroup = async (minionId, groupName) => {
    try {
      // 设置分组（groupName 为空表示清除分组）
      const resp = await saltStackAPI.setMinionGroup(minionId, groupName || '');
      if (resp.data?.success) {
        if (groupName) {
          message.success(t('saltstack.minionGroupSet', { id: minionId }));
        } else {
          message.success(t('saltstack.minionGroupRemoved', { id: minionId }));
        }
        await loadMinions(); // 刷新 minions 列表以更新分组信息
      } else {
        message.error(resp.data?.message || t('saltstack.minionGroupSetFailed'));
      }
    } catch (e) {
      message.error(t('saltstack.minionGroupSetFailed') + ': ' + (e?.response?.data?.message || e.message));
    }
  };

  // 打开新建分组弹窗
  const openCreateGroupModal = () => {
    setEditingGroup(null);
    groupForm.resetFields();
    setGroupModalVisible(true);
  };

  // 打开编辑分组弹窗
  const openEditGroupModal = (group) => {
    setEditingGroup(group);
    groupForm.setFieldsValue({
      name: group.name,
      description: group.description,
      color: group.color || 'blue',
      icon: group.icon || '',
    });
    setGroupModalVisible(true);
  };

  // 保存分组（新建或更新）
  const handleSaveGroup = async () => {
    try {
      const values = await groupForm.validateFields();
      if (editingGroup) {
        // 更新
        const resp = await saltStackAPI.updateMinionGroup(editingGroup.id, values);
        if (resp.data?.success) {
          message.success(t('saltstack.groupUpdated', '分组已更新'));
          await loadMinionGroups();
          setGroupModalVisible(false);
        } else {
          message.error(resp.data?.message || t('saltstack.groupUpdateFailed', '更新分组失败'));
        }
      } else {
        // 新建
        const resp = await saltStackAPI.createMinionGroup(values);
        if (resp.data?.success) {
          message.success(t('saltstack.groupCreated', '分组已创建'));
          await loadMinionGroups();
          setGroupModalVisible(false);
        } else {
          message.error(resp.data?.message || t('saltstack.groupCreateFailed', '创建分组失败'));
        }
      }
    } catch (e) {
      if (e.errorFields) return; // 表单验证失败
      message.error(t('saltstack.groupSaveFailed', '保存分组失败') + ': ' + (e?.response?.data?.message || e.message));
    }
  };

  // 快速创建分组（在批量安装弹窗中使用）
  const handleQuickCreateGroup = async () => {
    // 优先使用 quickGroupName（下拉框中输入的名称）
    const groupName = quickGroupName?.trim();
    if (!groupName) {
      message.warning(t('saltstack.pleaseInputGroupName', '请输入分组名称'));
      return null;
    }

    try {
      setQuickGroupCreating(true);
      
      const resp = await saltStackAPI.createMinionGroup({
        name: groupName,
        description: '',
        color: 'blue',
      });
      
      if (resp.data?.success) {
        message.success(t('saltstack.groupCreated', '分组已创建') + `: ${groupName}`);
        await loadMinionGroups(); // 刷新分组列表
        setQuickGroupName(''); // 清空输入框
        return resp.data?.data?.name || groupName; // 返回创建的分组名
      } else {
        message.error(resp.data?.message || t('saltstack.groupCreateFailed', '创建分组失败'));
      }
    } catch (e) {
      message.error(t('saltstack.groupCreateFailed', '创建分组失败') + ': ' + (e?.response?.data?.message || e.message));
    } finally {
      setQuickGroupCreating(false);
    }
    return null;
  };

  // ========== 批量安装 Categraf 相关函数 ==========

  // 打开批量安装 Categraf 弹窗（针对已接受的 Minion）
  const openBatchCategrafModal = () => {
    // 从已接受的 Minion 列表中获取主机
    const acceptedMinions = minions.filter(m => m.status === 'accepted');
    
    // 转换为弹窗所需的格式，默认全选
    const hostList = acceptedMinions.map(m => ({
      minion_id: m.minion_id,
      host: m.ip_address || m.minion_id,
      group: m.group || '',
      categraf_installed: m.categraf_installed || false,
      selected: !m.categraf_installed, // 默认选中未安装 Categraf 的
    }));
    
    setBatchCategrafVisible(true);
    setBatchCategrafEvents([]);
    setBatchCategrafRunning(false);
    setBatchCategrafHosts(hostList);
    setBatchCategrafTaskId('');
  };

  // 执行批量为 Minion 安装 Categraf（通过 Salt State）
  const handleBatchCategrafInstall = async () => {
    const selectedHosts = batchCategrafHosts.filter(h => h.selected);
    if (selectedHosts.length === 0) {
      message.warning(t('saltstack.selectAtLeastOneMinion', '请至少选择一个 Minion'));
      return;
    }

    setBatchCategrafRunning(true);
    setBatchCategrafEvents([]);

    try {
      const minionIds = selectedHosts.map(h => h.minion_id);
      
      // 调用后端 API 通过 Salt 安装 Categraf
      const resp = await saltStackAPI.installCategrafOnMinions({
        minion_ids: minionIds,
      });

      if (resp.data?.success && resp.data?.data?.task_id) {
        setBatchCategrafTaskId(resp.data.data.task_id);
        message.success(t('saltstack.categrafTaskCreated', { taskId: resp.data.data.task_id }));
        // 启动 SSE 监听
        startCategrafSSE(resp.data.data.task_id);
      } else {
        message.error(resp.data?.message || t('saltstack.categrafInstallFailed', '批量安装 Categraf 失败'));
        setBatchCategrafRunning(false);
      }
    } catch (e) {
      message.error(t('saltstack.categrafInstallFailed', '批量安装 Categraf 失败') + ': ' + (e?.response?.data?.message || e.message));
      setBatchCategrafRunning(false);
    }
  };

  // 关闭 Categraf 安装 SSE（针对 Minion）
  const closeBatchCategrafSSE = () => {
    if (batchCategrafSseRef.current) {
      batchCategrafSseRef.current.close();
      batchCategrafSseRef.current = null;
    }
  };

  // 部署节点指标采集
  const handleDeployNodeMetrics = async () => {
    try {
      const values = await deployMetricsForm.validateFields();
      setDeployMetricsLoading(true);
      
      const resp = await saltStackAPI.deployNodeMetricsState(values.target, values.interval || 3);
      
      if (resp.data?.success) {
        message.success(t('saltstack.deployMetricsSuccess', '指标采集已部署'));
        setDeployMetricsVisible(false);
        deployMetricsForm.resetFields();
        // 刷新 minion 列表以获取最新指标
        setTimeout(() => loadMinions(), 5000);
      } else {
        message.error(resp.data?.message || t('saltstack.deployMetricsFailed', '部署指标采集失败'));
      }
    } catch (e) {
      if (e.errorFields) return; // 表单验证错误
      message.error(t('saltstack.deployMetricsFailed', '部署指标采集失败') + ': ' + (e?.response?.data?.message || e.message));
    } finally {
      setDeployMetricsLoading(false);
    }
  };

  // 添加 Categraf 主机行
  const addCategrafHostRow = () => {
    setBatchCategrafHosts([
      ...batchCategrafHosts,
      { key: Date.now(), host: '', port: 22, username: 'root', password: '', use_sudo: false }
    ]);
  };

  // 更新 Categraf 主机行
  const updateCategrafHostRow = (key, field, value) => {
    setBatchCategrafHosts(batchCategrafHosts.map(h => 
      h.key === key ? { ...h, [field]: value } : h
    ));
  };

  // 删除 Categraf 主机行
  const removeCategrafHostRow = (key) => {
    if (batchCategrafHosts.length <= 1) {
      message.warning(t('saltstack.atLeastOneHost'));
      return;
    }
    setBatchCategrafHosts(batchCategrafHosts.filter(h => h.key !== key));
  };

  // 关闭 Categraf SSE
  const closeCategrafSSE = () => {
    if (batchCategrafSseRef.current) {
      batchCategrafSseRef.current.close();
      batchCategrafSseRef.current = null;
    }
  };

  // 执行批量安装 Categraf
  const handleBatchCategraf = async () => {
    try {
      const values = await batchCategrafForm.validateFields();
      
      // 验证主机列表
      const validHosts = batchCategrafHosts.filter(h => h.host && h.host.trim());
      if (validHosts.length === 0) {
        message.error(t('saltstack.atLeastOneHost'));
        return;
      }

      // 验证每个主机的凭据
      for (const host of validHosts) {
        if (!host.username || !host.password) {
          message.error(t('saltstack.missingCredentials', { host: host.host }));
          return;
        }
      }

      setBatchCategrafRunning(true);
      setBatchCategrafEvents([]);

      const payload = {
        hosts: validHosts.map(h => ({
          host: h.host.trim(),
          port: h.port || 22,
          username: h.username,
          password: h.password,
          use_sudo: h.use_sudo || false,
        })),
        parallel: values.parallel || 3,
        n9e_host: values.n9e_host || '',
        n9e_port: values.n9e_port || '17000',
        categraf_version: values.categraf_version || '',
      };

      const resp = await saltStackAPI.batchInstallCategraf(payload);
      if (resp.data?.success && resp.data?.data?.task_id) {
        message.success(t('saltstack.categrafTaskCreated', { taskId: resp.data.data.task_id }));
        // 启动 SSE 监听
        startCategrafSSE(resp.data.data.task_id);
      } else {
        message.error(resp.data?.message || t('saltstack.categrafInstallFailed', '批量安装 Categraf 失败'));
        setBatchCategrafRunning(false);
      }
    } catch (e) {
      if (e.errorFields) return;
      message.error(t('saltstack.categrafInstallFailed', '批量安装 Categraf 失败') + ': ' + (e?.response?.data?.message || e.message));
      setBatchCategrafRunning(false);
    }
  };

  // 启动 Categraf SSE
  const startCategrafSSE = (taskId) => {
    closeCategrafSSE();
    const url = saltStackAPI.getCategrafInstallStreamUrl ? 
      saltStackAPI.getCategrafInstallStreamUrl(taskId) :
      saltStackAPI.getBatchInstallStreamUrl(taskId); // 复用现有 URL
    
    const es = new EventSource(url, { withCredentials: false });
    batchCategrafSseRef.current = es;
    
    es.onmessage = (evt) => {
      try {
        const data = JSON.parse(evt.data);
        console.log('[Categraf SSE事件]', data.type, data);
        setBatchCategrafEvents((prev) => [...prev, data]);
        
        if (data.type === 'complete' || data.type === 'error' || data.type === 'closed') {
          setTimeout(() => {
            setBatchCategrafRunning(false);
            closeCategrafSSE();
          }, 500);
        }
      } catch (err) {
        console.error('[Categraf SSE] 解析消息失败:', err);
      }
    };
    
    es.onerror = (err) => {
      console.error('[Categraf SSE] 连接错误:', err);
      closeCategrafSSE();
      setBatchCategrafRunning(false);
    };
  };

  // 删除分组
  const handleDeleteGroup = async (groupId) => {
    try {
      const resp = await saltStackAPI.deleteMinionGroup(groupId);
      if (resp.data?.success) {
        message.success(t('saltstack.groupDeleted', '分组已删除'));
        await loadMinionGroups();
      } else {
        message.error(resp.data?.message || t('saltstack.groupDeleteFailed', '删除分组失败'));
      }
    } catch (e) {
      message.error(t('saltstack.groupDeleteFailed', '删除分组失败') + ': ' + (e?.response?.data?.message || e.message));
    }
  };

  // ========== Minion 删除/卸载相关函数 ==========

  // 删除 Minion（仅从 Salt Master 删除密钥，支持强制删除）
  // 优化：先在前端显示"删除中"状态，再执行实际删除，最后刷新列表
  const handleDeleteMinion = async (minionId, force = false) => {
    // 1. 立即将该 minion 标记为删除中（前端即时反馈）
    setDeletingMinionIds(prev => new Set([...prev, minionId]));
    
    // 2. 同时更新本地 minions 列表，将状态改为 deleting
    setMinions(prev => prev.map(m => 
      (m.id === minionId || m.name === minionId) 
        ? { ...m, status: 'deleting', pending_delete: true }
        : m
    ));
    
    try {
      // 3. 调用 API 执行实际删除
      const resp = await saltStackAPI.removeMinionKey(minionId, force);
      if (resp.data?.success) {
        message.success(t('saltstack.minionDeleted', { id: minionId }));
        // 4. 删除成功后刷新列表（此时该 minion 应该已从 Salt Master 移除）
        await loadMinions();
      } else {
        // 删除失败，恢复原状态
        message.error(resp.data?.error || t('saltstack.deleteMinionFailed'));
        await loadMinions(); // 刷新以恢复真实状态
      }
    } catch (e) {
      message.error(t('saltstack.deleteMinionFailed') + ': ' + (e?.response?.data?.message || e.message));
      await loadMinions(); // 刷新以恢复真实状态
    } finally {
      // 5. 从删除中列表移除
      setDeletingMinionIds(prev => {
        const newSet = new Set(prev);
        newSet.delete(minionId);
        return newSet;
      });
    }
  };

  // 忽略 IB 端口告警
  const handleIgnoreIBPort = async (minionId, portName, portNum, reason = '') => {
    try {
      await saltStackAPI.addIBPortIgnore(minionId, portName, portNum, reason);
      message.success(t('saltstack.ibPortIgnored', { port: portName }) || `已忽略端口 ${portName} 的告警`);
      // 刷新 minions 列表以更新告警状态
      await loadMinions();
    } catch (e) {
      message.error(t('saltstack.ibPortIgnoreFailed') || '忽略端口告警失败: ' + (e?.response?.data?.error || e.message));
    }
  };

  // 打开卸载 Minion 弹窗
  const openUninstallModal = (minionId) => {
    setUninstallMinionId(minionId);
    setUninstallModalVisible(true);
    uninstallForm.setFieldsValue({
      host: minionId,
      port: 22,
      username: 'root',
      password: '',
      use_sudo: false
    });
  };

  // 执行卸载 Minion
  const handleUninstallMinion = async () => {
    try {
      const values = await uninstallForm.validateFields();
      
      // Linux 中登录密码和 sudo 密码相同
      const resp = await saltStackAPI.uninstallMinion(uninstallMinionId, {
        host: values.host,
        port: values.port || 22,
        username: values.username,
        password: values.password,
        use_sudo: values.use_sudo,
        sudo_pass: values.password  // Linux 用户密码即 sudo 密码
      });

      if (resp.data?.success) {
        message.success(t('saltstack.uninstallSuccess', { id: uninstallMinionId }));
        setUninstallModalVisible(false);
        loadMinions();
      } else {
        message.error(resp.data?.error || t('saltstack.uninstallMinionFailed'));
      }
    } catch (e) {
      // 优先显示后端返回的错误信息
      const errorDetail = e?.response?.data?.error || e?.response?.data?.message || e.message;
      message.error(t('saltstack.uninstallFailed') + ': ' + errorDetail);
      console.error('Uninstall error:', e?.response?.data || e);
    }
  };

  // 批量执行命令
  const handleBatchExecution = async () => {
    try {
      const values = await batchExecForm.validateFields();
      const { targets, scriptCode } = values;
      
      if (!targets || targets.length === 0) {
        message.error(t('saltstack.selectTargetRequired'));
        return;
      }
      
      if (!scriptCode || !scriptCode.trim()) {
        message.error(t('saltstack.scriptRequired'));
        return;
      }
      
      // 生成任务ID
      const taskId = generateTaskId();
      setBatchExecTaskId(taskId);
      setBatchExecJid(null);
      setBatchExecLoading(true);
      setBatchExecResults([]);
      
      message.loading({ 
        content: t('saltstack.executingTask', '正在执行任务...') + ` [${taskId}]`, 
        key: 'batchExec',
        duration: 0 
      });
      
      // 对每个目标节点执行命令
      // 在命令前添加 TaskID 标记，方便后端精确匹配作业
      const taggedScript = `# TASK_ID=${taskId}\n${scriptCode}`;
      const targetList = targets.join(',');
      const resp = await saltStackAPI.executeCommand({
        target: targetList,
        fun: 'cmd.run',
        arg: [taggedScript],
        tgt_type: 'list',
        task_id: taskId  // 额外传递 TaskID 给后端
      });
      
      if (resp.data?.success || resp.data?.data?.success) {
        const results = resp.data.return?.[0] || resp.data.result?.return?.[0] || resp.data.data?.result?.return?.[0] || {};
        
        // 尝试从响应中获取 JID（后端会尝试查询并返回）
        const returnedJid = resp.data.jid || resp.data.data?.jid;
        if (returnedJid) {
          setBatchExecJid(returnedJid);
          addJidTaskIdMapping(returnedJid, taskId);
          console.log('[BatchExec] JID from backend:', returnedJid, '->', taskId);
        }
        
        const formattedResults = Object.entries(results).map(([minion, output]) => ({
          minion,
          output: typeof output === 'string' ? output : JSON.stringify(output, null, 2),
          success: !output?.toString()?.includes('ERROR') && !output?.toString()?.includes('error')
        }));
        setBatchExecResults(formattedResults);
        
        message.success({ 
          content: t('saltstack.executeSuccess') + ` [${taskId}]`, 
          key: 'batchExec' 
        });
        
        // 智能等待并跳转到作业历史
        // 启动轮询检查作业是否出现在历史列表中，并建立 JID-TaskID 映射
        let pollCount = 0;
        const maxPollCount = 30; // 最多轮询30次
        const pollInterval = 2000; // 每2秒轮询一次
        const maxTimeoutMs = 60000; // 最大超时时间60秒
        const pollStartTime = Date.now();
        let foundJid = returnedJid || null; // 如果后端已返回 JID，直接使用
        
        const pollJobHistory = async () => {
          pollCount++;
          const elapsedTime = Date.now() - pollStartTime;
          
          try {
            const jobsResp = await saltStackAPI.getJobs(20);
            const jobsList = jobsResp.data?.data || [];
            
            // 如果还没有找到对应的 JID，查找最新的 cmd.run 任务
            if (!foundJid) {
              // 方式1: 优先查找后端返回的 task_id 字段匹配的作业（最精确）
              const matchingJobByBackendTaskId = jobsList.find(job => job.task_id === taskId);
              
              if (matchingJobByBackendTaskId?.jid) {
                foundJid = matchingJobByBackendTaskId.jid;
                setBatchExecJid(foundJid);
                addJidTaskIdMapping(foundJid, taskId);
                console.log('[BatchExec] Found JID by backend task_id:', foundJid, '->', taskId);
              } else {
                // 方式2: 查找命令参数中带有 TaskID 标记的作业
                const taskIdMarker = `TASK_ID=${taskId}`;
              
                const matchingJobByTaskId = jobsList.find(job => {
                  const args = job.arguments || job.args || [];
                  const argStr = Array.isArray(args) ? args.join(' ') : String(args);
                  return argStr.includes(taskIdMarker);
                });
              
                if (matchingJobByTaskId?.jid) {
                  foundJid = matchingJobByTaskId.jid;
                  setBatchExecJid(foundJid);
                  addJidTaskIdMapping(foundJid, taskId);
                  console.log('[BatchExec] Found JID by TaskID marker:', foundJid, '->', taskId);
                } else {
                // 备用方案：查找最新的 cmd.run 任务（排除监控脚本任务）
                const cmdRunJobs = jobsList
                  .filter(job => job.function === 'cmd.run')
                  .filter(job => {
                    // 排除监控脚本任务
                    const args = job.arguments || job.args || [];
                    const argStr = Array.isArray(args) ? args.join(' ') : String(args);
                    
                    const isMonitoringScript = argStr.includes('get_cpu_memory') ||
                      argStr.includes('/proc/stat') ||
                      argStr.includes('/proc/meminfo') ||
                      argStr.includes('cpu_user1') ||
                      argStr.includes('nvidia-smi') ||
                      argStr.includes('ibstat');
                    
                    return !isMonitoringScript;
                  })
                  .filter(job => !getTaskIdByJid(job.jid)); // 排除已有映射的
                
                console.log('[BatchExec] Fallback: Candidate jobs:', cmdRunJobs.map(j => ({ jid: j.jid, args: j.arguments })));
                
                const matchingJob = cmdRunJobs[0];
                if (matchingJob?.jid) {
                  foundJid = matchingJob.jid;
                  setBatchExecJid(foundJid);
                  addJidTaskIdMapping(foundJid, taskId);
                  console.log('[BatchExec] Found JID by fallback:', foundJid, '->', taskId);
                }
                }
              }
            }
            
            // 关联 TaskID：优先使用后端返回的 task_id
            const jobsWithTaskId = jobsList.map(job => {
              const backendTaskId = job.task_id;
              const localTaskId = getTaskIdByJid(job.jid);
              const jobTaskId = backendTaskId || localTaskId;
              return jobTaskId ? { ...job, taskId: jobTaskId } : job;
            });
            
            // 检查是否能找到刚执行的任务
            const foundJob = foundJid 
              ? jobsList.find(job => job.jid === foundJid)
              : jobsList.length > 0;
            
            // 检查是否超时或达到最大轮询次数
            const isTimeout = elapsedTime >= maxTimeoutMs || pollCount >= maxPollCount;
            
            if (foundJob) {
              // 找到任务，跳转
              clearInterval(batchExecPollRef.current);
              batchExecPollRef.current = null;
              
              // 确保 localStorage 映射已保存
              console.log('[BatchExec] Task found, JID:', foundJid, 'TaskID:', taskId);
              console.log('[BatchExec] Current localStorage map:', Object.fromEntries(jidToTaskIdMapRef.current));
              
              // 先切换到作业历史标签
              setActiveTabKey('jobs');
              
              // 重新加载作业列表以确保 TaskID 关联生效
              await loadJobs();
              
              // 设置筛选条件
              setJobSearchTaskId(taskId);
              
              message.info({ 
                content: t('saltstack.taskFoundInHistory', '任务已记录到作业历史') + ` [${taskId}]`,
                key: 'jobSwitch'
              });
            } else if (isTimeout) {
              // 超时未获取到回调数据，通知用户
              clearInterval(batchExecPollRef.current);
              batchExecPollRef.current = null;
              
              // 切换标签并重新加载
              setActiveTabKey('jobs');
              await loadJobs();
              
              const timeoutSeconds = Math.round(elapsedTime / 1000);
              message.warning({ 
                content: t('saltstack.taskTimeoutWarning', '任务执行超时（{seconds}秒），未获取到回调数据。请在作业历史中手动查看执行结果').replace('{seconds}', timeoutSeconds) + ` [Task: ${taskId}]`,
                key: 'jobSwitch',
                duration: 8
              });
            }
          } catch (e) {
            console.error('轮询作业历史失败', e);
            const elapsedTimeOnError = Date.now() - pollStartTime;
            if (elapsedTimeOnError >= maxTimeoutMs || pollCount >= maxPollCount) {
              clearInterval(batchExecPollRef.current);
              batchExecPollRef.current = null;
              setActiveTabKey('jobs');
              await loadJobs();
              
              message.warning({ 
                content: t('saltstack.taskPollFailed', '轮询作业状态失败，请手动刷新查看执行结果'),
                key: 'jobSwitch',
                duration: 6
              });
            }
          }
        };
        
        // 启动轮询
        batchExecPollRef.current = setInterval(pollJobHistory, pollInterval);
        // 立即执行一次
        setTimeout(pollJobHistory, 500);
        
      } else {
        message.error({ 
          content: resp.data?.error || t('saltstack.executeFailed'), 
          key: 'batchExec' 
        });
      }
    } catch (e) {
      const errorDetail = e?.response?.data?.error || e?.response?.data?.message || e.message;
      message.error({ 
        content: t('saltstack.executeFailed') + ': ' + errorDetail, 
        key: 'batchExec' 
      });
      console.error('Batch execution error:', e?.response?.data || e);
    } finally {
      setBatchExecLoading(false);
    }
  };
  
  // 清理轮询定时器
  useEffect(() => {
    return () => {
      if (batchExecPollRef.current) {
        clearInterval(batchExecPollRef.current);
        batchExecPollRef.current = null;
      }
    };
  }, []);

  // 选择脚本模板时填充代码
  const handleScriptTemplateSelect = (templateId) => {
    setSelectedScriptTemplate(templateId);
    const template = scriptTemplates.find(t => t.id === templateId);
    if (template) {
      batchExecForm.setFieldsValue({
        scriptCode: template.code
      });
    }
  };

  // 格式化输出结果 - 智能压缩 JSON，突出重点
  const formatOutput = (output, compact = true) => {
    if (!output) return '';
    
    // 如果是字符串，尝试解析为 JSON
    let data = output;
    if (typeof output === 'string') {
      try {
        data = JSON.parse(output);
      } catch {
        // 不是 JSON，返回原始字符串
        return output;
      }
    }
    
    // 如果不需要压缩，返回格式化的 JSON
    if (!compact) {
      return typeof data === 'string' ? data : JSON.stringify(data, null, 2);
    }
    
    // 智能压缩 JSON
    const compressValue = (val, depth = 0) => {
      if (val === null || val === undefined) return String(val);
      if (typeof val === 'boolean' || typeof val === 'number') return String(val);
      if (typeof val === 'string') {
        // 长字符串截断
        if (val.length > 100 && depth > 0) {
          return `"${val.substring(0, 100)}..." (${val.length} chars)`;
        }
        return val;
      }
      if (Array.isArray(val)) {
        if (val.length === 0) return '[]';
        if (val.length > 5 && depth > 0) {
          const preview = val.slice(0, 3).map(v => compressValue(v, depth + 1));
          return `[${preview.join(', ')}, ... +${val.length - 3} more]`;
        }
        return val.map(v => compressValue(v, depth + 1));
      }
      if (typeof val === 'object') {
        const keys = Object.keys(val);
        if (keys.length === 0) return '{}';
        
        // Salt 特殊结构处理 - 提取关键信息
        // 处理 state 模块返回格式 (如 pkg.installed)
        if (val.__run_num__ !== undefined || val.result !== undefined || val.changes !== undefined) {
          const parts = [];
          if (val.result !== undefined) parts.push(`result: ${val.result ? '✓' : '✗'}`);
          if (val.comment) parts.push(`comment: ${val.comment.substring(0, 80)}${val.comment.length > 80 ? '...' : ''}`);
          if (val.changes && Object.keys(val.changes).length > 0) {
            const changeCount = Object.keys(val.changes).length;
            parts.push(`changes: ${changeCount} item${changeCount > 1 ? 's' : ''}`);
          }
          return parts.join(' | ');
        }
        
        // 处理 cmd.run 返回的复杂对象
        if (val.stdout !== undefined || val.stderr !== undefined || val.retcode !== undefined) {
          const parts = [];
          if (val.retcode !== undefined) parts.push(`retcode: ${val.retcode}`);
          if (val.stdout) parts.push(`stdout: ${val.stdout.substring(0, 150)}${val.stdout.length > 150 ? '...' : ''}`);
          if (val.stderr) parts.push(`stderr: ${val.stderr.substring(0, 100)}${val.stderr.length > 100 ? '...' : ''}`);
          return parts.join('\n');
        }
        
        // 深度超过 2 层时压缩显示
        if (depth > 2) {
          return `{${keys.length} keys: ${keys.slice(0, 3).join(', ')}${keys.length > 3 ? '...' : ''}}`;
        }
        
        // 递归处理
        const result = {};
        keys.forEach(k => {
          result[k] = compressValue(val[k], depth + 1);
        });
        return result;
      }
      return String(val);
    };
    
    const compressed = compressValue(data);
    return typeof compressed === 'string' ? compressed : JSON.stringify(compressed, null, 2);
  };

  // 展开/收起状态管理
  const [expandedOutputs, setExpandedOutputs] = useState({});
  const toggleOutputExpand = (key) => {
    setExpandedOutputs(prev => ({ ...prev, [key]: !prev[key] }));
  };

  useEffect(() => {
    return () => {
      closeSSE();
      closeBatchSSE();
    };
  }, []);

  const validateClientSide = (language, code) => {
    if (!code || !code.trim()) return t('saltstack.codeRequired');
    if (code.length > 20000) return t('saltstack.codeTooLong');
    // 简单引号平衡检查
    let single = 0, dbl = 0;
    for (let i = 0; i < code.length; i++) {
      const ch = code[i];
      if (ch === '\'') single ^= 1; else if (ch === '"') dbl ^= 1;
    }
    if (single || dbl) return t('saltstack.quoteUnbalanced');
    if (language === 'python') {
      const lines = code.split('\n');
      for (const ln of lines) {
        if (ln.startsWith('\t') && ln.trimStart().startsWith(' ')) return t('saltstack.pythonIndentMixed');
      }
    }
    return '';
  };

  const openExecModal = () => {
    setExecVisible(true);
    setExecEvents([]);
    setExecOpId('');
    execForm.setFieldsValue({ target: '*', language: 'bash', code: '# 例如: echo Hello\necho $(hostname)', timeout: 120 });
  };

  const handleSuggest = async () => {
    try {
      const values = await execForm.validateFields(['language', 'code']);
      const lang = values.language;
      const prompt = `Provide completion suggestions for ${lang} script executed via Salt, only provide code snippets, no explanation.`;
      await aiAPI.quickChat(prompt, 'salt-exec-suggest'); // 预留：后端应返回异步消息ID，这里仅调用以示占位
      message.info(t('saltstack.smartCompleteRequest'));
    } catch (e) {
      // 忽略
    }
  };

  const startSSE = (opId) => {
    closeSSE();
    const url = saltStackAPI.streamProgressUrl(opId);
    const es = new EventSource(url, { withCredentials: false });
    sseRef.current = es;
    es.onmessage = (evt) => {
      try {
        const data = JSON.parse(evt.data);
        console.log('[SSE事件]', data.type, data);
        setExecEvents((prev) => [...prev, data]);
        
        // 检查是否执行完成 - 只在收到 complete 或 error 事件时停止
        if (data.type === 'complete' || data.type === 'error') {
          console.log('[SSE] 收到完成事件，准备停止');
          // 延迟一点点以确保UI更新
          setTimeout(() => {
            console.log('[SSE] 设置 execRunning = false');
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
      // 自动关闭，避免内存泄漏
      closeSSE();
      setExecRunning(false);
    };
  };

  const handleExecute = async () => {
    try {
      const values = await execForm.validateFields();
      const err = validateClientSide(values.language, values.code);
      if (err) {
        message.error(err);
        return;
      }
      setExecRunning(true);
      setExecEvents([]);
      const resp = await saltStackAPI.executeCustomAsync({
        target: values.target,
        language: values.language,
        code: values.code,
        timeout: values.timeout || 120,
      });
      const opId = resp.data?.opId || resp.data?.data?.opId || resp.data?.id || resp.data?.op_id;
      if (!opId) {
        message.error(t('saltstack.noOpIdReturned'));
        setExecRunning(false);
        return;
      }
      setExecOpId(opId);
      startSSE(opId);
    } catch (e) {
      message.error(t('saltstack.submitExecFailed') + ': ' + (e?.response?.data?.error || e.message));
      setExecRunning(false);
    }
  };

  const execFooter = (
    <Space>
      <Button onClick={() => setExecVisible(false)} disabled={execRunning}>{t('saltstack.close')}</Button>
      <Button onClick={handleSuggest} disabled={execRunning}>{t('saltstack.smartComplete')}</Button>
      <Button type="primary" onClick={handleExecute} loading={execRunning}>{t('saltstack.execute')}</Button>
    </Space>
  );

  const getStatusColor = (state) => {
    switch (state?.toLowerCase()) {
      case 'up': case 'online': case 'running': return 'success';
      case 'down': case 'offline': case 'stopped': return 'error';
      case 'pending': case 'starting': return 'processing';
      default: return 'default';
    }
  };

  const getJobStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'success': case 'completed': return 'green';
      case 'failed': case 'error': return 'red';
      case 'running': case 'in_progress': return 'blue';
      case 'pending': case 'queued': return 'orange';
      default: return 'default';
    }
  };

  // 如果页面还未初始化，显示简单加载提示
  if (!pageLoaded) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" />
        <div style={{ marginTop: 16 }}>{t('saltstack.initInterface')}</div>
      </div>
    );
  }

  return (
    <Layout style={{ minHeight: '100vh', background: isDark ? '#141414' : '#f0f2f5' }}>
      <Content style={{ padding: 24, background: isDark ? '#141414' : '#f0f2f5' }}>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div style={{ 
            background: isDark ? '#1f1f1f' : '#fff', 
            padding: '16px 24px', 
            borderRadius: 8,
            border: isDark ? '1px solid #303030' : '1px solid #f0f0f0'
          }}>
            <Title level={2} style={{ color: isDark ? 'rgba(255, 255, 255, 0.85)' : 'inherit', marginBottom: 8 }}>
              <ThunderboltOutlined style={{ marginRight: 8, color: '#1890ff' }} />
              {t('saltstack.title')}
            </Title>
            <Paragraph style={{ color: isDark ? 'rgba(255, 255, 255, 0.45)' : 'rgba(0, 0, 0, 0.45)', marginBottom: 0 }}>
              {t('saltstack.subtitle')}
            </Paragraph>
          </div>

          {error && (
            <Alert 
              type="error" 
              showIcon 
              message={t('saltstack.connectionError')}
              description={
                <Space>
                  <span>{t('saltstack.connectionErrorDesc')}</span>
                  <Button size="small" onClick={loadAllData}>{t('saltstack.retry')}</Button>
                </Space>
              }
            />
          )}

          {demo && (
            <Alert 
              type="info" 
              showIcon 
              message={t('saltstack.demoMode')} 
              description={t('saltstack.demoModeDesc')}
            />
          )}

          {/* 数据加载进度提示 */}
          {(statusLoading || minionsLoading || jobsLoading) && (
            <Alert 
              type="info" 
              showIcon 
              message={t('saltstack.loadingData')} 
              description={
                <Space>
                  <span>
                    {t('saltstack.statusData')}: {statusLoading ? t('common.loading') : '✓'} | 
                    {t('saltstack.minionsData')}: {minionsLoading ? t('common.loading') : '✓'} | 
                    {t('saltstack.jobsData')}: {jobsLoading ? t('common.loading') : '✓'}
                  </span>
                </Space>
              }
            />
          )}

          {/* 状态概览 - 两行布局，每行两个卡片 */}
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={12}>
              <Card style={{ background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
                <Statistic 
                  title={t('saltstack.masterStatus')} 
                  value={status?.master_status || (statusLoading ? t('common.loading') : t('saltstack.unknown'))} 
                  prefix={<SettingOutlined />}
                  valueStyle={{ 
                    color: statusLoading ? '#999' : (status?.master_status === 'running' ? '#3f8600' : '#cf1322') 
                  }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12}>
              <Card style={{ background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
                <Statistic 
                  title={t('saltstack.apiStatus')} 
                  value={status?.api_status || (statusLoading ? t('saltstack.checking') : t('saltstack.unknown'))} 
                  prefix={<ApiOutlined />}
                  valueStyle={{ 
                    color: statusLoading ? '#999' : (status?.api_status === 'running' ? '#3f8600' : '#cf1322') 
                  }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12}>
              <Card style={{ background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
                <Statistic 
                  title={t('saltstack.onlineMinions')} 
                  value={status?.minions_up || (statusLoading ? '...' : 0)} 
                  prefix={<DesktopOutlined />}
                  valueStyle={{ color: statusLoading ? '#999' : '#3f8600' }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12}>
              <Card style={{ background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
                <Statistic 
                  title={t('saltstack.offlineMinions')} 
                  value={status?.minions_down || (statusLoading ? '...' : 0)} 
                  prefix={<ExclamationCircleOutlined />}
                  valueStyle={{ color: statusLoading ? '#999' : '#cf1322' }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
          </Row>

          {/* 详细信息选项卡 */}
          <Card style={{ background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
            <Tabs 
              activeKey={activeTabKey}
              onChange={(key) => {
                setActiveTabKey(key);
                if (key === 'install-tasks' && installTasks.length === 0 && !installTasksLoading) {
                  loadInstallTasks(1);
                }
                if (key === 'delete-tasks' && deleteTasks.length === 0 && !deleteTasksLoading) {
                  loadDeleteTasks(1);
                }
                if (key === 'jobs') {
                  loadJobs();
                }
                if (key === 'settings' && !jobConfig && !jobConfigLoading) {
                  loadJobConfig();
                }
              }}
              size="large"
            >
              <TabPane tab={t('saltstack.systemOverview')} key="overview" icon={<DatabaseOutlined />}>
                <Row gutter={16}>
                  <Col span={24}>
                    <Card 
                      title={t('saltstack.masterInfo')} 
                      size="small" 
                      loading={statusLoading} 
                      style={{ marginBottom: 16, background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}
                      extra={
                        <Space>
                          <Tooltip title={t('common.autoRefresh', '自动刷新')}>
                            <Space size="small">
                              <Switch 
                                size="small"
                                checked={autoRefreshOverview}
                                onChange={setAutoRefreshOverview}
                              />
                              <Text type="secondary" style={{ fontSize: 12 }}>
                                {autoRefreshOverview ? `${autoRefreshInterval}s` : t('common.autoRefresh', '自动刷新')}
                              </Text>
                            </Space>
                          </Tooltip>
                          <Button
                            icon={<ReloadOutlined spin={statusLoading} />}
                            onClick={() => { loadStatus(); loadMinions(true); }}
                            loading={statusLoading}
                            size="small"
                          >
                            {t('common.refresh')}
                          </Button>
                          <Button
                            icon={<DashboardOutlined />}
                            onClick={() => setDeployMetricsVisible(true)}
                            disabled={minions.filter(m => m.status === 'accepted').length === 0}
                          >
                            {t('saltstack.deployNodeMetrics', '部署指标采集')}
                          </Button>
                          <Button
                            icon={<ThunderboltOutlined />}
                            onClick={openBatchCategrafModal}
                            disabled={minions.filter(m => m.status === 'accepted').length === 0}
                          >
                            {t('saltstack.batchInstallCategraf', '批量安装 Categraf')}
                          </Button>
                        </Space>
                      }
                    >
                      <Descriptions size="small" column={4}>
                        <Descriptions.Item label={t('saltstack.saltVersion')}>
                          {status?.salt_version || (statusLoading ? t('common.loading') : t('common.unknown'))}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('saltstack.uptime')}>
                          {status?.uptime_str || status?.uptime || (statusLoading ? t('common.loading') : t('common.unknown'))}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('saltstack.configFile')}>
                          {status?.config_file || '/etc/salt/master'}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('saltstack.logLevel')}>
                          <Tag color="blue">{status?.log_level || 'info'}</Tag>
                        </Descriptions.Item>
                      </Descriptions>
                    </Card>
                  </Col>
                </Row>

                {/* 分组筛选和聚合统计 */}
                <Card 
                  size="small" 
                  style={{ marginBottom: 16, background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}
                  title={
                    <Space>
                      <TeamOutlined />
                      {t('saltstack.groupOverview', '分组概览')}
                    </Space>
                  }
                  extra={
                    <Space>
                      <Text type="secondary">{t('saltstack.filterByGroup', '按分组筛选')}:</Text>
                      <Select
                        value={overviewGroupFilter}
                        onChange={setOverviewGroupFilter}
                        style={{ width: 180 }}
                        size="small"
                      >
                        <Select.Option value="all">
                          {t('saltstack.allGroups', '全部分组')} ({minions.length})
                        </Select.Option>
                        <Select.Option value="ungrouped">
                          {t('saltstack.ungrouped', '未分组')} ({minions.filter(m => !m.group).length})
                        </Select.Option>
                        {minionGroups.map(g => (
                          <Select.Option key={g.id} value={g.name}>
                            <Tag color={g.color || 'default'} style={{ marginRight: 4 }}>{g.name}</Tag>
                            ({minions.filter(m => m.group === g.name).length})
                          </Select.Option>
                        ))}
                      </Select>
                    </Space>
                  }
                >
                  <Row gutter={[16, 16]}>
                    {/* 总体统计 */}
                    <Col xs={24} sm={12} md={6} lg={4}>
                      <Card size="small" style={{ textAlign: 'center', background: isDark ? '#162312' : '#f6ffed' }}>
                        <Statistic 
                          title={t('saltstack.totalMinions', '总节点数')} 
                          value={filteredMinions.length}
                          prefix={<DesktopOutlined />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6} lg={4}>
                      <Card size="small" style={{ textAlign: 'center', background: isDark ? '#111d2c' : '#e6f7ff' }}>
                        <Statistic 
                          title={t('saltstack.onlineMinions', '在线节点')} 
                          value={filteredMinions.filter(m => m.status?.toLowerCase() === 'up' || m.status?.toLowerCase() === 'accepted').length}
                          prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
                          valueStyle={{ color: '#52c41a' }}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6} lg={4}>
                      <Card size="small" style={{ textAlign: 'center', background: isDark ? '#2b1d11' : '#fff7e6' }}>
                        <Statistic 
                          title={t('saltstack.gpuNodes', 'GPU 节点')} 
                          value={groupStats.gpuInfo.withGpu}
                          suffix={`/ ${groupStats.gpuInfo.total} GPUs`}
                          prefix={<DashboardOutlined style={{ color: '#fa8c16' }} />}
                        />
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6} lg={4}>
                      <Card size="small" style={{ textAlign: 'center', background: isDark ? '#1a1f2e' : '#f0f5ff' }}>
                        <Statistic 
                          title={t('saltstack.npuNodes', 'NPU 节点')} 
                          value={groupStats.npuInfo.withNpu}
                          suffix={`/ ${groupStats.npuInfo.total} NPUs`}
                          prefix={<ThunderboltOutlined style={{ color: '#722ed1' }} />}
                        />
                        {groupStats.npuInfo.withNpu > 0 && (
                          <Text type="secondary" style={{ fontSize: 11 }}>
                            {Object.entries(groupStats.npuInfo.vendors).map(([vendor, count]) => (
                              <Tag key={vendor} size="small" style={{ fontSize: 10, marginTop: 4 }}>
                                {vendor}: {count}
                              </Tag>
                            ))}
                          </Text>
                        )}
                      </Card>
                    </Col>
                    <Col xs={24} sm={12} md={6} lg={4}>
                      <Card size="small" style={{ textAlign: 'center', background: groupStats.ibInfo.down > 0 ? (isDark ? '#2a1215' : '#fff1f0') : (isDark ? '#162312' : '#f6ffed') }}>
                        <Statistic 
                          title={t('saltstack.ibStatus', 'IB 网络')} 
                          value={groupStats.ibInfo.active}
                          suffix={`/ ${groupStats.ibInfo.total}`}
                          prefix={<ApiOutlined style={{ color: groupStats.ibInfo.down > 0 ? '#ff4d4f' : '#52c41a' }} />}
                          valueStyle={{ color: groupStats.ibInfo.down > 0 ? '#ff4d4f' : '#52c41a' }}
                        />
                        {groupStats.ibInfo.down > 0 && (
                          <Text type="danger" style={{ fontSize: 12 }}>
                            {groupStats.ibInfo.down} {t('saltstack.ibDown', '个离线')}
                          </Text>
                        )}
                      </Card>
                    </Col>
                  </Row>

                  {/* 按分组的统计表格 */}
                  {Object.keys(groupStats.byGroup).length > 1 && (
                    <div style={{ marginTop: 16 }}>
                      <Text strong style={{ marginBottom: 8, display: 'block' }}>
                        {t('saltstack.groupStatistics', '分组统计')}
                      </Text>
                      <Table
                        size="small"
                        dataSource={Object.entries(groupStats.byGroup).map(([name, stats]) => ({
                          key: name,
                          name,
                          ...stats,
                        }))}
                        pagination={false}
                        columns={[
                          {
                            title: t('saltstack.groupName', '分组名称'),
                            dataIndex: 'name',
                            key: 'name',
                            render: (name) => (
                              <Tag 
                                color={name === '未分组' ? 'default' : minionGroups.find(g => g.name === name)?.color || 'blue'}
                                style={{ cursor: 'pointer' }}
                                onClick={() => setOverviewGroupFilter(name === '未分组' ? 'ungrouped' : name)}
                              >
                                {name}
                              </Tag>
                            ),
                          },
                          {
                            title: t('saltstack.total', '总数'),
                            dataIndex: 'total',
                            key: 'total',
                            align: 'center',
                          },
                          {
                            title: t('saltstack.online', '在线'),
                            dataIndex: 'online',
                            key: 'online',
                            align: 'center',
                            render: (v) => <Text style={{ color: '#52c41a' }}>{v}</Text>,
                          },
                          {
                            title: t('saltstack.offline', '离线'),
                            dataIndex: 'offline',
                            key: 'offline',
                            align: 'center',
                            render: (v) => v > 0 ? <Text type="danger">{v}</Text> : '-',
                          },
                          {
                            title: 'GPU',
                            dataIndex: 'gpuCount',
                            key: 'gpuCount',
                            align: 'center',
                            render: (v) => v > 0 ? v : '-',
                          },
                          {
                            title: 'IB Active',
                            dataIndex: 'ibActive',
                            key: 'ibActive',
                            align: 'center',
                            render: (v) => v > 0 ? <Text style={{ color: '#52c41a' }}>{v}</Text> : '-',
                          },
                        ]}
                      />
                    </div>
                  )}
                </Card>
                
                {/* 可调整大小的性能指标面板 */}
                <ResizableMetricsPanel
                  title={`${t('saltstack.performanceMetrics')}${overviewGroupFilter !== 'all' ? ` - ${overviewGroupFilter === 'ungrouped' ? t('saltstack.ungrouped') : overviewGroupFilter}` : ''}`}
                  loading={statusLoading || minionsLoading}
                  minHeight={200}
                  maxHeight={600}
                  defaultHeight={350}
                  nodes={[
                    // Master 节点（仅在"全部"筛选时显示）
                    // 注意：Master 监控数据只使用后端返回的 status 中的数据，不回退使用 minion 数据
                    // Master 是容器运行的，后端通过 Docker API 获取容器指标
                    ...(overviewGroupFilter === 'all' ? [{
                      id: 'salt-master',
                      name: 'Salt Master',
                      metrics: {
                        status: status?.master_status === 'running' ? 'online' : 'offline',
                        // Master 监控数据来自后端 status API（Docker 容器指标）
                        cpu_usage: status?.cpu_usage || 0,
                        memory_usage: status?.memory_usage || 0,
                        active_connections: status?.active_connections || 0,
                        network_bandwidth: status?.network_bandwidth || 0,
                        ib_status: 'N/A',
                        roce_status: 'N/A',
                        gpu_utilization: 0,
                        gpu_memory: 0,
                      },
                    }] : []),
                    // Minion 节点 (使用筛选后的 minions)
                    ...filteredMinions.map(minion => {
                      const minionId = minion.id || minion.name;
                      // 检查是否正在删除中
                      const isDeleting = deletingMinionIds.has(minionId) || 
                                        minion.status?.toLowerCase() === 'deleting' || 
                                        minion.pending_delete;
                      // 确定显示状态
                      let displayStatus = 'offline';
                      if (isDeleting) {
                        displayStatus = 'deleting';
                      } else if (minion.status?.toLowerCase() === 'up' || minion.status?.toLowerCase() === 'online') {
                        displayStatus = 'online';
                      }
                      
                      return {
                        id: minionId,
                        name: minionId,
                        metrics: {
                          status: displayStatus,
                          // 从 minion 对象中读取数据（后端返回的是 cpu_usage_percent, memory_usage_percent）
                          cpu_usage: minion.cpu_usage_percent || minion.cpu_info?.usage || minion.cpu_usage || 0,
                          memory_usage: minion.memory_usage_percent || minion.memory_info?.usage_percent || minion.memory_usage || 0,
                          active_connections: minion.network_info?.active_connections || minion.active_connections || 0,
                          network_bandwidth: minion.network_bandwidth || 0,
                          ib_status: minion.ib_info?.active_count > 0 ? 'active' : (minion.ib_status || 'N/A'),
                          roce_status: minion.roce_info?.count > 0 ? 'active' : (minion.roce_status || 'N/A'),
                          gpu_utilization: minion.gpu_info?.utilization || minion.gpu_utilization || 0,
                          gpu_memory: minion.gpu_info?.memory_used || minion.gpu_memory || 0,
                        },
                      };
                    }),
                  ]}
                  onRefresh={loadAllData}
                />
              </TabPane>

              {/* 批量执行命令 */}
              <TabPane tab={t('saltstack.batchExecution')} key="batchExecution" icon={<CodeOutlined />}>
                <Row gutter={[16, 16]}>
                  <Col span={24}>
                    <Card 
                      title={t('saltstack.batchExecution')} 
                      size="small"
                      style={{ background: isDark ? '#1f1f1f' : '#fff' }}
                    >
                      <Form form={batchExecForm} layout="vertical">
                        <Row gutter={16}>
                          <Col span={8}>
                            <Form.Item
                              name="targetGroup"
                              label={t('saltstack.selectGroup', '选择分组')}
                            >
                              <Select
                                placeholder={t('saltstack.selectGroupPlaceholder', '按分组选择节点')}
                                allowClear
                                onChange={(groupName) => {
                                  if (groupName) {
                                    const group = minionGroups.find(g => g.name === groupName);
                                    if (group && group.minions) {
                                      batchExecForm.setFieldsValue({ targets: group.minions });
                                    }
                                  }
                                }}
                              >
                                {minionGroups.map(group => (
                                  <Select.Option key={group.name} value={group.name}>
                                    <Space>
                                      <Tag color={group.color || 'blue'} style={{ marginRight: 4 }}>
                                        {group.minions?.length || 0}
                                      </Tag>
                                      {group.name}
                                    </Space>
                                  </Select.Option>
                                ))}
                              </Select>
                            </Form.Item>
                          </Col>
                          <Col span={8}>
                            <Form.Item
                              name="targets"
                              label={
                                <Space>
                                  {t('saltstack.targetNodes')}
                                  <Button 
                                    type="link" 
                                    size="small" 
                                    onClick={() => {
                                      // 选择所有在线节点
                                      const onlineMinions = minions
                                        .filter(m => m.status?.toLowerCase() === 'up' || m.status?.toLowerCase() === 'online')
                                        .map(m => m.id);
                                      batchExecForm.setFieldsValue({ targets: onlineMinions, targetGroup: undefined });
                                    }}
                                    style={{ padding: 0 }}
                                  >
                                    {t('saltstack.selectAllOnline', '全选在线')}
                                  </Button>
                                </Space>
                              }
                              rules={[{ required: true, message: t('saltstack.selectTargetRequired') }]}
                            >
                              <Select
                                mode="multiple"
                                placeholder={t('saltstack.selectTargets')}
                                allowClear
                                showSearch
                                optionFilterProp="children"
                                style={{ width: '100%' }}
                                maxTagCount={5}
                                maxTagPlaceholder={(omittedValues) => `+${omittedValues.length} ${t('saltstack.moreNodes', '个节点')}`}
                              >
                                {minions.map(minion => (
                                  <Select.Option key={minion.id} value={minion.id}>
                                    <Space>
                                      <Badge 
                                        status={minion.status?.toLowerCase() === 'up' || minion.status?.toLowerCase() === 'online' ? 'success' : 'default'} 
                                      />
                                      {minion.id}
                                    </Space>
                                  </Select.Option>
                                ))}
                              </Select>
                            </Form.Item>
                          </Col>
                          <Col span={8}>
                            <Form.Item
                              name="scriptTemplate"
                              label={t('saltstack.scriptTemplates')}
                            >
                              <Select
                                placeholder={t('saltstack.selectScriptTemplate')}
                                allowClear
                                onChange={handleScriptTemplateSelect}
                                value={selectedScriptTemplate}
                              >
                                {scriptTemplates.map(template => (
                                  <Select.Option key={template.id} value={template.id}>
                                    {template.name}
                                  </Select.Option>
                                ))}
                              </Select>
                            </Form.Item>
                          </Col>
                        </Row>
                        <Form.Item
                          name="scriptCode"
                          label={t('saltstack.scriptCode')}
                          rules={[{ required: true, message: t('saltstack.scriptRequired') }]}
                        >
                          <Input.TextArea
                            rows={12}
                            placeholder={t('saltstack.enterScriptOrSelectTemplate')}
                            style={{ 
                              fontFamily: 'monospace',
                              fontSize: '12px',
                              background: isDark ? '#141414' : '#f5f5f5',
                              color: isDark ? '#d4d4d4' : '#333'
                            }}
                          />
                        </Form.Item>
                        <Form.Item>
                          <Space>
                            <Button
                              type="primary"
                              icon={<ThunderboltOutlined />}
                              onClick={handleBatchExecution}
                              loading={batchExecLoading}
                            >
                              {t('saltstack.executeNow')}
                            </Button>
                            <Button
                              onClick={() => {
                                batchExecForm.resetFields();
                                setSelectedScriptTemplate(null);
                                setBatchExecResults([]);
                              }}
                            >
                              {t('saltstack.clearAll')}
                            </Button>
                            <Button
                              icon={<HistoryOutlined />}
                              onClick={() => {
                                setActiveTabKey('jobs');
                                loadJobs();
                              }}
                            >
                              {t('saltstack.viewJobHistory', '查看执行历史')}
                            </Button>
                          </Space>
                        </Form.Item>
                      </Form>
                      
                      {/* 任务ID快速跳转 */}
                      <Divider style={{ margin: '12px 0' }} />
                      <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
                        <Tooltip title={t('saltstack.taskIdHelpTip', '任务ID由本页面执行时自动生成，用于快速定位任务。其他渠道执行的任务仅显示Salt原生JID。')}>
                          <Text type="secondary" style={{ cursor: 'help' }}>
                            {t('saltstack.quickJumpByTaskId', '按任务ID跳转')}:
                          </Text>
                        </Tooltip>
                        <Input.Search
                          placeholder={t('saltstack.enterTaskId', '输入任务ID (如: EXEC-20241216-...)') }
                          allowClear
                          enterButton={<><SearchOutlined /> {t('saltstack.jumpToTask', '跳转')}</>}
                          style={{ maxWidth: 400 }}
                          onSearch={(value) => {
                            if (value) {
                              setJobSearchTaskId(value);
                              setActiveTabKey('jobs');
                              loadJobs();
                            }
                          }}
                        />
                        {batchExecTaskId && (
                          <Button
                            type="link"
                            size="small"
                            onClick={() => {
                              setJobSearchTaskId(batchExecTaskId);
                              setActiveTabKey('jobs');
                              loadJobs();
                            }}
                          >
                            {t('saltstack.viewLastTask', '查看上次任务')}: <Text code style={{ fontSize: 11 }}>{batchExecTaskId}</Text>
                          </Button>
                        )}
                      </div>
                    </Card>
                  </Col>

                  {/* 执行结果 */}
                  {batchExecResults.length > 0 && (
                    <Col span={24}>
                      <Card 
                        title={
                          <Space>
                            {t('saltstack.executionResults')}
                            <Tag color="blue">{batchExecResults.length} {t('saltstack.moreNodes', '个节点')}</Tag>
                          </Space>
                        }
                        size="small"
                        style={{ background: isDark ? '#1f1f1f' : '#fff' }}
                        extra={
                          <Button 
                            type={batchExecResultSearchVisible ? 'primary' : 'default'}
                            icon={<SearchOutlined />}
                            size="small"
                            onClick={() => setBatchExecResultSearchVisible(!batchExecResultSearchVisible)}
                          >
                            {t('saltstack.search', '搜索')}
                          </Button>
                        }
                      >
                        {/* 搜索面板 */}
                        {batchExecResultSearchVisible && (
                          <div style={{ marginBottom: 12, padding: 12, background: isDark ? '#262626' : '#fafafa', borderRadius: 4 }}>
                            <Space wrap>
                              <Input
                                prefix={<SearchOutlined />}
                                placeholder={t('saltstack.searchResultPlaceholder', '搜索节点名或输出内容...')}
                                value={batchExecResultSearchText}
                                onChange={(e) => setBatchExecResultSearchText(e.target.value)}
                                style={{ width: 300 }}
                                allowClear
                              />
                              <Checkbox 
                                checked={batchExecResultSearchRegex} 
                                onChange={(e) => setBatchExecResultSearchRegex(e.target.checked)}
                              >
                                {t('saltstack.useRegex', '正则表达式')}
                              </Checkbox>
                              <Button 
                                size="small" 
                                onClick={() => { setBatchExecResultSearchText(''); setBatchExecResultSearchRegex(false); }}
                              >
                                {t('saltstack.clearSearch', '清除')}
                              </Button>
                            </Space>
                          </div>
                        )}

                        <div style={{ maxHeight: '500px', overflowY: 'auto' }}>
                          {batchExecResults
                            .filter((result) => {
                              if (!batchExecResultSearchText) return true;
                              const outputStr = result.output || '';
                              if (batchExecResultSearchRegex) {
                                try {
                                  const regex = new RegExp(batchExecResultSearchText, 'i');
                                  return regex.test(result.minion) || regex.test(outputStr);
                                } catch (e) {
                                  return result.minion.toLowerCase().includes(batchExecResultSearchText.toLowerCase()) ||
                                         outputStr.toLowerCase().includes(batchExecResultSearchText.toLowerCase());
                                }
                              }
                              return result.minion.toLowerCase().includes(batchExecResultSearchText.toLowerCase()) ||
                                     outputStr.toLowerCase().includes(batchExecResultSearchText.toLowerCase());
                            })
                            .map((result, index) => {
                              // 高亮搜索文本
                              const highlightText = (text) => {
                                if (!batchExecResultSearchText || !text) return text;
                                try {
                                  const regex = batchExecResultSearchRegex 
                                    ? new RegExp(`(${batchExecResultSearchText})`, 'gi')
                                    : new RegExp(`(${batchExecResultSearchText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
                                  return text.split(regex).map((part, i) => 
                                    regex.test(part) ? <mark key={i} style={{ background: '#ffe58f', padding: 0 }}>{part}</mark> : part
                                  );
                                } catch (e) {
                                  return text;
                                }
                              };
                              
                              const outputKey = `batch-${index}`;
                              const isExpanded = expandedOutputs[outputKey];
                              const displayOutput = isExpanded 
                                ? (result.output || t('saltstack.noOutput'))
                                : formatOutput(result.output, true);
                              
                              return (
                                <Card
                                  key={index}
                                  size="small"
                                  title={
                                    <Space>
                                      <Tag color={result.success ? 'success' : 'error'}>
                                        {highlightText(result.minion)}
                                      </Tag>
                                      {result.success ? (
                                        <CheckCircleOutlined style={{ color: '#52c41a' }} />
                                      ) : (
                                        <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
                                      )}
                                    </Space>
                                  }
                                  style={{ marginBottom: 8, background: isDark ? '#141414' : '#fafafa' }}
                                  extra={
                                    <Space size="small">
                                      <Tooltip title={isExpanded ? t('saltstack.compactView', '压缩视图') : t('saltstack.expandView', '展开视图')}>
                                        <Button
                                          type="text"
                                          icon={isExpanded ? <CompressOutlined /> : <ExpandOutlined />}
                                          size="small"
                                          onClick={() => toggleOutputExpand(outputKey)}
                                        />
                                      </Tooltip>
                                      <Tooltip title={t('saltstack.copy', '复制')}>
                                        <Button
                                          type="text"
                                          icon={<CopyOutlined />}
                                          size="small"
                                          onClick={() => {
                                            navigator.clipboard.writeText(result.output);
                                            message.success(t('saltstack.copied'));
                                          }}
                                        />
                                      </Tooltip>
                                    </Space>
                                  }
                                >
                                  <div
                                    tabIndex={0}
                                    style={{ 
                                      margin: 0, 
                                      whiteSpace: 'pre-wrap', 
                                      wordBreak: 'break-all',
                                      fontFamily: 'Consolas, Monaco, "Courier New", monospace',
                                      fontSize: '12px',
                                      lineHeight: '1.5',
                                      maxHeight: isExpanded ? '400px' : '150px',
                                      overflowY: 'auto',
                                      background: isDark ? '#0d0d0d' : '#f5f5f5',
                                      padding: '10px 12px',
                                      borderRadius: '4px',
                                      color: isDark ? '#d4d4d4' : '#333',
                                      userSelect: 'text',
                                      cursor: 'text',
                                      outline: 'none',
                                      border: `1px solid ${isDark ? '#303030' : '#e8e8e8'}`,
                                    }}
                                    onFocus={(e) => {
                                      e.target.style.borderColor = isDark ? '#177ddc' : '#40a9ff';
                                    }}
                                    onBlur={(e) => {
                                      e.target.style.borderColor = isDark ? '#303030' : '#e8e8e8';
                                    }}
                                  >
                                    {highlightText(displayOutput)}
                                  </div>
                                  {!isExpanded && result.output && result.output.length > 200 && (
                                    <div style={{ marginTop: 4, textAlign: 'right' }}>
                                      <Button 
                                        type="link" 
                                        size="small" 
                                        onClick={() => toggleOutputExpand(outputKey)}
                                        style={{ padding: 0, height: 'auto' }}
                                      >
                                        {t('saltstack.viewFullOutput', '查看完整输出')} ({result.output.length} {t('saltstack.chars', '字符')})
                                      </Button>
                                    </div>
                                  )}
                                </Card>
                              );
                            })}
                          {batchExecResultSearchText && batchExecResults.filter((result) => {
                            const outputStr = result.output || '';
                            if (batchExecResultSearchRegex) {
                              try {
                                const regex = new RegExp(batchExecResultSearchText, 'i');
                                return regex.test(result.minion) || regex.test(outputStr);
                              } catch (e) {
                                return result.minion.toLowerCase().includes(batchExecResultSearchText.toLowerCase()) ||
                                       outputStr.toLowerCase().includes(batchExecResultSearchText.toLowerCase());
                              }
                            }
                            return result.minion.toLowerCase().includes(batchExecResultSearchText.toLowerCase()) ||
                                   outputStr.toLowerCase().includes(batchExecResultSearchText.toLowerCase());
                          }).length === 0 && (
                            <Empty description={t('saltstack.noMatchingResults', '未找到匹配结果')} />
                          )}
                        </div>
                      </Card>
                    </Col>
                  )}
                </Row>
              </TabPane>

              <TabPane tab={t('saltstack.groupManagement', '分组管理')} key="groups" icon={<TeamOutlined />}>
                <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text type="secondary">
                    {t('saltstack.totalGroups', { count: minionGroups.length })}
                  </Text>
                  <Space>
                    <Button 
                      icon={<ReloadOutlined />} 
                      onClick={loadMinionGroups}
                      loading={groupsLoading}
                    >
                      {t('common.refresh')}
                    </Button>
                    <Button 
                      type="primary" 
                      icon={<PlusOutlined />} 
                      onClick={openCreateGroupModal}
                    >
                      {t('saltstack.createGroup', '创建分组')}
                    </Button>
                  </Space>
                </div>
                <Table
                  dataSource={minionGroups}
                  rowKey="id"
                  loading={groupsLoading}
                  size="small"
                  pagination={{
                    showSizeChanger: true,
                    showTotal: (total) => t('common.total', { count: total }),
                    defaultPageSize: 10,
                    pageSizeOptions: ['10', '20', '50'],
                  }}
                  columns={[
                    {
                      title: t('saltstack.groupName', '分组名称'),
                      dataIndex: 'name',
                      key: 'name',
                      width: 180,
                      render: (name, record) => (
                        <Tag color={record.color || 'default'} icon={record.icon ? <TeamOutlined /> : null}>
                          {name}
                        </Tag>
                      ),
                    },
                    {
                      title: t('saltstack.groupDescription', '描述'),
                      dataIndex: 'description',
                      key: 'description',
                      ellipsis: true,
                    },
                    {
                      title: t('saltstack.groupColor', '颜色'),
                      dataIndex: 'color',
                      key: 'color',
                      width: 100,
                      render: (color) => <Tag color={color || 'default'}>{color || 'default'}</Tag>,
                    },
                    {
                      title: t('common.createdAt', '创建时间'),
                      dataIndex: 'created_at',
                      key: 'created_at',
                      width: 180,
                      render: (time) => time ? new Date(time).toLocaleString('zh-CN') : '-',
                    },
                    {
                      title: t('common.actions', '操作'),
                      key: 'actions',
                      width: 150,
                      render: (_, record) => (
                        <Space>
                          <Tooltip title={t('common.edit', '编辑')}>
                            <Button 
                              type="link" 
                              size="small" 
                              icon={<EditOutlined />} 
                              onClick={() => openEditGroupModal(record)}
                            />
                          </Tooltip>
                          <Popconfirm
                            title={t('saltstack.confirmDeleteGroup', '确定要删除此分组吗？')}
                            description={t('saltstack.deleteGroupHint', '删除分组不会影响已分配的 Minion')}
                            onConfirm={() => handleDeleteGroup(record.id)}
                            okText={t('common.confirm', '确定')}
                            cancelText={t('common.cancel', '取消')}
                          >
                            <Tooltip title={t('common.delete', '删除')}>
                              <Button 
                                type="link" 
                                size="small" 
                                danger
                                icon={<DeleteOutlined />} 
                              />
                            </Tooltip>
                          </Popconfirm>
                        </Space>
                      ),
                    },
                  ]}
                />
                {minionGroups.length === 0 && !groupsLoading && (
                  <div style={{ textAlign: 'center', padding: '40px 0' }}>
                    <Text type="secondary">{t('saltstack.noGroups', '暂无分组')}</Text>
                    <div style={{ marginTop: 16 }}>
                      <Button 
                        type="primary" 
                        icon={<PlusOutlined />} 
                        onClick={openCreateGroupModal}
                      >
                        {t('saltstack.createGroup', '创建分组')}
                      </Button>
                    </div>
                  </div>
                )}
              </TabPane>

              <TabPane tab={t('saltstack.minionsManagement')} key="minions" icon={<DesktopOutlined />}>
                {/* 自动刷新控制头部 */}
                <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'flex-end', alignItems: 'center' }}>
                  <Space>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {t('common.autoRefresh', '自动刷新')}:
                    </Text>
                    <Switch 
                      size="small"
                      checked={autoRefreshMinions}
                      onChange={setAutoRefreshMinions}
                    />
                    {autoRefreshMinions && (
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        ({autoRefreshInterval}s)
                      </Text>
                    )}
                  </Space>
                </div>
                <MinionsTable
                  minions={minions}
                  loading={minionsLoading}
                  deletingMinionIds={deletingMinionIds}
                  onRefresh={() => loadMinions(true)}
                  onDelete={handleDeleteMinion}
                  groups={minionGroups}
                  selectedGroup={selectedGroup}
                  onGroupChange={setSelectedGroup}
                  onSetGroup={handleSetMinionGroup}
                  ibAlerts={ibAlerts}
                  onIgnoreIBPort={handleIgnoreIBPort}
                  onBatchDelete={async (minionIds, options = {}) => {
                    // options 可以包含: { force, uninstall, ssh_username, ssh_password, ssh_port, use_sudo }
                    const { force = false, ...restOptions } = options;
                    
                    // 1. 立即将所有待删除的 minion 标记为删除中（前端即时反馈）
                    setDeletingMinionIds(prev => new Set([...prev, ...minionIds]));
                    
                    // 2. 同时更新本地 minions 列表，将状态改为 deleting
                    setMinions(prev => prev.map(m => 
                      minionIds.includes(m.id) || minionIds.includes(m.name)
                        ? { ...m, status: 'deleting', pending_delete: true }
                        : m
                    ));
                    
                    try {
                      // 3. 调用 API 执行批量删除（传递完整的 options）
                      const resp = await saltStackAPI.batchRemoveMinionKeys(minionIds, { force, ...restOptions });
                      if (resp.data?.success) {
                        const uninstallMsg = restOptions.uninstall ? t('saltstack.batchUninstallSuccess', '（含卸载）') : '';
                        message.success(t('saltstack.batchDeleteSuccess', { count: resp.data?.success_count || minionIds.length }) + uninstallMsg);
                      } else if (resp.data?.failed_count > 0) {
                        message.warning(t('saltstack.batchDeletePartial', { 
                          success: resp.data?.success_count || 0, 
                          failed: resp.data?.failed_count || 0 
                        }));
                      }
                      // 4. 删除完成后刷新列表
                      await loadMinions();
                    } catch (e) {
                      message.error(t('saltstack.batchDeleteFailed') + ': ' + (e?.response?.data?.message || e.message));
                      await loadMinions(); // 刷新以恢复真实状态
                    } finally {
                      // 5. 从删除中列表移除
                      setDeletingMinionIds(prev => {
                        const newSet = new Set(prev);
                        minionIds.forEach(id => newSet.delete(id));
                        return newSet;
                      });
                    }
                  }}
                  onUninstall={openUninstallModal}
                />
              </TabPane>

              <TabPane tab={t('saltstack.jobsHistory')} key="jobs" icon={<PlayCircleOutlined />}>
                {jobsLoading ? (
                  <div style={{ textAlign: 'center', padding: '60px 0' }}>
                    <Spin size="large" />
                    <div style={{ marginTop: 16 }}>{t('common.loading')}...</div>
                  </div>
                ) : (
                  <>
                    <div style={{ marginBottom: 16 }}>
                      <Row gutter={[16, 8]} align="middle">
                        <Col flex="auto">
                          <Space wrap>
                            <Text type="secondary">{t('saltstack.total', { count: jobs.length })}</Text>
                            <Input.Search
                              placeholder={t('saltstack.searchByTaskIdOrFunction', '搜索任务ID/函数/目标')}
                              allowClear
                              value={jobSearchText}
                              onChange={(e) => setJobSearchText(e.target.value)}
                              style={{ width: 280 }}
                              onSearch={(v) => setJobSearchText(v)}
                            />
                            {jobSearchTaskId && (
                              <Tag 
                                closable 
                                onClose={() => setJobSearchTaskId('')}
                                color="blue"
                              >
                                {t('saltstack.filteringByTaskId', '筛选任务ID')}: {jobSearchTaskId}
                              </Tag>
                            )}
                          </Space>
                        </Col>
                        <Col>
                          <Space>
                            {(jobSearchTaskId || jobSearchText) && (
                              <Button 
                                size="small"
                                onClick={() => { setJobSearchTaskId(''); setJobSearchText(''); }}
                              >
                                {t('saltstack.clearFilter', '清除过滤')}
                              </Button>
                            )}
                            <Button 
                              icon={<ReloadOutlined />} 
                              onClick={loadJobs}
                              loading={jobsLoading}
                            >
                              {t('common.refresh')}
                            </Button>
                          </Space>
                        </Col>
                      </Row>
                    </div>
                    <Table
                      dataSource={jobs.filter(job => {
                        // 如果有任务ID筛选
                        if (jobSearchTaskId) {
                          // 精确匹配 taskId 或者 jid 包含搜索关键字
                          const taskIdMatch = job.taskId === jobSearchTaskId;
                          const jidMatch = job.jid?.includes(jobSearchTaskId);
                          if (!taskIdMatch && !jidMatch) {
                            return false;
                          }
                        }
                        // 通用搜索
                        if (jobSearchText) {
                          const searchLower = jobSearchText.toLowerCase();
                          return (
                            job.taskId?.toLowerCase()?.includes(searchLower) ||
                            job.jid?.toLowerCase()?.includes(searchLower) ||
                            job.function?.toLowerCase()?.includes(searchLower) ||
                            job.target?.toLowerCase()?.includes(searchLower) ||
                            job.user?.toLowerCase()?.includes(searchLower)
                          );
                        }
                        return true;
                      })}
                      rowKey={(record, index) => record.jid || record.id || index}
                      loading={jobsLoading}
                      size="small"
                      pagination={{
                        showSizeChanger: true,
                        showTotal: (total) => t('common.total', { count: total }),
                        defaultPageSize: 10,
                        pageSizeOptions: ['10', '20', '50'],
                      }}
                      columns={[
                        {
                          title: t('saltstack.taskId', '任务ID'),
                          dataIndex: 'taskId',
                          key: 'taskId',
                          width: 220,
                          ellipsis: true,
                          render: (taskId, record) => taskId ? (
                            <Tooltip title={t('saltstack.clickToCopy', '点击复制')}>
                              <Text 
                                code 
                                copyable={{ text: taskId }}
                                style={{ fontSize: 11, cursor: 'pointer' }}
                              >
                                {taskId}
                              </Text>
                            </Tooltip>
                          ) : (
                            <Tooltip title={t('saltstack.taskIdNotFromBatchExec', '此任务非从“批量执行”页面发起，显示Salt原生JID')}>
                              <Text type="secondary" style={{ fontSize: 11, cursor: 'help' }}>
                                JID: {record.jid?.slice(-12) || '-'}
                              </Text>
                            </Tooltip>
                          ),
                        },
                        {
                          title: t('saltstack.function'),
                          dataIndex: 'function',
                          key: 'function',
                          width: 180,
                          ellipsis: true,
                          render: (func, record) => (
                            <Text strong>{func || record.command || '-'}</Text>
                          ),
                        },
                        {
                          title: t('common.status'),
                          dataIndex: 'status',
                          key: 'status',
                          width: 100,
                          render: (status) => (
                            <Tag color={getJobStatusColor(status)}>
                              {status || t('saltstack.unknown')}
                            </Tag>
                          ),
                        },
                        {
                          title: t('saltstack.target'),
                          dataIndex: 'target',
                          key: 'target',
                          width: 150,
                          ellipsis: true,
                          render: (target) => target || t('saltstack.allNodes'),
                        },
                        {
                          title: t('saltstack.user'),
                          dataIndex: 'user',
                          key: 'user',
                          width: 100,
                          render: (user) => user || 'root',
                        },
                        {
                          title: t('saltstack.duration'),
                          dataIndex: 'duration',
                          key: 'duration',
                          width: 100,
                          render: (duration) => duration || '-',
                        },
                        {
                          title: t('saltstack.returnCode'),
                          dataIndex: 'return_code',
                          key: 'return_code',
                          width: 100,
                          render: (code) => (
                            <Tag color={code === 0 ? 'green' : code !== undefined ? 'red' : 'default'}>
                              {code ?? '-'}
                            </Tag>
                          ),
                        },
                        {
                          title: t('common.time'),
                          dataIndex: 'timestamp',
                          key: 'timestamp',
                          width: 180,
                          render: (time, record) => (
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              {time || record.start_time || '-'}
                            </Text>
                          ),
                        },
                        {
                          title: t('common.actions', '操作'),
                          key: 'action',
                          width: 100,
                          fixed: 'right',
                          render: (_, record) => (
                            <Button
                              type="link"
                              size="small"
                              icon={<EyeOutlined />}
                              onClick={() => viewJobDetail(record.jid)}
                            >
                              {t('saltstack.viewResult', '查看结果')}
                            </Button>
                          ),
                        },
                      ]}
                      expandable={{
                        expandedRowRender: (job) => job.result ? (
                          <div style={{ padding: '8px 0' }}>
                            <Text type="secondary">{t('saltstack.result')}:</Text>
                            <Paragraph 
                              code 
                              style={{ 
                                marginTop: 4, 
                                marginBottom: 0, 
                                maxHeight: 200, 
                                overflow: 'auto' 
                              }}
                            >
                              {typeof job.result === 'string' ? job.result : JSON.stringify(job.result, null, 2)}
                            </Paragraph>
                          </div>
                        ) : null,
                        rowExpandable: (job) => !!job.result,
                      }}
                      locale={{
                        emptyText: t('saltstack.noJobs'),
                      }}
                    />
                  </>
                )}
              </TabPane>

              <TabPane tab={t('saltstack.installTasksHistory')} key="install-tasks" icon={<HistoryOutlined />}>
                {installTasksLoading && installTasks.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: '60px 0' }}>
                    <Spin size="large" />
                    <div style={{ marginTop: 16 }}>{t('common.loading')}...</div>
                  </div>
                ) : (
                  <>
                    <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Text type="secondary">{t('saltstack.total', { count: installTasksTotal })}</Text>
                      <Space>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          {t('common.autoRefresh', '自动刷新')}:
                        </Text>
                        <Switch 
                          size="small"
                          checked={autoRefreshTasks}
                          onChange={setAutoRefreshTasks}
                        />
                        {autoRefreshTasks && (
                          <Text type="secondary" style={{ fontSize: 12 }}>
                            ({autoRefreshInterval}s)
                          </Text>
                        )}
                        <Button 
                          icon={<ReloadOutlined />} 
                          onClick={() => loadInstallTasks(1)} 
                          loading={installTasksLoading}
                        >
                          {t('common.refresh')}
                        </Button>
                      </Space>
                    </div>
                    <Table
                      dataSource={installTasks}
                      rowKey="id"
                      loading={installTasksLoading}
                      size="small"
                      pagination={{
                        current: installTasksPage.current,
                        pageSize: installTasksPage.pageSize,
                        total: installTasksTotal,
                        showSizeChanger: true,
                        showTotal: (total) => t('saltstack.total', { count: total }),
                        onChange: (page, pageSize) => loadInstallTasks(page, pageSize),
                      }}
                      expandable={{
                        expandedRowKeys: expandedTaskId ? [expandedTaskId] : [],
                        onExpand: (expanded, record) => {
                          setExpandedTaskId(expanded ? record.id : null);
                        },
                        expandedRowRender: (record) => (
                          <div style={{ padding: '8px 0' }}>
                            <Table
                              dataSource={record.hostResults || []}
                              rowKey="id"
                              size="small"
                              pagination={false}
                              columns={[
                                {
                                  title: t('saltstack.hostAddress'),
                                  dataIndex: 'host',
                                  key: 'host',
                                  width: 150,
                                  render: (host, row) => (
                                    <Tooltip title={`${row.user || 'root'}@${host}:${row.port || 22}`}>
                                      <Text code>{host}</Text>
                                    </Tooltip>
                                  ),
                                },
                                {
                                  title: t('saltstack.taskStatus'),
                                  dataIndex: 'status',
                                  key: 'status',
                                  width: 100,
                                  render: (status) => (
                                    <Tag 
                                      color={status === 'success' ? 'green' : 'red'}
                                      icon={status === 'success' ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}
                                    >
                                      {status === 'success' ? t('saltstack.success') : t('saltstack.failed')}
                                    </Tag>
                                  ),
                                },
                                {
                                  title: t('saltstack.duration'),
                                  dataIndex: 'duration',
                                  key: 'duration',
                                  width: 100,
                                  render: (duration) => {
                                    if (!duration) return '-';
                                    if (duration < 1000) return `${duration}ms`;
                                    const seconds = Math.floor(duration / 1000);
                                    if (seconds < 60) return `${seconds}s`;
                                    const minutes = Math.floor(seconds / 60);
                                    const remainingSeconds = seconds % 60;
                                    return `${minutes}m ${remainingSeconds}s`;
                                  },
                                },
                                {
                                  title: t('saltstack.error'),
                                  dataIndex: 'error',
                                  key: 'error',
                                  ellipsis: true,
                                  render: (error) => error ? (
                                    <Tooltip title={error}>
                                      <Text type="danger" ellipsis>{error}</Text>
                                    </Tooltip>
                                  ) : '-',
                                },
                              ]}
                            />
                          </div>
                        ),
                      }}
                      columns={[
                        {
                          title: t('saltstack.taskName'),
                          dataIndex: 'taskName',
                          key: 'taskName',
                          width: 200,
                          ellipsis: true,
                          render: (name, record) => (
                            <Space>
                              <Text strong>{name || `${t('saltstack.taskName')} #${record.id}`}</Text>
                            </Space>
                          ),
                        },
                        {
                          title: t('saltstack.taskStatus'),
                          dataIndex: 'status',
                          key: 'status',
                          width: 120,
                          filters: [
                            { text: t('saltstack.taskPending'), value: 'pending' },
                            { text: t('saltstack.taskRunning'), value: 'running' },
                            { text: t('saltstack.taskCompleted'), value: 'completed' },
                            { text: t('saltstack.taskFailed'), value: 'failed' },
                          ],
                          onFilter: (value, record) => record.status === value,
                          render: (status) => {
                            const statusConfig = {
                              pending: { color: 'default', icon: <ClockCircleOutlined />, text: t('saltstack.taskPending') },
                              running: { color: 'processing', icon: <SyncOutlined spin />, text: t('saltstack.taskRunning') },
                              completed: { color: 'success', icon: <CheckCircleOutlined />, text: t('saltstack.taskCompleted') },
                              failed: { color: 'error', icon: <ExclamationCircleOutlined />, text: t('saltstack.taskFailed') },
                            };
                            const config = statusConfig[status] || { color: 'default', icon: null, text: status };
                            return (
                              <Tag color={config.color} icon={config.icon}>
                                {config.text}
                              </Tag>
                            );
                          },
                        },
                        {
                          title: t('saltstack.progress'),
                          key: 'progress',
                          width: 180,
                          render: (_, record) => {
                            const total = record.totalHosts || 0;
                            const success = record.successHosts || 0;
                            const failed = record.failedHosts || 0;
                            const completed = success + failed;
                            const percent = total > 0 ? Math.round((completed / total) * 100) : 0;
                            
                            if (record.status === 'running') {
                              return (
                                <Space direction="vertical" size={0} style={{ width: '100%' }}>
                                  <Progress percent={percent} size="small" status="active" />
                                  <Text type="secondary" style={{ fontSize: 12 }}>
                                    {completed}/{total} {t('saltstack.hosts')}
                                  </Text>
                                </Space>
                              );
                            }
                            
                            return (
                              <Space>
                                <Tag color="green">{success} {t('saltstack.successCount')}</Tag>
                                {failed > 0 && <Tag color="red">{failed} {t('saltstack.failedCount')}</Tag>}
                                <Text type="secondary">/ {total}</Text>
                              </Space>
                            );
                          },
                        },
                        {
                          title: t('saltstack.startTime'),
                          dataIndex: 'startTime',
                          key: 'startTime',
                          width: 170,
                          sorter: (a, b) => new Date(a.startTime) - new Date(b.startTime),
                          defaultSortOrder: 'descend',
                          render: (time) => time ? new Date(time).toLocaleString('zh-CN') : '-',
                        },
                        {
                          title: t('saltstack.duration'),
                          dataIndex: 'duration',
                          key: 'duration',
                          width: 100,
                          render: (duration, record) => {
                            if (record.status === 'running') {
                              return <Tag color="processing">{t('saltstack.inProgress')}</Tag>;
                            }
                            if (!duration) return '-';
                            if (duration < 60) return `${duration}s`;
                            const minutes = Math.floor(duration / 60);
                            const seconds = duration % 60;
                            return `${minutes}m ${seconds}s`;
                          },
                        },
                      ]}
                    />
                    {installTasks.length === 0 && !installTasksLoading && (
                      <div style={{ textAlign: 'center', padding: '40px 0' }}>
                        <Text type="secondary">{t('saltstack.noInstallTasks')}</Text>
                        <div style={{ marginTop: 16 }}>
                          <Button 
                            type="primary" 
                            icon={<CloudUploadOutlined />} 
                            onClick={openBatchInstallModal}
                          >
                            {t('saltstack.startBatchInstall')}
                          </Button>
                        </div>
                      </div>
                    )}
                  </>
                )}
              </TabPane>

              <TabPane tab={t('saltstack.deleteTasksHistory', '删除任务历史')} key="delete-tasks" icon={<DeleteOutlined />}>
                {deleteTasksLoading && deleteTasks.length === 0 ? (
                  <div style={{ textAlign: 'center', padding: '60px 0' }}>
                    <Spin size="large" />
                    <div style={{ marginTop: 16 }}>{t('common.loading')}...</div>
                  </div>
                ) : (
                  <>
                    <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Text type="secondary">{t('saltstack.total', { count: deleteTasksTotal })}</Text>
                      <Space>
                        <Button 
                          icon={<ReloadOutlined />} 
                          onClick={() => loadDeleteTasks()} 
                          loading={deleteTasksLoading}
                        >
                          {t('common.refresh')}
                        </Button>
                      </Space>
                    </div>
                    <Table
                      dataSource={deleteTasks}
                      rowKey="id"
                      loading={deleteTasksLoading}
                      size="small"
                      pagination={{
                        pageSize: 10,
                        showSizeChanger: true,
                        showTotal: (total) => t('saltstack.total', { count: total }),
                      }}
                      expandable={{
                        expandedRowKeys: expandedDeleteTaskId ? [expandedDeleteTaskId] : [],
                        onExpand: (expanded, record) => {
                          setExpandedDeleteTaskId(expanded ? record.id : null);
                          if (expanded && record.minion_id) {
                            loadDeleteTaskLogs(record.minion_id);
                          }
                        },
                        expandedRowRender: (record) => (
                          <div style={{ padding: '8px 0' }}>
                            <Table
                              dataSource={deleteTaskLogs[record.minion_id] || []}
                              rowKey="id"
                              size="small"
                              pagination={false}
                              columns={[
                                {
                                  title: t('saltstack.step', '步骤'),
                                  dataIndex: 'step',
                                  key: 'step',
                                  width: 120,
                                },
                                {
                                  title: t('saltstack.taskStatus'),
                                  dataIndex: 'status',
                                  key: 'status',
                                  width: 80,
                                  render: (status) => (
                                    <Tag 
                                      color={status === 'success' ? 'green' : status === 'failed' ? 'red' : 'blue'}
                                      icon={status === 'success' ? <CheckCircleOutlined /> : status === 'failed' ? <ExclamationCircleOutlined /> : <SyncOutlined spin />}
                                    >
                                      {status === 'success' ? t('saltstack.success') : status === 'failed' ? t('saltstack.failed') : t('saltstack.inProgress')}
                                    </Tag>
                                  ),
                                },
                                {
                                  title: t('saltstack.message', '消息'),
                                  dataIndex: 'message',
                                  key: 'message',
                                },
                                {
                                  title: t('saltstack.output', '输出'),
                                  dataIndex: 'output',
                                  key: 'output',
                                  ellipsis: true,
                                },
                                {
                                  title: t('saltstack.error', '错误'),
                                  dataIndex: 'error',
                                  key: 'error',
                                  ellipsis: true,
                                  render: (error) => error ? <Text type="danger">{error}</Text> : '-',
                                },
                                {
                                  title: t('saltstack.time', '时间'),
                                  dataIndex: 'created_at',
                                  key: 'created_at',
                                  width: 170,
                                  render: (time) => time ? new Date(time).toLocaleString('zh-CN') : '-',
                                },
                              ]}
                              locale={{
                                emptyText: t('saltstack.noLogs', '暂无日志'),
                              }}
                            />
                          </div>
                        ),
                      }}
                      columns={[
                        {
                          title: t('saltstack.minionId'),
                          dataIndex: 'minion_id',
                          key: 'minion_id',
                          render: (minionId) => <Text code>{minionId}</Text>,
                        },
                        {
                          title: t('saltstack.taskStatus'),
                          dataIndex: 'status',
                          key: 'status',
                          width: 120,
                          filters: [
                            { text: t('saltstack.pending', '待处理'), value: 'pending' },
                            { text: t('saltstack.deleting', '删除中'), value: 'deleting' },
                            { text: t('saltstack.completed', '已完成'), value: 'completed' },
                            { text: t('saltstack.failed'), value: 'failed' },
                            { text: t('saltstack.cancelled', '已取消'), value: 'cancelled' },
                          ],
                          onFilter: (value, record) => record.status === value,
                          render: (status) => {
                            const statusConfig = {
                              pending: { color: 'orange', text: t('saltstack.pending', '待处理') },
                              deleting: { color: 'processing', text: t('saltstack.deleting', '删除中') },
                              completed: { color: 'green', text: t('saltstack.completed', '已完成') },
                              failed: { color: 'red', text: t('saltstack.failed') },
                              cancelled: { color: 'default', text: t('saltstack.cancelled', '已取消') },
                            };
                            const config = statusConfig[status] || { color: 'default', text: status };
                            return <Tag color={config.color}>{config.text}</Tag>;
                          },
                        },
                        {
                          title: t('saltstack.uninstall', '远程卸载'),
                          dataIndex: 'uninstall',
                          key: 'uninstall',
                          width: 100,
                          render: (uninstall) => uninstall ? <Tag color="blue">{t('common.yes', '是')}</Tag> : <Tag>{t('common.no', '否')}</Tag>,
                        },
                        {
                          title: t('saltstack.force', '强制删除'),
                          dataIndex: 'force',
                          key: 'force',
                          width: 100,
                          render: (force) => force ? <Tag color="orange">{t('common.yes', '是')}</Tag> : <Tag>{t('common.no', '否')}</Tag>,
                        },
                        {
                          title: t('saltstack.retryCount', '重试次数'),
                          dataIndex: 'retry_count',
                          key: 'retry_count',
                          width: 100,
                          render: (count, record) => `${count} / ${record.max_retries}`,
                        },
                        {
                          title: t('saltstack.createdAt', '创建时间'),
                          dataIndex: 'created_at',
                          key: 'created_at',
                          width: 170,
                          sorter: (a, b) => new Date(a.created_at) - new Date(b.created_at),
                          defaultSortOrder: 'descend',
                          render: (time) => time ? new Date(time).toLocaleString('zh-CN') : '-',
                        },
                        {
                          title: t('saltstack.duration'),
                          dataIndex: 'duration',
                          key: 'duration',
                          width: 100,
                          render: (duration, record) => {
                            if (record.status === 'deleting' || record.status === 'pending') {
                              return <Tag color="processing">{t('saltstack.inProgress')}</Tag>;
                            }
                            if (!duration) return '-';
                            if (duration < 1000) return `${duration}ms`;
                            const seconds = Math.floor(duration / 1000);
                            if (seconds < 60) return `${seconds}s`;
                            const minutes = Math.floor(seconds / 60);
                            const secs = seconds % 60;
                            return `${minutes}m ${secs}s`;
                          },
                        },
                        {
                          title: t('saltstack.error', '错误'),
                          dataIndex: 'error_message',
                          key: 'error_message',
                          ellipsis: true,
                          render: (error) => error ? (
                            <Tooltip title={error}>
                              <Text type="danger" ellipsis style={{ maxWidth: 200 }}>{error}</Text>
                            </Tooltip>
                          ) : '-',
                        },
                        {
                          title: t('common.actions', '操作'),
                          key: 'actions',
                          width: 120,
                          render: (_, record) => (
                            <Space>
                              {record.status === 'failed' && record.retry_count < record.max_retries && (
                                <Tooltip title={t('saltstack.retryDelete', '重试删除')}>
                                  <Button 
                                    type="link" 
                                    size="small" 
                                    icon={<ReloadOutlined />}
                                    onClick={async () => {
                                      try {
                                        await saltStackAPI.retryDeleteTask(record.minion_id);
                                        message.success(t('saltstack.retrySuccess', '重试任务已提交'));
                                        loadDeleteTasks();
                                      } catch (e) {
                                        message.error(e.response?.data?.error || t('common.error'));
                                      }
                                    }}
                                  />
                                </Tooltip>
                              )}
                              {(record.status === 'pending' || record.status === 'failed') && (
                                <Tooltip title={t('saltstack.cancelDelete', '取消删除')}>
                                  <Button 
                                    type="link" 
                                    size="small" 
                                    danger
                                    icon={<CloseCircleOutlined />}
                                    onClick={async () => {
                                      try {
                                        await saltStackAPI.cancelDeleteTask(record.minion_id);
                                        message.success(t('saltstack.cancelSuccess', '取消成功'));
                                        loadDeleteTasks();
                                      } catch (e) {
                                        message.error(e.response?.data?.error || t('common.error'));
                                      }
                                    }}
                                  />
                                </Tooltip>
                              )}
                            </Space>
                          ),
                        },
                      ]}
                      locale={{
                        emptyText: t('saltstack.noDeleteTasks', '暂无删除任务记录'),
                      }}
                    />
                  </>
                )}
              </TabPane>

              {/* 设置 Tab */}
              <TabPane tab={t('saltstack.settings', '设置')} key="settings" icon={<SettingOutlined />}>
                <Spin spinning={jobConfigLoading}>
                  <Row gutter={[16, 16]}>
                    {/* 作业统计 */}
                    <Col span={24}>
                      <Card title={t('saltstack.jobStatistics', '作业统计')} size="small">
                        {jobStats ? (
                          <Row gutter={16}>
                            <Col span={6}>
                              <Statistic title={t('saltstack.totalJobs', '总作业数')} value={jobStats.total_jobs || 0} />
                            </Col>
                            <Col span={6}>
                              <Statistic 
                                title={t('saltstack.runningJobs', '运行中')} 
                                value={jobStats.running_jobs || 0}
                                valueStyle={{ color: '#1890ff' }}
                              />
                            </Col>
                            <Col span={6}>
                              <Statistic 
                                title={t('saltstack.completedJobs', '已完成')} 
                                value={jobStats.completed_jobs || 0}
                                valueStyle={{ color: '#52c41a' }}
                              />
                            </Col>
                            <Col span={6}>
                              <Statistic 
                                title={t('saltstack.failedJobsCount', '失败')} 
                                value={jobStats.failed_jobs || 0}
                                valueStyle={{ color: '#ff4d4f' }}
                              />
                            </Col>
                          </Row>
                        ) : (
                          <Empty description={t('saltstack.noStats', '暂无统计数据')} />
                        )}
                        {jobStats && (
                          <Row gutter={16} style={{ marginTop: 16 }}>
                            <Col span={8}>
                              <Text type="secondary">{t('saltstack.oldestJob', '最早作业')}: </Text>
                              <Text>{jobStats.oldest_job_time ? new Date(jobStats.oldest_job_time).toLocaleString() : '-'}</Text>
                            </Col>
                            <Col span={8}>
                              <Text type="secondary">{t('saltstack.newestJob', '最新作业')}: </Text>
                              <Text>{jobStats.newest_job_time ? new Date(jobStats.newest_job_time).toLocaleString() : '-'}</Text>
                            </Col>
                            <Col span={8}>
                              <Text type="secondary">{t('saltstack.storageEstimate', '预估存储')}: </Text>
                              <Text>{jobStats.storage_estimate || '-'}</Text>
                            </Col>
                          </Row>
                        )}
                      </Card>
                    </Col>

                    {/* 作业保留配置 */}
                    <Col span={24}>
                      <Card 
                        title={t('saltstack.jobRetentionConfig', '作业保留配置')} 
                        size="small"
                        extra={
                          <Space>
                            <Button 
                              icon={<ReloadOutlined />} 
                              onClick={loadJobConfig}
                              loading={jobConfigLoading}
                            >
                              {t('common.refresh', '刷新')}
                            </Button>
                            <Popconfirm
                              title={t('saltstack.confirmCleanup', '确定要立即执行清理吗？')}
                              description={t('saltstack.cleanupDescription', '将删除超过保留天数的旧作业记录')}
                              onConfirm={triggerCleanup}
                              okText={t('common.confirm', '确定')}
                              cancelText={t('common.cancel', '取消')}
                            >
                              <Button 
                                icon={<DeleteOutlined />}
                                loading={cleanupLoading}
                                danger
                              >
                                {t('saltstack.manualCleanup', '手动清理')}
                              </Button>
                            </Popconfirm>
                          </Space>
                        }
                      >
                        <Form
                          form={jobConfigForm}
                          layout="vertical"
                          onFinish={saveJobConfig}
                          initialValues={{
                            retention_days: 30,
                            auto_cleanup_enabled: true,
                            cleanup_interval_hours: 24,
                            max_jobs_count: 10000,
                            redis_cache_days: 7,
                          }}
                        >
                          <Row gutter={16}>
                            <Col span={8}>
                              <Form.Item
                                name="retention_days"
                                label={t('saltstack.retentionDays', '作业保留天数')}
                                tooltip={t('saltstack.retentionDaysTooltip', '超过此天数的作业记录将被自动清理')}
                                rules={[{ required: true, message: t('common.required', '请输入') }]}
                              >
                                <InputNumber 
                                  min={1} 
                                  max={365} 
                                  style={{ width: '100%' }}
                                  addonAfter={t('common.days', '天')}
                                />
                              </Form.Item>
                            </Col>
                            <Col span={8}>
                              <Form.Item
                                name="cleanup_interval_hours"
                                label={t('saltstack.cleanupInterval', '清理检查间隔')}
                                tooltip={t('saltstack.cleanupIntervalTooltip', '系统自动检查并清理旧作业的时间间隔')}
                                rules={[{ required: true, message: t('common.required', '请输入') }]}
                              >
                                <InputNumber 
                                  min={1} 
                                  max={168} 
                                  style={{ width: '100%' }}
                                  addonAfter={t('common.hours', '小时')}
                                />
                              </Form.Item>
                            </Col>
                            <Col span={8}>
                              <Form.Item
                                name="max_jobs_count"
                                label={t('saltstack.maxJobsCount', '最大作业数量')}
                                tooltip={t('saltstack.maxJobsCountTooltip', '当作业数量超过此值时将触发清理')}
                                rules={[{ required: true, message: t('common.required', '请输入') }]}
                              >
                                <InputNumber 
                                  min={100} 
                                  max={1000000} 
                                  style={{ width: '100%' }}
                                  formatter={value => `${value}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
                                  parser={value => value.replace(/,/g, '')}
                                />
                              </Form.Item>
                            </Col>
                          </Row>
                          <Row gutter={16}>
                            <Col span={8}>
                              <Form.Item
                                name="redis_cache_days"
                                label={t('saltstack.redisCacheDays', 'Redis缓存天数')}
                                tooltip={t('saltstack.redisCacheDaysTooltip', 'Redis中作业缓存的过期时间')}
                                rules={[{ required: true, message: t('common.required', '请输入') }]}
                              >
                                <InputNumber 
                                  min={1} 
                                  max={30} 
                                  style={{ width: '100%' }}
                                  addonAfter={t('common.days', '天')}
                                />
                              </Form.Item>
                            </Col>
                            <Col span={8}>
                              <Form.Item
                                name="auto_cleanup_enabled"
                                label={t('saltstack.autoCleanup', '自动清理')}
                                valuePropName="checked"
                                tooltip={t('saltstack.autoCleanupTooltip', '是否启用自动清理功能')}
                              >
                                <Switch 
                                  checkedChildren={t('common.enabled', '启用')} 
                                  unCheckedChildren={t('common.disabled', '禁用')}
                                />
                              </Form.Item>
                            </Col>
                            <Col span={8}>
                              <Form.Item label=" " colon={false}>
                                <Button type="primary" htmlType="submit" icon={<SafetyCertificateOutlined />}>
                                  {t('common.save', '保存配置')}
                                </Button>
                              </Form.Item>
                            </Col>
                          </Row>
                        </Form>

                        {/* 上次清理信息 */}
                        {jobConfig && (
                          <Divider />
                        )}
                        {jobConfig && (
                          <Descriptions size="small" column={3}>
                            <Descriptions.Item label={t('saltstack.lastCleanup', '上次清理时间')}>
                              {jobConfig.last_cleanup_at ? new Date(jobConfig.last_cleanup_at).toLocaleString() : t('common.never', '从未')}
                            </Descriptions.Item>
                            <Descriptions.Item label={t('saltstack.totalCleaned', '累计清理数量')}>
                              {jobConfig.cleaned_count || 0}
                            </Descriptions.Item>
                            <Descriptions.Item label={t('saltstack.configUpdated', '配置更新时间')}>
                              {jobConfig.updated_at ? new Date(jobConfig.updated_at).toLocaleString() : '-'}
                            </Descriptions.Item>
                          </Descriptions>
                        )}
                      </Card>
                    </Col>
                  </Row>
                </Spin>
              </TabPane>
            </Tabs>
          </Card>

          {/* 操作按钮 */}
          <Card style={{ background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
            <Space>
              <Button 
                type="primary" 
                icon={<ReloadOutlined />} 
                onClick={loadAllData}
                loading={statusLoading || minionsLoading || jobsLoading}
              >
                {t('saltstack.refreshData')}
              </Button>
              <Button 
                icon={<PlayCircleOutlined />}
                onClick={openExecModal}
              >
                {t('saltstack.executeCommand')}
              </Button>
              <Button 
                icon={<CloudUploadOutlined />}
                onClick={openBatchInstallModal}
                type="primary"
                ghost
              >
                {t('saltstack.batchInstallMinion')}
              </Button>
              <Button 
                icon={<WifiOutlined />}
                onClick={openSSHTestModal}
              >
                {t('saltstack.sshTest')}
              </Button>
              <Button 
                icon={<SettingOutlined />}
                onClick={() => {
                  setConfigVisible(true);
                  configForm.setFieldsValue({ target: '*' });
                }}
              >
                {t('saltstack.configManagement')}
              </Button>
            </Space>
          </Card>

          {/* 作业详情弹窗 */}
          <Modal
            title={t('saltstack.jobDetail', '作业详情')}
            open={jobDetailVisible}
            onCancel={() => { 
              setJobDetailVisible(false); 
              setJobDetail(null); 
              setJobDetailSearchText('');
              setJobDetailSearchVisible(false);
            }}
            footer={[
              <Button key="close" onClick={() => { 
                setJobDetailVisible(false); 
                setJobDetail(null);
                setJobDetailSearchText('');
                setJobDetailSearchVisible(false);
              }}>
                {t('common.close', '关闭')}
              </Button>
            ]}
            width={1000}
          >
            {jobDetailLoading ? (
              <div style={{ textAlign: 'center', padding: '40px 0' }}>
                <Spin size="large" />
                <div style={{ marginTop: 16 }}>{t('common.loading')}...</div>
              </div>
            ) : jobDetail ? (
              <div>
                {/* 作业基本信息 */}
                <Descriptions title={t('saltstack.jobInfo', '作业信息')} bordered size="small" column={2}>
                  <Descriptions.Item label="JID">{jobDetail.jid}</Descriptions.Item>
                  <Descriptions.Item label={t('saltstack.function', '函数')}>
                    {jobDetail.info?.Function || '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label={t('saltstack.target', '目标')}>
                    {jobDetail.info?.Target || '*'}
                  </Descriptions.Item>
                  <Descriptions.Item label={t('saltstack.user', '用户')}>
                    {jobDetail.info?.User || '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label={t('saltstack.arguments', '参数')} span={2}>
                    <Text code style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
                      {jobDetail.info?.Arguments ? JSON.stringify(jobDetail.info.Arguments, null, 2) : '-'}
                    </Text>
                  </Descriptions.Item>
                </Descriptions>

                {/* 执行结果 */}
                <Divider />
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                  <Title level={5} style={{ margin: 0 }}>{t('saltstack.executionResult', '执行结果')}</Title>
                  <Space>
                    <Button 
                      type={jobDetailSearchVisible ? 'primary' : 'default'}
                      icon={<SearchOutlined />}
                      size="small"
                      onClick={() => setJobDetailSearchVisible(!jobDetailSearchVisible)}
                    >
                      {t('saltstack.search', '搜索')}
                    </Button>
                  </Space>
                </div>
                
                {/* 搜索面板 */}
                {jobDetailSearchVisible && (
                  <Card size="small" style={{ marginBottom: 12, background: isDark ? '#262626' : '#fafafa' }}>
                    <Space wrap>
                      <Input
                        prefix={<SearchOutlined />}
                        placeholder={t('saltstack.searchResultPlaceholder', '搜索节点名或输出内容...')}
                        value={jobDetailSearchText}
                        onChange={(e) => setJobDetailSearchText(e.target.value)}
                        style={{ width: 300 }}
                        allowClear
                      />
                      <Checkbox 
                        checked={jobDetailSearchRegex} 
                        onChange={(e) => setJobDetailSearchRegex(e.target.checked)}
                      >
                        {t('saltstack.useRegex', '正则表达式')}
                      </Checkbox>
                      <Button 
                        size="small" 
                        onClick={() => { setJobDetailSearchText(''); setJobDetailSearchRegex(false); }}
                      >
                        {t('saltstack.clearSearch', '清除')}
                      </Button>
                    </Space>
                  </Card>
                )}

                {jobDetail.result && Object.keys(jobDetail.result).length > 0 ? (
                  <div style={{ maxHeight: 450, overflow: 'auto' }}>
                    {Object.entries(jobDetail.result)
                      .filter(([minion, output]) => {
                        if (!jobDetailSearchText) return true;
                        const outputStr = typeof output === 'string' ? output : JSON.stringify(output, null, 2);
                        if (jobDetailSearchRegex) {
                          try {
                            const regex = new RegExp(jobDetailSearchText, 'i');
                            return regex.test(minion) || regex.test(outputStr);
                          } catch (e) {
                            return minion.toLowerCase().includes(jobDetailSearchText.toLowerCase()) ||
                                   outputStr.toLowerCase().includes(jobDetailSearchText.toLowerCase());
                          }
                        }
                        return minion.toLowerCase().includes(jobDetailSearchText.toLowerCase()) ||
                               outputStr.toLowerCase().includes(jobDetailSearchText.toLowerCase());
                      })
                      .map(([minion, output]) => {
                        const outputStr = typeof output === 'string' ? output : JSON.stringify(output, null, 2);
                        // 高亮搜索文本
                        const highlightText = (text) => {
                          if (!jobDetailSearchText) return text;
                          try {
                            const regex = jobDetailSearchRegex 
                              ? new RegExp(`(${jobDetailSearchText})`, 'gi')
                              : new RegExp(`(${jobDetailSearchText.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
                            return text.split(regex).map((part, i) => 
                              regex.test(part) ? <mark key={i} style={{ background: '#ffe58f', padding: 0 }}>{part}</mark> : part
                            );
                          } catch (e) {
                            return text;
                          }
                        };
                        
                        const outputKey = `job-${minion}`;
                        const isExpanded = expandedOutputs[outputKey];
                        const displayOutput = isExpanded ? outputStr : formatOutput(output, true);
                        
                        return (
                          <Card 
                            key={minion} 
                            size="small" 
                            title={
                              <Space>
                                <DesktopOutlined />
                                <Text strong>{highlightText(minion)}</Text>
                              </Space>
                            }
                            style={{ marginBottom: 8 }}
                            extra={
                              <Space size="small">
                                <Tooltip title={isExpanded ? t('saltstack.compactView', '压缩视图') : t('saltstack.expandView', '展开视图')}>
                                  <Button
                                    type="text"
                                    icon={isExpanded ? <CompressOutlined /> : <ExpandOutlined />}
                                    size="small"
                                    onClick={() => toggleOutputExpand(outputKey)}
                                  />
                                </Tooltip>
                                <Tooltip title={t('saltstack.copy', '复制')}>
                                  <Button
                                    type="text"
                                    icon={<CopyOutlined />}
                                    size="small"
                                    onClick={() => {
                                      navigator.clipboard.writeText(outputStr);
                                      message.success(t('saltstack.copied'));
                                    }}
                                  />
                                </Tooltip>
                              </Space>
                            }
                          >
                            <div
                              tabIndex={0}
                              style={{ 
                                marginBottom: 0, 
                                maxHeight: isExpanded ? '350px' : '150px', 
                                overflow: 'auto',
                                whiteSpace: 'pre-wrap',
                                wordBreak: 'break-all',
                                fontFamily: 'Consolas, Monaco, "Courier New", monospace',
                                fontSize: 12,
                                lineHeight: '1.5',
                                background: isDark ? '#1f1f1f' : '#f5f5f5',
                                padding: '10px 12px',
                                borderRadius: 4,
                                userSelect: 'text',
                                cursor: 'text',
                                outline: 'none',
                                border: `1px solid ${isDark ? '#303030' : '#e8e8e8'}`,
                              }}
                              onFocus={(e) => {
                                e.target.style.borderColor = isDark ? '#177ddc' : '#40a9ff';
                              }}
                              onBlur={(e) => {
                                e.target.style.borderColor = isDark ? '#303030' : '#e8e8e8';
                              }}
                            >
                              {highlightText(displayOutput)}
                            </div>
                            {!isExpanded && outputStr && outputStr.length > 200 && (
                              <div style={{ marginTop: 4, textAlign: 'right' }}>
                                <Button 
                                  type="link" 
                                  size="small" 
                                  onClick={() => toggleOutputExpand(outputKey)}
                                  style={{ padding: 0, height: 'auto' }}
                                >
                                  {t('saltstack.viewFullOutput', '查看完整输出')} ({outputStr.length} {t('saltstack.chars', '字符')})
                                </Button>
                              </div>
                            )}
                          </Card>
                        );
                      })}
                    {jobDetailSearchText && Object.entries(jobDetail.result).filter(([minion, output]) => {
                      const outputStr = typeof output === 'string' ? output : JSON.stringify(output, null, 2);
                      if (jobDetailSearchRegex) {
                        try {
                          const regex = new RegExp(jobDetailSearchText, 'i');
                          return regex.test(minion) || regex.test(outputStr);
                        } catch (e) {
                          return minion.toLowerCase().includes(jobDetailSearchText.toLowerCase()) ||
                                 outputStr.toLowerCase().includes(jobDetailSearchText.toLowerCase());
                        }
                      }
                      return minion.toLowerCase().includes(jobDetailSearchText.toLowerCase()) ||
                             outputStr.toLowerCase().includes(jobDetailSearchText.toLowerCase());
                    }).length === 0 && (
                      <Empty description={t('saltstack.noMatchingResults', '未找到匹配结果')} />
                    )}
                  </div>
                ) : (
                  <Empty description={t('saltstack.noResult', '暂无执行结果')} />
                )}
              </div>
            ) : (
              <Empty description={t('saltstack.noJobData', '无法获取作业数据')} />
            )}
          </Modal>

          {/* 执行命令弹窗 */}
          <Modal
            title={t('saltstack.executeCustomCommand')}
            open={execVisible}
            onCancel={() => { setExecVisible(false); closeSSE(); setExecRunning(false); }}
            footer={execFooter}
            width={900}
          >
            <Form form={execForm} layout="vertical">
              <Form.Item name="target" label={t('saltstack.targetNodes')} rules={[{ required: true, message: t('saltstack.targetRequired') }]}>
                <Input placeholder={t('saltstack.targetNodesPlaceholder')} />
              </Form.Item>
              <Form.Item name="language" label={t('saltstack.language')} rules={[{ required: true }]}> 
                <Select>
                  <Option value="bash">Bash</Option>
                  <Option value="python">Python</Option>
                </Select>
              </Form.Item>
              <Form.Item name="code" label={t('saltstack.code')} rules={[{ required: true, message: t('saltstack.codeRequired') }]}>
                <TextArea rows={10} placeholder={t('saltstack.codePlaceholder')} style={{ fontFamily: 'monospace' }} />
              </Form.Item>
              <Form.Item name="timeout" label={t('saltstack.timeout')}>
                <Input type="number" min={10} max={3600} placeholder="120" />
              </Form.Item>
            </Form>

            <Card size="small" title={t('saltstack.executeProgress')} style={{ marginTop: 12, background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
              <div style={{ maxHeight: 240, overflow: 'auto', background: '#0b1021', color: '#e6e6e6', padding: 8, borderRadius: 6 }}>
                {execEvents.length === 0 ? (
                  <Text type="secondary">{t('saltstack.waitingForExecution')}</Text>
                ) : (
                  execEvents.map((ev, idx) => (
                    <div key={idx} style={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
                      <span style={{ color: '#7aa2f7' }}>[{new Date(ev.ts || Date.now()).toLocaleTimeString()}]</span>
                      <span style={{ color: ev.type === 'error' ? '#f7768e' : '#9ece6a' }}> {ev.type} </span>
                      {ev.host ? <span style={{ color: '#bb9af7' }}>({ev.host})</span> : null}
                      <span> - {ev.message}</span>
                      {ev.data && (
                        <pre style={{ margin: 0, color: '#e0af68' }}>{typeof ev.data === 'string' ? ev.data : JSON.stringify(ev.data, null, 2)}</pre>
                      )}
                    </div>
                  ))
                )}
              </div>
            </Card>
          </Modal>

          {/* 配置管理弹窗 */}
          <Modal
            title={t('saltstack.configTemplateManagement')}
            open={configVisible}
            onCancel={() => setConfigVisible(false)}
            footer={[
              <Button key="cancel" onClick={() => setConfigVisible(false)}>{t('saltstack.cancel')}</Button>,
              <Button 
                key="apply" 
                type="primary" 
                onClick={() => {
                  configForm.validateFields().then(values => {
                    message.info(t('saltstack.applyTemplateInfo', { template: values.template, target: values.target }));
                    // TODO: 调用后端 API 应用配置模板
                    // saltStackAPI.applyTemplate({ template: values.template, target: values.target });
                    setConfigVisible(false);
                  });
                }}
              >
                {t('saltstack.applyConfig')}
              </Button>,
            ]}
            width={700}
          >
            <Form form={configForm} layout="vertical">
              <Form.Item 
                name="target" 
                label={t('saltstack.targetNodes')} 
                rules={[{ required: true, message: t('saltstack.targetRequired') }]}
              >
                <Input placeholder={t('saltstack.targetNodesPlaceholder')} />
              </Form.Item>
              <Form.Item 
                name="template" 
                label={t('saltstack.configTemplate')} 
                rules={[{ required: true, message: t('saltstack.selectTemplate') }]}
              >
                <Select placeholder={t('saltstack.selectTemplate')}>
                  {configTemplates.map(tpl => (
                    <Option key={tpl.id} value={tpl.id}>
                      {tpl.name} - {tpl.desc}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
              <Alert
                message={t('saltstack.hint')}
                description={t('saltstack.configHint')}
                type="info"
                showIcon
                style={{ marginTop: 16 }}
              />
            </Form>
          </Modal>

          {/* 批量安装 Salt Minion 弹窗 */}
          <Modal
            title={
              <Space>
                <CloudUploadOutlined />
                {t('saltstack.batchInstallMinion')}
              </Space>
            }
            open={batchInstallVisible}
            onCancel={() => { 
              setBatchInstallVisible(false); 
              closeBatchSSE(); 
              setBatchInstallRunning(false); 
            }}
            footer={[
              <Button 
                key="cancel" 
                onClick={() => { 
                  setBatchInstallVisible(false); 
                  closeBatchSSE(); 
                  setBatchInstallRunning(false); 
                }}
                disabled={batchInstallRunning}
              >
                {batchInstallRunning ? t('saltstack.cancel') : t('saltstack.close')}
              </Button>,
              <Button 
                key="install" 
                type="primary" 
                onClick={handleBatchInstall}
                loading={batchInstallRunning}
                icon={<CloudUploadOutlined />}
              >
                {t('saltstack.startInstall')}
              </Button>,
            ]}
            width={1000}
            destroyOnClose
          >
            <Form form={batchInstallForm} layout="vertical">
              <Row gutter={16}>
                <Col span={6}>
                  <Form.Item 
                    name="parallel" 
                    label={
                      <Space>
                        {t('saltstack.parallel')}
                        <Tooltip title={t('saltstack.parallelHint')}>
                          <QuestionCircleOutlined />
                        </Tooltip>
                      </Space>
                    }
                    initialValue={3}
                  >
                    <InputNumber min={1} max={20} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={6}>
                  <Form.Item 
                    name="master_host" 
                    label={t('saltstack.masterHost')}
                    initialValue="salt"
                  >
                    <Input placeholder="salt / 192.168.1.100" />
                  </Form.Item>
                </Col>
                <Col span={6}>
                  <Form.Item 
                    name="install_type" 
                    label={t('saltstack.installType')}
                    initialValue="saltstack"
                  >
                    <Select>
                      <Option value="saltstack">SaltStack Minion</Option>
                      <Option value="slurm">SLURM Client</Option>
                    </Select>
                  </Form.Item>
                </Col>
                <Col span={6}>
                  <Form.Item 
                    name="auto_accept" 
                    label={t('saltstack.autoAccept')}
                    valuePropName="checked"
                    initialValue={true}
                  >
                    <Switch checkedChildren="Yes" unCheckedChildren="No" />
                  </Form.Item>
                </Col>
              </Row>

              <Divider orientation="left">{t('saltstack.globalSudoSettings')}</Divider>
              <Row gutter={16}>
                <Col span={6}>
                  <Form.Item 
                    name="global_use_sudo" 
                    label={
                      <Space>
                        {t('saltstack.useSudo')}
                        <Tooltip title={t('saltstack.sudoHint')}>
                          <QuestionCircleOutlined />
                        </Tooltip>
                      </Space>
                    }
                    valuePropName="checked"
                    initialValue={false}
                  >
                    <Switch checkedChildren="Yes" unCheckedChildren="No" />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item 
                    name="global_group" 
                    label={
                      <Space>
                        {t('saltstack.globalGroup', '全局分组')}
                        <Tooltip title={t('saltstack.globalGroupHint', '为所有主机设置统一的分组，单独设置的分组优先')}>
                          <QuestionCircleOutlined />
                        </Tooltip>
                      </Space>
                    }
                  >
                    <Select
                      placeholder={t('saltstack.selectGroup', '选择分组')}
                      allowClear
                      style={{ width: '100%' }}
                      loading={groupsLoading}
                      onDropdownVisibleChange={(open) => {
                        if (open && minionGroups.length === 0) {
                          loadMinionGroups();
                        }
                      }}
                      dropdownRender={(menu) => (
                        <>
                          <div style={{ padding: '8px', borderBottom: '1px solid #f0f0f0' }}>
                            <Space.Compact style={{ width: '100%' }}>
                              <Input
                                placeholder={t('saltstack.quickCreateGroupPlaceholder', '输入新分组名称')}
                                value={quickGroupName}
                                onChange={(e) => setQuickGroupName(e.target.value)}
                                onKeyDown={(e) => {
                                  e.stopPropagation();
                                  if (e.key === 'Enter' && quickGroupName.trim()) {
                                    handleQuickCreateGroup();
                                  }
                                }}
                                style={{ flex: 1 }}
                              />
                              <Button
                                type="primary"
                                icon={<PlusOutlined />}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleQuickCreateGroup();
                                }}
                                loading={quickGroupCreating}
                                disabled={!quickGroupName.trim()}
                              >
                                {t('common.create', '创建')}
                              </Button>
                            </Space.Compact>
                          </div>
                          {menu}
                        </>
                      )}
                    >
                      {minionGroups.map(g => (
                        <Select.Option key={g.id} value={g.name}>
                          <Tag color={g.color || 'default'}>{g.name}</Tag>
                        </Select.Option>
                      ))}
                    </Select>
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    💡 {t('saltstack.sudoHint')}
                  </Text>
                </Col>
              </Row>

              <Divider orientation="left">{t('saltstack.monitoringSettings', '监控代理设置')}</Divider>
              <Row gutter={16}>
                <Col span={6}>
                  <Form.Item 
                    name="install_categraf" 
                    label={
                      <Space>
                        {t('saltstack.installCategraf', '安装 Categraf')}
                        <Tooltip title={t('saltstack.categrafHint', 'Categraf 是轻量级的监控采集代理，用于采集节点的 CPU、内存、磁盘等监控指标')}>
                          <QuestionCircleOutlined />
                        </Tooltip>
                      </Space>
                    }
                    valuePropName="checked"
                    initialValue={false}
                  >
                    <Switch checkedChildren="Yes" unCheckedChildren="No" />
                  </Form.Item>
                </Col>
                <Col span={6}>
                  <Form.Item 
                    name="n9e_host" 
                    label={t('saltstack.n9eHost', 'N9E 服务器地址')}
                    tooltip={t('saltstack.n9eHostHint', 'Nightingale 监控系统的服务器地址，留空则使用系统默认配置')}
                  >
                    <Input placeholder={t('saltstack.n9eHostPlaceholder', '留空使用默认地址')} />
                  </Form.Item>
                </Col>
                <Col span={4}>
                  <Form.Item 
                    name="n9e_port" 
                    label={t('saltstack.n9ePort', '端口')}
                    initialValue="17000"
                  >
                    <Input placeholder="17000" />
                  </Form.Item>
                </Col>
                <Col span={6}>
                  <Form.Item 
                    name="categraf_version" 
                    label={t('saltstack.categrafVersion', 'Categraf 版本')}
                    tooltip={t('saltstack.categrafVersionHint', '留空使用系统默认版本')}
                  >
                    <Input placeholder={t('saltstack.categrafVersionPlaceholder', '留空使用默认版本')} />
                  </Form.Item>
                </Col>
              </Row>

              <Divider orientation="left">
                <Space>
                  {t('saltstack.targetHostList')}
                  <Button type="link" size="small" icon={<PlusOutlined />} onClick={addHostRow}>
                    {t('saltstack.addHost')}
                  </Button>
                  <Upload
                    accept=".csv,.json,.yaml,.yml,.ini"
                    showUploadList={false}
                    beforeUpload={(file) => {
                      handleFileImport(file);
                      return false; // 阻止默认上传行为
                    }}
                    disabled={importLoading}
                  >
                    <Button type="link" size="small" icon={<UploadOutlined />} loading={importLoading}>
                      {t('saltstack.importFile')}
                    </Button>
                  </Upload>
                  <Button type="link" size="small" icon={<CopyOutlined />} onClick={openPasteImportModal}>
                    {t('saltstack.pasteImport', '粘贴导入')}
                  </Button>
                  <Dropdown overlay={templateMenu} trigger={['click']}>
                    <Button type="link" size="small" icon={<DownloadOutlined />}>
                      {t('saltstack.downloadTemplate')}
                    </Button>
                  </Dropdown>
                </Space>
              </Divider>

              <Alert
                type="info"
                showIcon
                style={{ marginBottom: 12 }}
                message={t('saltstack.importFileHint')}
              />

              <div style={{ maxHeight: 300, overflow: 'auto' }}>
                {batchInstallHosts.map((host, index) => (
                  <Row gutter={8} key={host.key} style={{ marginBottom: 8 }}>
                    <Col span={4}>
                      <Input 
                        placeholder={t('saltstack.hostAddressPlaceholder')} 
                        value={host.host}
                        onChange={(e) => updateHostRow(host.key, 'host', e.target.value)}
                        addonBefore={
                          <Space size={4}>
                            {index > 0 && (
                              <Tooltip title={t('saltstack.copyFirstRowConfig', '复制第一行配置')}>
                                <CopyOutlined 
                                  style={{ cursor: 'pointer', color: '#1890ff' }}
                                  onClick={() => copyFirstRowConfig(host.key)}
                                />
                              </Tooltip>
                            )}
                            <span>{`#${index + 1}`}</span>
                          </Space>
                        }
                      />
                    </Col>
                    <Col span={2}>
                      <InputNumber 
                        placeholder={t('saltstack.port')} 
                        value={host.port}
                        onChange={(v) => updateHostRow(host.key, 'port', v)}
                        min={1}
                        max={65535}
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col span={3}>
                      <Input 
                        placeholder={t('saltstack.usernamePlaceholder')} 
                        value={host.username}
                        onChange={(e) => updateHostRow(host.key, 'username', e.target.value)}
                      />
                    </Col>
                    <Col span={5}>
                      <Input.Password 
                        placeholder={t('saltstack.passwordPlaceholder')} 
                        value={host.password}
                        onChange={(e) => updateHostRow(host.key, 'password', e.target.value)}
                      />
                    </Col>
                    <Col span={4}>
                      <Select
                        placeholder={t('saltstack.selectGroup', '选择分组')}
                        value={host.group || undefined}
                        onChange={(v) => updateHostRow(host.key, 'group', v)}
                        allowClear
                        style={{ width: '100%' }}
                        loading={groupsLoading}
                        onDropdownVisibleChange={(open) => {
                          if (open && minionGroups.length === 0) {
                            loadMinionGroups();
                          }
                        }}
                        dropdownRender={(menu) => (
                          <>
                            <div style={{ padding: '8px', borderBottom: '1px solid #f0f0f0' }}>
                              <Space.Compact style={{ width: '100%' }}>
                                <Input
                                  placeholder={t('saltstack.quickCreateGroupPlaceholder', '输入新分组名称')}
                                  value={quickGroupName}
                                  onChange={(e) => setQuickGroupName(e.target.value)}
                                  onKeyDown={(e) => {
                                    e.stopPropagation();
                                    if (e.key === 'Enter' && quickGroupName.trim()) {
                                      handleQuickCreateGroup();
                                    }
                                  }}
                                  style={{ flex: 1 }}
                                />
                                <Button
                                  type="primary"
                                  icon={<PlusOutlined />}
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleQuickCreateGroup();
                                  }}
                                  loading={quickGroupCreating}
                                  disabled={!quickGroupName.trim()}
                                >
                                  {t('common.create', '创建')}
                                </Button>
                              </Space.Compact>
                            </div>
                            <div style={{ padding: '4px 8px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'flex-end' }}>
                              <Button
                                type="link"
                                size="small"
                                icon={<ReloadOutlined spin={groupsLoading} />}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  loadMinionGroups();
                                }}
                                loading={groupsLoading}
                              >
                                {t('common.refresh', '刷新')}
                              </Button>
                            </div>
                            {menu}
                          </>
                        )}
                      >
                        {minionGroups.map(g => (
                          <Select.Option key={g.id} value={g.name}>
                            <Tag color={g.color || 'default'}>{g.name}</Tag>
                          </Select.Option>
                        ))}
                      </Select>
                    </Col>
                    <Col span={5}>
                      <Space size="small">
                        <Tooltip title={t('saltstack.useSudo')}>
                          <Switch 
                            size="small"
                            checked={host.use_sudo}
                            onChange={(v) => updateHostRow(host.key, 'use_sudo', v)}
                            checkedChildren="sudo"
                            unCheckedChildren="sudo"
                          />
                        </Tooltip>
                        <Tooltip title={t('saltstack.installCategraf', '安装 Categraf')}>
                          <Switch 
                            size="small"
                            checked={host.install_categraf}
                            onChange={(v) => updateHostRow(host.key, 'install_categraf', v)}
                            checkedChildren="Categraf"
                            unCheckedChildren="Categraf"
                          />
                        </Tooltip>
                        <Button 
                          type="text" 
                          danger 
                          icon={<DeleteOutlined />} 
                          onClick={() => removeHostRow(host.key)}
                          disabled={batchInstallHosts.length <= 1}
                          size="small"
                        />
                      </Space>
                    </Col>
                  </Row>
                ))}
              </div>

              {/* 动态并行度信息 */}
              {parallelInfo.host_count > 0 && (
                <Alert
                  type="success"
                  showIcon
                  style={{ marginTop: 12 }}
                  message={
                    <Space>
                      <span>{t('saltstack.dynamicParallel', '动态并行度')}: </span>
                      <Tag color="blue">{parallelInfo.parallel} {t('saltstack.workers', '并发')}</Tag>
                      <span>/</span>
                      <span>{parallelInfo.host_count} {t('saltstack.hosts', '台主机')}</span>
                      <span>({parallelInfo.percentage.toFixed(1)}%)</span>
                    </Space>
                  }
                  description={
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {t('saltstack.dynamicParallelHint', '根据主机数量自动计算最优并发数，避免网络/资源过载')}
                    </Text>
                  }
                />
              )}
            </Form>

            {/* 安装进度 */}
            <Card size="small" title={t('saltstack.installProgress')} style={{ marginTop: 16, background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
              {batchInstallTaskId && (
                <div style={{ marginBottom: 8 }}>
                  <Text type="secondary">{t('saltstack.taskId')}: </Text>
                  <Text copyable>{batchInstallTaskId}</Text>
                </div>
              )}
              <div 
                style={{ maxHeight: 280, overflow: 'auto', background: '#0b1021', color: '#e6e6e6', padding: 12, borderRadius: 6 }}
                tabIndex={0}
                onKeyDown={(e) => {
                  // 拦截 Ctrl+A，只选中日志框内的内容
                  if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
                    e.preventDefault();
                    e.stopPropagation();
                    const selection = window.getSelection();
                    const range = document.createRange();
                    range.selectNodeContents(e.currentTarget);
                    selection.removeAllRanges();
                    selection.addRange(range);
                  }
                }}
              >
                {batchInstallEvents.length === 0 ? (
                  <Text type="secondary">{t('saltstack.waitingForInstall')}</Text>
                ) : (
                  batchInstallEvents.map((ev, idx) => (
                    <div key={idx} style={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace', fontSize: 12, lineHeight: 1.5 }}>
                      <span style={{ color: '#7aa2f7' }}>
                        [{ev.ts ? new Date(ev.ts).toLocaleTimeString() : new Date().toLocaleTimeString()}]
                      </span>
                      <span style={{ 
                        color: ev.type === 'error' ? '#f7768e' : 
                               ev.type === 'complete' ? '#9ece6a' : 
                               ev.type === 'progress' ? '#bb9af7' : '#e0af68' 
                      }}>
                        {' '}{ev.type}{' '}
                      </span>
                      {ev.host && <span style={{ color: '#73daca' }}>({ev.host})</span>}
                      <span> - {ev.message}</span>
                      {ev.data && typeof ev.data === 'object' && (
                        <pre style={{ margin: '4px 0 0 20px', color: '#e0af68', fontSize: 11 }}>
                          {JSON.stringify(ev.data, null, 2)}
                        </pre>
                      )}
                    </div>
                  ))
                )}
              </div>
            </Card>
          </Modal>

          {/* 粘贴导入弹窗 */}
          <Modal
            title={
              <Space>
                <CopyOutlined />
                {t('saltstack.pasteImportTitle', '粘贴导入配置')}
              </Space>
            }
            open={pasteImportVisible}
            onCancel={() => {
              setPasteImportVisible(false);
              setPasteContent('');
            }}
            footer={[
              <Button 
                key="cancel" 
                onClick={() => {
                  setPasteImportVisible(false);
                  setPasteContent('');
                }}
              >
                {t('saltstack.cancel', '取消')}
              </Button>,
              <Button 
                key="import" 
                type="primary" 
                onClick={handlePasteImport}
                loading={pasteImportLoading}
                icon={<CloudUploadOutlined />}
                disabled={!pasteContent || !pasteContent.trim()}
              >
                {t('saltstack.importNow', '立即导入')}
              </Button>,
            ]}
            width={800}
            destroyOnClose
          >
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message={t('saltstack.pasteImportHint', '请将 CSV、JSON 或 YAML 格式的主机配置粘贴到下方文本框中')}
              description={t('saltstack.pasteImportDesc', '支持的格式：CSV（逗号分隔）、JSON（数组格式）、YAML（hosts 列表）、Ansible INI 格式')}
            />
            
            <Row gutter={16} style={{ marginBottom: 16 }}>
              <Col span={6}>
                <Text strong>{t('saltstack.selectFormat', '选择格式')}:</Text>
              </Col>
              <Col span={18}>
                <Select
                  value={pasteFormat}
                  onChange={setPasteFormat}
                  style={{ width: 200 }}
                >
                  <Option value="csv">CSV (.csv)</Option>
                  <Option value="json">JSON (.json)</Option>
                  <Option value="yaml">YAML (.yaml)</Option>
                  <Option value="ini">Ansible INI (.ini)</Option>
                </Select>
                <Button 
                  type="link" 
                  size="small"
                  onClick={() => setPasteContent(getPasteFormatExample(pasteFormat))}
                  style={{ marginLeft: 8 }}
                >
                  {t('saltstack.fillExample', '填入示例')}
                </Button>
              </Col>
            </Row>

            <TextArea
              rows={12}
              value={pasteContent}
              onChange={(e) => setPasteContent(e.target.value)}
              placeholder={getPasteFormatExample(pasteFormat)}
              style={{ fontFamily: 'monospace', fontSize: 12 }}
            />

            <div style={{ marginTop: 12 }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                💡 {t('saltstack.pasteFormatTip', '提示：可以直接从 Excel、文本编辑器或其他来源复制数据粘贴到上方')}
              </Text>
            </div>

            <Divider orientation="left" style={{ marginTop: 16, marginBottom: 12 }}>
              {t('saltstack.formatReference', '格式参考')}
            </Divider>

            <Row gutter={16}>
              <Col span={12}>
                <Card size="small" title="CSV 格式" style={{ marginBottom: 8, background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
                  <pre style={{ fontSize: 10, margin: 0, overflow: 'auto', maxHeight: 80, color: isDark ? 'rgba(255,255,255,0.85)' : 'inherit' }}>
{`host,port,username,password,use_sudo,group
192.168.1.100,22,root,pass123,false,web`}
                  </pre>
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small" title="JSON 格式" style={{ marginBottom: 8, background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
                  <pre style={{ fontSize: 10, margin: 0, overflow: 'auto', maxHeight: 80, color: isDark ? 'rgba(255,255,255,0.85)' : 'inherit' }}>
{`[{"host":"192.168.1.100","port":22,
  "username":"root","password":"pass"}]`}
                  </pre>
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small" title="YAML 格式" style={{ background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
                  <pre style={{ fontSize: 10, margin: 0, overflow: 'auto', maxHeight: 80, color: isDark ? 'rgba(255,255,255,0.85)' : 'inherit' }}>
{`hosts:
  - host: 192.168.1.100
    port: 22
    username: root`}
                  </pre>
                </Card>
              </Col>
              <Col span={12}>
                <Card size="small" title="Ansible INI 格式" style={{ background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
                  <pre style={{ fontSize: 10, margin: 0, overflow: 'auto', maxHeight: 80, color: isDark ? 'rgba(255,255,255,0.85)' : 'inherit' }}>
{`[web]
192.168.1.100 ansible_user=root`}
                  </pre>
                </Card>
              </Col>
            </Row>
          </Modal>

          {/* 批量安装 Categraf 弹窗 */}
          <Modal
            title={
              <Space>
                <ThunderboltOutlined />
                {t('saltstack.batchCategrafTitle', '批量安装 Categraf')}
              </Space>
            }
            open={batchCategrafVisible}
            onCancel={() => {
              closeBatchCategrafSSE();
              setBatchCategrafVisible(false);
              setBatchCategrafHosts([]);
              setBatchCategrafEvents([]);
            }}
            footer={batchCategrafRunning ? null : [
              <Button 
                key="cancel" 
                onClick={() => {
                  closeBatchCategrafSSE();
                  setBatchCategrafVisible(false);
                  setBatchCategrafHosts([]);
                  setBatchCategrafEvents([]);
                }}
              >
                {t('saltstack.cancel', '取消')}
              </Button>,
              <Button 
                key="install" 
                type="primary" 
                onClick={handleBatchCategrafInstall}
                disabled={batchCategrafHosts.filter(h => h.selected).length === 0}
                icon={<CloudUploadOutlined />}
              >
                {t('saltstack.installCategraf', '安装 Categraf')}
                {batchCategrafHosts.filter(h => h.selected).length > 0 && 
                  ` (${batchCategrafHosts.filter(h => h.selected).length})`
                }
              </Button>,
            ]}
            width={900}
            destroyOnClose
          >
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message={t('saltstack.batchCategrafHint', '为已安装 Salt Minion 但未安装 Categraf 的节点补充安装监控代理')}
              description={t('saltstack.batchCategrafDesc', 'Categraf 是一个轻量级的监控代理，用于收集系统指标并发送到 Nightingale 监控平台')}
            />

            {!batchCategrafRunning ? (
              <>
                <Divider orientation="left">
                  <Space>
                    {t('saltstack.selectTargetMinions', '选择目标 Minion')}
                    <Checkbox
                      checked={batchCategrafHosts.length > 0 && batchCategrafHosts.every(h => h.selected)}
                      indeterminate={batchCategrafHosts.some(h => h.selected) && !batchCategrafHosts.every(h => h.selected)}
                      onChange={(e) => {
                        setBatchCategrafHosts(batchCategrafHosts.map(h => ({
                          ...h,
                          selected: e.target.checked
                        })));
                      }}
                    >
                      {t('saltstack.selectAll', '全选')}
                    </Checkbox>
                  </Space>
                </Divider>

                {batchCategrafHosts.length === 0 ? (
                  <Empty description={t('saltstack.noMinionsNeedCategraf', '没有需要安装 Categraf 的 Minion')} />
                ) : (
                  <Table
                    size="small"
                    dataSource={batchCategrafHosts}
                    rowKey="minion_id"
                    pagination={{ pageSize: 10 }}
                    columns={[
                      {
                        title: t('saltstack.select', '选择'),
                        width: 60,
                        render: (_, record) => (
                          <Checkbox
                            checked={record.selected}
                            onChange={(e) => {
                              setBatchCategrafHosts(batchCategrafHosts.map(h => 
                                h.minion_id === record.minion_id 
                                  ? { ...h, selected: e.target.checked }
                                  : h
                              ));
                            }}
                          />
                        ),
                      },
                      {
                        title: 'Minion ID',
                        dataIndex: 'minion_id',
                        width: 200,
                      },
                      {
                        title: t('saltstack.hostAddress', '主机地址'),
                        dataIndex: 'host',
                        width: 150,
                      },
                      {
                        title: t('saltstack.group', '分组'),
                        dataIndex: 'group',
                        width: 120,
                        render: (group) => group ? <Tag color="blue">{group}</Tag> : '-',
                      },
                      {
                        title: t('saltstack.categrafStatus', 'Categraf 状态'),
                        dataIndex: 'categraf_installed',
                        width: 120,
                        render: (installed) => installed 
                          ? <Tag color="green">{t('saltstack.installed', '已安装')}</Tag>
                          : <Tag color="orange">{t('saltstack.notInstalled', '未安装')}</Tag>,
                      },
                    ]}
                  />
                )}
              </>
            ) : (
              <Card size="small" title={t('saltstack.installProgress', '安装进度')} style={{ background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
                {batchCategrafTaskId && (
                  <div style={{ marginBottom: 8 }}>
                    <Text type="secondary">{t('saltstack.taskId', '任务ID')}: </Text>
                    <Text copyable>{batchCategrafTaskId}</Text>
                  </div>
                )}
                
                <Timeline style={{ maxHeight: 400, overflow: 'auto' }}>
                  {batchCategrafEvents.map((event, idx) => (
                    <Timeline.Item 
                      key={idx} 
                      color={
                        event.status === 'success' ? 'green' : 
                        event.status === 'error' ? 'red' : 
                        event.status === 'running' ? 'blue' : 'gray'
                      }
                    >
                      <Space>
                        {event.status === 'running' && <LoadingOutlined />}
                        <Text type={event.status === 'error' ? 'danger' : undefined}>
                          {event.host || event.minion_id || ''}: {event.message}
                        </Text>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          {event.timestamp}
                        </Text>
                      </Space>
                    </Timeline.Item>
                  ))}
                </Timeline>
                
                {batchCategrafRunning && (
                  <div style={{ textAlign: 'center', marginTop: 16 }}>
                    <Spin tip={t('saltstack.installing', '安装中...')} />
                  </div>
                )}
              </Card>
            )}
          </Modal>

          {/* 部署节点指标采集弹窗 */}
          <Modal
            title={
              <Space>
                <DashboardOutlined />
                {t('saltstack.deployNodeMetrics', '部署指标采集')}
              </Space>
            }
            open={deployMetricsVisible}
            onCancel={() => {
              setDeployMetricsVisible(false);
              deployMetricsForm.resetFields();
            }}
            footer={[
              <Button 
                key="cancel" 
                onClick={() => {
                  setDeployMetricsVisible(false);
                  deployMetricsForm.resetFields();
                }}
              >
                {t('saltstack.cancel', '取消')}
              </Button>,
              <Button 
                key="deploy" 
                type="primary" 
                loading={deployMetricsLoading}
                onClick={handleDeployNodeMetrics}
                icon={<CloudUploadOutlined />}
              >
                {t('saltstack.deploy', '部署')}
              </Button>,
            ]}
            width={600}
            destroyOnClose
          >
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
              message={t('saltstack.deployNodeMetricsDesc', '向选定节点部署 GPU/IB 指标采集脚本和定时任务')}
              description={t('saltstack.deployNodeMetricsHint', '部署后，节点将定期采集 GPU 驱动版本、CUDA 版本、IB 端口状态等信息并上报到系统')}
            />

            <Form
              form={deployMetricsForm}
              layout="vertical"
              initialValues={{ interval: 3, target: '*' }}
            >
              <Form.Item
                name="target"
                label={t('saltstack.targetMinions', '目标节点')}
                rules={[{ required: true, message: t('saltstack.targetRequired', '请输入目标节点') }]}
                tooltip={t('saltstack.targetTooltip', '可以是单个 Minion ID、通配符（如 gpu-* 或 *）或逗号分隔的多个 ID')}
              >
                <Input placeholder={t('saltstack.targetPlaceholder', '例如: * 或 gpu-node-* 或 node1,node2')} />
              </Form.Item>

              <Form.Item
                name="interval"
                label={t('saltstack.collectInterval', '采集间隔（分钟）')}
                rules={[{ required: true, message: t('saltstack.intervalRequired', '请输入采集间隔') }]}
              >
                <InputNumber min={1} max={60} style={{ width: '100%' }} />
              </Form.Item>
            </Form>

            <Divider style={{ margin: '16px 0' }} />

            <Text type="secondary">
              {t('saltstack.metricsInfo', '采集的指标包括：')}
            </Text>
            <ul style={{ margin: '8px 0', paddingLeft: 20 }}>
              <li>{t('saltstack.gpuMetrics', 'GPU 信息：驱动版本、CUDA 版本、GPU 数量、型号、显存')}</li>
              <li>{t('saltstack.ibMetrics', 'IB 网络：活跃端口数量、端口状态、速率、固件版本')}</li>
              <li>{t('saltstack.sysMetrics', '系统信息：内核版本、操作系统版本')}</li>
            </ul>
          </Modal>

          {/* SSH 测试弹窗 */}
          <Modal
            title={
              <Space>
                <WifiOutlined />
                {t('saltstack.sshTestTitle')}
              </Space>
            }
            open={sshTestVisible}
            onCancel={() => setSSHTestVisible(false)}
            footer={[
              <Button key="cancel" onClick={() => setSSHTestVisible(false)}>
                {t('saltstack.close')}
              </Button>,
              <Button 
                key="test" 
                type="primary" 
                onClick={handleSSHTest}
                loading={sshTestRunning}
                icon={<SafetyCertificateOutlined />}
              >
                {t('saltstack.startTest')}
              </Button>,
            ]}
            width={1000}
            destroyOnClose
          >
            <Alert
              message={t('saltstack.sshTest')}
              description={t('saltstack.sshTestDesc')}
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <Divider orientation="left">
              <Space>
                {t('saltstack.targetHostList')}
                <Button type="link" size="small" icon={<PlusOutlined />} onClick={addSSHTestHostRow}>
                  {t('saltstack.addHost')}
                </Button>
              </Space>
            </Divider>

            <div style={{ maxHeight: 250, overflow: 'auto' }}>
              {sshTestHosts.map((host, index) => (
                <Row gutter={8} key={host.key} style={{ marginBottom: 8 }}>
                  <Col span={5}>
                    <Input 
                      placeholder={t('saltstack.hostAddress')} 
                      value={host.host}
                      onChange={(e) => updateSSHTestHostRow(host.key, 'host', e.target.value)}
                      addonBefore={`#${index + 1}`}
                    />
                  </Col>
                  <Col span={2}>
                    <InputNumber 
                      placeholder={t('saltstack.port')} 
                      value={host.port}
                      onChange={(v) => updateSSHTestHostRow(host.key, 'port', v)}
                      min={1}
                      max={65535}
                      style={{ width: '100%' }}
                    />
                  </Col>
                  <Col span={4}>
                    <Input 
                      placeholder={t('saltstack.username')} 
                      value={host.username}
                      onChange={(e) => updateSSHTestHostRow(host.key, 'username', e.target.value)}
                    />
                  </Col>
                  <Col span={7}>
                    <Input.Password 
                      placeholder={t('saltstack.passwordHint')} 
                      value={host.password}
                      onChange={(e) => updateSSHTestHostRow(host.key, 'password', e.target.value)}
                    />
                  </Col>
                  <Col span={1}>
                    <Button 
                      type="text" 
                      danger 
                      icon={<DeleteOutlined />} 
                      onClick={() => removeSSHTestHostRow(host.key)}
                      disabled={sshTestHosts.length <= 1}
                    />
                  </Col>
                </Row>
              ))}
            </div>

            {/* 测试结果 */}
            {sshTestResults.length > 0 && (
              <Card size="small" title={t('saltstack.result')} style={{ marginTop: 16, background: isDark ? '#1f1f1f' : '#fff', borderColor: isDark ? '#303030' : '#f0f0f0' }}>
                <Table
                  dataSource={sshTestResults}
                  rowKey="host"
                  size="small"
                  pagination={false}
                  columns={[
                    {
                      title: t('saltstack.hostAddress'),
                      dataIndex: 'host',
                      width: 150,
                    },
                    {
                      title: t('saltstack.connectionStatus'),
                      dataIndex: 'connected',
                      width: 100,
                      render: (v) => v ? 
                        <Tag color="success" icon={<CheckCircleOutlined />}>{t('saltstack.connectionSuccess')}</Tag> : 
                        <Tag color="error" icon={<ExclamationCircleOutlined />}>{t('saltstack.connectionFailed')}</Tag>
                    },
                    {
                      title: t('saltstack.authMethod'),
                      dataIndex: 'auth_method',
                      width: 100,
                      render: (v) => v ? <Tag icon={<KeyOutlined />}>{v}</Tag> : '-'
                    },
                    {
                      title: t('saltstack.sudoPermission'),
                      dataIndex: 'has_sudo',
                      width: 120,
                      render: (v, record) => v ? 
                        <Tag color="success" icon={<LockOutlined />}>
                          {record.sudo_no_password ? t('saltstack.passwordlessSudo') : t('saltstack.needPassword')}
                        </Tag> : 
                        <Tag color="warning">{t('saltstack.noSudo')}</Tag>
                    },
                    {
                      title: t('saltstack.hostname'),
                      dataIndex: 'hostname',
                      width: 150,
                    },
                    {
                      title: t('saltstack.osInfo'),
                      dataIndex: 'os_info',
                      ellipsis: true,
                    },
                    {
                      title: t('saltstack.duration') + '(ms)',
                      dataIndex: 'duration',
                      width: 80,
                    },
                    {
                      title: t('saltstack.error'),
                      dataIndex: 'error',
                      ellipsis: true,
                      render: (v) => v ? <Text type="danger">{v}</Text> : '-'
                    },
                  ]}
                />
              </Card>
            )}
          </Modal>

          {/* 卸载 Minion 弹窗 */}
          <Modal
            title={
              <Space>
                <DeleteOutlined />
                {t('saltstack.uninstallTitle', { id: uninstallMinionId })}
              </Space>
            }
            open={uninstallModalVisible}
            onCancel={() => setUninstallModalVisible(false)}
            onOk={handleUninstallMinion}
            okText={t('saltstack.confirmUninstall')}
            okButtonProps={{ danger: true }}
            cancelText={t('saltstack.cancel')}
            width={600}
          >
            <Alert
              message={t('saltstack.warning')}
              description={t('saltstack.uninstallWarning')}
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <Form form={uninstallForm} layout="vertical">
              <Row gutter={16}>
                <Col span={16}>
                  <Form.Item 
                    name="host" 
                    label={t('saltstack.hostAddress')}
                    rules={[{ required: true, message: t('saltstack.targetRequired') }]}
                  >
                    <Input placeholder="IP / Domain" />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item name="port" label={t('saltstack.port')} initialValue={22}>
                    <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item 
                    name="username" 
                    label={t('saltstack.username')}
                    rules={[{ required: true, message: t('saltstack.targetRequired') }]}
                  >
                    <Input placeholder="root" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item 
                    name="password" 
                    label={t('saltstack.passwordHint')}
                    rules={[{ required: true, message: t('saltstack.targetRequired') }]}
                  >
                    <Input.Password placeholder={t('saltstack.passwordHint')} />
                  </Form.Item>
                </Col>
              </Row>
              <Row gutter={16}>
                <Col span={8}>
                  <Form.Item 
                    name="use_sudo" 
                    label={t('saltstack.useSudo')}
                    valuePropName="checked"
                  >
                    <Switch checkedChildren="Yes" unCheckedChildren="No" />
                  </Form.Item>
                </Col>
                <Col span={16}>
                  <Text type="secondary" style={{ lineHeight: '32px' }}>
                    💡 {t('saltstack.sudoHint')}
                  </Text>
                </Col>
              </Row>
            </Form>
          </Modal>

          {/* 分组管理弹窗 */}
          <Modal
            title={
              <Space>
                <TeamOutlined />
                {editingGroup ? t('saltstack.editGroup', '编辑分组') : t('saltstack.createGroup', '创建分组')}
              </Space>
            }
            open={groupModalVisible}
            onCancel={() => setGroupModalVisible(false)}
            onOk={handleSaveGroup}
            okText={t('common.save', '保存')}
            cancelText={t('common.cancel', '取消')}
            width={500}
          >
            <Form form={groupForm} layout="vertical">
              <Form.Item 
                name="name" 
                label={t('saltstack.groupName', '分组名称')}
                rules={[
                  { required: true, message: t('saltstack.groupNameRequired', '请输入分组名称') },
                  { max: 100, message: t('saltstack.groupNameMaxLength', '分组名称最多100个字符') },
                ]}
              >
                <Input placeholder={t('saltstack.groupNamePlaceholder', '如：compute、gpu、storage')} />
              </Form.Item>
              <Form.Item 
                name="description" 
                label={t('saltstack.groupDescription', '描述')}
                rules={[
                  { max: 500, message: t('saltstack.groupDescMaxLength', '描述最多500个字符') },
                ]}
              >
                <Input.TextArea 
                  placeholder={t('saltstack.groupDescPlaceholder', '分组的用途说明')} 
                  rows={3}
                />
              </Form.Item>
              <Form.Item 
                name="color" 
                label={t('saltstack.groupColor', '标签颜色')}
                initialValue="blue"
              >
                <Select>
                  <Select.Option value="default"><Tag color="default">default</Tag></Select.Option>
                  <Select.Option value="blue"><Tag color="blue">blue</Tag></Select.Option>
                  <Select.Option value="green"><Tag color="green">green</Tag></Select.Option>
                  <Select.Option value="red"><Tag color="red">red</Tag></Select.Option>
                  <Select.Option value="orange"><Tag color="orange">orange</Tag></Select.Option>
                  <Select.Option value="purple"><Tag color="purple">purple</Tag></Select.Option>
                  <Select.Option value="cyan"><Tag color="cyan">cyan</Tag></Select.Option>
                  <Select.Option value="gold"><Tag color="gold">gold</Tag></Select.Option>
                  <Select.Option value="magenta"><Tag color="magenta">magenta</Tag></Select.Option>
                  <Select.Option value="volcano"><Tag color="volcano">volcano</Tag></Select.Option>
                  <Select.Option value="geekblue"><Tag color="geekblue">geekblue</Tag></Select.Option>
                  <Select.Option value="lime"><Tag color="lime">lime</Tag></Select.Option>
                </Select>
              </Form.Item>
            </Form>
          </Modal>
        </Space>
      </Content>
    </Layout>
  );
};

export default SaltStackDashboard;
