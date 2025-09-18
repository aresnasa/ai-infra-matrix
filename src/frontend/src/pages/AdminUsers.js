import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, Tag, Space, Popconfirm, message, Divider, Drawer, Tabs, Descriptions, Alert } from 'antd';
import { UserOutlined, EditOutlined, DeleteOutlined, PlusOutlined, KeyOutlined, TeamOutlined, CheckOutlined, CloseOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { adminAPI } from '../services/api';

const { Option } = Select;
const { TabPane } = Tabs;

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

  // 审批相关状态
  const [pendingApprovals, setPendingApprovals] = useState([]);
  const [approvalLoading, setApprovalLoading] = useState(false);
  const [approvalModalVisible, setApprovalModalVisible] = useState(false);
  const [selectedApproval, setSelectedApproval] = useState(null);
  const [rejectReason, setRejectReason] = useState('');

  useEffect(() => {
    fetchUsers();
    fetchPendingApprovals();
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

  const fetchPendingApprovals = async () => {
    try {
      const response = await adminAPI.getPendingApprovals();
      setPendingApprovals(response.data || []);
    } catch (error) {
      console.error('获取待审批申请失败:', error);
    }
  };

  const handleEdit = (user) => {
    setEditingUser(user);
    // 临时处理：将角色数组转换为单个角色用于编辑
    const primaryRole = user.roles && user.roles.length > 0 
      ? (user.roles[0]?.name || user.roles[0]) 
      : user.role; // 回退到旧的 role 字段
    
    form.setFieldsValue({
      username: user.username,
      email: user.email,
      status: user.status,
      role: primaryRole,
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

  // 审批相关方法
  const handleApprove = async (approval) => {
    setApprovalLoading(true);
    try {
      await adminAPI.approveRegistration(approval.id);
      message.success('注册申请已批准');
      fetchPendingApprovals();
      fetchUsers();
    } catch (error) {
      message.error('批准失败：' + (error.response?.data?.error || '未知错误'));
    } finally {
      setApprovalLoading(false);
    }
  };

  const handleReject = (approval) => {
    setSelectedApproval(approval);
    setRejectReason('');
    setApprovalModalVisible(true);
  };

  const handleRejectConfirm = async () => {
    if (!rejectReason.trim()) {
      message.error('请填写拒绝原因');
      return;
    }

    setApprovalLoading(true);
    try {
      await adminAPI.rejectRegistration(selectedApproval.id, rejectReason);
      message.success('注册申请已拒绝');
      setApprovalModalVisible(false);
      fetchPendingApprovals();
    } catch (error) {
      message.error('拒绝失败：' + (error.response?.data?.error || '未知错误'));
    } finally {
      setApprovalLoading(false);
    }
  };

  const getRoleTemplateName = (template) => {
    const templates = {
      'model-developer': '模型开发人员',
      'sre': 'SRE工程师',
      'engineer': '工程研发人员'
    };
    return templates[template] || template;
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

  const getRoleTag = (roles) => {
    if (!Array.isArray(roles) || roles.length === 0) {
      return <Tag color="default">无角色</Tag>;
    }
    
    const roleMap = {
      'admin': { color: 'purple', text: '管理员' },
      'user': { color: 'blue', text: '普通用户' },
      'viewer': { color: 'cyan', text: '查看者' },
    };
    
    return (
      <div>
        {(roles || []).map((roleObj, index) => {
          const roleName = roleObj?.name || roleObj;
          const config = roleMap[roleName] || { color: 'default', text: roleName };
          return <Tag key={index} color={config.color}>{config.text}</Tag>;
        })}
      </div>
    );
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
      dataIndex: 'roles',
      key: 'roles',
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
              record.auth_source === 'local' && record.roles?.some(role => (role?.name || role) === 'admin')
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
              disabled={record.auth_source === 'local' && record.roles?.some(role => (role?.name || role) === 'admin') && record.status === 'active'}
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
      <div style={{ marginBottom: 16 }}>
        <h2>用户管理</h2>
      </div>

      <Tabs defaultActiveKey="users" type="card">
        <TabPane tab="用户列表" key="users">
          <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
            <span>管理现有用户账户</span>
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
        </TabPane>

        <TabPane tab={`注册审批 (${pendingApprovals.length})`} key="approvals">
          <div style={{ marginBottom: 16 }}>
            <Alert
              message="待审批的注册申请"
              description="以下用户已提交注册申请，需要管理员审批后才能使用系统。"
              type="info"
              showIcon
            />
          </div>

          <Table
            dataSource={pendingApprovals}
            rowKey="id"
            loading={approvalLoading}
            columns={[
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
                title: '部门',
                dataIndex: 'department',
                key: 'department',
              },
              {
                title: '角色模板',
                dataIndex: 'role_template',
                key: 'role_template',
                render: (template) => (
                  <Tag color="blue">{getRoleTemplateName(template)}</Tag>
                ),
              },
              {
                title: '申请时间',
                dataIndex: 'created_at',
                key: 'created_at',
                render: (text) => new Date(text).toLocaleString('zh-CN'),
              },
              {
                title: '操作',
                key: 'action',
                render: (_, record) => (
                  <Space>
                    <Button
                      type="primary"
                      icon={<CheckOutlined />}
                      onClick={() => handleApprove(record)}
                      loading={approvalLoading}
                    >
                      批准
                    </Button>
                    <Button
                      danger
                      icon={<CloseOutlined />}
                      onClick={() => handleReject(record)}
                    >
                      拒绝
                    </Button>
                  </Space>
                ),
              },
            ]}
          />
        </TabPane>
      </Tabs>

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
              disabled={editingUser?.auth_source === 'local' && editingUser?.roles?.some(role => (role?.name || role) === 'admin')}
            >
              <Option value="active">激活</Option>
              <Option value="inactive">停用</Option>
              <Option value="pending">待激活</Option>
            </Select>
          </Form.Item>
          
          {editingUser?.auth_source === 'local' && editingUser?.roles?.some(role => (role?.name || role) === 'admin') && (
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

      {/* 审批拒绝模态框 */}
      <Modal
        title="拒绝注册申请"
        open={approvalModalVisible}
        onOk={handleRejectConfirm}
        onCancel={() => {
          setApprovalModalVisible(false);
          setSelectedApproval(null);
          setRejectReason('');
        }}
        okText="确认拒绝"
        cancelText="取消"
        okButtonProps={{ danger: true }}
        confirmLoading={approvalLoading}
      >
        <div style={{ marginBottom: 16 }}>
          <Descriptions title="申请信息" size="small" column={1}>
            <Descriptions.Item label="用户名">{selectedApproval?.username}</Descriptions.Item>
            <Descriptions.Item label="邮箱">{selectedApproval?.email}</Descriptions.Item>
            <Descriptions.Item label="部门">{selectedApproval?.department}</Descriptions.Item>
            <Descriptions.Item label="角色模板">{getRoleTemplateName(selectedApproval?.role_template)}</Descriptions.Item>
          </Descriptions>
        </div>
        <Form layout="vertical">
          <Form.Item
            label="拒绝原因"
            required
            rules={[{ required: true, message: '请填写拒绝原因' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="请说明拒绝该注册申请的原因..."
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default AdminUsers;
