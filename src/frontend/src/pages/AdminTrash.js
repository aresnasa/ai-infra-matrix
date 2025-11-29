import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Typography,
  Tag,
  Popconfirm,
  message,
  Modal,
  Tooltip,
  Input,
  Select,
  DatePicker,
  Empty
} from 'antd';
import {
  DeleteOutlined,
  UndoOutlined,
  ClearOutlined,
  SearchOutlined,
  ExclamationCircleOutlined,
  CalendarOutlined,
  UserOutlined,
  ProjectOutlined
} from '@ant-design/icons';
import { adminAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';
import dayjs from 'dayjs';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;
const { Option } = Select;

const AdminTrash = () => {
  const { t } = useI18n();
  const [loading, setLoading] = useState(false);
  const [projects, setProjects] = useState([]);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0
  });
  const [filters, setFilters] = useState({
    search: '',
    date_range: null,
    sort: 'deleted_at_desc'
  });

  useEffect(() => {
    loadTrashProjects();
  }, [pagination.current, pagination.pageSize, filters]);

  const loadTrashProjects = async () => {
    setLoading(true);
    try {
      const params = {
        page: pagination.current,
        limit: pagination.pageSize,
        search: filters.search || undefined,
        start_date: filters.date_range?.[0]?.format('YYYY-MM-DD') || undefined,
        end_date: filters.date_range?.[1]?.format('YYYY-MM-DD') || undefined,
        sort: filters.sort
      };

      const response = await adminAPI.getProjectsTrash(params);
      setProjects(response.data.projects || []);
      setPagination(prev => ({
        ...prev,
        total: response.data.total || 0
      }));
    } catch (error) {
      message.error(t('admin.loadTrashFailed'));
    } finally {
      setLoading(false);
    }
  };

  const handleRestore = async (projectId, projectName) => {
    try {
      await adminAPI.restoreProject(projectId);
      message.success(t('admin.restoreSuccess').replace('{name}', projectName));
      loadTrashProjects();
    } catch (error) {
      message.error(error.response?.data?.message || t('admin.restoreFailed'));
    }
  };

  const handleForceDelete = async (projectId, projectName) => {
    try {
      await adminAPI.forceDeleteProject(projectId);
      message.success(t('admin.permanentDeleteSuccess').replace('{name}', projectName));
      loadTrashProjects();
    } catch (error) {
      message.error(error.response?.data?.message || t('admin.permanentDeleteFailed'));
    }
  };

  const handleClearTrash = () => {
    Modal.confirm({
      title: t('admin.clearTrashConfirm'),
      icon: <ExclamationCircleOutlined />,
      content: (
        <div>
          <p>{t('admin.clearTrashConfirmDesc')}</p>
          <p style={{ color: '#ff4d4f', marginBottom: 0 }}>
            ⚠️ {t('admin.clearTrashWarning')}
          </p>
        </div>
      ),
      okText: t('admin.confirmClear'),
      okType: 'danger',
      cancelText: t('admin.cancel'),
      onOk: async () => {
        try {
          await adminAPI.clearTrash();
          message.success(t('admin.clearTrashSuccess'));
          loadTrashProjects();
        } catch (error) {
          message.error(error.response?.data?.message || t('admin.clearTrashFailed'));
        }
      }
    });
  };

  const handleSearch = (value) => {
    setFilters(prev => ({ ...prev, search: value }));
    setPagination(prev => ({ ...prev, current: 1 }));
  };

  const handleDateRangeChange = (dates) => {
    setFilters(prev => ({ ...prev, date_range: dates }));
    setPagination(prev => ({ ...prev, current: 1 }));
  };

  const handleSortChange = (value) => {
    setFilters(prev => ({ ...prev, sort: value }));
    setPagination(prev => ({ ...prev, current: 1 }));
  };

  const handleTableChange = (paginationConfig) => {
    setPagination(prev => ({
      ...prev,
      current: paginationConfig.current,
      pageSize: paginationConfig.pageSize
    }));
  };

  const columns = [
    {
      title: t('admin.projectName'),
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Space direction="vertical" size={0}>
          <Text strong>{text}</Text>
          {record.description && (
            <Text type="secondary" style={{ fontSize: '12px' }}>
              {record.description}
            </Text>
          )}
        </Space>
      )
    },
    {
      title: t('admin.creator'),
      dataIndex: 'creator_name',
      key: 'creator_name',
      width: 120,
      render: (text) => (
        <Space>
          <UserOutlined />
          <Text>{text || t('admin.unknown')}</Text>
        </Space>
      )
    },
    {
      title: t('admin.deletedBy'),
      dataIndex: 'deleted_by_name',
      key: 'deleted_by_name',
      width: 120,
      render: (text) => (
        <Space>
          <UserOutlined />
          <Text>{text || t('admin.unknown')}</Text>
        </Space>
      )
    },
    {
      title: t('admin.deletedAt'),
      dataIndex: 'deleted_at',
      key: 'deleted_at',
      width: 180,
      render: (text) => (
        <Space>
          <CalendarOutlined />
          <Text>{text ? dayjs(text).format('YYYY-MM-DD HH:mm:ss') : '-'}</Text>
        </Space>
      )
    },
    {
      title: t('admin.filesCount'),
      dataIndex: 'files_count',
      key: 'files_count',
      width: 100,
      render: (count) => (
        <Tag color="blue">{count || 0} {t('admin.files')}</Tag>
      )
    },
    {
      title: t('admin.action'),
      key: 'actions',
      width: 200,
      render: (_, record) => (
        <Space>
          <Tooltip title={t('admin.restoreProject')}>
            <Popconfirm
              title={t('admin.restoreProject')}
              description={t('admin.confirmRestore').replace('{name}', record.name)}
              onConfirm={() => handleRestore(record.id, record.name)}
              okText={t('admin.confirm')}
              cancelText={t('admin.cancel')}
            >
              <Button
                type="primary"
                size="small"
                icon={<UndoOutlined />}
              >
                {t('admin.restore')}
              </Button>
            </Popconfirm>
          </Tooltip>
          
          <Tooltip title={t('admin.permanentDelete')}>
            <Popconfirm
              title={t('admin.permanentDelete')}
              description={
                <div>
                  <p>{t('admin.confirmPermanentDelete').replace('{name}', record.name)}</p>
                  <p style={{ color: '#ff4d4f', marginBottom: 0 }}>
                    ⚠️ {t('admin.permanentDeleteWarning')}
                  </p>
                </div>
              }
              onConfirm={() => handleForceDelete(record.id, record.name)}
              okText={t('admin.confirm')}
              okType="danger"
              cancelText={t('admin.cancel')}
            >
              <Button
                danger
                size="small"
                icon={<DeleteOutlined />}
              >
                {t('admin.delete')}
              </Button>
            </Popconfirm>
          </Tooltip>
        </Space>
      )
    }
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <div style={{ marginBottom: '24px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <Title level={2}>
                <DeleteOutlined style={{ marginRight: '8px', color: '#ff4d4f' }} />
                {t('admin.trashManagement')}
              </Title>
              <Text type="secondary">
                {t('admin.trashManagementDesc')}
              </Text>
            </div>
            <Button
              danger
              icon={<ClearOutlined />}
              onClick={handleClearTrash}
              disabled={projects.length === 0}
            >
              {t('admin.clearTrash')}
            </Button>
          </div>
        </div>

        {/* 筛选工具栏 */}
        <Card size="small" style={{ marginBottom: '16px' }}>
          <Space wrap>
            <Input.Search
              placeholder={t('admin.searchProjectName')}
              style={{ width: 200 }}
              onSearch={handleSearch}
              allowClear
            />
            
            <RangePicker
              placeholder={[t('admin.startDate'), t('admin.endDate')]}
              onChange={handleDateRangeChange}
              style={{ width: 240 }}
            />
            
            <Select
              value={filters.sort}
              onChange={handleSortChange}
              style={{ width: 160 }}
            >
              <Option value="deleted_at_desc">{t('admin.sortDeletedAtDesc')}</Option>
              <Option value="deleted_at_asc">{t('admin.sortDeletedAtAsc')}</Option>
              <Option value="name_asc">{t('admin.sortNameAsc')}</Option>
              <Option value="name_desc">{t('admin.sortNameDesc')}</Option>
            </Select>
          </Space>
        </Card>

        {/* 统计信息 */}
        {pagination.total > 0 && (
          <div style={{ marginBottom: '16px' }}>
            <Space>
              <Tag icon={<ProjectOutlined />} color="orange">
                {t('admin.trashCount').replace('{count}', pagination.total)}
              </Tag>
            </Space>
          </div>
        )}

        {/* 项目列表 */}
        <Table
          columns={columns}
          dataSource={projects}
          rowKey="id"
          loading={loading}
          pagination={{
            ...pagination,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              t('admin.showing').replace('{start}', range[0]).replace('{end}', range[1]).replace('{total}', total)
          }}
          onChange={handleTableChange}
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={t('admin.trashEmpty')}
              />
            )
          }}
        />
      </Card>
    </div>
  );
};

export default AdminTrash;
