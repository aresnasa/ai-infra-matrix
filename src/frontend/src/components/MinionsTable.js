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
  Form,
  Input,
  InputNumber,
  Checkbox,
  Collapse,
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
  deleting: 'warning',      // 软删除状态 - 正在后台删除
  pending_delete: 'warning', // 待删除状态
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
 * @param {Set} props.deletingMinionIds - 正在删除中的 Minion ID 集合
 * @param {function} props.onRefresh - 刷新回调
 * @param {function} props.onDelete - 删除单个 Minion 回调
 * @param {function} props.onBatchDelete - 批量删除回调
 * @param {function} props.onUninstall - 卸载 Minion 回调
 * @param {function} props.onRemoteSearch - 远程搜索回调 (可选，用于全文索引)
 * @param {boolean} props.compact - 是否使用简洁模式 (只显示 ID 和状态)
 * @param {boolean} props.showActions - 是否显示操作列 (默认 true)
 */
const MinionsTable = ({
  minions = [],
  loading = false,
  deletingMinionIds = new Set(),
  onRefresh,
  onDelete,
  onBatchDelete,
  onUninstall,
  onRemoteSearch,
  compact = false,
  showActions = true,
}) => {
  const { t, locale } = useI18n();
  
  // 选中的行
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  // 批量操作确认弹窗
  const [batchDeleteVisible, setBatchDeleteVisible] = useState(false);
  // 强制删除选项
  const [forceDelete, setForceDelete] = useState(false);
  // SSH 卸载选项
  const [uninstallMode, setUninstallMode] = useState(false);
  const [sshConfig, setSshConfig] = useState({
    ssh_username: 'root',
    ssh_password: '',
    ssh_port: 22,
    use_sudo: false,
  });

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

  // 处理删除单个（支持强制删除）
  const handleDelete = useCallback((minionId, force = false) => {
    onDelete?.(minionId, force);
    setSelectedRowKeys(prev => prev.filter(k => k !== minionId));
  }, [onDelete]);

  // 处理批量删除
  const handleBatchDelete = useCallback(() => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('saltstack.atLeastOneHost'));
      return;
    }
    setForceDelete(false); // 重置强制删除选项
    setUninstallMode(false); // 重置卸载模式
    setSshConfig({
      ssh_username: 'root',
      ssh_password: '',
      ssh_port: 22,
      use_sudo: false,
    });
    setBatchDeleteVisible(true);
  }, [selectedRowKeys, t]);

  // 确认批量删除
  const confirmBatchDelete = useCallback(async () => {
    try {
      // 构建删除选项
      const options = {
        force: forceDelete,
        uninstall: uninstallMode,
        ...(uninstallMode ? sshConfig : {}),
      };
      await onBatchDelete?.(selectedRowKeys, options);
      setSelectedRowKeys([]);
      setBatchDeleteVisible(false);
      setForceDelete(false);
      setUninstallMode(false);
      message.success(t('saltstack.deletedNodes', { count: selectedRowKeys.length }));
    } catch (error) {
      message.error(t('common.failed') + ': ' + error.message);
    }
  }, [selectedRowKeys, onBatchDelete, forceDelete, uninstallMode, sshConfig, t]);

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

  // 简洁版 ID 显示：只保留最后一段（去掉长域名前缀）
  const getSimplifiedId = useCallback((id) => {
    if (!id) return '-';
    // 如果 ID 包含点号，只取最后一段（通常是主机名）
    const parts = id.split('.');
    return parts.length > 1 ? parts[0] : id;
  }, []);

  // 基础列定义（ID 和 Status）
  const baseColumns = [
    {
      title: t('minions.columns.id'),
      dataIndex: 'id',
      key: 'id',
      width: compact ? 150 : 180,
      fixed: compact ? undefined : 'left',
      sorter: (a, b) => (a.id || a.name || '').localeCompare(b.id || b.name || ''),
      render: (id, record) => {
        const displayId = compact ? getSimplifiedId(id || record.name) : (id || record.name);
        const fullId = id || record.name;
        return (
          <Space>
            <Badge status={STATUS_COLORS[record.status?.toLowerCase()] || 'default'} />
            {compact ? (
              <Tooltip title={fullId}>
                <Text strong>{renderHighlightedText(displayId, record, 'id')}</Text>
              </Tooltip>
            ) : (
              <Text strong>{renderHighlightedText(fullId, record, 'id')}</Text>
            )}
          </Space>
        );
      },
    },
    {
      title: t('minions.columns.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      filters: [
        { text: t('minions.status.online'), value: 'up' },
        { text: t('minions.status.online'), value: 'online' },
        { text: t('minions.status.offline'), value: 'down' },
        { text: t('minions.status.offline'), value: 'offline' },
        { text: t('minions.status.deleting') || '删除中', value: 'deleting' },
      ],
      onFilter: (value, record) => record.status?.toLowerCase() === value,
      render: (status, record) => {
        const minionId = record.id || record.name;
        const lowerStatus = status?.toLowerCase();
        const color = STATUS_COLORS[lowerStatus] || 'default';
        const isOnline = lowerStatus === 'up' || lowerStatus === 'online';
        // 优先使用 deletingMinionIds 判断是否正在删除（前端实时状态）
        const isDeleting = deletingMinionIds.has(minionId) || 
                          lowerStatus === 'deleting' || 
                          lowerStatus === 'pending_delete' || 
                          record.pending_delete;
        
        if (isDeleting) {
          return (
            <Tag color="warning" icon={<SyncOutlined spin />}>
              {t('minions.status.deleting') || '删除中'}
            </Tag>
          );
        }
        
        return (
          <Tag color={color} icon={color === 'success' ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}>
            {isOnline ? t('minions.status.online') : status === 'down' || status === 'offline' ? t('minions.status.offline') : status || t('minions.status.unknown')}
          </Tag>
        );
      },
    },
  ];

  // 扩展列定义（OS, Arch, Salt Version 等）
  const extendedColumns = [
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
  ];

  // 操作列
  const actionColumn = {
    title: t('minions.columns.actions'),
    key: 'actions',
    width: 140,
    fixed: compact ? undefined : 'right',
    render: (_, record) => {
      const minionId = record.id || record.name;
      const isOnline = ['up', 'online', 'running'].includes(record.status?.toLowerCase());
      // 检查是否正在删除中
      const isDeleting = deletingMinionIds.has(minionId) || 
                        record.status?.toLowerCase() === 'deleting' || 
                        record.pending_delete;
      
      // 如果正在删除中，显示禁用的按钮
      if (isDeleting) {
        return (
          <Space size="small">
            <Tooltip title={t('minions.status.deleting') || '删除中'}>
              <Button
                type="text"
                size="small"
                icon={<SyncOutlined spin />}
                disabled
              />
            </Tooltip>
          </Space>
        );
      }
      
      const deleteMenu = (
        <Menu>
          <Menu.Item 
            key="delete" 
            icon={<DeleteOutlined />}
            onClick={() => {
              Modal.confirm({
                title: t('minions.actions.confirmDelete', '确认删除'),
                content: t('minions.actions.confirmDeleteContent', { id: minionId }),
                okText: t('common.confirm', '确认'),
                cancelText: t('common.cancel', '取消'),
                okButtonProps: { danger: true },
                onOk: () => handleDelete(minionId, false),
              });
            }}
            disabled={isOnline}
          >
            {t('minions.actions.deleteKey', '删除密钥')}
            {isOnline && <Text type="secondary" style={{ marginLeft: 4, fontSize: 11 }}>({t('minions.actions.nodeOnline', '节点在线')})</Text>}
          </Menu.Item>
          <Menu.Item 
            key="forceDelete" 
            icon={<DeleteOutlined />}
            danger
            onClick={() => {
              Modal.confirm({
                title: t('minions.actions.confirmForceDelete', '确认强制删除'),
                content: t('minions.actions.confirmForceDeleteContent', { id: minionId }),
                okText: t('common.confirm', '确认'),
                cancelText: t('common.cancel', '取消'),
                okButtonProps: { danger: true },
                onOk: () => handleDelete(minionId, true),
              });
            }}
          >
            {t('minions.actions.forceDelete', '强制删除')}
          </Menu.Item>
        </Menu>
      );
      
      return (
        <Space size="small">
          <Tooltip title={t('minions.actions.uninstall')}>
            <Button
              type="text"
              size="small"
              icon={<SettingOutlined />}
              onClick={() => onUninstall?.(record.id || record.name)}
            />
          </Tooltip>
          <Dropdown overlay={deleteMenu} trigger={['click']}>
            <Tooltip title={t('minions.actions.deleteKey')}>
              <Button
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Dropdown>
        </Space>
      );
    },
  };

  // 根据 compact 模式和 showActions 决定最终列
  const columns = useMemo(() => {
    if (compact) {
      // 简洁模式: 只显示 ID 和 Status
      return showActions ? [...baseColumns, actionColumn] : baseColumns;
    }
    // 完整模式: 显示所有列
    return showActions ? [...baseColumns, ...extendedColumns, actionColumn] : [...baseColumns, ...extendedColumns];
  }, [compact, showActions, baseColumns, extendedColumns, actionColumn]);

  return (
    <div>
      {/* 工具栏 - compact 模式下简化显示 */}
      {!compact && (
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
      )}

      {/* 表格 */}
      <Table
        columns={columns}
        dataSource={filteredMinions}
        rowKey={(record) => record.id || record.name}
        rowSelection={compact ? undefined : rowSelection}
        loading={loading}
        size="small"
        scroll={compact ? undefined : { x: 1200 }}
        pagination={{
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => t('common.total', { count: total }),
          defaultPageSize: compact ? 10 : 20,
          pageSizeOptions: compact ? ['5', '10', '20'] : ['10', '20', '50', '100'],
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
        onCancel={() => {
          setBatchDeleteVisible(false);
          setForceDelete(false);
          setUninstallMode(false);
        }}
        okText={t('minions.batch.confirmDelete')}
        okButtonProps={{ danger: true }}
        cancelText={t('common.cancel')}
        width={600}
      >
        <p>{t('minions.batch.deleteConfirmMessage', { count: selectedRowKeys.length })}</p>
        <div style={{ maxHeight: 150, overflow: 'auto', background: '#f5f5f5', padding: 12, borderRadius: 6, marginBottom: 16 }}>
          {selectedRowKeys.map(key => (
            <Tag key={key} style={{ margin: 2 }}>{key}</Tag>
          ))}
        </div>
        
        {/* 删除选项 */}
        <div style={{ marginBottom: 16 }}>
          <Checkbox
            checked={forceDelete}
            onChange={(e) => setForceDelete(e.target.checked)}
          >
            <span style={{ color: forceDelete ? '#ff4d4f' : '#666' }}>
              {t('minions.batch.forceDelete', '强制删除（包括在线节点）')}
            </span>
          </Checkbox>
        </div>
        
        {/* SSH 卸载选项 */}
        <div style={{ marginBottom: 16 }}>
          <Checkbox
            checked={uninstallMode}
            onChange={(e) => setUninstallMode(e.target.checked)}
          >
            <span style={{ color: uninstallMode ? '#1890ff' : '#666' }}>
              {t('minions.batch.uninstallMode', '同时卸载 salt-minion 组件（需要 SSH 凭证）')}
            </span>
          </Checkbox>
        </div>
        
        {/* SSH 配置表单 */}
        {uninstallMode && (
          <Collapse defaultActiveKey={['ssh']} style={{ marginBottom: 16 }}>
            <Collapse.Panel header={t('minions.batch.sshConfig', 'SSH 连接配置')} key="ssh">
              <Form layout="vertical" size="small">
                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item label={t('saltstack.batchInstall.username', '用户名')}>
                      <Input
                        value={sshConfig.ssh_username}
                        onChange={(e) => setSshConfig(prev => ({ ...prev, ssh_username: e.target.value }))}
                        placeholder="root"
                      />
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item label={t('saltstack.batchInstall.port', '端口')}>
                      <InputNumber
                        value={sshConfig.ssh_port}
                        onChange={(val) => setSshConfig(prev => ({ ...prev, ssh_port: val || 22 }))}
                        min={1}
                        max={65535}
                        style={{ width: '100%' }}
                      />
                    </Form.Item>
                  </Col>
                </Row>
                <Form.Item label={t('saltstack.batchInstall.password', '密码')}>
                  <Input.Password
                    value={sshConfig.ssh_password}
                    onChange={(e) => setSshConfig(prev => ({ ...prev, ssh_password: e.target.value }))}
                    placeholder={t('saltstack.batchInstall.passwordPlaceholder', '输入 SSH 密码')}
                  />
                </Form.Item>
                <Form.Item>
                  <Checkbox
                    checked={sshConfig.use_sudo}
                    onChange={(e) => setSshConfig(prev => ({ ...prev, use_sudo: e.target.checked }))}
                  >
                    {t('saltstack.batchInstall.useSudo', '使用 sudo 执行卸载命令')}
                  </Checkbox>
                </Form.Item>
              </Form>
            </Collapse.Panel>
          </Collapse>
        )}
        
        <p style={{ color: '#999', fontSize: 12 }}>
          {t('minions.batch.deleteNote')}
          {uninstallMode && (
            <span style={{ display: 'block', marginTop: 4, color: '#faad14' }}>
              {t('minions.batch.uninstallNote', '注意：卸载模式将通过 SSH 连接到目标节点并卸载 salt-minion 软件包。')}
            </span>
          )}
        </p>
      </Modal>
    </div>
  );
};

export default MinionsTable;
