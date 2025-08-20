import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Space, Alert, Spin, Button, Layout, Typography, Progress, Tabs } from 'antd';
import { slurmAPI } from '../services/api';
import SaltStackStatus from '../components/SaltStackStatus';

const { Sider, Content } = Layout;
const { Title } = Typography;

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
  const [loading, setLoading] = useState(true);
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

  useEffect(() => {
    load();
    const t = setInterval(load, 15000);
    return () => clearInterval(t);
  }, []);

  return (
    <Layout style={{ minHeight: '100vh', background: '#f0f2f5' }}>
      <Content style={{ padding: 24 }}>
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

          <Card title="节点列表" extra={!loading ? null : <Spin size="small" />}>
            <Table rowKey="name" dataSource={nodes} columns={columnsNodes} size="small" pagination={{ pageSize: 8 }} />
          </Card>

          <Card title="作业队列" extra={!loading ? null : <Spin size="small" />}>
            <Table rowKey="id" dataSource={jobs} columns={columnsJobs} size="small" pagination={{ pageSize: 8 }} />
          </Card>
        </Space>
      </Content>
      
      <Sider width={400} style={{ background: '#fff', boxShadow: '-2px 0 8px rgba(0,0,0,0.1)' }}>
        <div style={{ padding: '24px 16px' }}>
          <Title level={4} style={{ marginBottom: 16 }}>SaltStack 状态</Title>
          <SaltStackStatus />
        </div>
      </Sider>
    </Layout>
  );
};

export default SlurmDashboard;
