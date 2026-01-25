import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Button, Space, Alert, Card, theme, Tabs, Typography } from 'antd';
import { 
  ReloadOutlined, 
  ExportOutlined, 
  SafetyCertificateOutlined,
  SettingOutlined,
  UserOutlined,
  LoginOutlined
} from '@ant-design/icons';
import { useI18n } from '../hooks/useI18n';

const { useToken } = theme;
const { Text } = Typography;

/**
 * Keycloak 嵌入页面
 * 支持嵌入 Keycloak 管理控制台和账户管理页面
 * URL 优先级: window.__KEYCLOAK_URL__ (运行时) -> REACT_APP_KEYCLOAK_URL (构建时) -> 默认路径
 */
const resolveKeycloakUrl = () => {
  // runtime override
  // eslint-disable-next-line no-underscore-dangle
  const runtime = typeof window !== 'undefined' && window.__KEYCLOAK_URL__;
  const env = process.env.REACT_APP_KEYCLOAK_URL;
  
  // Prefer configured URL
  if (runtime && typeof runtime === 'string') return runtime;
  if (env && typeof env === 'string') return env;
  
  // 默认使用同源路径（需要 nginx 反向代理配置）
  // 如果直接访问 Keycloak 端口，使用 /auth 路径
  const { protocol, hostname } = window.location;
  
  // 尝试通过 nginx 代理访问
  const proxyPath = '/keycloak-admin/';
  
  // 直接端口访问（备用）
  const directUrl = `${protocol}//${hostname}:8180/auth`;
  
  return { proxyPath, directUrl };
};

const KeycloakEmbed = () => {
  const { t, locale } = useI18n();
  const { token } = useToken();
  const iframeRef = useRef(null);
  const baseUrls = useMemo(() => resolveKeycloakUrl(), []);
  
  const [activeTab, setActiveTab] = useState('admin');
  const [currentUrl, setCurrentUrl] = useState('');
  const [iframeKey, setIframeKey] = useState(0);
  const [useDirectUrl, setUseDirectUrl] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // 根据 tab 获取对应的 URL
  const getUrlForTab = (tab, useDirect = useDirectUrl) => {
    const base = useDirect ? baseUrls.directUrl : baseUrls.proxyPath;
    
    if (typeof baseUrls === 'string') {
      // 如果是字符串（环境变量配置），直接使用
      switch (tab) {
        case 'admin':
          return `${baseUrls}/admin/master/console/`;
        case 'account':
          return `${baseUrls}/realms/ai-infra/account/`;
        case 'login':
          return `${baseUrls}/realms/ai-infra/protocol/openid-connect/auth?client_id=account&redirect_uri=${encodeURIComponent(window.location.origin)}&response_type=code`;
        default:
          return baseUrls;
      }
    }
    
    switch (tab) {
      case 'admin':
        return useDirect 
          ? `${baseUrls.directUrl}/admin/master/console/`
          : `${baseUrls.proxyPath}admin/master/console/`;
      case 'account':
        return useDirect
          ? `${baseUrls.directUrl}/realms/ai-infra/account/`
          : `${baseUrls.proxyPath}realms/ai-infra/account/`;
      case 'login':
        const redirectUri = encodeURIComponent(window.location.origin);
        return useDirect
          ? `${baseUrls.directUrl}/realms/ai-infra/protocol/openid-connect/auth?client_id=account&redirect_uri=${redirectUri}&response_type=code`
          : `${baseUrls.proxyPath}realms/ai-infra/protocol/openid-connect/auth?client_id=account&redirect_uri=${redirectUri}&response_type=code`;
      default:
        return useDirect ? baseUrls.directUrl : baseUrls.proxyPath;
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
    setError('无法加载 Keycloak 页面，请尝试在新窗口中打开');
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
      key: 'admin',
      label: (
        <span>
          <SettingOutlined />
          {t('keycloak.adminConsole', '管理控制台')}
        </span>
      )
    },
    {
      key: 'account',
      label: (
        <span>
          <UserOutlined />
          {t('keycloak.accountManagement', '账户管理')}
        </span>
      )
    },
    {
      key: 'login',
      label: (
        <span>
          <LoginOutlined />
          {t('keycloak.login', '登录页面')}
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
              <SafetyCertificateOutlined style={{ color: token.colorPrimary }} />
              <span style={{ fontSize: 14 }}>{t('keycloak.title', 'Keycloak 身份认证')}</span>
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
                  {t('keycloak.embeddedUrl', '嵌入地址')}: <code>{currentUrl}</code>
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
                      ? t('keycloak.directAccess', '当前使用直接端口访问')
                      : t('keycloak.proxyAccess', '当前通过代理访问')
                    }
                    <Button 
                      size="small" 
                      style={{ marginLeft: 8 }} 
                      onClick={switchToDirectUrl}
                    >
                      {useDirectUrl 
                        ? t('keycloak.switchToProxy', '切换到代理')
                        : t('keycloak.switchToDirect', '切换到直接访问')
                      }
                    </Button>
                  </span>
                }
                style={{ padding: '6px 8px' }}
              />
            )}
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t('keycloak.embedHint', '提示：如果页面无法加载，请尝试在新窗口中打开。默认管理员账号: admin')}
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
            <SafetyCertificateOutlined style={{ fontSize: 48, color: token.colorTextSecondary }} />
            <div style={{ marginTop: 16, color: token.colorTextSecondary }}>
              {t('keycloak.loading', '正在加载 Keycloak...')}
            </div>
          </div>
        )}

        {error && (
          <Alert
            type="error"
            showIcon
            message={error}
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
          title="embedded-keycloak"
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

export default KeycloakEmbed;
