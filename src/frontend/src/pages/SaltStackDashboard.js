import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Space, Alert, Spin, Button, Layout, Typography, List, Progress, Descriptions, Badge, Tabs, Modal, Form, Input, Select, message, Skeleton, InputNumber, Switch, Divider, Tooltip, Popconfirm } from 'antd';
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
  QuestionCircleOutlined,
  SafetyCertificateOutlined,
  WifiOutlined,
  KeyOutlined,
  LockOutlined
} from '@ant-design/icons';
import { saltStackAPI, aiAPI } from '../services/api';

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
    { id: 'nginx', name: 'Nginx 配置', desc: '安装和配置 Nginx Web 服务器' },
    { id: 'mysql', name: 'MySQL 配置', desc: '安装和配置 MySQL 数据库' },
    { id: 'docker', name: 'Docker 配置', desc: '安装和配置 Docker 容器引擎' },
    { id: 'firewall', name: '防火墙配置', desc: '配置系统防火墙规则' },
    { id: 'user', name: '用户管理', desc: '添加、删除和管理系统用户' },
  ]);

  // 批量安装 Salt Minion 弹窗
  const [batchInstallVisible, setBatchInstallVisible] = useState(false);
  const [batchInstallForm] = Form.useForm();
  const [batchInstallRunning, setBatchInstallRunning] = useState(false);
  const [batchInstallTaskId, setBatchInstallTaskId] = useState('');
  const [batchInstallEvents, setBatchInstallEvents] = useState([]);
  const [batchInstallHosts, setBatchInstallHosts] = useState([
    { key: Date.now(), host: '', port: 22, username: 'root', password: '', use_sudo: false, sudo_pass: '' }
  ]);
  const batchSseRef = useRef(null);

  // SSH 测试弹窗
  const [sshTestVisible, setSSHTestVisible] = useState(false);
  const [sshTestForm] = Form.useForm();
  const [sshTestRunning, setSSHTestRunning] = useState(false);
  const [sshTestResults, setSSHTestResults] = useState([]);
  const [sshTestHosts, setSSHTestHosts] = useState([
    { key: Date.now(), host: '', port: 22, username: 'root', password: '', sudo_pass: '' }
  ]);

  // 删除/卸载 Minion 状态
  const [deletingMinion, setDeletingMinion] = useState(null);
  const [uninstallModalVisible, setUninstallModalVisible] = useState(false);
  const [uninstallForm] = Form.useForm();
  const [uninstallMinionId, setUninstallMinionId] = useState('');

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
      const response = await saltStackAPI.getMinions();
      setMinions(response.data?.data || []);
      setDemo(prev => prev || Boolean(response.data?.demo));
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

  const loadAllData = async () => {
    // 并行加载所有数据，但不阻塞页面渲染
    await Promise.all([loadStatus(), loadMinions(), loadJobs()]);
  };

  // 页面初始化效果 - 立即显示静态内容
  useEffect(() => {
    // 标记页面已加载，显示静态内容
    setPageLoaded(true);
    
    // 异步加载数据（非阻塞）
    setTimeout(() => {
      loadAllData();
    }, 100); // 延迟100ms让静态内容先渲染
    
    // 设置定时刷新（60秒）
    const interval = setInterval(loadAllData, 60000);
    return () => clearInterval(interval);
  }, []);

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
      { key: Date.now(), host: '', port: 22, username: 'root', password: '', use_sudo: false, sudo_pass: '' }
    ]);
  };

  // 删除主机行
  const removeHostRow = (key) => {
    if (batchInstallHosts.length <= 1) {
      message.warning('至少保留一个主机');
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

  // 打开批量安装弹窗
  const openBatchInstallModal = () => {
    setBatchInstallVisible(true);
    setBatchInstallEvents([]);
    setBatchInstallTaskId('');
    setBatchInstallRunning(false);
    setBatchInstallHosts([
      { key: Date.now(), host: '', port: 22, username: 'root', password: '', use_sudo: false, sudo_pass: '' }
    ]);
    batchInstallForm.setFieldsValue({
      parallel: 3,
      master_host: 'salt',
      install_type: 'saltstack',
      auto_accept: true,
      global_use_sudo: false,
      global_sudo_pass: ''
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
        
        if (data.type === 'complete' || data.type === 'error' || data.type === 'closed') {
          setTimeout(() => {
            setBatchInstallRunning(false);
            closeBatchSSE();
            // 刷新 minions 列表
            loadMinions();
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
    };
  };

  // 执行批量安装
  const handleBatchInstall = async () => {
    try {
      const values = await batchInstallForm.validateFields();
      
      // 验证主机列表
      const validHosts = batchInstallHosts.filter(h => h.host && h.host.trim());
      if (validHosts.length === 0) {
        message.error('请至少添加一个有效的主机');
        return;
      }

      // 检查必填字段
      for (const h of validHosts) {
        if (!h.username || !h.password) {
          message.error(`主机 ${h.host} 缺少用户名或密码`);
          return;
        }
      }

      setBatchInstallRunning(true);
      setBatchInstallEvents([]);

      // 构建请求
      const payload = {
        hosts: validHosts.map(h => ({
          host: h.host.trim(),
          port: h.port || 22,
          username: h.username,
          password: h.password,
          use_sudo: values.global_use_sudo || h.use_sudo,
          sudo_pass: (values.global_use_sudo ? values.global_sudo_pass : h.sudo_pass) || h.password
        })),
        parallel: values.parallel || 3,
        master_host: values.master_host || 'salt',
        install_type: values.install_type || 'saltstack',
        auto_accept: values.auto_accept ?? true
      };

      const resp = await saltStackAPI.batchInstallMinion(payload);
      
      if (!resp.data?.success) {
        message.error(resp.data?.message || '启动批量安装失败');
        setBatchInstallRunning(false);
        return;
      }

      const taskId = resp.data?.task_id;
      if (!taskId) {
        message.error('未返回任务ID');
        setBatchInstallRunning(false);
        return;
      }

      setBatchInstallTaskId(taskId);
      message.success(`批量安装任务已创建: ${taskId}`);
      startBatchInstallSSE(taskId);
    } catch (e) {
      message.error('提交批量安装失败: ' + (e?.response?.data?.message || e.message));
      setBatchInstallRunning(false);
    }
  };

  // ========== SSH 测试相关函数 ==========
  
  // 打开 SSH 测试弹窗
  const openSSHTestModal = () => {
    setSSHTestVisible(true);
    setSSHTestResults([]);
    setSSHTestHosts([
      { key: Date.now(), host: '', port: 22, username: 'root', password: '', sudo_pass: '' }
    ]);
  };

  // 添加 SSH 测试主机行
  const addSSHTestHostRow = () => {
    setSSHTestHosts([
      ...sshTestHosts,
      { key: Date.now(), host: '', port: 22, username: 'root', password: '', sudo_pass: '' }
    ]);
  };

  // 删除 SSH 测试主机行
  const removeSSHTestHostRow = (key) => {
    if (sshTestHosts.length <= 1) {
      message.warning('至少保留一个主机');
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
      message.error('请至少添加一个有效的主机');
      return;
    }

    for (const h of validHosts) {
      if (!h.username || !h.password) {
        message.error(`主机 ${h.host} 缺少用户名或密码`);
        return;
      }
    }

    setSSHTestRunning(true);
    setSSHTestResults([]);

    try {
      const payload = {
        hosts: validHosts.map(h => ({
          host: h.host.trim(),
          port: h.port || 22,
          username: h.username,
          password: h.password,
          sudo_pass: h.sudo_pass || h.password
        })),
        parallel: 5
      };

      const resp = await saltStackAPI.batchTestSSH(payload);
      
      if (resp.data?.success) {
        setSSHTestResults(resp.data.data?.results || []);
        message.success(`测试完成：${resp.data.data?.connected_count}/${resp.data.data?.total} 连接成功，${resp.data.data?.sudo_count} 有 sudo 权限`);
      } else {
        message.error(resp.data?.error || 'SSH 测试失败');
      }
    } catch (e) {
      message.error('SSH 测试失败: ' + (e?.response?.data?.message || e.message));
    } finally {
      setSSHTestRunning(false);
    }
  };

  // ========== Minion 删除/卸载相关函数 ==========

  // 删除 Minion（仅从 Salt Master 删除密钥）
  const handleDeleteMinion = async (minionId) => {
    setDeletingMinion(minionId);
    try {
      const resp = await saltStackAPI.removeMinionKey(minionId);
      if (resp.data?.success) {
        message.success(`Minion ${minionId} 已从 Salt Master 删除`);
        loadMinions(); // 刷新列表
      } else {
        message.error(resp.data?.error || '删除失败');
      }
    } catch (e) {
      message.error('删除 Minion 失败: ' + (e?.response?.data?.message || e.message));
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
      use_sudo: false,
      sudo_pass: ''
    });
  };

  // 执行卸载 Minion
  const handleUninstallMinion = async () => {
    try {
      const values = await uninstallForm.validateFields();
      
      const resp = await saltStackAPI.uninstallMinion(uninstallMinionId, {
        host: values.host,
        port: values.port || 22,
        username: values.username,
        password: values.password,
        use_sudo: values.use_sudo,
        sudo_pass: values.sudo_pass || values.password
      });

      if (resp.data?.success) {
        message.success(`Minion ${uninstallMinionId} 已卸载并从 Master 删除`);
        setUninstallModalVisible(false);
        loadMinions();
      } else {
        message.error(resp.data?.error || '卸载失败');
      }
    } catch (e) {
      message.error('卸载 Minion 失败: ' + (e?.response?.data?.message || e.message));
    }
  };

  useEffect(() => {
    return () => {
      closeSSE();
      closeBatchSSE();
    };
  }, []);

  const validateClientSide = (language, code) => {
    if (!code || !code.trim()) return '请输入要执行的代码';
    if (code.length > 20000) return '代码过长，最大20000字符';
    // 简单引号平衡检查
    let single = 0, dbl = 0;
    for (let i = 0; i < code.length; i++) {
      const ch = code[i];
      if (ch === '\'') single ^= 1; else if (ch === '"') dbl ^= 1;
    }
    if (single || dbl) return '引号不平衡，请检查代码';
    if (language === 'python') {
      const lines = code.split('\n');
      for (const ln of lines) {
        if (ln.startsWith('\t') && ln.trimStart().startsWith(' ')) return 'Python 缩进混用制表符与空格';
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
      const prompt = `为 Salt 下发的${lang}脚本提供补全建议，仅给出代码片段，不要解释。`;
      await aiAPI.quickChat(prompt, 'salt-exec-suggest'); // 预留：后端应返回异步消息ID，这里仅调用以示占位
      message.info('已发送智能补全请求（占位），后端实现后将展示建议');
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
        message.error('未返回操作ID');
        setExecRunning(false);
        return;
      }
      setExecOpId(opId);
      startSSE(opId);
    } catch (e) {
      message.error('提交执行失败: ' + (e?.response?.data?.error || e.message));
      setExecRunning(false);
    }
  };

  const execFooter = (
    <Space>
      <Button onClick={() => setExecVisible(false)} disabled={execRunning}>关闭</Button>
      <Button onClick={handleSuggest} disabled={execRunning}>智能补全（预留）</Button>
      <Button type="primary" onClick={handleExecute} loading={execRunning}>执行</Button>
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
        <div style={{ marginTop: 16 }}>初始化SaltStack界面...</div>
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
              SaltStack 配置管理
            </Title>
            <Paragraph type="secondary">
              基于 SaltStack 的基础设施自动化配置管理系统
            </Paragraph>
          </div>

          {error && (
            <Alert 
              type="error" 
              showIcon 
              message="无法连接到SaltStack"
              description={
                <Space>
                  <span>请确认SaltStack服务正在运行且后端API可达。</span>
                  <Button size="small" onClick={loadAllData}>重试</Button>
                </Space>
              }
            />
          )}

          {demo && (
            <Alert 
              type="info" 
              showIcon 
              message="演示模式" 
              description="显示SaltStack演示数据"
            />
          )}

          {/* 数据加载进度提示 */}
          {(statusLoading || minionsLoading || jobsLoading) && (
            <Alert 
              type="info" 
              showIcon 
              message="正在加载数据" 
              description={
                <Space>
                  <span>
                    状态数据: {statusLoading ? '加载中...' : '✓'} | 
                    Minions: {minionsLoading ? '加载中...' : '✓'} | 
                    作业历史: {jobsLoading ? '加载中...' : '✓'}
                  </span>
                </Space>
              }
            />
          )}

          {/* 状态概览 - 每个卡片独立加载 */}
          <Row gutter={16}>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="Master状态" 
                  value={status?.master_status || (statusLoading ? '加载中...' : '未知')} 
                  prefix={<SettingOutlined />}
                  valueStyle={{ 
                    color: statusLoading ? '#999' : (status?.master_status === 'running' ? '#3f8600' : '#cf1322') 
                  }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="在线Minions" 
                  value={status?.minions_up || (statusLoading ? '...' : 0)} 
                  prefix={<DesktopOutlined />}
                  valueStyle={{ color: statusLoading ? '#999' : '#3f8600' }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="离线Minions" 
                  value={status?.minions_down || (statusLoading ? '...' : 0)} 
                  prefix={<ExclamationCircleOutlined />}
                  valueStyle={{ color: statusLoading ? '#999' : '#cf1322' }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="API状态" 
                  value={status?.api_status || (statusLoading ? '检测中...' : '未知')} 
                  prefix={<ApiOutlined />}
                  valueStyle={{ 
                    color: statusLoading ? '#999' : (status?.api_status === 'running' ? '#3f8600' : '#cf1322') 
                  }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
          </Row>

          {/* 详细信息选项卡 */}
          <Card>
            <Tabs defaultActiveKey="overview" size="large">
              <TabPane tab="系统概览" key="overview" icon={<DatabaseOutlined />}>
                <Row gutter={16}>
                  <Col span={12}>
                    <Card title="Master信息" size="small" loading={statusLoading}>
                      <Descriptions size="small" column={1}>
                        <Descriptions.Item label="版本">
                          {status?.salt_version || (statusLoading ? '加载中...' : '未知')}
                        </Descriptions.Item>
                        <Descriptions.Item label="启动时间">
                          {status?.uptime || (statusLoading ? '获取中...' : '未知')}
                        </Descriptions.Item>
                        <Descriptions.Item label="配置文件">
                          {status?.config_file || '/etc/salt/master'}
                        </Descriptions.Item>
                        <Descriptions.Item label="日志级别">
                          <Tag color="blue">{status?.log_level || 'info'}</Tag>
                        </Descriptions.Item>
                      </Descriptions>
                    </Card>
                  </Col>
                  <Col span={12}>
                    <Card title="性能指标" size="small" loading={statusLoading}>
                      {statusLoading ? (
                        <div style={{ textAlign: 'center', padding: '20px 0' }}>
                          <Spin size="small" />
                          <div style={{ marginTop: 8 }}>正在获取性能数据...</div>
                        </div>
                      ) : (
                        <Space direction="vertical" style={{ width: '100%' }}>
                          <div>
                            <Text>CPU使用率</Text>
                            <Progress 
                              percent={status?.cpu_usage || 0} 
                              size="small" 
                              status={status?.cpu_usage > 80 ? 'exception' : 'active'}
                            />
                          </div>
                          <div>
                            <Text>内存使用率</Text>
                            <Progress 
                              percent={status?.memory_usage || 0} 
                              size="small" 
                              status={status?.memory_usage > 85 ? 'exception' : 'active'}
                            />
                          </div>
                          <div>
                            <Text>活跃连接数</Text>
                            <Progress 
                              percent={Math.min((status?.active_connections || 0) / 100 * 100, 100)} 
                              size="small"
                              showInfo={false}
                            />
                            <Text type="secondary"> {status?.active_connections || 0}/100</Text>
                          </div>
                        </Space>
                      )}
                    </Card>
                  </Col>
                </Row>
              </TabPane>

              <TabPane tab="Minions管理" key="minions" icon={<DesktopOutlined />}>
                {minionsLoading ? (
                  <div style={{ textAlign: 'center', padding: '60px 0' }}>
                    <Spin size="large" />
                    <div style={{ marginTop: 16 }}>正在加载Minions数据...</div>
                  </div>
                ) : (
                  <>
                    <List
                      grid={{ gutter: 16, column: 2 }}
                      dataSource={minions}
                      renderItem={minion => (
                        <List.Item>
                          <Card 
                            size="small" 
                            title={
                              <Space>
                                <Badge status={getStatusColor(minion.status)} />
                                {minion.id || minion.name}
                              </Space>
                            }
                            extra={
                              <Space>
                                <Tag color={getStatusColor(minion.status)}>
                                  {minion.status || '未知'}
                                </Tag>
                                <Tooltip title="卸载 Minion（SSH远程卸载）">
                                  <Button 
                                    type="text" 
                                    size="small" 
                                    icon={<SettingOutlined />}
                                    onClick={() => openUninstallModal(minion.id || minion.name)}
                                  />
                                </Tooltip>
                                <Popconfirm
                                  title="删除 Minion"
                                  description="确定要从 Salt Master 删除此 Minion 密钥吗？这不会卸载目标机器上的 Salt Minion 软件。"
                                  onConfirm={() => handleDeleteMinion(minion.id || minion.name)}
                                  okText="确定删除"
                                  cancelText="取消"
                                >
                                  <Tooltip title="删除 Minion 密钥">
                                    <Button 
                                      type="text" 
                                      size="small" 
                                      danger
                                      icon={<DeleteOutlined />}
                                      loading={deletingMinion === (minion.id || minion.name)}
                                    />
                                  </Tooltip>
                                </Popconfirm>
                              </Space>
                            }
                          >
                            <Descriptions size="small" column={1}>
                              <Descriptions.Item label="操作系统">
                                {minion.os || '未知'}
                              </Descriptions.Item>
                              <Descriptions.Item label="架构">
                                {minion.arch || '未知'}
                              </Descriptions.Item>
                              <Descriptions.Item label="Salt版本">
                                {minion.salt_version || '未知'}
                              </Descriptions.Item>
                              <Descriptions.Item label="最后响应">
                                {minion.last_seen || '未知'}
                              </Descriptions.Item>
                            </Descriptions>
                          </Card>
                        </List.Item>
                      )}
                    />
                    {minions.length === 0 && (
                      <div style={{ textAlign: 'center', padding: '40px 0' }}>
                        <Text type="secondary">暂无Minion数据</Text>
                      </div>
                    )}
                  </>
                )}
              </TabPane>

              <TabPane tab="作业历史" key="jobs" icon={<PlayCircleOutlined />}>
                {jobsLoading ? (
                  <div style={{ textAlign: 'center', padding: '60px 0' }}>
                    <Spin size="large" />
                    <div style={{ marginTop: 16 }}>正在加载作业历史...</div>
                  </div>
                ) : (
                  <>
                    <List
                      dataSource={jobs}
                      renderItem={job => (
                        <List.Item>
                          <Card 
                            size="small" 
                            style={{ width: '100%' }}
                            title={
                              <Space>
                                <Tag color={getJobStatusColor(job.status)}>
                                  {job.status || '未知'}
                                </Tag>
                                <Text strong>{job.function || job.command}</Text>
                              </Space>
                            }
                            extra={
                              <Text type="secondary">
                                {job.timestamp || job.start_time}
                              </Text>
                            }
                          >
                            <Descriptions size="small" column={2}>
                              <Descriptions.Item label="目标">
                                {job.target || '所有节点'}
                              </Descriptions.Item>
                              <Descriptions.Item label="用户">
                                {job.user || 'root'}
                              </Descriptions.Item>
                              <Descriptions.Item label="持续时间">
                                {job.duration || '未知'}
                              </Descriptions.Item>
                              <Descriptions.Item label="返回码">
                                <Tag color={job.return_code === 0 ? 'green' : 'red'}>
                                  {job.return_code ?? '未知'}
                                </Tag>
                              </Descriptions.Item>
                            </Descriptions>
                            {job.result && (
                              <div style={{ marginTop: 8 }}>
                                <Text type="secondary">结果:</Text>
                                <Paragraph 
                                  code 
                                  style={{ 
                                    marginTop: 4, 
                                    marginBottom: 0, 
                                    maxHeight: 100, 
                                    overflow: 'auto' 
                                  }}
                                >
                                  {typeof job.result === 'string' ? job.result : JSON.stringify(job.result, null, 2)}
                                </Paragraph>
                              </div>
                            )}
                          </Card>
                        </List.Item>
                      )}
                    />
                    {jobs.length === 0 && (
                      <div style={{ textAlign: 'center', padding: '40px 0' }}>
                        <Text type="secondary">暂无作业历史</Text>
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
                刷新数据
              </Button>
              <Button 
                icon={<PlayCircleOutlined />}
                onClick={openExecModal}
              >
                执行命令
              </Button>
              <Button 
                icon={<CloudUploadOutlined />}
                onClick={openBatchInstallModal}
                type="primary"
                ghost
              >
                批量安装 Minion
              </Button>
              <Button 
                icon={<WifiOutlined />}
                onClick={openSSHTestModal}
              >
                SSH 测试
              </Button>
              <Button 
                icon={<SettingOutlined />}
                onClick={() => {
                  setConfigVisible(true);
                  configForm.setFieldsValue({ target: '*' });
                }}
              >
                配置管理
              </Button>
            </Space>
          </Card>

          {/* 执行命令弹窗 */}
          <Modal
            title="执行自定义命令（Bash / Python）"
            open={execVisible}
            onCancel={() => { setExecVisible(false); closeSSE(); setExecRunning(false); }}
            footer={execFooter}
            width={900}
          >
            <Form form={execForm} layout="vertical">
              <Form.Item name="target" label="目标节点" rules={[{ required: true, message: '请输入目标，例如 * 或 compute* 或 列表' }]}>
                <Input placeholder="例如: * 或 compute* 或 ai-infra-web-01" />
              </Form.Item>
              <Form.Item name="language" label="语言" rules={[{ required: true }]}> 
                <Select>
                  <Option value="bash">Bash</Option>
                  <Option value="python">Python</Option>
                </Select>
              </Form.Item>
              <Form.Item name="code" label="代码" rules={[{ required: true, message: '请输入要执行的代码' }]}>
                <TextArea rows={10} placeholder="# 在此粘贴脚本..." style={{ fontFamily: 'monospace' }} />
              </Form.Item>
              <Form.Item name="timeout" label="超时（秒）">
                <Input type="number" min={10} max={3600} placeholder="120" />
              </Form.Item>
            </Form>

            <Card size="small" title="执行进度" style={{ marginTop: 12 }}>
              <div style={{ maxHeight: 240, overflow: 'auto', background: '#0b1021', color: '#e6e6e6', padding: 8, borderRadius: 6 }}>
                {execEvents.length === 0 ? (
                  <Text type="secondary">等待执行或无日志...</Text>
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
            title="Salt 配置模板管理"
            open={configVisible}
            onCancel={() => setConfigVisible(false)}
            footer={[
              <Button key="cancel" onClick={() => setConfigVisible(false)}>取消</Button>,
              <Button 
                key="apply" 
                type="primary" 
                onClick={() => {
                  configForm.validateFields().then(values => {
                    message.info(`将应用配置模板: ${values.template} 到目标: ${values.target}`);
                    // TODO: 调用后端 API 应用配置模板
                    // saltStackAPI.applyTemplate({ template: values.template, target: values.target });
                    setConfigVisible(false);
                  });
                }}
              >
                应用配置
              </Button>,
            ]}
            width={700}
          >
            <Form form={configForm} layout="vertical">
              <Form.Item 
                name="target" 
                label="目标节点" 
                rules={[{ required: true, message: '请输入目标节点' }]}
              >
                <Input placeholder="例如: * 或 web* 或 db01" />
              </Form.Item>
              <Form.Item 
                name="template" 
                label="配置模板" 
                rules={[{ required: true, message: '请选择配置模板' }]}
              >
                <Select placeholder="选择要应用的配置模板">
                  {configTemplates.map(t => (
                    <Option key={t.id} value={t.id}>
                      {t.name} - {t.desc}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
              <Alert
                message="提示"
                description="选择配置模板后，将通过 Salt State 在目标节点上应用相应的配置。此功能需要后端 API 支持。"
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
                批量安装 Salt Minion
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
                {batchInstallRunning ? '取消' : '关闭'}
              </Button>,
              <Button 
                key="install" 
                type="primary" 
                onClick={handleBatchInstall}
                loading={batchInstallRunning}
                icon={<CloudUploadOutlined />}
              >
                开始安装
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
                        并发数
                        <Tooltip title="同时安装的主机数量，建议不超过10">
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
                    label="Salt Master 地址"
                    initialValue="salt"
                  >
                    <Input placeholder="例如: salt 或 192.168.1.100" />
                  </Form.Item>
                </Col>
                <Col span={6}>
                  <Form.Item 
                    name="install_type" 
                    label="安装类型"
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
                    label="自动接受 Key"
                    valuePropName="checked"
                    initialValue={true}
                  >
                    <Switch checkedChildren="是" unCheckedChildren="否" />
                  </Form.Item>
                </Col>
              </Row>

              <Divider orientation="left">全局 sudo 设置</Divider>
              <Row gutter={16}>
                <Col span={6}>
                  <Form.Item 
                    name="global_use_sudo" 
                    label={
                      <Space>
                        使用 sudo
                        <Tooltip title="如果使用非 root 用户登录，开启此选项使用 sudo 执行安装命令">
                          <QuestionCircleOutlined />
                        </Tooltip>
                      </Space>
                    }
                    valuePropName="checked"
                    initialValue={false}
                  >
                    <Switch checkedChildren="是" unCheckedChildren="否" />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item 
                    name="global_sudo_pass" 
                    label="sudo 密码（留空则使用登录密码）"
                  >
                    <Input.Password placeholder="如果sudo需要密码" />
                  </Form.Item>
                </Col>
              </Row>

              <Divider orientation="left">
                <Space>
                  目标主机列表
                  <Button type="link" size="small" icon={<PlusOutlined />} onClick={addHostRow}>
                    添加主机
                  </Button>
                </Space>
              </Divider>

              <div style={{ maxHeight: 300, overflow: 'auto' }}>
                {batchInstallHosts.map((host, index) => (
                  <Row gutter={8} key={host.key} style={{ marginBottom: 8 }}>
                    <Col span={5}>
                      <Input 
                        placeholder="主机地址 (IP 或域名)" 
                        value={host.host}
                        onChange={(e) => updateHostRow(host.key, 'host', e.target.value)}
                        addonBefore={`#${index + 1}`}
                      />
                    </Col>
                    <Col span={2}>
                      <InputNumber 
                        placeholder="端口" 
                        value={host.port}
                        onChange={(v) => updateHostRow(host.key, 'port', v)}
                        min={1}
                        max={65535}
                        style={{ width: '100%' }}
                      />
                    </Col>
                    <Col span={4}>
                      <Input 
                        placeholder="用户名" 
                        value={host.username}
                        onChange={(e) => updateHostRow(host.key, 'username', e.target.value)}
                      />
                    </Col>
                    <Col span={5}>
                      <Input.Password 
                        placeholder="密码" 
                        value={host.password}
                        onChange={(e) => updateHostRow(host.key, 'password', e.target.value)}
                      />
                    </Col>
                    <Col span={3}>
                      <Space>
                        <Tooltip title="使用 sudo">
                          <Switch 
                            size="small"
                            checked={host.use_sudo}
                            onChange={(v) => updateHostRow(host.key, 'use_sudo', v)}
                          />
                        </Tooltip>
                        <span style={{ fontSize: 12 }}>sudo</span>
                      </Space>
                    </Col>
                    <Col span={4}>
                      {host.use_sudo && (
                        <Input.Password 
                          placeholder="sudo密码" 
                          value={host.sudo_pass}
                          onChange={(e) => updateHostRow(host.key, 'sudo_pass', e.target.value)}
                          size="small"
                        />
                      )}
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
            </Form>

            {/* 安装进度 */}
            <Card size="small" title="安装进度" style={{ marginTop: 16 }}>
              {batchInstallTaskId && (
                <div style={{ marginBottom: 8 }}>
                  <Text type="secondary">任务ID: </Text>
                  <Text copyable>{batchInstallTaskId}</Text>
                </div>
              )}
              <div style={{ maxHeight: 280, overflow: 'auto', background: '#0b1021', color: '#e6e6e6', padding: 12, borderRadius: 6 }}>
                {batchInstallEvents.length === 0 ? (
                  <Text type="secondary">等待开始安装...</Text>
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
                SSH 连接测试（含 sudo 权限检查）
              </Space>
            }
            open={sshTestVisible}
            onCancel={() => setSSHTestVisible(false)}
            footer={[
              <Button key="cancel" onClick={() => setSSHTestVisible(false)}>
                关闭
              </Button>,
              <Button 
                key="test" 
                type="primary" 
                onClick={handleSSHTest}
                loading={sshTestRunning}
                icon={<SafetyCertificateOutlined />}
              >
                开始测试
              </Button>,
            ]}
            width={1000}
            destroyOnClose
          >
            <Alert
              message="SSH 测试说明"
              description="此功能将测试 SSH 连接是否成功，并检查是否有 sudo 权限。这对于批量安装前的预检非常有用。"
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <Divider orientation="left">
              <Space>
                目标主机列表
                <Button type="link" size="small" icon={<PlusOutlined />} onClick={addSSHTestHostRow}>
                  添加主机
                </Button>
              </Space>
            </Divider>

            <div style={{ maxHeight: 250, overflow: 'auto' }}>
              {sshTestHosts.map((host, index) => (
                <Row gutter={8} key={host.key} style={{ marginBottom: 8 }}>
                  <Col span={5}>
                    <Input 
                      placeholder="主机地址" 
                      value={host.host}
                      onChange={(e) => updateSSHTestHostRow(host.key, 'host', e.target.value)}
                      addonBefore={`#${index + 1}`}
                    />
                  </Col>
                  <Col span={2}>
                    <InputNumber 
                      placeholder="端口" 
                      value={host.port}
                      onChange={(v) => updateSSHTestHostRow(host.key, 'port', v)}
                      min={1}
                      max={65535}
                      style={{ width: '100%' }}
                    />
                  </Col>
                  <Col span={4}>
                    <Input 
                      placeholder="用户名" 
                      value={host.username}
                      onChange={(e) => updateSSHTestHostRow(host.key, 'username', e.target.value)}
                    />
                  </Col>
                  <Col span={5}>
                    <Input.Password 
                      placeholder="密码" 
                      value={host.password}
                      onChange={(e) => updateSSHTestHostRow(host.key, 'password', e.target.value)}
                    />
                  </Col>
                  <Col span={5}>
                    <Input.Password 
                      placeholder="sudo密码（可选）" 
                      value={host.sudo_pass}
                      onChange={(e) => updateSSHTestHostRow(host.key, 'sudo_pass', e.target.value)}
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
              <Card size="small" title="测试结果" style={{ marginTop: 16 }}>
                <Table
                  dataSource={sshTestResults}
                  rowKey="host"
                  size="small"
                  pagination={false}
                  columns={[
                    {
                      title: '主机',
                      dataIndex: 'host',
                      width: 150,
                    },
                    {
                      title: '连接状态',
                      dataIndex: 'connected',
                      width: 100,
                      render: (v) => v ? 
                        <Tag color="success" icon={<CheckCircleOutlined />}>连接成功</Tag> : 
                        <Tag color="error" icon={<ExclamationCircleOutlined />}>连接失败</Tag>
                    },
                    {
                      title: '认证方式',
                      dataIndex: 'auth_method',
                      width: 100,
                      render: (v) => v ? <Tag icon={<KeyOutlined />}>{v}</Tag> : '-'
                    },
                    {
                      title: 'sudo 权限',
                      dataIndex: 'has_sudo',
                      width: 120,
                      render: (v, record) => v ? 
                        <Tag color="success" icon={<LockOutlined />}>
                          {record.sudo_no_password ? '免密sudo' : '需要密码'}
                        </Tag> : 
                        <Tag color="warning">无sudo</Tag>
                    },
                    {
                      title: '主机名',
                      dataIndex: 'hostname',
                      width: 150,
                    },
                    {
                      title: '操作系统',
                      dataIndex: 'os_info',
                      ellipsis: true,
                    },
                    {
                      title: '耗时(ms)',
                      dataIndex: 'duration',
                      width: 80,
                    },
                    {
                      title: '错误',
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
                卸载 Minion: {uninstallMinionId}
              </Space>
            }
            open={uninstallModalVisible}
            onCancel={() => setUninstallModalVisible(false)}
            onOk={handleUninstallMinion}
            okText="确认卸载"
            okButtonProps={{ danger: true }}
            cancelText="取消"
            width={600}
          >
            <Alert
              message="警告"
              description="此操作将通过 SSH 连接到目标主机，卸载 Salt Minion 软件包并清理配置文件，同时从 Salt Master 删除该 Minion 的密钥。"
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <Form form={uninstallForm} layout="vertical">
              <Row gutter={16}>
                <Col span={16}>
                  <Form.Item 
                    name="host" 
                    label="主机地址"
                    rules={[{ required: true, message: '请输入主机地址' }]}
                  >
                    <Input placeholder="IP 或域名" />
                  </Form.Item>
                </Col>
                <Col span={8}>
                  <Form.Item name="port" label="端口" initialValue={22}>
                    <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item 
                    name="username" 
                    label="用户名"
                    rules={[{ required: true, message: '请输入用户名' }]}
                  >
                    <Input placeholder="例如: root" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item 
                    name="password" 
                    label="密码"
                    rules={[{ required: true, message: '请输入密码' }]}
                  >
                    <Input.Password placeholder="SSH 登录密码" />
                  </Form.Item>
                </Col>
              </Row>
              <Row gutter={16}>
                <Col span={8}>
                  <Form.Item 
                    name="use_sudo" 
                    label="使用 sudo"
                    valuePropName="checked"
                  >
                    <Switch checkedChildren="是" unCheckedChildren="否" />
                  </Form.Item>
                </Col>
                <Col span={16}>
                  <Form.Item name="sudo_pass" label="sudo 密码（留空则使用登录密码）">
                    <Input.Password placeholder="如果 sudo 需要密码" />
                  </Form.Item>
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
