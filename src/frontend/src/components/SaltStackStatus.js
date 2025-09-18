import React, { useEffect, useState } from 'react';
import { Card, Statistic, Tag, Space, Alert, Spin, Button, Tabs, List, Typography, Progress, Descriptions, Badge } from 'antd';
import { 
  CheckCircleOutlined, 
  ExclamationCircleOutlined, 
  ClockCircleOutlined, 
  DesktopOutlined,
  SettingOutlined,
  PlayCircleOutlined,
  ReloadOutlined
} from '@ant-design/icons';
import { saltStackAPI } from '../services/api';

const { Text, Paragraph } = Typography;
const { TabPane } = Tabs;

const SaltStackStatus = () => {
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

  const loadAll = async () => {
    setLoading(true);
    try {
      await Promise.all([loadStatus(), loadMinions(), loadJobs()]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAll();
    const interval = setInterval(loadAll, 60000); // 每60秒刷新，减少对后端的压力
    return () => clearInterval(interval);
  }, []);

  const getStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'connected':
      case 'up':
      case 'running':
        return 'success';
      case 'demo':
        return 'processing';
      case 'down':
      case 'disconnected':
        return 'error';
      default:
        return 'default';
    }
  };

  const getStatusIcon = (status) => {
    switch (status?.toLowerCase()) {
      case 'connected':
      case 'up':
      case 'running':
        return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
      case 'demo':
        return <ExclamationCircleOutlined style={{ color: '#1890ff' }} />;
      case 'down':
      case 'disconnected':
        return <ExclamationCircleOutlined style={{ color: '#f5222d' }} />;
      default:
        return <ClockCircleOutlined style={{ color: '#d9d9d9' }} />;
    }
  };

  const formatUptime = (uptime) => {
    if (!uptime) return '未知';
    const hours = Math.floor(uptime / 3600);
    const minutes = Math.floor((uptime % 3600) / 60);
    if (hours > 24) {
      const days = Math.floor(hours / 24);
      return `${days}天 ${hours % 24}小时`;
    }
    return `${hours}小时 ${minutes}分钟`;
  };

  const formatTimestamp = (timestamp) => {
    if (!timestamp) return '未知';
    return new Date(timestamp).toLocaleString('zh-CN');
  };

  if (loading && !status) {
    return (
      <div style={{ textAlign: 'center', padding: 20 }}>
        <Spin size="large" />
        <div style={{ marginTop: 16 }}>加载SaltStack状态...</div>
      </div>
    );
  }

  return (
    <Space direction="vertical" size="middle" style={{ width: '100%' }}>
      {error && (
        <Alert 
          type="error" 
          showIcon 
          size="small"
          message="连接失败"
          description={
            <Space>
              <span>无法连接到SaltStack</span>
              <Button size="small" onClick={loadAll}>重试</Button>
            </Space>
          }
        />
      )}

      {demo && (
        <Alert 
          type="info" 
          showIcon 
          size="small"
          message="演示模式" 
          description="显示SaltStack演示数据"
        />
      )}

      <Tabs size="small" type="card" style={{ marginTop: -8 }}>
        <TabPane tab={<span><DesktopOutlined />状态</span>} key="status">
          <Space direction="vertical" size="small" style={{ width: '100%' }}>
            <Card size="small">
              <Space>
                {getStatusIcon(status?.status)}
                <Text strong>
                  {status?.status === 'demo' ? '演示模式' : 
                   status?.status === 'connected' ? '已连接' : 
                   status?.status || '未知'}
                </Text>
                <Button 
                  size="small" 
                  icon={<ReloadOutlined />} 
                  onClick={loadStatus}
                  loading={loading}
                />
              </Space>
            </Card>

            <Card size="small" title="基本信息">
              <Descriptions size="small" column={1}>
                <Descriptions.Item label="Master版本">
                  {status?.master_version || '未知'}
                </Descriptions.Item>
                <Descriptions.Item label="API版本">
                  {status?.api_version || '未知'}
                </Descriptions.Item>
                <Descriptions.Item label="运行时长">
                  {formatUptime(status?.uptime)}
                </Descriptions.Item>
                <Descriptions.Item label="连接Minions">
                  <Badge count={status?.connected_minions || 0} showZero color="#52c41a" />
                </Descriptions.Item>
                <Descriptions.Item label="最后更新">
                  {formatTimestamp(status?.last_updated)}
                </Descriptions.Item>
              </Descriptions>
            </Card>

            <Card size="small" title="服务状态">
              <Space direction="vertical" size="small" style={{ width: '100%' }}>
                {status?.services && Object.entries(status.services).map(([service, state]) => (
                  <div key={service} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Text>{service}</Text>
                    <Tag color={getStatusColor(state)}>
                      {state === 'running' ? '运行中' : 
                       state === 'stopped' ? '已停止' : state}
                    </Tag>
                  </div>
                ))}
              </Space>
            </Card>

            <Card size="small" title="密钥统计">
              <Space direction="vertical" size="small" style={{ width: '100%' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Text>已接受</Text>
                  <Badge count={status?.accepted_keys?.length || 0} showZero color="#52c41a" />
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Text>待接受</Text>
                  <Badge count={status?.unaccepted_keys?.length || 0} showZero color="#fa8c16" />
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Text>已拒绝</Text>
                  <Badge count={status?.rejected_keys?.length || 0} showZero color="#f5222d" />
                </div>
              </Space>
            </Card>
          </Space>
        </TabPane>

        <TabPane tab={<span><SettingOutlined />Minions</span>} key="minions">
          <List 
            size="small"
            dataSource={minions}
            renderItem={minion => (
              <List.Item>
                <List.Item.Meta
                  avatar={getStatusIcon(minion.status)}
                  title={<Text strong>{minion.id}</Text>}
                  description={
                    <Space direction="vertical" size={2}>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        {minion.os} {minion.os_version} ({minion.architecture})
                      </Text>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        最后连接: {new Date(minion.last_seen).toLocaleString('zh-CN')}
                      </Text>
                      {minion.grains?.roles && Array.isArray(minion.grains.roles) && (
                        <Space size={4}>
                          {minion.grains.roles.map(role => (
                            <Tag key={role} size="small">{role}</Tag>
                          ))}
                        </Space>
                      )}
                    </Space>
                  }
                />
              </List.Item>
            )}
          />
        </TabPane>

        <TabPane tab={<span><PlayCircleOutlined />作业</span>} key="jobs">
          <List 
            size="small"
            dataSource={jobs}
            renderItem={job => (
              <List.Item>
                <List.Item.Meta
                  title={
                    <Space>
                      <Text strong>{job.jid}</Text>
                      <Tag color={getStatusColor(job.status)}>
                        {job.status === 'completed' ? '已完成' : 
                         job.status === 'running' ? '运行中' : 
                         job.status === 'failed' ? '失败' : job.status}
                      </Tag>
                    </Space>
                  }
                  description={
                    <Space direction="vertical" size={2}>
                      <Text style={{ fontSize: 12 }}>
                        <Text strong>功能:</Text> {job.function}
                      </Text>
                      <Text style={{ fontSize: 12 }}>
                        <Text strong>目标:</Text> {job.target}
                      </Text>
                      <Text style={{ fontSize: 12 }}>
                        <Text strong>用户:</Text> {job.user}
                      </Text>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        开始: {new Date(job.start_time).toLocaleString('zh-CN')}
                      </Text>
                      {job.end_time && (
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          结束: {new Date(job.end_time).toLocaleString('zh-CN')}
                        </Text>
                      )}
                    </Space>
                  }
                />
              </List.Item>
            )}
          />
        </TabPane>
      </Tabs>
    </Space>
  );
};

export default SaltStackStatus;
