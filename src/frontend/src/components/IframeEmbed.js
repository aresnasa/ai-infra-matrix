import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Card, Spin, Typography, Button, Space, Alert, Tag, Tooltip } from 'antd';
import { ReloadOutlined, LinkOutlined, InfoCircleOutlined } from '@ant-design/icons';

const { Text } = Typography;

/**
 * IframeEmbed
 * A resilient iframe wrapper with diagnostics and graceful fallbacks.
 *
 * Props:
 * - src: string (required)
 * - title: string
 * - style: object
 * - className: string
 * - timeoutMs: number (default: 12000) time to wait before showing diagnostics
 * - allow: string (iframe allow attr)
 * - sandbox: string (iframe sandbox attr)
 * - referrerPolicy: string
 * - onReady: () => void
 * - onError: (reason?: string) => void
 * - openInNewWindow: boolean (default: true) show an "open" button
 */
export default function IframeEmbed({
  src,
  title,
  style,
  className,
  id,
  timeoutMs = 12000,
  allow = 'fullscreen; clipboard-read; clipboard-write',
  sandbox,
  referrerPolicy = 'no-referrer-when-downgrade',
  onReady,
  onError,
  openInNewWindow = true,
}) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [reason, setReason] = useState(null);
  const [headers, setHeaders] = useState(null);
  const [key, setKey] = useState(() => Date.now());
  const timerRef = useRef();
  const iframeRef = useRef();

  const finalSrc = useMemo(() => {
    if (!src) return src;
    // enforce trailing slash for SPA consoles that expect path-rooted assets
    try {
      const u = new URL(src, window.location.href);
      // keep query/hash intact
      if (!u.pathname.endsWith('/')) {
        u.pathname = `${u.pathname}/`;
      }
      return u.toString();
    } catch {
      // fallback for non-URL strings
      return src.endsWith('/') ? src : `${src}/`;
    }
  }, [src]);

  // Lightweight diagnostics: probe headers to detect common blockers
  useEffect(() => {
    let abort = false;
    setHeaders(null);
    setReason(null);
    if (!finalSrc) return;

    (async () => {
      try {
        // Use GET no-store to avoid cached redirects; same-origin expected
        const res = await fetch(finalSrc, { method: 'GET', cache: 'no-store', credentials: 'include' });
        if (abort) return;
        const xf = res.headers.get('x-frame-options');
        const csp = res.headers.get('content-security-policy');
        const loc = res.headers.get('location');
        setHeaders({ status: res.status, xf, csp, loc });

        if (xf && /deny|sameorigin/i.test(xf) && window.location.origin) {
          // SAMEORIGIN is fine for same-origin, flag only if DENY
          if (/deny/i.test(xf)) setReason('X-Frame-Options: DENY');
        }
        if (csp && /frame-ancestors/i.test(csp)) {
          const ancestorRule = (csp.match(/frame-ancestors[^;]*/i) || [])[0];
          // If rule excludes self/current origin, warn
          const origin = window.location.origin;
          if (ancestorRule && !ancestorRule.includes("'self'") && !ancestorRule.includes(origin)) {
            setReason(`CSP ${ancestorRule}`);
          }
        }
      } catch (e) {
        // ignore fetch errors; the iframe may still load
      }
    })();

    return () => { abort = true; };
  }, [finalSrc, key]);

  // Timeout guard
  useEffect(() => {
    if (!finalSrc) return;
    setLoading(true);
    setError(null);

    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      if (loading) {
        const msg = reason || '内容加载超时，可能被上游安全策略或重定向阻止';
        setError(msg);
        onError && onError(msg);
      }
    }, timeoutMs);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [finalSrc, key, reason]);

  const handleLoad = () => {
    setLoading(false);
    setError(null);
    if (timerRef.current) clearTimeout(timerRef.current);
    onReady && onReady();

    // Same-origin sanity: if accessible, ensure we're not bounced to '/'
    try {
      const doc = iframeRef.current?.contentWindow?.document;
      const loc = iframeRef.current?.contentWindow?.location?.href;
      if (doc && loc) {
        // no-op, but keeps a hook for future path corrections if needed
      }
    } catch {
      // cross-origin, ignore
    }
  };

  const handleError = () => {
    setLoading(false);
    const msg = reason || '内容加载失败';
    setError(msg);
    onError && onError(msg);
  };

  const doReload = () => {
    setLoading(true);
    setError(null);
    setKey(Date.now());
  };

  if (!finalSrc) {
    return (
      <Center>
        <Card>
          <Text type="secondary">未提供有效的地址</Text>
        </Card>
      </Center>
    );
  }

  return (
    <div style={{ position: 'relative', width: '100%', height: '100%' }} className={className}>
      {(loading || error) && (
        <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 2 }}>
          <Card>
            <div style={{ minWidth: 320 }}>
              {loading && (
                <div style={{ textAlign: 'center', padding: 16 }}>
                  <Spin size="large" />
                  <div style={{ marginTop: 8 }}>
                    <Text>正在加载内容...</Text>
                  </div>
                </div>
              )}
              {error && (
                <Alert
                  type="error"
                  showIcon
                  icon={<InfoCircleOutlined />}
                  message="嵌入内容加载失败"
                  description={
                    <div>
                      <div style={{ marginBottom: 8 }}>{error}</div>
                      {headers && (
                        <div style={{ fontSize: 12, color: '#666' }}>
                          <div>状态: {headers.status ?? '未知'}</div>
                          {headers.xf && <div>X-Frame-Options: <code>{headers.xf}</code></div>}
                          {headers.csp && <div>Content-Security-Policy: <code>{headers.csp}</code></div>}
                          {headers.loc && <div>Location: <code>{headers.loc}</code></div>}
                        </div>
                      )}
                      <div style={{ marginTop: 12 }}>
                        <Space>
                          <Button icon={<ReloadOutlined />} onClick={doReload}>重试</Button>
                          {openInNewWindow && (
                            <Button icon={<LinkOutlined />} onClick={() => window.open(finalSrc, '_blank')}>新窗口打开</Button>
                          )}
                          {headers?.csp && <Tag color="orange">CSP可能限制了嵌入</Tag>}
                        </Space>
                      </div>
                    </div>
                  }
                />
              )}
            </div>
          </Card>
        </div>
      )}

      <iframe
        key={key}
        ref={iframeRef}
        id={id}
        src={finalSrc}
        title={title || 'embedded-frame'}
        allow={allow}
        sandbox={sandbox}
        referrerPolicy={referrerPolicy}
        style={{ width: '100%', height: '100%', border: 'none', backgroundColor: 'white', ...style }}
        onLoad={handleLoad}
        onError={handleError}
      />
    </div>
  );
}

function Center({ children }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%' }}>
      {children}
    </div>
  );
}
