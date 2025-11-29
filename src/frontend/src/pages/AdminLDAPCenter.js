import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Form,
  Input,
  Button,
  Switch,
  Select,
  message,
  Alert,
  Space,
  Typography,
  Spin,
  Tooltip,
  Row,
  Col,
  Tabs,
  Table,
  Tag,
  Modal,
  Descriptions,
  Statistic,
  Progress,
  Empty,
  Timeline,
  Badge,
  Divider,
  Popconfirm
} from 'antd';
import {
  SafetyOutlined,
  ExperimentOutlined,
  SaveOutlined,
  InfoCircleOutlined,
  BookOutlined,
  UserOutlined,
  TeamOutlined,
  SyncOutlined,
  SettingOutlined,
  LockOutlined,
  UnlockOutlined,
  EyeOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined
} from '@ant-design/icons';
import { adminAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TextArea } = Input;
const { TabPane } = Tabs;

/**
 * 统一的LDAP管理中心
 * 整合了LDAP配置、用户管理、同步等功能
 * 采用只读LDAP策略，保证数据安全
 */
const AdminLDAPCenter = () => {
  const { t } = useI18n();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [saving, setSaving] = useState(false);
  
  // LDAP配置相关状态
  const [config, setConfig] = useState(null);
  const [testResult, setTestResult] = useState(null);
  const [ldapEnabled, setLdapEnabled] = useState(false);
  
  // 用户管理相关状态
  const [users, setUsers] = useState([]);
  const [ldapUsers, setLdapUsers] = useState([]);
  const [syncStatus, setSyncStatus] = useState(null);
  const [syncHistory, setSyncHistory] = useState([]);
  const [selectedUser, setSelectedUser] = useState(null);
  const [userModalVisible, setUserModalVisible] = useState(false);
  
  // 活动Tab
  const [activeTab, setActiveTab] = useState('config');

  useEffect(() => {
    loadLDAPConfig();
    loadUsers();
    loadSyncHistory();
  }, []);

  // 加载LDAP配置
  const loadLDAPConfig = async () => {
    setLoading(true);
    try {
      const response = await adminAPI.getLDAPConfig();
      const ldapConfig = response.data;
      setConfig(ldapConfig);
      form.setFieldsValue(ldapConfig);
      setLdapEnabled(ldapConfig.enabled || ldapConfig.is_enabled || false);
    } catch (error) {
      if (error.response?.status !== 404) {
        message.error(t('admin.ldapConfigSaveFailed'));
      }
    } finally {
      setLoading(false);
    }
  };

  // 加载用户列表
  const loadUsers = async () => {
    try {
      const [localResponse, ldapResponse] = await Promise.all([
        adminAPI.getUsers(),
        adminAPI.getLDAPUsers().catch(() => ({ data: { users: [] } }))
      ]);
      
      setUsers(localResponse.data.users || localResponse.data || []);
      setLdapUsers(ldapResponse.data.users || []);
    } catch (error) {
      message.error(t('admin.loadUsersFailed'));
    }
  };

  // 加载同步历史
  const loadSyncHistory = async () => {
    try {
      const response = await adminAPI.getLDAPSyncHistory();
      setSyncHistory(response.data.history || []);
    } catch (error) {
      // 同步历史可能不存在，忽略错误
    }
  };

  // 保存LDAP配置
  const handleSave = async (values) => {
    setSaving(true);
    try {
      await adminAPI.updateLDAPConfig(values);
      message.success(t('admin.ldapConfigSaveSuccess'));
      await loadLDAPConfig();
      setTestResult(null); // 清除之前的测试结果
    } catch (error) {
      message.error(error.response?.data?.message || t('admin.ldapConfigSaveFailed'));
    } finally {
      setSaving(false);
    }
  };

  // 测试LDAP连接
  const handleTest = async () => {
    try {
      const values = await form.validateFields();
      if (!values.enabled && !values.is_enabled) {
        message.warning(t('admin.enableLdapAuth'));
        return;
      }
      
      setTesting(true);
      const response = await adminAPI.testLDAPConnection(values);
      setTestResult({
        success: true,
        message: response.data.message || t('admin.ldapTestSuccess')
      });
      message.success(t('admin.ldapTestSuccess'));
    } catch (error) {
      if (error.errorFields) {
        message.error(t('admin.pleaseCompleteLdapForm'));
        return;
      }
      setTestResult({
        success: false,
        message: error.response?.data?.message || error.response?.data?.error || t('admin.ldapTestFailed')
      });
      message.error(t('admin.ldapTestFailed'));
    } finally {
      setTesting(false);
    }
  };

  // 同步LDAP用户
  const handleSyncUsers = async () => {
    if (!ldapEnabled) {
      message.warning(t('admin.enableLdapAuth'));
      return;
    }

    setSyncing(true);
    try {
      const response = await adminAPI.syncLDAPUsers();
      message.success(t('admin.ldapSyncComplete'));
      setSyncStatus(response.data);
      await loadUsers();
      await loadSyncHistory();
    } catch (error) {
      message.error(error.response?.data?.message || t('admin.ldapSyncError'));
    } finally {
      setSyncing(false);
    }
  };

  // 切换用户启用状态
  const toggleUserStatus = async (userId, currentStatus) => {
    try {
      await adminAPI.toggleUserStatus(userId, !currentStatus);
      message.success(t('admin.toggleUserSuccess', { action: currentStatus ? t('admin.disable') : t('admin.enable') }));
      await loadUsers();
    } catch (error) {
      message.error(t('admin.toggleUserFailed'));
    }
  };

  // 重置表单
  const handleReset = () => {
    if (config) {
      form.setFieldsValue(config);
      setLdapEnabled(config.enabled || config.is_enabled || false);
    } else {
      form.resetFields();
      setLdapEnabled(false);
    }
    setTestResult(null);
  };

  // LDAP开关切换
  const handleLdapToggle = (enabled) => {
    setLdapEnabled(enabled);
    if (!enabled) {
      setTestResult(null);
    }
  };

  // 查看用户详情
  const showUserDetails = (user) => {
    setSelectedUser(user);
    setUserModalVisible(true);
  };

  // 用户表格列定义
  const userColumns = [
    {
      title: t('common.username'),
      dataIndex: 'username',
      key: 'username',
      render: (text, record) => (
        <Space>
          <Text strong>{text}</Text>
          {record.auth_source === 'ldap' && (
            <Tag color="blue">LDAP</Tag>
          )}
          {!record.is_active && (
            <Tag color="red">{t('admin.disable')}</Tag>
          )}
        </Space>
      )
    },
    {
      title: t('common.email'),
      dataIndex: 'email',
      key: 'email'
    },
    {
      title: t('admin.name'),
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: t('admin.authSource'),
      dataIndex: 'auth_source',
      key: 'auth_source',
      render: (source) => (
        <Tag color={source === 'ldap' ? 'blue' : 'green'}>
          {source === 'ldap' ? 'LDAP' : t('admin.local')}
        </Tag>
      )
    },
    {
      title: t('common.status'),
      dataIndex: 'is_active',
      key: 'is_active',
      render: (active) => (
        <Badge 
          status={active ? 'success' : 'error'} 
          text={active ? t('admin.enable') : t('admin.disable')} 
        />
      )
    },
    {
      title: t('admin.lastLogin'),
      dataIndex: 'last_login',
      key: 'last_login',
      render: (lastLogin) => lastLogin ? new Date(lastLogin).toLocaleString() : t('admin.neverLoggedIn')
    },
    {
      title: t('common.action'),
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button
            size="small"
            icon={<EyeOutlined />}
            onClick={() => showUserDetails(record)}
          >
            {t('admin.details')}
          </Button>
          {record.auth_source === 'ldap' && (
            <Popconfirm
              title={t('admin.confirmToggleUser', { action: record.is_active ? t('admin.disable') : t('admin.enable') })}
              description={t('admin.toggleUserNote')}
              onConfirm={() => toggleUserStatus(record.id, record.is_active)}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
            >
              <Button
                size="small"
                icon={record.is_active ? <LockOutlined /> : <UnlockOutlined />}
                danger={record.is_active}
              >
                {record.is_active ? t('admin.disable') : t('admin.enable')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      )
    }
  ];

  // 渲染LDAP配置Tab
  const renderConfigTab = () => (
    <div>
      {/* 安全提示 */}
      <Alert
        message={t('admin.readOnlyMode')}
        description={
          <div>
            <p>✅ <strong>{t('admin.readOnlyModeDesc1')}</strong></p>
            <p>🔒 <strong>{t('admin.readOnlyModeDesc2')}</strong></p>
            <p>📋 <strong>{t('admin.readOnlyModeDesc3')}</strong></p>
          </div>
        }
        type="info"
        showIcon
        style={{ marginBottom: 24 }}
      />

      {/* 测试结果显示 */}
      {testResult && (
        <Alert
          message={testResult.success ? t('admin.ldapTestSuccess') : t('admin.ldapTestFailed')}
          description={testResult.message}
          type={testResult.success ? 'success' : 'error'}
          showIcon
          closable
          style={{ marginBottom: 24 }}
        />
      )}

      <Form
        form={form}
        layout="vertical"
        onFinish={handleSave}
        initialValues={{
          enabled: false,
          is_enabled: false,
          port: 389,
          use_ssl: false,
          skip_verify: false,
          user_filter: "(objectClass=person)",
          username_attr: "uid",
          name_attr: "cn",
          email_attr: "mail"
        }}
      >
        {/* 基本配置 */}
        <Card size="small" title={t('admin.basicConfig')} style={{ marginBottom: 16 }}>
          <Form.Item
            name={['enabled', 'is_enabled']}
            label={t('admin.enableLdapAuthLabel')}
            valuePropName="checked"
            extra={ldapEnabled ? t('admin.ldapEnabled') : t('admin.ldapDisabled')}
          >
            <Switch 
              checked={ldapEnabled}
              onChange={handleLdapToggle}
              checkedChildren={t('admin.enabled')} 
              unCheckedChildren={t('admin.disabledLabel')}
            />
          </Form.Item>

          <Row gutter={16}>
            <Col span={16}>
              <Form.Item
                name="server"
                label={t('admin.ldapServerLabel')}
                rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
              >
                <Input placeholder="ldap.company.com" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="port"
                label={t('admin.port')}
                rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
              >
                <Input placeholder="389" type="number" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name={['use_ssl', 'useSSL']}
                label={t('admin.useSSLTLS')}
                valuePropName="checked"
              >
                <Switch checkedChildren={t('admin.enabled')} unCheckedChildren={t('admin.disabledLabel')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name={['skip_verify', 'skipVerify']}
                label={t('admin.skipCertVerify')}
                valuePropName="checked"
              >
                <Switch checkedChildren={t('admin.skip')} unCheckedChildren={t('admin.verify')} />
              </Form.Item>
            </Col>
          </Row>
        </Card>

        {/* 绑定配置 */}
        <Card size="small" title={t('admin.bindAuth')} style={{ marginBottom: 16 }}>
          <Form.Item
            name={['bind_dn', 'bindDN']}
            label={t('admin.bindDn')}
            rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
          >
            <Input placeholder="cn=admin,dc=company,dc=com" />
          </Form.Item>

          <Form.Item
            name={['bind_password', 'bindPassword']}
            label={t('admin.bindPassword')}
            rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
          >
            <Input.Password placeholder={t('admin.adminPassword')} />
          </Form.Item>

          <Form.Item
            name={['base_dn', 'baseDN']}
            label={t('admin.baseDn')}
            rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
          >
            <Input placeholder="dc=company,dc=com" />
          </Form.Item>
        </Card>

        {/* 用户配置 */}
        <Card size="small" title={t('admin.userAttrMappingLabel')} style={{ marginBottom: 16 }}>
          <Form.Item
            name={['user_filter', 'userFilter']}
            label={t('admin.userFilter')}
            extra={t('admin.userFilterHint')}
          >
            <Input placeholder="(uid={username})" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name={['username_attr', 'usernameAttr']}
                label={t('admin.usernameAttr')}
              >
                <Input placeholder="uid" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name={['name_attr', 'nameAttr']}
                label={t('admin.nameAttr')}
              >
                <Input placeholder="cn" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name={['email_attr', 'emailAttr']}
                label={t('admin.emailAttr')}
              >
                <Input placeholder="mail" />
              </Form.Item>
            </Col>
          </Row>
        </Card>

        {/* 操作按钮 */}
        <Card size="small">
          <Space>
            <Button 
              type="primary" 
              icon={<SaveOutlined />}
              htmlType="submit"
              loading={saving}
              disabled={!ldapEnabled}
            >
              {t('admin.saveConfig')}
            </Button>
            
            <Button 
              icon={<ExperimentOutlined />} 
              onClick={handleTest}
              loading={testing}
              disabled={!ldapEnabled}
            >
              {t('admin.testConnectionBtn')}
            </Button>
            
            <Button 
              icon={<ReloadOutlined />} 
              onClick={handleReset}
            >
              {t('admin.reset')}
            </Button>
          </Space>
        </Card>
      </Form>
    </div>
  );

  // 渲染用户管理Tab
  const renderUsersTab = () => (
    <div>
      {/* 用户统计 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title={t('admin.totalCount', { count: '' })}
              value={users.length}
              prefix={<UserOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title={t('admin.ldapUsers')}
              value={users.filter(u => u.auth_source === 'ldap').length}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title={t('admin.enabledUsers')}
              value={users.filter(u => u.is_active).length}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title={t('admin.disabledUsers')}
              value={users.filter(u => !u.is_active).length}
              prefix={<LockOutlined />}
              valueStyle={{ color: '#cf1322' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 操作区 */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space>
          <Button
            type="primary"
            icon={<SyncOutlined />}
            onClick={handleSyncUsers}
            loading={syncing}
            disabled={!ldapEnabled}
          >
            {t('admin.syncLdapUsers')}
          </Button>
          
          <Button
            icon={<ReloadOutlined />}
            onClick={loadUsers}
          >
            {t('admin.refreshList')}
          </Button>
        </Space>
      </Card>

      {/* 用户列表 */}
      <Card title={t('admin.userManagementTab')}>
        <Table
          columns={userColumns}
          dataSource={users}
          rowKey="id"
          loading={loading}
          pagination={{
            total: users.length,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => t('admin.totalCount', { count: total })
          }}
        />
      </Card>
    </div>
  );

  // 渲染同步历史Tab
  const renderSyncTab = () => (
    <div>
      {syncStatus && (
        <Alert
          message={t('admin.latestSyncResult')}
          description={
            <div>
              <p>{t('admin.syncTime')}: {new Date(syncStatus.start_time).toLocaleString()}</p>
              <p>{t('admin.processedUsers')}: {syncStatus.total_users} | {t('admin.newUsers')}: {syncStatus.created_users} | {t('admin.updatedUsers')}: {syncStatus.updated_users}</p>
            </div>
          }
          type="success"
          style={{ marginBottom: 16 }}
        />
      )}

      <Card title={t('admin.syncHistoryTab')}>
        {Array.isArray(syncHistory) && syncHistory.length > 0 ? (
          <Timeline>
            {syncHistory.map((record, index) => (
              <Timeline.Item
                key={index}
                color={record.status === 'success' ? 'green' : 'red'}
                dot={record.status === 'success' ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}
              >
                <div>
                  <Text strong>{new Date(record.start_time).toLocaleString()}</Text>
                  <br />
                  <Text type="secondary">
                    {t('admin.processed')} {record.total_users} {t('admin.users')}, {t('admin.created')} {record.created_users}, {t('admin.updated')} {record.updated_users}
                  </Text>
                  {record.error_message && (
                    <div>
                      <Text type="danger">{t('admin.error')}: {record.error_message}</Text>
                    </div>
                  )}
                </div>
              </Timeline.Item>
            ))}
          </Timeline>
        ) : (
          <Empty description={t('admin.noSyncHistory')} />
        )}
      </Card>
    </div>
  );

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Title level={2}>
          <SettingOutlined style={{ marginRight: 8 }} />
          {t('admin.ldapCenter')}
        </Title>
        <Paragraph type="secondary">
          {t('admin.ldapCenterDesc')}
        </Paragraph>
      </div>

      <Spin spinning={loading}>
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane 
            tab={
              <span>
                <SettingOutlined />
                {t('admin.ldapConfigTab')}
              </span>
            } 
            key="config"
          >
            {renderConfigTab()}
          </TabPane>
          
          <TabPane 
            tab={
              <span>
                <UserOutlined />
                {t('admin.userManagementTab')}
              </span>
            } 
            key="users"
          >
            {renderUsersTab()}
          </TabPane>
          
          <TabPane 
            tab={
              <span>
                <SyncOutlined />
                {t('admin.syncHistoryTab')}
              </span>
            } 
            key="sync"
          >
            {renderSyncTab()}
          </TabPane>
        </Tabs>
      </Spin>

      {/* 用户详情模态框 */}
      <Modal
        title={t('admin.userDetails')}
        visible={userModalVisible}
        onCancel={() => setUserModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setUserModalVisible(false)}>
            {t('common.close')}
          </Button>
        ]}
        width={600}
      >
        {selectedUser && (
          <Descriptions column={2} bordered>
            <Descriptions.Item label={t('common.username')}>{selectedUser.username}</Descriptions.Item>
            <Descriptions.Item label={t('common.email')}>{selectedUser.email}</Descriptions.Item>
            <Descriptions.Item label={t('admin.name')}>{selectedUser.name || t('admin.none')}</Descriptions.Item>
            <Descriptions.Item label={t('admin.authSource')}>
              <Tag color={selectedUser.auth_source === 'ldap' ? 'blue' : 'green'}>
                {selectedUser.auth_source === 'ldap' ? 'LDAP' : t('admin.local')}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('common.status')}>
              <Badge 
                status={selectedUser.is_active ? 'success' : 'error'} 
                text={selectedUser.is_active ? t('admin.enable') : t('admin.disable')} 
              />
            </Descriptions.Item>
            <Descriptions.Item label={t('admin.ldapDn')}>
              {selectedUser.ldap_dn || t('admin.none')}
            </Descriptions.Item>
            <Descriptions.Item label={t('admin.lastLogin')}>
              {selectedUser.last_login ? new Date(selectedUser.last_login).toLocaleString() : t('admin.neverLoggedIn')}
            </Descriptions.Item>
            <Descriptions.Item label={t('common.createdAt')}>
              {new Date(selectedUser.created_at).toLocaleString()}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default AdminLDAPCenter;