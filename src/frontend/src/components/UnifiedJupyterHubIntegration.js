// 统一认证JupyterHub集成组件
import React, { useState, useEffect } from 'react';
import { 
  Card, 
  Button, 
  message, 
  Space, 
  Typography, 
  Tag, 
  Spin, 
  Alert,
  Modal,
  Form,
  Input,
  Tooltip,
  Row,
  Col
} from 'antd';
import { 
  RocketOutlined, 
  UserOutlined, 
  SettingOutlined,
  LoginOutlined,
  LogoutOutlined,
  InfoCircleOutlined,
  ReloadOutlined
} from '@ant-design/icons';
import api from '../services/api';

const { Title, Text, Paragraph } = Typography;

const UnifiedJupyterHubIntegration = () => {
  const [loading, setLoading] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState('unknown');
  const [jupyterHubInfo, setJupyterHubInfo] = useState(null);
  const [userSession, setUserSession] = useState(null);
  const [showLoginModal, setShowLoginModal] = useState(false);
  const [loginForm] = Form.useForm();

  // JupyterHub 配置 - 通过 Nginx 统一入口访问
  const jupyterHubConfig = {
    url: process.env.REACT_APP_JUPYTERHUB_URL || `${window.location.origin}/jupyter`,
    apiUrl: process.env.REACT_APP_API_URL || '/api'
  };

  useEffect(() => {
    checkJupyterHubStatus();
    checkUserSession();
  }, []);

  // 检查JupyterHub状态
  const checkJupyterHubStatus = async () => {
    setLoading(true);
    try {
      // 通过后端API检查JupyterHub状态
      const response = await api.get('/jupyterhub/status');
      setJupyterHubInfo(response.data);
      setConnectionStatus('connected');
    } catch (error) {
      console.error('检查JupyterHub状态失败:', error);
      setConnectionStatus('disconnected');
      // 尝试直接连接JupyterHub
      try {
        const directResponse = await fetch(`${jupyterHubConfig.url}/hub/api/info`);
        if (directResponse.ok) {
          const info = await directResponse.json();
          setJupyterHubInfo(info);
          setConnectionStatus('connected');
        }
      } catch (directError) {
        console.error('直接连接JupyterHub失败:', directError);
        setConnectionStatus('error');
      }
    } finally {
      setLoading(false);
    }
  };

  // 检查用户会话
  const checkUserSession = async () => {
    try {
      const response = await api.get('/auth/user');
      if (response.data && response.data.username) {
        setUserSession(response.data);
        // 检查JupyterHub会话
        checkJupyterHubSession(response.data.username);
      }
    } catch (error) {
      console.error('检查用户会话失败:', error);
    }
  };

  // 检查JupyterHub会话
  const checkJupyterHubSession = async (username) => {
    try {
      const response = await fetch(`${jupyterHubConfig.url}/hub/api/user-session`, {
        credentials: 'include'
      });
      if (response.ok) {
        const sessionData = await response.json();
        setUserSession(prev => ({
          ...prev,
          jupyterhub_session: sessionData
        }));
      }
    } catch (error) {
      console.error('检查JupyterHub会话失败:', error);
    }
  };

  // 统一认证登录到JupyterHub
  const handleUnifiedLogin = async () => {
    if (!userSession) {
      message.warning('请先登录系统');
      return;
    }

    setLoading(true);
    try {
      // 从localStorage获取JWT token
      const token = localStorage.getItem('token');
      if (!token) {
        message.error('未找到有效的登录令牌，请重新登录系统');
        return;
      }

      // 直接使用JWT token登录JupyterHub
      const loginUrl = `${jupyterHubConfig.url}/hub/login?next=%2Fhub%2F&jwt_token=${encodeURIComponent(token)}`;
      window.open(loginUrl, '_blank');
      message.success('正在使用当前登录状态跳转到JupyterHub...');
      
      // 更新会话状态
      setTimeout(() => {
        checkJupyterHubSession(userSession.username);
      }, 2000);
    } catch (error) {
      console.error('统一登录失败:', error);
      message.error('登录失败: ' + (error.response?.data?.message || error.message));
    } finally {
      setLoading(false);
    }
  };

  // 直接登录JupyterHub（独立认证）
  const handleDirectLogin = async (values) => {
    setLoading(true);
    try {
      // 直接向JupyterHub提交登录请求
      const formData = new FormData();
      formData.append('username', values.username);
      formData.append('password', values.password);

      const response = await fetch(`${jupyterHubConfig.url}/hub/login`, {
        method: 'POST',
        body: formData,
        credentials: 'include'
      });

      if (response.ok) {
        message.success('登录成功');
        setShowLoginModal(false);
        loginForm.resetFields();
        window.open(`${jupyterHubConfig.url}/hub/`, '_blank');
        checkJupyterHubSession(values.username);
      } else {
        message.error('用户名或密码错误');
      }
    } catch (error) {
      console.error('直接登录失败:', error);
      message.error('登录失败');
    } finally {
      setLoading(false);
    }
  };

  // 启动Notebook服务器
  const startNotebookServer = async () => {
    if (!userSession) {
      message.warning('请先登录');
      return;
    }

    setLoading(true);
    try {
      const response = await api.post('/jupyterhub/start-server', {
        username: userSession.username
      });

      if (response.data.success) {
        const serverUrl = `${jupyterHubConfig.url}/user/${userSession.username}/lab`;
        window.open(serverUrl, '_blank');
        message.success('Notebook服务器启动成功');
      } else {
        message.error('启动服务器失败');
      }
    } catch (error) {
      console.error('启动服务器失败:', error);
      message.error('启动失败: ' + (error.response?.data?.message || error.message));
    } finally {
      setLoading(false);
    }
  };

  // 停止Notebook服务器
  const stopNotebookServer = async () => {
    if (!userSession) return;

    setLoading(true);
    try {
      await api.post('/jupyterhub/stop-server', {
        username: userSession.username
      });
      message.success('Notebook服务器已停止');
      checkUserSession();
    } catch (error) {
      console.error('停止服务器失败:', error);
      message.error('停止失败: ' + (error.response?.data?.message || error.message));
    } finally {
      setLoading(false);
    }
  };

  // 登出所有会话
  const handleLogoutAll = async () => {
    setLoading(true);
    try {
      await api.post('/auth/logout-all');
      // 清除JupyterHub会话
      await fetch(`${jupyterHubConfig.url}/hub/api/logout-all`, {
        method: 'POST',
        credentials: 'include'
      });
      message.success('已登出所有会话');
      setUserSession(null);
    } catch (error) {
      console.error('登出失败:', error);
      message.error('登出失败');
    } finally {
      setLoading(false);
    }
  };

  // 渲染连接状态
  const renderConnectionStatus = () => {
    const statusConfig = {
      connected: { color: 'success', text: '已连接' },
      disconnected: { color: 'warning', text: '未连接' },
      error: { color: 'error', text: '连接错误' },
      unknown: { color: 'default', text: '未知' }
    };

    const config = statusConfig[connectionStatus] || statusConfig.unknown;
    
    return (
      <Tag color={config.color} icon={<InfoCircleOutlined />}>
        {config.text}
      </Tag>
    );
  };

  return (
    <div style={{ padding: '24px' }}>
      <Card
        title={
          <Space>
            <RocketOutlined />
            <Title level={4} style={{ margin: 0 }}>统一认证JupyterHub集成</Title>
          </Space>
        }
        extra={
          <Space>
            {renderConnectionStatus()}
            <Button 
              icon={<ReloadOutlined />} 
              onClick={checkJupyterHubStatus}
              loading={loading}
            >
              刷新状态
            </Button>
          </Space>
        }
        style={{ marginBottom: '24px' }}
      >
        <Row gutter={[24, 24]}>
          <Col xs={24} lg={12}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Alert
                message="统一认证系统"
                description="使用与后端系统相同的用户账户直接登录JupyterHub，无需单独注册。"
                type="info"
                showIcon
              />
              
              {jupyterHubInfo && (
                <Card size="small" title="JupyterHub信息">
                  <p><strong>版本:</strong> {jupyterHubInfo.version}</p>
                  <p><strong>URL:</strong> {jupyterHubConfig.url}</p>
                  <p><strong>认证方式:</strong> PostgreSQL + Redis</p>
                </Card>
              )}
            </Space>
          </Col>
          
          <Col xs={24} lg={12}>
            <Space direction="vertical" style={{ width: '100%' }}>
              {userSession ? (
                <Card size="small" title="用户会话">
                  <p><strong>用户名:</strong> {userSession.username}</p>
                  <p><strong>邮箱:</strong> {userSession.email}</p>
                  {userSession.roles && (
                    <p><strong>角色:</strong> {userSession.roles.join(', ')}</p>
                  )}
                  {userSession.jupyterhub_session && (
                    <Tag color="green">JupyterHub已登录</Tag>
                  )}
                </Card>
              ) : (
                <Alert
                  message="未登录"
                  description="请先登录系统以使用统一认证功能"
                  type="warning"
                  showIcon
                />
              )}
            </Space>
          </Col>
        </Row>
      </Card>

      {/* 操作按钮区域 */}
      <Card title="操作中心">
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Button
              type="primary"
              size="large"
              block
              icon={<LoginOutlined />}
              onClick={handleUnifiedLogin}
              disabled={!userSession || connectionStatus !== 'connected'}
              loading={loading}
            >
              统一认证登录
            </Button>
          </Col>
          
          <Col xs={24} sm={12} lg={6}>
            <Button
              size="large"
              block
              icon={<UserOutlined />}
              onClick={() => setShowLoginModal(true)}
              disabled={connectionStatus !== 'connected'}
            >
              独立登录
            </Button>
          </Col>
          
          <Col xs={24} sm={12} lg={6}>
            <Button
              type="default"
              size="large"
              block
              icon={<RocketOutlined />}
              onClick={startNotebookServer}
              disabled={!userSession || connectionStatus !== 'connected'}
              loading={loading}
            >
              启动Notebook
            </Button>
          </Col>
          
          <Col xs={24} sm={12} lg={6}>
            <Button
              size="large"
              block
              icon={<SettingOutlined />}
              onClick={() => window.open(`${jupyterHubConfig.url}/hub/admin`, '_blank')}
              disabled={!userSession || !userSession.roles?.includes('admin')}
            >
              管理界面
            </Button>
          </Col>
        </Row>
        
        <Row gutter={[16, 16]} style={{ marginTop: '16px' }}>
          <Col xs={24} sm={12}>
            <Button
              danger
              size="large"
              block
              icon={<LogoutOutlined />}
              onClick={handleLogoutAll}
              disabled={!userSession}
              loading={loading}
            >
              登出所有会话
            </Button>
          </Col>
          
          <Col xs={24} sm={12}>
            <Button
              size="large"
              block
              onClick={() => window.open(`${jupyterHubConfig.url}/hub/`, '_blank')}
              disabled={connectionStatus !== 'connected'}
            >
              直接访问JupyterHub
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 独立登录模态框 */}
      <Modal
        title="JupyterHub独立登录"
        open={showLoginModal}
        onCancel={() => setShowLoginModal(false)}
        footer={null}
      >
        <Form
          form={loginForm}
          layout="vertical"
          onFinish={handleDirectLogin}
        >
          <Form.Item
            label="用户名"
            name="username"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input />
          </Form.Item>
          
          <Form.Item
            label="密码"
            name="password"
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password />
          </Form.Item>
          
          <Form.Item>
            <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
              <Button onClick={() => setShowLoginModal(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={loading}>
                登录
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default UnifiedJupyterHubIntegration;
