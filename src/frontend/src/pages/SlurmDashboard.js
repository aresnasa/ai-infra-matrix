import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Space, Alert, Spin, Button, Typography, Divider, Modal, message, Dropdown, Tabs } from 'antd';
import { slurmAPI, saltStackAPI } from '../services/api';
import { CloudServerOutlined, HddOutlined, CheckCircleOutlined, SyncOutlined, PlayCircleOutlined, PauseCircleOutlined, StopOutlined, DownOutlined, CloseCircleOutlined, ReloadOutlined, HourglassOutlined } from '@ant-design/icons';
import SaltCommandExecutor from '../components/SaltCommandExecutor';
import SlurmClusterStatus from '../components/SlurmClusterStatus';

const { Title, Text } = Typography;
const { TabPane } = Tabs;

const SlurmDashboard = () => {
  const [summary, setSummary] = useState(null);
  const [nodes, setNodes] = useState([]);
  const [jobs, setJobs] = useState([]);
  const [partitions, setPartitions] = useState([]);
  const [saltStackData, setSaltStackData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saltStackLoading, setSaltStackLoading] = useState(false);
  const [demo, setDemo] = useState(false);
  const [error, setError] = useState(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [selectedJobKeys, setSelectedJobKeys] = useState([]);
  const [operationLoading, setOperationLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('overview');

  const load = async () => {
    setLoading(true);
    try {
      const [s, n, j, p] = await Promise.all([
        slurmAPI.getSummary(),
        slurmAPI.getNodes(),
        slurmAPI.getJobs(),
        slurmAPI.getPartitions(),
      ]);
      setSummary(s.data?.data);
      setNodes(n.data?.data || []);
      setJobs(j.data?.data || []);
      setPartitions(p.data?.data || []);
      // demo 标记兼容后端在 data 内或顶层返回
      setDemo(Boolean(s.data?.data?.demo || s.data?.demo || n.data?.demo || j.data?.demo));
      setError(null);
    } catch (e) {
      console.error('加载Slurm数据失败', e);
      setError(e);
    } finally {
      setLoading(false);
    }
  };

  const loadSaltStackIntegration = async () => {
    setSaltStackLoading(true);
    try {
      const response = await saltStackAPI.getSaltStackIntegration();
      setSaltStackData(response.data?.data || null);
    } catch (e) {
      console.error('加载SaltStack集成数据失败', e);
      // 不显示错误，因为这是可选功能
    } finally {
      setSaltStackLoading(false);
    }
  };

  useEffect(() => {
    load();
    loadSaltStackIntegration();
    const t = setInterval(() => {
      load();
      loadSaltStackIntegration();
    }, 15000);
    return () => clearInterval(t);
  }, []);

  // 节点管理函数
  const handleNodeOperation = async (action, actionLabel, reason = '') => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要操作的节点');
      return;
    }

    Modal.confirm({
      title: `确认${actionLabel}节点`,
      content: `您确定要将选中的 ${selectedRowKeys.length} 个节点设置为 ${actionLabel} 状态吗？`,
      okText: '确认',
      cancelText: '取消',
      onOk: async () => {
        setOperationLoading(true);
        try {
          const response = await slurmAPI.manageNodes(selectedRowKeys, action, reason);
          if (response.data?.success) {
            message.success(response.data.message || `成功${actionLabel} ${selectedRowKeys.length} 个节点`);
            setSelectedRowKeys([]);
            // 重新加载节点列表
            await load();
          } else {
            message.error(response.data?.error || `${actionLabel}节点失败`);
          }
        } catch (error) {
          console.error(`${actionLabel}节点失败:`, error);
          message.error(error.response?.data?.error || `${actionLabel}节点失败，请稍后重试`);
        } finally {
          setOperationLoading(false);
        }
      },
    });
  };

  // 作业管理函数
  const handleJobOperation = async (action, actionLabel) => {
    if (selectedJobKeys.length === 0) {
      message.warning('请先选择要操作的作业');
      return;
    }

    Modal.confirm({
      title: `确认${actionLabel}作业`,
      content: `您确定要对选中的 ${selectedJobKeys.length} 个作业执行 ${actionLabel} 操作吗？`,
      okText: '确认',
      cancelText: '取消',
      onOk: async () => {
        setOperationLoading(true);
        try {
          const response = await slurmAPI.manageJobs(selectedJobKeys, action);
          if (response.data?.success) {
            message.success(response.data.message || `成功${actionLabel} ${selectedJobKeys.length} 个作业`);
            setSelectedJobKeys([]);
            // 重新加载作业列表
            await load();
          } else {
            message.error(response.data?.error || `${actionLabel}作业失败`);
          }
        } catch (error) {
          console.error(`${actionLabel}作业失败:`, error);
          message.error(error.response?.data?.error || `${actionLabel}作业失败，请稍后重试`);
        } finally {
          setOperationLoading(false);
        }
      },
    });
  };

  // 节点操作菜单
  const nodeOperationMenuItems = [
    {
      key: 'resume',
      label: '恢复 (RESUME)',
      icon: <PlayCircleOutlined />,
      onClick: () => handleNodeOperation('resume', '恢复', '手动恢复节点'),
    },
    {
      key: 'drain',
      label: '排空 (DRAIN)',
      icon: <PauseCircleOutlined />,
      onClick: () => handleNodeOperation('drain', '排空', '节点维护'),
    },
    {
      key: 'down',
      label: '下线 (DOWN)',
      icon: <StopOutlined />,
      onClick: () => handleNodeOperation('down', '下线', '节点故障或维护'),
    },
    {
      key: 'idle',
      label: '空闲 (IDLE)',
      icon: <CheckCircleOutlined />,
      onClick: () => handleNodeOperation('idle', '设为空闲', '手动设置空闲'),
    },
  ];

  // 作业操作菜单
  const jobOperationMenuItems = [
    {
      key: 'cancel',
      label: '取消作业 (Cancel)',
      icon: <CloseCircleOutlined />,
      danger: true,
      onClick: () => handleJobOperation('cancel', '取消'),
    },
    {
      key: 'hold',
      label: '暂停调度 (Hold)',
      icon: <PauseCircleOutlined />,
      onClick: () => handleJobOperation('hold', '暂停调度'),
    },
    {
      key: 'release',
      label: '释放调度 (Release)',
      icon: <PlayCircleOutlined />,
      onClick: () => handleJobOperation('release', '释放调度'),
    },
    {
      key: 'suspend',
      label: '挂起作业 (Suspend)',
      icon: <HourglassOutlined />,
      onClick: () => handleJobOperation('suspend', '挂起'),
    },
    {
      key: 'resume',
      label: '恢复作业 (Resume)',
      icon: <PlayCircleOutlined />,
      onClick: () => handleJobOperation('resume', '恢复'),
    },
    {
      key: 'requeue',
      label: '重新排队 (Requeue)',
      icon: <ReloadOutlined />,
      onClick: () => handleJobOperation('requeue', '重新排队'),
    },
  ];

  const columnsNodes = [
    { title: '节点', dataIndex: 'name', key: 'name' },
    { title: '分区', dataIndex: 'partition', key: 'partition' },
    { 
      title: '状态', 
      dataIndex: 'state', 
      key: 'state', 
      render: (s) => {
        const state = s.toLowerCase();
        let color = 'default';
        if (state.includes('idle')) color = 'green';
        else if (state.includes('alloc') || state.includes('mixed')) color = 'blue';
        else if (state.includes('down') || state.includes('drain')) color = 'red';
        else if (state.includes('unk')) color = 'orange';
        return <Tag color={color}>{s}</Tag>;
      }
    },
    { title: 'CPU', dataIndex: 'cpus', key: 'cpus' },
    { title: '内存(MB)', dataIndex: 'memory_mb', key: 'memory_mb' },
  ];

  const columnsJobs = [
    { title: '作业ID', dataIndex: 'id', key: 'id' },
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '用户', dataIndex: 'user', key: 'user' },
    { title: '分区', dataIndex: 'partition', key: 'partition' },
    { title: '状态', dataIndex: 'state', key: 'state', render: (s) => <Tag color={s === 'RUNNING' ? 'blue' : s === 'PENDING' ? 'orange' : 'default'}>{s}</Tag> },
    { title: '耗时', dataIndex: 'elapsed', key: 'elapsed' },
    { title: '节点数', dataIndex: 'nodes', key: 'nodes' },
    { title: '原因', dataIndex: 'reason', key: 'reason' },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Title level={2}>Slurm 集群管理</Title>
        
        {error && (
          <Alert 
            type="error" 
            showIcon 
            message="无法加载Slurm数据"
            description={
              <Space>
                <span>请确认已登录且后端 /api/slurm 接口可达。</span>
                <Button size="small" onClick={load}>重试</Button>
              </Space>
            }
          />
        )}
        {demo && (
          <Alert type="info" showIcon message="使用演示数据：未检测到Slurm命令(sinfo/squeue)，展示示例统计" />
        )}

        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          {/* 概览页签 */}
          <TabPane tab="集群概览" key="overview">

        <Row gutter={16}>
          <Col span={6}>
            <Card>
              <Statistic title="节点总数" value={summary?.nodes_total || 0} loading={loading} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title="空闲节点" value={summary?.nodes_idle || 0} loading={loading} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title="占用节点" value={summary?.nodes_alloc || 0} loading={loading} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title="分区数量" value={summary?.partitions || 0} loading={loading} />
            </Card>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={8}>
            <Card>
              <Statistic title="运行中作业" value={summary?.jobs_running || 0} loading={loading} />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic title="等待中作业" value={summary?.jobs_pending || 0} loading={loading} />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic title="其他状态" value={summary?.jobs_other || 0} loading={loading} />
            </Card>
          </Col>
        </Row>

        <Divider />

        {/* SaltStack 集成状态 */}
        {saltStackData && (
          <Card 
            title={
              <Space>
                <CloudServerOutlined />
                <span>SaltStack 集成状态</span>
                {saltStackData.enabled && (
                  <Tag color="green" icon={<CheckCircleOutlined />}>已启用</Tag>
                )}
                {!saltStackData.enabled && (
                  <Tag color="default">未启用</Tag>
                )}
              </Space>
            }
            extra={saltStackLoading ? <Spin size="small" /> : null}
            style={{ marginBottom: '16px' }}
          >
            {/* Master 和 API 状态 */}
            <Row gutter={16} style={{ marginBottom: '16px' }}>
              <Col span={8}>
                <Card size="small">
                  <Statistic
                    title="Master 状态"
                    value={saltStackData.master_status || '未知'}
                    valueStyle={{ 
                      color: saltStackData.master_status === 'running' ? '#3f8600' : '#cf1322',
                      fontSize: '16px'
                    }}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card size="small">
                  <Statistic
                    title="API 状态"
                    value={saltStackData.api_status || '未知'}
                    valueStyle={{ 
                      color: saltStackData.api_status === 'connected' ? '#3f8600' : '#cf1322',
                      fontSize: '16px'
                    }}
                  />
                </Card>
              </Col>
              <Col span={8}>
                <Card size="small">
                  <Statistic
                    title="活跃作业"
                    value={saltStackData.recent_jobs || 0}
                    prefix={<SyncOutlined />}
                  />
                </Card>
              </Col>
            </Row>

            {/* Minion 统计 */}
            <Row gutter={16}>
              <Col span={8}>
                <Statistic
                  title="连接的 Minions"
                  value={saltStackData.minions?.online || 0}
                  valueStyle={{ color: '#3f8600' }}
                  prefix={<CheckCircleOutlined />}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="离线 Minions"
                  value={saltStackData.minions?.offline || 0}
                  valueStyle={{ color: '#cf1322' }}
                />
              </Col>
              <Col span={8}>
                <Statistic
                  title="Minion 总数"
                  value={saltStackData.minions?.total || 0}
                  prefix={<HddOutlined />}
                />
              </Col>
            </Row>

            {saltStackData.minion_list && saltStackData.minion_list.length > 0 && (
              <div style={{ marginTop: '16px' }}>
                <Text strong>Minion 节点列表:</Text>
                <div style={{ marginTop: '8px' }}>
                  <Space wrap>
                    {saltStackData.minion_list.map((minion) => (
                      <Tag
                        key={minion.id}
                        color={minion.status === 'online' ? 'green' : minion.status === 'pending' ? 'orange' : 'default'}
                        icon={minion.status === 'online' ? <CheckCircleOutlined /> : null}
                      >
                        {minion.name || minion.id}
                      </Tag>
                    ))}
                  </Space>
                </div>
              </div>
            )}
          </Card>
        )}

        <Card 
          title="节点列表" 
          extra={
            <Space>
              {selectedRowKeys.length > 0 && (
                <>
                  <Text type="secondary">已选择 {selectedRowKeys.length} 个节点</Text>
                  <Dropdown 
                    menu={{ items: nodeOperationMenuItems }}
                    placement="bottomRight"
                    disabled={operationLoading}
                  >
                    <Button 
                      type="primary" 
                      loading={operationLoading}
                      icon={<DownOutlined />}
                    >
                      节点操作
                    </Button>
                  </Dropdown>
                </>
              )}
              {loading && <Spin size="small" />}
            </Space>
          }
        >
          <Table 
            rowKey="name" 
            dataSource={nodes} 
            columns={columnsNodes} 
            size="small" 
            pagination={{ pageSize: 8 }}
            rowSelection={{
              selectedRowKeys,
              onChange: setSelectedRowKeys,
              selections: [
                Table.SELECTION_ALL,
                Table.SELECTION_INVERT,
                Table.SELECTION_NONE,
              ],
            }}
          />
        </Card>

        <Card
          title="作业队列"
          extra={
            <Space>
              {selectedJobKeys.length > 0 && (
                <>
                  <Text type="secondary">已选择 {selectedJobKeys.length} 个作业</Text>
                  <Dropdown menu={{ items: jobOperationMenuItems }}>
                    <Button loading={operationLoading}>
                      作业操作 <DownOutlined />
                    </Button>
                  </Dropdown>
                </>
              )}
              {loading && <Spin size="small" />}
            </Space>
          }
        >
          <Table
            rowKey="id"
            dataSource={jobs}
            columns={columnsJobs}
            size="small"
            pagination={{ pageSize: 8 }}
            rowSelection={{
              selectedRowKeys: selectedJobKeys,
              onChange: setSelectedJobKeys,
            }}
          />
        </Card>
          </TabPane>

          {/* 集群状态监控页签 */}
          <TabPane tab="集群状态监控" key="cluster-status">
            <SlurmClusterStatus />
          </TabPane>

          {/* SaltStack 命令执行页签 */}
          <TabPane tab="SaltStack 命令执行" key="salt-commands">
            <SaltCommandExecutor />
          </TabPane>
        </Tabs>
      </Space>
    </div>
  );
};

export default SlurmDashboard;
