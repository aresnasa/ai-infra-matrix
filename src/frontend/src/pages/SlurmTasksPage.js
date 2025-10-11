import React, { useEffect, useState, useRef } from 'react';
import {
  Card, Row, Col, Table, Tag, Space, Alert, Spin, Button,
  Typography, Progress, Descriptions, Badge, Tooltip, Modal,
  List, Divider, message, Select, DatePicker, Input, Popconfirm,
  Statistic, Timeline, Tabs, Empty
} from 'antd';
import {
  ReloadOutlined, EyeOutlined, CheckCircleOutlined,
  ExclamationCircleOutlined, ClockCircleOutlined, PlayCircleOutlined,
  StopOutlined, ThunderboltOutlined, DesktopOutlined, BarChartOutlined,
  HistoryOutlined, RedoOutlined, SearchOutlined, FilterOutlined,
  DeleteOutlined, InfoCircleOutlined
} from '@ant-design/icons';
import { slurmAPI } from '../services/api';
import { useLocation, useNavigate } from 'react-router-dom';
import dayjs from 'dayjs';

const { Title, Text, Paragraph } = Typography;

// 任务状态颜色映射
const getTaskStatusColor = (status) => {
  switch (status) {
    case 'pending': return 'default';
    case 'running': return 'blue';
    case 'completed': return 'green';
    case 'failed': return 'red';
    case 'cancelled': return 'orange';
    // 兼容旧状态
    case 'complete': return 'green';
    default: return 'default';
  }
};

// 任务状态图标映射
const getTaskStatusIcon = (status) => {
  switch (status) {
    case 'pending': return <PlayCircleOutlined />;
    case 'running': return <ClockCircleOutlined />;
    case 'completed': return <CheckCircleOutlined />;
    case 'failed': return <ExclamationCircleOutlined />;
    case 'cancelled': return <StopOutlined />;
    // 兼容旧状态
    case 'complete': return <CheckCircleOutlined />;
    default: return <PlayCircleOutlined />;
  }
};

// 格式化持续时间
const formatDuration = (seconds) => {
  if (seconds < 60) return `${Math.round(seconds)}秒`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}分钟`;
  return `${Math.round(seconds / 3600)}小时`;
};

// 格式化时间戳
const formatTimestamp = (timestamp) => {
  return new Date(timestamp * 1000).toLocaleString();
};

const { TabPane } = Tabs;
const { RangePicker } = DatePicker;
const { Search } = Input;
const { Option } = Select;

const SlurmTasksPage = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const autoRefreshRef = useRef(null);
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedTask, setSelectedTask] = useState(null);
  const [taskDetailModal, setTaskDetailModal] = useState(false);
  const [taskProgress, setTaskProgress] = useState(null);
  const [statistics, setStatistics] = useState(null);
  const [activeTab, setActiveTab] = useState('tasks');
  const [lastRefresh, setLastRefresh] = useState(Date.now());
  const [refreshInterval, setRefreshInterval] = useState(30000); // 默认30秒
  const [isAutoRefreshEnabled, setIsAutoRefreshEnabled] = useState(true);
  
  // 过滤和搜索状态
  const [filters, setFilters] = useState({
    status: null,
    type: null,
    user_id: null,
    cluster_id: null,
    search: '',
    date_range: null,
  });
  
  // 分页状态
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  // 表格列定义
  const columns = [
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      render: (name, record) => (
        <Space>
          {getTaskStatusIcon(record.status)}
          <div>
            <Text strong>{name}</Text>
            <br />
            <Text type="secondary" style={{ fontSize: '12px' }}>
              {record.type} • ID: {record.id}
            </Text>
          </div>
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={getTaskStatusColor(status)}>
          {status === 'pending' ? '等待中' :
           status === 'running' ? '运行中' :
           status === 'completed' ? '已完成' :
           status === 'failed' ? '失败' :
           status === 'cancelled' ? '已取消' : status}
        </Tag>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type) => (
        <Tag color="blue">
          {type === 'scale_up' ? '扩容' :
           type === 'scale_down' ? '缩容' :
           type === 'node_init' ? '节点初始化' :
           type === 'cluster_setup' ? '集群配置' : type}
        </Tag>
      ),
    },
    {
      title: '进度',
      dataIndex: 'progress',
      key: 'progress',
      render: (progress, record) => {
        // 确保进度值是数字
        const progressValue = typeof progress === 'number' ? progress : 0;
        
        if (record.status === 'running') {
          const percent = Math.round(progressValue * 100);
          return <Progress percent={percent} size="small" showInfo={true} />;
        } else if (record.status === 'completed') {
          return <Progress percent={100} size="small" status="success" />;
        } else if (record.status === 'failed') {
          return <Progress percent={100} size="small" status="exception" />;
        } else if (record.status === 'pending') {
          return <Progress percent={0} size="small" />;
        }
        return '-';
      },
    },
    {
      title: '集群',
      dataIndex: 'cluster_name',
      key: 'cluster_name',
      render: (name) => name || '-',
    },
    {
      title: '用户',
      dataIndex: 'username',
      key: 'username',
      render: (username) => username || '-',
    },
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
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button
              size="small"
              icon={<EyeOutlined />}
              onClick={() => handleViewTaskDetail(record)}
            />
          </Tooltip>
          {record.status === 'running' && (
            <>
              <Tooltip title="刷新进度">
                <Button
                  size="small"
                  icon={<ReloadOutlined />}
                  onClick={() => handleRefreshTask(record.id)}
                />
              </Tooltip>
              <Popconfirm
                title="确定要取消这个任务吗？"
                description="取消后任务将停止执行"
                onConfirm={() => handleCancelTask(record.id)}
                okText="确定"
                cancelText="取消"
              >
                <Tooltip title="取消任务">
                  <Button
                    size="small"
                    icon={<StopOutlined />}
                    danger
                  />
                </Tooltip>
              </Popconfirm>
            </>
          )}
          {(record.status === 'failed' || record.status === 'cancelled') && (
            <Tooltip title="重试任务">
              <Button
                size="small"
                icon={<RedoOutlined />}
                onClick={() => handleRetryTask(record.id)}
              />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  // 加载任务列表
  const loadTasks = async (params = {}) => {
    setLoading(true);
    try {
      const queryParams = {
        page: pagination.current,
        limit: pagination.pageSize,
        ...filters,
        ...params,
      };

      // 处理日期范围
      if (queryParams.date_range && Array.isArray(queryParams.date_range)) {
        queryParams.start_date = queryParams.date_range[0].format('YYYY-MM-DD');
        queryParams.end_date = queryParams.date_range[1].format('YYYY-MM-DD');
        delete queryParams.date_range;
      }

      // 移除空值
      Object.keys(queryParams).forEach(key => {
        if (queryParams[key] === null || queryParams[key] === '' || queryParams[key] === undefined) {
          delete queryParams[key];
        }
      });

      const response = await slurmAPI.getTasks(queryParams);
      const data = response.data?.data || {};
      
      setTasks(data.tasks || []);
      setPagination(prev => ({
        ...prev,
        total: data.total || 0,
      }));
      setError(null);
    } catch (e) {
      console.error('加载任务列表失败', e);
      setError(e.message || '加载任务列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 加载统计信息
  const loadStatistics = async () => {
    try {
      const params = {};
      if (filters.date_range && Array.isArray(filters.date_range)) {
        params.start_date = filters.date_range[0].format('YYYY-MM-DD');
        params.end_date = filters.date_range[1].format('YYYY-MM-DD');
      }

      const response = await slurmAPI.getTaskStatistics(params);
      setStatistics(response.data?.data || null);
    } catch (e) {
      console.error('加载统计信息失败', e);
    }
  };

  // 查看任务详情
  const handleViewTaskDetail = async (task) => {
    setSelectedTask(task);
    setTaskDetailModal(true);

    try {
      // 尝试获取详细的任务信息
      let detailData = {};
      try {
        const response = await slurmAPI.getTaskDetail(task.id);
        detailData = response.data?.data || response.data || {};
      } catch (detailError) {
        console.warn('获取任务详情失败，使用基础信息', detailError);
        // 如果获取详情失败，使用传入的基础任务信息
        detailData = task;
      }
      
      setSelectedTask({
        ...task,
        ...detailData,
      });

      // 如果任务正在运行，获取实时进度
      if (task.status === 'running') {
        try {
          const progressResponse = await slurmAPI.getProgress(task.id);
          setTaskProgress(progressResponse.data?.data || progressResponse.data);
        } catch (progressError) {
          console.warn('获取任务进度失败', progressError);
          // 进度获取失败不影响详情显示
        }
      }
    } catch (e) {
      console.error('查看任务详情失败', e);
      // 即使出错，也显示基础信息
      setSelectedTask(task);
    }
  };

  // 刷新任务进度
  const handleRefreshTask = async (taskId) => {
    try {
      const response = await slurmAPI.getProgress(taskId);
      const updatedProgress = response.data?.data;

      // 更新任务列表中的进度
      setTasks(prevTasks =>
        prevTasks.map(task =>
          task.id === taskId
            ? {
                ...task,
                progress: updatedProgress.events[updatedProgress.events.length - 1]?.progress || task.progress,
                current_step: updatedProgress.events[updatedProgress.events.length - 1]?.step || task.current_step,
                last_message: updatedProgress.events[updatedProgress.events.length - 1]?.message || task.last_message,
              }
            : task
        )
      );

      message.success('任务进度已刷新');
    } catch (e) {
      console.error('刷新任务进度失败', e);
      message.error('刷新任务进度失败');
    }
  };

  // 取消任务
  const handleCancelTask = async (taskId, reason = '用户手动取消') => {
    try {
      await slurmAPI.cancelTask(taskId, reason);
      message.success('任务已取消');
      loadTasks();
    } catch (e) {
      console.error('取消任务失败', e);
      message.error('取消任务失败: ' + (e.response?.data?.error || e.message));
    }
  };

  // 重试任务
  const handleRetryTask = async (taskId) => {
    try {
      const response = await slurmAPI.retryTask(taskId);
      message.success(`任务重试已启动，新任务ID: ${response.data?.data?.id}`);
      loadTasks();
    } catch (e) {
      console.error('重试任务失败', e);
      message.error('重试任务失败: ' + (e.response?.data?.error || e.message));
    }
  };

  // 处理过滤器变化
  const handleFilterChange = (key, value) => {
    const newFilters = { ...filters, [key]: value };
    setFilters(newFilters);
    setPagination(prev => ({ ...prev, current: 1 }));
  };

  // 处理搜索
  const handleSearch = (value) => {
    handleFilterChange('search', value);
  };

  // 处理分页变化
  const handleTableChange = (paginationInfo, filters, sorter) => {
    setPagination(prev => ({
      ...prev,
      current: paginationInfo.current,
      pageSize: paginationInfo.pageSize,
    }));
  };

  // 重置过滤器
  const handleResetFilters = () => {
    setFilters({
      status: null,
      type: null,
      user_id: null,
      cluster_id: null,
      search: '',
      date_range: null,
    });
    setPagination(prev => ({ ...prev, current: 1 }));
  };

  // 关闭任务详情模态框
  const handleCloseTaskDetail = () => {
    setTaskDetailModal(false);
    setSelectedTask(null);
    setTaskProgress(null);
  };

  // 初始化加载和URL参数处理
  useEffect(() => {
    // 处理URL参数
    const searchParams = new URLSearchParams(location.search);
    const statusParam = searchParams.get('status');
    const taskIdParam = searchParams.get('taskId');
    
    if (statusParam) {
      setFilters(prev => ({ ...prev, status: statusParam }));
    }
    
    if (activeTab === 'tasks') {
      loadTasks().then(() => {
        // 如果有指定的任务ID，自动打开详情
        if (taskIdParam && tasks.length > 0) {
          const targetTask = tasks.find(task => task.id === taskIdParam);
          if (targetTask) {
            handleViewTaskDetail(targetTask);
          }
        }
      });
    } else if (activeTab === 'statistics') {
      loadStatistics();
    }
  }, [filters, pagination.current, pagination.pageSize, activeTab]);

  // 智能刷新间隔调整
  const adjustRefreshInterval = (runningTasksCount) => {
    if (runningTasksCount === 0) {
      return 0; // 无运行任务时不刷新
    } else if (runningTasksCount <= 2) {
      return 30000; // 1-2个任务：30秒
    } else if (runningTasksCount <= 5) {
      return 20000; // 3-5个任务：20秒
    } else {
      return 15000; // 5个以上任务：15秒
    }
  };

  // 自动刷新和页面可见性检测
  useEffect(() => {
    const runningTasks = tasks.filter(task => 
      task.status === 'running' || task.status === 'pending'
    );
    const runningTasksCount = runningTasks.length;
    
    // 清理之前的定时器
    if (autoRefreshRef.current) {
      clearInterval(autoRefreshRef.current);
    }

    // 智能调整刷新间隔
    const newInterval = adjustRefreshInterval(runningTasksCount);
    setRefreshInterval(newInterval);

    // 只有在有运行中任务且启用自动刷新时才设置定时器
    if (activeTab === 'tasks' && runningTasksCount > 0 && isAutoRefreshEnabled && newInterval > 0) {
      console.log(`设置自动刷新：${newInterval/1000}秒间隔，${runningTasksCount}个运行中任务`);
      autoRefreshRef.current = setInterval(() => {
        console.log(`自动刷新任务列表... (${runningTasksCount}个运行中任务)`);
        loadTasks();
        setLastRefresh(Date.now());
      }, newInterval);
    }

    // 页面可见性变化时的处理（降低频率）
    let visibilityTimer = null;
    const handleVisibilityChange = () => {
      if (!document.hidden && activeTab === 'tasks') {
        // 防抖处理，避免频繁切换
        if (visibilityTimer) {
          clearTimeout(visibilityTimer);
        }
        visibilityTimer = setTimeout(() => {
          console.log('页面变为可见，刷新任务列表...');
          loadTasks();
          setLastRefresh(Date.now());
        }, 1000); // 1秒延迟
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);

    return () => {
      if (autoRefreshRef.current) {
        clearInterval(autoRefreshRef.current);
      }
      if (visibilityTimer) {
        clearTimeout(visibilityTimer);
      }
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [tasks, activeTab, isAutoRefreshEnabled]);

  if (loading && tasks.length === 0) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" />
        <div style={{ marginTop: '16px' }}>
          <Text>加载任务列表中...</Text>
        </div>
      </div>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>
        <ThunderboltOutlined style={{ marginRight: '8px' }} />
        Slurm 任务管理
      </Title>

      <Paragraph>
        查看和管理 Slurm 集群的各项任务进度，包括客户端安装、节点初始化、扩缩容等操作。
      </Paragraph>

      <Divider />

      {error && (
        <Alert
          message="加载失败"
          description={error}
          type="error"
          showIcon
          style={{ marginBottom: '16px' }}
          action={
            <Button size="small" onClick={() => loadTasks()}>
              重试
            </Button>
          }
        />
      )}

      {tasks.some(task => task.status === 'running') && (
        <Alert
          message={`${tasks.filter(task => task.status === 'running').length} 个任务正在运行`}
          description={
            <div>
              {isAutoRefreshEnabled ? (
                refreshInterval > 0 ? (
                  `自动刷新已启用，间隔 ${refreshInterval/1000} 秒`
                ) : (
                  '无运行任务，自动刷新已暂停'
                )
              ) : (
                '自动刷新已关闭，点击上方按钮手动刷新或启用自动刷新'
              )}
              <span style={{ marginLeft: '16px', color: '#666' }}>
                上次更新: {dayjs(lastRefresh).format('HH:mm:ss')}
              </span>
            </div>
          }
          type="info"
          showIcon
          style={{ marginBottom: '16px' }}
        />
      )}

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane
          tab={
            <span>
              <DesktopOutlined />
              任务列表
            </span>
          }
          key="tasks"
        >
          {/* 过滤器区域 */}
          <Card style={{ marginBottom: '16px' }}>
            <Row gutter={[16, 16]} align="middle">
              <Col xs={24} sm={12} md={6}>
                <Search
                  placeholder="搜索任务名称..."
                  value={filters.search}
                  onChange={(e) => handleFilterChange('search', e.target.value)}
                  onSearch={handleSearch}
                  enterButton={<SearchOutlined />}
                  allowClear
                />
              </Col>
              <Col xs={12} sm={6} md={4}>
                <Select
                  placeholder="任务状态"
                  value={filters.status}
                  onChange={(value) => handleFilterChange('status', value)}
                  allowClear
                  style={{ width: '100%' }}
                >
                  <Option value="pending">等待中</Option>
                  <Option value="running">运行中</Option>
                  <Option value="completed">已完成</Option>
                  <Option value="failed">失败</Option>
                  <Option value="cancelled">已取消</Option>
                </Select>
              </Col>
              <Col xs={12} sm={6} md={4}>
                <Select
                  placeholder="任务类型"
                  value={filters.type}
                  onChange={(value) => handleFilterChange('type', value)}
                  allowClear
                  style={{ width: '100%' }}
                >
                  <Option value="scale_up">扩容</Option>
                  <Option value="scale_down">缩容</Option>
                  <Option value="node_init">节点初始化</Option>
                  <Option value="cluster_setup">集群配置</Option>
                </Select>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <RangePicker
                  value={filters.date_range}
                  onChange={(dates) => handleFilterChange('date_range', dates)}
                  placeholder={['开始日期', '结束日期']}
                  style={{ width: '100%' }}
                />
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Space wrap>
                  <Button
                    icon={<FilterOutlined />}
                    onClick={handleResetFilters}
                    size="small"
                  >
                    重置
                  </Button>
                  <Button
                    icon={<ReloadOutlined />}
                    onClick={() => {
                      loadTasks();
                      setLastRefresh(Date.now());
                    }}
                    loading={loading}
                    type="primary"
                    size="small"
                  >
                    刷新
                  </Button>
                  <Button
                    size="small"
                    type={isAutoRefreshEnabled ? "primary" : "default"}
                    onClick={() => setIsAutoRefreshEnabled(!isAutoRefreshEnabled)}
                    ghost={isAutoRefreshEnabled}
                    style={{ minWidth: '88px' }}
                  >
                    {isAutoRefreshEnabled ? '🔄' : '⏸️'} 自动
                  </Button>
                </Space>
              </Col>
              <Col xs={24} sm={24} md={8}>
                <div style={{ 
                  fontSize: '12px', 
                  color: '#666',
                  textAlign: 'right',
                  padding: '4px 0',
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis'
                }}>
                  上次更新: {dayjs(lastRefresh).format('HH:mm:ss')}
                  {refreshInterval > 0 && isAutoRefreshEnabled && (
                    <span style={{ marginLeft: '8px' }}>
                      (每{refreshInterval/1000}秒)
                    </span>
                  )}
                </div>
              </Col>
            </Row>
          </Card>

          {/* 任务列表 */}
          <Card
            title={
              <Space>
                <DesktopOutlined />
                任务列表
                <Badge count={pagination.total} />
                {tasks.some(task => task.status === 'running') && (
                  <Tag color="blue" icon={<ClockCircleOutlined />}>
                    {tasks.filter(task => task.status === 'running').length} 个运行中
                  </Tag>
                )}
              </Space>
            }
          >
            <Table
              columns={columns}
              dataSource={tasks}
              rowKey="id"
              loading={loading}
              pagination={{
                ...pagination,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
              }}
              onChange={handleTableChange}
              locale={{
                emptyText: loading ? '加载中...' : '暂无任务记录',
              }}
              scroll={{ x: 1200 }}
              size="small"
            />
          </Card>
        </TabPane>

        <TabPane
          tab={
            <span>
              <BarChartOutlined />
              统计信息
            </span>
          }
          key="statistics"
        >
          {statistics ? (
            <Row gutter={[16, 16]}>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="总任务数"
                    value={statistics.total_tasks}
                    prefix={<DesktopOutlined />}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="运行中"
                    value={statistics.running_tasks}
                    prefix={<ClockCircleOutlined />}
                    valueStyle={{ color: '#1890ff' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="已完成"
                    value={statistics.completed_tasks}
                    prefix={<CheckCircleOutlined />}
                    valueStyle={{ color: '#52c41a' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={6}>
                <Card>
                  <Statistic
                    title="失败"
                    value={statistics.failed_tasks}
                    prefix={<ExclamationCircleOutlined />}
                    valueStyle={{ color: '#ff4d4f' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Card>
                  <Statistic
                    title="成功率"
                    value={statistics.success_rate}
                    suffix="%"
                    precision={1}
                    prefix={<CheckCircleOutlined />}
                    valueStyle={{ 
                      color: statistics.success_rate > 80 ? '#52c41a' : 
                             statistics.success_rate > 50 ? '#faad14' : '#ff4d4f' 
                    }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Card>
                  <Statistic
                    title="平均执行时间"
                    value={statistics.avg_duration}
                    suffix="秒"
                    prefix={<ClockCircleOutlined />}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Card>
                  <Statistic
                    title="活跃用户数"
                    value={statistics.active_users}
                    prefix={<DesktopOutlined />}
                  />
                </Card>
              </Col>
            </Row>
          ) : (
            <Card>
              <Empty description="暂无统计数据" />
            </Card>
          )}
        </TabPane>
      </Tabs>

      {/* 任务详情模态框 */}
      <Modal
        title={
          <Space>
            {selectedTask && getTaskStatusIcon(selectedTask.status)}
            {selectedTask?.name} - 任务详情
          </Space>
        }
        open={taskDetailModal}
        onCancel={handleCloseTaskDetail}
        footer={[
          selectedTask?.status === 'running' && (
            <Popconfirm
              key="cancel"
              title="确定要取消这个任务吗？"
              description="取消后任务将停止执行"
              onConfirm={() => {
                handleCancelTask(selectedTask.id);
                handleCloseTaskDetail();
              }}
              okText="确定"
              cancelText="取消"
            >
              <Button danger icon={<StopOutlined />}>
                取消任务
              </Button>
            </Popconfirm>
          ),
          (selectedTask?.status === 'failed' || selectedTask?.status === 'cancelled') && (
            <Button
              key="retry"
              icon={<RedoOutlined />}
              onClick={() => {
                handleRetryTask(selectedTask.id);
                handleCloseTaskDetail();
              }}
            >
              重试任务
            </Button>
          ),
          <Button key="close" onClick={handleCloseTaskDetail}>
            关闭
          </Button>
        ]}
        width={900}
      >
        {selectedTask && (
          <Tabs defaultActiveKey="basic">
            <TabPane
              tab={
                <span>
                  <InfoCircleOutlined />
                  基本信息
                </span>
              }
              key="basic"
            >
              <Descriptions bordered column={2}>
                <Descriptions.Item label="任务ID">{selectedTask.id}</Descriptions.Item>
                <Descriptions.Item label="状态">
                  <Tag color={getTaskStatusColor(selectedTask.status)}>
                    {selectedTask.status === 'pending' ? '等待中' :
                     selectedTask.status === 'running' ? '运行中' :
                     selectedTask.status === 'completed' ? '已完成' :
                     selectedTask.status === 'failed' ? '失败' :
                     selectedTask.status === 'cancelled' ? '已取消' : selectedTask.status}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="任务类型">
                  <Tag color="blue">
                    {selectedTask.type === 'scale_up' ? '扩容' :
                     selectedTask.type === 'scale_down' ? '缩容' :
                     selectedTask.type === 'node_init' ? '节点初始化' :
                     selectedTask.type === 'cluster_setup' ? '集群配置' : selectedTask.type}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="优先级">
                  {selectedTask.priority || '默认'}
                </Descriptions.Item>
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
                <Descriptions.Item label="集群">{selectedTask.cluster_name || '-'}</Descriptions.Item>
                <Descriptions.Item label="用户">{selectedTask.username || '-'}</Descriptions.Item>
                <Descriptions.Item label="重试次数">{selectedTask.retry_count || 0}</Descriptions.Item>
                <Descriptions.Item label="最大重试次数">{selectedTask.max_retries || 3}</Descriptions.Item>
                {selectedTask.description && (
                  <Descriptions.Item label="描述" span={2}>
                    {selectedTask.description}
                  </Descriptions.Item>
                )}
                {selectedTask.error_message && (
                  <Descriptions.Item label="错误信息" span={2}>
                    <Text type="danger">{selectedTask.error_message}</Text>
                  </Descriptions.Item>
                )}
              </Descriptions>

              {selectedTask.status === 'running' && selectedTask.progress !== undefined && (
                <div style={{ marginTop: '16px' }}>
                  <Text strong>当前进度：</Text>
                  <Progress
                    percent={Math.round(selectedTask.progress * 100)}
                    status="active"
                    style={{ marginTop: '8px' }}
                  />
                </div>
              )}
            </TabPane>

            <TabPane
              tab={
                <span>
                  <DesktopOutlined />
                  参数配置
                </span>
              }
              key="parameters"
            >
              {selectedTask.parameters ? (
                <pre style={{ 
                  background: '#f5f5f5', 
                  padding: '12px', 
                  borderRadius: '4px',
                  maxHeight: '400px',
                  overflow: 'auto'
                }}>
                  {JSON.stringify(selectedTask.parameters, null, 2)}
                </pre>
              ) : (
                <Empty description="无参数配置" />
              )}
            </TabPane>

            <TabPane
              tab={
                <span>
                  <HistoryOutlined />
                  执行日志
                </span>
              }
              key="events"
            >
              {selectedTask.events && selectedTask.events.length > 0 ? (
                <Timeline style={{ marginTop: '16px' }}>
                  {selectedTask.events.map((event, index) => (
                    <Timeline.Item
                      key={index}
                      color={
                        event.level === 'error' ? 'red' :
                        event.level === 'warning' ? 'orange' :
                        event.level === 'success' ? 'green' : 'blue'
                      }
                    >
                      <div>
                        <Space>
                          <Text strong>{event.message}</Text>
                          <Text type="secondary" style={{ fontSize: '12px' }}>
                            {event.created_at ? (() => {
                              const ts = event.created_at < 10000000000 ? event.created_at * 1000 : event.created_at;
                              return dayjs(ts).format('YYYY-MM-DD HH:mm:ss');
                            })() : ''}
                          </Text>
                        </Space>
                        {event.details && (
                          <div style={{ marginTop: '4px' }}>
                            <Text type="secondary">{event.details}</Text>
                          </div>
                        )}
                      </div>
                    </Timeline.Item>
                  ))}
                </Timeline>
              ) : taskProgress && taskProgress.events ? (
                <List
                  size="small"
                  bordered
                  dataSource={taskProgress.events}
                  renderItem={(event) => (
                    <List.Item>
                      <Space direction="vertical" style={{ width: '100%' }}>
                        <Space>
                          <Text strong>{event.step}</Text>
                          <Text type="secondary">
                            {(() => {
                              const ts = event.ts < 10000000000 ? event.ts * 1000 : event.ts;
                              return new Date(ts).toLocaleString();
                            })()}
                          </Text>
                          {event.host && (
                            <Tag size="small" color="blue">{event.host}</Tag>
                          )}
                        </Space>
                        <Text>{event.message}</Text>
                        {event.progress !== undefined && (
                          <Progress
                            percent={Math.round(event.progress * 100)}
                            size="small"
                            showInfo={false}
                          />
                        )}
                      </Space>
                    </List.Item>
                  )}
                  style={{ maxHeight: '400px', overflow: 'auto' }}
                />
              ) : (
                <Empty description="暂无执行日志" />
              )}
            </TabPane>

            {selectedTask.statistics && (
              <TabPane
                tab={
                  <span>
                    <BarChartOutlined />
                    统计信息
                  </span>
                }
                key="statistics"
              >
                <Row gutter={[16, 16]}>
                  <Col span={8}>
                    <Statistic
                      title="处理的节点数"
                      value={selectedTask.statistics.nodes_processed || 0}
                    />
                  </Col>
                  <Col span={8}>
                    <Statistic
                      title="成功的节点数"
                      value={selectedTask.statistics.nodes_successful || 0}
                    />
                  </Col>
                  <Col span={8}>
                    <Statistic
                      title="失败的节点数"
                      value={selectedTask.statistics.nodes_failed || 0}
                    />
                  </Col>
                </Row>
                {selectedTask.statistics.node_results && (
                  <div style={{ marginTop: '16px' }}>
                    <Title level={5}>节点详情</Title>
                    <List
                      size="small"
                      bordered
                      dataSource={selectedTask.statistics.node_results}
                      renderItem={(result) => (
                        <List.Item>
                          <Space>
                            <Text strong>{result.node_name}</Text>
                            <Tag color={result.status === 'success' ? 'green' : 'red'}>
                              {result.status === 'success' ? '成功' : '失败'}
                            </Tag>
                            <Text type="secondary">{result.message}</Text>
                          </Space>
                        </List.Item>
                      )}
                    />
                  </div>
                )}
              </TabPane>
            )}
          </Tabs>
        )}
      </Modal>
    </div>
  );
};

export default SlurmTasksPage;