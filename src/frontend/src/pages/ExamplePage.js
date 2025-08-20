import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Button, Space, Typography, Alert } from 'antd';
import { ReloadOutlined, ApiOutlined } from '@ant-design/icons';
import { usePageAPIStatus } from '../hooks/useAPIHealth';
import { projectAPI, authAPI } from '../services/api';
import EnhancedLoading from '../components/EnhancedLoading';

const { Title, Text } = Typography;

/**
 * 示例页面 - 展示如何使用API监控功能
 */
const ExamplePage = ({ onError, retryCount }) => {
  const [localData, setLocalData] = useState(null);
  
  // 使用页面级API状态监控
  const {
    loading,
    error,
    data,
    partialErrors,
    retry,
    refresh,
    lastUpdate
  } = usePageAPIStatus([
    () => projectAPI.getProjects(),
    () => authAPI.getCurrentUser()
  ], [retryCount]); // 依赖于重试次数

  // 处理错误
  useEffect(() => {
    if (error && onError) {
      onError(error);
    }
  }, [error, onError]);

  // 如果有错误，让父组件处理
  if (error) {
    return null; // withLazyLoading会显示错误界面
  }

  return (
    <EnhancedLoading loading={loading} error={error}>
      <div style={{ padding: '24px' }}>
        <Title level={2}>
          <ApiOutlined /> API监控示例页面
        </Title>
        
        {partialErrors && partialErrors.length > 0 && (
          <Alert
            type="warning"
            message="部分数据加载失败"
            description={
              <div>
                {partialErrors.map((err, index) => (
                  <div key={index}>
                    {err.message} {err.response?.status && `(${err.response.status})`}
                  </div>
                ))}
              </div>
            }
            style={{ marginBottom: 16 }}
            action={
              <Button size="small" onClick={retry}>
                重试
              </Button>
            }
          />
        )}

        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Card 
              title="项目数据" 
              extra={
                <Button 
                  size="small" 
                  icon={<ReloadOutlined />} 
                  onClick={refresh}
                >
                  刷新
                </Button>
              }
            >
              {data && data[0] ? (
                <div>
                  <Text>加载了 {data[0].data.length} 个项目</Text>
                  <div style={{ marginTop: 8 }}>
                    <Text type="secondary">
                      最后更新: {lastUpdate?.toLocaleTimeString()}
                    </Text>
                  </div>
                </div>
              ) : (
                <Text type="secondary">暂无数据</Text>
              )}
            </Card>
          </Col>

          <Col xs={24} md={12}>
            <Card title="用户信息">
              {data && data[1] ? (
                <div>
                  <Text>用户: {data[1].data.username}</Text>
                  <div style={{ marginTop: 8 }}>
                    <Text type="secondary">
                      角色: {data[1].data.role}
                    </Text>
                  </div>
                </div>
              ) : (
                <Text type="secondary">暂无数据</Text>
              )}
            </Card>
          </Col>
        </Row>

        <Card style={{ marginTop: 16 }} title="API监控说明">
          <Space direction="vertical" style={{ width: '100%' }}>
            <Text>
              本页面演示了以下功能：
            </Text>
            <ul>
              <li>自动API健康检查（每60秒）</li>
              <li>页面级别的API状态监控</li>
              <li>部分API失败的优雅处理</li>
              <li>实时错误反馈和重试机制</li>
              <li>网络状态变化检测</li>
            </ul>
            <Text type="secondary">
              查看浏览器控制台可以看到详细的监控日志
            </Text>
          </Space>
        </Card>
      </div>
    </EnhancedLoading>
  );
};

export default ExamplePage;
