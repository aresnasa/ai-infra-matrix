import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Space, Alert, Spin, Button, Typography, Divider } from 'antd';
import { slurmAPI, saltStackAPI } from '../services/api';
import { CloudServerOutlined, HddOutlined, CheckCircleOutlined, SyncOutlined } from '@ant-design/icons';

const { Title, Text } = Typography;

const columnsNodes = [
  { title: '节点', dataIndex: 'name', key: 'name' },
  { title: '分区', dataIndex: 'partition', key: 'partition' },
  { title: '状态', dataIndex: 'state', key: 'state', render: (s) => <Tag color={s.toLowerCase().includes('idle') ? 'green' : s.toLowerCase().includes('alloc') ? 'blue' : 'orange'}>{s}</Tag> },
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

const SlurmDashboard = () => {
  const [summary, setSummary] = useState(null);
  const [nodes, setNodes] = useState([]);
  const [jobs, setJobs] = useState([]);
  const [saltStackData, setSaltStackData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [saltStackLoading, setSaltStackLoading] = useState(false);
  const [demo, setDemo] = useState(false);
  const [error, setError] = useState(null);

  const load = async () => {
    setLoading(true);
    try {
      const [s, n, j] = await Promise.all([
        slurmAPI.getSummary(),
        slurmAPI.getNodes(),
        slurmAPI.getJobs(),
      ]);
      setSummary(s.data?.data);
      setNodes(n.data?.data || []);
      setJobs(j.data?.data || []);
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
            <Row gutter={16}>
              <Col span={6}>
                <Statistic
                  title="Minion 总数"
                  value={saltStackData.minions?.total || 0}
                  prefix={<HddOutlined />}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="在线 Minion"
                  value={saltStackData.minions?.online || 0}
                  valueStyle={{ color: '#3f8600' }}
                  prefix={<CheckCircleOutlined />}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="离线 Minion"
                  value={saltStackData.minions?.offline || 0}
                  valueStyle={{ color: '#cf1322' }}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="最近任务"
                  value={saltStackData.recent_jobs || 0}
                  prefix={<SyncOutlined />}
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
                        color={minion.status === 'online' ? 'green' : 'default'}
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

        <Card title="节点列表" extra={!loading ? null : <Spin size="small" />}>
          <Table rowKey="name" dataSource={nodes} columns={columnsNodes} size="small" pagination={{ pageSize: 8 }} />
        </Card>

        <Card title="作业队列" extra={!loading ? null : <Spin size="small" />}>
          <Table rowKey="id" dataSource={jobs} columns={columnsJobs} size="small" pagination={{ pageSize: 8 }} />
        </Card>
      </Space>
    </div>
  );
};

export default SlurmDashboard;
