import React, { useState, useEffect } from 'react';
import {
  Card,
  Button,
  Form,
  Input,
  Alert,
  Space,
  Tag,
  Descriptions,
  Modal,
  message,
  Switch,
  Table,
  Tooltip
} from 'antd';
import {
  KeyOutlined,
  UserOutlined,
  SecurityScanOutlined,
  SyncOutlined,
  SettingOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined
} from '@ant-design/icons';

const JupyterHubAuthManager = () => {
  const [loading, setLoading] = useState(false);
  const [authConfig, setAuthConfig] = useState({
    backend_url: process.env.REACT_APP_AI_INFRA_BACKEND_URL || 'http://localhost:8080',
    api_token: '',
    auto_login: true,
    token_refresh: true,
    admin_users: ['admin', 'jupyter-admin']
  });
  const [authStatus, setAuthStatus] = useState(null);
  const [tokenInfo, setTokenInfo] = useState(null);
  const [users, setUsers] = useState([]);
  const [showTokenModal, setShowTokenModal] = useState(false);
  const [testToken, setTestToken] = useState('');

  // 检查认证状态
  const checkAuthStatus = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/auth/status');
      const data = await response.json();
      setAuthStatus(data);
    } catch (error) {
      console.error('检查认证状态失败:', error);
      setAuthStatus({ connected: false, error: error.message });
    }
    setLoading(false);
  };

  // 验证token
  const verifyToken = async (token) => {
    try {
      const response = await fetch('/api/auth/verify-token', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ token })
      });
      
      const data = await response.json();
      if (data.valid) {
        setTokenInfo(data);
        message.success('Token验证成功');
      } else {
        message.error(`Token验证失败: ${data.error}`);
        setTokenInfo(null);
      }
    } catch (error) {
      message.error(`Token验证失败: ${error.message}`);
      setTokenInfo(null);
    }
  };

  // 获取用户列表
  const fetchUsers = async () => {
    try {
      const response = await fetch('/api/users', {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        setUsers(data);
      }
    } catch (error) {
      console.error('获取用户列表失败:', error);
    }
  };

  // 测试JupyterHub登录
  const testJupyterHubLogin = async (credentials) => {
    setLoading(true);
    try {
      const response = await fetch('/api/auth/jupyterhub-login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(credentials)
      });
      
      const data = await response.json();
      if (data.success) {
        message.success('JupyterHub登录测试成功');
        return data;
      } else {
        message.error(`登录测试失败: ${data.error}`);
        return null;
      }
    } catch (error) {
      message.error(`登录测试失败: ${error.message}`);
      return null;
    } finally {
      setLoading(false);
    }
  };

  // 刷新token
  const refreshToken = async (token) => {
    try {
      const response = await fetch('/api/auth/refresh-token', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ token })
      });
      
      const data = await response.json();
      if (data.success) {
        message.success('Token刷新成功');
        return data.token;
      } else {
        message.error(`Token刷新失败: ${data.error}`);
        return null;
      }
    } catch (error) {
      message.error(`Token刷新失败: ${error.message}`);
      return null;
    }
  };

  useEffect(() => {
    checkAuthStatus();
    fetchUsers();
  }, []);

  const userColumns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (text) => <Tag icon={<UserOutlined />}>{text}</Tag>
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email'
    },
    {
      title: '认证源',
      dataIndex: 'auth_source',
      key: 'auth_source',
      render: (source) => (
        <Tag color={source === 'local' ? 'blue' : 'green'}>
          {source === 'local' ? '本地' : 'LDAP'}
        </Tag>
      )
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (active) => (
        <Tag color={active ? 'success' : 'error'}>
          {active ? '活跃' : '禁用'}
        </Tag>
      )
    },
    {
      title: '最后登录',
      dataIndex: 'last_login',
      key: 'last_login',
      render: (time) => time ? new Date(time).toLocaleString() : '从未登录'
    }
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* 标题和操作 */}
        <Card>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <h2 style={{ margin: 0 }}>
                <SecurityScanOutlined style={{ marginRight: 8 }} />
                JupyterHub 统一认证管理
              </h2>
              <p style={{ margin: '8px 0 0 0', color: '#666' }}>
                管理JupyterHub与AI基础设施矩阵的统一身份认证
              </p>
            </div>
            <Space>
              <Button icon={<SyncOutlined />} onClick={checkAuthStatus} loading={loading}>
                刷新状态
              </Button>
              <Button 
                type="primary" 
                icon={<KeyOutlined />} 
                onClick={() => setShowTokenModal(true)}
              >
                测试Token
              </Button>
            </Space>
          </div>
        </Card>

        {/* 认证状态 */}
        <Card title="认证系统状态" extra={
          <Tag color={authStatus?.connected ? 'success' : 'error'}>
            {authStatus?.connected ? '已连接' : '连接失败'}
          </Tag>
        }>
          {authStatus ? (
            <Descriptions column={2} size="small">
              <Descriptions.Item label="后端API">
                <Space>
                  {authConfig.backend_url}
                  {authStatus.connected ? (
                    <CheckCircleOutlined style={{ color: '#52c41a' }} />
                  ) : (
                    <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
                  )}
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="JupyterHub端口">8090</Descriptions.Item>
              <Descriptions.Item label="认证器">AI基础设施矩阵统一认证</Descriptions.Item>
              <Descriptions.Item label="自动登录">
                <Tag color={authConfig.auto_login ? 'success' : 'default'}>
                  {authConfig.auto_login ? '启用' : '禁用'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Token自动刷新">
                <Tag color={authConfig.token_refresh ? 'success' : 'default'}>
                  {authConfig.token_refresh ? '启用' : '禁用'}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="管理员用户">
                <Space wrap>
                  {authConfig.admin_users.map(user => (
                    <Tag key={user} color="orange">{user}</Tag>
                  ))}
                </Space>
              </Descriptions.Item>
            </Descriptions>
          ) : (
            <Alert 
              message="正在检查认证状态..." 
              type="info" 
              showIcon 
              icon={<SyncOutlined spin />} 
            />
          )}
        </Card>

        {/* 配置管理 */}
        <Card title="认证配置" extra={
          <Button icon={<SettingOutlined />} type="link">
            编辑配置
          </Button>
        }>
          <Form layout="vertical">
            <Form.Item label="后端API地址">
              <Input value={authConfig.backend_url} disabled />
            </Form.Item>
            <Space>
              <Form.Item label="自动登录">
                <Switch 
                  checked={authConfig.auto_login} 
                  onChange={(checked) => setAuthConfig({...authConfig, auto_login: checked})}
                />
              </Form.Item>
              <Form.Item label="Token自动刷新">
                <Switch 
                  checked={authConfig.token_refresh} 
                  onChange={(checked) => setAuthConfig({...authConfig, token_refresh: checked})}
                />
              </Form.Item>
            </Space>
          </Form>
        </Card>

        {/* 用户管理 */}
        <Card title="用户管理" extra={
          <Button onClick={fetchUsers} icon={<SyncOutlined />}>
            刷新用户列表
          </Button>
        }>
          <Table 
            columns={userColumns}
            dataSource={users}
            rowKey="id"
            size="small"
            pagination={{ pageSize: 10 }}
          />
        </Card>

        {/* Token测试工具 */}
        <Card title="认证测试工具">
          <Space direction="vertical" style={{ width: '100%' }}>
            <Alert
              message="测试功能"
              description="使用此工具测试JWT token验证、用户登录和token刷新功能"
              type="info"
              showIcon
            />
            <Form 
              onFinish={testJupyterHubLogin}
              layout="inline"
            >
              <Form.Item 
                name="username" 
                rules={[{ required: true, message: '请输入用户名' }]}
              >
                <Input placeholder="用户名" prefix={<UserOutlined />} />
              </Form.Item>
              <Form.Item 
                name="password" 
                rules={[{ required: true, message: '请输入密码' }]}
              >
                <Input.Password placeholder="密码" prefix={<KeyOutlined />} />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" loading={loading}>
                  测试登录
                </Button>
              </Form.Item>
            </Form>
          </Space>
        </Card>

        {/* Token信息显示 */}
        {tokenInfo && (
          <Card title="Token信息">
            <Descriptions column={1} size="small">
              <Descriptions.Item label="用户">
                {tokenInfo.user.username} ({tokenInfo.user.email})
              </Descriptions.Item>
              <Descriptions.Item label="角色">
                <Space wrap>
                  {tokenInfo.user.roles.map(role => (
                    <Tag key={role} color="blue">{role}</Tag>
                  ))}
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="过期时间">
                {tokenInfo.expires_at}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        )}
      </Space>

      {/* Token测试弹窗 */}
      <Modal
        title="Token验证测试"
        open={showTokenModal}
        onCancel={() => setShowTokenModal(false)}
        footer={[
          <Button key="cancel" onClick={() => setShowTokenModal(false)}>
            取消
          </Button>,
          <Button 
            key="verify" 
            type="primary" 
            onClick={() => verifyToken(testToken)}
            disabled={!testToken}
          >
            验证Token
          </Button>,
          <Button 
            key="refresh" 
            onClick={() => refreshToken(testToken)}
            disabled={!testToken}
          >
            刷新Token
          </Button>
        ]}
      >
        <Form layout="vertical">
          <Form.Item label="JWT Token">
            <Input.TextArea
              value={testToken}
              onChange={(e) => setTestToken(e.target.value)}
              placeholder="粘贴JWT token..."
              rows={4}
            />
          </Form.Item>
        </Form>
        {tokenInfo && (
          <Alert
            message="Token验证成功"
            description={`用户: ${tokenInfo.user.username}, 过期时间: ${tokenInfo.expires_at}`}
            type="success"
            showIcon
            style={{ marginTop: 16 }}
          />
        )}
      </Modal>
    </div>
  );
};

export default JupyterHubAuthManager;
