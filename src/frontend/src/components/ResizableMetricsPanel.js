/**
 * 可调整大小的性能指标面板组件
 * 支持拖拽调整高度，展示多个节点的性能指标
 */

import React, { useState, useCallback, useRef, useEffect } from 'react';
import { Card, Row, Col, Progress, Typography, Space, Tag, Tooltip, Spin, Empty, Select, Tabs } from 'antd';
import {
  DashboardOutlined,
  ThunderboltOutlined,
  CloudServerOutlined,
  WifiOutlined,
  DesktopOutlined,
  AreaChartOutlined,
  DragOutlined,
} from '@ant-design/icons';
import { useI18n } from '../hooks/useI18n';

const { Text, Title } = Typography;
const { Option } = Select;

/**
 * 单个性能指标卡片
 */
const MetricCard = ({ 
  title, 
  value, 
  maxValue = 100, 
  unit = '%', 
  icon, 
  color = '#1890ff',
  showProgress = true,
  status = 'normal',
  tooltip,
  loading = false,
}) => {
  const percent = maxValue > 0 ? Math.min((value / maxValue) * 100, 100) : 0;
  
  // 根据值确定状态颜色
  const getStatusColor = () => {
    if (status === 'warning' || percent >= 70) return '#faad14';
    if (status === 'error' || percent >= 90) return '#f5222d';
    return color;
  };

  const content = (
    <Card 
      size="small" 
      style={{ height: '100%' }}
      bodyStyle={{ padding: '12px' }}
    >
      <Space direction="vertical" style={{ width: '100%' }} size={4}>
        <Space>
          {icon && <span style={{ color: getStatusColor() }}>{icon}</span>}
          <Text type="secondary" style={{ fontSize: 12 }}>{title}</Text>
        </Space>
        {loading ? (
          <Spin size="small" />
        ) : (
          <>
            <Text strong style={{ fontSize: 20, color: getStatusColor() }}>
              {typeof value === 'number' ? value.toFixed(1) : value}
              <span style={{ fontSize: 12, fontWeight: 'normal' }}>{unit}</span>
            </Text>
            {showProgress && (
              <Progress 
                percent={percent} 
                size="small" 
                showInfo={false}
                strokeColor={getStatusColor()}
                trailColor="#f0f0f0"
              />
            )}
          </>
        )}
      </Space>
    </Card>
  );

  return tooltip ? <Tooltip title={tooltip}>{content}</Tooltip> : content;
};

/**
 * 节点性能指标面板
 */
const NodeMetricsPanel = ({ 
  nodeId, 
  nodeName, 
  metrics = {},
  loading = false,
  expanded = false,
  t,
}) => {
  return (
    <Card 
      title={
        <Space>
          <DesktopOutlined />
          <Text strong>{nodeName || nodeId}</Text>
          <Tag color={metrics.status === 'online' ? 'green' : 'red'}>
            {metrics.status === 'online' ? t('minions.status.online') : t('minions.status.unknown')}
          </Tag>
        </Space>
      }
      size="small"
      style={{ marginBottom: 8 }}
    >
      <Row gutter={[8, 8]}>
        <Col span={6}>
          <MetricCard
            title={t('metrics.cpuUsage')}
            value={metrics.cpu_usage || 0}
            icon={<DashboardOutlined />}
            color="#1890ff"
            loading={loading}
            tooltip={t('metrics.cpuUsageTooltip')}
          />
        </Col>
        <Col span={6}>
          <MetricCard
            title={t('metrics.memoryUsage')}
            value={metrics.memory_usage || 0}
            icon={<CloudServerOutlined />}
            color="#52c41a"
            loading={loading}
            tooltip={t('metrics.memoryUsageTooltip')}
          />
        </Col>
        <Col span={6}>
          <MetricCard
            title={t('metrics.activeConnections')}
            value={metrics.active_connections || 0}
            maxValue={1000}
            unit=""
            icon={<WifiOutlined />}
            color="#722ed1"
            showProgress={false}
            loading={loading}
            tooltip={t('metrics.activeConnectionsTooltip')}
          />
        </Col>
        <Col span={6}>
          <MetricCard
            title={t('metrics.networkBandwidth')}
            value={metrics.network_bandwidth || 0}
            unit=" Mbps"
            maxValue={10000}
            icon={<ThunderboltOutlined />}
            color="#fa8c16"
            showProgress={false}
            loading={loading}
            tooltip={t('metrics.networkBandwidthTooltip')}
          />
        </Col>
        
        {/* 扩展指标 - IB/ROCE/GPU 等 */}
        {expanded && (
          <>
            <Col span={6}>
              <MetricCard
                title={t('metrics.ibStatus')}
                value={metrics.ib_status || 'N/A'}
                unit=""
                icon={<AreaChartOutlined />}
                color="#13c2c2"
                showProgress={false}
                loading={loading}
                tooltip={t('metrics.ibStatusTooltip')}
              />
            </Col>
            <Col span={6}>
              <MetricCard
                title={t('metrics.roceNetwork')}
                value={metrics.roce_status || 'N/A'}
                unit=""
                icon={<WifiOutlined />}
                color="#eb2f96"
                showProgress={false}
                loading={loading}
                tooltip={t('metrics.roceNetworkTooltip')}
              />
            </Col>
            <Col span={6}>
              <MetricCard
                title={t('metrics.gpuUtilization')}
                value={metrics.gpu_utilization || 0}
                icon={<ThunderboltOutlined />}
                color="#f5222d"
                loading={loading}
                tooltip={t('metrics.gpuUtilizationTooltip')}
              />
            </Col>
            <Col span={6}>
              <MetricCard
                title={t('metrics.gpuMemory')}
                value={metrics.gpu_memory || 0}
                icon={<CloudServerOutlined />}
                color="#faad14"
                loading={loading}
                tooltip={t('metrics.gpuMemoryTooltip')}
              />
            </Col>
          </>
        )}
      </Row>
    </Card>
  );
};

/**
 * 可调整大小的性能指标主面板
 * 
 * @param {object} props
 * @param {array} props.nodes - 节点列表 [{ id, name, metrics }]
 * @param {boolean} props.loading - 是否加载中
 * @param {function} props.onRefresh - 刷新回调
 * @param {number} props.minHeight - 最小高度 (px)
 * @param {number} props.maxHeight - 最大高度 (px)
 * @param {number} props.defaultHeight - 默认高度 (px)
 * @param {string} props.title - 面板标题
 */
const ResizableMetricsPanel = ({
  nodes = [],
  loading = false,
  onRefresh,
  minHeight = 200,
  maxHeight = 800,
  defaultHeight = 400,
  title,
}) => {
  const { t } = useI18n();
  const [height, setHeight] = useState(defaultHeight);
  const [isResizing, setIsResizing] = useState(false);
  const [selectedNode, setSelectedNode] = useState('all');
  const [showExpanded, setShowExpanded] = useState(false);
  const panelRef = useRef(null);
  const startYRef = useRef(0);
  const startHeightRef = useRef(0);

  // 开始调整大小
  const handleResizeStart = useCallback((e) => {
    e.preventDefault();
    setIsResizing(true);
    startYRef.current = e.clientY;
    startHeightRef.current = height;
    document.body.style.cursor = 'ns-resize';
    document.body.style.userSelect = 'none';
  }, [height]);

  // 调整大小中
  const handleResizeMove = useCallback((e) => {
    if (!isResizing) return;
    
    const deltaY = e.clientY - startYRef.current;
    const newHeight = Math.min(maxHeight, Math.max(minHeight, startHeightRef.current + deltaY));
    setHeight(newHeight);
  }, [isResizing, minHeight, maxHeight]);

  // 结束调整大小
  const handleResizeEnd = useCallback(() => {
    setIsResizing(false);
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
  }, []);

  // 绑定全局事件
  useEffect(() => {
    if (isResizing) {
      document.addEventListener('mousemove', handleResizeMove);
      document.addEventListener('mouseup', handleResizeEnd);
    }
    return () => {
      document.removeEventListener('mousemove', handleResizeMove);
      document.removeEventListener('mouseup', handleResizeEnd);
    };
  }, [isResizing, handleResizeMove, handleResizeEnd]);

  // 筛选显示的节点
  const displayNodes = selectedNode === 'all' 
    ? nodes 
    : nodes.filter(n => n.id === selectedNode);

  return (
    <Card
      ref={panelRef}
      title={
        <Space>
          <DashboardOutlined />
          <span>{title || t('metrics.title')}</span>
          <Tag color="blue">{t('metrics.nodeCount', { count: nodes.length })}</Tag>
        </Space>
      }
      extra={
        <Space>
          <Select 
            value={selectedNode} 
            onChange={setSelectedNode}
            style={{ width: 180 }}
            size="small"
          >
            <Option value="all">{t('metrics.allNodes')}</Option>
            {nodes.map(node => (
              <Option key={node.id} value={node.id}>{node.name || node.id}</Option>
            ))}
          </Select>
          <Tooltip title={showExpanded ? t('metrics.collapseExtended') : t('metrics.expandExtended')}>
            <Tag 
              color={showExpanded ? 'blue' : 'default'}
              style={{ cursor: 'pointer' }}
              onClick={() => setShowExpanded(!showExpanded)}
            >
              <AreaChartOutlined /> {showExpanded ? t('metrics.collapse') : t('metrics.expand')}
            </Tag>
          </Tooltip>
        </Space>
      }
      bodyStyle={{ 
        height: height, 
        overflow: 'auto',
        padding: '12px',
        transition: isResizing ? 'none' : 'height 0.2s',
      }}
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: '40px 0' }}>
          <Spin size="large" />
          <div style={{ marginTop: 16 }}>
            <Text type="secondary">{t('metrics.loadingData')}</Text>
          </div>
        </div>
      ) : displayNodes.length === 0 ? (
        <Empty description={t('metrics.noNodeData')} />
      ) : (
        <Tabs
          type="card"
          size="small"
          items={displayNodes.map(node => ({
            key: node.id,
            label: (
              <Space size={4}>
                <DesktopOutlined />
                {node.name || node.id}
                <Tag 
                  color={node.metrics?.status === 'online' ? 'green' : 'red'}
                  style={{ marginLeft: 4, marginRight: 0 }}
                >
                  {node.metrics?.status === 'online' ? t('minions.status.online') : t('minions.status.offline')}
                </Tag>
              </Space>
            ),
            children: (
              <Row gutter={[12, 12]}>
                <Col span={6}>
                  <MetricCard
                    title={t('metrics.cpuUsage')}
                    value={node.metrics?.cpu_usage || 0}
                    icon={<DashboardOutlined />}
                    color="#1890ff"
                    loading={loading}
                  />
                </Col>
                <Col span={6}>
                  <MetricCard
                    title={t('metrics.memoryUsage')}
                    value={node.metrics?.memory_usage || 0}
                    icon={<CloudServerOutlined />}
                    color="#52c41a"
                    loading={loading}
                  />
                </Col>
                <Col span={6}>
                  <MetricCard
                    title={t('metrics.activeConnections')}
                    value={node.metrics?.active_connections || 0}
                    maxValue={1000}
                    unit=""
                    icon={<WifiOutlined />}
                    color="#722ed1"
                    showProgress={false}
                    loading={loading}
                  />
                </Col>
                <Col span={6}>
                  <MetricCard
                    title={t('metrics.networkBandwidth')}
                    value={node.metrics?.network_bandwidth || 0}
                    unit=" Mbps"
                    maxValue={10000}
                    icon={<ThunderboltOutlined />}
                    color="#fa8c16"
                    showProgress={false}
                    loading={loading}
                  />
                </Col>
                
                {showExpanded && (
                  <>
                    <Col span={6}>
                      <MetricCard
                        title={t('metrics.ibStatus')}
                        value={node.metrics?.ib_status || 'N/A'}
                        unit=""
                        icon={<AreaChartOutlined />}
                        color="#13c2c2"
                        showProgress={false}
                        tooltip={t('metrics.ibStatusTooltip')}
                      />
                    </Col>
                    <Col span={6}>
                      <MetricCard
                        title={t('metrics.roceNetwork')}
                        value={node.metrics?.roce_status || 'N/A'}
                        unit=""
                        icon={<WifiOutlined />}
                        color="#eb2f96"
                        showProgress={false}
                        tooltip={t('metrics.roceNetworkTooltip')}
                      />
                    </Col>
                    <Col span={6}>
                      <MetricCard
                        title={t('metrics.gpuUtilization')}
                        value={node.metrics?.gpu_utilization || 0}
                        icon={<ThunderboltOutlined />}
                        color="#f5222d"
                        tooltip={t('metrics.gpuUtilizationTooltip')}
                      />
                    </Col>
                    <Col span={6}>
                      <MetricCard
                        title={t('metrics.gpuMemory')}
                        value={node.metrics?.gpu_memory || 0}
                        icon={<CloudServerOutlined />}
                        color="#faad14"
                        tooltip={t('metrics.gpuMemoryTooltip')}
                      />
                    </Col>
                  </>
                )}
              </Row>
            ),
          }))}
        />
      )}
      
      {/* 可调整大小的拖动条 */}
      <div
        style={{
          position: 'absolute',
          bottom: 0,
          left: 0,
          right: 0,
          height: 8,
          cursor: 'ns-resize',
          background: isResizing ? '#e6f7ff' : 'transparent',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          transition: 'background 0.2s',
        }}
        onMouseDown={handleResizeStart}
        onMouseEnter={(e) => e.currentTarget.style.background = '#f0f0f0'}
        onMouseLeave={(e) => {
          if (!isResizing) e.currentTarget.style.background = 'transparent';
        }}
      >
        <DragOutlined style={{ color: '#999', fontSize: 10 }} />
      </div>
    </Card>
  );
};

export { MetricCard, NodeMetricsPanel };
export default ResizableMetricsPanel;
