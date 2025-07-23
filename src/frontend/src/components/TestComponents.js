import React, { useState } from 'react';
import { Card, Button, Space, message, Typography, Alert, Divider } from 'antd';
import { PlayCircleOutlined, CloudServerOutlined, FileTextOutlined } from '@ant-design/icons';
import { kubernetesAPI, ansibleAPI } from '../services/api';

const { Title, Text } = Typography;

const TestComponents = () => {
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState({});

  const testKubernetes = async () => {
    setLoading(true);
    try {
      const response = await kubernetesAPI.getClusters();
      setResults(prev => ({
        ...prev,
        kubernetes: { success: true, data: response.data }
      }));
      message.success('Kubernetes API 连接成功');
    } catch (error) {
      setResults(prev => ({
        ...prev,
        kubernetes: { success: false, error: error.message }
      }));
      message.error('Kubernetes API 连接失败');
    } finally {
      setLoading(false);
    }
  };

  const testAnsible = async () => {
    setLoading(true);
    try {
      const response = await ansibleAPI.getPlaybooks();
      setResults(prev => ({
        ...prev,
        ansible: { success: true, data: response.data }
      }));
      message.success('Ansible API 连接成功');
    } catch (error) {
      setResults(prev => ({
        ...prev,
        ansible: { success: false, error: error.message }
      }));
      message.error('Ansible API 连接失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <Title level={2}>API 连接测试</Title>
        <Text type="secondary">测试前端与后端API的连接状态</Text>
        
        <Divider />
        
        <Space direction="vertical" style={{ width: '100%' }} size="large">
          <Card size="small">
            <Space>
              <CloudServerOutlined style={{ color: '#1890ff', fontSize: '18px' }} />
              <Text strong>Kubernetes API 测试</Text>
              <Button 
                type="primary" 
                icon={<PlayCircleOutlined />}
                onClick={testKubernetes}
                loading={loading}
                size="small"
              >
                测试连接
              </Button>
            </Space>
            
            {results.kubernetes && (
              <div style={{ marginTop: 16 }}>
                {results.kubernetes.success ? (
                  <Alert
                    message="连接成功"
                    description={`成功获取到 ${results.kubernetes.data?.data?.length || 0} 个集群`}
                    type="success"
                    showIcon
                  />
                ) : (
                  <Alert
                    message="连接失败"
                    description={results.kubernetes.error}
                    type="error"
                    showIcon
                  />
                )}
              </div>
            )}
          </Card>

          <Card size="small">
            <Space>
              <FileTextOutlined style={{ color: '#52c41a', fontSize: '18px' }} />
              <Text strong>Ansible API 测试</Text>
              <Button 
                type="primary" 
                icon={<PlayCircleOutlined />}
                onClick={testAnsible}
                loading={loading}
                size="small"
              >
                测试连接
              </Button>
            </Space>
            
            {results.ansible && (
              <div style={{ marginTop: 16 }}>
                {results.ansible.success ? (
                  <Alert
                    message="连接成功"
                    description={`成功获取到 ${results.ansible.data?.data?.length || 0} 个Playbook`}
                    type="success"
                    showIcon
                  />
                ) : (
                  <Alert
                    message="连接失败"
                    description={results.ansible.error}
                    type="error"
                    showIcon
                  />
                )}
              </div>
            )}
          </Card>
        </Space>
      </Card>
    </div>
  );
};

export default TestComponents;
