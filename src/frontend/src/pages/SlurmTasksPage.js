import React, { useEffect, useState } from 'react';
import {
  Card, Row, Col, Table, Tag, Space, Alert, Spin, Button,
  Typography, Progress, Descriptions, Badge, Tooltip, Modal,
  List, Divider, message
} from 'antd';
import {
  ReloadOutlined, EyeOutlined, CheckCircleOutlined,
  ExclamationCircleOutlined, ClockCircleOutlined, PlayCircleOutlined,
  StopOutlined, ThunderboltOutlined, DesktopOutlined
} from '@ant-design/icons';
import { slurmAPI } from '../services/api';

const { Title, Text, Paragraph } = Typography;

// 任务状态颜色映射
const getTaskStatusColor = (status) => {
  switch (status) {
    case 'running': return 'blue';
    case 'complete': return 'green';
    case 'failed': return 'red';
    default: return 'default';
  }
};

// 任务状态图标映射
const getTaskStatusIcon = (status) => {
  switch (status) {
    case 'running': return <ClockCircleOutlined />;
    case 'complete': return <CheckCircleOutlined />;
    case 'failed': return <ExclamationCircleOutlined />;
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

const SlurmTasksPage = () => {
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedTask, setSelectedTask] = useState(null);
  const [taskDetailModal, setTaskDetailModal] = useState(false);
  const [taskProgress, setTaskProgress] = useState(null);

  // 表格列定义
  const columns = [
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      render: (name, record) => (
        <Space>
          {getTaskStatusIcon(record.status)}
          <Text strong>{name}</Text>
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={getTaskStatusColor(status)}>
          {status === 'running' ? '运行中' :
           status === 'complete' ? '已完成' :
           status === 'failed' ? '失败' : status}
        </Tag>
      ),
    },
    {
      title: '当前步骤',
      dataIndex: 'current_step',
      key: 'current_step',
      render: (step) => step || '-',
    },
    {
      title: '进度',
      dataIndex: 'progress',
      key: 'progress',
      render: (progress, record) => {
        if (record.status === 'running' && progress !== undefined) {
          return <Progress percent={Math.round(progress * 100)} size="small" />;
        }
        return record.status === 'complete' ? '100%' : '-';
      },
    },
    {
      title: '持续时间',
      dataIndex: 'duration',
      key: 'duration',
      render: (duration) => formatDuration(duration),
    },
    {
      title: '开始时间',
      dataIndex: 'started_at',
      key: 'started_at',
      render: (timestamp) => formatTimestamp(timestamp),
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
            <Tooltip title="刷新进度">
              <Button
                size="small"
                icon={<ReloadOutlined />}
                onClick={() => handleRefreshTask(record.id)}
              />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  // 加载任务列表
  const loadTasks = async () => {
    setLoading(true);
    try {
      const response = await slurmAPI.getTasks();
      setTasks(response.data?.data || []);
      setError(null);
    } catch (e) {
      console.error('加载任务列表失败', e);
      setError(e.message || '加载任务列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 查看任务详情
  const handleViewTaskDetail = async (task) => {
    setSelectedTask(task);
    setTaskDetailModal(true);

    // 如果任务正在运行，获取实时进度
    if (task.status === 'running') {
      try {
        const response = await slurmAPI.getProgress(task.id);
        setTaskProgress(response.data?.data);
      } catch (e) {
        console.error('获取任务进度失败', e);
        message.error('获取任务进度失败');
      }
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

  // 关闭任务详情模态框
  const handleCloseTaskDetail = () => {
    setTaskDetailModal(false);
    setSelectedTask(null);
    setTaskProgress(null);
  };

  useEffect(() => {
    loadTasks();

    // 设置定时刷新（每30秒）
    const interval = setInterval(() => {
      if (tasks.some(task => task.status === 'running')) {
        loadTasks();
      }
    }, 30000);

    return () => clearInterval(interval);
  }, []);

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
        />
      )}

      <Card
        title={
          <Space>
            <DesktopOutlined />
            任务列表
            <Button
              icon={<ReloadOutlined />}
              onClick={loadTasks}
              loading={loading}
              size="small"
            >
              刷新
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={tasks}
          rowKey="id"
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          locale={{
            emptyText: '暂无任务记录',
          }}
        />
      </Card>

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
          <Button key="close" onClick={handleCloseTaskDetail}>
            关闭
          </Button>
        ]}
        width={800}
      >
        {selectedTask && (
          <div>
            <Descriptions bordered column={2}>
              <Descriptions.Item label="任务ID">{selectedTask.id}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Badge
                  status={selectedTask.status === 'running' ? 'processing' :
                          selectedTask.status === 'complete' ? 'success' : 'error'}
                  text={selectedTask.status === 'running' ? '运行中' :
                        selectedTask.status === 'complete' ? '已完成' : '失败'}
                />
              </Descriptions.Item>
              <Descriptions.Item label="开始时间">
                {formatTimestamp(selectedTask.started_at)}
              </Descriptions.Item>
              <Descriptions.Item label="持续时间">
                {formatDuration(selectedTask.duration)}
              </Descriptions.Item>
              <Descriptions.Item label="当前步骤" span={2}>
                {selectedTask.current_step || '无'}
              </Descriptions.Item>
              <Descriptions.Item label="最后消息" span={2}>
                {selectedTask.last_message || '无'}
              </Descriptions.Item>
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

            {taskProgress && (
              <div style={{ marginTop: '16px' }}>
                <Title level={4}>执行日志</Title>
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
                            {new Date(event.ts).toLocaleString()}
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
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default SlurmTasksPage;