import React, { useState } from 'react';
import { Form, Input, Button, Card, message, Tabs, Row, Col } from 'antd';
import { UserOutlined, LockOutlined, MailOutlined } from '@ant-design/icons';
import { authAPI } from '../services/api';
import './Auth.css';

const { TabPane } = Tabs;

const AuthPage = ({ onLogin }) => {
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('login');

  const handleLogin = async (values) => {
    setLoading(true);
    try {
      const response = await authAPI.login(values);
      const { token, user, expires_at } = response.data;
      
      console.log('=== 登录API调用成功 ===');
      console.log('获得token:', token ? '是' : '否');
      console.log('登录用户:', user);
      
      // 保存token
      localStorage.setItem('token', token);
      localStorage.setItem('token_expires', expires_at);
      
      // 确保localStorage写入完成
      await new Promise(resolve => setTimeout(resolve, 100));
      
      message.success('登录成功！正在加载权限信息...');
      
      // 传递完整的登录响应数据，包括token信息
      onLogin({
        token,
        expires_at,
        user
      });
      
    } catch (error) {
      console.error('登录失败:', error);
      message.error(error.response?.data?.error || '登录失败，请检查用户名和密码');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (values) => {
    setLoading(true);
    try {
      await authAPI.register(values);
      message.success('注册成功！请登录');
      setActiveTab('login');
    } catch (error) {
      message.error(error.response?.data?.error || '注册失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-container">
      <Row justify="center" align="middle" style={{ minHeight: '100vh' }}>
        <Col xs={22} sm={16} md={12} lg={8} xl={6}>
          <Card title={
            <div style={{ textAlign: 'center' }}>
              <h2>AI-Infra-Matrix</h2>
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

                  <Form.Item>
                    <Button type="primary" htmlType="submit" loading={loading} block>
                      注册
                    </Button>
                  </Form.Item>
                </Form>
              </TabPane>
            </Tabs>
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default AuthPage;
