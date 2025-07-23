import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, Tag, Space, Popconfirm, message, Divider, Drawer } from 'antd';
import { UserOutlined, EditOutlined, DeleteOutlined, PlusOutlined, KeyOutlined, TeamOutlined } from '@ant-design/icons';
import { adminAPI } from '../services/api';

const { Option } = Select;

const AdminUsers = () => {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [resetPasswordModalVisible, setResetPasswordModalVisible] = useState(false);
  const [userGroupsDrawerVisible, setUserGroupsDrawerVisible] = useState(false);
  const [editingUser, setEditingUser] = useState(null);
  const [selectedUser, setSelectedUser] = useState(null);
  const [form] = Form.useForm();
  const [resetPasswordForm] = Form.useForm();
  const [userGroupsForm] = Form.useForm();
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });

  useEffect(() => {
    fetchUsers();
  }, [pagination.current, pagination.pageSize]);

  const fetchUsers = async () => {
    setLoading(true);
    try {
      const response = await adminAPI.getAllUsers({
        page: pagination.current,
        page_size: pagination.pageSize,
      });
      
      // 获取每个用户的详细信息，包括认证来源
      const usersWithAuthSource = await Promise.all(
        (response.data.users || []).map(async (user) => {
          try {
            const userDetail = await adminAPI.getUserWithAuthSource(user.id);
            return { ...user, ...userDetail.data };
          } catch (error) {
            console.warn(`Failed to get auth source for user ${user.id}:`, error);
            return { ...user, auth_source: 'unknown' };
          }
        })
      );
      
      setUsers(usersWithAuthSource);
      setPagination(prev => ({
        ...prev,
        total: response.data.total || 0,
      }));
    } catch (error) {
      message.error('获取用户列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleEdit = (user) => {
    setEditingUser(user);
    form.setFieldsValue({
      username: user.username,
      email: user.email,
      status: user.status,
      role: user.role,
    });
    setModalVisible(true);
  };

  const handleDelete = async (userId) => {
    try {
      await adminAPI.deleteUser(userId);
      message.success('删除用户成功');
      fetchUsers();
    } catch (error) {
      message.error('删除用户失败');
    }
  };

  const handleResetPassword = (user) => {
    setSelectedUser(user);
    resetPasswordForm.resetFields();
    setResetPasswordModalVisible(true);
  };

  const handleManageUserGroups = (user) => {
    setSelectedUser(user);
    userGroupsForm.setFieldsValue({
      group_ids: user.user_groups?.map(group => group.id) || []
    });
    setUserGroupsDrawerVisible(true);
  };

  const handleResetPasswordOk = async () => {
    try {
      const values = await resetPasswordForm.validateFields();
      await adminAPI.resetUserPassword(selectedUser.id, values.new_password);
      message.success('密码重置成功');
      setResetPasswordModalVisible(false);
      setSelectedUser(null);
      resetPasswordForm.resetFields();
    } catch (error) {
      message.error('密码重置失败');
    }
  };

  const handleUserGroupsOk = async () => {
    try {
      const values = await userGroupsForm.validateFields();
      await adminAPI.updateUserGroups(selectedUser.id, values.group_ids);
      message.success('用户组更新成功');
      setUserGroupsDrawerVisible(false);
      setSelectedUser(null);
      userGroupsForm.resetFields();
      fetchUsers();
    } catch (error) {
      message.error('用户组更新失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      if (editingUser) {
        // 使用增强版状态更新API，提供保护机制
        await adminAPI.updateUserStatusEnhanced(editingUser.id, { 
          status: values.status,
          reason: values.reason || '管理员操作'
        });
        message.success('更新用户状态成功');
      }
      setModalVisible(false);
      setEditingUser(null);
      form.resetFields();
      fetchUsers();
    } catch (error) {
      message.error(editingUser ? '更新用户失败' : '创建用户失败');
    }
  };

  const handleModalCancel = () => {
    setModalVisible(false);
    setEditingUser(null);
    form.resetFields();
  };

  const handleTableChange = (newPagination) => {
    setPagination(newPagination);
  };

  const getAuthSourceTag = (authSource) => {
    const authSourceMap = {
      'local': { color: 'blue', text: '本地认证' },
      'ldap': { color: 'green', text: 'LDAP认证' },
      'unknown': { color: 'gray', text: '未知' },
    };
    const config = authSourceMap[authSource] || { color: 'default', text: authSource };
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  const getStatusTag = (status) => {
    const statusMap = {
      'active': { color: 'green', text: '激活' },
      'inactive': { color: 'red', text: '停用' },
      'pending': { color: 'orange', text: '待激活' },
    };
    const config = statusMap[status] || { color: 'default', text: status };
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  const getRoleTag = (role) => {
    const roleMap = {
      'admin': { color: 'purple', text: '管理员' },
      'user': { color: 'blue', text: '普通用户' },
      'viewer': { color: 'cyan', text: '查看者' },
    };
    const config = roleMap[role] || { color: 'default', text: role };
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (text) => (
        <Space>
          <UserOutlined />
          {text}
        </Space>
      ),
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '认证来源',
      dataIndex: 'auth_source',
      key: 'auth_source',
      render: getAuthSourceTag,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: getStatusTag,
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: getRoleTag,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (text) => new Date(text).toLocaleString('zh-CN'),
    },
    {
      title: '最后登录',
      dataIndex: 'last_login',
      key: 'last_login',
      render: (text) => text ? new Date(text).toLocaleString('zh-CN') : '从未登录',
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space split={<Divider type="vertical" />}>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          {record.auth_source === 'local' && (
            <Button
              type="link"
              icon={<KeyOutlined />}
              onClick={() => handleResetPassword(record)}
            >
              重置密码
            </Button>
          )}
          <Button
            type="link"
            icon={<TeamOutlined />}
            onClick={() => handleManageUserGroups(record)}
          >
            用户组
          </Button>
          <Popconfirm
            title={
              record.auth_source === 'local' && record.role === 'admin' 
                ? "警告：这是本地管理员账户，删除后可能无法恢复管理权限。确定要删除吗？"
                : "确定要删除这个用户吗？"
            }
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button
              type="link"
              danger
              icon={<DeleteOutlined />}
              disabled={record.auth_source === 'local' && record.role === 'admin' && record.status === 'active'}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <h2>用户管理</h2>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setModalVisible(true)}
        >
          添加用户
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={users}
        loading={loading}
        pagination={pagination}
        onChange={handleTableChange}
        rowKey="id"
      />

      <Modal
        title={editingUser ? '编辑用户' : '添加用户'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={handleModalCancel}
        okText="确定"
        cancelText="取消"
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            status: 'active',
            role: 'user',
          }}
        >
          {!editingUser && (
            <>
              <Form.Item
                label="用户名"
                name="username"
                rules={[
                  { required: true, message: '请输入用户名' },
                  { min: 3, message: '用户名至少3个字符' },
                ]}
              >
                <Input placeholder="请输入用户名" />
              </Form.Item>

              <Form.Item
                label="邮箱"
                name="email"
                rules={[
                  { required: true, message: '请输入邮箱' },
                  { type: 'email', message: '请输入有效的邮箱地址' },
                ]}
              >
                <Input placeholder="请输入邮箱" />
              </Form.Item>

              <Form.Item
                label="密码"
                name="password"
                rules={[
                  { required: true, message: '请输入密码' },
                  { min: 6, message: '密码至少6个字符' },
                ]}
              >
                <Input.Password placeholder="请输入密码" />
              </Form.Item>

              <Form.Item
                label="角色"
                name="role"
                rules={[{ required: true, message: '请选择角色' }]}
              >
                <Select placeholder="请选择角色">
                  <Option value="user">普通用户</Option>
                  <Option value="admin">管理员</Option>
                  <Option value="viewer">查看者</Option>
                </Select>
              </Form.Item>
            </>
          )}

          <Form.Item
            label="状态"
            name="status"
            rules={[{ required: true, message: '请选择状态' }]}
          >
            <Select 
              placeholder="请选择状态"
              disabled={editingUser?.auth_source === 'local' && editingUser?.role === 'admin'}
            >
              <Option value="active">激活</Option>
              <Option value="inactive">停用</Option>
              <Option value="pending">待激活</Option>
            </Select>
          </Form.Item>
          
          {editingUser?.auth_source === 'local' && editingUser?.role === 'admin' && (
            <div style={{ 
              padding: '12px', 
              backgroundColor: '#fff7e6', 
              border: '1px solid #ffd591',
              borderRadius: '6px',
              marginTop: '16px'
            }}>
              <strong>注意：</strong>这是本地管理员账户，为了安全考虑，无法禁用此账户。
            </div>
          )}
          
          {editingUser && (
            <Form.Item
              label="操作原因"
              name="reason"
              rules={[{ required: false }]}
            >
              <Input.TextArea 
                placeholder="请输入操作原因（可选）" 
                rows={3}
              />
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* 密码重置模态框 */}
      <Modal
        title="重置用户密码"
        open={resetPasswordModalVisible}
        onOk={handleResetPasswordOk}
        onCancel={() => {
          setResetPasswordModalVisible(false);
          setSelectedUser(null);
          resetPasswordForm.resetFields();
        }}
        okText="确定"
        cancelText="取消"
      >
        <Form
          form={resetPasswordForm}
          layout="vertical"
        >
          <div style={{ marginBottom: 16 }}>
            <strong>用户：</strong>{selectedUser?.username}
          </div>
          <Form.Item
            label="新密码"
            name="new_password"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码至少6个字符' },
            ]}
          >
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>
          <Form.Item
            label="确认密码"
            name="confirm_password"
            dependencies={['new_password']}
            rules={[
              { required: true, message: '请确认密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password placeholder="请再次输入新密码" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 用户组管理抽屉 */}
      <Drawer
        title="管理用户组"
        placement="right"
        onClose={() => {
          setUserGroupsDrawerVisible(false);
          setSelectedUser(null);
          userGroupsForm.resetFields();
        }}
        open={userGroupsDrawerVisible}
        width={400}
        extra={
          <Button type="primary" onClick={handleUserGroupsOk}>
            保存
          </Button>
        }
      >
        <Form
          form={userGroupsForm}
          layout="vertical"
        >
          <div style={{ marginBottom: 16 }}>
            <strong>用户：</strong>{selectedUser?.username}
          </div>
          <Form.Item
            label="用户组"
            name="group_ids"
            rules={[{ required: false }]}
          >
            <Select
              mode="multiple"
              placeholder="请选择用户组"
              options={[
                { value: 1, label: '开发组' },
                { value: 2, label: '测试组' },
                { value: 3, label: '运维组' },
                { value: 4, label: '产品组' },
              ]}
            />
          </Form.Item>
          <div style={{ marginTop: 16, fontSize: '12px', color: '#666' }}>
            注意：用户组决定了用户可以访问哪些项目和资源。
          </div>
        </Form>
      </Drawer>
    </div>
  );
};

export default AdminUsers;
