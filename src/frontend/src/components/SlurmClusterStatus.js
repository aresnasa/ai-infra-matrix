import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Table,
  Tag,
  Space,
  Button,
  Progress,
  Typography,
  Timeline,
  Alert,
  Spin,
  Descriptions
} from 'antd';
import {
  ClusterOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
  WarningOutlined,
  ClockCircleOutlined,
  ReloadOutlined,
  NodeIndexOutlined,
  HddOutlined
} from '@ant-design/icons';
import { slurmAPI, saltStackAPI } from '../services/api';

const { Text, Title } = Typography;

/**
 * SLURM 集群状态管理组件
 * 功能：
 * 1. 集群整体状态概览
 * 2. 节点健康度统计
 * 3. SaltStack 集成状态
 * 4. 集群操作历史
 */
const SlurmClusterStatus = () => {
  const [loading, setLoading] = useState(true);
  const [clusterSummary, setClusterSummary] = useState(null);
  const [nodes, setNodes] = useState([]);
  const [partitions, setPartitions] = useState([]);
  const [saltStackStatus, setSaltStackStatus] = useState(null);
  const [clusterHealth, setClusterHealth] = useState({ score: 0, level: 'unknown' });

  // 加载集群数据
  const loadClusterData = async () => {
    setLoading(true);
    try {
      const [summaryRes, nodesRes, partitionsRes] = await Promise.all([
        slurmAPI.getSummary(),
        slurmAPI.getNodes(),
        slurmAPI.getPartitions()
      ]);

      const summary = summaryRes.data?.data || {};
      const nodesList = nodesRes.data?.data || [];
      const partitionsList = partitionsRes.data?.data || [];

      setClusterSummary(summary);
      setNodes(nodesList);
      setPartitions(partitionsList);

      // 计算集群健康度
      calculateClusterHealth(summary, nodesList);
    } catch (error) {
      console.error('加载集群数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载 SaltStack 状态
  const loadSaltStackStatus = async () => {
    try {
      const response = await saltStackAPI.getSaltStackIntegration();
      setSaltStackStatus(response.data?.data || null);
    } catch (error) {
      console.error('加载 SaltStack 状态失败:', error);
    }
  };

  useEffect(() => {
    loadClusterData();
    loadSaltStackStatus();

    const interval = setInterval(() => {
      loadClusterData();
      loadSaltStackStatus();
    }, 30000); // 每30秒刷新

    return () => clearInterval(interval);
  }, []);

  // 计算集群健康度
  const calculateClusterHealth = (summary, nodesList) => {
    let score = 100;
    let issues = [];

    // 节点在线率
    const totalNodes = nodesList.length;
    const onlineNodes = nodesList.filter(n => 
      n.state?.toLowerCase().includes('idle') || 
      n.state?.toLowerCase().includes('alloc') ||
      n.state?.toLowerCase().includes('mix')
    ).length;
    
    if (totalNodes > 0) {
      const onlineRate = onlineNodes / totalNodes;
      if (onlineRate < 0.8) {
        score -= 20;
        issues.push('节点在线率低于80%');
      } else if (onlineRate < 0.9) {
        score -= 10;
        issues.push('节点在线率低于90%');
      }
    } else {
      score -= 30;
      issues.push('没有可用节点');
    }

    // 资源使用率
    if (summary.resources) {
      const cpuUsage = summary.resources.cpu_used / (summary.resources.cpu_total || 1);
      const memUsage = summary.resources.mem_used / (summary.resources.mem_total || 1);
      
      if (cpuUsage > 0.9) {
        score -= 15;
        issues.push('CPU使用率超过90%');
      }
      
      if (memUsage > 0.9) {
        score -= 15;
        issues.push('内存使用率超过90%');
      }
    }

    // 错误节点
    const errorNodes = nodesList.filter(n => 
      n.state?.toLowerCase().includes('down') ||
      n.state?.toLowerCase().includes('error') ||
      n.state?.toLowerCase().includes('fail')
    ).length;
    
    if (errorNodes > 0) {
      score -= errorNodes * 5;
      issues.push(`${errorNodes}个节点处于错误状态`);
    }

    // 确定健康等级
    let level = 'excellent';
    let color = '#52c41a';
    if (score >= 90) {
      level = 'excellent';
      color = '#52c41a';
    } else if (score >= 70) {
      level = 'good';
      color = '#1890ff';
    } else if (score >= 50) {
      level = 'warning';
      color = '#faad14';
    } else {
      level = 'critical';
      color = '#f5222d';
    }

    setClusterHealth({ score: Math.max(0, score), level, color, issues });
  };

  // 节点状态统计
  const getNodeStatistics = () => {
    const stats = {
      total: nodes.length,
      idle: 0,
      allocated: 0,
      mixed: 0,
      down: 0,
      drain: 0,
      other: 0
    };

    nodes.forEach(node => {
      const state = node.state?.toLowerCase() || '';
      if (state.includes('idle')) stats.idle++;
      else if (state.includes('alloc')) stats.allocated++;
      else if (state.includes('mix')) stats.mixed++;
      else if (state.includes('down')) stats.down++;
      else if (state.includes('drain')) stats.drain++;
      else stats.other++;
    });

    return stats;
  };

  const nodeStats = getNodeStatistics();

  // 分区表格列
  const partitionColumns = [
    {
      title: '分区名',
      dataIndex: 'name',
      key: 'name',
      render: (name) => <Text strong>{name}</Text>
    },
    {
      title: '可用性',
      dataIndex: 'available',
      key: 'available',
      render: (available) => (
        <Tag color={available === 'up' ? 'green' : 'red'}>
          {available === 'up' ? '可用' : '不可用'}
        </Tag>
      )
    },
    {
      title: '节点数',
      dataIndex: 'nodes',
      key: 'nodes',
      render: (nodes) => <Text>{nodes || 0}</Text>
    },
    {
      title: '节点列表',
      dataIndex: 'node_list',
      key: 'node_list',
      ellipsis: true,
      render: (list) => list || '-'
    },
    {
      title: '状态',
      dataIndex: 'state',
      key: 'state',
      render: (state) => {
        const stateMap = {
          'up': { color: 'green', text: '正常' },
          'down': { color: 'red', text: '下线' },
          'idle': { color: 'blue', text: '空闲' },
          'alloc': { color: 'orange', text: '已分配' }
        };
        const info = stateMap[state?.toLowerCase()] || { color: 'default', text: state };
        return <Tag color={info.color}>{info.text}</Tag>;
      }
    }
  ];

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '100px 0' }}>
        <Spin size="large" tip="加载集群状态..." />
      </div>
    );
  }

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {/* 刷新按钮 */}
      <div style={{ textAlign: 'right' }}>
        <Button
          icon={<ReloadOutlined />}
          onClick={() => {
            loadClusterData();
            loadSaltStackStatus();
          }}
          loading={loading}
        >
          刷新状态
        </Button>
      </div>

      {/* 集群健康度 */}
      <Card
        title={
          <Space>
            <ClusterOutlined />
            <span>集群健康度</span>
          </Space>
        }
      >
        <Row gutter={[16, 16]}>
          <Col span={6}>
            <div style={{ textAlign: 'center' }}>
              <Progress
                type="dashboard"
                percent={clusterHealth.score}
                strokeColor={clusterHealth.color}
                format={(percent) => `${percent.toFixed(0)}`}
              />
              <div style={{ marginTop: '16px' }}>
                <Tag color={clusterHealth.color} style={{ fontSize: '14px', padding: '4px 12px' }}>
                  {clusterHealth.level === 'excellent' && '优秀'}
                  {clusterHealth.level === 'good' && '良好'}
                  {clusterHealth.level === 'warning' && '警告'}
                  {clusterHealth.level === 'critical' && '严重'}
                </Tag>
              </div>
            </div>
          </Col>
          <Col span={18}>
            <Timeline
              items={[
                {
                  color: 'green',
                  children: (
                    <div>
                      <Text strong>节点总数:</Text> {nodeStats.total}
                      <Text type="secondary" style={{ marginLeft: '16px' }}>
                        在线: {nodeStats.idle + nodeStats.allocated + nodeStats.mixed}
                      </Text>
                    </div>
                  )
                },
                {
                  color: nodeStats.down > 0 || nodeStats.drain > 0 ? 'red' : 'green',
                  children: (
                    <div>
                      <Text strong>异常节点:</Text> {nodeStats.down + nodeStats.drain}
                      <Text type="secondary" style={{ marginLeft: '16px' }}>
                        Down: {nodeStats.down}, Drain: {nodeStats.drain}
                      </Text>
                    </div>
                  )
                },
                {
                  color: clusterHealth.issues.length > 0 ? 'orange' : 'green',
                  children: (
                    <div>
                      <Text strong>问题数量:</Text> {clusterHealth.issues.length}
                      {clusterHealth.issues.length > 0 && (
                        <div style={{ marginTop: '8px' }}>
                          {clusterHealth.issues.map((issue, index) => (
                            <Alert
                              key={index}
                              message={issue}
                              type="warning"
                              showIcon
                              icon={<WarningOutlined />}
                              style={{ marginTop: '4px' }}
                            />
                          ))}
                        </div>
                      )}
                    </div>
                  )
                }
              ]}
            />
          </Col>
        </Row>
      </Card>

      {/* 节点状态统计 */}
      <Card
        title={
          <Space>
            <NodeIndexOutlined />
            <span>节点状态统计</span>
          </Space>
        }
      >
        <Row gutter={16}>
          <Col span={4}>
            <Statistic
              title="总节点数"
              value={nodeStats.total}
              prefix={<HddOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Col>
          <Col span={4}>
            <Statistic
              title="空闲节点"
              value={nodeStats.idle}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Col>
          <Col span={4}>
            <Statistic
              title="已分配"
              value={nodeStats.allocated}
              prefix={<SyncOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Col>
          <Col span={4}>
            <Statistic
              title="混合状态"
              value={nodeStats.mixed}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#faad14' }}
            />
          </Col>
          <Col span={4}>
            <Statistic
              title="离线节点"
              value={nodeStats.down}
              prefix={<CloseCircleOutlined />}
              valueStyle={{ color: '#f5222d' }}
            />
          </Col>
          <Col span={4}>
            <Statistic
              title="排空节点"
              value={nodeStats.drain}
              prefix={<WarningOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Col>
        </Row>
      </Card>

      {/* 资源使用情况 */}
      {clusterSummary?.resources && (
        <Card
          title={
            <Space>
              <HddOutlined />
              <span>资源使用情况</span>
            </Space>
          }
        >
          <Row gutter={[16, 16]}>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="CPU 使用率"
                  value={(clusterSummary.resources.cpu_used / (clusterSummary.resources.cpu_total || 1) * 100).toFixed(1)}
                  suffix="%"
                  valueStyle={{
                    color: (clusterSummary.resources.cpu_used / clusterSummary.resources.cpu_total) > 0.8 ? '#f5222d' : '#1890ff'
                  }}
                />
                <Progress
                  percent={(clusterSummary.resources.cpu_used / (clusterSummary.resources.cpu_total || 1) * 100).toFixed(1)}
                  status={(clusterSummary.resources.cpu_used / clusterSummary.resources.cpu_total) > 0.8 ? 'exception' : 'active'}
                  style={{ marginTop: '12px' }}
                />
                <Text type="secondary" style={{ fontSize: '12px' }}>
                  {clusterSummary.resources.cpu_used} / {clusterSummary.resources.cpu_total} 核
                </Text>
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="内存使用率"
                  value={(clusterSummary.resources.mem_used / (clusterSummary.resources.mem_total || 1) * 100).toFixed(1)}
                  suffix="%"
                  valueStyle={{
                    color: (clusterSummary.resources.mem_used / clusterSummary.resources.mem_total) > 0.8 ? '#f5222d' : '#1890ff'
                  }}
                />
                <Progress
                  percent={(clusterSummary.resources.mem_used / (clusterSummary.resources.mem_total || 1) * 100).toFixed(1)}
                  status={(clusterSummary.resources.mem_used / clusterSummary.resources.mem_total) > 0.8 ? 'exception' : 'active'}
                  style={{ marginTop: '12px' }}
                />
                <Text type="secondary" style={{ fontSize: '12px' }}>
                  {(clusterSummary.resources.mem_used / 1024).toFixed(1)} / {(clusterSummary.resources.mem_total / 1024).toFixed(1)} GB
                </Text>
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="GPU 使用情况"
                  value={clusterSummary.resources.gpu_used || 0}
                  suffix={`/ ${clusterSummary.resources.gpu_total || 0}`}
                />
                {clusterSummary.resources.gpu_total > 0 && (
                  <>
                    <Progress
                      percent={((clusterSummary.resources.gpu_used || 0) / clusterSummary.resources.gpu_total * 100).toFixed(1)}
                      status="active"
                      style={{ marginTop: '12px' }}
                    />
                    <Text type="secondary" style={{ fontSize: '12px' }}>
                      可用: {clusterSummary.resources.gpu_total - (clusterSummary.resources.gpu_used || 0)} 张
                    </Text>
                  </>
                )}
              </Card>
            </Col>
          </Row>
        </Card>
      )}

      {/* 分区信息 */}
      <Card
        title={
          <Space>
            <ClusterOutlined />
            <span>分区信息</span>
          </Space>
        }
        extra={<Tag color="blue">{partitions.length} 个分区</Tag>}
      >
        {partitions.length > 0 ? (
          <Table
            dataSource={partitions}
            columns={partitionColumns}
            rowKey="name"
            pagination={false}
            size="small"
          />
        ) : (
          <Alert
            type="info"
            message="暂无分区信息"
            description="请检查 SLURM 配置或稍后重试"
            showIcon
          />
        )}
      </Card>

      {/* SaltStack 集成状态 */}
      {saltStackStatus && (
        <Card
          title={
            <Space>
              <SyncOutlined />
              <span>SaltStack 集成状态</span>
            </Space>
          }
        >
          <Descriptions bordered column={2} size="small">
            <Descriptions.Item label="Master 状态">
              <Tag color={saltStackStatus.master_status === 'connected' ? 'green' : 'red'}>
                {saltStackStatus.master_status || '未知'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="API 状态">
              <Tag color={saltStackStatus.api_status === 'available' ? 'green' : 'red'}>
                {saltStackStatus.api_status || '未知'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="在线 Minions">
              {saltStackStatus.minions?.online || 0}
            </Descriptions.Item>
            <Descriptions.Item label="离线 Minions">
              {saltStackStatus.minions?.offline || 0}
            </Descriptions.Item>
            <Descriptions.Item label="总 Minions">
              {saltStackStatus.minions?.total || 0}
            </Descriptions.Item>
            <Descriptions.Item label="最近作业">
              {saltStackStatus.recent_jobs || 0}
            </Descriptions.Item>
          </Descriptions>
        </Card>
      )}
    </Space>
  );
};

export default SlurmClusterStatus;
