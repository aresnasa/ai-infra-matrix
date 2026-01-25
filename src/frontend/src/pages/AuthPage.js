import React, { useState, useEffect } from 'react';
import { Form, Input, Button, Card, message, Tabs, Row, Col, Select, Checkbox, Alert, Descriptions, Divider, Modal, Space } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined, TeamOutlined, CheckCircleOutlined, ExclamationCircleOutlined, SafetyOutlined, LoadingOutlined } from '@ant-design/icons';
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
  
  // 注册配置状态
  const [registrationConfig, setRegistrationConfig] = useState({
    requireInvitationCode: true, // 默认需要邀请码
    disableRegistration: false,
    allowApprovalMode: false,
  });
  const [ldapValidationStatus, setLdapValidationStatus] = useState(null); // null, 'validating', 'success', 'error'
  
  // 邀请码相关状态
  const [invitationCode, setInvitationCode] = useState('');
  const [invitationCodeValid, setInvitationCodeValid] = useState(null); // null, true, false
  const [invitationCodeChecking, setInvitationCodeChecking] = useState(false);
  const [invitationCodeInfo, setInvitationCodeInfo] = useState(null);
  
  // 2FA 相关状态
  const [show2FAModal, setShow2FAModal] = useState(false);
  const [twoFACode, setTwoFACode] = useState('');
  const [pendingLoginData, setPendingLoginData] = useState(null);
  const [verifying2FA, setVerifying2FA] = useState(false);
  const [useRecoveryCode, setUseRecoveryCode] = useState(false);

  // 获取注册配置
  useEffect(() => {
    const fetchRegistrationConfig = async () => {
      try {
        const response = await authAPI.getRegistrationConfig();
        if (response.data) {
          setRegistrationConfig({
            requireInvitationCode: response.data.require_invitation_code ?? true,
            disableRegistration: response.data.disable_registration ?? false,
            allowApprovalMode: response.data.allow_approval_mode ?? false,
          });
        }
      } catch (error) {
        console.error('Failed to fetch registration config:', error);
        // 默认使用最严格的配置
        setRegistrationConfig({
          requireInvitationCode: true,
          disableRegistration: false,
          allowApprovalMode: false,
        });
      }
    };
    fetchRegistrationConfig();
  }, []);

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
    if (!useRecoveryCode && (!twoFACode || twoFACode.length !== 6)) {
      message.error('请输入6位验证码');
      return;
    }
    if (useRecoveryCode && !twoFACode) {
      message.error('请输入恢复码');
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
    setUseRecoveryCode(false);
  };

  // 切换恢复码模式
  const toggleRecoveryMode = () => {
    setUseRecoveryCode(!useRecoveryCode);
    setTwoFACode('');
  };

  // 验证邀请码
  const validateInvitationCode = async (code) => {
    if (!code || code.trim() === '') {
      setInvitationCodeValid(null);
      setInvitationCodeInfo(null);
      return;
    }

    setInvitationCodeChecking(true);
    try {
      const response = await authAPI.validateInvitationCode(code.trim());
      if (response.data.valid) {
        setInvitationCodeValid(true);
        setInvitationCodeInfo(response.data);
        // 如果邀请码预设了角色模板，自动选中
        if (response.data.role_template) {
          setSelectedRoleTemplate(response.data.role_template);
        }
      } else {
        setInvitationCodeValid(false);
        setInvitationCodeInfo(null);
      }
    } catch (error) {
      setInvitationCodeValid(false);
      setInvitationCodeInfo(null);
    } finally {
      setInvitationCodeChecking(false);
    }
  };

  // 处理邀请码输入变化（防抖）
  const handleInvitationCodeChange = (e) => {
    const code = e.target.value;
    setInvitationCode(code);
    // 使用防抖延迟验证
    if (code.trim().length >= 16) {
      validateInvitationCode(code);
    } else {
      setInvitationCodeValid(null);
      setInvitationCodeInfo(null);
    }
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
        invitation_code: invitationCode.trim() || undefined, // 添加邀请码
      };

      const response = await authAPI.register(registerData);

      // 根据返回结果判断是邀请码注册还是普通注册
      if (response.data.activated) {
        // 邀请码注册成功，可以直接登录
        message.success(t('auth.invitationRegisterSuccess'));
        setActiveTab('login');
      } else {
        // 普通注册，需要等待审批
        message.success(t('auth.registerSuccess'));
        setActiveTab('login');
      }

      // 清空邀请码
      setInvitationCode('');
      setInvitationCodeValid(null);
      setInvitationCodeInfo(null);

    } catch (error) {
      console.error('Register failed:', error);
      
      const errorMessage = error.response?.data?.error || '';
      const statusCode = error.response?.status;
      
      // 只有在 LDAP 验证阶段（而非注册阶段）出错时才设置 LDAP 错误状态
      if (errorMessage.includes('LDAP') || ldapValidationStatus === 'validating') {
        setLdapValidationStatus('error');
        message.error(t('auth.ldapValidateFailed'));
      } else if (statusCode === 409 || errorMessage.includes('已存在') || errorMessage.includes('already exists')) {
        message.error(errorMessage || t('auth.userAlreadyExists'));
      } else if (errorMessage.includes('邀请码')) {
        // 邀请码相关错误
        message.error(errorMessage);
        setInvitationCodeValid(false);
      } else if (errorMessage.includes('待审批')) {
        message.error(errorMessage);
      } else {
        message.error(errorMessage || t('auth.loginFailed'));
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

                  {/* 邀请码输入框 */}
                  <Form.Item
                    name="invitation_code"
                    rules={registrationConfig.requireInvitationCode ? [
                      { required: true, message: t('auth.invitationCodeRequired') }
                    ] : []}
                    extra={invitationCodeValid === true ? (
                      <span style={{ color: '#52c41a' }}>
                        ✓ {t('auth.invitationCodeValid')}
                        {invitationCodeInfo?.role_template && ` (${t('auth.presetRole')}: ${getRoleTemplateName(invitationCodeInfo.role_template)})`}
                      </span>
                    ) : invitationCodeValid === false ? (
                      <span style={{ color: '#ff4d4f' }}>✗ {t('auth.invitationCodeInvalid')}</span>
                    ) : registrationConfig.requireInvitationCode ? (
                      <span style={{ color: '#ff4d4f' }}>{t('auth.invitationCodeRequiredHint')}</span>
                    ) : (
                      <span style={{ color: '#666' }}>{t('auth.invitationCodeHint')}</span>
                    )}
                  >
                    <Input
                      prefix={<SafetyOutlined />}
                      placeholder={registrationConfig.requireInvitationCode ? t('auth.invitationCodeRequiredPlaceholder') : t('auth.invitationCode')}
                      value={invitationCode}
                      onChange={handleInvitationCodeChange}
                      suffix={invitationCodeChecking ? <LoadingOutlined /> : null}
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
                      value={selectedRoleTemplate}
                      disabled={invitationCodeValid === true && invitationCodeInfo?.role_template}
                    >
                      {Object.entries(ROLE_TEMPLATES || {}).map(([key, template]) => (
                        <Option key={key} value={key}>
                          {getRoleTemplateName(key)}
                        </Option>
                      ))}
                    </Select>
                  </Form.Item>

                  {renderRoleTemplateInfo()}

                  {/* 根据注册配置和邀请码状态显示不同的提示 */}
                  {invitationCodeValid === true ? (
                    <Alert
                      message={t('auth.invitationCodeMode')}
                      description={t('auth.invitationCodeModeDesc')}
                      type="success"
                      showIcon
                      style={{ marginBottom: 16 }}
                    />
                  ) : registrationConfig.requireInvitationCode ? (
                    <Alert
                      message={t('auth.invitationCodeRequiredMode')}
                      description={t('auth.invitationCodeRequiredModeDesc')}
                      type="warning"
                      showIcon
                      style={{ marginBottom: 16 }}
                    />
                  ) : (
                    <Alert
                      message={t('auth.approvalRequired')}
                      description={t('auth.approvalRequiredDesc')}
                      type="info"
                      showIcon
                      style={{ marginBottom: 16 }}
                    />
                  )}}

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
                      {t('auth.submitRegister')}
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
            description={useRecoveryCode 
              ? "请输入您保存的恢复码。每个恢复码只能使用一次。" 
              : "请打开您的身份验证器应用（如 Google Authenticator），输入6位验证码完成登录。"}
            type="info"
            showIcon
            style={{ marginBottom: '24px' }}
          />
          
          <Form onFinish={handle2FAVerify}>
            <Form.Item>
              {useRecoveryCode ? (
                <Input
                  size="large"
                  placeholder="请输入恢复码"
                  value={twoFACode}
                  onChange={(e) => setTwoFACode(e.target.value)}
                  style={{ 
                    textAlign: 'center', 
                    fontSize: '16px', 
                    fontFamily: 'monospace'
                  }}
                  autoFocus
                />
              ) : (
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
              )}
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
                  disabled={useRecoveryCode ? !twoFACode : twoFACode.length !== 6}
                >
                  验证并登录
                </Button>
              </Space>
            </Form.Item>
          </Form>
          
          <Divider />
          
          <div style={{ textAlign: 'center' }}>
            <Button type="link" size="small" onClick={toggleRecoveryMode}>
              {useRecoveryCode ? '使用验证码登录' : '无法访问验证器？使用恢复码'}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default AuthPage;
