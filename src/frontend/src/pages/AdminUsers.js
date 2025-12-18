import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Select, Tag, Space, message, Card, Typography, Alert, Spin, Tooltip, Popconfirm } from 'antd';
import { UserOutlined, CrownOutlined, UserSwitchOutlined, CheckCircleOutlined, ExclamationCircleOutlined, SafetyOutlined, KeyOutlined } from '@ant-design/icons';
import { userAPI, securityAPI } from '../services/api';
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
  const [twoFAModalVisible, setTwoFAModalVisible] = useState(false);
  const [twoFAUser, setTwoFAUser] = useState(null);
  const [twoFAStatus, setTwoFAStatus] = useState({});

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

  // 更新用户权限 - 使用正确的API端点
  const updateUserRole = async (userId, newRoleTemplate) => {
    setUpdatingUser(true);
    try {
      // 使用正确的 role-template API
      await userAPI.updateUserRoleTemplate(userId, { role_template: newRoleTemplate });
      const roleText = newRoleTemplate === 'admin' ? t('admin.admin') : t('admin.regularUser');
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

  // 获取用户2FA状态
  const fetchUser2FAStatus = async (userId) => {
    try {
      const res = await securityAPI.admin2FAStatus(userId);
      const data = res.data?.data || res.data;
      return data?.enabled || false;
    } catch (error) {
      console.error('获取用户2FA状态失败:', error);
      return false;
    }
  };

  // 批量获取用户2FA状态
  const fetchAll2FAStatus = async (userList) => {
    const statusMap = {};
    for (const user of userList) {
      statusMap[user.id] = await fetchUser2FAStatus(user.id);
    }
    setTwoFAStatus(statusMap);
  };

  // 管理员为用户启用2FA
  const handleEnable2FA = async (user) => {
    try {
      await securityAPI.adminEnable2FA(user.id);
      message.success(`已为用户 ${user.username} 启用2FA`);
      setTwoFAStatus(prev => ({ ...prev, [user.id]: true }));
    } catch (error) {
      console.error('启用2FA失败:', error);
      message.error('启用2FA失败: ' + (error.response?.data?.error || error.message));
    }
  };

  // 管理员为用户禁用2FA
  const handleDisable2FA = async (user) => {
    try {
      await securityAPI.adminDisable2FA(user.id);
      message.success(`已为用户 ${user.username} 禁用2FA`);
      setTwoFAStatus(prev => ({ ...prev, [user.id]: false }));
    } catch (error) {
      console.error('禁用2FA失败:', error);
      message.error('禁用2FA失败: ' + (error.response?.data?.error || error.message));
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

  // 处理权限变更 - 映射前端值到后端role_template
  const handleRoleChange = (newRole) => {
    if (selectedUser) {
      // 前端 'admin' -> 后端 'admin', 前端 'user' -> 后端 'engineer' (普通用户)
      const roleTemplateMap = {
        'admin': 'admin',
        'user': 'engineer'  // 普通用户对应 engineer 角色模板
      };
      const roleTemplate = roleTemplateMap[newRole] || newRole;
      updateUserRole(selectedUser.id, roleTemplate);
    }
  };

  // 获取用户角色标签 - 使用 role_template 字段
  const getRoleTag = (roleTemplate) => {
    if (roleTemplate === 'admin') {
      return <Tag color="red" icon={<CrownOutlined />}>{t('admin.admin')}</Tag>;
    }
    return <Tag color="blue" icon={<UserOutlined />}>{t('admin.regularUser')}</Tag>;
  };

  // 获取2FA状态标签
  const get2FATag = (userId) => {
    const enabled = twoFAStatus[userId];
    if (enabled) {
      return <Tag color="green" icon={<SafetyOutlined />}>2FA已启用</Tag>;
    }
    return <Tag color="default">2FA未启用</Tag>;
  };

  // 表格列配置
  const columns = [
    {
      title: t('admin.username'),
      dataIndex: 'username',
      key: 'username',
      render: (username, record) => (
        <Space>
          {record.role_template === 'admin' ? <CrownOutlined style={{ color: '#f5222d' }} /> : <UserOutlined />}
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
      dataIndex: 'role_template',
      key: 'role_template',
      render: (roleTemplate) => getRoleTag(roleTemplate),
    },
    {
      title: '2FA状态',
      key: '2fa_status',
      render: (_, record) => get2FATag(record.id),
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
          {twoFAStatus[record.id] ? (
            <Popconfirm
              title="确认禁用2FA？"
              description={`这将禁用用户 ${record.username} 的双因素认证`}
              onConfirm={() => handleDisable2FA(record)}
              okText="确认"
              cancelText="取消"
            >
              <Button
                size="small"
                danger
                icon={<KeyOutlined />}
              >
                禁用2FA
              </Button>
            </Popconfirm>
          ) : (
            <Popconfirm
              title="确认启用2FA？"
              description={`这将为用户 ${record.username} 强制启用双因素认证`}
              onConfirm={() => handleEnable2FA(record)}
              okText="确认"
              cancelText="取消"
            >
              <Button
                size="small"
                type="default"
                icon={<SafetyOutlined />}
              >
                启用2FA
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  // 组件挂载时获取用户列表
  useEffect(() => {
    fetchUsers();
  }, []);

  // 用户列表更新后获取2FA状态
  useEffect(() => {
    if (users.length > 0) {
      fetchAll2FAStatus(users);
    }
  }, [users]);

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
                {getRoleTag(selectedUser.role_template)}
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
