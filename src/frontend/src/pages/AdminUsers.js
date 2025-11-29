import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Select, Tag, Space, message, Card, Typography, Alert, Spin } from 'antd';
import { UserOutlined, CrownOutlined, UserSwitchOutlined, CheckCircleOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { userAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';

const { Title, Text } = Typography;
const { Option } = Select;

const AdminUsers = () => {
  const { t } = useI18n();
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
        message.warning(t('admin.getUsersFormatError'));
      }
    } catch (error) {
      console.error('获取用户列表失败:', error);
      message.error(t('admin.getUsersFailed'));
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
      const roleText = newRole === 'admin' ? t('admin.admin') : t('admin.regularUser');
      message.success(t('admin.permissionUpdated').replace('{role}', roleText));
      fetchUsers(); // 重新获取用户列表
      setPermissionModalVisible(false);
      setSelectedUser(null);
    } catch (error) {
      console.error('更新用户权限失败:', error);
      message.error(t('admin.updatePermissionFailed'));
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
      return <Tag color="red" icon={<CrownOutlined />}>{t('admin.admin')}</Tag>;
    }
    return <Tag color="blue" icon={<UserOutlined />}>{t('admin.regularUser')}</Tag>;
  };

  // 表格列配置
  const columns = [
    {
      title: t('admin.username'),
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
      title: t('admin.email'),
      dataIndex: 'email',
      key: 'email',
      render: (email) => email || t('admin.notSet'),
    },
    {
      title: t('admin.role'),
      dataIndex: 'role',
      key: 'role',
      render: (role) => getRoleTag(role),
    },
    {
      title: t('admin.status'),
      dataIndex: 'is_active',
      key: 'is_active',
      render: (isActive) => (
        <Tag color={isActive ? 'green' : 'red'}>
          {isActive ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}
          {isActive ? t('admin.active') : t('admin.disabled')}
        </Tag>
      ),
    },
    {
      title: t('admin.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (createdAt) => {
        if (!createdAt) return t('admin.unknown');
        try {
          return new Date(createdAt).toLocaleString('zh-CN');
        } catch {
          return createdAt;
        }
      },
    },
    {
      title: t('admin.action'),
      key: 'action',
      render: (_, record) => (
        <Space size="middle">
          <Button
            type="primary"
            size="small"
            icon={<UserSwitchOutlined />}
            onClick={() => openPermissionModal(record)}
          >
            {t('admin.modifyPermission')}
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
              {t('admin.userPermissions')}
            </Title>
            <Text type="secondary">
              {t('admin.userPermissionsDesc')}
            </Text>
          </div>

          <Alert
            message={t('admin.permissionNote')}
            description={t('admin.permissionNoteDesc')}
            type="info"
            showIcon
            style={{ marginBottom: '24px' }}
          />

          <Card title={t('admin.userList')} size="small">
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
                showTotal: (total, range) => t('admin.showing').replace('{start}', range[0]).replace('{end}', range[1]).replace('{total}', total),
              }}
              locale={{
                emptyText: t('admin.noUserData'),
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
            {t('admin.modifyUserPermission')}
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
                <Text strong>{t('admin.username')}：</Text>
                <Text>{selectedUser.username}</Text>
              </div>
              <div>
                <Text strong>{t('admin.currentRole')}：</Text>
                {getRoleTag(selectedUser.role)}
              </div>

              <div>
                <Text strong style={{ marginBottom: '8px', display: 'block' }}>
                  {t('admin.selectNewRole')}：
                </Text>
                <Select
                  style={{ width: '100%' }}
                  placeholder={t('admin.selectUserRole')}
                  onChange={handleRoleChange}
                  loading={updatingUser}
                  disabled={updatingUser}
                >
                  <Option value="user">
                    <Space>
                      <UserOutlined />
                      {t('admin.regularUser')}
                    </Space>
                  </Option>
                  <Option value="admin">
                    <Space>
                      <CrownOutlined />
                      {t('admin.admin')}
                    </Space>
                  </Option>
                </Select>
              </div>

              {updatingUser && (
                <div style={{ textAlign: 'center', padding: '16px' }}>
                  <Spin size="small" />
                  <div style={{ marginTop: '8px' }}>
                    <Text type="secondary">{t('admin.updatingPermission')}</Text>
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
