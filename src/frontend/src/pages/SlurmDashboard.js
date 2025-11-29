import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Space, Alert, Spin, Button, Typography, Divider, Modal, message, Dropdown, Tabs, Tooltip } from 'antd';
import { slurmAPI, saltStackAPI } from '../services/api';
import { CloudServerOutlined, HddOutlined, CheckCircleOutlined, SyncOutlined, PlayCircleOutlined, PauseCircleOutlined, StopOutlined, DownOutlined, CloseCircleOutlined, ReloadOutlined, HourglassOutlined, WarningOutlined } from '@ant-design/icons';
import SaltCommandExecutor from '../components/SaltCommandExecutor';
import SlurmClusterStatus from '../components/SlurmClusterStatus';
import { useI18n } from '../hooks/useI18n';

const { Title, Text } = Typography;
const { TabPane } = Tabs;

const SlurmDashboard = () => {
  const { t } = useI18n();
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
      message.warning(t('slurm.selectNodes'));
      return;
    }

    Modal.confirm({
      title: `${t('slurm.confirm')}${actionLabel}${t('slurm.nodes')}`,
      content: t('slurm.confirmNodeOperation', { count: selectedRowKeys.length, action: actionLabel }),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        setOperationLoading(true);
        try {
          const response = await slurmAPI.manageNodes(selectedRowKeys, action, reason);
          if (response.data?.success) {
            message.success(response.data.message || t('slurm.nodeOperationSuccess', { action: actionLabel, count: selectedRowKeys.length }));
            setSelectedRowKeys([]);
            // 重新加载节点列表
            await load();
          } else {
            message.error(response.data?.error || t('slurm.nodeOperationFailed', { action: actionLabel }));
          }
        } catch (error) {
          console.error(`${actionLabel}节点失败:`, error);
          message.error(error.response?.data?.error || t('slurm.nodeOperationFailedRetry', { action: actionLabel }));
        } finally {
          setOperationLoading(false);
        }
      },
    });
  };

  // 作业管理函数
  const handleJobOperation = async (action, actionLabel) => {
    if (selectedJobKeys.length === 0) {
      message.warning(t('slurm.selectJobs'));
      return;
    }

    Modal.confirm({
      title: `${t('slurm.confirm')}${actionLabel}${t('slurm.jobs')}`,
      content: t('slurm.confirmJobOperation', { count: selectedJobKeys.length, action: actionLabel }),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        setOperationLoading(true);
        try {
          const response = await slurmAPI.manageJobs(selectedJobKeys, action);
          if (response.data?.success) {
            message.success(response.data.message || t('slurm.jobOperationSuccess', { action: actionLabel, count: selectedJobKeys.length }));
            setSelectedJobKeys([]);
            // 重新加载作业列表
            await load();
          } else {
            message.error(response.data?.error || t('slurm.jobOperationFailed', { action: actionLabel }));
          }
        } catch (error) {
          console.error(`${actionLabel}作业失败:`, error);
          message.error(error.response?.data?.error || t('slurm.jobOperationFailedRetry', { action: actionLabel }));
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
      label: t('slurm.resume'),
      icon: <PlayCircleOutlined />,
      onClick: () => handleNodeOperation('resume', t('slurm.resume'), t('slurm.manualResume')),
    },
    {
      key: 'drain',
      label: t('slurm.drain'),
      icon: <PauseCircleOutlined />,
      onClick: () => handleNodeOperation('drain', t('slurm.drain'), t('slurm.nodeMaintenance')),
    },
    {
      key: 'down',
      label: t('slurm.down'),
      icon: <StopOutlined />,
      onClick: () => handleNodeOperation('down', t('slurm.down'), t('slurm.nodeFailure')),
    },
    {
      key: 'idle',
      label: t('slurm.idle'),
      icon: <CheckCircleOutlined />,
      onClick: () => handleNodeOperation('idle', t('slurm.idle'), t('slurm.manualSetIdle')),
    },
  ];

  // 作业操作菜单
  const jobOperationMenuItems = [
    {
      key: 'cancel',
      label: t('slurm.cancelJob'),
      icon: <CloseCircleOutlined />,
      danger: true,
      onClick: () => handleJobOperation('cancel', t('slurm.cancel')),
    },
    {
      key: 'hold',
      label: t('slurm.holdJob'),
      icon: <PauseCircleOutlined />,
      onClick: () => handleJobOperation('hold', t('slurm.hold')),
    },
    {
      key: 'release',
      label: t('slurm.releaseJob'),
      icon: <PlayCircleOutlined />,
      onClick: () => handleJobOperation('release', t('slurm.release')),
    },
    {
      key: 'suspend',
      label: t('slurm.suspendJob'),
      icon: <HourglassOutlined />,
      onClick: () => handleJobOperation('suspend', t('slurm.suspend')),
    },
    {
      key: 'resume',
      label: t('slurm.resumeJob'),
      icon: <PlayCircleOutlined />,
      onClick: () => handleJobOperation('resume', t('slurm.resume')),
    },
    {
      key: 'requeue',
      label: t('slurm.requeueJob'),
      icon: <ReloadOutlined />,
      onClick: () => handleJobOperation('requeue', t('slurm.requeue')),
    },
  ];

  const columnsNodes = [
    { title: t('slurm.node'), dataIndex: 'name', key: 'name' },
    { title: t('slurm.partition'), dataIndex: 'partition', key: 'partition' },
    { 
      title: t('slurm.slurmStatus'), 
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
    {
      title: t('slurm.saltStackStatus'),
      dataIndex: 'salt_status',
      key: 'salt_status',
      render: (status, record) => {
        // 处理 API 错误的情况
        if (record.salt_status_error) {
          return (
            <Tooltip title={record.salt_status_error}>
              <Tag color="default" icon={<WarningOutlined />}>
                {t('slurm.apiError')}
              </Tag>
            </Tooltip>
          );
        }
        
        // 处理未配置或未知状态
        if (!status || status === 'unknown' || status === 'not_configured') {
          return (
            <Tooltip title={t('slurm.notConfiguredTooltip')}>
              <Tag color="default" icon={<CloseCircleOutlined />}>
                {t('slurm.notConfigured')}
              </Tag>
            </Tooltip>
          );
        }
        
        const statusConfig = {
          'accepted': { color: 'green', icon: <CheckCircleOutlined />, text: t('slurm.connected') },
          'pending': { color: 'orange', icon: <HourglassOutlined />, text: t('slurm.pending') },
          'rejected': { color: 'red', icon: <CloseCircleOutlined />, text: t('slurm.rejected') },
          'denied': { color: 'red', icon: <CloseCircleOutlined />, text: t('slurm.rejected') },
        };
        
        const config = statusConfig[status] || { 
          color: 'default', 
          icon: <CloseCircleOutlined />,
          text: t('slurm.notConfigured')
        };
        
        return (
          <Tag color={config.color} icon={config.icon}>
            {config.text}
            {record.salt_minion_id && record.salt_minion_id !== 'unknown' && (
              <Text type="secondary" style={{ marginLeft: 4, fontSize: '12px' }}>
                ({record.salt_minion_id})
              </Text>
            )}
          </Tag>
        );
      }
    },
    { title: 'CPU', dataIndex: 'cpus', key: 'cpus' },
    { title: t('slurm.memoryMB'), dataIndex: 'memory_mb', key: 'memory_mb' },
  ];

  const columnsJobs = [
    { title: t('slurm.jobId'), dataIndex: 'id', key: 'id' },
    { title: t('slurm.jobName'), dataIndex: 'name', key: 'name' },
    { title: t('slurm.user'), dataIndex: 'user', key: 'user' },
    { title: t('slurm.partition'), dataIndex: 'partition', key: 'partition' },
    { title: t('slurm.status'), dataIndex: 'state', key: 'state', render: (s) => <Tag color={s === 'RUNNING' ? 'blue' : s === 'PENDING' ? 'orange' : 'default'}>{s}</Tag> },
    { title: t('slurm.elapsed'), dataIndex: 'elapsed', key: 'elapsed' },
    { title: t('slurm.nodeCount'), dataIndex: 'nodes', key: 'nodes' },
    { title: t('slurm.reason'), dataIndex: 'reason', key: 'reason' },
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space direction="vertical" size="middle" style={{ width: '100%' }}>
        <Title level={2}>{t('slurm.title')}</Title>
        
        {error && (
          <Alert 
            type="error" 
            showIcon 
            message={t('slurm.loadFailed')}
            description={
              <Space>
                <span>{t('slurm.loadFailedDesc')}</span>
                <Button size="small" onClick={load}>{t('common.retry')}</Button>
              </Space>
            }
          />
        )}
        {demo && (
          <Alert type="info" showIcon message={t('slurm.demoMode')} />
        )}

        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          {/* 概览页签 */}
          <TabPane tab={t('slurm.clusterOverview')} key="overview">

        <Row gutter={16}>
          <Col span={6}>
            <Card>
              <Statistic title={t('slurm.totalNodes')} value={summary?.nodes_total || 0} loading={loading} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title={t('slurm.idleNodes')} value={summary?.nodes_idle || 0} loading={loading} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title={t('slurm.allocNodes')} value={summary?.nodes_alloc || 0} loading={loading} />
            </Card>
          </Col>
          <Col span={6}>
            <Card>
              <Statistic title={t('slurm.partitionCount')} value={summary?.partitions || 0} loading={loading} />
            </Card>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={8}>
            <Card>
              <Statistic title={t('slurm.runningJobs')} value={summary?.jobs_running || 0} loading={loading} />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic title={t('slurm.pendingJobs')} value={summary?.jobs_pending || 0} loading={loading} />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic title={t('slurm.otherJobs')} value={summary?.jobs_other || 0} loading={loading} />
            </Card>
          </Col>
        </Row>

        <Divider />

        {/* SaltStack 集成状态 - 始终显示，即使数据未加载也显示状态 */}
        <Card 
          title={
            <Space>
              <CloudServerOutlined />
              <span>{t('slurm.saltStackIntegration')}</span>
              {saltStackData?.enabled && (
                <Tag color="green" icon={<CheckCircleOutlined />}>{t('slurm.enabled')}</Tag>
              )}
              {saltStackData && !saltStackData.enabled && (
                <Tag color="default">{t('slurm.notEnabled')}</Tag>
              )}
              {!saltStackData && (
                <Tag color="orange" icon={<SyncOutlined spin />}>{t('common.loading')}</Tag>
              )}
            </Space>
          }
          extra={
            <Space>
              {saltStackLoading && <Spin size="small" />}
              <Button 
                size="small" 
                icon={<ReloadOutlined />} 
                onClick={loadSaltStackIntegration}
              >
                {t('common.refresh')}
              </Button>
            </Space>
          }
          style={{ marginBottom: '16px' }}
        >
          {!saltStackData && !saltStackLoading && (
            <Alert
              message={t('slurm.saltStackLoadFailed')}
              description={t('slurm.saltStackLoadFailedDesc')}
              type="warning"
              showIcon
            />
          )}
          
          {saltStackData && (
            <>
              {/* Master 和 API 状态 */}
              <Row gutter={16} style={{ marginBottom: '16px' }}>
                <Col span={8}>
                  <Card size="small">
                    <Statistic
                      title={t('slurm.masterStatus')}
                      value={saltStackData.master_status || t('slurm.unknown')}
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
                      title={t('slurm.apiStatus')}
                      value={saltStackData.api_status || t('slurm.unknown')}
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
                      title={t('slurm.activeJobs')}
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
                    title={t('slurm.onlineMinions')}
                    value={saltStackData.minions?.online || 0}
                    valueStyle={{ color: '#3f8600' }}
                    prefix={<CheckCircleOutlined />}
                  />
                </Col>
                <Col span={8}>
                  <Statistic
                    title={t('slurm.offlineMinions')}
                    value={saltStackData.minions?.offline || 0}
                    valueStyle={{ color: '#cf1322' }}
                  />
                </Col>
                <Col span={8}>
                  <Statistic
                    title={t('slurm.totalMinions')}
                    value={saltStackData.minions?.total || 0}
                    prefix={<HddOutlined />}
                  />
                </Col>
              </Row>

              {saltStackData.minion_list && saltStackData.minion_list.length > 0 && (
                <div style={{ marginTop: '16px' }}>
                  <Text strong>{t('slurm.minionList')}:</Text>
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
            </>
          )}
        </Card>

        <Card 
          title={t('slurm.nodesList')} 
          extra={
            <Space>
              {selectedRowKeys.length > 0 && (
                <>
                  <Text type="secondary">{t('slurm.selectedNodes', { count: selectedRowKeys.length })}</Text>
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
                      {t('slurm.nodeOperation')}
                    </Button>
                  </Dropdown>
                </>
              )}
              <Button 
                icon={<ReloadOutlined />} 
                onClick={load}
                loading={loading}
                size="small"
              >
                {t('slurm.refreshStatus')}
              </Button>
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
          title={t('slurm.jobQueue')}
          extra={
            <Space>
              {selectedJobKeys.length > 0 && (
                <>
                  <Text type="secondary">{t('slurm.selectedJobs', { count: selectedJobKeys.length })}</Text>
                  <Dropdown menu={{ items: jobOperationMenuItems }}>
                    <Button loading={operationLoading}>
                      {t('slurm.jobOperation')} <DownOutlined />
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
          <TabPane tab={t('slurm.clusterStatusMonitor')} key="cluster-status">
            <SlurmClusterStatus />
          </TabPane>

          {/* SaltStack 命令执行页签 */}
          <TabPane tab={t('slurm.saltStackCommand')} key="salt-commands">
            <SaltCommandExecutor />
          </TabPane>
        </Tabs>
      </Space>
    </div>
  );
};

export default SlurmDashboard;
