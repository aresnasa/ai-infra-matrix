import React, { useState } from 'react';
import { Form, Input, Button, Card, message, Tabs, Row, Col, Select, Checkbox, Alert, Descriptions, Divider } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined, TeamOutlined, CheckCircleOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import { authAPI } from '../services/api';
import './Auth.css';

const { TabPane } = Tabs;
const { Option } = Select;

// 角色模板配置 - 根据团队权限重新定义
const ROLE_TEMPLATES = {
  'data-developer': {
    name: '数据开发团队',
    description: '专注于数据分析和模型开发，主要使用Jupyter和Slurm环境',
    permissions: ['JupyterHub访问', 'Slurm作业调度', '项目管理', '数据分析工具'],
    allowedRoutes: ['/projects', '/jupyterhub', '/slurm', '/dashboard', '/enhanced-dashboard'],
    restrictedRoutes: ['/admin', '/saltstack', '/ansible', '/kubernetes', '/kafka-ui']
  },
  'sre': {
    name: 'SRE运维团队',
    description: '负责基础设施运维，拥有SaltStack、Ansible和K8s管理权限',
    permissions: ['SaltStack管理', 'Ansible自动化', 'Kubernetes集群管理', '主机管理', '系统监控', '日志管理'],
    allowedRoutes: ['/projects', '/jupyterhub', '/slurm', '/saltstack', '/ansible', '/kubernetes', '/dashboard', '/enhanced-dashboard', '/admin'],
    restrictedRoutes: []
  },
  'audit': {
    name: '审计审核团队',
    description: '负责系统审计和聊天机器人审核，拥有Kafka和审核工具权限',
    permissions: ['Kafka消息队列管理', '聊天机器人审核', '系统审计', '日志分析', '合规检查'],
    allowedRoutes: ['/projects', '/kafka-ui', '/dashboard', '/enhanced-dashboard', '/audit-logs'],
    restrictedRoutes: ['/admin', '/saltstack', '/ansible', '/kubernetes', '/jupyterhub', '/slurm']
  }
};

const AuthPage = ({ onLogin }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('login');
  const [selectedRoleTemplate, setSelectedRoleTemplate] = useState('');
  const [requiresApproval, setRequiresApproval] = useState(true);
  const [ldapValidationStatus, setLdapValidationStatus] = useState(null); // null, 'validating', 'success', 'error'

  const handleLogin = async (values) => {
    setLoading(true);
    try {
      const response = await authAPI.login(values);
      const { token, user, expires_at } = response.data;

      console.log('=== 登录API调用成功 ===');
      console.log('获得token:', token ? '是' : '否');
      console.log('登录用户:', user);
      console.log('登录前的位置:', location.state?.from);

      // 保存token
      localStorage.setItem('token', token);
      localStorage.setItem('token_expires', expires_at);

      // 确保localStorage写入完成
      await new Promise(resolve => setTimeout(resolve, 100));

      message.success('登录成功！正在加载权限信息...');

      // 传递完整的登录响应数据，包括token信息
      await onLogin({
        token,
        expires_at,
        user
      });

      // 登录成功后，重定向到用户之前想访问的页面
      const from = location.state?.from?.pathname || '/projects';
      console.log('登录后重定向到:', from);
      navigate(from, { replace: true });

    } catch (error) {
      console.error('登录失败:', error);
      message.error(error.response?.data?.error || '登录失败，请检查用户名和密码');
    } finally {
      setLoading(false);
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
        message.error('LDAP验证失败：用户名或密码错误，或用户不存在于LDAP中');
        return;
      }

      setLdapValidationStatus('success');
      message.success('LDAP验证成功！正在提交注册申请...');

      // 提交注册申请
      const registerData = {
        ...values,
        role_template: selectedRoleTemplate,
        requires_approval: requiresApproval
      };

      const response = await authAPI.register(registerData);

      if (requiresApproval) {
        message.success('注册申请已提交！请等待管理员审批。');
        setActiveTab('login');
      } else {
        message.success('注册成功！请登录');
        setActiveTab('login');
      }

    } catch (error) {
      console.error('注册失败:', error);
      setLdapValidationStatus('error');

      if (error.response?.data?.error?.includes('LDAP')) {
        message.error('LDAP验证失败：请确保您在LDAP系统中存在且密码正确');
      } else {
        message.error(error.response?.data?.error || '注册失败');
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
    return (
      <Alert
        message={`${template.name} - 权限说明`}
        description={
          <div>
            <p>{template.description}</p>
            <Divider style={{ margin: '8px 0' }} />
            <strong>包含权限：</strong>
            <ul style={{ margin: '8px 0', paddingLeft: '20px' }}>
              {(template.permissions || []).map((perm, index) => (
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
      <Row justify="center" align="middle" style={{ minHeight: '100vh' }}>
        <Col xs={22} sm={20} md={16} lg={12} xl={10}>
          <Card title={
            <div style={{ textAlign: 'center' }}>
              <h2>AI-Infra-Matrix</h2>
              <p style={{ color: '#666', margin: '8px 0' }}>智能基础设施管理平台</p>
            </div>
          }>
            <Tabs activeKey={activeTab} onChange={setActiveTab} centered>
              <TabPane tab="登录" key="login">
                <Form
                  name="login"
                  onFinish={handleLogin}
                  autoComplete="off"
                  size="large"
                >
                  <Form.Item
                    name="username"
                    rules={[{ required: true, message: '请输入用户名!' }]}
                  >
                    <Input
                      prefix={<UserOutlined />}
                      placeholder="用户名"
                    />
                  </Form.Item>

                  <Form.Item
                    name="password"
                    rules={[{ required: true, message: '请输入密码!' }]}
                  >
                    <Input.Password
                      prefix={<LockOutlined />}
                      placeholder="密码"
                    />
                  </Form.Item>

                  <Form.Item>
                    <Button type="primary" htmlType="submit" loading={loading} block>
                      登录
                    </Button>
                  </Form.Item>
                </Form>

                <div style={{ textAlign: 'center', marginTop: 16, fontSize: '12px', color: '#888' }}>
                  <p>默认管理员账户：</p>
                  <p>用户名: admin</p>
                  <p>密码: admin123</p>
                </div>
              </TabPane>

              <TabPane tab="注册" key="register">
                <Alert
                  message="注册须知"
                  description="注册前请确保您已在LDAP系统中存在。注册后需要管理员审批才能使用系统。"
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
                      { required: true, message: '请输入用户名!' },
                      { min: 3, max: 50, message: '用户名长度为3-50个字符!' }
                    ]}
                  >
                    <Input
                      prefix={<UserOutlined />}
                      placeholder="用户名"
                      onChange={() => setLdapValidationStatus(null)}
                    />
                  </Form.Item>

                  <Form.Item
                    name="email"
                    rules={[
                      { required: true, message: '请输入邮箱!' },
                      { type: 'email', message: '请输入有效的邮箱地址!' }
                    ]}
                  >
                    <Input
                      prefix={<MailOutlined />}
                      placeholder="邮箱"
                    />
                  </Form.Item>

                  <Form.Item
                    name="password"
                    rules={[
                      { required: true, message: '请输入密码!' },
                      { min: 6, message: '密码至少6位!' }
                    ]}
                  >
                    <Input.Password
                      prefix={<LockOutlined />}
                      placeholder="密码"
                      onChange={() => setLdapValidationStatus(null)}
                    />
                  </Form.Item>

                  <Form.Item
                    name="confirmPassword"
                    dependencies={['password']}
                    rules={[
                      { required: true, message: '请确认密码!' },
                      ({ getFieldValue }) => ({
                        validator(_, value) {
                          if (!value || getFieldValue('password') === value) {
                            return Promise.resolve();
                          }
                          return Promise.reject(new Error('两次输入的密码不一致!'));
                        },
                      }),
                    ]}
                  >
                    <Input.Password
                      prefix={<LockOutlined />}
                      placeholder="确认密码"
                    />
                  </Form.Item>

                  <Form.Item
                    name="department"
                    rules={[{ required: true, message: '请输入部门!' }]}
                  >
                    <Input
                      prefix={<TeamOutlined />}
                      placeholder="部门"
                    />
                  </Form.Item>

                  <Form.Item
                    name="role_template"
                    rules={[{ required: true, message: '请选择角色模板!' }]}
                  >
                    <Select
                      placeholder="选择角色模板"
                      onChange={handleRoleTemplateChange}
                      size="large"
                    >
                      {Object.entries(ROLE_TEMPLATES || {}).map(([key, template]) => (
                        <Option key={key} value={key}>
                          {template.name}
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
                      需要管理员审批（推荐）
                    </Checkbox>
                  </Form.Item>

                  {ldapValidationStatus === 'validating' && (
                    <Alert
                      message="正在验证LDAP..."
                      description="正在验证您的LDAP账户信息，请稍候..."
                      type="info"
                      showIcon
                      style={{ marginBottom: 16 }}
                    />
                  )}

                  {ldapValidationStatus === 'success' && (
                    <Alert
                      message="LDAP验证成功"
                      description="您的LDAP账户验证通过，可以继续注册。"
                      type="success"
                      showIcon
                      icon={<CheckCircleOutlined />}
                      style={{ marginBottom: 16 }}
                    />
                  )}

                  {ldapValidationStatus === 'error' && (
                    <Alert
                      message="LDAP验证失败"
                      description="请检查用户名和密码，或联系管理员确认您在LDAP系统中的账户状态。"
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
                      {requiresApproval ? '提交注册申请' : '注册'}
                    </Button>
                  </Form.Item>
                </Form>

                <div style={{ marginTop: 16, padding: 16, backgroundColor: '#f9f9f9', borderRadius: 4 }}>
                  <h4 style={{ marginBottom: 8 }}>注册流程说明：</h4>
                  <ol style={{ margin: 0, paddingLeft: 20, fontSize: '12px', color: '#666' }}>
                    <li>输入LDAP账户信息进行验证</li>
                    <li>选择适合您的角色模板</li>
                    <li>提交注册申请等待管理员审批</li>
                    <li>审批通过后即可登录使用系统</li>
                  </ol>
                </div>
              </TabPane>
            </Tabs>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default AuthPage;
