/**
 * Minions 管理表格组件
 * 支持多选、反选、模糊搜索、批量操作
 */

import React, { useState, useMemo, useCallback } from 'react';
import { 
  Table, 
  Tag, 
  Space, 
  Button, 
  Tooltip, 
  Typography, 
  Popconfirm, 
  Dropdown, 
  Menu, 
  Modal,
  message,
  Badge,
  Row,
  Col,
} from 'antd';
import {
  DeleteOutlined,
  SettingOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  DesktopOutlined,
  MoreOutlined,
  SelectOutlined,
  CloseSquareOutlined,
  FilterOutlined,
  ExportOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import SearchInput from './SearchInput';
import { useSearch, highlightText } from '../hooks/useSearch';

const { Text } = Typography;

/**
 * 状态颜色映射
 */
const STATUS_COLORS = {
  up: 'success',
  online: 'success',
  running: 'success',
  down: 'error',
  offline: 'error',
  stopped: 'error',
  pending: 'processing',
  starting: 'processing',
  unknown: 'default',
};

/**
 * 可搜索字段定义
 */
const SEARCH_FIELDS = [
  'id',
  'name',
  'ip',
  'os',
  'os_version',
  'arch',
  'kernel_version',
  'salt_version',
  'gpu_driver_version',
];

/**
 * 表格列筛选器生成
 */
const generateFilters = (data, field) => {
  const values = [...new Set(data.map(item => item[field]).filter(Boolean))];
  return values.map(v => ({ text: v, value: v }));
};

/**
 * MinionsTable 组件
 * 
 * @param {object} props
 * @param {array} props.minions - Minion 数据数组
 * @param {boolean} props.loading - 是否加载中
 * @param {function} props.onRefresh - 刷新回调
 * @param {function} props.onDelete - 删除单个 Minion 回调
 * @param {function} props.onBatchDelete - 批量删除回调
 * @param {function} props.onUninstall - 卸载 Minion 回调
 * @param {function} props.onRemoteSearch - 远程搜索回调 (可选，用于全文索引)
 */
const MinionsTable = ({
  minions = [],
  loading = false,
  onRefresh,
  onDelete,
  onBatchDelete,
  onUninstall,
  onRemoteSearch,
}) => {
  // 选中的行
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  // 批量操作确认弹窗
  const [batchDeleteVisible, setBatchDeleteVisible] = useState(false);

  // 使用通用搜索 Hook
  const {
    searchText,
    setSearchText,
    results: filteredMinions,
    isSearching,
  } = useSearch({
    data: minions,
    searchFields: SEARCH_FIELDS,
    onRemoteSearch: onRemoteSearch,
    debounceMs: 300,
  });

  // 搜索统计
  const searchStats = useMemo(() => ({
    resultCount: filteredMinions.length,
    totalCount: minions.length,
  }), [filteredMinions.length, minions.length]);

  // 行选择配置
  const rowSelection = {
    selectedRowKeys,
    onChange: setSelectedRowKeys,
    selections: [
      Table.SELECTION_ALL,
      Table.SELECTION_INVERT,
      Table.SELECTION_NONE,
      {
        key: 'online',
        text: '选择在线节点',
        onSelect: (changableRowKeys) => {
          const onlineKeys = filteredMinions
            .filter(m => ['up', 'online', 'running'].includes(m.status?.toLowerCase()))
            .map(m => m.id || m.name);
          setSelectedRowKeys(onlineKeys);
        },
      },
      {
        key: 'offline',
        text: '选择离线节点',
        onSelect: (changableRowKeys) => {
          const offlineKeys = filteredMinions
            .filter(m => ['down', 'offline', 'stopped'].includes(m.status?.toLowerCase()))
            .map(m => m.id || m.name);
          setSelectedRowKeys(offlineKeys);
        },
      },
    ],
  };

  // 处理删除单个
  const handleDelete = useCallback((minionId) => {
    onDelete?.(minionId);
    setSelectedRowKeys(prev => prev.filter(k => k !== minionId));
  }, [onDelete]);

  // 处理批量删除
  const handleBatchDelete = useCallback(() => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的节点');
      return;
    }
    setBatchDeleteVisible(true);
  }, [selectedRowKeys]);

  // 确认批量删除
  const confirmBatchDelete = useCallback(async () => {
    try {
      await onBatchDelete?.(selectedRowKeys);
      setSelectedRowKeys([]);
      setBatchDeleteVisible(false);
      message.success(`已删除 ${selectedRowKeys.length} 个节点`);
    } catch (error) {
      message.error('批量删除失败: ' + error.message);
    }
  }, [selectedRowKeys, onBatchDelete]);

  // 导出选中数据
  const handleExport = useCallback(() => {
    const exportData = selectedRowKeys.length > 0
      ? filteredMinions.filter(m => selectedRowKeys.includes(m.id || m.name))
      : filteredMinions;
    
    const csv = [
      ['ID', '操作系统', '架构', 'Salt版本', '内核版本', 'GPU驱动', '状态', '最后响应'].join(','),
      ...exportData.map(m => [
        m.id || m.name,
        m.os || '',
        m.arch || '',
        m.salt_version || '',
        m.kernel_version || '',
        m.gpu_driver_version || '',
        m.status || '',
        m.last_seen || '',
      ].join(','))
    ].join('\n');
    
    const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `minions_export_${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
    message.success(`已导出 ${exportData.length} 条数据`);
  }, [filteredMinions, selectedRowKeys]);

  // 批量操作菜单
  const batchMenu = (
    <Menu>
      <Menu.Item 
        key="delete" 
        icon={<DeleteOutlined />} 
        danger
        onClick={handleBatchDelete}
        disabled={selectedRowKeys.length === 0}
      >
        批量删除 ({selectedRowKeys.length})
      </Menu.Item>
      <Menu.Divider />
      <Menu.Item 
        key="export" 
        icon={<ExportOutlined />}
        onClick={handleExport}
      >
        导出 {selectedRowKeys.length > 0 ? `选中 (${selectedRowKeys.length})` : '全部'}
      </Menu.Item>
      <Menu.Item 
        key="selectAll" 
        icon={<SelectOutlined />}
        onClick={() => setSelectedRowKeys(filteredMinions.map(m => m.id || m.name))}
      >
        全选当前页
      </Menu.Item>
      <Menu.Item 
        key="invertSelect" 
        icon={<CloseSquareOutlined />}
        onClick={() => {
          const allKeys = filteredMinions.map(m => m.id || m.name);
          const inverted = allKeys.filter(k => !selectedRowKeys.includes(k));
          setSelectedRowKeys(inverted);
        }}
      >
        反选
      </Menu.Item>
      <Menu.Item 
        key="clearSelect" 
        icon={<CloseSquareOutlined />}
        onClick={() => setSelectedRowKeys([])}
        disabled={selectedRowKeys.length === 0}
      >
        清空选择
      </Menu.Item>
    </Menu>
  );

  // 渲染带高亮的文本
  const renderHighlightedText = useCallback((text, record, field) => {
    if (!searchText || !record._searchMatch?.fieldMatches?.[field]) {
      return text || '-';
    }
    const { highlights } = record._searchMatch.fieldMatches[field];
    return highlightText(text, highlights);
  }, [searchText]);

  // 表格列定义
  const columns = [
    {
      title: 'ID / 主机名',
      dataIndex: 'id',
      key: 'id',
      width: 180,
      fixed: 'left',
      sorter: (a, b) => (a.id || a.name || '').localeCompare(b.id || b.name || ''),
      render: (id, record) => (
        <Space>
          <Badge status={STATUS_COLORS[record.status?.toLowerCase()] || 'default'} />
          <Text strong>{renderHighlightedText(id || record.name, record, 'id')}</Text>
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      filters: [
        { text: '在线', value: 'up' },
        { text: '在线', value: 'online' },
        { text: '离线', value: 'down' },
        { text: '离线', value: 'offline' },
      ],
      onFilter: (value, record) => record.status?.toLowerCase() === value,
      render: (status) => {
        const color = STATUS_COLORS[status?.toLowerCase()] || 'default';
        return (
          <Tag color={color} icon={color === 'success' ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}>
            {status === 'up' || status === 'online' ? '在线' : status === 'down' || status === 'offline' ? '离线' : status || '未知'}
          </Tag>
        );
      },
    },
    {
      title: '操作系统',
      dataIndex: 'os',
      key: 'os',
      width: 120,
      ellipsis: true,
      filters: generateFilters(minions, 'os'),
      onFilter: (value, record) => record.os === value,
      render: (os, record) => (
        <Tooltip title={record.os_version || os}>
          {renderHighlightedText(os, record, 'os')}
        </Tooltip>
      ),
    },
    {
      title: 'CPU架构',
      dataIndex: 'arch',
      key: 'arch',
      width: 100,
      filters: generateFilters(minions, 'arch'),
      onFilter: (value, record) => record.arch === value,
      render: (arch, record) => (
        <Tag>{renderHighlightedText(arch, record, 'arch') || '-'}</Tag>
      ),
    },
    {
      title: 'Salt版本',
      dataIndex: 'salt_version',
      key: 'salt_version',
      width: 120,
      sorter: (a, b) => (a.salt_version || '').localeCompare(b.salt_version || ''),
      render: (version, record) => (
        <Text code style={{ fontSize: 12 }}>
          {renderHighlightedText(version, record, 'salt_version') || '-'}
        </Text>
      ),
    },
    {
      title: '内核版本',
      dataIndex: 'kernel_version',
      key: 'kernel_version',
      width: 180,
      ellipsis: true,
      render: (version, record) => (
        <Tooltip title={version}>
          <Text style={{ fontSize: 12 }}>
            {renderHighlightedText(version, record, 'kernel_version') || '-'}
          </Text>
        </Tooltip>
      ),
    },
    {
      title: 'GPU驱动',
      dataIndex: 'gpu_driver_version',
      key: 'gpu_driver_version',
      width: 120,
      render: (version, record) => (
        version ? (
          <Tag color="purple">
            {renderHighlightedText(version, record, 'gpu_driver_version')}
          </Tag>
        ) : (
          <Text type="secondary">-</Text>
        )
      ),
    },
    {
      title: '最后响应',
      dataIndex: 'last_seen',
      key: 'last_seen',
      width: 160,
      sorter: (a, b) => new Date(a.last_seen || 0) - new Date(b.last_seen || 0),
      defaultSortOrder: 'descend',
      render: (time) => (
        time ? (
          <Tooltip title={time}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {new Date(time).toLocaleString('zh-CN')}
            </Text>
          </Tooltip>
        ) : (
          <Text type="secondary">-</Text>
        )
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="卸载 Minion">
            <Button
              type="text"
              size="small"
              icon={<SettingOutlined />}
              onClick={() => onUninstall?.(record.id || record.name)}
            />
          </Tooltip>
          <Popconfirm
            title="删除 Minion"
            description="确定要从 Salt Master 删除此 Minion 密钥吗？"
            onConfirm={() => handleDelete(record.id || record.name)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除密钥">
              <Button
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      {/* 工具栏 */}
      <Row gutter={16} style={{ marginBottom: 16 }} align="middle">
        <Col flex="auto">
          <SearchInput
            value={searchText}
            onChange={setSearchText}
            placeholder="搜索 ID、IP、操作系统、内核版本、驱动版本..."
            loading={isSearching}
            namespace="minions"
            searchFields={SEARCH_FIELDS}
            resultCount={searchStats.resultCount}
            totalCount={searchStats.totalCount}
            showStats={!!searchText}
            style={{ maxWidth: 500 }}
          />
        </Col>
        <Col>
          <Space>
            {selectedRowKeys.length > 0 && (
              <Tag color="blue">
                已选 {selectedRowKeys.length} 项
              </Tag>
            )}
            <Dropdown overlay={batchMenu} trigger={['click']}>
              <Button icon={<MoreOutlined />}>
                批量操作
              </Button>
            </Dropdown>
            <Button 
              icon={<ReloadOutlined spin={loading} />}
              onClick={onRefresh}
              loading={loading}
            >
              刷新
            </Button>
          </Space>
        </Col>
      </Row>

      {/* 表格 */}
      <Table
        columns={columns}
        dataSource={filteredMinions}
        rowKey={(record) => record.id || record.name}
        rowSelection={rowSelection}
        loading={loading}
        size="small"
        scroll={{ x: 1200 }}
        pagination={{
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`,
          defaultPageSize: 20,
          pageSizeOptions: ['10', '20', '50', '100'],
        }}
        locale={{
          emptyText: searchText ? '未找到匹配的节点' : '暂无 Minion 数据',
        }}
      />

      {/* 批量删除确认弹窗 */}
      <Modal
        title={
          <Space>
            <ExclamationCircleOutlined style={{ color: '#faad14' }} />
            确认批量删除
          </Space>
        }
        open={batchDeleteVisible}
        onOk={confirmBatchDelete}
        onCancel={() => setBatchDeleteVisible(false)}
        okText="确认删除"
        okButtonProps={{ danger: true }}
        cancelText="取消"
      >
        <p>确定要删除以下 <Text strong>{selectedRowKeys.length}</Text> 个 Minion 的密钥吗？</p>
        <div style={{ maxHeight: 200, overflow: 'auto', background: '#f5f5f5', padding: 12, borderRadius: 6 }}>
          {selectedRowKeys.map(key => (
            <Tag key={key} style={{ margin: 2 }}>{key}</Tag>
          ))}
        </div>
        <p style={{ marginTop: 12, color: '#999' }}>
          注意：此操作仅从 Salt Master 删除密钥，不会卸载目标机器上的 Salt Minion 软件。
        </p>
      </Modal>
    </div>
  );
};

export default MinionsTable;
