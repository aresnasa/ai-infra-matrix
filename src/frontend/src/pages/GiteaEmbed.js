import React, { useMemo, useRef, useState } from 'react';
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
  const [iframeKey, setIframeKey] = useState(0);

  const reload = () => {
    // Force reload without leaving the page
    setIframeKey(Date.now());
  };

  const openNew = () => window.open(base, '_blank', 'noopener,noreferrer');

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
          <Alert
            type="info"
            banner
            showIcon
            message={<span style={{ fontSize: 12 }}>内嵌同源 Gitea。若页面未显示，请检查代理配置或安全策略。</span>}
            style={{ padding: '6px 8px' }}
          />
        </Card>
        <iframe
          ref={iframeRef}
          key={iframeKey}
          title="embedded-gitea"
          src={base}
          style={iframeStyle}
          allow="clipboard-read; clipboard-write; fullscreen"
        />
      </Space>
    </div>
  );
};

export default GiteaEmbed;
