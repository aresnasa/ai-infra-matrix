import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Button, Space, Alert, Card } from 'antd';
import { ReloadOutlined, ExportOutlined } from '@ant-design/icons';

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
  const iframeRef = useRef(null);
  const base = useMemo(() => resolveGiteaUrl(), []);
  const [currentUrl, setCurrentUrl] = useState(base);
  const [iframeKey, setIframeKey] = useState(0);

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
          setCurrentUrl('/gitea/');
          setIframeKey(Date.now());
        }
      } catch (err) {
        // ignore; keep external URL and show hint below
      }
    };
    trySwitchToLocal();
    return () => { cancelled = true; };
  }, [isCrossOrigin]);

  const reload = () => {
    // Force reload without leaving the page
    setIframeKey(Date.now());
  };

  const openNew = () => window.open(currentUrl, '_blank', 'noopener,noreferrer');

  const switchToSameOrigin = () => {
    // eslint-disable-next-line no-underscore-dangle
    window.__GITEA_URL__ = '/gitea/';
    setCurrentUrl('/gitea/');
    setIframeKey(Date.now());
  };

  const iframeStyle = {
    width: '100%',
    height: 'calc(100vh - 64px - 48px)', // align with EmbeddedJupyter
    border: '1px solid #f0f0f0',
    borderRadius: 6,
    background: '#fff'
  };

  return (
    <div style={{ padding: 24 }}>
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <Card
          size="small"
          title={<span style={{ fontSize: 14 }}>Gitea 代码托管</span>}
          extra={
            <Space>
              <Button icon={<ReloadOutlined />} onClick={reload} size="small">刷新</Button>
              <Button icon={<ExportOutlined />} onClick={openNew} size="small">新窗口打开</Button>
            </Space>
          }
          bodyStyle={{ padding: 12 }}
        >
          <Space direction="vertical" style={{ width: '100%' }}>
            <Alert
              type="info"
              banner
              showIcon
              message={<span style={{ fontSize: 12 }}>内嵌 Gitea 地址: <code>{currentUrl}</code></span>}
              style={{ padding: '6px 8px' }}
            />
            {isCrossOrigin && (
              <Alert
                type="warning"
                showIcon
                message={
                  <span style={{ fontSize: 12 }}>
                    当前配置为跨域地址，可能被目标站点的 X-Frame-Options/CSP 阻止内嵌。建议切换为同源网关路径 <code>/gitea/</code>。
                    <Button size="small" style={{ marginLeft: 8 }} onClick={switchToSameOrigin}>切换为同源</Button>
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
    </div>
  );
};

export default GiteaEmbed;
