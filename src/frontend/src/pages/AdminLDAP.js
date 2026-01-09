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
  Collapse,
  theme
} from 'antd';
import {
  SafetyOutlined,
  ExperimentOutlined,
  SaveOutlined,
  InfoCircleOutlined,
  BookOutlined
} from '@ant-design/icons';
import { adminAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';

const { Title, Text } = Typography;
const { Option } = Select;
const { TextArea } = Input;
const { useToken } = theme;
const { Panel } = Collapse;

const AdminLDAP = () => {
  const { t } = useI18n();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState(null);
  const [testResult, setTestResult] = useState(null);
  const [ldapEnabled, setLdapEnabled] = useState(false);

  useEffect(() => {
    loadLDAPConfig();
  }, []);

  const loadLDAPConfig = async () => {
    setLoading(true);
    try {
      const response = await adminAPI.getLDAPConfig();
      setConfig(response.data);
      form.setFieldsValue(response.data);
      setLdapEnabled(response.data.enabled || false);
    } catch (error) {
      if (error.response?.status !== 404) {
        message.error(t('admin.ldapConfigSaveFailed'));
      }
    } finally {
      setLoading(false);
    }
  };

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

  const handleTest = async () => {
    try {
      const values = await form.validateFields();
      if (!values.enabled) {
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
        message: error.response?.data?.message || t('admin.ldapTestFailed')
      });
      message.error(t('admin.ldapTestFailed'));
    } finally {
      setTesting(false);
    }
  };

  const handleReset = () => {
    if (config) {
      form.setFieldsValue(config);
      setLdapEnabled(config.enabled || false);
    } else {
      form.resetFields();
      setLdapEnabled(false);
    }
    setTestResult(null);
  };

  const handleLdapToggle = (enabled) => {
    setLdapEnabled(enabled);
    if (!enabled) {
      setTestResult(null); // 禁用LDAP时清除测试结果
    }
  };

  const fillTestConfig = () => {
    const testConfig = {
      enabled: true,
      server: 'openldap',
      port: 389,
      security: 'none',
      timeout: 30,
      bind_dn: 'cn=admin,dc=testcompany,dc=com',
      bind_password: 'admin123',
      base_dn: 'dc=testcompany,dc=com',
      user_filter: '(uid={username})',
      username_attr: 'uid',
      email_attr: 'mail',
      display_name_attr: 'cn',
      admin_group_dn: '',
      group_member_attr: 'member'
    };
    
    form.setFieldsValue(testConfig);
    setLdapEnabled(true);
    message.success(t('admin.testConfigFilled'));
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <div style={{ marginBottom: '24px' }}>
          <Title level={2}>
            <SafetyOutlined style={{ marginRight: '8px', color: '#1890ff' }} />
            {t('admin.ldapConfigManagement')}
          </Title>
          <Text type="secondary">
            {t('admin.ldapConfigManagementDesc')}
          </Text>
        </div>

        {/* 配置样例 */}
        <Collapse 
          ghost 
          style={{ 
            marginBottom: '24px',
            backgroundColor: '#f6f8fa',
            borderRadius: '6px',
            border: '1px solid #e1e4e8'
          }}
        >
          <Panel 
            header={
              <span>
                <BookOutlined style={{ marginRight: '8px', color: '#28a745' }} />
                <strong>{t('admin.configExampleRef')}</strong>
                <Text type="secondary" style={{ marginLeft: '8px' }}>
                  {t('admin.clickToViewExample')}
                </Text>
              </span>
            } 
            key="1"
          >
            <div style={{ 
              backgroundColor: '#fff', 
              padding: '16px', 
              borderRadius: '4px',
              border: '1px dashed #d9d9d9'
            }}>
              <Title level={4} style={{ marginBottom: '16px', color: '#52c41a' }}>
                🔧 {t('admin.testEnvConfig')}
              </Title>
              <div style={{ 
                fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
                fontSize: '13px',
                backgroundColor: '#f8f9fa',
                padding: '12px',
                borderRadius: '4px',
                border: '1px solid #e9ecef',
                marginBottom: '16px'
              }}>
                <div style={{ color: '#6a737d', marginBottom: '8px' }}>/* {t('admin.basicConnConfig')} */</div>
                <div><strong>{t('admin.serverAddress')}:</strong> openldap</div>
                <div><strong>{t('admin.port')}:</strong> 389</div>
                <div><strong>{t('admin.useSSL')}:</strong> false</div>
                <div><strong>{t('admin.startTLS')}:</strong> false</div>
                <div style={{ marginTop: '12px', color: '#6a737d', marginBottom: '8px' }}>/* {t('admin.authInfo')} */</div>
                <div><strong>{t('admin.bindDn')}:</strong> cn=admin,dc=testcompany,dc=com</div>
                <div><strong>{t('admin.bindPassword')}:</strong> admin123</div>
                <div><strong>{t('admin.baseSearchDn')}:</strong> dc=testcompany,dc=com</div>
              </div>
              
              <Alert
                message={t('admin.configTips')}
                description={
                  <div>
                    <p>✅ <strong>{t('admin.testEnvTip')}</strong></p>
                    <p>⚙️ <strong>{t('admin.prodEnvTip')}</strong></p>
                    <p>🔐 <strong>{t('admin.securityTip')}</strong></p>
                  </div>
                }
                type="info"
                showIcon
              />
              
              <div style={{ marginTop: '16px', textAlign: 'center' }}>
                <Button 
                  type="primary" 
                  size="small" 
                  onClick={fillTestConfig}
                  style={{ backgroundColor: '#52c41a', borderColor: '#52c41a' }}
                >
                  🚀 {t('admin.quickFillTestConfig')}
                </Button>
                <Text type="secondary" style={{ marginLeft: '8px', fontSize: '12px' }}>
                  {t('admin.oneClickFill')}
                </Text>
              </div>
            </div>
          </Panel>
        </Collapse>

        {testResult && (
          <Alert
            message={testResult.success ? t('admin.ldapTestSuccess') : t('admin.ldapTestFailed')}
            description={testResult.message}
            type={testResult.success ? 'success' : 'error'}
            showIcon
            closable
            style={{ marginBottom: '24px' }}
          />
        )}

        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
          initialValues={{
            enabled: false,
            port: 389,
            security: 'none',
            timeout: 30
          }}
        >
          <Card size="small" title={t('admin.basicConfig')} style={{ marginBottom: '16px' }}>
            <Form.Item
              name="enabled"
              label={t('admin.enableLdapAuthLabel')}
              valuePropName="checked"
              extra={ldapEnabled ? t('admin.ldapEnabled') : t('admin.ldapDisabled')}
            >
              <Switch 
                checkedChildren={t('admin.enabled')} 
                unCheckedChildren={t('admin.disabledLabel')}
                onChange={handleLdapToggle}
              />
            </Form.Item>

            <Form.Item
              name="server"
              label={
                <span>
                  {t('admin.ldapServer')}
                  <Tooltip title={t('admin.ldapServerPlaceholder')}>
                    <InfoCircleOutlined style={{ marginLeft: 4 }} />
                  </Tooltip>
                </span>
              }
              rules={[
                { 
                  required: ldapEnabled, 
                  message: t('admin.pleaseCompleteLdapForm') 
                }
              ]}
            >
              <Input 
                placeholder={t('admin.serverAddressPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="port"
              label={t('admin.port')}
              rules={[
                { required: ldapEnabled, message: t('admin.pleaseCompleteLdapForm') },
                { type: 'number', min: 1, max: 65535, message: t('admin.pleaseCompleteLdapForm') }
              ]}
            >
              <Input 
                type="number" 
                placeholder={t('admin.portPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="security"
              label={t('admin.securityConnection')}
              rules={[{ required: ldapEnabled, message: t('admin.pleaseCompleteLdapForm') }]}
            >
              <Select disabled={!ldapEnabled}>
                <Option value="none">{t('admin.noEncryption')}</Option>
                <Option value="ssl">SSL/TLS</Option>
                <Option value="starttls">StartTLS</Option>
              </Select>
            </Form.Item>

            <Form.Item
              name="timeout"
              label={t('admin.connectionTimeoutSeconds')}
              rules={[
                { required: ldapEnabled, message: t('admin.pleaseCompleteLdapForm') },
                { type: 'number', min: 1, max: 300, message: t('admin.pleaseCompleteLdapForm') }
              ]}
            >
              <Input 
                type="number" 
                disabled={!ldapEnabled}
              />
            </Form.Item>
          </Card>

          <Card 
            size="small" 
            title={t('admin.authConfig')} 
            style={{ 
              marginBottom: '16px',
              opacity: ldapEnabled ? 1 : 0.6
            }}
          >
            <Form.Item
              name="bind_dn"
              label={
                <span>
                  {t('admin.bindDn')}
                  <Tooltip title={t('admin.bindDnTooltip')}>
                    <InfoCircleOutlined style={{ marginLeft: 4 }} />
                  </Tooltip>
                </span>
              }
              rules={[{ required: ldapEnabled, message: t('admin.pleaseCompleteLdapForm') }]}
            >
              <Input 
                placeholder={t('admin.bindDnPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="bind_password"
              label={t('admin.bindPassword')}
              rules={[{ required: ldapEnabled, message: t('admin.pleaseCompleteLdapForm') }]}
            >
              <Input.Password 
                placeholder={t('admin.bindPasswordPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="base_dn"
              label={
                <span>
                  {t('admin.baseSearchDn')}
                  <Tooltip title={t('admin.searchBaseDnTooltip')}>
                    <InfoCircleOutlined style={{ marginLeft: 4 }} />
                  </Tooltip>
                </span>
              }
              rules={[{ required: ldapEnabled, message: t('admin.pleaseCompleteLdapForm') }]}
            >
              <Input 
                placeholder={t('admin.searchBaseDnPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="user_filter"
              label={
                <span>
                  {t('admin.userSearchFilterLabel')}
                  <Tooltip title={t('admin.userSearchFilterTooltipLabel')}>
                    <InfoCircleOutlined style={{ marginLeft: 4 }} />
                  </Tooltip>
                </span>
              }
              rules={[{ required: ldapEnabled, message: t('admin.pleaseCompleteLdapForm') }]}
            >
              <Input 
                placeholder={t('admin.userSearchFilterPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>
          </Card>

          <Card 
            size="small" 
            title={t('admin.userAttrMapping')} 
            style={{ 
              marginBottom: '16px',
              opacity: ldapEnabled ? 1 : 0.6
            }}
          >
            <Form.Item
              name="username_attr"
              label={t('admin.usernameAttr')}
              rules={[{ required: ldapEnabled, message: t('admin.pleaseCompleteLdapForm') }]}
            >
              <Input 
                placeholder={t('admin.usernameAttrPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="email_attr"
              label={t('admin.emailAttr')}
              rules={[{ required: ldapEnabled, message: t('admin.pleaseCompleteLdapForm') }]}
            >
              <Input 
                placeholder={t('admin.emailAttrPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="display_name_attr"
              label={t('admin.displayNameAttr')}
            >
              <Input 
                placeholder={t('admin.displayNameAttrPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>
          </Card>

          <Card 
            size="small" 
            title={t('admin.adminPermissions')} 
            style={{ 
              marginBottom: '24px',
              opacity: ldapEnabled ? 1 : 0.6
            }}
          >
            <Form.Item
              name="admin_group_dn"
              label={
                <span>
                  {t('admin.adminGroupDn')}
                  <Tooltip title={t('admin.adminGroupDnTooltip')}>
                    <InfoCircleOutlined style={{ marginLeft: 4 }} />
                  </Tooltip>
                </span>
              }
            >
              <Input 
                placeholder={t('admin.adminGroupDnPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="group_member_attr"
              label={t('admin.groupMemberAttr')}
            >
              <Input 
                placeholder={t('admin.groupMemberAttrPlaceholder')} 
                disabled={!ldapEnabled}
              />
            </Form.Item>
          </Card>

          <Divider />

          <Space>
            <Button
              type="primary"
              htmlType="submit"
              icon={<SaveOutlined />}
              loading={saving}
            >
              {t('admin.saveConfig')}
            </Button>
            
            <Button
              icon={<ExperimentOutlined />}
              loading={testing}
              onClick={handleTest}
              disabled={!ldapEnabled}
            >
              {t('admin.testConnectionBtn')}
            </Button>
            
            <Button onClick={handleReset}>
              {t('admin.reset')}
            </Button>
          </Space>
        </Form>
      </Card>
    </div>
  );
};

export default AdminLDAP;
