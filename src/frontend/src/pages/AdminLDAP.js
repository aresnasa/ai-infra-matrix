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
  Collapse
} from 'antd';
import {
  SafetyOutlined,
  ExperimentOutlined,
  SaveOutlined,
  InfoCircleOutlined,
  BookOutlined
} from '@ant-design/icons';
import { adminAPI } from '../services/api';

const { Title, Text } = Typography;
const { Option } = Select;
const { TextArea } = Input;
const { Panel } = Collapse;

const AdminLDAP = () => {
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
        message.error('加载LDAP配置失败');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (values) => {
    setSaving(true);
    try {
      await adminAPI.updateLDAPConfig(values);
      message.success('LDAP配置保存成功');
      await loadLDAPConfig();
      setTestResult(null); // 清除之前的测试结果
    } catch (error) {
      message.error(error.response?.data?.message || '保存LDAP配置失败');
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    try {
      const values = await form.validateFields();
      if (!values.enabled) {
        message.warning('请先启用LDAP认证');
        return;
      }
      setTesting(true);
      const response = await adminAPI.testLDAPConnection(values);
      setTestResult({
        success: true,
        message: response.data.message || '连接测试成功'
      });
      message.success('LDAP连接测试成功');
    } catch (error) {
      if (error.errorFields) {
        message.error('请先完善表单信息');
        return;
      }
      setTestResult({
        success: false,
        message: error.response?.data?.message || '连接测试失败'
      });
      message.error('LDAP连接测试失败');
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
    message.success('已填充测试环境配置');
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
            LDAP配置管理
          </Title>
          <Text type="secondary">
            配置LDAP服务器连接信息，启用企业级用户认证
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
                <strong>配置样例参考</strong>
                <Text type="secondary" style={{ marginLeft: '8px' }}>
                  点击展开查看完整的LDAP配置示例
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
                🔧 测试环境配置样例
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
                <div style={{ color: '#6a737d', marginBottom: '8px' }}>/* 基本连接配置 */</div>
                <div><strong>服务器地址:</strong> openldap</div>
                <div><strong>端口:</strong> 389</div>
                <div><strong>使用SSL:</strong> false</div>
                <div><strong>StartTLS:</strong> false</div>
                <div style={{ marginTop: '12px', color: '#6a737d', marginBottom: '8px' }}>/* 认证信息 */</div>
                <div><strong>绑定DN:</strong> cn=admin,dc=testcompany,dc=com</div>
                <div><strong>绑定密码:</strong> admin123</div>
                <div><strong>基准DN:</strong> dc=testcompany,dc=com</div>
              </div>
              
              <Alert
                message="配置提示"
                description={
                  <div>
                    <p>✅ <strong>测试环境</strong>: 使用上述配置可直接连接当前Docker环境中的LDAP服务</p>
                    <p>⚙️ <strong>生产环境</strong>: 请根据您的实际LDAP服务器信息进行调整</p>
                    <p>🔐 <strong>安全建议</strong>: 生产环境建议启用SSL/TLS加密连接</p>
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
                  🚀 快速填充测试配置
                </Button>
                <Text type="secondary" style={{ marginLeft: '8px', fontSize: '12px' }}>
                  一键填充上述测试环境配置到表单
                </Text>
              </div>
            </div>
          </Panel>
        </Collapse>

        {testResult && (
          <Alert
            message={testResult.success ? '连接测试成功' : '连接测试失败'}
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
          <Card size="small" title="基本配置" style={{ marginBottom: '16px' }}>
            <Form.Item
              name="enabled"
              label="启用LDAP认证"
              valuePropName="checked"
              extra={ldapEnabled ? "LDAP认证已启用，用户可通过企业账户登录" : "LDAP认证已禁用，仅本地账户可登录"}
            >
              <Switch 
                checkedChildren="启用" 
                unCheckedChildren="禁用"
                onChange={handleLdapToggle}
              />
            </Form.Item>

            <Form.Item
              name="server"
              label={
                <span>
                  LDAP服务器地址
                  <Tooltip title="LDAP服务器的IP地址或域名">
                    <InfoCircleOutlined style={{ marginLeft: 4 }} />
                  </Tooltip>
                </span>
              }
              rules={[
                { 
                  required: ldapEnabled, 
                  message: '请输入LDAP服务器地址' 
                }
              ]}
            >
              <Input 
                placeholder="测试环境: openldap | 生产环境: ldap.company.com" 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="port"
              label="端口"
              rules={[
                { required: ldapEnabled, message: '请输入端口号' },
                { type: 'number', min: 1, max: 65535, message: '端口号范围1-65535' }
              ]}
            >
              <Input 
                type="number" 
                placeholder="389 (LDAP) 或 636 (LDAPS)" 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="security"
              label="安全连接"
              rules={[{ required: ldapEnabled, message: '请选择安全连接类型' }]}
            >
              <Select disabled={!ldapEnabled}>
                <Option value="none">无加密</Option>
                <Option value="ssl">SSL/TLS</Option>
                <Option value="starttls">StartTLS</Option>
              </Select>
            </Form.Item>

            <Form.Item
              name="timeout"
              label="连接超时(秒)"
              rules={[
                { required: ldapEnabled, message: '请输入超时时间' },
                { type: 'number', min: 1, max: 300, message: '超时时间范围1-300秒' }
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
            title="认证配置" 
            style={{ 
              marginBottom: '16px',
              opacity: ldapEnabled ? 1 : 0.6
            }}
          >
            <Form.Item
              name="bind_dn"
              label={
                <span>
                  绑定DN
                  <Tooltip title="用于连接LDAP的管理员账户DN">
                    <InfoCircleOutlined style={{ marginLeft: 4 }} />
                  </Tooltip>
                </span>
              }
              rules={[{ required: ldapEnabled, message: '请输入绑定DN' }]}
            >
              <Input 
                placeholder="测试环境: cn=admin,dc=testcompany,dc=com" 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="bind_password"
              label="绑定密码"
              rules={[{ required: ldapEnabled, message: '请输入绑定密码' }]}
            >
              <Input.Password 
                placeholder="测试环境: admin123" 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="base_dn"
              label={
                <span>
                  搜索基准DN
                  <Tooltip title="用户搜索的起始位置">
                    <InfoCircleOutlined style={{ marginLeft: 4 }} />
                  </Tooltip>
                </span>
              }
              rules={[{ required: ldapEnabled, message: '请输入搜索基准DN' }]}
            >
              <Input 
                placeholder="测试环境: dc=testcompany,dc=com | 生产环境: ou=users,dc=company,dc=com" 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="user_filter"
              label={
                <span>
                  用户搜索过滤器
                  <Tooltip title="用于搜索用户的LDAP过滤器，{username}会被实际用户名替换">
                    <InfoCircleOutlined style={{ marginLeft: 4 }} />
                  </Tooltip>
                </span>
              }
              rules={[{ required: ldapEnabled, message: '请输入用户搜索过滤器' }]}
            >
              <Input 
                placeholder="例如: (uid={username}) 或 (sAMAccountName={username})" 
                disabled={!ldapEnabled}
              />
            </Form.Item>
          </Card>

          <Card 
            size="small" 
            title="用户属性映射" 
            style={{ 
              marginBottom: '16px',
              opacity: ldapEnabled ? 1 : 0.6
            }}
          >
            <Form.Item
              name="username_attr"
              label="用户名属性"
              rules={[{ required: ldapEnabled, message: '请输入用户名属性' }]}
            >
              <Input 
                placeholder="例如: uid 或 sAMAccountName" 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="email_attr"
              label="邮箱属性"
              rules={[{ required: ldapEnabled, message: '请输入邮箱属性' }]}
            >
              <Input 
                placeholder="例如: mail" 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="display_name_attr"
              label="显示名称属性"
            >
              <Input 
                placeholder="例如: displayName 或 cn" 
                disabled={!ldapEnabled}
              />
            </Form.Item>
          </Card>

          <Card 
            size="small" 
            title="管理员权限" 
            style={{ 
              marginBottom: '24px',
              opacity: ldapEnabled ? 1 : 0.6
            }}
          >
            <Form.Item
              name="admin_group_dn"
              label={
                <span>
                  管理员组DN
                  <Tooltip title="具有管理员权限的LDAP组，留空则所有用户都是普通用户">
                    <InfoCircleOutlined style={{ marginLeft: 4 }} />
                  </Tooltip>
                </span>
              }
            >
              <Input 
                placeholder="例如: cn=admins,ou=groups,dc=company,dc=com" 
                disabled={!ldapEnabled}
              />
            </Form.Item>

            <Form.Item
              name="group_member_attr"
              label="组成员属性"
            >
              <Input 
                placeholder="例如: member 或 memberUid" 
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
              保存配置
            </Button>
            
            <Button
              icon={<ExperimentOutlined />}
              loading={testing}
              onClick={handleTest}
              disabled={!ldapEnabled}
            >
              测试连接
            </Button>
            
            <Button onClick={handleReset}>
              重置
            </Button>
          </Space>
        </Form>
      </Card>
    </div>
  );
};

export default AdminLDAP;
