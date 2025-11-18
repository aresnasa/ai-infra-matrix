import React, { useState, useEffect } from 'react';
import { 
  Card, Button, Space, Typography, Spin, Alert, message, 
  Row, Col, Tag, Breadcrumb, Tooltip
} from 'antd';
import {
  ArrowLeftOutlined, ReloadOutlined, FullscreenOutlined,
  DatabaseOutlined, LinkOutlined, SettingOutlined,
  ExclamationCircleOutlined, CheckCircleOutlined, InfoCircleOutlined
} from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { objectStorageAPI } from '../services/api';
import IframeEmbed from '../components/IframeEmbed';

const { Title, Text } = Typography;

const MinIOConsolePage = () => {
  const navigate = useNavigate();
  const { configId } = useParams();
  
  const [loading, setLoading] = useState(true);
  const [config, setConfig] = useState(null);
  const [connectionStatus, setConnectionStatus] = useState('checking');
  const [iframeUrl, setIframeUrl] = useState('');
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [diagnosticNote, setDiagnosticNote] = useState('');
  const [autoLoginDone, setAutoLoginDone] = useState(false);

  // 尝试对MinIO Console执行同源自动登录（AK/SK）
  const tryMinIOAutoLogin = async (ak, sk, opts = {}) => {
    if (!ak || !sk) return false;
    const { csrfToken } = opts || {};
    const endpoints = [
      '/minio-console/api/v1/login',
      '/minio-console/api/login',
    ];
    // 为兼容不同版本，尝试 JSON 与 x-www-form-urlencoded 两种编码
    const payloads = [
      // JSON 变体
      () => ({ username: ak, password: sk }),
      () => ({ accessKey: ak, secretKey: sk }),
      // 表单变体
      () => new URLSearchParams({ username: ak, password: sk }),
      () => new URLSearchParams({ accessKey: ak, secretKey: sk }),
    ];
    const baseHeaders = {
      'Accept': 'application/json, text/plain, */*',
      'X-Requested-With': 'XMLHttpRequest',
      'Origin': window.location.origin,
      'Referer': `${window.location.origin}/minio-console/`,
    };
    const contentTypes = [
      'application/json',
      'application/x-www-form-urlencoded',
    ];
    for (const ep of endpoints) {
      for (let i = 0; i < payloads.length; i++) {
        const makeBody = payloads[i];
        const isForm = makeBody() instanceof URLSearchParams;
        const headers = { ...baseHeaders, 'Content-Type': isForm ? contentTypes[1] : contentTypes[0] };
        if (csrfToken) headers['X-CSRF-Token'] = csrfToken;
        try {
          const body = isForm ? makeBody() : JSON.stringify(makeBody());
          const resp = await fetch(ep, {
            method: 'POST',
            headers,
            credentials: 'include',
            body
          });
          if (resp.ok) {
            // 尝试读取token并写入localStorage，便于控制台SPA识别
            try {
              const contentType = resp.headers.get('content-type') || '';
              if (contentType.includes('application/json')) {
                const data = await resp.json();
                const token = data?.token || data?.jwt || data?.id || data?.accessToken || data?.session || '';
                if (token) {
                  try {
                    localStorage.setItem('token', token);
                    localStorage.setItem('jwt', token);
                    localStorage.setItem('minio_token', token);
                    localStorage.setItem('minio_jwt', token);
                  } catch (_) {}
                }
              }
            } catch (_) {}
            return true;
          }
        } catch (e) {
          // 忽略，尝试下一个变体
        }
      }
    }
    return false;
  };

  // 优先使用同源代理；若提供了config则先尝试自动登录再设置iframe
  const prepareSameOriginConsole = async (configData) => {
    const sameOriginUrl = `${window.location.origin}/minio-console/`;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 3000);
    try {
      const res = await fetch(sameOriginUrl, {
        method: 'GET',
        cache: 'no-store',
        credentials: 'include',
        redirect: 'manual',
        signal: controller.signal,
      });
      clearTimeout(timeout);
      if (res.ok || (res.status >= 200 && res.status < 400)) {
        // 提取可能的CSRF Token（响应头或HTML内容）
        let csrfToken = res.headers.get('x-csrf-token') || res.headers.get('x-xsrf-token') || '';
        try {
          if (!csrfToken) {
            const html = await res.text();
            // 常见Meta/隐藏字段标记
            const metaMatch = html.match(/<meta[^>]*name=["']csrf-token["'][^>]*content=["']([^"']+)["'][^>]*>/i);
            const meta2Match = html.match(/<meta[^>]*name=["']x-csrf-token["'][^>]*content=["']([^"']+)["'][^>]*>/i);
            const inputMatch = html.match(/name=["'](?:gorilla\.csrf\.Token|csrf|_csrf|xsrf)["'][^>]*value=["']([^"']+)["']/i);
            csrfToken = (metaMatch?.[1] || meta2Match?.[1] || inputMatch?.[1] || '').trim();
          }
        } catch (_) {}
        // 自动登录（如果还未尝试且拿到了配置中的AK/SK）
        if (!autoLoginDone && configData?.access_key && configData?.secret_key) {
          // 先尝试服务端代理设置Cookie（同源，稳定）
          try {
            const proxyResp = await fetch('/api/object-storage/minio/console/proxy-login', {
              method: 'POST',
              headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json',
              },
              credentials: 'include',
              body: JSON.stringify({ access_key: configData.access_key, secret_key: configData.secret_key })
            });
            if (proxyResp.ok) {
              setAutoLoginDone(true);
            } else {
              // 回退到前端直连尝试
              const okLogin = await tryMinIOAutoLogin(configData.access_key, configData.secret_key, { csrfToken });
              if (okLogin) setAutoLoginDone(true);
            }
          } catch (_) {
            // 回退到前端直连尝试
            const okLogin = await tryMinIOAutoLogin(configData.access_key, configData.secret_key, { csrfToken });
            if (okLogin) setAutoLoginDone(true);
          }
        }
        setIframeUrl(sameOriginUrl);
        setDiagnosticNote('已使用同源Nginx代理路径集成MinIO控制台');
        setConnectionStatus('connected');
        return true;
      }
    } catch (e) {
      clearTimeout(timeout);
    }
    return false;
  };

  // 加载配置信息
  const loadConfig = async () => {
    setLoading(true);
    try {
      const response = await objectStorageAPI.getConfig(configId);
      const configData = response.data?.data;
      
      if (!configData) {
        message.warning('未找到对应配置，尝试通过同源代理直接访问MinIO控制台');
        const ok = await prepareSameOriginConsole();
        if (!ok) {
          message.error('配置不存在，且同源代理不可用');
        }
        return; // 留在本页以展示降级结果
      }
      
      if (configData.type !== 'minio') {
        message.warning('该配置不是MinIO类型，尝试同源代理降级');
        const ok = await prepareSameOriginConsole();
        if (!ok) {
          message.error('非MinIO类型，且同源代理不可用');
        }
        return; // 留在本页以展示降级结果
      }
      
      setConfig(configData);
      
      // 检查连接状态并构建iframe URL
      await checkConnectionAndSetup(configData);
      
    } catch (error) {
      console.error('加载配置失败:', error);
      message.error('加载配置失败: ' + (error.response?.data?.error || error.message));
      // 配置加载失败时，尝试直接降级到同源代理
      await prepareSameOriginConsole();
    } finally {
      setLoading(false);
    }
  };

  // 检查连接并设置iframe
  const checkConnectionAndSetup = async (configData) => {
    try {
      setConnectionStatus('checking');
      
      // 检查连接状态
      const statusResponse = await objectStorageAPI.checkConnection(configData.id);
      const status = statusResponse.data?.data?.status;
      
      setConnectionStatus(status);
      
      if (status === 'connected') {
        // 强优先使用同源Nginx代理路径，避免跨域与frame-ancestors限制
  const ok = await prepareSameOriginConsole(configData);
        if (ok) return;

        // 降级：使用配置的web_url（可能跨域/被浏览器策略限制）
        if (configData?.web_url) {
          let consoleUrl = configData.web_url;
          if (!consoleUrl.startsWith('http://') && !consoleUrl.startsWith('https://')) {
            consoleUrl = `http://${consoleUrl}`;
          }
          consoleUrl = consoleUrl.replace(/\/$/, '');
          setIframeUrl(consoleUrl);
          setDiagnosticNote('使用配置的Web控制台地址；如被浏览器策略拦截，请改为同源代理');
        } else {
          // 没有可用的web_url时，尝试同源代理兜底
          const fallbackOk = await prepareSameOriginConsole(configData);
          if (!fallbackOk) setConnectionStatus('error');
        }
      } else {
        // 未连接状态也尝试同源代理兜底
        const ok = await prepareSameOriginConsole(configData);
        if (!ok) setConnectionStatus('error');
      }
      
    } catch (error) {
      console.error('检查连接状态失败:', error);
      // 检查失败时也尝试同源代理兜底
      const ok = await prepareSameOriginConsole(configData);
      if (!ok) setConnectionStatus('error');
    }
  };

  // 刷新页面
  const handleRefresh = () => {
    if (config) {
      checkConnectionAndSetup(config);
    }
    
    // 刷新iframe
    const iframe = document.getElementById('minio-console-iframe');
    if (iframe) {
      iframe.src = iframe.src;
    }
  };

  // 切换全屏
  const toggleFullscreen = () => {
    const container = document.getElementById('minio-console-container');
    
    if (!isFullscreen) {
      // 进入全屏
      if (container.requestFullscreen) {
        container.requestFullscreen();
      } else if (container.mozRequestFullScreen) {
        container.mozRequestFullScreen();
      } else if (container.webkitRequestFullscreen) {
        container.webkitRequestFullscreen();
      } else if (container.msRequestFullscreen) {
        container.msRequestFullscreen();
      }
    } else {
      // 退出全屏
      if (document.exitFullscreen) {
        document.exitFullscreen();
      } else if (document.mozCancelFullScreen) {
        document.mozCancelFullScreen();
      } else if (document.webkitExitFullscreen) {
        document.webkitExitFullscreen();
      } else if (document.msExitFullscreen) {
        document.msExitFullscreen();
      }
    }
    
    setIsFullscreen(!isFullscreen);
  };

  // 监听全屏变化
  useEffect(() => {
    const handleFullscreenChange = () => {
      const isCurrentlyFullscreen = !!(
        document.fullscreenElement ||
        document.mozFullScreenElement ||
        document.webkitFullscreenElement ||
        document.msFullscreenElement
      );
      setIsFullscreen(isCurrentlyFullscreen);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    document.addEventListener('mozfullscreenchange', handleFullscreenChange);
    document.addEventListener('webkitfullscreenchange', handleFullscreenChange);
    document.addEventListener('MSFullscreenChange', handleFullscreenChange);

    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
      document.removeEventListener('mozfullscreenchange', handleFullscreenChange);
      document.removeEventListener('webkitfullscreenchange', handleFullscreenChange);
      document.removeEventListener('MSFullscreenChange', handleFullscreenChange);
    };
  }, []);

  useEffect(() => {
    if (configId) {
      loadConfig();
    }
  }, [configId]);

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" />
        <div style={{ marginTop: '16px' }}>
          <Text>加载MinIO控制台中...</Text>
        </div>
      </div>
    );
  }

  // 若无法加载配置但已探测到同源代理且构建了iframeUrl，则允许继续渲染控制台（无配置信息降级模式）
  if (!config && !iframeUrl) {
    return (
      <div style={{ padding: '24px' }}>
        <Alert
          message="配置未找到"
          description="请检查配置是否存在或返回重新选择；系统也会尝试通过同源代理自动降级访问 MinIO 控制台。"
          type="error"
          showIcon
          action={
            <Button size="small" onClick={() => navigate('/object-storage')}>
              返回
            </Button>
          }
        />
      </div>
    );
  }

  // 渲染连接状态
  const renderConnectionStatus = () => {
    switch (connectionStatus) {
      case 'checking':
        return <Tag icon={<Spin size="small" />} color="processing">检查连接中</Tag>;
      case 'connected':
        return <Tag icon={<CheckCircleOutlined />} color="success">已连接</Tag>;
      case 'error':
      case 'disconnected':
        return <Tag icon={<ExclamationCircleOutlined />} color="error">连接失败</Tag>;
      default:
        return <Tag color="default">未知状态</Tag>;
    }
  };

  return (
    <div 
      id="minio-console-container"
      style={{ 
        height: isFullscreen ? '100vh' : 'calc(100vh - 64px)',
        display: 'flex',
        flexDirection: 'column'
      }}
    >
      {/* 顶部操作栏 */}
      <div style={{ 
        padding: isFullscreen ? '8px 16px' : '16px 24px', 
        borderBottom: '1px solid #f0f0f0',
        backgroundColor: 'white',
        zIndex: 1000
      }}>
        <Row justify="space-between" align="middle">
          <Col>
            <Space>
              {!isFullscreen && (
                <>
                  <Button
                    icon={<ArrowLeftOutlined />}
                    onClick={() => navigate('/object-storage')}
                  >
                    返回
                  </Button>
                  
                  <Breadcrumb>
                    <Breadcrumb.Item>
                      <DatabaseOutlined /> 对象存储
                    </Breadcrumb.Item>
                    <Breadcrumb.Item>MinIO控制台</Breadcrumb.Item>
                    {config?.name && (
                      <Breadcrumb.Item>{config.name}</Breadcrumb.Item>
                    )}
                  </Breadcrumb>
                </>
              )}
              
              {isFullscreen && (
                <Title level={5} style={{ margin: 0, color: '#1890ff' }}>
                  <DatabaseOutlined /> MinIO{config?.name ? ` - ${config.name}` : ''}
                </Title>
              )}
            </Space>
          </Col>
          
          <Col>
            <Space>
              {renderConnectionStatus()}
              
              <Tooltip title="刷新控制台">
                <Button
                  icon={<ReloadOutlined />}
                  onClick={handleRefresh}
                />
              </Tooltip>
              
              <Tooltip title={isFullscreen ? "退出全屏" : "全屏显示"}>
                <Button
                  icon={<FullscreenOutlined />}
                  onClick={toggleFullscreen}
                />
              </Tooltip>
              
              {!isFullscreen && (
                <Tooltip title="打开新窗口">
                  <Button
                    icon={<LinkOutlined />}
                    onClick={() => window.open(iframeUrl, '_blank')}
                    disabled={!iframeUrl}
                  />
                </Tooltip>
              )}
            </Space>
          </Col>
        </Row>
        
        {!isFullscreen && (
          <div style={{ marginTop: '8px' }}>
            <Space direction="vertical" size={4}>
              <Space>
                {config?.endpoint && (
                  <Text type="secondary">
                    <DatabaseOutlined /> {config.endpoint}
                  </Text>
                )}
                {config?.web_url && (
                  <Text type="secondary">
                    <LinkOutlined /> {config.web_url}
                  </Text>
                )}
              </Space>
              {diagnosticNote && (
                <Text type="secondary">
                  <InfoCircleOutlined /> {diagnosticNote}
                </Text>
              )}
            </Space>
          </div>
        )}
      </div>

      {/* MinIO控制台内容区 */}
      <div style={{ flex: 1, position: 'relative', backgroundColor: '#f5f5f5' }}>
        {iframeUrl ? (
          <IframeEmbed
            src={iframeUrl}
            title="MinIO Console"
            timeoutMs={15000}
            id="minio-console-iframe"
            onReady={() => console.log('MinIO控制台加载完成')}
            onError={(why) => message.error(why || 'MinIO控制台加载失败')}
          />
        ) : connectionStatus === 'checking' ? (
          <div style={{ 
            display: 'flex', 
            justifyContent: 'center', 
            alignItems: 'center', 
            height: '100%' 
          }}>
            <Card>
              <div style={{ textAlign: 'center', padding: '40px' }}>
                <Spin size="large" />
                <div style={{ marginTop: '16px' }}>
                  <Text>正在连接MinIO服务...</Text>
                </div>
              </div>
            </Card>
          </div>
        ) : (
          <div style={{ 
            display: 'flex', 
            justifyContent: 'center', 
            alignItems: 'center', 
            height: '100%' 
          }}>
            <Card>
              <div style={{ textAlign: 'center', padding: '40px' }}>
                <ExclamationCircleOutlined 
                  style={{ fontSize: '48px', color: '#ff4d4f', marginBottom: '16px' }} 
                />
                <Title level={4}>无法连接MinIO服务</Title>
                <Text type="secondary">
                  请检查MinIO服务是否正常运行，或者配置是否正确
                </Text>
                <div style={{ marginTop: '16px' }}>
                  <Space>
                    <Button onClick={handleRefresh} icon={<ReloadOutlined />}>
                      重试连接
                    </Button>
                    {config?.id && (
                      <Button 
                        icon={<SettingOutlined />}
                        onClick={() => navigate(`/admin/object-storage?edit=${config.id}`)}
                      >
                        编辑配置
                      </Button>
                    )}
                  </Space>
                </div>
              </div>
            </Card>
          </div>
        )}
      </div>
    </div>
  );
};

export default MinIOConsolePage;