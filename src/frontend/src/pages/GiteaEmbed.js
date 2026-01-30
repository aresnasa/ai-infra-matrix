import React, { useEffect, useMemo, useRef, useState, useCallback } from 'react';
import { Button, Space, Alert, Card, theme, Spin, message } from 'antd';
import { ReloadOutlined, ExportOutlined, UserSwitchOutlined, SyncOutlined } from '@ant-design/icons';
import { useI18n } from '../hooks/useI18n';

const { useToken } = theme;

// 获取当前登录用户
const getCurrentUser = () => {
  try {
    const savedUser = localStorage.getItem('user');
    if (savedUser) {
      return JSON.parse(savedUser);
    }
  } catch (e) {
    console.warn('Failed to parse user from localStorage:', e);
  }
  return null;
};

// Simple iframe wrapper to embed a Gitea instance inside the portal.
// URL priority: window.__GITEA_URL__ (runtime) -> REACT_APP_GITEA_URL (build-time) ->
// fallback '/gitea' (expect reverse proxy) -> finally 'https://try.gitea.io'.
const resolveGiteaUrl = () => {
  // runtime override
  // eslint-disable-next-line no-underscore-dangle
  const runtime = typeof window !== 'undefined' && window.__GITEA_URL__;
  const env = process.env.REACT_APP_GITEA_URL;
  // Prefer configured URL
  if (runtime && typeof runtime === 'string') return runtime;
  if (env && typeof env === 'string') return env;
  // Local reverse-proxy path (user can map Nginx location /gitea -> gitea service)
  // If not set up, the external demo will be used as a last resort.
  const localPath = '/gitea/';
  // Heuristic: if hosting at same origin and no backend mapping, localPath may 404.
  // Still return it; user can switch via REACT_APP_GITEA_URL without rebuild using window.__GITEA_URL__.
  return localPath || 'https://try.gitea.io';
};

const GiteaEmbed = () => {
  const { t, locale } = useI18n();
  const { token } = useToken();
  const iframeRef = useRef(null);
  const base = useMemo(() => resolveGiteaUrl(), []);
  
  // 当前 SSO 用户
  const [currentSSOUser, setCurrentSSOUser] = useState(() => getCurrentUser());
  // Gitea iframe 内的用户（从 localStorage 缓存读取）
  const [giteaUser, setGiteaUser] = useState(() => {
    try {
      return localStorage.getItem('gitea_current_user') || null;
    } catch { return null; }
  });
  // 用户同步状态
  const [syncing, setSyncing] = useState(false);
  const [userMismatch, setUserMismatch] = useState(false);
  
  // 添加语言参数到 URL，Gitea 支持 lang 参数
  const getUrlWithLang = (url) => {
    // Gitea 语言代码：zh-CN, en-US 等
    const separator = url.includes('?') ? '&' : '?';
    return `${url}${separator}lang=${locale}`;
  };
  
  const [currentUrl, setCurrentUrl] = useState(() => getUrlWithLang(base));
  const [iframeKey, setIframeKey] = useState(0);

  // 检测用户不匹配
  useEffect(() => {
    const ssoUsername = currentSSOUser?.username || currentSSOUser?.name;
    if (ssoUsername && giteaUser && ssoUsername !== giteaUser) {
      setUserMismatch(true);
    } else {
      setUserMismatch(false);
    }
  }, [currentSSOUser, giteaUser]);

  // 监听 SSO 用户变化
  useEffect(() => {
    const handleStorageChange = (e) => {
      if (e.key === 'user') {
        setCurrentSSOUser(getCurrentUser());
      }
    };
    window.addEventListener('storage', handleStorageChange);
    
    // 定期检查用户变化
    const checkInterval = setInterval(() => {
      const newUser = getCurrentUser();
      const newUsername = newUser?.username || newUser?.name;
      const currentUsername = currentSSOUser?.username || currentSSOUser?.name;
      if (newUsername !== currentUsername) {
        setCurrentSSOUser(newUser);
      }
    }, 2000);
    
    return () => {
      window.removeEventListener('storage', handleStorageChange);
      clearInterval(checkInterval);
    };
  }, [currentSSOUser]);

  // 同步 Gitea 用户会话
  const syncGiteaUser = useCallback(async () => {
    const ssoUsername = currentSSOUser?.username || currentSSOUser?.name;
    if (!ssoUsername) {
      message.warning(t('gitea.noSSOUser') || '未检测到登录用户');
      return;
    }
    
    setSyncing(true);
    try {
      // Step 1: 调用 Gitea 登出接口清除旧会话
      await fetch('/gitea/user/logout', { 
        method: 'GET', 
        credentials: 'include',
        redirect: 'manual'
      }).catch(() => {});
      
      // Step 2: 清除 Gitea 相关 cookies（通过后端代理）
      await fetch('/gitea/_logout', { 
        method: 'GET', 
        credentials: 'include' 
      }).catch(() => {});
      
      // Step 3: 访问 /gitea/user/login 触发 SSO 认证建立新会话
      // Nginx 会注入当前 SSO 用户信息到请求头
      await fetch('/gitea/user/login', {
        method: 'GET',
        credentials: 'include',
        redirect: 'manual'
      }).catch(() => {});
      
      // Step 4: 更新本地缓存的 Gitea 用户
      localStorage.setItem('gitea_current_user', ssoUsername);
      setGiteaUser(ssoUsername);
      setUserMismatch(false);
      
      // Step 5: 刷新 iframe
      setIframeKey(Date.now());
      message.success(t('gitea.userSynced') || `已切换到用户: ${ssoUsername}`);
      
    } catch (error) {
      console.error('Gitea user sync failed:', error);
      message.error(t('gitea.syncFailed') || '用户同步失败');
    } finally {
      setSyncing(false);
    }
  }, [currentSSOUser, t]);

  // 组件挂载时自动检测并同步用户
  useEffect(() => {
    const ssoUsername = currentSSOUser?.username || currentSSOUser?.name;
    const cachedGiteaUser = localStorage.getItem('gitea_current_user');
    
    // 如果 SSO 用户存在且与缓存的 Gitea 用户不同，自动同步
    if (ssoUsername && cachedGiteaUser && ssoUsername !== cachedGiteaUser) {
      syncGiteaUser();
    } else if (ssoUsername && !cachedGiteaUser) {
      // 首次访问，记录当前用户
      localStorage.setItem('gitea_current_user', ssoUsername);
      setGiteaUser(ssoUsername);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 监听语言变化，刷新 iframe
  useEffect(() => {
    setCurrentUrl(getUrlWithLang(base));
    setIframeKey(Date.now());
  }, [locale, base]);

  // Detect if configured URL is cross-origin (very likely to be blocked by X-Frame-Options/CSP)
  const isCrossOrigin = useMemo(() => {
    try {
      const u = new URL(currentUrl, window.location.origin);
      return u.origin !== window.location.origin;
    } catch (e) {
      return false;
    }
  }, [currentUrl]);

  // If cross-origin but local gateway /gitea/ is available, auto-switch to same-origin to avoid frame blocking
  useEffect(() => {
    let cancelled = false;
    const trySwitchToLocal = async () => {
      if (!isCrossOrigin) return;
      try {
        const resp = await fetch('/gitea/', { method: 'HEAD', credentials: 'include' });
        if (!cancelled && resp.ok) {
          // prefer same-origin path for embedding
          // eslint-disable-next-line no-underscore-dangle
          window.__GITEA_URL__ = '/gitea/';
          setCurrentUrl(getUrlWithLang('/gitea/'));
          setIframeKey(Date.now());
        }
      } catch (err) {
        // ignore; keep external URL and show hint below
      }
    };
    trySwitchToLocal();
    return () => { cancelled = true; };
  }, [isCrossOrigin, locale]);

  const reload = () => {
    // Force reload without leaving the page
    setIframeKey(Date.now());
  };

  const openNew = () => window.open(currentUrl, '_blank', 'noopener,noreferrer');

  const switchToSameOrigin = () => {
    // eslint-disable-next-line no-underscore-dangle
    window.__GITEA_URL__ = '/gitea/';
    setCurrentUrl(getUrlWithLang('/gitea/'));
    setIframeKey(Date.now());
  };

  // 获取显示的用户名
  const getSSOUsername = () => currentSSOUser?.username || currentSSOUser?.name || t('gitea.unknownUser') || '未知';

  const iframeStyle = {
    width: '100%',
    height: 'calc(100vh - 64px - 48px)', // align with EmbeddedJupyter
    border: `1px solid ${token.colorBorderSecondary}`,
    borderRadius: 6,
    background: token.colorBgContainer
  };

  return (
    <div style={{ padding: 24 }}>
      <Spin spinning={syncing} tip={t('gitea.syncing') || '正在同步用户...'}>
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Card
            size="small"
            title={<span style={{ fontSize: 14 }}>{t('gitea.title')}</span>}
            extra={
              <Space>
                <Button 
                  icon={<SyncOutlined spin={syncing} />} 
                  onClick={syncGiteaUser} 
                  size="small"
                  type={userMismatch ? 'primary' : 'default'}
                  danger={userMismatch}
                  disabled={syncing}
                >
                  {t('gitea.syncUser') || '同步用户'}
                </Button>
                <Button icon={<ReloadOutlined />} onClick={reload} size="small">{t('common.refresh')}</Button>
                <Button icon={<ExportOutlined />} onClick={openNew} size="small">{t('gitea.openNewWindow')}</Button>
              </Space>
            }
            bodyStyle={{ padding: 12 }}
          >
            <Space direction="vertical" style={{ width: '100%' }}>
              {/* 用户状态显示 */}
              <Alert
                type={userMismatch ? 'warning' : 'success'}
                showIcon
                icon={<UserSwitchOutlined />}
                message={
                  <span style={{ fontSize: 12 }}>
                    {t('gitea.currentSSOUser') || 'SSO 用户'}: <strong>{getSSOUsername()}</strong>
                    {giteaUser && (
                      <>
                        {' | '}{t('gitea.giteaUser') || 'Gitea 用户'}: <strong>{giteaUser}</strong>
                      </>
                    )}
                    {userMismatch && (
                      <span style={{ color: token.colorError, marginLeft: 8 }}>
                        ({t('gitea.userMismatch') || '用户不匹配，请点击同步'})
                      </span>
                    )}
                  </span>
                }
                style={{ padding: '6px 8px' }}
              />
              
              <Alert
                type="info"
                banner
                showIcon
                message={<span style={{ fontSize: 12 }}>{t('gitea.embeddedUrl')}: <code>{currentUrl}</code></span>}
                style={{ padding: '6px 8px' }}
              />
              {isCrossOrigin && (
                <Alert
                  type="warning"
                  showIcon
                  message={
                    <span style={{ fontSize: 12 }}>
                      {t('gitea.crossOriginWarning')} <code>/gitea/</code>
                      <Button size="small" style={{ marginLeft: 8 }} onClick={switchToSameOrigin}>{t('gitea.switchToSameOrigin')}</Button>
                    </span>
                  }
                  style={{ padding: '6px 8px' }}
                />
              )}
            </Space>
          </Card>
          <iframe
            ref={iframeRef}
            key={iframeKey}
            title="embedded-gitea"
            src={currentUrl}
            style={iframeStyle}
            allow="clipboard-read; clipboard-write; fullscreen"
          />
        </Space>
      </Spin>
    </div>
  );
};

export default GiteaEmbed;
