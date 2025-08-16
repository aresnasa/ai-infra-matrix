import React, { useEffect, useState } from 'react';
import { Card, Button, message, Space, Typography, Alert, Spin } from 'antd';
import { PlayCircleOutlined, LinkOutlined, WarningOutlined, AppstoreOutlined } from '@ant-design/icons';
import { resolveSSOTarget } from '../utils/ssoTarget';

const { Title, Paragraph, Text } = Typography;

const JupyterHubSSO = () => {
  const [loading, setLoading] = useState(false);
  const [ssoStatus, setSSO状态] = useState('idle'); // idle, checking, success, error
  const [errorMessage, setErrorMessage] = useState('');
  const target = resolveSSOTarget(); // { key, name, nextPath, authenticatedPath }

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

  const handleAccess = async () => {
    setLoading(true);
    setSSO状态('checking');
    
    try {
      // 调用后端API检查认证状态和获取访问权限
      const response = await fetch('/api/jupyter/access', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token') || ''}`
        },
        body: JSON.stringify({
          redirect_uri: target.authenticatedPath,
          source: 'frontend_sso_component'
        })
      });

      const data = await response.json();
      
      if (data.success && data.action === 'authenticated') {
        // 认证成功，可以直接跳转
        setSSO状态('success');
        message.success(`认证成功，正在跳转到${target.name}...`);
        window.location.href = data.redirect_url;
        
      } else if (data.action === 'redirect') {
        // 需要重定向到SSO登录
        setSSO状态('error');
        message.info('需要登录，正在跳转到SSO...');
        window.location.href = data.redirect_url;
        
      } else {
        // 其他错误情况
        throw new Error(data.message || '未知错误');
      }

    } catch (error) {
      console.error('访问失败:', error);
      setErrorMessage(error.message);
      setSSO状态('error');
      message.error(`访问失败: ${error.message}`);
      
      // 出错时也跳转到SSO登录页面
      setTimeout(() => {
        window.location.href = `/sso/?redirect_uri=${encodeURIComponent(target.authenticatedPath)}`;
      }, 1500);
    } finally {
      setLoading(false);
    }
  };

    const handleIframe = () => {
    if (ssoStatus === 'success') {
      // 获取当前token
      const token = localStorage.getItem('token');
      // 使用认证后的地址，并传递token参数
      const base = target.authenticatedPath;
      const iframeSrc = token ? `${base}${base.includes('?') ? '&' : '?'}token=${encodeURIComponent(token)}` : base;
      
      // 创建iframe模态框
      const modal = document.createElement('div');
      modal.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.8);
        z-index: 10000;
        display: flex;
        justify-content: center;
        align-items: center;
      `;

      const iframeContainer = document.createElement('div');
      iframeContainer.style.cssText = `
        position: relative;
        width: 95%;
        height: 95%;
        background: white;
        border-radius: 8px;
        overflow: hidden;
        box-shadow: 0 4px 30px rgba(0, 0, 0, 0.3);
      `;

      const closeButton = document.createElement('button');
      closeButton.innerHTML = '✕';
      closeButton.style.cssText = `
        position: absolute;
        top: 10px;
        right: 15px;
        background: #ff4d4f;
        color: white;
        border: none;
        border-radius: 50%;
        width: 30px;
        height: 30px;
        cursor: pointer;
        font-size: 16px;
        z-index: 10001;
        display: flex;
        align-items: center;
        justify-content: center;
      `;

      const iframe = document.createElement('iframe');
      iframe.src = iframeSrc;
      iframe.style.cssText = `
        width: 100%;
        height: 100%;
        border: none;
      `;

      // 监听iframe消息
  const messageHandler = (event) => {
        if (event.data.type === 'jupyterhub_auth_error') {
          console.error('JupyterHub认证错误:', event.data.message);
          document.body.removeChild(modal);
          window.removeEventListener('message', messageHandler);
        }
      };
      window.addEventListener('message', messageHandler);

      closeButton.onclick = () => {
        document.body.removeChild(modal);
        window.removeEventListener('message', messageHandler);
      };
      modal.onclick = (e) => {
        if (e.target === modal) {
          document.body.removeChild(modal);
          window.removeEventListener('message', messageHandler);
        }
      };

      iframeContainer.appendChild(iframe);
      iframeContainer.appendChild(closeButton);
      modal.appendChild(iframeContainer);
      document.body.appendChild(modal);
    } else {
      message.error('请先完成SSO认证');
    }
  };

  const handleSSOBridge = () => {
    // 跳转到SSO桥接页面
    window.location.href = '/sso?next=' + encodeURIComponent(target.nextPath);
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
            description={`即将跳转到${target.name}，您无需重新登录`}
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
          <Title level={4} style={{ margin: 0 }}>{target.name} 访问中心</Title>
        </Space>
      }
      style={{ marginBottom: 24 }}
    >
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <div>
          {target.key === 'jupyter' ? (
            <>
              <Paragraph>
                <Text strong>JupyterHub</Text> 是您的在线数据科学和机器学习工作台。
                通过单点登录，您可以直接访问个人的Jupyter笔记本环境。
              </Paragraph>
              <Paragraph type="secondary">
                支持Python、R、Julia等多种编程环境，预装了常用的数据科学库。
              </Paragraph>
            </>
          ) : (
            <>
              <Paragraph>
                <Text strong>Gitea</Text> 是轻量级的代码托管平台。
                通过单点登录，您可以直接访问企业内部的Git仓库与协作工具。
              </Paragraph>
              <Paragraph type="secondary">
                支持仓库、Issues、Pull Requests、CI集成等功能，已适配同源内嵌。
              </Paragraph>
            </>
          )}
        </div>

        {renderSSO状态()}

        <Space size="middle">
          <Button
            type="primary"
            size="large"
            icon={<LinkOutlined />}
            loading={loading}
            onClick={handleAccess}
            disabled={ssoStatus === 'error'}
          >
            新窗口访问
          </Button>

          <Button
            type="default"
            size="large"
            icon={<AppstoreOutlined />}
            loading={loading}
            onClick={handleIframe}
            disabled={ssoStatus === 'error'}
          >
            iframe内访问
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
