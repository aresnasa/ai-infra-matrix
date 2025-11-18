import React, { useState, useEffect } from 'react';
import { Card, Spin, Alert, Button, Space, message } from 'antd';
import { ReloadOutlined, ExportOutlined } from '@ant-design/icons';

const KafkaUIPage = () => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [iframeLoaded, setIframeLoaded] = useState(false);

  // Kafka UI的URL
  const kafkaUIUrl = 'http://localhost:9095';

  useEffect(() => {
    // 检查Kafka UI服务是否可访问
    const checkKafkaUIHealth = async () => {
      try {
        // 简单的健康检查 - 尝试访问Kafka UI
        const response = await fetch(kafkaUIUrl, {
          method: 'HEAD',
          mode: 'no-cors' // 避免CORS问题
        });

        setLoading(false);
        setError(null);
      } catch (err) {
        console.log('Kafka UI健康检查:', err.message);
        setLoading(false);
        setError('Kafka UI服务暂时不可用，请稍后重试');
      }
    };

    checkKafkaUIHealth();
  }, []);

  const handleIframeLoad = () => {
    setIframeLoaded(true);
    setLoading(false);
  };

  const handleIframeError = () => {
    setLoading(false);
    setError('无法加载Kafka UI界面');
  };

  const handleRefresh = () => {
    setLoading(true);
    setError(null);
    setIframeLoaded(false);
    // 重新加载iframe
    const iframe = document.getElementById('kafka-ui-iframe');
    if (iframe) {
      iframe.src = iframe.src;
    }
  };

  const handleOpenInNewTab = () => {
    window.open(kafkaUIUrl, '_blank');
  };

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '400px' }}>
        <Spin size="large" tip="正在加载Kafka UI..." />
      </div>
    );
  }

  if (error) {
    return (
      <Card
        title="Kafka UI 管理界面"
        style={{ margin: '20px' }}
        extra={
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={handleRefresh}
              loading={loading}
            >
              重试
            </Button>
            <Button
              icon={<ExportOutlined />}
              onClick={handleOpenInNewTab}
            >
              新标签页打开
            </Button>
          </Space>
        }
      >
        <Alert
          message="服务连接失败"
          description={
            <div>
              <p>{error}</p>
              <p>您可以尝试：</p>
              <ul>
                <li>检查Kafka服务是否正常运行</li>
                <li>确认Kafka UI服务是否已启动</li>
                <li>在新标签页中直接访问: <a href={kafkaUIUrl} target="_blank" rel="noopener noreferrer">{kafkaUIUrl}</a></li>
              </ul>
            </div>
          }
          type="warning"
          showIcon
        />
      </Card>
    );
  }

  return (
    <Card
      title="Kafka UI 管理界面"
      style={{ margin: '20px' }}
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={handleRefresh}
            loading={loading}
          >
            刷新
          </Button>
          <Button
            icon={<ExportOutlined />}
            onClick={handleOpenInNewTab}
          >
            新标签页打开
          </Button>
        </Space>
      }
    >
      <div style={{ position: 'relative', width: '100%', height: '800px' }}>
        {!iframeLoaded && (
          <div style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            zIndex: 10
          }}>
            <Spin size="large" tip="正在加载Kafka UI..." />
          </div>
        )}
        <iframe
          id="kafka-ui-iframe"
          src={kafkaUIUrl}
          style={{
            width: '100%',
            height: '100%',
            border: '1px solid #d9d9d9',
            borderRadius: '6px',
            display: iframeLoaded ? 'block' : 'none'
          }}
          onLoad={handleIframeLoad}
          onError={handleIframeError}
          title="Kafka UI"
        />
      </div>
    </Card>
  );
};

export default KafkaUIPage;