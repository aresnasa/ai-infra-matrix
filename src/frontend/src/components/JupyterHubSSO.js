import React, { useEffect, useState } from 'react';
import { Card, Button, message, Space, Typography, Alert, Spin } from 'antd';
import { PlayCircleOutlined, LinkOutlined, WarningOutlined } from '@ant-design/icons';

const { Title, Paragraph, Text } = Typography;

const JupyterHubSSO = () => {
  const [loading, setLoading] = useState(false);
  const [ssoStatus, setSSO状态] = useState('idle'); // idle, checking, success, error
  const [errorMessage, setErrorMessage] = useState('');

  const checkSSOStatus = async () => {
    try {
      // 检查认证服务是否可用
      if (!window.authService) {
        throw new Error('认证服务未初始化');
      }

      const token = window.authService.getToken();
      if (!token) {
        throw new Error('未找到认证token，请先登录');
      }

      // 验证token有效性
      const verification = await window.authService.verifyToken();
      if (!verification.valid) {
        throw new Error('认证已过期，请重新登录');
      }

      return true;
    } catch (error) {
      setErrorMessage(error.message);
      return false;
    }
  };

  const handleJupyterHubAccess = async () => {
    setLoading(true);
    setSSO状态('checking');
    
    try {
      // 检查认证状态
      const isAuthenticated = await checkSSOStatus();
      if (!isAuthenticated) {
        setSSO状态('error');
        return;
      }

      setSSO状态('success');
      message.success('正在跳转到JupyterHub...');

      // 使用认证服务跳转（会自动设置SSO状态）
      if (window.authService) {
        await window.authService.goToJupyterHub('/');
      } else {
        // 降级方案：直接跳转
        window.location.href = '/jupyter/hub/';
      }

    } catch (error) {
      console.error('JupyterHub访问失败:', error);
      setErrorMessage(error.message);
      setSSO状态('error');
      message.error(`访问失败: ${error.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleSSOBridge = () => {
    // 跳转到SSO桥接页面
    window.location.href = '/sso?next=' + encodeURIComponent('/jupyter/hub/');
  };

  const renderSSO状态 = () => {
    switch (ssoStatus) {
      case 'checking':
        return (
          <Alert
            type="info"
            message="正在验证认证状态..."
            description="请稍候，正在准备单点登录..."
            showIcon
            icon={<Spin size="small" />}
          />
        );
      case 'success':
        return (
          <Alert
            type="success"
            message="认证验证成功"
            description="即将跳转到JupyterHub，您无需重新登录"
            showIcon
          />
        );
      case 'error':
        return (
          <Alert
            type="error"
            message="认证验证失败"
            description={errorMessage}
            showIcon
            action={
              <Button size="small" onClick={() => setSSO状态('idle')}>
                重试
              </Button>
            }
          />
        );
      default:
        return null;
    }
  };

  useEffect(() => {
    // 页面加载时自动检查认证状态
    checkSSOStatus();
  }, []);

  return (
    <Card
      title={
        <Space>
          <PlayCircleOutlined />
          <Title level={4} style={{ margin: 0 }}>JupyterHub 访问中心</Title>
        </Space>
      }
      style={{ marginBottom: 24 }}
    >
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <div>
          <Paragraph>
            <Text strong>JupyterHub</Text> 是您的在线数据科学和机器学习工作台。
            通过单点登录，您可以直接访问个人的Jupyter笔记本环境。
          </Paragraph>
          
          <Paragraph type="secondary">
            支持Python、R、Julia等多种编程环境，预装了常用的数据科学库。
          </Paragraph>
        </div>

        {renderSSO状态()}

        <Space size="middle">
          <Button
            type="primary"
            size="large"
            icon={<LinkOutlined />}
            loading={loading}
            onClick={handleJupyterHubAccess}
            disabled={ssoStatus === 'error'}
          >
            访问 JupyterHub
          </Button>

          <Button
            size="large"
            icon={<WarningOutlined />}
            onClick={handleSSOBridge}
            disabled={loading}
          >
            SSO桥接页面
          </Button>
        </Space>

        <div style={{ marginTop: 16 }}>
          <Text type="secondary" style={{ fontSize: '12px' }}>
            💡 如果遇到登录问题，请尝试使用"SSO桥接页面"或联系管理员
          </Text>
        </div>
      </Space>
    </Card>
  );
};

export default JupyterHubSSO;
