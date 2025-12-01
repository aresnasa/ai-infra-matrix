import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Space, Alert, Spin, Button, Layout, Typography, List, Progress, Descriptions, Badge, Tabs, Modal, Form, Input, Select, message, Skeleton, InputNumber, Switch, Divider, Tooltip, Popconfirm, Upload, Dropdown, Menu } from 'antd';
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
  CopyOutlined
} from '@ant-design/icons';
import { saltStackAPI, aiAPI } from '../services/api';
import MinionsTable from '../components/MinionsTable';
import ResizableMetricsPanel from '../components/ResizableMetricsPanel';
import { useI18n } from '../hooks/useI18n';

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
  
  // 页面状态管理
  const [pageLoaded, setPageLoaded] = useState(false);
  
  // 数据状态 - 分别管理loading状态
  const [status, setStatus] = useState(null);
  const [minions, setMinions] = useState([]);
  const [jobs, setJobs] = useState([]);
  
  // 加载状态 - 分别管理每个数据块的加载状态
  const [statusLoading, setStatusLoading] = useState(false);
  const [minionsLoading, setMinionsLoading] = useState(false);
  const [jobsLoading, setJobsLoading] = useState(false);
  
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
    { key: Date.now(), host: '', port: 22, username: 'root', password: '', use_sudo: false }
  ]);
  const batchSseRef = useRef(null);
  
  // 动态并行度信息
  const [parallelInfo, setParallelInfo] = useState({ parallel: 0, percentage: 0, is_auto_calculate: true });
  
  // 文件导入相关状态
  const [importLoading, setImportLoading] = useState(false);

  // SSH 测试弹窗
  const [sshTestVisible, setSSHTestVisible] = useState(false);
  const [sshTestForm] = Form.useForm();
  const [sshTestRunning, setSSHTestRunning] = useState(false);
  const [sshTestResults, setSSHTestResults] = useState([]);
  const [sshTestHosts, setSSHTestHosts] = useState([
    { key: Date.now(), host: '', port: 22, username: 'root', password: '' }
  ]);

  // 删除/卸载 Minion 状态
  const [deletingMinion, setDeletingMinion] = useState(null);
  const [uninstallModalVisible, setUninstallModalVisible] = useState(false);
  const [uninstallForm] = Form.useForm();
  const [uninstallMinionId, setUninstallMinionId] = useState('');

  // 安装任务历史状态
  const [installTasks, setInstallTasks] = useState([]);
  const [installTasksLoading, setInstallTasksLoading] = useState(false);
  const [installTasksTotal, setInstallTasksTotal] = useState(0);
  const [installTasksPage, setInstallTasksPage] = useState({ current: 1, pageSize: 10 });
  const [expandedTaskId, setExpandedTaskId] = useState(null);

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

  const loadMinions = async () => {
    setMinionsLoading(true);
    try {
      // 并行获取 Minion 列表和待删除状态
      const [minionsRes, pendingDeletesRes] = await Promise.all([
        saltStackAPI.getMinions(),
        saltStackAPI.getPendingDeleteMinions().catch(() => ({ data: { minion_ids: [] } })),
      ]);
      
      const minionList = minionsRes.data?.data || [];
      const pendingDeleteIds = new Set(pendingDeletesRes.data?.minion_ids || []);
      
      // 标记待删除的 Minion
      const minionsWithDeleteStatus = minionList.map(minion => ({
        ...minion,
        pending_delete: pendingDeleteIds.has(minion.id || minion.name),
        status: pendingDeleteIds.has(minion.id || minion.name) ? 'deleting' : minion.status,
      }));
      
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
      setJobs(response.data?.data || []);
      setDemo(prev => prev || Boolean(response.data?.demo));
    } catch (e) {
      console.error('加载SaltStack Jobs失败', e);
    } finally {
      setJobsLoading(false);
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

  const loadAllData = async () => {
    // 先加载 master 状态，确保 SaltStack 服务可用
    await loadStatus();
    // 然后并行加载 minion 列表和 jobs
    await Promise.all([loadMinions(), loadJobs()]);
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
      { key: Date.now(), host: '', port: 22, username: 'root', password: '', use_sudo: false }
    ]);
  };

  // 复制第一行配置到当前行（仅复制端口、用户名、密码、sudo 配置，不复制 host）
  const copyFirstRowConfig = (targetKey) => {
    if (batchInstallHosts.length === 0) return;
    const firstRow = batchInstallHosts[0];
    setBatchInstallHosts(batchInstallHosts.map(h => 
      h.key === targetKey ? { 
        ...h, 
        port: firstRow.port, 
        username: firstRow.username, 
        password: firstRow.password, 
        use_sudo: firstRow.use_sudo 
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

  // 更新主机行
  const updateHostRow = (key, field, value) => {
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

  // 导入主机文件
  const handleFileImport = async (file) => {
    setImportLoading(true);
    try {
      const content = await file.text();
      const response = await fetch('/api/saltstack/hosts/parse', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content, filename: file.name })
      });
      
      const result = await response.json();
      if (!result.success) {
        throw new Error(result.message || result.error || t('saltstack.parseFailed'));
      }

      const hosts = result.data?.hosts || [];
      if (hosts.length === 0) {
        message.warning(t('saltstack.noValidHostConfig'));
        return;
      }

      // 将解析的主机添加到列表
      const newHosts = hosts.map((h, idx) => ({
        key: Date.now() + idx,
        host: h.host || '',
        port: h.port || 22,
        username: h.username || 'root',
        password: h.password || '',
        use_sudo: h.use_sudo || false,
        minion_id: h.minion_id || '',
        group: h.group || ''
      }));

      // 如果当前只有一个空行，则替换；否则追加
      if (batchInstallHosts.length === 1 && !batchInstallHosts[0].host) {
        setBatchInstallHosts(newHosts);
      } else {
        setBatchInstallHosts([...batchInstallHosts, ...newHosts]);
      }

      message.success(t('saltstack.importedHosts', { count: hosts.length }));
    } catch (e) {
      message.error(t('saltstack.importFailed') + ': ' + e.message);
    } finally {
      setImportLoading(false);
    }
    return false; // 阻止默认上传行为
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
          sudo_pass: h.password  // Linux 用户密码即 sudo 密码
        })),
        parallel: values.parallel || 0, // 0 表示自动计算并行度
        master_host: values.master_host || 'salt',
        install_type: values.install_type || 'saltstack',
        auto_accept: values.auto_accept ?? true
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

  // ========== Minion 删除/卸载相关函数 ==========

  // 删除 Minion（仅从 Salt Master 删除密钥，支持强制删除）
  const handleDeleteMinion = async (minionId, force = false) => {
    setDeletingMinion(minionId);
    try {
      const resp = await saltStackAPI.removeMinionKey(minionId, force);
      if (resp.data?.success) {
        message.success(t('saltstack.minionDeleted', { id: minionId }));
        loadMinions(); // 刷新列表
      } else {
        message.error(resp.data?.error || t('saltstack.deleteMinionFailed'));
      }
    } catch (e) {
      message.error(t('saltstack.deleteMinionFailed') + ': ' + (e?.response?.data?.message || e.message));
    } finally {
      setDeletingMinion(null);
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
      message.error(t('saltstack.uninstallFailed') + ': ' + (e?.response?.data?.message || e.message));
    }
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
    <Layout style={{ minHeight: '100vh', background: '#f0f2f5' }}>
      <Content style={{ padding: 24 }}>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div>
            <Title level={2}>
              <ThunderboltOutlined style={{ marginRight: 8, color: '#1890ff' }} />
              {t('saltstack.title')}
            </Title>
            <Paragraph type="secondary">
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
              <Card>
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
              <Card>
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
              <Card>
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
              <Card>
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
          <Card>
            <Tabs 
              defaultActiveKey="overview" 
              size="large"
              onChange={(key) => {
                if (key === 'install-tasks' && installTasks.length === 0 && !installTasksLoading) {
                  loadInstallTasks(1);
                }
              }}
            >
              <TabPane tab={t('saltstack.systemOverview')} key="overview" icon={<DatabaseOutlined />}>
                <Row gutter={16}>
                  <Col span={24}>
                    <Card title={t('saltstack.masterInfo')} size="small" loading={statusLoading} style={{ marginBottom: 16 }}>
                      <Descriptions size="small" column={4}>
                        <Descriptions.Item label={t('saltstack.saltVersion')}>
                          {status?.salt_version || (statusLoading ? t('common.loading') : t('minions.status.unknown'))}
                        </Descriptions.Item>
                        <Descriptions.Item label={t('saltstack.uptime')}>
                          {status?.uptime || (statusLoading ? t('common.loading') : t('minions.status.unknown'))}
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
                
                {/* 可调整大小的性能指标面板 */}
                <ResizableMetricsPanel
                  title={t('saltstack.performanceMetrics')}
                  loading={statusLoading || minionsLoading}
                  minHeight={200}
                  maxHeight={600}
                  defaultHeight={350}
                  nodes={[
                    // Master 节点
                    {
                      id: 'salt-master',
                      name: 'Salt Master',
                      metrics: {
                        status: status?.master_status === 'running' ? 'online' : 'offline',
                        cpu_usage: status?.cpu_usage || 0,
                        memory_usage: status?.memory_usage || 0,
                        active_connections: status?.active_connections || 0,
                        network_bandwidth: status?.network_bandwidth || 0,
                        ib_status: 'N/A',
                        roce_status: 'N/A',
                        gpu_utilization: 0,
                        gpu_memory: 0,
                      },
                    },
                    // Minion 节点 (从 minions 数据动态生成)
                    ...minions.map(minion => ({
                      id: minion.id || minion.name,
                      name: minion.id || minion.name,
                      metrics: {
                        status: minion.status?.toLowerCase() === 'up' || minion.status?.toLowerCase() === 'online' ? 'online' : 'offline',
                        cpu_usage: minion.cpu_usage || 0,
                        memory_usage: minion.memory_usage || 0,
                        active_connections: minion.active_connections || 0,
                        network_bandwidth: minion.network_bandwidth || 0,
                        ib_status: minion.ib_status || 'N/A',
                        roce_status: minion.roce_status || 'N/A',
                        gpu_utilization: minion.gpu_utilization || 0,
                        gpu_memory: minion.gpu_memory || 0,
                      },
                    })),
                  ]}
                  onRefresh={loadAllData}
                />
              </TabPane>

              <TabPane tab={t('saltstack.minionsManagement')} key="minions" icon={<DesktopOutlined />}>
                <MinionsTable
                  minions={minions}
                  loading={minionsLoading}
                  onRefresh={loadMinions}
                  onDelete={handleDeleteMinion}
                  onBatchDelete={async (minionIds, force = false) => {
                    // 批量删除，支持强制删除选项
                    try {
                      const resp = await saltStackAPI.batchRemoveMinionKeys(minionIds, force);
                      if (resp.data?.success) {
                        message.success(t('saltstack.batchDeleteSuccess', { count: resp.data?.success_count || minionIds.length }));
                      } else if (resp.data?.failed_count > 0) {
                        message.warning(t('saltstack.batchDeletePartial', { 
                          success: resp.data?.success_count || 0, 
                          failed: resp.data?.failed_count || 0 
                        }));
                      }
                      loadMinions(); // 刷新列表
                    } catch (e) {
                      message.error(t('saltstack.batchDeleteFailed') + ': ' + (e?.response?.data?.message || e.message));
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
                    <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Text type="secondary">{t('saltstack.total', { count: jobs.length })}</Text>
                      <Button 
                        icon={<ReloadOutlined />} 
                        onClick={loadJobs} 
                        loading={jobsLoading}
                      >
                        {t('common.refresh')}
                      </Button>
                    </div>
                    <Table
                      dataSource={jobs}
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
                          title: t('saltstack.function'),
                          dataIndex: 'function',
                          key: 'function',
                          width: 200,
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
                      <Button 
                        icon={<ReloadOutlined />} 
                        onClick={() => loadInstallTasks(1)} 
                        loading={installTasksLoading}
                      >
                        {t('common.refresh')}
                      </Button>
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
            </Tabs>
          </Card>

          {/* 操作按钮 */}
          <Card>
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

            <Card size="small" title={t('saltstack.executeProgress')} style={{ marginTop: 12 }}>
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
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    💡 {t('saltstack.sudoHint')}
                  </Text>
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
                    beforeUpload={handleFileImport}
                    disabled={importLoading}
                  >
                    <Button type="link" size="small" icon={<UploadOutlined />} loading={importLoading}>
                      {t('saltstack.importFile')}
                    </Button>
                  </Upload>
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
                    <Col span={5}>
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
                    <Col span={4}>
                      <Input 
                        placeholder={t('saltstack.usernamePlaceholder')} 
                        value={host.username}
                        onChange={(e) => updateHostRow(host.key, 'username', e.target.value)}
                      />
                    </Col>
                    <Col span={6}>
                      <Input.Password 
                        placeholder={t('saltstack.passwordPlaceholder')} 
                        value={host.password}
                        onChange={(e) => updateHostRow(host.key, 'password', e.target.value)}
                      />
                    </Col>
                    <Col span={4}>
                      <Space>
                        <Tooltip title={t('saltstack.useSudo')}>
                          <Switch 
                            size="small"
                            checked={host.use_sudo}
                            onChange={(v) => updateHostRow(host.key, 'use_sudo', v)}
                          />
                        </Tooltip>
                        <span style={{ fontSize: 12 }}>sudo</span>
                      </Space>
                    </Col>
                    <Col span={1}>
                      <Button 
                        type="text" 
                        danger 
                        icon={<DeleteOutlined />} 
                        onClick={() => removeHostRow(host.key)}
                        disabled={batchInstallHosts.length <= 1}
                      />
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
            <Card size="small" title={t('saltstack.installProgress')} style={{ marginTop: 16 }}>
              {batchInstallTaskId && (
                <div style={{ marginBottom: 8 }}>
                  <Text type="secondary">{t('saltstack.taskId')}: </Text>
                  <Text copyable>{batchInstallTaskId}</Text>
                </div>
              )}
              <div style={{ maxHeight: 280, overflow: 'auto', background: '#0b1021', color: '#e6e6e6', padding: 12, borderRadius: 6 }}>
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
              <Card size="small" title={t('saltstack.result')} style={{ marginTop: 16 }}>
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
        </Space>
      </Content>
    </Layout>
  );
};

export default SaltStackDashboard;
