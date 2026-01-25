import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Button, Space, Alert, Card, theme, Tabs, Typography } from 'antd';
import { 
  ReloadOutlined, 
  ExportOutlined, 
  DeploymentUnitOutlined,
  AppstoreOutlined,
  SettingOutlined,
  LoginOutlined
} from '@ant-design/icons';
import { useI18n } from '../hooks/useI18n';

const { useToken } = theme;
const { Text } = Typography;

/**
 * ArgoCD 嵌入页面
 * 支持嵌入 ArgoCD Web UI
 * URL 优先级: window.__ARGOCD_URL__ (运行时) -> REACT_APP_ARGOCD_URL (构建时) -> 默认路径
 */
const resolveArgoCDUrl = () => {
  // runtime override
  // eslint-disable-next-line no-underscore-dangle
  const runtime = typeof window !== 'undefined' && window.__ARGOCD_URL__;
  const env = process.env.REACT_APP_ARGOCD_URL;
  
  // Prefer configured URL
  if (runtime && typeof runtime === 'string') return runtime;
  if (env && typeof env === 'string') return env;
  
  // 默认使用同源路径（需要 nginx 反向代理配置）
  const { protocol, hostname } = window.location;
  
  // 尝试通过 nginx 代理访问
  const proxyPath = '/argocd-ui/';
  
  // 直接端口访问（备用）
  const directUrl = `${protocol}//${hostname}:8280`;
  
  return { proxyPath, directUrl };
};

const ArgoCDEmbed = () => {
  const { t, locale } = useI18n();
  const { token } = useToken();
  const iframeRef = useRef(null);
  const baseUrls = useMemo(() => resolveArgoCDUrl(), []);
  
  const [activeTab, setActiveTab] = useState('dashboard');
  const [currentUrl, setCurrentUrl] = useState('');
  const [iframeKey, setIframeKey] = useState(0);
  const [useDirectUrl, setUseDirectUrl] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // 根据 tab 获取对应的 URL
  const getUrlForTab = (tab, useDirect = useDirectUrl) => {
    if (typeof baseUrls === 'string') {
      // 如果是字符串（环境变量配置），直接使用
      switch (tab) {
        case 'dashboard':
          return baseUrls;
        case 'applications':
          return `${baseUrls}/applications`;
        case 'settings':
          return `${baseUrls}/settings`;
        case 'login':
          return `${baseUrls}/login`;
        default:
          return baseUrls;
      }
    }
    
    const base = useDirect ? baseUrls.directUrl : baseUrls.proxyPath;
    
    switch (tab) {
      case 'dashboard':
        return base;
      case 'applications':
        return useDirect 
          ? `${baseUrls.directUrl}/applications`
          : `${baseUrls.proxyPath}applications`;
      case 'settings':
        return useDirect
          ? `${baseUrls.directUrl}/settings`
          : `${baseUrls.proxyPath}settings`;
      case 'login':
        return useDirect
          ? `${baseUrls.directUrl}/login`
          : `${baseUrls.proxyPath}login`;
      default:
        return base;
    }
  };

  // 初始化和 tab 切换时更新 URL
  useEffect(() => {
    const url = getUrlForTab(activeTab);
    setCurrentUrl(url);
    setIframeKey(Date.now());
    setLoading(true);
    setError(null);
  }, [activeTab, useDirectUrl]);

  // 检测代理路径是否可用
  useEffect(() => {
    let cancelled = false;
    const checkProxy = async () => {
      if (typeof baseUrls === 'string') return;
      
      try {
        const resp = await fetch(baseUrls.proxyPath, { 
          method: 'HEAD', 
          credentials: 'include',
          mode: 'no-cors'
        });
        // no-cors 模式下无法获取状态，默认使用代理路径
        if (!cancelled) {
          setUseDirectUrl(false);
        }
      } catch (err) {
        // 代理不可用，切换到直接访问
        if (!cancelled) {
          setUseDirectUrl(true);
        }
      }
    };
    checkProxy();
    return () => { cancelled = true; };
  }, [baseUrls]);

  const reload = () => {
    setIframeKey(Date.now());
    setLoading(true);
    setError(null);
  };

  const openNew = () => window.open(currentUrl, '_blank', 'noopener,noreferrer');

  const switchToDirectUrl = () => {
    setUseDirectUrl(!useDirectUrl);
  };

  const handleIframeLoad = () => {
    setLoading(false);
  };

  const handleIframeError = () => {
    setLoading(false);
    setError('无法加载 ArgoCD 页面，请尝试在新窗口中打开');
  };

  const iframeStyle = {
    width: '100%',
    height: 'calc(100vh - 64px - 120px)',
    border: `1px solid ${token.colorBorderSecondary}`,
    borderRadius: 6,
    background: token.colorBgContainer,
    display: loading ? 'none' : 'block'
  };

  const tabItems = [
    {
      key: 'dashboard',
      label: (
        <span>
          <DeploymentUnitOutlined />
          {t('argocd.dashboard', '仪表盘')}
        </span>
      )
    },
    {
      key: 'applications',
      label: (
        <span>
          <AppstoreOutlined />
          {t('argocd.applications', '应用')}
        </span>
      )
    },
    {
      key: 'settings',
      label: (
        <span>
          <SettingOutlined />
          {t('argocd.settings', '设置')}
        </span>
      )
    },
    {
      key: 'login',
      label: (
        <span>
          <LoginOutlined />
          {t('argocd.login', '登录')}
        </span>
      )
    }
  ];

  return (
    <div style={{ padding: 24 }}>
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <Card
          size="small"
          title={
            <Space>
              <DeploymentUnitOutlined style={{ color: token.colorPrimary }} />
              <span style={{ fontSize: 14 }}>{t('argocd.title', 'ArgoCD GitOps')}</span>
            </Space>
          }
          extra={
            <Space>
              <Button icon={<ReloadOutlined />} onClick={reload} size="small">
                {t('common.refresh', '刷新')}
              </Button>
              <Button icon={<ExportOutlined />} onClick={openNew} size="small">
                {t('common.openNewWindow', '新窗口打开')}
              </Button>
            </Space>
          }
          bodyStyle={{ padding: 12 }}
        >
          <Space direction="vertical" style={{ width: '100%' }}>
            <Alert
              type="info"
              banner
              showIcon
              message={
                <span style={{ fontSize: 12 }}>
                  {t('argocd.embeddedUrl', '嵌入地址')}: <code>{currentUrl}</code>
                </span>
              }
              style={{ padding: '6px 8px' }}
            />
            {typeof baseUrls !== 'string' && (
              <Alert
                type={useDirectUrl ? 'warning' : 'success'}
                showIcon
                message={
                  <span style={{ fontSize: 12 }}>
                    {useDirectUrl 
                      ? t('argocd.directAccess', '当前使用直接端口访问')
                      : t('argocd.proxyAccess', '当前通过代理访问')
                    }
                    <Button 
                      size="small" 
                      style={{ marginLeft: 8 }} 
                      onClick={switchToDirectUrl}
                    >
                      {useDirectUrl 
                        ? t('argocd.switchToProxy', '切换到代理')
                        : t('argocd.switchToDirect', '切换到直接访问')
                      }
                    </Button>
                  </span>
                }
                style={{ padding: '6px 8px' }}
              />
            )}
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t('argocd.embedHint', '提示：如果页面无法加载，请尝试在新窗口中打开。ArgoCD 需要 Kubernetes 集群支持。')}
            </Text>
          </Space>
        </Card>

        <Tabs 
          activeKey={activeTab} 
          onChange={setActiveTab}
          items={tabItems}
          style={{ marginBottom: 0 }}
        />

        {loading && (
          <div style={{ 
            textAlign: 'center', 
            padding: '100px 0',
            background: token.colorBgContainer,
            borderRadius: 6,
            border: `1px solid ${token.colorBorderSecondary}`
          }}>
            <DeploymentUnitOutlined style={{ fontSize: 48, color: token.colorTextSecondary }} />
            <div style={{ marginTop: 16, color: token.colorTextSecondary }}>
              {t('argocd.loading', '正在加载 ArgoCD...')}
            </div>
          </div>
        )}

        {error && (
          <Alert
            type="error"
            showIcon
            message={error}
            description={
              <span>
                {t('argocd.errorHint', 'ArgoCD 服务可能未启动或需要 Kubernetes 集群支持。')}
              </span>
            }
            action={
              <Button size="small" onClick={openNew}>
                {t('common.openNewWindow', '新窗口打开')}
              </Button>
            }
          />
        )}

        <iframe
          ref={iframeRef}
          key={iframeKey}
          title="embedded-argocd"
          src={currentUrl}
          style={iframeStyle}
          allow="clipboard-read; clipboard-write; fullscreen"
          onLoad={handleIframeLoad}
          onError={handleIframeError}
        />
      </Space>
    </div>
  );
};

export default ArgoCDEmbed;
