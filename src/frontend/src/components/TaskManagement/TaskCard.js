import React from 'react';
import { Card, Tag, Progress, Space, Button, Tooltip } from 'antd';
import {
  EyeOutlined, ReloadOutlined, StopOutlined, RedoOutlined,
  CheckCircleOutlined, ExclamationCircleOutlined, ClockCircleOutlined,
  PlayCircleOutlined
} from '@ant-design/icons';

const { Meta } = Card;

// 状态颜色和图标映射
const getStatusConfig = (status) => {
  const configs = {
    pending: { color: 'default', icon: <PlayCircleOutlined /> },
    running: { color: 'blue', icon: <ClockCircleOutlined /> },
    completed: { color: 'green', icon: <CheckCircleOutlined /> },
    failed: { color: 'red', icon: <ExclamationCircleOutlined /> },
    cancelled: { color: 'orange', icon: <StopOutlined /> },
  };
  return configs[status] || configs.pending;
};

// 任务类型映射
const getTaskTypeLabel = (type) => {
  const types = {
    scale_up: '扩容',
    scale_down: '缩容',
    node_init: '节点初始化',
    cluster_setup: '集群配置',
  };
  return types[type] || type;
};

// 格式化持续时间
const formatDuration = (seconds) => {
  if (seconds < 60) return `${Math.round(seconds)}秒`;
  if (seconds < 3600) return `${Math.round(seconds / 60)}分钟`;
  return `${Math.round(seconds / 3600)}小时`;
};

const TaskCard = ({ 
  task, 
  onViewDetail, 
  onRefresh, 
  onCancel, 
  onRetry 
}) => {
  const statusConfig = getStatusConfig(task.status);

  const actions = [
    <Tooltip title="查看详情">
      <Button 
        size="small" 
        icon={<EyeOutlined />} 
        onClick={() => onViewDetail(task)}
      />
    </Tooltip>
  ];

  if (task.status === 'running') {
    actions.push(
      <Tooltip title="刷新进度">
        <Button 
          size="small" 
          icon={<ReloadOutlined />} 
          onClick={() => onRefresh(task.id)}
        />
      </Tooltip>,
      <Tooltip title="取消任务">
        <Button 
          size="small" 
          danger
          icon={<StopOutlined />} 
          onClick={() => onCancel(task.id)}
        />
      </Tooltip>
    );
  }

  if (task.status === 'failed' || task.status === 'cancelled') {
    actions.push(
      <Tooltip title="重试任务">
        <Button 
          size="small" 
          icon={<RedoOutlined />} 
          onClick={() => onRetry(task.id)}
        />
      </Tooltip>
    );
  }

  // 后端返回的progress已经是百分比(0-100)，不需要再乘100
  const progress = task.status === 'running' && task.progress !== undefined 
    ? Math.round(task.progress) 
    : task.status === 'completed' ? 100 : 0;

  const duration = task.created_at 
    ? formatDuration(
        task.completed_at 
          ? new Date(task.completed_at) - new Date(task.created_at)
          : Date.now() - new Date(task.created_at)
      ) / 1000
    : '-';

  return (
    <Card
      size="small"
      actions={actions}
      extra={
        <Space>
          <Tag color={statusConfig.color}>
            {statusConfig.icon}
            {task.status}
          </Tag>
        </Space>
      }
    >
      <Meta
        title={
          <Space>
            <span>{task.name}</span>
            <Tag color="blue" size="small">
              {getTaskTypeLabel(task.type)}
            </Tag>
          </Space>
        }
        description={
          <div>
            <div style={{ marginBottom: '8px' }}>
              <span>集群: {task.cluster_name || '-'}</span>
              {' • '}
              <span>用户: {task.username || '-'}</span>
            </div>
            <div style={{ marginBottom: '8px' }}>
              <span>持续时间: {duration}</span>
              {task.retry_count > 0 && (
                <>
                  {' • '}
                  <span>重试: {task.retry_count}/{task.max_retries || 3}</span>
                </>
              )}
            </div>
            {task.status === 'running' && (
              <Progress 
                percent={progress} 
                size="small" 
                status="active"
              />
            )}
            {task.error_message && (
              <div style={{ color: '#ff4d4f', fontSize: '12px', marginTop: '4px' }}>
                {task.error_message.substring(0, 100)}...
              </div>
            )}
          </div>
        }
      />
    </Card>
  );
};

export default TaskCard;