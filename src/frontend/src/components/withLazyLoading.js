import React, { useState, useEffect } from 'react';
import { Spin, Alert, Button, Card, Typography } from 'antd';
import { ReloadOutlined, ExclamationCircleOutlined } from '@ant-design/icons';

const { Title, Text } = Typography;

/**
 * 通用懒加载高阶组件
 * 提供统一的加载状态、错误处理和重试机制
 */
const withLazyLoading = (WrappedComponent, options = {}) => {
  const {
    loadingText = '正在加载...',
    errorTitle = '页面加载失败',
    retryText = '重试',
    showRefreshHint = true
  } = options;

  return function LazyLoadedComponent(props) {
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [retryCount, setRetryCount] = useState(0);

    useEffect(() => {
      // 模拟组件加载
      const timer = setTimeout(() => {
        setLoading(false);
      }, 300);

      return () => clearTimeout(timer);
    }, [retryCount]);

    const handleRetry = () => {
      setError(null);
      setLoading(true);
      setRetryCount(prev => prev + 1);
    };

    const handleError = (error) => {
      console.error('页面加载错误:', error);
      setError(error);
      setLoading(false);
    };

    if (loading) {
      return (
        <div style={{
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          alignItems: 'center',
          minHeight: '400px',
          padding: '20px'
        }}>
          <Spin size="large" />
          <Text style={{ marginTop: 16, color: '#666' }}>
            {loadingText}
          </Text>
        </div>
      );
    }

    if (error) {
      return (
        <div style={{ padding: '20px', maxWidth: '600px', margin: '0 auto' }}>
          <Card>
            <div style={{ textAlign: 'center' }}>
              <ExclamationCircleOutlined 
                style={{ fontSize: 48, color: '#ff4d4f', marginBottom: 16 }} 
              />
              <Title level={3}>{errorTitle}</Title>
              <Alert
                type="error"
                message="错误详情"
                description={
                  <div>
                    <Text>{error.message || '未知错误'}</Text>
                    {error.response && (
                      <div style={{ marginTop: 8 }}>
                        <Text type="secondary">
                          状态码: {error.response.status}
                        </Text>
                        {error.response.data?.error && (
                          <div>
                            <Text type="secondary">
                              服务器错误: {error.response.data.error}
                            </Text>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                }
                style={{ marginBottom: 16, textAlign: 'left' }}
              />
              <div>
                <Button 
                  type="primary" 
                  icon={<ReloadOutlined />}
                  onClick={handleRetry}
                  style={{ marginRight: 8 }}
                >
                  {retryText}
                </Button>
                {showRefreshHint && (
                  <Button onClick={() => window.location.reload()}>
                    刷新页面
                  </Button>
                )}
              </div>
              {retryCount > 0 && (
                <Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
                  重试次数: {retryCount}
                </Text>
              )}
            </div>
          </Card>
        </div>
      );
    }

    // 包装组件，传递错误处理函数
    return (
      <WrappedComponent 
        {...props} 
        onError={handleError}
        retryCount={retryCount}
      />
    );
  };
};

export default withLazyLoading;
