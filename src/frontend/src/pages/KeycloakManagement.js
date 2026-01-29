import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  Card, 
  Row, 
  Col, 
  Button, 
  Table, 
  Tag, 
  Space, 
  message, 
  Modal, 
  Form, 
  Input, 
  Select,
  Tooltip,
  Statistic,
  Tabs,
  Alert,
  Badge,
  Typography,
  Drawer,
  Descriptions,
  Switch,
  Divider,
  Spin
} from 'antd';
import { 
  UserOutlined, 
  TeamOutlined, 
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
  KeyOutlined,
  LockOutlined,
  SettingOutlined,
  LinkOutlined,
  GlobalOutlined,
  ApiOutlined,
  EditOutlined,
  DeleteOutlined,
  ExportOutlined,
  DesktopOutlined
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import api from '../services/api';

const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;
const { Option } = Select;

const KeycloakManagement = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [serverInfo, setServerInfo] = useState(null);
  const [realms, setRealms] = useState([]);
  const [users, setUsers] = useState([]);
  const [clients, setClients] = useState([]);
  const [groups, setGroups] = useState([]);
  const [selectedRealm, setSelectedRealm] = useState('ai-infra');
  const [statistics, setStatistics] = useState({
    totalUsers: 0,
    activeUsers: 0,
    totalClients: 0,
    totalGroups: 0,
    totalRealms: 0
  });
  const [activeTab, setActiveTab] = useState('overview');
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [selectedUser, setSelectedUser] = useState(null);
  const [addUserModalVisible, setAddUserModalVisible] = useState(false);
  const [form] = Form.useForm();

  // 获取 Keycloak 服务器信息
  const fetchServerInfo = useCallback(async () => {
    try {
      const response = await api.get('/keycloak/server-info');
      setServerInfo(response.data);
    } catch (error) {
      console.error('Failed to fetch Keycloak server info:', error);
    }
  }, []);

  // 获取 Realms 列表
  const fetchRealms = useCallback(async () => {
    try {
      const response = await api.get('/keycloak/realms');
      setRealms(response.data || []);
      setStatistics(prev => ({ ...prev, totalRealms: (response.data || []).length }));
    } catch (error) {
      console.error('Failed to fetch realms:', error);
    }
  }, []);

  // 获取用户列表
  const fetchUsers = useCallback(async () => {
    try {
      const response = await api.get(`/keycloak/realms/${selectedRealm}/users`);
      const userList = response.data || [];
      setUsers(userList);
      setStatistics(prev => ({
        ...prev,
        totalUsers: userList.length,
        activeUsers: userList.filter(u => u.enabled).length
      }));
    } catch (error) {
      console.error('Failed to fetch users:', error);
      setUsers([]);
    }
  }, [selectedRealm]);

  // 获取客户端列表
  const fetchClients = useCallback(async () => {
    try {
      const response = await api.get(`/keycloak/realms/${selectedRealm}/clients`);
      const clientList = response.data || [];
      setClients(clientList);
      setStatistics(prev => ({ ...prev, totalClients: clientList.length }));
    } catch (error) {
      console.error('Failed to fetch clients:', error);
      setClients([]);
    }
  }, [selectedRealm]);

  // 获取组列表
  const fetchGroups = useCallback(async () => {
    try {
      const response = await api.get(`/keycloak/realms/${selectedRealm}/groups`);
      const groupList = response.data || [];
      setGroups(groupList);
      setStatistics(prev => ({ ...prev, totalGroups: groupList.length }));
    } catch (error) {
      console.error('Failed to fetch groups:', error);
      setGroups([]);
    }
  }, [selectedRealm]);

  // 加载所有数据
  const fetchAllData = useCallback(async () => {
    setLoading(true);
    try {
      await Promise.all([
        fetchServerInfo(),
        fetchRealms(),
        fetchUsers(),
        fetchClients(),
        fetchGroups()
      ]);
    } catch (error) {
      message.error(t('keycloak.fetchFailed', '获取 Keycloak 数据失败'));
    } finally {
      setLoading(false);
    }
  }, [fetchServerInfo, fetchRealms, fetchUsers, fetchClients, fetchGroups, t]);

  useEffect(() => {
    fetchAllData();
  }, [fetchAllData]);

  // 当选择的 Realm 变化时重新获取数据
  useEffect(() => {
    if (selectedRealm) {
      fetchUsers();
      fetchClients();
      fetchGroups();
    }
  }, [selectedRealm, fetchUsers, fetchClients, fetchGroups]);

  // 打开 Keycloak 控制台
  const openKeycloakConsole = () => {
    const keycloakUrl = process.env.REACT_APP_KEYCLOAK_URL || `${window.location.protocol}//${window.location.hostname}:8180/auth`;
    window.open(`${keycloakUrl}/admin`, '_blank');
  };

  // 创建用户
  const handleCreateUser = async (values) => {
    try {
      await api.post(`/keycloak/realms/${selectedRealm}/users`, values);
      message.success(t('keycloak.userCreated', '用户创建成功'));
      setAddUserModalVisible(false);
      form.resetFields();
      fetchUsers();
    } catch (error) {
      message.error(t('keycloak.userCreateFailed', '创建用户失败'));
    }
  };

  // 启用/禁用用户
  const handleToggleUser = async (userId, enabled) => {
    try {
      await api.put(`/keycloak/realms/${selectedRealm}/users/${userId}`, { enabled: !enabled });
      message.success(enabled ? t('keycloak.userDisabled', '用户已禁用') : t('keycloak.userEnabled', '用户已启用'));
      fetchUsers();
    } catch (error) {
      message.error(t('keycloak.toggleFailed', '操作失败'));
    }
  };

  // 删除用户
  const handleDeleteUser = async (userId, username) => {
    Modal.confirm({
      title: t('keycloak.confirmDelete', '确认删除'),
      content: t('keycloak.confirmDeleteUser', `确定要删除用户 ${username} 吗？`),
      okText: t('common.delete', '删除'),
      okType: 'danger',
      cancelText: t('common.cancel', '取消'),
      onOk: async () => {
        try {
          await api.delete(`/keycloak/realms/${selectedRealm}/users/${userId}`);
          message.success(t('keycloak.userDeleted', '用户已删除'));
          fetchUsers();
        } catch (error) {
          message.error(t('keycloak.deleteFailed', '删除失败'));
        }
      }
    });
  };

  // 用户表格列定义
  const userColumns = [
    {
      title: t('keycloak.username', '用户名'),
      dataIndex: 'username',
      key: 'username',
      render: (text, record) => (
        <Space>
          <UserOutlined />
          <a onClick={() => { setSelectedUser(record); setDetailDrawerVisible(true); }}>{text}</a>
        </Space>
      )
    },
    {
      title: t('keycloak.email', '邮箱'),
      dataIndex: 'email',
      key: 'email'
    },
    {
      title: t('keycloak.firstName', '名'),
      dataIndex: 'firstName',
      key: 'firstName'
    },
    {
      title: t('keycloak.lastName', '姓'),
      dataIndex: 'lastName',
      key: 'lastName'
    },
    {
      title: t('keycloak.status', '状态'),
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled) => (
        <Tag color={enabled ? 'success' : 'error'}>
          {enabled ? t('keycloak.enabled', '启用') : t('keycloak.disabled', '禁用')}
        </Tag>
      )
    },
    {
      title: t('keycloak.emailVerified', '邮箱验证'),
      dataIndex: 'emailVerified',
      key: 'emailVerified',
      render: (verified) => (
        verified ? <CheckCircleOutlined style={{ color: '#52c41a' }} /> : <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
      )
    },
    {
      title: t('common.actions', '操作'),
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Tooltip title={record.enabled ? t('keycloak.disable', '禁用') : t('keycloak.enable', '启用')}>
            <Switch 
              size="small"
              checked={record.enabled}
              onChange={() => handleToggleUser(record.id, record.enabled)}
            />
          </Tooltip>
          <Tooltip title={t('common.delete', '删除')}>
            <Button 
              type="text" 
              danger 
              icon={<DeleteOutlined />}
              onClick={() => handleDeleteUser(record.id, record.username)}
            />
          </Tooltip>
        </Space>
      )
    }
  ];

  // 客户端表格列定义
  const clientColumns = [
    {
      title: t('keycloak.clientId', '客户端ID'),
      dataIndex: 'clientId',
      key: 'clientId',
      render: (text) => (
        <Space>
          <ApiOutlined />
          <Text strong>{text}</Text>
        </Space>
      )
    },
    {
      title: t('keycloak.name', '名称'),
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: t('keycloak.protocol', '协议'),
      dataIndex: 'protocol',
      key: 'protocol',
      render: (protocol) => <Tag color="blue">{protocol || 'openid-connect'}</Tag>
    },
    {
      title: t('keycloak.enabled', '状态'),
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled) => (
        <Badge status={enabled ? 'success' : 'default'} text={enabled ? '启用' : '禁用'} />
      )
    },
    {
      title: t('keycloak.publicClient', '公开客户端'),
      dataIndex: 'publicClient',
      key: 'publicClient',
      render: (isPublic) => isPublic ? '是' : '否'
    }
  ];

  // 组表格列定义
  const groupColumns = [
    {
      title: t('keycloak.groupName', '组名'),
      dataIndex: 'name',
      key: 'name',
      render: (text) => (
        <Space>
          <TeamOutlined />
          <Text strong>{text}</Text>
        </Space>
      )
    },
    {
      title: t('keycloak.path', '路径'),
      dataIndex: 'path',
      key: 'path',
      render: (path) => <Text code>{path}</Text>
    },
    {
      title: t('keycloak.subGroups', '子组数量'),
      dataIndex: 'subGroups',
      key: 'subGroups',
      render: (subGroups) => (subGroups || []).length
    }
  ];

  return (
    <div style={{ padding: '24px' }}>
      {/* 页面标题 */}
      <Row justify="space-between" align="middle" style={{ marginBottom: 24 }}>
        <Col>
          <Title level={2}>
            <SafetyCertificateOutlined style={{ marginRight: 12 }} />
            {t('keycloak.title', 'Keycloak 身份认证管理')}
          </Title>
          <Text type="secondary">
            {t('keycloak.description', '统一身份认证、单点登录 (SSO)、用户和客户端管理')}
          </Text>
        </Col>
        <Col>
          <Space>
            <Button 
              type="primary"
              icon={<DesktopOutlined />}
              onClick={() => navigate('/keycloak-ui')}
            >
              {t('keycloak.openEmbedUI', '嵌入式控制台')}
            </Button>
            <Button 
              icon={<ExportOutlined />}
              onClick={openKeycloakConsole}
            >
              {t('keycloak.openConsole', '打开管理控制台')}
            </Button>
            <Button 
              icon={<ReloadOutlined />} 
              onClick={fetchAllData}
              loading={loading}
            >
              {t('common.refresh', '刷新')}
            </Button>
          </Space>
        </Col>
      </Row>

      {/* 服务状态提示 */}
      {serverInfo ? (
        <Alert
          message={t('keycloak.serverOnline', 'Keycloak 服务运行正常')}
          description={`版本: ${serverInfo.systemInfo?.version || 'Unknown'} | Realm: ${selectedRealm}`}
          type="success"
          showIcon
          style={{ marginBottom: 24 }}
        />
      ) : (
        <Alert
          message={t('keycloak.serverOffline', 'Keycloak 服务状态未知')}
          description={t('keycloak.serverOfflineDesc', '无法连接到 Keycloak 服务器，请检查服务是否正常运行')}
          type="warning"
          showIcon
          style={{ marginBottom: 24 }}
        />
      )}

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title={t('keycloak.totalUsers', '用户总数')}
              value={statistics.totalUsers}
              prefix={<UserOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title={t('keycloak.activeUsers', '活跃用户')}
              value={statistics.activeUsers}
              prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title={t('keycloak.clients', 'OAuth 客户端')}
              value={statistics.totalClients}
              prefix={<ApiOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title={t('keycloak.groups', '用户组')}
              value={statistics.totalGroups}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* Realm 选择器 */}
      <Card style={{ marginBottom: 24 }}>
        <Space>
          <Text strong>{t('keycloak.selectRealm', '选择 Realm')}:</Text>
          <Select
            value={selectedRealm}
            onChange={setSelectedRealm}
            style={{ width: 200 }}
          >
            {realms.map(realm => (
              <Option key={realm.realm || realm.id} value={realm.realm || realm.id}>
                <GlobalOutlined style={{ marginRight: 8 }} />
                {realm.realm || realm.displayName || realm.id}
              </Option>
            ))}
            {realms.length === 0 && (
              <Option value="ai-infra">
                <GlobalOutlined style={{ marginRight: 8 }} />
                ai-infra
              </Option>
            )}
          </Select>
        </Space>
      </Card>

      {/* 标签页内容 */}
      <Card>
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane 
            tab={<span><UserOutlined /> {t('keycloak.users', '用户管理')}</span>} 
            key="users"
          >
            <Space style={{ marginBottom: 16 }}>
              <Button 
                type="primary" 
                icon={<PlusOutlined />}
                onClick={() => setAddUserModalVisible(true)}
              >
                {t('keycloak.addUser', '添加用户')}
              </Button>
            </Space>
            <Table
              columns={userColumns}
              dataSource={users}
              rowKey="id"
              loading={loading}
              pagination={{ pageSize: 10 }}
            />
          </TabPane>

          <TabPane 
            tab={<span><ApiOutlined /> {t('keycloak.clients', '客户端')}</span>} 
            key="clients"
          >
            <Table
              columns={clientColumns}
              dataSource={clients}
              rowKey="id"
              loading={loading}
              pagination={{ pageSize: 10 }}
            />
          </TabPane>

          <TabPane 
            tab={<span><TeamOutlined /> {t('keycloak.groups', '用户组')}</span>} 
            key="groups"
          >
            <Table
              columns={groupColumns}
              dataSource={groups}
              rowKey="id"
              loading={loading}
              pagination={{ pageSize: 10 }}
            />
          </TabPane>

          <TabPane 
            tab={<span><SettingOutlined /> {t('keycloak.settings', '设置')}</span>} 
            key="settings"
          >
            <Descriptions bordered column={2}>
              <Descriptions.Item label={t('keycloak.currentRealm', '当前 Realm')}>
                {selectedRealm}
              </Descriptions.Item>
              <Descriptions.Item label={t('keycloak.serverVersion', '服务器版本')}>
                {serverInfo?.systemInfo?.version || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('keycloak.serverTime', '服务器时间')}>
                {serverInfo?.systemInfo?.serverTime || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('keycloak.uptime', '运行时间')}>
                {serverInfo?.systemInfo?.uptime || '-'}
              </Descriptions.Item>
            </Descriptions>
            <Divider />
            <Space direction="vertical" style={{ width: '100%' }}>
              <Button 
                type="primary" 
                icon={<ExportOutlined />}
                onClick={openKeycloakConsole}
              >
                {t('keycloak.openAdminConsole', '打开 Keycloak 管理控制台')}
              </Button>
              <Text type="secondary">
                {t('keycloak.consoleHint', '在 Keycloak 管理控制台中可以进行更多高级配置')}
              </Text>
            </Space>
          </TabPane>
        </Tabs>
      </Card>

      {/* 添加用户弹窗 */}
      <Modal
        title={t('keycloak.addUser', '添加用户')}
        open={addUserModalVisible}
        onCancel={() => { setAddUserModalVisible(false); form.resetFields(); }}
        footer={null}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateUser}
        >
          <Form.Item
            name="username"
            label={t('keycloak.username', '用户名')}
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input prefix={<UserOutlined />} placeholder="用户名" />
          </Form.Item>
          <Form.Item
            name="email"
            label={t('keycloak.email', '邮箱')}
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '请输入有效的邮箱地址' }
            ]}
          >
            <Input placeholder="user@example.com" />
          </Form.Item>
          <Form.Item
            name="firstName"
            label={t('keycloak.firstName', '名')}
          >
            <Input placeholder="名" />
          </Form.Item>
          <Form.Item
            name="lastName"
            label={t('keycloak.lastName', '姓')}
          >
            <Input placeholder="姓" />
          </Form.Item>
          <Form.Item
            name="enabled"
            label={t('keycloak.enabled', '启用')}
            valuePropName="checked"
            initialValue={true}
          >
            <Switch />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('common.create', '创建')}
              </Button>
              <Button onClick={() => { setAddUserModalVisible(false); form.resetFields(); }}>
                {t('common.cancel', '取消')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 用户详情抽屉 */}
      <Drawer
        title={t('keycloak.userDetails', '用户详情')}
        placement="right"
        width={500}
        open={detailDrawerVisible}
        onClose={() => { setDetailDrawerVisible(false); setSelectedUser(null); }}
      >
        {selectedUser && (
          <Descriptions bordered column={1}>
            <Descriptions.Item label={t('keycloak.userId', '用户ID')}>
              <Text copyable>{selectedUser.id}</Text>
            </Descriptions.Item>
            <Descriptions.Item label={t('keycloak.username', '用户名')}>
              {selectedUser.username}
            </Descriptions.Item>
            <Descriptions.Item label={t('keycloak.email', '邮箱')}>
              {selectedUser.email || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('keycloak.firstName', '名')}>
              {selectedUser.firstName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('keycloak.lastName', '姓')}>
              {selectedUser.lastName || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('keycloak.status', '状态')}>
              <Tag color={selectedUser.enabled ? 'success' : 'error'}>
                {selectedUser.enabled ? '启用' : '禁用'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('keycloak.emailVerified', '邮箱已验证')}>
              {selectedUser.emailVerified ? '是' : '否'}
            </Descriptions.Item>
            <Descriptions.Item label={t('keycloak.createdAt', '创建时间')}>
              {selectedUser.createdTimestamp ? new Date(selectedUser.createdTimestamp).toLocaleString() : '-'}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Drawer>
    </div>
  );
};

export default KeycloakManagement;
