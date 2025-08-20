import React, { useEffect, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Space, Alert, Spin, Button, Layout, Typography, List, Progress, Descriptions, Badge, Tabs } from 'antd';
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
  ApiOutlined
} from '@ant-design/icons';
import { saltStackAPI } from '../services/api';

const { Content } = Layout;
const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;

const SaltStackDashboard = () => {
  const [status, setStatus] = useState(null);
  const [minions, setMinions] = useState([]);
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [demo, setDemo] = useState(false);
  const [error, setError] = useState(null);

  const loadStatus = async () => {
    try {
      const response = await saltStackAPI.getStatus();
      setStatus(response.data?.data);
      setDemo(Boolean(response.data?.data?.demo));
      setError(null);
    } catch (e) {
      console.error('加载SaltStack状态失败', e);
      setError(e);
    }
  };

  const loadMinions = async () => {
    try {
      const response = await saltStackAPI.getMinions();
      setMinions(response.data?.data || []);
      setDemo(prev => prev || Boolean(response.data?.demo));
    } catch (e) {
      console.error('加载SaltStack Minions失败', e);
    }
  };

  const loadJobs = async () => {
    try {
      const response = await saltStackAPI.getJobs(10);
      setJobs(response.data?.data || []);
      setDemo(prev => prev || Boolean(response.data?.demo));
    } catch (e) {
      console.error('加载SaltStack Jobs失败', e);
    }
  };

  const loadData = async () => {
    setLoading(true);
    try {
      await Promise.all([loadStatus(), loadMinions(), loadJobs()]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 60000); // 60秒刷新一次，减少对后端的压力
    return () => clearInterval(interval);
  }, []);

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

  if (loading && !status) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" />
        <div style={{ marginTop: 16 }}>加载SaltStack状态...</div>
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
                  <Button size="small" onClick={loadData}>重试</Button>
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

          {/* 状态概览 */}
          <Row gutter={16}>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="Master状态" 
                  value={status?.master_status || '未知'} 
                  prefix={<SettingOutlined />}
                  valueStyle={{ color: status?.master_status === 'running' ? '#3f8600' : '#cf1322' }}
                  loading={loading}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="在线Minions" 
                  value={status?.minions_up || 0} 
                  prefix={<DesktopOutlined />}
                  valueStyle={{ color: '#3f8600' }}
                  loading={loading}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="离线Minions" 
                  value={status?.minions_down || 0} 
                  prefix={<ExclamationCircleOutlined />}
                  valueStyle={{ color: '#cf1322' }}
                  loading={loading}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="API状态" 
                  value={status?.api_status || '未知'} 
                  prefix={<ApiOutlined />}
                  valueStyle={{ color: status?.api_status === 'running' ? '#3f8600' : '#cf1322' }}
                  loading={loading}
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
                    <Card title="Master信息" size="small">
                      <Descriptions size="small" column={1}>
                        <Descriptions.Item label="版本">
                          {status?.salt_version || '未知'}
                        </Descriptions.Item>
                        <Descriptions.Item label="启动时间">
                          {status?.uptime || '未知'}
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
                    <Card title="性能指标" size="small">
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
                    </Card>
                  </Col>
                </Row>
              </TabPane>

              <TabPane tab="Minions管理" key="minions" icon={<DesktopOutlined />}>
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
                          <Tag color={getStatusColor(minion.status)}>
                            {minion.status || '未知'}
                          </Tag>
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
              </TabPane>

              <TabPane tab="作业历史" key="jobs" icon={<PlayCircleOutlined />}>
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
              </TabPane>
            </Tabs>
          </Card>

          {/* 操作按钮 */}
          <Card>
            <Space>
              <Button 
                type="primary" 
                icon={<ReloadOutlined />} 
                onClick={loadData}
                loading={loading}
              >
                刷新数据
              </Button>
              <Button 
                icon={<PlayCircleOutlined />}
                onClick={() => {
                  // TODO: 实现执行Salt命令功能
                  console.log('执行Salt命令功能待实现');
                }}
              >
                执行命令
              </Button>
              <Button 
                icon={<SettingOutlined />}
                onClick={() => {
                  // TODO: 实现配置管理功能
                  console.log('配置管理功能待实现');
                }}
              >
                配置管理
              </Button>
            </Space>
          </Card>
        </Space>
      </Content>
    </Layout>
  );
};

export default SaltStackDashboard;
