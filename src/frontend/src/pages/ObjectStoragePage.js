import React, { useState, useEffect } from 'react';
import {
  Card, Row, Col, Button, Typography, Space, Alert, Spin, 
  Tabs, Statistic, Tag, List, Avatar, Progress, message
} from 'antd';
import {
  CloudServerOutlined, DatabaseOutlined, SettingOutlined,
  PlusOutlined, EyeOutlined, ApiOutlined, MonitorOutlined,
  SafetyOutlined, LinkOutlined, CheckCircleOutlined
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { objectStorageAPI } from '../services/api';

const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;

const ObjectStoragePage = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [storageConfigs, setStorageConfigs] = useState([]);
  const [activeConfig, setActiveConfig] = useState(null);
  const [statistics, setStatistics] = useState(null);

  // 加载存储配置
  const loadStorageConfigs = async () => {
    setLoading(true);
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
    } catch (error) {
      console.error('加载存储配置失败:', error);
      message.error('加载存储配置失败: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
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

  useEffect(() => {
    loadStorageConfigs();
  }, []);

  // 存储类型配置
  const storageTypeConfigs = {
    minio: {
      name: 'MinIO',
      icon: <DatabaseOutlined />,
      color: '#C73A2F',
      description: '高性能分布式对象存储，兼容S3 API'
    },
    aws_s3: {
      name: 'Amazon S3',
      icon: <CloudServerOutlined />,
      color: '#FF9900',
      description: 'AWS原生对象存储服务'
    },
    aliyun_oss: {
      name: '阿里云OSS',
      icon: <CloudServerOutlined />,
      color: '#FF6A00',
      description: '阿里云对象存储服务'
    },
    tencent_cos: {
      name: '腾讯云COS',
      icon: <CloudServerOutlined />,
      color: '#006EFF',
      description: '腾讯云对象存储'
    }
  };

  // 获取存储类型配置
  const getStorageTypeConfig = (type) => {
    return storageTypeConfigs[type] || {
      name: type.toUpperCase(),
      icon: <CloudServerOutlined />,
      color: '#1890ff',
      description: '对象存储服务'
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
                  当前激活
                </Tag>
              )}
              <Tag color={config.status === 'connected' ? 'green' : 'red'}>
                {config.status === 'connected' ? '已连接' : '未连接'}
              </Tag>
              <Button
                size="small"
                icon={<EyeOutlined />}
                onClick={() => handleAccessStorage(config)}
                disabled={config.status !== 'connected'}
              >
                访问
              </Button>
            </Space>
          </Col>
        </Row>
      </Card>
    );
  };

  // 处理访问存储
  const handleAccessStorage = (config) => {
    if (config.type === 'minio' && config.web_url) {
      // 跳转到Minio控制台页面
      navigate(`/object-storage/minio/${config.id}`);
    } else {
      message.info('该存储类型暂不支持Web控制台访问');
    }
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" />
        <div style={{ marginTop: '16px' }}>
          <Text>加载对象存储配置中...</Text>
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
            对象存储管理
          </Title>
          <Paragraph type="secondary">
            统一管理MinIO、S3等各种对象存储服务，提供文件上传、下载和管理功能
          </Paragraph>
        </div>
        <Space>
          <Button 
            icon={<SettingOutlined />}
            onClick={() => navigate('/admin/object-storage')}
          >
            存储配置
          </Button>
          <Button 
            type="primary" 
            icon={<PlusOutlined />}
            onClick={() => navigate('/admin/object-storage?action=add')}
          >
            添加存储
          </Button>
        </Space>
      </div>

      {storageConfigs.length === 0 ? (
        <Card>
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <CloudServerOutlined style={{ fontSize: '64px', color: '#d9d9d9', marginBottom: '16px' }} />
            <Title level={4} type="secondary">尚未配置对象存储</Title>
            <Paragraph type="secondary">
              请先配置至少一个对象存储服务才能使用此功能
            </Paragraph>
            <Button 
              type="primary" 
              icon={<PlusOutlined />}
              onClick={() => navigate('/admin/object-storage?action=add')}
            >
              立即配置
            </Button>
          </div>
        </Card>
      ) : (
        <Tabs defaultActiveKey="overview">
          <TabPane
            tab={
              <span>
                <MonitorOutlined />
                概览
              </span>
            }
            key="overview"
          >
            <Row gutter={[16, 16]}>
              <Col xs={24} lg={16}>
                <Card title="存储服务列表" style={{ marginBottom: '16px' }}>
                  {storageConfigs.map(renderConfigCard)}
                </Card>
              </Col>
              
              <Col xs={24} lg={8}>
                {statistics && (
                  <Card title="存储统计" style={{ marginBottom: '16px' }}>
                    <Row gutter={16}>
                      <Col span={12}>
                        <Statistic
                          title="存储桶数量"
                          value={statistics.bucket_count || 0}
                          prefix={<DatabaseOutlined />}
                        />
                      </Col>
                      <Col span={12}>
                        <Statistic
                          title="对象数量"
                          value={statistics.object_count || 0}
                          prefix={<ApiOutlined />}
                        />
                      </Col>
                    </Row>
                    <div style={{ marginTop: '16px' }}>
                      <Text>已用存储空间</Text>
                      <Progress
                        percent={statistics.usage_percent || 0}
                        format={() => statistics.used_space || '0 B'}
                        style={{ marginTop: '4px' }}
                      />
                    </div>
                    <div style={{ marginTop: '12px' }}>
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        总容量: {statistics.total_space || 'N/A'}
                      </Text>
                    </div>
                  </Card>
                )}

                <Card title="快速操作">
                  <Space direction="vertical" style={{ width: '100%' }}>
                    {activeConfig && activeConfig.type === 'minio' && (
                      <Button
                        block
                        icon={<LinkOutlined />}
                        onClick={() => handleAccessStorage(activeConfig)}
                      >
                        访问MinIO控制台
                      </Button>
                    )}
                    <Button
                      block
                      icon={<SettingOutlined />}
                      onClick={() => navigate('/admin/object-storage')}
                    >
                      管理存储配置
                    </Button>
                    <Button
                      block
                      icon={<SafetyOutlined />}
                      onClick={() => message.info('权限管理功能开发中')}
                    >
                      权限管理
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
                存储服务
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
              message="支持多种对象存储"
              description="系统支持MinIO、AWS S3、阿里云OSS、腾讯云COS等多种对象存储服务，可以根据需要配置和切换。"
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