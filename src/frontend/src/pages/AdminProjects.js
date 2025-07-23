import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, Tag, Space, Popconfirm, message, Divider, Card, Row, Col } from 'antd';
import { ProjectOutlined, EditOutlined, DeleteOutlined, UserOutlined, EyeOutlined } from '@ant-design/icons';
import { adminAPI } from '../services/api';

const { Option } = Select;
const { TextArea } = Input;

const AdminProjects = () => {
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
      message.error('获取项目列表失败');
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
        message.success('项目更新成功');
      }
      setModalVisible(false);
      setEditingProject(null);
      form.resetFields();
      fetchProjects();
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleDelete = async (id) => {
    try {
      await adminAPI.deleteProject(id);
      message.success('项目删除成功');
      fetchProjects();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleViewUsers = async (project) => {
    setSelectedProject(project);
    try {
      const response = await adminAPI.getProjectUsers(project.id);
      setProjectUsers(response.data.users || []);
      setUserModalVisible(true);
    } catch (error) {
      message.error('获取项目用户失败');
    }
  };

  const handleRemoveUser = async (userId) => {
    try {
      await adminAPI.removeUserFromProject(selectedProject.id, userId);
      message.success('用户移除成功');
      // 重新获取项目用户列表
      const response = await adminAPI.getProjectUsers(selectedProject.id);
      setProjectUsers(response.data.users || []);
    } catch (error) {
      message.error('移除用户失败');
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
      case 'active': return '活跃';
      case 'inactive': return '非活跃';
      case 'archived': return '已归档';
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
      case 'owner': return '所有者';
      case 'admin': return '管理员';
      case 'member': return '成员';
      case 'viewer': return '查看者';
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
      title: '项目名称',
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
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      width: 200,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={getStatusColor(status)}>
          {getStatusText(status)}
        </Tag>
      ),
    },
    {
      title: '所有者',
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
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="middle">
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => handleViewUsers(record)}
          >
            查看用户
          </Button>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这个项目吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button
              type="link"
              danger
              icon={<DeleteOutlined />}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const userColumns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (role) => (
        <Tag color={getRoleColor(role)}>
          {getRoleText(role)}
        </Tag>
      ),
    },
    {
      title: '加入时间',
      dataIndex: 'joined_at',
      key: 'joined_at',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Popconfirm
          title="确定要移除这个用户吗？"
          onConfirm={() => handleRemoveUser(record.id)}
          okText="确定"
          cancelText="取消"
        >
          <Button type="link" danger size="small">
            移除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <div style={{ marginBottom: 16 }}>
          <h2>项目管理</h2>
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
            showTotal: (total) => `共 ${total} 项`,
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
        title={editingProject ? '编辑项目' : '添加项目'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={() => {
          setModalVisible(false);
          setEditingProject(null);
          form.resetFields();
        }}
        okText="确定"
        cancelText="取消"
      >
        <Form
          form={form}
          layout="vertical"
          name="projectForm"
        >
          <Form.Item
            name="name"
            label="项目名称"
            rules={[{ required: true, message: '请输入项目名称' }]}
          >
            <Input disabled={!!editingProject} placeholder="请输入项目名称" />
          </Form.Item>

          <Form.Item
            name="description"
            label="项目描述"
          >
            <TextArea disabled={!!editingProject} rows={4} placeholder="请输入项目描述" />
          </Form.Item>

          <Form.Item
            name="status"
            label="状态"
            rules={[{ required: true, message: '请选择状态' }]}
          >
            <Select placeholder="请选择状态">
              <Option value="active">活跃</Option>
              <Option value="inactive">非活跃</Option>
              <Option value="archived">已归档</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 查看项目用户模态框 */}
      <Modal
        title={`项目用户 - ${selectedProject?.name}`}
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
            关闭
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
