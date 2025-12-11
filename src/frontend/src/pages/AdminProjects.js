import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, Tag, Space, Popconfirm, message, Divider, Card, Row, Col } from 'antd';
import { ProjectOutlined, EditOutlined, DeleteOutlined, UserOutlined, EyeOutlined } from '@ant-design/icons';
import { adminAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';

const { Option } = Select;
const { TextArea } = Input;

const AdminProjects = () => {
  const { t } = useI18n();
  const [projects, setProjects] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [userModalVisible, setUserModalVisible] = useState(false);
  const [editingProject, setEditingProject] = useState(null);
  const [selectedProject, setSelectedProject] = useState(null);
  const [projectUsers, setProjectUsers] = useState([]);
  const [form] = Form.useForm();
  const [userForm] = Form.useForm();
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  useEffect(() => {
    fetchProjects();
  }, [pagination.current, pagination.pageSize]);

  const fetchProjects = async () => {
    setLoading(true);
    try {
      const response = await adminAPI.getAllProjects({
        page: pagination.current,
        page_size: pagination.pageSize,
      });
      setProjects(response.data.projects || []);
      setPagination(prev => ({
        ...prev,
        total: response.data.total || 0,
      }));
    } catch (error) {
      message.error(t('admin.getProjectsFailed'));
    } finally {
      setLoading(false);
    }
  };

  const handleEdit = (project) => {
    setEditingProject(project);
    form.setFieldsValue({
      name: project.name,
      description: project.description,
      status: project.status,
    });
    setModalVisible(true);
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editingProject) {
        await adminAPI.updateProjectStatus(editingProject.id, values.status);
        message.success(t('admin.updateProjectSuccess'));
      }
      setModalVisible(false);
      setEditingProject(null);
      form.resetFields();
      fetchProjects();
    } catch (error) {
      message.error(t('admin.operationFailed'));
    }
  };

  const handleDelete = async (id) => {
    try {
      await adminAPI.deleteProject(id);
      message.success(t('admin.deleteProjectSuccess'));
      fetchProjects();
    } catch (error) {
      message.error(t('admin.deleteFailed'));
    }
  };

  const handleViewUsers = async (project) => {
    setSelectedProject(project);
    try {
      const response = await adminAPI.getProjectUsers(project.id);
      setProjectUsers(response.data.users || []);
      setUserModalVisible(true);
    } catch (error) {
      message.error(t('admin.getProjectUsersFailed'));
    }
  };

  const handleRemoveUser = async (userId) => {
    try {
      await adminAPI.removeUserFromProject(selectedProject.id, userId);
      message.success(t('admin.removeUserSuccess'));
      // 重新获取项目用户列表
      const response = await adminAPI.getProjectUsers(selectedProject.id);
      setProjectUsers(response.data.users || []);
    } catch (error) {
      message.error(t('admin.removeUserFailed'));
    }
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'active': return 'green';
      case 'inactive': return 'orange';
      case 'archived': return 'red';
      default: return 'default';
    }
  };

  const getStatusText = (status) => {
    switch (status) {
      case 'active': return t('admin.activeStatus');
      case 'inactive': return t('admin.inactiveStatus');
      case 'archived': return t('admin.archivedStatus');
      default: return status;
    }
  };

  const getRoleColor = (role) => {
    switch (role) {
      case 'owner': return 'purple';
      case 'admin': return 'red';
      case 'member': return 'blue';
      case 'viewer': return 'green';
      default: return 'default';
    }
  };

  const getRoleText = (role) => {
    switch (role) {
      case 'owner': return t('admin.ownerRole');
      case 'admin': return t('admin.adminRole');
      case 'member': return t('admin.memberRole');
      case 'viewer': return t('admin.viewerRole');
      default: return role;
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: t('admin.projectName'),
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Space>
          <ProjectOutlined />
          <span>{text}</span>
        </Space>
      ),
    },
    {
      title: t('admin.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      width: 200,
    },
    {
      title: t('admin.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={getStatusColor(status)}>
          {getStatusText(status)}
        </Tag>
      ),
    },
    {
      title: t('admin.owner'),
      dataIndex: 'owner_name',
      key: 'owner_name',
      render: (text) => (
        <Space>
          <UserOutlined />
          <span>{text}</span>
        </Space>
      ),
    },
    {
      title: t('admin.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: t('admin.updatedAt'),
      dataIndex: 'updated_at',
      key: 'updated_at',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: t('admin.action'),
      key: 'action',
      render: (_, record) => (
        <Space size="middle">
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => handleViewUsers(record)}
          >
            {t('admin.viewUsers')}
          </Button>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            {t('admin.edit')}
          </Button>
          <Popconfirm
            title={t('admin.confirmDelete')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('admin.confirm')}
            cancelText={t('admin.cancel')}
          >
            <Button
              type="link"
              danger
              icon={<DeleteOutlined />}
            >
              {t('admin.delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const userColumns = [
    {
      title: t('admin.username'),
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: t('admin.email'),
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: t('admin.role'),
      dataIndex: 'role',
      key: 'role',
      render: (role) => (
        <Tag color={getRoleColor(role)}>
          {getRoleText(role)}
        </Tag>
      ),
    },
    {
      title: t('admin.joinedAt'),
      dataIndex: 'joined_at',
      key: 'joined_at',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: t('admin.action'),
      key: 'action',
      render: (_, record) => (
        <Popconfirm
          title={t('admin.confirmRemoveUser')}
          onConfirm={() => handleRemoveUser(record.id)}
          okText={t('admin.confirm')}
          cancelText={t('admin.cancel')}
        >
          <Button type="link" danger size="small">
            {t('admin.remove')}
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <div style={{ marginBottom: 16 }}>
          <h2>{t('admin.projectManagement')}</h2>
        </div>
        
        <Table
          columns={columns}
          dataSource={projects}
          rowKey="id"
          loading={loading}
          pagination={{
            ...pagination,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => t('admin.total').replace('{count}', total),
            onChange: (page, pageSize) => {
              setPagination(prev => ({
                ...prev,
                current: page,
                pageSize: pageSize,
              }));
            },
          }}
        />
      </Card>

      {/* 编辑项目模态框 */}
      <Modal
        title={editingProject ? t('admin.editProject') : t('admin.addProject')}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => {
          setModalVisible(false);
          setEditingProject(null);
          form.resetFields();
        }}
        okText={t('admin.confirm')}
        cancelText={t('admin.cancel')}
      >
        <Form
          form={form}
          layout="vertical"
          name="projectForm"
        >
          <Form.Item
            name="name"
            label={t('admin.projectName')}
            rules={[{ required: true, message: t('admin.inputProjectName') }]}
          >
            <Input disabled={!!editingProject} placeholder={t('admin.inputProjectName')} />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('admin.description')}
          >
            <TextArea disabled={!!editingProject} rows={4} placeholder={t('admin.inputProjectDesc')} />
          </Form.Item>

          <Form.Item
            name="status"
            label={t('admin.status')}
            rules={[{ required: true, message: t('admin.selectStatus') }]}
          >
            <Select placeholder={t('admin.selectStatus')}>
              <Option value="active">{t('admin.activeStatus')}</Option>
              <Option value="inactive">{t('admin.inactiveStatus')}</Option>
              <Option value="archived">{t('admin.archivedStatus')}</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 查看项目用户模态框 */}
      <Modal
        title={`${t('admin.projectUsers')} - ${selectedProject?.name}`}
        open={userModalVisible}
        onCancel={() => {
          setUserModalVisible(false);
          setSelectedProject(null);
          setProjectUsers([]);
        }}
        footer={[
          <Button key="close" onClick={() => {
            setUserModalVisible(false);
            setSelectedProject(null);
            setProjectUsers([]);
          }}>
            {t('admin.close')}
          </Button>
        ]}
        width={800}
      >
        <Table
          columns={userColumns}
          dataSource={projectUsers}
          rowKey="id"
          pagination={{
            pageSize: 5,
            showSizeChanger: false,
          }}
        />
      </Modal>
    </div>
  );
};

export default AdminProjects;
