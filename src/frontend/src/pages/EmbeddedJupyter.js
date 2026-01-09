import React, { useEffect, useMemo, useState, useCallback } from 'react';
import { Card, Alert, Button, Space, Spin, Typography, theme } from 'antd';
import { ReloadOutlined, ExportOutlined } from '@ant-design/icons';
import { useI18n } from '../hooks/useI18n';

const { useToken } = theme;

// Embedded Jupyter subpage with SSO preflight and full-height iframe
const EmbeddedJupyter = () => {
  const { t } = useI18n();
  const { token } = useToken();
  const [ready, setReady] = useState(false);
  const [checking, setChecking] = useState(true);
  const [error, setError] = useState(null);
  const jupyterBase = useMemo(() => `${window.location.origin}/jupyter/`, []);

  const [iframeKey, setIframeKey] = useState(0);

  const preflight = useCallback(async () => {
    let cancelled = false;
    setChecking(true);
    setError(null);
    try {
      const res = await fetch('/api/jupyter/access', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token') || ''}`
        },
        body: JSON.stringify({
          redirect_uri: '/jupyter',
          source: 'embedded_jupyter'
        })
      });

      let data = null;
      try { data = await res.json(); } catch (_) {}

      if (!cancelled) {
        if (data?.success && data?.action === 'authenticated') {
          setReady(true);
        } else if (data?.action === 'redirect' && data?.redirect_url) {
          window.location.href = data.redirect_url;
          return;
        } else {
          setReady(true);
        }
      }
    } catch (e) {
      if (!cancelled) {
        setError(t('jupyter.preflightError'));
        setReady(true);
      }
    } finally {
      if (!cancelled) setChecking(false);
    }
    return () => { cancelled = true; };
  }, [t]);

  useEffect(() => {
    preflight();
  }, [preflight]);

  const openInNewTab = () => window.open(`${jupyterBase}hub/`, '_blank', 'noopener');
  const handleRefresh = async () => {
    await preflight();
    // Force reload iframe without leaving the subpage
    setIframeKey(Date.now());
  };

  // Viewport-filling iframe height: subtract header (64) and content paddings/margins (~48)
  const iframeStyle = {
    width: '100%',
    height: 'calc(100vh - 64px - 48px)',
    border: `1px solid ${token.colorBorderSecondary}`,
    borderRadius: 6,
    background: token.colorBgContainer
  };

  return (
    <div style={{ padding: 24 }}>
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        <Card
          size="small"
          title={<span style={{ fontSize: 14 }}>{t('jupyter.title')}</span>}
          extra={
            <Space>
              <Button icon={<ReloadOutlined />} onClick={handleRefresh} size="small">{t('common.refresh')}</Button>
              <Button icon={<ExportOutlined />} onClick={openInNewTab} size="small">{t('jupyter.openNewWindow')}</Button>
            </Space>
          }
          bodyStyle={{ padding: 12 }}
        >
          <Alert
            type="info"
            banner
            showIcon
            message={
              <span style={{ fontSize: 12 }}>
                {t('jupyter.embeddedInfo')}
              </span>
            }
            style={{ padding: '6px 8px' }}
          />
        </Card>

        {checking && !ready ? (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', minHeight: 240 }}>
            <Spin size="large" tip={t('jupyter.preparing')} />
          </div>
        ) : (
          <>
            {error && (
              <Alert type="warning" message={error} showIcon />
            )}
            <iframe
              title="embedded-jupyterhub"
              key={iframeKey}
              src={jupyterBase}
              style={iframeStyle}
              allow="clipboard-read; clipboard-write; fullscreen"
            />
          </>
        )}
      </Space>
    </div>
  );
};

export default EmbeddedJupyter;
