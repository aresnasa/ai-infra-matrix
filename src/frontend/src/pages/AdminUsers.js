import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Select, Tag, Space, message, Card, Typography, Alert, Spin, Popconfirm, Input, Divider, theme, Tabs, Badge, Checkbox, Tooltip, Empty } from 'antd';
import { UserOutlined, CrownOutlined, UserSwitchOutlined, CheckCircleOutlined, ExclamationCircleOutlined, SafetyOutlined, KeyOutlined, QrcodeOutlined, CopyOutlined, ReloadOutlined, ClockCircleOutlined, CheckOutlined, CloseOutlined, TeamOutlined, ToolOutlined, DeleteOutlined, StopOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { QRCodeSVG } from 'qrcode.react';
import { userAPI, securityAPI, adminAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';

const { Title, Text } = Typography;
const { Option } = Select;
const { useToken } = theme;

const AdminUsers = () => {
  const { t } = useI18n();
  const { token } = useToken();
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [permissionModalVisible, setPermissionModalVisible] = useState(false);
  const [selectedUser, setSelectedUser] = useState(null);
  const [updatingUser, setUpdatingUser] = useState(false);
  const [twoFAModalVisible, setTwoFAModalVisible] = useState(false);
  const [twoFAUser, setTwoFAUser] = useState(null);
  const [twoFAStatus, setTwoFAStatus] = useState({});
  const [twoFASetupData, setTwoFASetupData] = useState(null);
  const [twoFALoading, setTwoFALoading] = useState(false);
  const [activeTab, setActiveTab] = useState('users');
  const [pendingApprovals, setPendingApprovals] = useState([]);
  const [pendingLoading, setPendingLoading] = useState(false);
  const [approvalModalVisible, setApprovalModalVisible] = useState(false);
  const [selectedApproval, setSelectedApproval] = useState(null);
  const [approvalLoading, setApprovalLoading] = useState(false);
  const [selectedModules, setSelectedModules] = useState([]);
  const [rejectReason, setRejectReason] = useState('');
  const [rejectModalVisible, setRejectModalVisible] = useState(false);
  const [actionLoading, setActionLoading] = useState({});

  // 删除用户
  const handleDeleteUser = async (user) => {
    setActionLoading(prev => ({ ...prev, [user.id]: 'delete' }));
    try {
      await adminAPI.deleteUser(user.id);
      message.success(t('admin.deleteUserSuccess') || `用户 ${user.username} 已删除`);
      fetchUsers();
    } catch (error) {
      console.error('删除用户失败:', error);
      message.error(t('admin.deleteUserFailed') || `删除用户失败: ${error.response?.data?.error || error.message}`);
    } finally {
      setActionLoading(prev => ({ ...prev, [user.id]: null }));
    }
  };

  // 禁用/启用用户账号
  const handleToggleUserStatus = async (user) => {
    const newStatus = !user.is_active;
    setActionLoading(prev => ({ ...prev, [user.id]: 'status' }));
    try {
      await adminAPI.toggleUserStatus(user.id, newStatus);
      message.success(newStatus 
        ? (t('admin.enableUserSuccess') || `用户 ${user.username} 已启用`) 
        : (t('admin.disableUserSuccess') || `用户 ${user.username} 已禁用`)
      );
      fetchUsers();
    } catch (error) {
      console.error('切换用户状态失败:', error);
      message.error(t('admin.toggleStatusFailed') || `操作失败: ${error.response?.data?.error || error.message}`);
    } finally {
      setActionLoading(prev => ({ ...prev, [user.id]: null }));
    }
  };

  // SRE 权限模块定义
  const SRE_MODULES = [
    { key: 'saltstack', label: 'SaltStack管理', labelEn: 'SaltStack Management', icon: <ToolOutlined /> },
    { key: 'ansible', label: 'Ansible自动化', labelEn: 'Ansible Automation', icon: <ToolOutlined /> },
    { key: 'kubernetes', label: 'Kubernetes集群管理', labelEn: 'Kubernetes Management', icon: <TeamOutlined /> },
    { key: 'hosts', label: '主机管理', labelEn: 'Host Management', icon: <TeamOutlined /> },
    { key: 'nightingale', label: '系统监控', labelEn: 'System Monitoring', icon: <ToolOutlined /> },
    { key: 'audit-logs', label: '日志管理', labelEn: 'Log Management', icon: <SafetyOutlined /> },
  ];

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

  // 获取待审批列表
  const fetchPendingApprovals = async () => {
    setPendingLoading(true);
    try {
      const response = await userAPI.getPendingApprovals();
      const data = response.data || [];
      setPendingApprovals(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('获取待审批列表失败:', error);
      message.error(t('admin.getPendingApprovalsFailed'));
      setPendingApprovals([]);
    } finally {
      setPendingLoading(false);
    }
  };

  // 打开审批模态框
  const openApprovalModal = (approval) => {
    setSelectedApproval(approval);
    // 根据角色模板预选模块
    if (approval.role_template === 'sre') {
      setSelectedModules(SRE_MODULES.map(m => m.key));
    } else {
      setSelectedModules([]);
    }
    setApprovalModalVisible(true);
  };

  // 审批通过
  const handleApprove = async () => {
    if (!selectedApproval) return;
    
    setApprovalLoading(true);
    try {
      await userAPI.approveRegistration(selectedApproval.id);
      
      // 如果选择了模块，为用户授予对应权限
      if (selectedModules.length > 0 && selectedApproval.user_id) {
        try {
          await userAPI.grantUserModules(selectedApproval.user_id, {
            modules: selectedModules,
            verbs: ['read', 'create', 'update', 'delete', 'list']
          });
        } catch (err) {
          console.warn('授予模块权限失败，但用户已审批通过:', err);
        }
      }
      
      message.success(t('admin.approvalSuccess'));
      setApprovalModalVisible(false);
      setSelectedApproval(null);
      setSelectedModules([]);
      fetchPendingApprovals();
      fetchUsers();
    } catch (error) {
      console.error('审批失败:', error);
      message.error(t('admin.approvalFailed') + ': ' + (error.response?.data?.error || error.message));
    } finally {
      setApprovalLoading(false);
    }
  };

  // 打开拒绝模态框
  const openRejectModal = (approval) => {
    setSelectedApproval(approval);
    setRejectReason('');
    setRejectModalVisible(true);
  };

  // 拒绝申请
  const handleReject = async () => {
    if (!selectedApproval) return;
    
    if (!rejectReason.trim()) {
      message.warning(t('admin.rejectReasonRequired'));
      return;
    }
    
    setApprovalLoading(true);
    try {
      await userAPI.rejectRegistration(selectedApproval.id, rejectReason);
      message.success(t('admin.rejectSuccess'));
      setRejectModalVisible(false);
      setSelectedApproval(null);
      setRejectReason('');
      fetchPendingApprovals();
    } catch (error) {
      console.error('拒绝申请失败:', error);
      message.error(t('admin.rejectFailed') + ': ' + (error.response?.data?.error || error.message));
    } finally {
      setApprovalLoading(false);
    }
  };

  // 获取角色模板标签
  const getRoleTemplateTag = (template) => {
    const templates = {
      'admin': { color: 'red', icon: <CrownOutlined />, label: t('roleTemplates.admin') },
      'sre': { color: 'blue', icon: <ToolOutlined />, label: t('roleTemplates.sre') },
      'data-developer': { color: 'green', icon: <TeamOutlined />, label: t('roleTemplates.dataDeveloper') },
      'model-developer': { color: 'purple', icon: <TeamOutlined />, label: t('roleTemplates.modelDeveloper') },
      'engineer': { color: 'orange', icon: <UserOutlined />, label: t('roleTemplates.engineer') },
      'audit': { color: 'cyan', icon: <SafetyOutlined />, label: t('roleTemplates.audit') },
    };
    const config = templates[template] || { color: 'default', icon: <UserOutlined />, label: template || t('admin.regularUser') };
    return <Tag color={config.color} icon={config.icon}>{config.label}</Tag>;
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

  // 管理员为用户启用2FA - 显示二维码弹窗
  const handleEnable2FA = async (user) => {
    setTwoFAUser(user);
    setTwoFALoading(true);
    setTwoFAModalVisible(true);
    
    try {
      const res = await securityAPI.adminEnable2FA(user.id);
      const data = res.data?.data || res.data;
      setTwoFASetupData({
        secret: data.secret,
        qrCode: data.qr_code,
        issuer: data.issuer || 'AI-Infra-Matrix',
        account: data.account || user.username,
        recoveryCodes: data.recovery_codes || []
      });
      setTwoFAStatus(prev => ({ ...prev, [user.id]: true }));
      message.success(`已为用户 ${user.username} 启用2FA`);
    } catch (error) {
      console.error('启用2FA失败:', error);
      message.error('启用2FA失败: ' + (error.response?.data?.error || error.message));
      setTwoFAModalVisible(false);
    } finally {
      setTwoFALoading(false);
    }
  };

  // 关闭2FA设置弹窗
  const close2FAModal = () => {
    setTwoFAModalVisible(false);
    setTwoFAUser(null);
    setTwoFASetupData(null);
  };

  // 复制到剪贴板（兼容 HTTP 和 HTTPS 环境）
  const copyToClipboard = async (text, label) => {
    // 首先尝试现代 Clipboard API
    if (navigator.clipboard && window.isSecureContext) {
      try {
        await navigator.clipboard.writeText(text);
        message.success(`${label} 已复制到剪贴板`);
        return;
      } catch (err) {
        console.warn('Clipboard API failed, falling back to execCommand:', err);
      }
    }
    
    // 备用方案：使用传统的 execCommand 方式
    const textArea = document.createElement('textarea');
    textArea.value = text;
    
    // 避免滚动到底部
    textArea.style.position = 'fixed';
    textArea.style.left = '-9999px';
    textArea.style.top = '-9999px';
    textArea.style.opacity = '0';
    
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    
    try {
      const successful = document.execCommand('copy');
      if (successful) {
        message.success(`${label} 已复制到剪贴板`);
      } else {
        message.error('复制失败，请手动复制');
      }
    } catch (err) {
      console.error('execCommand copy failed:', err);
      message.error('复制失败，请手动复制');
    } finally {
      document.body.removeChild(textArea);
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
          {/* 禁用/启用账号按钮 */}
          <Popconfirm
            title={record.is_active ? (t('admin.confirmDisableUser') || '确认禁用账号？') : (t('admin.confirmEnableUser') || '确认启用账号？')}
            description={record.is_active 
              ? `禁用后用户 ${record.username} 将无法登录`
              : `启用后用户 ${record.username} 将恢复登录权限`
            }
            onConfirm={() => handleToggleUserStatus(record)}
            okText={t('common.confirm') || '确认'}
            cancelText={t('common.cancel') || '取消'}
          >
            <Tooltip title={record.is_active ? (t('admin.disableUser') || '禁用账号') : (t('admin.enableUser') || '启用账号')}>
              <Button
                size="small"
                type={record.is_active ? 'default' : 'primary'}
                icon={record.is_active ? <StopOutlined /> : <PlayCircleOutlined />}
                loading={actionLoading[record.id] === 'status'}
              >
                {record.is_active ? (t('admin.disable') || '禁用') : (t('admin.enable') || '启用')}
              </Button>
            </Tooltip>
          </Popconfirm>
          {/* 删除用户按钮 */}
          <Popconfirm
            title={t('admin.confirmDeleteUser') || '确认删除用户？'}
            description={`这将永久删除用户 ${record.username}，此操作不可恢复！`}
            onConfirm={() => handleDeleteUser(record)}
            okText={t('common.confirm') || '确认'}
            cancelText={t('common.cancel') || '取消'}
            okButtonProps={{ danger: true }}
          >
            <Tooltip title={t('admin.deleteUser') || '删除用户'}>
              <Button
                size="small"
                danger
                icon={<DeleteOutlined />}
                loading={actionLoading[record.id] === 'delete'}
              >
                {t('common.delete') || '删除'}
              </Button>
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // 组件挂载时获取用户列表
  useEffect(() => {
    fetchUsers();
    fetchPendingApprovals();
  }, []);

  // 用户列表更新后获取2FA状态
  useEffect(() => {
    if (users.length > 0) {
      fetchAll2FAStatus(users);
    }
  }, [users]);

  // 待审批列表表格列配置
  const pendingColumns = [
    {
      title: t('admin.username'),
      dataIndex: 'username',
      key: 'username',
      render: (username) => (
        <Space>
          <UserOutlined />
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
      title: t('admin.department'),
      dataIndex: 'department',
      key: 'department',
      render: (department) => department || '-',
    },
    {
      title: t('admin.roleTemplate'),
      dataIndex: 'role_template',
      key: 'role_template',
      render: (roleTemplate) => getRoleTemplateTag(roleTemplate),
    },
    {
      title: t('admin.applyTime'),
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
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="primary"
            size="small"
            icon={<CheckOutlined />}
            onClick={() => openApprovalModal(record)}
          >
            {t('admin.approve')}
          </Button>
          <Button
            danger
            size="small"
            icon={<CloseOutlined />}
            onClick={() => openRejectModal(record)}
          >
            {t('admin.reject')}
          </Button>
        </Space>
      ),
    },
  ];

  // Tab 配置
  const tabItems = [
    {
      key: 'users',
      label: (
        <span>
          <UserOutlined />
          {t('admin.userList')}
        </span>
      ),
      children: (
        <>
          <Alert
            message={t('admin.permissionNote')}
            description={t('admin.permissionNoteDesc')}
            type="info"
            showIcon
            style={{ marginBottom: '24px' }}
          />
          <Card 
            title={t('admin.userList')} 
            size="small"
            extra={
              <Button icon={<ReloadOutlined />} onClick={fetchUsers} loading={loading}>
                {t('common.refresh')}
              </Button>
            }
          >
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
        </>
      ),
    },
    {
      key: 'pending',
      label: (
        <Badge count={pendingApprovals.length} offset={[10, 0]}>
          <span>
            <ClockCircleOutlined />
            {t('admin.pendingApprovals')}
          </span>
        </Badge>
      ),
      children: (
        <>
          <Alert
            message={t('admin.pendingApprovalsNote')}
            description={t('admin.pendingApprovalsNoteDesc')}
            type="warning"
            showIcon
            style={{ marginBottom: '24px' }}
          />
          <Card 
            title={t('admin.pendingApprovalsList')} 
            size="small"
            extra={
              <Button icon={<ReloadOutlined />} onClick={fetchPendingApprovals} loading={pendingLoading}>
                {t('common.refresh')}
              </Button>
            }
          >
            {pendingApprovals.length === 0 ? (
              <Empty description={t('admin.noPendingApprovals')} />
            ) : (
              <Table
                columns={pendingColumns}
                dataSource={pendingApprovals}
                loading={pendingLoading}
                rowKey="id"
                pagination={{
                  total: pendingApprovals.length,
                  pageSize: 10,
                  showSizeChanger: true,
                  showQuickJumper: true,
                  showTotal: (total, range) => t('admin.showing').replace('{start}', range[0]).replace('{end}', range[1]).replace('{total}', total),
                }}
              />
            )}
          </Card>
        </>
      ),
    },
  ];

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

          <Tabs 
            activeKey={activeTab} 
            onChange={setActiveTab}
            items={tabItems}
          />
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

      {/* 2FA设置弹窗 - 显示二维码和恢复码 */}
      <Modal
        title={
          <Space>
            <QrcodeOutlined />
            为用户 {twoFAUser?.username} 设置双因素认证
          </Space>
        }
        open={twoFAModalVisible}
        onCancel={close2FAModal}
        footer={[
          <Button key="close" type="primary" onClick={close2FAModal}>
            我已保存，关闭
          </Button>
        ]}
        width={600}
        centered
      >
        {twoFALoading ? (
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <Spin size="large" />
            <div style={{ marginTop: '16px' }}>
              <Text>正在生成2FA密钥...</Text>
            </div>
          </div>
        ) : twoFASetupData ? (
          <div>
            <Alert
              message="重要提示"
              description="请让用户使用 Google Authenticator、Microsoft Authenticator 或其他 TOTP 应用扫描下方二维码。恢复码需要安全保存，丢失后将无法恢复账户。"
              type="warning"
              showIcon
              style={{ marginBottom: '24px' }}
            />

            {/* 二维码区域 */}
            <Card size="small" title="扫描二维码" style={{ marginBottom: '16px' }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: '24px' }}>
                <div style={{ 
                  padding: '16px', 
                  background: token.colorBgContainer,
                  borderRadius: '8px',
                  border: `1px solid ${token.colorBorder}`
                }}>
                  <QRCodeSVG 
                    value={twoFASetupData.qrCode} 
                    size={180}
                    level="M"
                    bgColor={token.colorBgContainer}
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ marginBottom: '12px' }}>
                    <Text strong>账户: </Text>
                    <Text>{twoFASetupData.account}</Text>
                  </div>
                  <div style={{ marginBottom: '12px' }}>
                    <Text strong>发行者: </Text>
                    <Text>{twoFASetupData.issuer}</Text>
                  </div>
                  <Divider style={{ margin: '12px 0' }} />
                  <div>
                    <Text strong>手动输入密钥: </Text>
                    <div style={{ 
                      display: 'flex', 
                      alignItems: 'center', 
                      gap: '8px',
                      marginTop: '8px'
                    }}>
                      <Input.Password 
                        value={twoFASetupData.secret} 
                        readOnly
                        style={{ fontFamily: 'monospace' }}
                      />
                      <Button 
                        icon={<CopyOutlined />}
                        onClick={() => copyToClipboard(twoFASetupData.secret, '密钥')}
                      />
                    </div>
                  </div>
                </div>
              </div>
            </Card>

            {/* 恢复码区域 */}
            {twoFASetupData.recoveryCodes?.length > 0 && (
              <Card 
                size="small" 
                title={
                  <Space>
                    <KeyOutlined />
                    恢复码（请妥善保存）
                  </Space>
                }
                extra={
                  <Button 
                    size="small"
                    icon={<CopyOutlined />}
                    onClick={() => copyToClipboard(twoFASetupData.recoveryCodes.join('\n'), '恢复码')}
                  >
                    复制全部
                  </Button>
                }
              >
                <Alert
                  message="恢复码用于在无法访问认证器应用时登录账户"
                  description="每个恢复码只能使用一次，请打印或安全保存"
                  type="info"
                  showIcon
                  style={{ marginBottom: '12px' }}
                />
                <div style={{ 
                  display: 'grid', 
                  gridTemplateColumns: 'repeat(2, 1fr)', 
                  gap: '8px',
                  padding: '12px',
                  background: token.colorBgLayout,
                  borderRadius: '4px',
                  fontFamily: 'monospace'
                }}>
                  {twoFASetupData.recoveryCodes.map((code, index) => (
                    <div key={index} style={{ 
                      padding: '4px 8px',
                      background: token.colorBgContainer,
                      border: `1px solid ${token.colorBorder}`,
                      borderRadius: '4px',
                      textAlign: 'center'
                    }}>
                      {code}
                    </div>
                  ))}
                </div>
              </Card>
            )}
          </div>
        ) : (
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <Text type="secondary">无数据</Text>
          </div>
        )}
      </Modal>

      {/* 审批确认模态框 */}
      <Modal
        title={
          <Space>
            <CheckCircleOutlined style={{ color: '#52c41a' }} />
            {t('admin.approveRegistration')}
          </Space>
        }
        open={approvalModalVisible}
        onCancel={() => {
          setApprovalModalVisible(false);
          setSelectedApproval(null);
          setSelectedModules([]);
        }}
        footer={[
          <Button key="cancel" onClick={() => {
            setApprovalModalVisible(false);
            setSelectedApproval(null);
            setSelectedModules([]);
          }}>
            {t('common.cancel')}
          </Button>,
          <Button 
            key="approve" 
            type="primary" 
            loading={approvalLoading}
            onClick={handleApprove}
            icon={<CheckOutlined />}
          >
            {t('admin.confirmApprove')}
          </Button>
        ]}
        width={600}
      >
        {selectedApproval && (
          <div>
            <Alert
              message={t('admin.approvalConfirmMessage')}
              type="info"
              showIcon
              style={{ marginBottom: '16px' }}
            />
            
            <Card size="small" style={{ marginBottom: '16px' }}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <div>
                  <Text strong>{t('admin.username')}: </Text>
                  <Text>{selectedApproval.username}</Text>
                </div>
                <div>
                  <Text strong>{t('admin.email')}: </Text>
                  <Text>{selectedApproval.email || t('admin.notSet')}</Text>
                </div>
                <div>
                  <Text strong>{t('admin.department')}: </Text>
                  <Text>{selectedApproval.department || '-'}</Text>
                </div>
                <div>
                  <Text strong>{t('admin.roleTemplate')}: </Text>
                  {getRoleTemplateTag(selectedApproval.role_template)}
                </div>
              </Space>
            </Card>

            {/* 权限模块选择 - 仅对 SRE 和其他需要细分权限的角色显示 */}
            {(selectedApproval.role_template === 'sre' || selectedApproval.role_template === 'engineer') && (
              <Card 
                size="small" 
                title={
                  <Space>
                    <ToolOutlined />
                    {t('admin.selectPermissionModules')}
                  </Space>
                }
              >
                <Alert
                  message={t('admin.permissionModulesNote')}
                  type="warning"
                  showIcon
                  style={{ marginBottom: '12px' }}
                />
                <Checkbox.Group
                  value={selectedModules}
                  onChange={setSelectedModules}
                  style={{ width: '100%' }}
                >
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '8px' }}>
                    {SRE_MODULES.map(module => (
                      <Checkbox key={module.key} value={module.key}>
                        <Space>
                          {module.icon}
                          <span>{module.label}</span>
                        </Space>
                      </Checkbox>
                    ))}
                  </div>
                </Checkbox.Group>
                <div style={{ marginTop: '12px' }}>
                  <Button 
                    size="small" 
                    onClick={() => setSelectedModules(SRE_MODULES.map(m => m.key))}
                  >
                    {t('admin.selectAll')}
                  </Button>
                  <Button 
                    size="small" 
                    style={{ marginLeft: '8px' }}
                    onClick={() => setSelectedModules([])}
                  >
                    {t('admin.clearSelection')}
                  </Button>
                </div>
              </Card>
            )}
          </div>
        )}
      </Modal>

      {/* 拒绝申请模态框 */}
      <Modal
        title={
          <Space>
            <CloseOutlined style={{ color: '#ff4d4f' }} />
            {t('admin.rejectRegistration')}
          </Space>
        }
        open={rejectModalVisible}
        onCancel={() => {
          setRejectModalVisible(false);
          setSelectedApproval(null);
          setRejectReason('');
        }}
        footer={[
          <Button key="cancel" onClick={() => {
            setRejectModalVisible(false);
            setSelectedApproval(null);
            setRejectReason('');
          }}>
            {t('common.cancel')}
          </Button>,
          <Button 
            key="reject" 
            danger 
            loading={approvalLoading}
            onClick={handleReject}
            icon={<CloseOutlined />}
          >
            {t('admin.confirmReject')}
          </Button>
        ]}
        width={500}
      >
        {selectedApproval && (
          <div>
            <Alert
              message={t('admin.rejectConfirmMessage')}
              description={`${t('admin.username')}: ${selectedApproval.username}`}
              type="warning"
              showIcon
              style={{ marginBottom: '16px' }}
            />
            
            <div>
              <Text strong>{t('admin.rejectReason')}: </Text>
              <Text type="danger">*</Text>
            </div>
            <Input.TextArea
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
              placeholder={t('admin.rejectReasonPlaceholder')}
              rows={4}
              style={{ marginTop: '8px' }}
            />
          </div>
        )}
      </Modal>
    </div>
  );
};

export default AdminUsers;
