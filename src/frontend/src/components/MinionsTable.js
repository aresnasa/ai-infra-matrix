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
import { useI18n } from '../hooks/useI18n';

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
  const { t, locale } = useI18n();
  
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
        text: t('saltstack.selectOnlineNodes'),
        onSelect: (changableRowKeys) => {
          const onlineKeys = filteredMinions
            .filter(m => ['up', 'online', 'running'].includes(m.status?.toLowerCase()))
            .map(m => m.id || m.name);
          setSelectedRowKeys(onlineKeys);
        },
      },
      {
        key: 'offline',
        text: t('saltstack.selectOfflineNodes'),
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
      message.warning(t('saltstack.atLeastOneHost'));
      return;
    }
    setBatchDeleteVisible(true);
  }, [selectedRowKeys, t]);

  // 确认批量删除
  const confirmBatchDelete = useCallback(async () => {
    try {
      await onBatchDelete?.(selectedRowKeys);
      setSelectedRowKeys([]);
      setBatchDeleteVisible(false);
      message.success(t('saltstack.deletedNodes', { count: selectedRowKeys.length }));
    } catch (error) {
      message.error(t('common.failed') + ': ' + error.message);
    }
  }, [selectedRowKeys, onBatchDelete, t]);

  // 导出选中数据
  const handleExport = useCallback(() => {
    const exportData = selectedRowKeys.length > 0
      ? filteredMinions.filter(m => selectedRowKeys.includes(m.id || m.name))
      : filteredMinions;
    
    const csv = [
      ['ID', t('saltstack.os'), t('saltstack.arch'), t('saltstack.saltVersion'), t('saltstack.kernelVersion'), t('saltstack.gpuDriver'), t('common.status'), t('saltstack.lastSeen')].join(','),
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
    message.success(t('common.export') + `: ${exportData.length}`);
  }, [filteredMinions, selectedRowKeys, t]);

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
        {t('saltstack.batchDelete')} ({selectedRowKeys.length})
      </Menu.Item>
      <Menu.Divider />
      <Menu.Item 
        key="export" 
        icon={<ExportOutlined />}
        onClick={handleExport}
      >
        {selectedRowKeys.length > 0 ? t('saltstack.exportSelected') + ` (${selectedRowKeys.length})` : t('saltstack.exportAll')}
      </Menu.Item>
      <Menu.Item 
        key="selectAll" 
        icon={<SelectOutlined />}
        onClick={() => setSelectedRowKeys(filteredMinions.map(m => m.id || m.name))}
      >
        {t('common.selectAll')}
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
        {t('minions.batch.inverseSelect')}
      </Menu.Item>
      <Menu.Item 
        key="clearSelect" 
        icon={<CloseSquareOutlined />}
        onClick={() => setSelectedRowKeys([])}
        disabled={selectedRowKeys.length === 0}
      >
        {t('minions.batch.clearSelect')}
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
      title: t('minions.columns.id'),
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
      title: t('minions.columns.status'),
      dataIndex: 'status',
      key: 'status',
      width: 90,
      filters: [
        { text: t('minions.status.online'), value: 'up' },
        { text: t('minions.status.online'), value: 'online' },
        { text: t('minions.status.offline'), value: 'down' },
        { text: t('minions.status.offline'), value: 'offline' },
      ],
      onFilter: (value, record) => record.status?.toLowerCase() === value,
      render: (status) => {
        const color = STATUS_COLORS[status?.toLowerCase()] || 'default';
        const isOnline = status === 'up' || status === 'online';
        return (
          <Tag color={color} icon={color === 'success' ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}>
            {isOnline ? t('minions.status.online') : status === 'down' || status === 'offline' ? t('minions.status.offline') : status || t('minions.status.unknown')}
          </Tag>
        );
      },
    },
    {
      title: t('minions.columns.os'),
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
      title: t('minions.columns.arch'),
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
      title: t('minions.columns.saltVersion'),
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
      title: t('minions.columns.kernel'),
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
      title: t('minions.columns.gpuDriver'),
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
      title: t('minions.columns.lastSeen'),
      dataIndex: 'last_seen',
      key: 'last_seen',
      width: 160,
      sorter: (a, b) => new Date(a.last_seen || 0) - new Date(b.last_seen || 0),
      defaultSortOrder: 'descend',
      render: (time) => (
        time ? (
          <Tooltip title={time}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {new Date(time).toLocaleString(locale === 'zh-CN' ? 'zh-CN' : 'en-US')}
            </Text>
          </Tooltip>
        ) : (
          <Text type="secondary">-</Text>
        )
      ),
    },
    {
      title: t('minions.columns.actions'),
      key: 'actions',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Tooltip title={t('minions.actions.uninstall')}>
            <Button
              type="text"
              size="small"
              icon={<SettingOutlined />}
              onClick={() => onUninstall?.(record.id || record.name)}
            />
          </Tooltip>
          <Popconfirm
            title={t('minions.actions.deleteTitle')}
            description={t('minions.actions.deleteConfirm')}
            onConfirm={() => handleDelete(record.id || record.name)}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
          >
            <Tooltip title={t('minions.actions.deleteKey')}>
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
            placeholder={t('minions.search.placeholder')}
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
                {t('minions.selected', { count: selectedRowKeys.length })}
              </Tag>
            )}
            <Dropdown overlay={batchMenu} trigger={['click']}>
              <Button icon={<MoreOutlined />}>
                {t('minions.batch.title')}
              </Button>
            </Dropdown>
            <Button 
              icon={<ReloadOutlined spin={loading} />}
              onClick={onRefresh}
              loading={loading}
            >
              {t('common.refresh')}
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
          showTotal: (total) => t('common.total', { count: total }),
          defaultPageSize: 20,
          pageSizeOptions: ['10', '20', '50', '100'],
        }}
        locale={{
          emptyText: searchText ? t('minions.search.noResults') : t('minions.noData'),
        }}
      />

      {/* 批量删除确认弹窗 */}
      <Modal
        title={
          <Space>
            <ExclamationCircleOutlined style={{ color: '#faad14' }} />
            {t('minions.batch.deleteConfirmTitle')}
          </Space>
        }
        open={batchDeleteVisible}
        onOk={confirmBatchDelete}
        onCancel={() => setBatchDeleteVisible(false)}
        okText={t('minions.batch.confirmDelete')}
        okButtonProps={{ danger: true }}
        cancelText={t('common.cancel')}
      >
        <p>{t('minions.batch.deleteConfirmMessage', { count: selectedRowKeys.length })}</p>
        <div style={{ maxHeight: 200, overflow: 'auto', background: '#f5f5f5', padding: 12, borderRadius: 6 }}>
          {selectedRowKeys.map(key => (
            <Tag key={key} style={{ margin: 2 }}>{key}</Tag>
          ))}
        </div>
        <p style={{ marginTop: 12, color: '#999' }}>
          {t('minions.batch.deleteNote')}
        </p>
      </Modal>
    </div>
  );
};

export default MinionsTable;
