import React, { useState, useEffect } from 'react';
import {
  Card, Row, Col, Button, Typography, Space, Alert, Spin, 
  Tabs, Statistic, Tag, List, Avatar, Progress, message
} from 'antd';
import {
  CloudServerOutlined, DatabaseOutlined, SettingOutlined,
  PlusOutlined, EyeOutlined, ApiOutlined, MonitorOutlined,
  SafetyOutlined, LinkOutlined, CheckCircleOutlined, ReloadOutlined
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { objectStorageAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';

const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;

const ObjectStoragePage = () => {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [storageConfigs, setStorageConfigs] = useState([]);
  const [activeConfig, setActiveConfig] = useState(null);
  const [statistics, setStatistics] = useState(null);
  const [lastRefresh, setLastRefresh] = useState(Date.now());
  const [autoRefreshEnabled, setAutoRefreshEnabled] = useState(true);

  // 加载存储配置
  const loadStorageConfigs = async (silent = false) => {
    if (!silent) {
      setLoading(true);
    }
    try {
      const response = await objectStorageAPI.getConfigs();
      const configs = response.data?.data || [];
      setStorageConfigs(configs);
      
      // 设置默认激活的配置
      const activeConf = configs.find(c => c.is_active) || configs[0];
      setActiveConfig(activeConf);
      
      // 如果有激活配置，加载统计信息
      if (activeConf) {
        await loadStatistics(activeConf.id);
      }
      
      setLastRefresh(Date.now());
    } catch (error) {
      console.error('加载存储配置失败:', error);
      if (!silent) {
        message.error('加载存储配置失败: ' + (error.response?.data?.error || error.message));
      }
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  };

  // 加载统计信息
  const loadStatistics = async (configId) => {
    try {
      const response = await objectStorageAPI.getStatistics(configId);
      setStatistics(response.data?.data);
    } catch (error) {
      console.error('加载统计信息失败:', error);
    }
  };

  // 初始加载
  useEffect(() => {
    loadStorageConfigs();
  }, []);

  // 自动刷新机制
  useEffect(() => {
    if (!autoRefreshEnabled) {
      console.log('对象存储自动刷新已禁用');
      return;
    }

    console.log('启动对象存储自动刷新，间隔: 30秒');
    const intervalId = setInterval(() => {
      console.log('自动刷新对象存储配置...');
      loadStorageConfigs(true); // 静默刷新
    }, 30000); // 每30秒刷新一次

    return () => {
      console.log('清除对象存储自动刷新定时器');
      clearInterval(intervalId);
    };
  }, [autoRefreshEnabled]); // eslint-disable-line react-hooks/exhaustive-deps

  // 页面可见性变化时刷新
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (!document.hidden && autoRefreshEnabled) {
        console.log('页面变为可见，刷新对象存储配置...');
        loadStorageConfigs(true);
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, [autoRefreshEnabled]);

  // 存储类型配置
  const storageTypeConfigs = {
    seaweedfs: {
      name: t('objectStorage.seaweedfs'),
      icon: <DatabaseOutlined />,
      color: '#00C853',
      description: t('objectStorage.seaweedfsDesc')
    },
    minio: {
      name: t('objectStorage.minio'),
      icon: <DatabaseOutlined />,
      color: '#C73A2F',
      description: t('objectStorage.minioDesc')
    },
    aws_s3: {
      name: t('objectStorage.awsS3'),
      icon: <CloudServerOutlined />,
      color: '#FF9900',
      description: t('objectStorage.awsS3Desc')
    },
    aliyun_oss: {
      name: t('objectStorage.aliyunOss'),
      icon: <CloudServerOutlined />,
      color: '#FF6A00',
      description: t('objectStorage.aliyunOssDesc')
    },
    tencent_cos: {
      name: t('objectStorage.tencentCos'),
      icon: <CloudServerOutlined />,
      color: '#006EFF',
      description: t('objectStorage.tencentCosDesc')
    }
  };

  // 获取存储类型配置
  const getStorageTypeConfig = (type) => {
    return storageTypeConfigs[type] || {
      name: type.toUpperCase(),
      icon: <CloudServerOutlined />,
      color: '#1890ff',
      description: t('objectStorage.storageServices')
    };
  };

  // 渲染配置卡片
  const renderConfigCard = (config) => {
    const typeConfig = getStorageTypeConfig(config.type);
    const isActive = activeConfig?.id === config.id;

    return (
      <Card
        key={config.id}
        size="small"
        style={{ 
          marginBottom: '16px',
          border: isActive ? '2px solid #1890ff' : '1px solid #d9d9d9',
          boxShadow: isActive ? '0 4px 12px rgba(24, 144, 255, 0.15)' : undefined
        }}
        bodyStyle={{ padding: '16px' }}
      >
        <Row justify="space-between" align="middle">
          <Col flex="1">
            <Space>
              <Avatar 
                style={{ backgroundColor: typeConfig.color, color: 'white' }}
                icon={typeConfig.icon}
              />
              <div>
                <Text strong>{config.name}</Text>
                <br />
                <Text type="secondary" style={{ fontSize: '12px' }}>
                  {typeConfig.name} • {config.endpoint}
                </Text>
              </div>
            </Space>
          </Col>
          <Col>
            <Space>
              {config.is_active && (
                <Tag color="green" icon={<CheckCircleOutlined />}>
                  {t('objectStorage.currentActive')}
                </Tag>
              )}
              <Tag color={config.status === 'connected' ? 'green' : 'red'}>
                {config.status === 'connected' ? t('objectStorage.connected') : t('objectStorage.notConnected')}
              </Tag>
              <Button
                size="small"
                icon={<EyeOutlined />}
                onClick={() => handleAccessStorage(config)}
                disabled={config.status !== 'connected'}
              >
                {t('objectStorage.access')}
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>
    );
  };

  // 处理访问存储
  const handleAccessStorage = (config) => {
    // SeaweedFS 和 MinIO 支持 Web 控制台访问
    // SeaweedFS 通过 Filer 提供 Web UI，即使没有配置 web_url 也可以通过同源代理访问
    if (config.type === 'seaweedfs' || (config.type === 'minio' && config.web_url)) {
      // 跳转到存储控制台页面
      navigate(`/object-storage/console/${config.id}`);
    } else {
      message.info(t('objectStorage.notSupportWebConsole'));
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" />
        <div style={{ marginTop: '16px' }}>
          <Text>{t('objectStorage.loading')}</Text>
        </div>
      </div>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div>
          <Title level={2}>
            <CloudServerOutlined style={{ marginRight: '8px', color: '#1890ff' }} />
            {t('objectStorage.title')}
          </Title>
          <Paragraph type="secondary">
            {t('objectStorage.subtitle')}
            {lastRefresh && (
              <span style={{ marginLeft: '16px', fontSize: '12px' }}>
                {t('objectStorage.lastUpdate')}: {new Date(lastRefresh).toLocaleTimeString()}
              </span>
            )}
          </Paragraph>
        </div>
        <Space>
          <Button
            icon={<ReloadOutlined spin={loading} />}
            onClick={() => loadStorageConfigs()}
            loading={loading}
          >
            {t('objectStorage.refresh')}
          </Button>
          <Button
            type={autoRefreshEnabled ? "primary" : "default"}
            onClick={() => setAutoRefreshEnabled(!autoRefreshEnabled)}
            ghost={autoRefreshEnabled}
          >
            {autoRefreshEnabled ? `🔄 ${t('objectStorage.autoRefresh')}` : `⏸️ ${t('objectStorage.paused')}`}
          </Button>
          <Button 
            icon={<SettingOutlined />}
            onClick={() => navigate('/admin/object-storage')}
          >
            {t('objectStorage.storageConfig')}
          </Button>
          <Button 
            type="primary" 
            icon={<PlusOutlined />}
            onClick={() => navigate('/admin/object-storage?action=add')}
          >
            {t('objectStorage.addStorage')}
          </Button>
        </Space>
      </div>

      {storageConfigs.length === 0 ? (
        <Card>
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <CloudServerOutlined style={{ fontSize: '64px', color: '#d9d9d9', marginBottom: '16px' }} />
            <Title level={4} type="secondary">{t('objectStorage.noStorageConfig')}</Title>
            <Paragraph type="secondary">
              {t('objectStorage.noStorageConfigDesc')}
            </Paragraph>
            <Button 
              type="primary" 
              icon={<PlusOutlined />}
              onClick={() => navigate('/admin/object-storage?action=add')}
            >
              {t('objectStorage.configNow')}
            </Button>
          </div>
        </Card>
      ) : (
        <Tabs defaultActiveKey="overview">
          <TabPane
            tab={
              <span>
                <MonitorOutlined />
                {t('objectStorage.overview')}
              </span>
            }
            key="overview"
          >
            <Row gutter={[16, 16]}>
              <Col xs={24} lg={16}>
                <Card title={t('objectStorage.storageList')} style={{ marginBottom: '16px' }}>
                  {storageConfigs.map(renderConfigCard)}
                </Card>
              </Col>
              
              <Col xs={24} lg={8}>
                {statistics && (
                  <Card title={t('objectStorage.storageStats')} style={{ marginBottom: '16px' }}>
                    <Row gutter={16}>
                      <Col span={12}>
                        <Statistic
                          title={t('objectStorage.bucketCount')}
                          value={statistics.bucket_count || 0}
                          prefix={<DatabaseOutlined />}
                        />
                      </Col>
                      <Col span={12}>
                        <Statistic
                          title={t('objectStorage.objectCount')}
                          value={statistics.object_count || 0}
                          prefix={<ApiOutlined />}
                        />
                      </Col>
                    </Row>
                    <div style={{ marginTop: '16px' }}>
                      <Text>{t('objectStorage.usedSpace')}</Text>
                      <Progress
                        percent={statistics.usage_percent || 0}
                        format={() => statistics.used_space || '0 B'}
                        style={{ marginTop: '4px' }}
                      />
                    </div>
                    <div style={{ marginTop: '12px' }}>
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        {t('objectStorage.totalCapacity')}: {statistics.total_space || 'N/A'}
                      </Text>
                    </div>
                  </Card>
                )}

                <Card title={t('objectStorage.quickActions')}>
                  <Space direction="vertical" style={{ width: '100%' }}>
                    {activeConfig && (activeConfig.type === 'seaweedfs' || activeConfig.type === 'minio') && (
                      <Button
                        block
                        icon={<LinkOutlined />}
                        onClick={() => handleAccessStorage(activeConfig)}
                      >
                        {activeConfig.type === 'seaweedfs' 
                          ? t('objectStorage.accessSeaweedFSConsole') 
                          : t('objectStorage.accessMinioConsole')}
                      </Button>
                    )}
                    <Button
                      block
                      icon={<SettingOutlined />}
                      onClick={() => navigate('/admin/object-storage')}
                    >
                      {t('objectStorage.manageStorageConfig')}
                    </Button>
                    <Button
                      block
                      icon={<SafetyOutlined />}
                      onClick={() => message.info(t('objectStorage.featureInDevelopment'))}
                    >
                      {t('objectStorage.permissionManagement')}
                    </Button>
                  </Space>
                </Card>
              </Col>
            </Row>
          </TabPane>

          <TabPane
            tab={
              <span>
                <DatabaseOutlined />
                {t('objectStorage.storageServices')}
              </span>
            }
            key="services"
          >
            <Row gutter={[16, 16]}>
              {Object.entries(storageTypeConfigs).map(([type, config]) => (
                <Col xs={24} sm={12} lg={6} key={type}>
                  <Card
                    hoverable
                    style={{ textAlign: 'center', height: '160px' }}
                    bodyStyle={{ padding: '24px' }}
                  >
                    <div style={{ color: config.color, fontSize: '32px', marginBottom: '12px' }}>
                      {config.icon}
                    </div>
                    <Title level={5} style={{ margin: 0, marginBottom: '4px' }}>
                      {config.name}
                    </Title>
                    <Text type="secondary" style={{ fontSize: '12px' }}>
                      {config.description}
                    </Text>
                  </Card>
                </Col>
              ))}
            </Row>

            <Alert
              style={{ marginTop: '16px' }}
              message={t('objectStorage.supportMultipleStorage')}
              description={t('objectStorage.supportMultipleStorageDesc')}
              type="info"
              showIcon
            />
          </TabPane>
        </Tabs>
      )}
    </div>
  );
};

export default ObjectStoragePage;