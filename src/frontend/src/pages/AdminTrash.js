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
import dayjs from 'dayjs';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;
const { Option } = Select;

const AdminTrash = () => {
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
      message.error('加载回收站数据失败');
    } finally {
      setLoading(false);
    }
  };

  const handleRestore = async (projectId, projectName) => {
    try {
      await adminAPI.restoreProject(projectId);
      message.success(`项目 "${projectName}" 恢复成功`);
      loadTrashProjects();
    } catch (error) {
      message.error(error.response?.data?.message || '恢复项目失败');
    }
  };

  const handleForceDelete = async (projectId, projectName) => {
    try {
      await adminAPI.forceDeleteProject(projectId);
      message.success(`项目 "${projectName}" 已永久删除`);
      loadTrashProjects();
    } catch (error) {
      message.error(error.response?.data?.message || '删除项目失败');
    }
  };

  const handleClearTrash = () => {
    Modal.confirm({
      title: '清空回收站',
      icon: <ExclamationCircleOutlined />,
      content: (
        <div>
          <p>确定要清空整个回收站吗？</p>
          <p style={{ color: '#ff4d4f', marginBottom: 0 }}>
            ⚠️ 此操作将永久删除回收站中的所有项目，无法恢复！
          </p>
        </div>
      ),
      okText: '确定清空',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await adminAPI.clearTrash();
          message.success('回收站清空成功');
          loadTrashProjects();
        } catch (error) {
          message.error(error.response?.data?.message || '清空回收站失败');
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
      title: '项目名称',
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
      title: '创建者',
      dataIndex: 'creator_name',
      key: 'creator_name',
      width: 120,
      render: (text) => (
        <Space>
          <UserOutlined />
          <Text>{text || '未知'}</Text>
        </Space>
      )
    },
    {
      title: '删除者',
      dataIndex: 'deleted_by_name',
      key: 'deleted_by_name',
      width: 120,
      render: (text) => (
        <Space>
          <UserOutlined />
          <Text>{text || '未知'}</Text>
        </Space>
      )
    },
    {
      title: '删除时间',
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
      title: '文件数量',
      dataIndex: 'files_count',
      key: 'files_count',
      width: 100,
      render: (count) => (
        <Tag color="blue">{count || 0} 个文件</Tag>
      )
    },
    {
      title: '操作',
      key: 'actions',
      width: 200,
      render: (_, record) => (
        <Space>
          <Tooltip title="恢复项目">
            <Popconfirm
              title="恢复项目"
              description={`确定要恢复项目 "${record.name}" 吗？`}
              onConfirm={() => handleRestore(record.id, record.name)}
              okText="确定"
              cancelText="取消"
            >
              <Button
                type="primary"
                size="small"
                icon={<UndoOutlined />}
              >
                恢复
              </Button>
            </Popconfirm>
          </Tooltip>
          
          <Tooltip title="永久删除">
            <Popconfirm
              title="永久删除项目"
              description={
                <div>
                  <p>确定要永久删除项目 "{record.name}" 吗？</p>
                  <p style={{ color: '#ff4d4f', marginBottom: 0 }}>
                    ⚠️ 此操作无法恢复！
                  </p>
                </div>
              }
              onConfirm={() => handleForceDelete(record.id, record.name)}
              okText="确定删除"
              okType="danger"
              cancelText="取消"
            >
              <Button
                danger
                size="small"
                icon={<DeleteOutlined />}
              >
                删除
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
                回收站管理
              </Title>
              <Text type="secondary">
                管理已删除的项目，支持恢复或永久删除
              </Text>
            </div>
            <Button
              danger
              icon={<ClearOutlined />}
              onClick={handleClearTrash}
              disabled={projects.length === 0}
            >
              清空回收站
            </Button>
          </div>
        </div>

        {/* 筛选工具栏 */}
        <Card size="small" style={{ marginBottom: '16px' }}>
          <Space wrap>
            <Input.Search
              placeholder="搜索项目名称..."
              style={{ width: 200 }}
              onSearch={handleSearch}
              allowClear
            />
            
            <RangePicker
              placeholder={['开始日期', '结束日期']}
              onChange={handleDateRangeChange}
              style={{ width: 240 }}
            />
            
            <Select
              value={filters.sort}
              onChange={handleSortChange}
              style={{ width: 160 }}
            >
              <Option value="deleted_at_desc">删除时间 ↓</Option>
              <Option value="deleted_at_asc">删除时间 ↑</Option>
              <Option value="name_asc">项目名称 A-Z</Option>
              <Option value="name_desc">项目名称 Z-A</Option>
            </Select>
          </Space>
        </Card>

        {/* 统计信息 */}
        {pagination.total > 0 && (
          <div style={{ marginBottom: '16px' }}>
            <Space>
              <Tag icon={<ProjectOutlined />} color="orange">
                回收站中共有 {pagination.total} 个已删除项目
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
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`
          }}
          onChange={handleTableChange}
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="回收站为空"
              />
            )
          }}
        />
      </Card>
    </div>
  );
};

export default AdminTrash;
