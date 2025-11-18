import React, { useState, useEffect, useRef, memo } from 'react';
import { Spin, Alert, Button, Card, Typography } from 'antd';
import { ReloadOutlined, ExclamationCircleOutlined } from '@ant-design/icons';

const { Title, Text } = Typography;

/**
 * 通用懒加载高阶组件 - 优化版本
 * 提供统一的加载状态、错误处理和重试机制
 * 减少不必要的重新渲染和API调用
 */
const withLazyLoading = (WrappedComponent, options = {}) => {
  const {
    loadingText = '正在加载...',
    errorTitle = '页面加载失败',
    retryText = '重试',
    showRefreshHint = true,
    cacheComponent = true // 是否缓存组件实例
  } = options;

  const LazyLoadedComponent = memo(function LazyLoadedComponent(props) {
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);
    const [retryCount, setRetryCount] = useState(0);
    const [componentReady, setComponentReady] = useState(false);
    const componentRef = useRef(null);
    const mountedRef = useRef(false);

    useEffect(() => {
      mountedRef.current = true;
      
      // 优化：减少加载时间
      const timer = setTimeout(() => {
        if (mountedRef.current) {
          setLoading(false);
          setComponentReady(true);
        }
      }, 100); // 减少到100ms

      return () => {
        clearTimeout(timer);
        mountedRef.current = false;
      };
    }, [retryCount]);

    const handleRetry = () => {
      if (!mountedRef.current) return;
      
      setError(null);
      setLoading(true);
      setComponentReady(false);
      setRetryCount(prev => prev + 1);
    };

    const handleError = (error) => {
      if (!mountedRef.current) return;
      
      console.error('页面加载错误:', error);
      setError(error);
      setLoading(false);
      setComponentReady(false);
    };

    // 错误边界处理
    useEffect(() => {
      const handleUnhandledRejection = (event) => {
        if (mountedRef.current) {
          handleError(new Error(event.reason));
        }
      };

      window.addEventListener('unhandledrejection', handleUnhandledRejection);
      
      return () => {
        window.removeEventListener('unhandledrejection', handleUnhandledRejection);
      };
    }, []);

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
        <div style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          minHeight: '400px',
          padding: '20px'
        }}>
          <Card style={{ maxWidth: 500, textAlign: 'center' }}>
            <ExclamationCircleOutlined 
              style={{ 
                fontSize: 48, 
                color: '#ff4d4f', 
                marginBottom: 16 
              }} 
            />
            <Title level={4}>{errorTitle}</Title>
            <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
              {error.message || '组件加载失败，请稍后重试'}
            </Text>
            <div style={{ marginBottom: 16 }}>
              <Button 
                type="primary" 
                icon={<ReloadOutlined />} 
                onClick={handleRetry}
                loading={loading}
              >
                {retryText}
              </Button>
            </div>
            {showRefreshHint && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                如果问题持续存在，请尝试刷新页面
              </Text>
            )}
          </Card>
        </div>
      );
    }

    if (!componentReady) {
      return (
        <div style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          minHeight: '200px'
        }}>
          <Spin />
        </div>
      );
    }

    try {
      return (
        <div ref={componentRef}>
          <WrappedComponent {...props} />
        </div>
      );
    } catch (renderError) {
      handleError(renderError);
      return null;
    }
  });

  // 设置显示名称，便于调试
  LazyLoadedComponent.displayName = `withLazyLoading(${WrappedComponent.displayName || WrappedComponent.name || 'Component'})`;

  return LazyLoadedComponent;
};

export default withLazyLoading;
