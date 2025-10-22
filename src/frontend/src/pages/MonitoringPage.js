import React, { useState, useEffect } from 'react';
import { Card, Spin, Alert, Button } from 'antd';
import { ReloadOutlined, FullscreenOutlined } from '@ant-design/icons';
import '../App.css';

/**
 * MonitoringPage - 监控仪表板页面
 * 使用 iframe 嵌入 Nightingale 监控系统
 * 通过 ProxyAuth 实现单点登录
 */
const MonitoringPage = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [iframeKey, setIframeKey] = useState(0);

  // Nightingale 服务地址 - 使用环境变量或动态构建
  // 支持完整URL或仅端口号配置，默认使用 nginx 代理路径
  const getNightingaleUrl = () => {
    // 优先使用完整的 URL 配置
    if (process.env.REACT_APP_NIGHTINGALE_URL) {
      return process.env.REACT_APP_NIGHTINGALE_URL;
    }
    
    // 如果配置了端口，使用直接端口访问
    if (process.env.REACT_APP_NIGHTINGALE_PORT) {
      const port = process.env.REACT_APP_NIGHTINGALE_PORT;
      return `${window.location.protocol}//${window.location.hostname}:${port}`;
    }
    
    // 默认使用 nginx 代理路径（推荐，支持 ProxyAuth SSO）
    const currentPort = window.location.port ? `:${window.location.port}` : '';
    return `${window.location.protocol}//${window.location.hostname}${currentPort}/nightingale/`;
  };
  
  const nightingaleUrl = getNightingaleUrl();

  useEffect(() => {
    // 组件挂载时的初始化
    console.log('MonitoringPage mounted, Nightingale URL:', nightingaleUrl);
    
    // 设置超时检测iframe加载
    const timeout = setTimeout(() => {
      if (loading) {
        setLoading(false);
        setError('监控系统加载超时，请检查网络连接');
      }
    }, 15000); // 15秒超时

    return () => clearTimeout(timeout);
  }, [nightingaleUrl, loading]);

  // iframe 加载完成处理
  const handleIframeLoad = () => {
    console.log('Nightingale iframe loaded successfully');
    setLoading(false);
    setError(null);
  };

  // iframe 加载错误处理
  const handleIframeError = () => {
    console.error('Failed to load Nightingale iframe');
    setLoading(false);
    setError('无法加载监控系统，请检查 Nightingale 服务是否正常运行');
  };

  // 刷新 iframe
  const handleRefresh = () => {
    setLoading(true);
    setError(null);
    setIframeKey(prev => prev + 1);
  };

  // 全屏打开
  const handleFullscreen = () => {
    window.open(nightingaleUrl, '_blank');
  };

  return (
    <div style={{ padding: '24px', height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Card 
        title="监控仪表板" 
        style={{ flex: 1, display: 'flex', flexDirection: 'column' }}
        bodyStyle={{ flex: 1, padding: 0, display: 'flex', flexDirection: 'column' }}
        extra={
          <div>
            <Button 
              icon={<ReloadOutlined />} 
              onClick={handleRefresh}
              style={{ marginRight: 8 }}
            >
              刷新
            </Button>
            <Button 
              icon={<FullscreenOutlined />} 
              onClick={handleFullscreen}
            >
              新窗口打开
            </Button>
          </div>
        }
      >
        {error && (
          <Alert
            message="加载错误"
            description={error}
            type="error"
            showIcon
            closable
            onClose={() => setError(null)}
            style={{ margin: '16px' }}
          />
        )}

        {loading && (
          <div style={{ 
            display: 'flex', 
            justifyContent: 'center', 
            alignItems: 'center', 
            height: '100%',
            flexDirection: 'column'
          }}>
            <Spin size="large" />
            <div style={{ marginTop: 16, color: '#999' }}>
              正在加载监控系统...
            </div>
          </div>
        )}

        <iframe
          key={iframeKey}
          src={nightingaleUrl}
          title="Nightingale Monitoring"
          onLoad={handleIframeLoad}
          onError={handleIframeError}
          style={{
            width: '100%',
            height: '100%',
            border: 'none',
            display: loading ? 'none' : 'block'
          }}
          allow="fullscreen"
        />
      </Card>
    </div>
  );
};

export default MonitoringPage;
