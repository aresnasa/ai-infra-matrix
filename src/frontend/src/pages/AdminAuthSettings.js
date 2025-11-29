import React, { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Input,
  Button,
  Switch,
  Select,
  message,
  Alert,
  Divider,
  Space,
  Typography,
  Spin,
  Tooltip,
  Radio,
  Row,
  Col,
  Modal
} from 'antd';
import {
  SafetyOutlined,
  ExperimentOutlined,
  SaveOutlined,
  InfoCircleOutlined,
  UserOutlined,
  LockOutlined,
  SettingOutlined,
  SyncOutlined
} from '@ant-design/icons';
import { adminAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TextArea } = Input;

const AdminAuthSettings = () => {
  const { t } = useI18n();
  const [form] = Form.useForm();
  const [ldapForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [authMode, setAuthMode] = useState('local');
  const [ldapConfig, setLdapConfig] = useState(null);
  const [testResult, setTestResult] = useState(null);
  const [testModalVisible, setTestModalVisible] = useState(false);
  const [syncResult, setSyncResult] = useState(null);
  const [syncModalVisible, setSyncModalVisible] = useState(false);
  const [syncHistory, setSyncHistory] = useState([]);

  useEffect(() => {
    loadAuthSettings();
    loadSyncHistory();
  }, []);

  const loadAuthSettings = async () => {
    setLoading(true);
    try {
      // 加载当前认证模式和LDAP配置
      const ldapResponse = await adminAPI.getLDAPConfig();
      setLdapConfig(ldapResponse.data);
      setAuthMode(ldapResponse.data.enabled ? 'ldap' : 'local');
      ldapForm.setFieldsValue(ldapResponse.data);
    } catch (error) {
      if (error.response?.status !== 404) {
        message.error(t('admin.loadAuthSettingsFailed'));
      }
    } finally {
      setLoading(false);
    }
  };

  const handleAuthModeChange = (e) => {
    setAuthMode(e.target.value);
  };

  const handleSaveSettings = async () => {
    setSaving(true);
    try {
      if (authMode === 'ldap') {
        // 验证LDAP表单
        const ldapValues = await ldapForm.validateFields();
        
        // 保存LDAP配置并启用
        await adminAPI.updateLDAPConfig({
          ...ldapValues,
          enabled: true
        });
        
        message.success(t('admin.ldapConfigSaved'));
      } else {
        // 禁用LDAP，使用本地认证
        if (ldapConfig) {
          await adminAPI.updateLDAPConfig({
            ...ldapConfig,
            enabled: false
          });
        }
        
        message.success(t('admin.switchToLocalAuth'));
      }
      
      await loadAuthSettings();
    } catch (error) {
      message.error(t('admin.saveAuthSettingsFailed'));
    } finally {
      setSaving(false);
    }
  };

  const handleTestLDAP = async () => {
    setTesting(true);
    setTestResult(null);
    
    try {
      const values = await ldapForm.validateFields();
      const response = await adminAPI.testLDAPConnection(values);
      
      setTestResult({
        success: true,
        message: response.data.message || t('admin.testSuccess'),
        details: response.data
      });
    } catch (error) {
      setTestResult({
        success: false,
        message: error.response?.data?.error || t('admin.testFailed'),
        details: error.response?.data
      });
    } finally {
      setTesting(false);
    }
  };

  const showTestModal = () => {
    setTestModalVisible(true);
    setTestResult(null);
  };

  const handleSyncLDAP = async () => {
    if (!ldapConfig || !ldapConfig.enabled) {
      message.error(t('admin.enableLdapFirst'));
      return;
    }

    setSyncing(true);
    setSyncResult(null);
    
    try {
      const response = await adminAPI.syncLDAPUsers();
      const syncId = response.data.sync_id;
      
      // 显示同步模态框
      setSyncModalVisible(true);
      setSyncResult({
        syncId: syncId,
        status: 'running',
        message: t('admin.preparingSync'),
        progress: 0
      });

      // 轮询检查同步状态
      const checkSyncStatus = async () => {
        try {
          const statusResponse = await adminAPI.getLDAPSyncStatus(syncId);
          const status = statusResponse.data;
          
          setSyncResult(status);
          
          if (status.status === 'completed' || status.status === 'failed') {
            // 同步完成，刷新同步历史
            loadSyncHistory();
            clearInterval(statusInterval);
          }
        } catch (error) {
          console.error('Failed to check sync status:', error);
          clearInterval(statusInterval);
        }
      };

      // 每2秒检查一次状态
      const statusInterval = setInterval(checkSyncStatus, 2000);
      
      // 初始状态检查
      setTimeout(checkSyncStatus, 1000);
      
      message.success(t('admin.ldapSyncStarted'));
    } catch (error) {
      message.error(t('admin.ldapSyncFailed') + ': ' + (error.response?.data?.error || error.message));
      setSyncResult({
        status: 'failed',
        message: error.response?.data?.error || t('admin.ldapSyncFailed'),
        error: error.message
      });
    } finally {
      setSyncing(false);
    }
  };

  const loadSyncHistory = async () => {
    try {
      const response = await adminAPI.getLDAPSyncHistory(5);
      setSyncHistory(response.data.history || []);
    } catch (error) {
      console.error('Failed to load sync history:', error);
    }
  };

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '400px' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>
        <SettingOutlined style={{ marginRight: '8px' }} />
        {t('admin.authSettings')}
      </Title>
      
      <Alert
        message={t('admin.authModeNote')}
        description={t('admin.authModeNoteDesc')}
        type="info"
        showIcon
        style={{ marginBottom: '24px' }}
      />

      <Card title={t('admin.authModeSelect')} style={{ marginBottom: '24px' }}>
        <Radio.Group value={authMode} onChange={handleAuthModeChange} size="large">
          <Space direction="vertical" size="large">
            <Radio value="local">
              <UserOutlined style={{ marginRight: '8px' }} />
              {t('admin.localAuth')}
              <Paragraph type="secondary" style={{ marginLeft: '24px', marginBottom: 0 }}>
                {t('admin.localAuthDesc')}
              </Paragraph>
            </Radio>
            <Radio value="ldap">
              <LockOutlined style={{ marginRight: '8px' }} />
              {t('admin.ldapAuthMode')}
              <Paragraph type="secondary" style={{ marginLeft: '24px', marginBottom: 0 }}>
                {t('admin.ldapAuthModeDesc')}
              </Paragraph>
            </Radio>
          </Space>
        </Radio.Group>
      </Card>

      {authMode === 'ldap' && (
        <Card 
          title={t('admin.ldapConfiguration')} 
          extra={
            <Space>
              <Button 
                icon={<ExperimentOutlined />} 
                onClick={showTestModal}
                disabled={testing}
              >
                {t('admin.testConnection')}
              </Button>
              {ldapConfig?.enabled && (
                <Button 
                  type="primary"
                  icon={<UserOutlined />} 
                  onClick={handleSyncLDAP}
                  loading={syncing}
                  disabled={testing || saving}
                >
                  {t('admin.syncUsers')}
                </Button>
              )}
            </Space>
          }
        >
          <Form form={ldapForm} layout="vertical">
            <Row gutter={[16, 0]}>
              <Col span={12}>
                <Form.Item
                  label={
                    <span>
                      {t('admin.ldapServer')}
                      <Tooltip title={t('admin.ldapServerPlaceholder')}>
                        <InfoCircleOutlined style={{ marginLeft: '4px' }} />
                      </Tooltip>
                    </span>
                  }
                  name="server"
                  rules={[
                    { required: true, message: t('admin.pleaseCompleteLdapForm') },
                    { pattern: /^ldaps?:\/\/.+/, message: t('admin.pleaseCompleteLdapForm') }
                  ]}
                >
                  <Input placeholder="ldap://ldap.company.com:389" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label={t('admin.port')}
                  name="port"
                  rules={[
                    { required: true, message: t('admin.pleaseCompleteLdapForm') },
                    { type: 'number', min: 1, max: 65535, message: t('admin.pleaseCompleteLdapForm') }
                  ]}
                >
                  <Input type="number" placeholder="389" />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={[16, 0]}>
              <Col span={12}>
                <Form.Item
                  label={
                    <span>
                      {t('admin.baseDn')}
                      <Tooltip title={t('admin.baseDnTooltip')}>
                        <InfoCircleOutlined style={{ marginLeft: '4px' }} />
                      </Tooltip>
                    </span>
                  }
                  name="base_dn"
                  rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
                >
                  <Input placeholder="dc=company,dc=com" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label={
                    <span>
                      {t('admin.userDnTemplate')}
                      <Tooltip title={t('admin.userDnTemplateTooltip')}>
                        <InfoCircleOutlined style={{ marginLeft: '4px' }} />
                      </Tooltip>
                    </span>
                  }
                  name="user_dn_template"
                  rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
                >
                  <Input placeholder="uid={username},ou=users,dc=company,dc=com" />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={[16, 0]}>
              <Col span={12}>
                <Form.Item
                  label={t('admin.adminDn')}
                  name="bind_dn"
                  rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
                >
                  <Input placeholder="cn=admin,dc=company,dc=com" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label={t('admin.adminPassword')}
                  name="bind_password"
                  rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
                >
                  <Input.Password placeholder={t('admin.adminPassword')} />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={[16, 0]}>
              <Col span={8}>
                <Form.Item
                  label={t('admin.usernameAttr')}
                  name="username_attribute"
                  initialValue="uid"
                  rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
                >
                  <Input placeholder="uid" />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item
                  label={t('admin.emailAttr')}
                  name="email_attribute"
                  initialValue="mail"
                  rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
                >
                  <Input placeholder="mail" />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item
                  label={t('admin.displayNameAttr')}
                  name="display_name_attribute"
                  initialValue="cn"
                  rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
                >
                  <Input placeholder="cn" />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={[16, 0]}>
              <Col span={12}>
                <Form.Item
                  label={
                    <span>
                      {t('admin.userSearchFilter')}
                      <Tooltip title={t('admin.userSearchFilterTooltip')}>
                        <InfoCircleOutlined style={{ marginLeft: '4px' }} />
                      </Tooltip>
                    </span>
                  }
                  name="user_filter"
                  initialValue="(&(objectClass=inetOrgPerson)(uid={username}))"
                  rules={[{ required: true, message: t('admin.pleaseCompleteLdapForm') }]}
                >
                  <Input placeholder="(&(objectClass=inetOrgPerson)(uid={username}))" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label={t('admin.userSearchBase')}
                  name="user_search_base"
                  initialValue="ou=users"
                >
                  <Input placeholder="ou=users" />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              label={t('admin.connectionTimeout')}
              name="timeout"
              initialValue={30}
              rules={[
                { required: true, message: t('admin.pleaseCompleteLdapForm') },
                { type: 'number', min: 1, max: 300, message: t('admin.pleaseCompleteLdapForm') }
              ]}
            >
              <Input type="number" placeholder="30" />
            </Form.Item>

            <Form.Item label={t('admin.enableTls')} name="enable_tls" valuePropName="checked">
              <Switch />
            </Form.Item>

            <Form.Item label={t('admin.skipTlsVerify')} name="skip_tls_verify" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Form>
        </Card>
      )}

      {authMode === 'ldap' && ldapConfig?.enabled && (
        <Card 
          title={t('admin.syncHistory')} 
          style={{ marginTop: '24px' }}
          extra={
            <Button 
              size="small"
              onClick={loadSyncHistory}
            >
              {t('admin.refresh')}
            </Button>
          }
        >
          {Array.isArray(syncHistory) && syncHistory.length > 0 ? (
            <div>
              {syncHistory.map((sync, index) => (
                <div key={sync.id || index} style={{ 
                  padding: '12px', 
                  border: '1px solid #f0f0f0', 
                  borderRadius: '6px',
                  marginBottom: '8px',
                  backgroundColor: sync.status === 'completed' ? '#f6ffed' : 
                                   sync.status === 'failed' ? '#fff2f0' : '#f0f0f0'
                }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                      <Text strong>
                        {sync.status === 'completed' && `✅ ${t('admin.syncCompleted')}`}
                        {sync.status === 'failed' && `❌ ${t('admin.syncFailed')}`}
                        {sync.status === 'running' && `🔄 ${t('admin.syncing')}`}
                      </Text>
                      <div style={{ marginTop: '4px' }}>
                        <Text type="secondary" style={{ fontSize: '12px' }}>
                          {new Date(sync.start_time).toLocaleString()}
                        </Text>
                        {sync.duration && (
                          <Text type="secondary" style={{ fontSize: '12px', marginLeft: '12px' }}>
                            {t('admin.duration')}: {Math.round(sync.duration / 1000000000)}{t('admin.seconds')}
                          </Text>
                        )}
                      </div>
                    </div>
                    {sync.result && (
                      <div style={{ textAlign: 'right' }}>
                        <div style={{ fontSize: '12px' }}>
                          <Text type="secondary">{t('admin.users')}: </Text>
                          <Text>{sync.result.users_created + sync.result.users_updated}</Text>
                          <Text type="secondary" style={{ marginLeft: '8px' }}>{t('admin.groups')}: </Text>
                          <Text>{sync.result.groups_created + sync.result.groups_updated}</Text>
                        </div>
                      </div>
                    )}
                  </div>
                  {sync.message && (
                    <div style={{ marginTop: '8px' }}>
                      <Text style={{ fontSize: '12px' }}>{sync.message}</Text>
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div style={{ textAlign: 'center', padding: '24px' }}>
              <Text type="secondary">{t('admin.noSyncRecord')}</Text>
            </div>
          )}
        </Card>
      )}

      <Card style={{ marginTop: '24px' }}>
        <Space>
          <Button 
            type="primary" 
            icon={<SaveOutlined />} 
            onClick={handleSaveSettings}
            loading={saving}
            size="large"
          >
            {t('admin.saveSettings')}
          </Button>
          <Button onClick={loadAuthSettings} disabled={saving}>
            {t('admin.reset')}
          </Button>
        </Space>
      </Card>

      {/* LDAP测试连接模态框 */}
      <Modal
        title={t('admin.testConnection')}
        open={testModalVisible}
        onCancel={() => setTestModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setTestModalVisible(false)}>
            {t('common.close')}
          </Button>,
          <Button 
            key="test" 
            type="primary" 
            icon={<ExperimentOutlined />}
            onClick={handleTestLDAP}
            loading={testing}
          >
            {t('admin.testConnection')}
          </Button>
        ]}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Text>{t('admin.testConnectionBtn')}</Text>
          
          {testing && (
            <div style={{ textAlign: 'center', padding: '20px' }}>
              <Spin />
              <div style={{ marginTop: '8px' }}>{t('admin.testingConnection')}</div>
            </div>
          )}
          
          {testResult && (
            <Alert
              type={testResult.success ? 'success' : 'error'}
              message={testResult.message}
              description={testResult.details && (
                <div>
                  {testResult.details.server_info && (
                    <div>{t('admin.serverInfo')}: {testResult.details.server_info}</div>
                  )}
                  {testResult.details.bind_result && (
                    <div>{t('admin.bindResult')}: {testResult.details.bind_result}</div>
                  )}
                  {testResult.details.search_result && (
                    <div>{t('admin.searchResult')}: {testResult.details.search_result}</div>
                  )}
                </div>
              )}
              showIcon
            />
          )}
        </Space>
      </Modal>

      {/* LDAP同步状态模态框 */}
      <Modal
        title={t('admin.ldapSyncTitle')}
        open={syncModalVisible}
        onCancel={() => setSyncModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setSyncModalVisible(false)}>
            {t('common.close')}
          </Button>
        ]}
        width={600}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          {syncResult && (
            <div>
              {syncResult.status === 'running' && (
                <div style={{ textAlign: 'center', padding: '20px' }}>
                  <Spin size="large" />
                  <div style={{ marginTop: '16px' }}>
                    <Text strong>{syncResult.message}</Text>
                  </div>
                  {syncResult.progress > 0 && (
                    <div style={{ marginTop: '8px' }}>
                      <Text type="secondary">{t('admin.progress')}: {syncResult.progress.toFixed(1)}%</Text>
                    </div>
                  )}
                </div>
              )}
              
              {syncResult.status === 'completed' && (
                <Alert
                  type="success"
                  message={t('admin.syncCompleted')}
                  description={
                    <div>
                      <div>{t('admin.syncId')}: {syncResult.id}</div>
                      <div>{t('admin.startTime')}: {new Date(syncResult.start_time).toLocaleString()}</div>
                      {syncResult.end_time && (
                        <div>{t('admin.endTime')}: {new Date(syncResult.end_time).toLocaleString()}</div>
                      )}
                      {syncResult.duration && (
                        <div>{t('admin.duration')}: {Math.round(syncResult.duration / 1000000000)}{t('admin.seconds')}</div>
                      )}
                      {syncResult.result && (
                        <div style={{ marginTop: '12px' }}>
                          <Text strong>{t('admin.syncResult')}:</Text>
                          <ul style={{ marginTop: '8px', paddingLeft: '20px' }}>
                            <li>{t('admin.usersCreated')}: {syncResult.result.users_created}</li>
                            <li>{t('admin.usersUpdated')}: {syncResult.result.users_updated}</li>
                            <li>{t('admin.groupsCreated')}: {syncResult.result.groups_created}</li>
                            <li>{t('admin.groupsUpdated')}: {syncResult.result.groups_updated}</li>
                            <li>{t('admin.rolesAssigned')}: {syncResult.result.roles_assigned}</li>
                            <li>{t('admin.totalUsersCount')}: {syncResult.result.total_users}</li>
                            <li>{t('admin.totalGroupsCount')}: {syncResult.result.total_groups}</li>
                          </ul>
                          {syncResult.result.errors && Array.isArray(syncResult.result.errors) && syncResult.result.errors.length > 0 && (
                            <div style={{ marginTop: '12px' }}>
                              <Text type="danger">{t('admin.errorInfo')}:</Text>
                              <ul style={{ marginTop: '8px', paddingLeft: '20px' }}>
                                {syncResult.result.errors.map((error, index) => (
                                  <li key={index} style={{ color: '#ff4d4f' }}>{error}</li>
                                ))}
                              </ul>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  }
                  showIcon
                />
              )}
              
              {syncResult.status === 'failed' && (
                <Alert
                  type="error"
                  message={t('admin.syncFailed')}
                  description={
                    <div>
                      <div>{t('admin.syncId')}: {syncResult.id}</div>
                      <div>{t('admin.startTime')}: {new Date(syncResult.start_time).toLocaleString()}</div>
                      {syncResult.end_time && (
                        <div>{t('admin.endTime')}: {new Date(syncResult.end_time).toLocaleString()}</div>
                      )}
                      <div style={{ marginTop: '12px' }}>
                        <Text type="danger">{t('admin.errorInfo')}: {syncResult.error || syncResult.message}</Text>
                      </div>
                    </div>
                  }
                  showIcon
                />
              )}
            </div>
          )}
        </Space>
      </Modal>
    </div>
  );
};

export default AdminAuthSettings;
