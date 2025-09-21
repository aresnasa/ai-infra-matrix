import React, { useEffect, useMemo, useState } from 'react';
import { Badge, Button, Empty, Popover, Space, Tag, Tooltip, Typography } from 'antd';
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
    startedAt: t.started_at ? dayjs.unix(t.started_at) : null,
    completedAt: t.completed_at ? dayjs.unix(t.completed_at) : null,
    latest: t.latest_event || null,
  };
};

export default function SlurmTaskBar({ refreshInterval = 10000, maxItems = 8, style }) {
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const load = async () => {
    setLoading(true);
    try {
      const res = await slurmAPI.getTasks();
      const data = res?.data?.data || [];
      setTasks(data.map(formatTask));
    } catch (e) {
      // console.debug('Failed to load slurm tasks', e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    if (refreshInterval > 0) {
      const t = setInterval(load, refreshInterval);
      return () => clearInterval(t);
    }
  }, [refreshInterval]);

  const items = useMemo(() => tasks.slice(0, maxItems), [tasks, maxItems]);

  const content = (
    <Space direction="vertical" size="small" style={{ maxWidth: 420 }}>
      {items.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无任务" />
      ) : (
        items.map((t) => (
          <Space key={t.id} style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Space>
              <Badge status={badgeStatus(t.status)} />
              <Text strong>{t.name}</Text>
              <Tag color="default">{t.status}</Tag>
            </Space>
            <Space>
              {t.startedAt && <Text type="secondary">{t.startedAt.fromNow()}</Text>}
            </Space>
          </Space>
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
}
