import React, { useState } from 'react';
import { Form, Input, Button, Card, message, Tabs, Row, Col, Select, Checkbox, Alert, Descriptions, Divider, Modal, Space } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined, TeamOutlined, CheckCircleOutlined, ExclamationCircleOutlined, SafetyOutlined } from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import { authAPI, securityAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';
import { useTheme } from '../hooks/useTheme';
import LanguageSwitcher from '../components/LanguageSwitcher';
import ThemeSwitcher from '../components/ThemeSwitcher';
import './Auth.css';

const { TabPane } = Tabs;
const { Option } = Select;

// 角色模板配置 - 根据团队权限重新定义
const ROLE_TEMPLATES = {
  'data-developer': {
    nameKey: 'roleTemplates.dataDeveloper',
    descKey: 'roleTemplates.dataDeveloperDesc',
    permissions: ['JupyterHub访问', 'Slurm作业调度', '项目管理', '数据分析工具'],
    permissionsEn: ['JupyterHub Access', 'Slurm Job Scheduling', 'Project Management', 'Data Analysis Tools'],
    allowedRoutes: ['/projects', '/jupyterhub', '/slurm', '/dashboard', '/enhanced-dashboard'],
    restrictedRoutes: ['/admin', '/saltstack', '/ansible', '/kubernetes', '/kafka-ui']
  },
  'sre': {
    nameKey: 'roleTemplates.sre',
    descKey: 'roleTemplates.sreDesc',
    permissions: ['SaltStack管理', 'Ansible自动化', 'Kubernetes集群管理', '主机管理', '系统监控', '日志管理'],
    permissionsEn: ['SaltStack Management', 'Ansible Automation', 'Kubernetes Cluster', 'Host Management', 'System Monitoring', 'Log Management'],
    allowedRoutes: ['/projects', '/jupyterhub', '/slurm', '/saltstack', '/ansible', '/kubernetes', '/dashboard', '/enhanced-dashboard', '/admin'],
    restrictedRoutes: []
  },
  'audit': {
    nameKey: 'roleTemplates.audit',
    descKey: 'roleTemplates.auditDesc',
    permissions: ['Kafka消息队列管理', '聊天机器人审核', '系统审计', '日志分析', '合规检查'],
    permissionsEn: ['Kafka Queue Management', 'Chatbot Review', 'System Audit', 'Log Analysis', 'Compliance Check'],
    allowedRoutes: ['/projects', '/kafka-ui', '/dashboard', '/enhanced-dashboard', '/audit-logs'],
    restrictedRoutes: ['/admin', '/saltstack', '/ansible', '/kubernetes', '/jupyterhub', '/slurm']
  }
};

const AuthPage = ({ onLogin }) => {
  const { t, locale } = useI18n();
  const navigate = useNavigate();
  const location = useLocation();
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('login');
  const [selectedRoleTemplate, setSelectedRoleTemplate] = useState('');
  const [requiresApproval, setRequiresApproval] = useState(true);
  const [ldapValidationStatus, setLdapValidationStatus] = useState(null); // null, 'validating', 'success', 'error'
  
  // 2FA 相关状态
  const [show2FAModal, setShow2FAModal] = useState(false);
  const [twoFACode, setTwoFACode] = useState('');
  const [pendingLoginData, setPendingLoginData] = useState(null);
  const [verifying2FA, setVerifying2FA] = useState(false);

  // 获取角色模板名称
  const getRoleTemplateName = (key) => {
    const names = {
      'data-developer': { 'zh-CN': '数据开发团队', 'en-US': 'Data Developer Team' },
      'sre': { 'zh-CN': 'SRE运维团队', 'en-US': 'SRE Operations Team' },
      'audit': { 'zh-CN': '审计审核团队', 'en-US': 'Audit Review Team' }
    };
    return names[key]?.[locale] || key;
  };

  // 获取角色模板描述
  const getRoleTemplateDesc = (key) => {
    const descs = {
      'data-developer': { 
        'zh-CN': '专注于数据分析和模型开发，主要使用Jupyter和Slurm环境', 
        'en-US': 'Focus on data analysis and model development, mainly using Jupyter and Slurm environments' 
      },
      'sre': { 
        'zh-CN': '负责基础设施运维，拥有SaltStack、Ansible和K8s管理权限', 
        'en-US': 'Responsible for infrastructure operations, with SaltStack, Ansible and K8s management permissions' 
      },
      'audit': { 
        'zh-CN': '负责系统审计和聊天机器人审核，拥有Kafka和审核工具权限', 
        'en-US': 'Responsible for system audit and chatbot review, with Kafka and audit tools permissions' 
      }
    };
    return descs[key]?.[locale] || '';
  };

  const handleLogin = async (values) => {
    setLoading(true);
    try {
      const response = await authAPI.login(values);
      const data = response.data;

      // 检查是否需要2FA验证
      if (data.requires_2fa) {
        console.log('2FA required for user:', values.username);
        setPendingLoginData({
          ...values,
          tempToken: data.temp_token,
          userId: data.user_id
        });
        setShow2FAModal(true);
        setLoading(false);
        return;
      }

      // 正常登录流程
      const { token, user, expires_at } = data;

      console.log('=== Login API Success ===');
      console.log('Got token:', token ? 'yes' : 'no');
      console.log('Login user:', user);
      console.log('Location before login:', location.state?.from);

      // 保存token
      localStorage.setItem('token', token);
      localStorage.setItem('token_expires', expires_at);

      // 确保localStorage写入完成
      await new Promise(resolve => setTimeout(resolve, 100));

      message.success(t('auth.loginSuccess'));

      // 传递完整的登录响应数据，包括token信息
      await onLogin({
        token,
        expires_at,
        user
      });

      // 登录成功后，重定向到用户之前想访问的页面
      const from = location.state?.from?.pathname || '/projects';
      console.log('Redirect after login:', from);
      navigate(from, { replace: true });

    } catch (error) {
      console.error('Login failed:', error);
      message.error(error.response?.data?.error || t('auth.loginFailed'));
    } finally {
      setLoading(false);
    }
  };

  // 处理2FA验证
  const handle2FAVerify = async () => {
    if (!twoFACode || twoFACode.length !== 6) {
      message.error('请输入6位验证码');
      return;
    }

    setVerifying2FA(true);
    try {
      const response = await authAPI.verify2FALogin({
        temp_token: pendingLoginData.tempToken,
        code: twoFACode,
        username: pendingLoginData.username
      });

      const { token, user, expires_at } = response.data;

      // 保存token
      localStorage.setItem('token', token);
      localStorage.setItem('token_expires', expires_at);

      await new Promise(resolve => setTimeout(resolve, 100));

      message.success(t('auth.loginSuccess'));
      setShow2FAModal(false);
      setTwoFACode('');
      setPendingLoginData(null);

      await onLogin({
        token,
        expires_at,
        user
      });

      const from = location.state?.from?.pathname || '/projects';
      navigate(from, { replace: true });

    } catch (error) {
      console.error('2FA verification failed:', error);
      message.error(error.response?.data?.error || '验证码错误或已过期');
    } finally {
      setVerifying2FA(false);
    }
  };

  // 取消2FA验证
  const cancel2FA = () => {
    setShow2FAModal(false);
    setTwoFACode('');
    setPendingLoginData(null);
  };

  const handleRegister = async (values) => {
    setLoading(true);
    setLdapValidationStatus('validating');

    try {
      // 首先进行LDAP验证
      const ldapCheckResponse = await authAPI.validateLDAP({
        username: values.username,
        password: values.password
      });

      if (!ldapCheckResponse.data.valid) {
        setLdapValidationStatus('error');
        message.error(t('auth.ldapValidateError'));
        return;
      }

      setLdapValidationStatus('success');
      message.success(t('auth.ldapValidateSuccessMsg'));

      // 提交注册申请
      const registerData = {
        ...values,
        role_template: selectedRoleTemplate,
        requires_approval: requiresApproval
      };

      const response = await authAPI.register(registerData);

      if (requiresApproval) {
        message.success(t('auth.registerSuccess'));
        setActiveTab('login');
      } else {
        message.success(t('auth.registerSuccessNoApproval'));
        setActiveTab('login');
      }

    } catch (error) {
      console.error('Register failed:', error);
      setLdapValidationStatus('error');

      if (error.response?.data?.error?.includes('LDAP')) {
        message.error(t('auth.ldapValidateFailed'));
      } else {
        message.error(error.response?.data?.error || t('auth.loginFailed'));
      }
    } finally {
      setLoading(false);
    }
  };

  const handleRoleTemplateChange = (value) => {
    setSelectedRoleTemplate(value);
  };

  const renderRoleTemplateInfo = () => {
    if (!selectedRoleTemplate) return null;

    const template = ROLE_TEMPLATES[selectedRoleTemplate];
    const templateName = getRoleTemplateName(selectedRoleTemplate);
    const templateDesc = getRoleTemplateDesc(selectedRoleTemplate);
    const permissions = locale === 'en-US' ? template.permissionsEn : template.permissions;
    
    return (
      <Alert
        message={`${templateName} - ${t('auth.roleTemplatePermissions')}`}
        description={
          <div>
            <p>{templateDesc}</p>
            <Divider style={{ margin: '8px 0' }} />
            <strong>{t('auth.includedPermissions')}：</strong>
            <ul style={{ margin: '8px 0', paddingLeft: '20px' }}>
              {(permissions || []).map((perm, index) => (
                <li key={index}>{perm}</li>
              ))}
            </ul>
          </div>
        }
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
    );
  };

  return (
    <div className="auth-container">
      {/* 右上角语言切换和主题切换 */}
      <div className="auth-header-controls">
        <ThemeSwitcher showLabel={false} size="middle" />
        <LanguageSwitcher showLabel={false} />
      </div>
      
      <Row justify="center" align="middle" style={{ minHeight: '100vh', padding: '20px 0' }}>
        <Col xs={24} sm={22} md={18} lg={14} xl={10} style={{ display: 'flex', justifyContent: 'center' }}>
          <Card title={
            <div style={{ textAlign: 'center' }}>
              <h2>{t('auth.systemTitle')}</h2>
              <p style={{ color: '#666', margin: '8px 0' }}>{t('auth.systemSubtitle')}</p>
            </div>
          }>
            <Tabs activeKey={activeTab} onChange={setActiveTab} centered>
              <TabPane tab={t('auth.login')} key="login">
                <Form
                  name="login"
                  onFinish={handleLogin}
                  autoComplete="off"
                  size="large"
                >
                  <Form.Item
                    name="username"
                    rules={[{ required: true, message: t('auth.usernameRequired') }]}
                  >
                    <Input
                      prefix={<UserOutlined />}
                      placeholder={t('auth.username')}
                    />
                  </Form.Item>

                  <Form.Item
                    name="password"
                    rules={[{ required: true, message: t('auth.passwordRequired') }]}
                  >
                    <Input.Password
                      prefix={<LockOutlined />}
                      placeholder={t('auth.password')}
                    />
                  </Form.Item>

                  <Form.Item>
                    <Button type="primary" htmlType="submit" loading={loading} block>
                      {t('auth.login')}
                    </Button>
                  </Form.Item>
                </Form>

                <div style={{ textAlign: 'center', marginTop: 16, fontSize: '12px', color: '#888' }}>
                  <p>{t('auth.defaultAdmin')}</p>
                  <p>{t('auth.defaultUsername')}</p>
                  <p>{t('auth.defaultPassword')}</p>
                </div>
              </TabPane>

              <TabPane tab={t('auth.register')} key="register">
                <Alert
                  message={t('auth.registerNotice')}
                  description={t('auth.registerNoticeDesc')}
                  type="warning"
                  showIcon
                  style={{ marginBottom: 16 }}
                />

                <Form
                  name="register"
                  onFinish={handleRegister}
                  autoComplete="off"
                  size="large"
                >
                  <Form.Item
                    name="username"
                    rules={[
                      { required: true, message: t('auth.usernameRequired') },
                      { min: 3, max: 50, message: t('auth.usernameLength') }
                    ]}
                  >
                    <Input
                      prefix={<UserOutlined />}
                      placeholder={t('auth.username')}
                      onChange={() => setLdapValidationStatus(null)}
                    />
                  </Form.Item>

                  <Form.Item
                    name="email"
                    rules={[
                      { required: true, message: t('auth.emailRequired') },
                      { type: 'email', message: t('auth.emailInvalid') }
                    ]}
                  >
                    <Input
                      prefix={<MailOutlined />}
                      placeholder={t('auth.email')}
                    />
                  </Form.Item>

                  <Form.Item
                    name="password"
                    rules={[
                      { required: true, message: t('auth.passwordRequired') },
                      { min: 6, message: t('auth.passwordMinLength') }
                    ]}
                  >
                    <Input.Password
                      prefix={<LockOutlined />}
                      placeholder={t('auth.password')}
                      onChange={() => setLdapValidationStatus(null)}
                    />
                  </Form.Item>

                  <Form.Item
                    name="confirmPassword"
                    dependencies={['password']}
                    rules={[
                      { required: true, message: t('auth.confirmPasswordRequired') },
                      ({ getFieldValue }) => ({
                        validator(_, value) {
                          if (!value || getFieldValue('password') === value) {
                            return Promise.resolve();
                          }
                          return Promise.reject(new Error(t('auth.passwordMismatch')));
                        },
                      }),
                    ]}
                  >
                    <Input.Password
                      prefix={<LockOutlined />}
                      placeholder={t('auth.confirmPassword')}
                    />
                  </Form.Item>

                  <Form.Item
                    name="department"
                    rules={[{ required: true, message: t('auth.departmentRequired') }]}
                  >
                    <Input
                      prefix={<TeamOutlined />}
                      placeholder={t('auth.department')}
                    />
                  </Form.Item>

                  <Form.Item
                    name="role_template"
                    rules={[{ required: true, message: t('auth.roleTemplateRequired') }]}
                  >
                    <Select
                      placeholder={t('auth.selectRoleTemplate')}
                      onChange={handleRoleTemplateChange}
                      size="large"
                    >
                      {Object.entries(ROLE_TEMPLATES || {}).map(([key, template]) => (
                        <Option key={key} value={key}>
                          {getRoleTemplateName(key)}
                        </Option>
                      ))}
                    </Select>
                  </Form.Item>

                  {renderRoleTemplateInfo()}

                  <Form.Item>
                    <Checkbox
                      checked={requiresApproval}
                      onChange={(e) => setRequiresApproval(e.target.checked)}
                    >
                      {t('auth.requiresApproval')}
                    </Checkbox>
                  </Form.Item>

                  {ldapValidationStatus === 'validating' && (
                    <Alert
                      message={t('auth.ldapValidating')}
                      description={t('auth.ldapValidatingDesc')}
                      type="info"
                      showIcon
                      style={{ marginBottom: 16 }}
                    />
                  )}

                  {ldapValidationStatus === 'success' && (
                    <Alert
                      message={t('auth.ldapValidateSuccess')}
                      description={t('auth.ldapValidateSuccessDesc')}
                      type="success"
                      showIcon
                      icon={<CheckCircleOutlined />}
                      style={{ marginBottom: 16 }}
                    />
                  )}

                  {ldapValidationStatus === 'error' && (
                    <Alert
                      message={t('auth.ldapValidateFailed')}
                      description={t('auth.ldapValidateFailedDesc')}
                      type="error"
                      showIcon
                      icon={<ExclamationCircleOutlined />}
                      style={{ marginBottom: 16 }}
                    />
                  )}

                  <Form.Item>
                    <Button
                      type="primary"
                      htmlType="submit"
                      loading={loading}
                      block
                      disabled={ldapValidationStatus === 'error'}
                    >
                      {requiresApproval ? t('auth.submitRegister') : t('auth.register')}
                    </Button>
                  </Form.Item>
                </Form>

                <div style={{ marginTop: 16, padding: 16, backgroundColor: '#f9f9f9', borderRadius: 4 }}>
                  <h4 style={{ marginBottom: 8 }}>{t('auth.registerProcess')}</h4>
                  <ol style={{ margin: 0, paddingLeft: 20, fontSize: '12px', color: '#666' }}>
                    <li>{t('auth.registerStep1')}</li>
                    <li>{t('auth.registerStep2')}</li>
                    <li>{t('auth.registerStep3')}</li>
                    <li>{t('auth.registerStep4')}</li>
                  </ol>
                </div>
              </TabPane>
            </Tabs>
          </Card>
        </Col>
      </Row>

      {/* 2FA验证弹窗 */}
      <Modal
        title={
          <Space>
            <SafetyOutlined />
            双因素认证验证
          </Space>
        }
        open={show2FAModal}
        onCancel={cancel2FA}
        footer={null}
        centered
        maskClosable={false}
        width={400}
      >
        <div style={{ padding: '20px 0' }}>
          <Alert
            message="此账户已启用双因素认证"
            description="请打开您的身份验证器应用（如 Google Authenticator），输入6位验证码完成登录。"
            type="info"
            showIcon
            style={{ marginBottom: '24px' }}
          />
          
          <Form onFinish={handle2FAVerify}>
            <Form.Item>
              <Input
                size="large"
                placeholder="请输入6位验证码"
                value={twoFACode}
                onChange={(e) => setTwoFACode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                maxLength={6}
                style={{ 
                  textAlign: 'center', 
                  fontSize: '24px', 
                  letterSpacing: '8px',
                  fontFamily: 'monospace'
                }}
                autoFocus
              />
            </Form.Item>
            
            <Form.Item style={{ marginBottom: 0 }}>
              <Space style={{ width: '100%', justifyContent: 'space-between' }}>
                <Button onClick={cancel2FA}>
                  取消
                </Button>
                <Button 
                  type="primary" 
                  htmlType="submit"
                  loading={verifying2FA}
                  disabled={twoFACode.length !== 6}
                >
                  验证并登录
                </Button>
              </Space>
            </Form.Item>
          </Form>
          
          <Divider />
          
          <div style={{ textAlign: 'center' }}>
            <Button type="link" size="small">
              无法访问验证器？使用恢复码
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default AuthPage;
