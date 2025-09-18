import React, { useState, useEffect } from 'react';
import { 
  Card, 
  Table, 
  Button, 
  Modal, 
  Form, 
  Input, 
  Select, 
  message, 
  Space, 
  Tag, 
  Switch,
  Divider,
  Tooltip,
  Progress,
  List,
  Typography
} from 'antd';
import { 
  PlusOutlined, 
  EditOutlined, 
  DeleteOutlined,
  SyncOutlined,
  UserOutlined,
  TeamOutlined,
  LockOutlined,
  UnlockOutlined,
  ReloadOutlined,
  DownloadOutlined,
  UploadOutlined
} from '@ant-design/icons';
import { userAPI, ldapAPI } from '../services/api';

const { Option } = Select;
const { Title, Text } = Typography;
const { TextArea } = Input;

const EnhancedUserManagement = () => {
  const [users, setUsers] = useState([]);
  const [userGroups, setUserGroups] = useState([]);
  const [roles, setRoles] = useState([]);
  const [loading, setLoading] = useState(false);
  const [syncLoading, setSyncLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [groupModalVisible, setGroupModalVisible] = useState(false);
  const [syncModalVisible, setSyncModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState(null);
  const [editingGroup, setEditingGroup] = useState(null);
  const [form] = Form.useForm();
  const [groupForm] = Form.useForm();
  const [syncResults, setSyncResults] = useState(null);
  const [ldapConfig, setLdapConfig] = useState(null);

  // 获取用户列表
  const fetchUsers = async () => {
    setLoading(true);
    try {
      const response = await userAPI.getUsers();
      setUsers(response.data || []);
    } catch (error) {
      message.error('获取用户列表失败');
      console.error('Error fetching users:', error);
    } finally {
      setLoading(false);
    }
  };

  // 获取用户组列表
  const fetchUserGroups = async () => {
    try {
      const response = await userAPI.getUserGroups();
      setUserGroups(response.data || []);
    } catch (error) {
      console.error('Error fetching user groups:', error);
    }
  };

  // 获取角色列表
  const fetchRoles = async () => {
    try {
      const response = await userAPI.getRoles();
      setRoles(response.data || []);
    } catch (error) {
      console.error('Error fetching roles:', error);
    }
  };

  // 获取LDAP配置
  const fetchLdapConfig = async () => {
    try {
      const response = await ldapAPI.getConfig();
      setLdapConfig(response.data);
    } catch (error) {
      console.error('Error fetching LDAP config:', error);
    }
  };

  useEffect(() => {
    fetchUsers();
    fetchUserGroups();
    fetchRoles();
    fetchLdapConfig();
  }, []);

  // LDAP同步
  const handleLdapSync = async (options = {}) => {
    setSyncLoading(true);
    try {
      const response = await ldapAPI.syncUsers(options);
      setSyncResults(response.data);
      setSyncModalVisible(true);
      message.success('LDAP同步完成');
      fetchUsers(); // 刷新用户列表
    } catch (error) {
      message.error('LDAP同步失败: ' + (error.response?.data?.message || error.message));
      console.error('LDAP sync error:', error);
    } finally {
      setSyncLoading(false);
    }
  };

  // 测试LDAP连接
  const testLdapConnection = async () => {
    try {
      const response = await ldapAPI.testConnection();
      if (response.data.success) {
        message.success('LDAP连接测试成功');
      } else {
        message.error('LDAP连接测试失败: ' + response.data.message);
      }
    } catch (error) {
      message.error('LDAP连接测试失败');
      console.error('LDAP test error:', error);
    }
  };

  // 打开用户模态框
  const openUserModal = (user = null) => {
    setEditingUser(user);
    if (user) {
      form.setFieldsValue({
        ...user,
        role_ids: user.roles?.map(r => r.id) || [],
        user_group_ids: user.user_groups?.map(g => g.id) || []
      });
    } else {
      form.resetFields();
    }
    setModalVisible(true);
  };

  // 保存用户
  const handleSaveUser = async (values) => {
    try {
      if (editingUser) {
        await userAPI.updateUser(editingUser.id, values);
        message.success('用户更新成功');
      } else {
        await userAPI.createUser(values);
        message.success('用户创建成功');
      }
      setModalVisible(false);
      fetchUsers();
    } catch (error) {
      message.error(editingUser ? '用户更新失败' : '用户创建失败');
      console.error('Error saving user:', error);
    }
  };

  // 删除用户
  const handleDeleteUser = async (userId) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这个用户吗？此操作不可恢复。',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await userAPI.deleteUser(userId);
          message.success('用户删除成功');
          fetchUsers();
        } catch (error) {
          message.error('用户删除失败');
          console.error('Error deleting user:', error);
        }
      }
    });
  };

  // 重置用户密码
  const handleResetPassword = async (userId) => {
    Modal.confirm({
      title: '重置密码',
      content: '确定要重置这个用户的密码吗？新密码将通过邮件发送给用户。',
      okText: '重置',
      cancelText: '取消',
      onOk: async () => {
        try {
          const response = await userAPI.resetPassword(userId);
          message.success('密码重置成功，新密码：' + response.data.password);
        } catch (error) {
          message.error('密码重置失败');
          console.error('Error resetting password:', error);
        }
      }
    });
  };

  // 切换用户状态
  const toggleUserStatus = async (userId, currentStatus) => {
    try {
      await userAPI.updateUser(userId, { is_active: !currentStatus });
      message.success(currentStatus ? '用户已禁用' : '用户已启用');
      fetchUsers();
    } catch (error) {
      message.error('状态更新失败');
      console.error('Error updating user status:', error);
    }
  };

  // 打开用户组模态框
  const openGroupModal = (group = null) => {
    setEditingGroup(group);
    if (group) {
      groupForm.setFieldsValue(group);
    } else {
      groupForm.resetFields();
    }
    setGroupModalVisible(true);
  };

  // 保存用户组
  const handleSaveGroup = async (values) => {
    try {
      if (editingGroup) {
        await userAPI.updateUserGroup(editingGroup.id, values);
        message.success('用户组更新成功');
      } else {
        await userAPI.createUserGroup(values);
        message.success('用户组创建成功');
      }
      setGroupModalVisible(false);
      fetchUserGroups();
    } catch (error) {
      message.error(editingGroup ? '用户组更新失败' : '用户组创建失败');
      console.error('Error saving group:', error);
    }
  };

  // 用户表格列
  const userColumns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (text, record) => (
        <Space>
          <UserOutlined />
          <span>{text}</span>
          {record.auth_source === 'ldap' && <Tag color="blue">LDAP</Tag>}
        </Space>
      )
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email'
    },
    {
      title: '角色',
      dataIndex: 'roles',
      key: 'roles',
      render: (roles) => (
        <>
          {roles?.map(role => (
            <Tag key={role.id} color="green">{role.name}</Tag>
          ))}
        </>
      )
    },
    {
      title: '用户组',
      dataIndex: 'user_groups',
      key: 'user_groups',
      render: (groups) => (
        <>
          {groups?.map(group => (
            <Tag key={group.id} color="purple">{group.name}</Tag>
          ))}
        </>
      )
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (isActive, record) => (
        <Switch
          checked={isActive}
          onChange={() => toggleUserStatus(record.id, isActive)}
          checkedChildren="启用"
          unCheckedChildren="禁用"
        />
      )
    },
    {
      title: '最后登录',
      dataIndex: 'last_login',
      key: 'last_login',
      render: (text) => text ? new Date(text).toLocaleString() : '从未登录'
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Tooltip title="编辑">
            <Button 
              type="text" 
              icon={<EditOutlined />} 
              onClick={() => openUserModal(record)}
            />
          </Tooltip>
          <Tooltip title="重置密码">
            <Button 
              type="text" 
              icon={<LockOutlined />} 
              onClick={() => handleResetPassword(record.id)}
            />
          </Tooltip>
          <Tooltip title="删除">
            <Button 
              type="text" 
              danger 
              icon={<DeleteOutlined />} 
              onClick={() => handleDeleteUser(record.id)}
            />
          </Tooltip>
        </Space>
      )
    }
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>用户管理</Title>
      
      {/* LDAP同步区域 */}
      <Card 
        title="LDAP同步" 
        style={{ marginBottom: '24px' }}
        extra={
          <Space>
            <Button 
              icon={<ReloadOutlined />} 
              onClick={testLdapConnection}
            >
              测试连接
            </Button>
            <Button 
              type="primary" 
              icon={<SyncOutlined />} 
              loading={syncLoading}
              onClick={() => handleLdapSync()}
            >
              立即同步
            </Button>
          </Space>
        }
      >
        {ldapConfig ? (
          <div>
            <Text>LDAP服务器: {ldapConfig.server}</Text><br />
            <Text>基础DN: {ldapConfig.base_dn}</Text><br />
            <Text>用户过滤器: {ldapConfig.user_filter}</Text><br />
            <Text>上次同步: {ldapConfig.last_sync ? new Date(ldapConfig.last_sync).toLocaleString() : '从未同步'}</Text>
          </div>
        ) : (
          <Text type="secondary">LDAP配置未设置</Text>
        )}
      </Card>

      {/* 用户组管理 */}
      <Card 
        title="用户组管理" 
        style={{ marginBottom: '24px' }}
        extra={
          <Button 
            type="primary" 
            icon={<PlusOutlined />} 
            onClick={() => openGroupModal()}
          >
            添加用户组
          </Button>
        }
      >
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
          {userGroups?.map(group => (
            <Tag 
              key={group.id} 
              color="purple" 
              style={{ cursor: 'pointer', padding: '4px 8px' }}
              onClick={() => openGroupModal(group)}
            >
              <TeamOutlined /> {group.name} ({group.users?.length || 0})
            </Tag>
          )) || []}
        </div>
      </Card>

      {/* 用户列表 */}
      <Card 
        title="用户列表" 
        extra={
          <Space>
            <Button 
              icon={<ReloadOutlined />} 
              onClick={fetchUsers}
            >
              刷新
            </Button>
            <Button 
              type="primary" 
              icon={<PlusOutlined />} 
              onClick={() => openUserModal()}
            >
              添加用户
            </Button>
          </Space>
        }
      >
        <Table
          columns={userColumns}
          dataSource={users}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      {/* 用户编辑模态框 */}
      <Modal
        title={editingUser ? '编辑用户' : '添加用户'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
        okText="保存"
        cancelText="取消"
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSaveUser}
        >
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="用户名" />
          </Form.Item>

          <Form.Item
            name="email"
            label="邮箱"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '邮箱格式不正确' }
            ]}
          >
            <Input placeholder="邮箱" />
          </Form.Item>

          {!editingUser && (
            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true, message: '请输入密码' }]}
            >
              <Input.Password placeholder="密码" />
            </Form.Item>
          )}

          <Form.Item
            name="auth_source"
            label="认证来源"
            initialValue="local"
          >
            <Select>
              <Option value="local">本地</Option>
              <Option value="ldap">LDAP</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="role_ids"
            label="角色"
          >
            <Select mode="multiple" placeholder="选择角色">
              {roles?.map(role => (
                <Option key={role.id} value={role.id}>{role.name}</Option>
              )) || []}
            </Select>
          </Form.Item>

          <Form.Item
            name="user_group_ids"
            label="用户组"
          >
            <Select mode="multiple" placeholder="选择用户组">
              {userGroups?.map(group => (
                <Option key={group.id} value={group.id}>{group.name}</Option>
              )) || []}
            </Select>
          </Form.Item>

          <Form.Item
            name="is_active"
            label="账户状态"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 用户组编辑模态框 */}
      <Modal
        title={editingGroup ? '编辑用户组' : '添加用户组'}
        open={groupModalVisible}
        onCancel={() => setGroupModalVisible(false)}
        onOk={() => groupForm.submit()}
        okText="保存"
        cancelText="取消"
      >
        <Form
          form={groupForm}
          layout="vertical"
          onFinish={handleSaveGroup}
        >
          <Form.Item
            name="name"
            label="用户组名称"
            rules={[{ required: true, message: '请输入用户组名称' }]}
          >
            <Input placeholder="用户组名称" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <TextArea rows={3} placeholder="用户组描述" />
          </Form.Item>
        </Form>
      </Modal>

      {/* LDAP同步结果模态框 */}
      <Modal
        title="LDAP同步结果"
        open={syncModalVisible}
        onCancel={() => setSyncModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setSyncModalVisible(false)}>
            关闭
          </Button>
        ]}
        width={600}
      >
        {syncResults && (
          <div>
            <div style={{ marginBottom: '16px' }}>
              <Text strong>同步统计:</Text>
              <div style={{ marginTop: '8px' }}>
                <Tag color="green">新增: {syncResults.created || 0}</Tag>
                <Tag color="blue">更新: {syncResults.updated || 0}</Tag>
                <Tag color="orange">跳过: {syncResults.skipped || 0}</Tag>
                {syncResults.errors > 0 && (
                  <Tag color="red">错误: {syncResults.errors}</Tag>
                )}
              </div>
            </div>

            {syncResults.details && syncResults.details.length > 0 && (
              <div>
                <Text strong>详细信息:</Text>
                <List
                  size="small"
                  dataSource={syncResults.details}
                  renderItem={item => (
                    <List.Item>
                      <Text>{item.action}: {item.username} - {item.message}</Text>
                    </List.Item>
                  )}
                  style={{ maxHeight: '300px', overflow: 'auto', marginTop: '8px' }}
                />
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default EnhancedUserManagement;
