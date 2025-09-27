import React, { useState, useEffect } from 'react';
import { 
  Card, Button, Space, Typography, Spin, Alert, message, 
  Row, Col, Tag, Breadcrumb, Tooltip
} from 'antd';
import {
  ArrowLeftOutlined, ReloadOutlined, FullscreenOutlined,
  DatabaseOutlined, LinkOutlined, SettingOutlined,
  ExclamationCircleOutlined, CheckCircleOutlined
} from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { objectStorageAPI } from '../services/api';

const { Title, Text } = Typography;

const MinIOConsolePage = () => {
  const navigate = useNavigate();
  const { configId } = useParams();
  
  const [loading, setLoading] = useState(true);
  const [config, setConfig] = useState(null);
  const [connectionStatus, setConnectionStatus] = useState('checking');
  const [iframeUrl, setIframeUrl] = useState('');
  const [isFullscreen, setIsFullscreen] = useState(false);

  // 加载配置信息
  const loadConfig = async () => {
    setLoading(true);
    try {
      const response = await objectStorageAPI.getConfig(configId);
      const configData = response.data?.data;
      
      if (!configData) {
        message.error('配置不存在');
        navigate('/object-storage');
        return;
      }
      
      if (configData.type !== 'minio') {
        message.error('该配置不是MinIO类型');
        navigate('/object-storage');
        return;
      }
      
      setConfig(configData);
      
      // 检查连接状态并构建iframe URL
      await checkConnectionAndSetup(configData);
      
    } catch (error) {
      console.error('加载配置失败:', error);
      message.error('加载配置失败: ' + (error.response?.data?.error || error.message));
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
      
      if (status === 'connected' && configData.web_url) {
        // 构建MinIO控制台URL
        let consoleUrl = configData.web_url;
        
        // 确保URL格式正确
        if (!consoleUrl.startsWith('http://') && !consoleUrl.startsWith('https://')) {
          consoleUrl = `http://${consoleUrl}`;
        }
        
        // 移除末尾的斜杠
        consoleUrl = consoleUrl.replace(/\/$/, '');
        
        setIframeUrl(consoleUrl);
      }
      
    } catch (error) {
      console.error('检查连接状态失败:', error);
      setConnectionStatus('error');
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

  if (!config) {
    return (
      <div style={{ padding: '24px' }}>
        <Alert
          message="配置未找到"
          description="请检查配置是否存在或返回重新选择"
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
                    <Breadcrumb.Item>{config.name}</Breadcrumb.Item>
                  </Breadcrumb>
                </>
              )}
              
              {isFullscreen && (
                <Title level={5} style={{ margin: 0, color: '#1890ff' }}>
                  <DatabaseOutlined /> MinIO - {config.name}
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
            <Space>
              <Text type="secondary">
                <DatabaseOutlined /> {config.endpoint}
              </Text>
              {config.web_url && (
                <Text type="secondary">
                  <LinkOutlined /> {config.web_url}
                </Text>
              )}
            </Space>
          </div>
        )}
      </div>

      {/* MinIO控制台内容区 */}
      <div style={{ flex: 1, position: 'relative', backgroundColor: '#f5f5f5' }}>
        {connectionStatus === 'connected' && iframeUrl ? (
          <iframe
            id="minio-console-iframe"
            src={iframeUrl}
            style={{
              width: '100%',
              height: '100%',
              border: 'none',
              backgroundColor: 'white'
            }}
            onLoad={() => {
              console.log('MinIO控制台加载完成');
            }}
            onError={() => {
              message.error('MinIO控制台加载失败');
            }}
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
                    <Button 
                      icon={<SettingOutlined />}
                      onClick={() => navigate(`/admin/object-storage?edit=${config.id}`)}
                    >
                      编辑配置
                    </Button>
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