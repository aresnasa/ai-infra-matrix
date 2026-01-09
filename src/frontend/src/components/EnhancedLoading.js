import React from 'react';
import { Spin, Alert, Progress, Typography, Space, Tag, theme } from 'antd';
import { 
  LoadingOutlined, 
  CheckCircleOutlined, 
  ExclamationCircleOutlined,
  CloseCircleOutlined,
  WifiOutlined
} from '@ant-design/icons';

const { Text } = Typography;
const { useToken } = theme;

/**
 * 增强的加载组件
 * 显示详细的加载状态和API健康信息
 */
const EnhancedLoading = ({ 
  loading = true, 
  error = null, 
  apiHealth = null,
  loadingText = '正在加载...',
  progress = null,
  showAPIStatus = true,
  children 
}) => {
  const { token } = useToken();
  
  // API状态指示器
  const renderAPIStatus = () => {
    if (!showAPIStatus || !apiHealth) return null;

    const getStatusIcon = () => {
      switch (apiHealth.status) {
        case 'healthy':
          return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
        case 'degraded':
          return <ExclamationCircleOutlined style={{ color: '#faad14' }} />;
        case 'down':
          return <CloseCircleOutlined style={{ color: '#ff4d4f' }} />;
        default:
          return <WifiOutlined style={{ color: '#d9d9d9' }} />;
      }
    };

    const getStatusText = () => {
      switch (apiHealth.status) {
        case 'healthy':
          return '服务正常';
        case 'degraded':
          return '服务异常';
        case 'down':
          return '服务中断';
        default:
          return '检查中';
      }
    };

    const getStatusColor = () => {
      switch (apiHealth.status) {
        case 'healthy':
          return 'success';
        case 'degraded':
          return 'warning';
        case 'down':
          return 'error';
        default:
          return 'default';
      }
    };

    return (
      <div style={{ marginTop: 12 }}>
        <Space size="small">
          {getStatusIcon()}
          <Tag color={getStatusColor()}>{getStatusText()}</Tag>
          {apiHealth.responseTime && (
            <Text type="secondary" style={{ fontSize: '12px' }}>
              响应时间: {apiHealth.responseTime}ms
            </Text>
          )}
        </Space>
        {apiHealth.lastCheck && (
          <div style={{ marginTop: 4 }}>
            <Text type="secondary" style={{ fontSize: '12px' }}>
              最后检查: {apiHealth.lastCheck.toLocaleTimeString()}
            </Text>
          </div>
        )}
      </div>
    );
  };

  // 错误显示
  if (error) {
    return (
      <div style={{
        padding: '40px 20px',
        textAlign: 'center',
        maxWidth: '500px',
        margin: '0 auto'
      }}>
        <Alert
          type="error"
          showIcon
          message="加载失败"
          description={
            <div>
              <div>{error.message || '发生未知错误'}</div>
              {error.response && (
                <div style={{ marginTop: 8 }}>
                  <Text type="secondary">
                    状态码: {error.response.status}
                  </Text>
                  {error.response.data?.error && (
                    <div>
                      <Text type="secondary">
                        详细信息: {error.response.data.error}
                      </Text>
                    </div>
                  )}
                </div>
              )}
            </div>
          }
          style={{ textAlign: 'left' }}
        />
        {renderAPIStatus()}
      </div>
    );
  }

  // 加载状态
  if (loading) {
    return (
      <div style={{
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '300px',
        padding: '40px 20px'
      }}>
        <Spin 
          size="large" 
          indicator={<LoadingOutlined style={{ fontSize: 32 }} />}
        />
        <div style={{ marginTop: 16, textAlign: 'center' }}>
          <Text style={{ fontSize: '16px', color: '#666' }}>
            {loadingText}
          </Text>
          {progress !== null && (
            <div style={{ marginTop: 12, width: '200px' }}>
              <Progress 
                percent={progress} 
                size="small" 
                showInfo={false}
                strokeColor={{
                  '0%': '#108ee9',
                  '100%': '#87d068',
                }}
              />
              <Text type="secondary" style={{ fontSize: '12px' }}>
                {progress}%
              </Text>
            </div>
          )}
        </div>
        {renderAPIStatus()}
      </div>
    );
  }

  // 正常渲染子组件
  return (
    <div>
      {children}
      {showAPIStatus && apiHealth && apiHealth.status !== 'healthy' && (
        <div style={{
          position: 'fixed',
          bottom: 16,
          right: 16,
          zIndex: 1000,
          background: token.colorBgContainer,
          padding: '8px 12px',
          borderRadius: '6px',
          boxShadow: token.boxShadow,
          border: `1px solid ${token.colorBorder}`
        }}>
          {renderAPIStatus()}
        </div>
      )}
    </div>
  );
};

export default EnhancedLoading;
