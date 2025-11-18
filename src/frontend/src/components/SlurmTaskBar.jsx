import React, { useEffect, useMemo, useState, useImperativeHandle, forwardRef } from 'react';
import { Badge, Button, Empty, Popover, Space, Tag, Tooltip, Typography, Progress } from 'antd';
import { ThunderboltOutlined, ReloadOutlined, UnorderedListOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { slurmAPI } from '../services/api';
import { useNavigate } from 'react-router-dom';

dayjs.extend(relativeTime);

const { Text } = Typography;

/**
 * SlurmTaskBar
 * A lightweight task bar showing recent Slurm/operations tasks from /api/slurm/tasks.
 * - Polls every 10s by default (configurable via refreshInterval prop)
 * - Click to open a popover list and navigate to /slurm-tasks
 */

// Map backend status to AntD badge status
const badgeStatus = (s) => {
  const v = String(s || '').toLowerCase();
  if (v.includes('running') || v.includes('in_progress') || v.includes('processing')) return 'processing';
  if (v.includes('success') || v.includes('completed') || v.includes('done')) return 'success';
  if (v.includes('pending') || v.includes('waiting') || v.includes('queued')) return 'warning';
  if (v.includes('failed') || v.includes('error')) return 'error';
  return 'default';
};

const formatTask = (t) => {
  return {
    id: t.id,
    name: t.name || t.id,
    status: t.status,
    progress: t.progress || 0,
    currentStep: t.current_step || '',
    lastMessage: t.last_message || '',
    startedAt: t.started_at ? dayjs.unix(t.started_at) : null,
    completedAt: t.completed_at && t.completed_at > 0 ? dayjs.unix(t.completed_at) : null,
    latest: t.latest_event || null,
    duration: t.duration || '',
    errorMessage: t.error_message || '',
  };
};

const SlurmTaskBar = forwardRef(({ refreshInterval = 10000, maxItems = 8, style }, ref) => {
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const load = async () => {
    setLoading(true);
    
    // 检查token是否存在
    const token = localStorage.getItem('token');
    if (!token) {
      console.warn('SlurmTaskBar: No authentication token found in localStorage');
      setTasks([]);
      setLoading(false);
      return;
    }
    
    try {
      console.log('SlurmTaskBar: Loading tasks with token:', token.substring(0, 20) + '...');
      const res = await slurmAPI.getTasks();
      console.log('SLURM Tasks API Response:', res);
      
      // 检查嵌套的响应结构: res.data.data.tasks 
      if (res && res.data && res.data.data && res.data.data.tasks) {
        const data = res.data.data.tasks;
        console.log('Parsed tasks data:', data);
        const formattedTasks = data.map(formatTask);
        console.log('Formatted tasks:', formattedTasks);
        setTasks(formattedTasks);
      } else {
        console.warn('SlurmTaskBar: Invalid response format:', res);
        console.warn('SlurmTaskBar: Expected res.data.data.tasks, got:', res?.data);
        setTasks([]);
      }
    } catch (e) {
      console.error('SlurmTaskBar: Failed to load slurm tasks:', e);
      if (e.response) {
        console.error('SlurmTaskBar: Response status:', e.response.status);
        console.error('SlurmTaskBar: Response data:', e.response.data);
      }
      setTasks([]);
    } finally {
      setLoading(false);
    }
  };

  // 暴露 refresh 方法给父组件
  useImperativeHandle(ref, () => ({
    refresh: load
  }));

  useEffect(() => {
    console.log('SlurmTaskBar: Component mounted, loading tasks...');
    load();
    
    let interval;
    if (refreshInterval > 0) {
      interval = setInterval(() => {
        console.log('SlurmTaskBar: Auto-refreshing tasks...');
        load();
      }, refreshInterval);
    }
    
    // 监听localStorage变化（token更新）
    const handleStorageChange = (e) => {
      if (e.key === 'token') {
        console.log('SlurmTaskBar: Token changed, reloading tasks');
        load();
      }
    };
    
    window.addEventListener('storage', handleStorageChange);
    
    return () => {
      if (interval) clearInterval(interval);
      window.removeEventListener('storage', handleStorageChange);
    };
  }, [refreshInterval]);

  const items = useMemo(() => tasks.slice(0, maxItems), [tasks, maxItems]);

  const content = (
    <Space direction="vertical" size="small" style={{ maxWidth: 420 }}>
      {items.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无任务" />
      ) : (
        items.map((t) => (
          <div key={t.id} style={{ marginBottom: 8, padding: 8, border: '1px solid #f0f0f0', borderRadius: 4 }}>
            <Space style={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}>
              <Space>
                <Badge status={badgeStatus(t.status)} />
                <Text strong>{t.name}</Text>
                <Tag color={t.status === 'running' ? 'blue' : t.status === 'failed' ? 'red' : 'default'}>
                  {t.status}
                </Tag>
              </Space>
              <Space>
                {t.startedAt && <Text type="secondary">{t.startedAt.fromNow()}</Text>}
              </Space>
            </Space>
            {t.progress > 0 && (
              <div style={{ marginTop: 4 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px' }}>
                  <Text type="secondary">{t.currentStep}</Text>
                  <Text type="secondary">{Math.round(t.progress)}%</Text>
                </div>
                <div style={{ 
                  width: '100%', 
                  height: '4px', 
                  backgroundColor: '#f0f0f0', 
                  borderRadius: '2px',
                  overflow: 'hidden',
                  marginTop: '2px'
                }}>
                  <div style={{
                    width: `${Math.round(t.progress)}%`,
                    height: '100%',
                    backgroundColor: t.status === 'running' ? '#1890ff' : t.status === 'failed' ? '#ff4d4f' : '#52c41a',
                    transition: 'width 0.3s ease'
                  }} />
                </div>
              </div>
            )}
            {t.lastMessage && (
              <Text type="secondary" style={{ fontSize: '12px', display: 'block', marginTop: 4 }}>
                {t.lastMessage}
              </Text>
            )}
          </div>
        ))
      )}
      <Space style={{ width: '100%', justifyContent: 'space-between' }}>
        <Button icon={<ReloadOutlined />} size="small" onClick={load} loading={loading}>
          刷新
        </Button>
        <Button icon={<UnorderedListOutlined />} size="small" type="link" onClick={() => navigate('/slurm-tasks')}>
          查看全部
        </Button>
      </Space>
    </Space>
  );

  return (
    <div style={{ padding: 8, background: '#fff', borderRadius: 6, ...style }}>
      <Space>
        <ThunderboltOutlined />
        <Text strong>任务栏</Text>
        <Popover placement="bottomLeft" content={content} trigger="click">
          <Button size="small">{items.length} 个任务</Button>
        </Popover>
      </Space>
    </div>
  );
});

export default SlurmTaskBar;
