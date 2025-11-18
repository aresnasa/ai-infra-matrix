import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Select, Tag, Space, message, Card, Typography, Alert, Spin } from 'antd';
import { UserOutlined, CrownOutlined, UserSwitchOutlined, CheckCircleOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { userAPI } from '../services/api';

const { Title, Text } = Typography;
const { Option } = Select;

const AdminUsers = () => {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [permissionModalVisible, setPermissionModalVisible] = useState(false);
  const [selectedUser, setSelectedUser] = useState(null);
  const [updatingUser, setUpdatingUser] = useState(false);

  // 获取用户列表
  const fetchUsers = async () => {
    setLoading(true);
    try {
      const response = await userAPI.getUsers();
      if (response.data && Array.isArray(response.data)) {
        setUsers(response.data);
      } else if (response.data && Array.isArray(response.data.users)) {
        setUsers(response.data.users);
      } else {
        setUsers([]);
        message.warning('获取用户数据格式异常');
      }
    } catch (error) {
      console.error('获取用户列表失败:', error);
      message.error('获取用户列表失败');
      setUsers([]);
    } finally {
      setLoading(false);
    }
  };

  // 更新用户权限
  const updateUserRole = async (userId, newRole) => {
    setUpdatingUser(true);
    try {
      // 使用用户API更新用户信息，包含角色信息
      const userData = { role: newRole };
      await userAPI.updateUser(userId, userData);
      message.success(`用户权限已更新为${newRole === 'admin' ? '管理员' : '普通用户'}`);
      fetchUsers(); // 重新获取用户列表
      setPermissionModalVisible(false);
      setSelectedUser(null);
    } catch (error) {
      console.error('更新用户权限失败:', error);
      message.error('更新用户权限失败');
    } finally {
      setUpdatingUser(false);
    }
  };

  // 打开权限修改模态框
  const openPermissionModal = (user) => {
    setSelectedUser(user);
    setPermissionModalVisible(true);
  };

  // 关闭权限修改模态框
  const closePermissionModal = () => {
    setPermissionModalVisible(false);
    setSelectedUser(null);
  };

  // 处理权限变更
  const handleRoleChange = (newRole) => {
    if (selectedUser) {
      updateUserRole(selectedUser.id, newRole);
    }
  };

  // 获取用户角色标签
  const getRoleTag = (role) => {
    if (role === 'admin') {
      return <Tag color="red" icon={<CrownOutlined />}>管理员</Tag>;
    }
    return <Tag color="blue" icon={<UserOutlined />}>普通用户</Tag>;
  };

  // 表格列配置
  const columns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (username, record) => (
        <Space>
          {record.role === 'admin' ? <CrownOutlined style={{ color: '#f5222d' }} /> : <UserOutlined />}
          <Text strong>{username}</Text>
        </Space>
      ),
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      render: (email) => email || '未设置',
    },
    {
      title: '角色',
      dataIndex: 'role',
      key: 'role',
      render: (role) => getRoleTag(role),
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (isActive) => (
        <Tag color={isActive ? 'green' : 'red'}>
          {isActive ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}
          {isActive ? '活跃' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (createdAt) => {
        if (!createdAt) return '未知';
        try {
          return new Date(createdAt).toLocaleString('zh-CN');
        } catch {
          return createdAt;
        }
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="middle">
          <Button
            type="primary"
            size="small"
            icon={<UserSwitchOutlined />}
            onClick={() => openPermissionModal(record)}
          >
            修改权限
          </Button>
        </Space>
      ),
    },
  ];

  // 组件挂载时获取用户列表
  useEffect(() => {
    fetchUsers();
  }, []);

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div>
            <Title level={2} style={{ marginBottom: '8px' }}>
              <UserOutlined style={{ marginRight: '12px' }} />
              用户权限管理
            </Title>
            <Text type="secondary">
              管理系统用户权限，只有管理员用户才能访问此页面
            </Text>
          </div>

          <Alert
            message="权限说明"
            description="管理员用户可以管理系统所有功能，普通用户只能使用基本功能。您可以在这里调整用户的权限级别。"
            type="info"
            showIcon
            style={{ marginBottom: '24px' }}
          />

          <Card title="用户列表" size="small">
            <Table
              columns={columns}
              dataSource={users}
              loading={loading}
              rowKey="id"
              pagination={{
                total: users.length,
                pageSize: 10,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
              }}
              locale={{
                emptyText: '暂无用户数据',
              }}
            />
          </Card>
        </Space>
      </Card>

      {/* 权限修改模态框 */}
      <Modal
        title={
          <Space>
            <UserSwitchOutlined />
            修改用户权限
          </Space>
        }
        open={permissionModalVisible}
        onCancel={closePermissionModal}
        footer={null}
        width={400}
      >
        {selectedUser && (
          <div style={{ padding: '16px 0' }}>
            <Space direction="vertical" size="middle" style={{ width: '100%' }}>
              <div>
                <Text strong>用户名：</Text>
                <Text>{selectedUser.username}</Text>
              </div>
              <div>
                <Text strong>当前角色：</Text>
                {getRoleTag(selectedUser.role)}
              </div>

              <div>
                <Text strong style={{ marginBottom: '8px', display: 'block' }}>
                  选择新角色：
                </Text>
                <Select
                  style={{ width: '100%' }}
                  placeholder="请选择用户角色"
                  onChange={handleRoleChange}
                  loading={updatingUser}
                  disabled={updatingUser}
                >
                  <Option value="user">
                    <Space>
                      <UserOutlined />
                      普通用户
                    </Space>
                  </Option>
                  <Option value="admin">
                    <Space>
                      <CrownOutlined />
                      管理员
                    </Space>
                  </Option>
                </Select>
              </div>

              {updatingUser && (
                <div style={{ textAlign: 'center', padding: '16px' }}>
                  <Spin size="small" />
                  <div style={{ marginTop: '8px' }}>
                    <Text type="secondary">正在更新用户权限...</Text>
                  </div>
                </div>
              )}
            </Space>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default AdminUsers;
