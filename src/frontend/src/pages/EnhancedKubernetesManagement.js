import React, { useState, useEffect } from 'react';
import {
  Layout,
  Row,
  Col,
  Card,
  Select,
  Typography,
  Space,
  Tag,
  Alert,
  Button,
  Divider
} from 'antd';
import {
  CloudServerOutlined,
  ReloadOutlined,
  ApiOutlined,
  DatabaseOutlined
} from '@ant-design/icons';
import axios from 'axios';
import ResourceTree from '../components/kubernetes/ResourceTree';
import ResourceList from '../components/kubernetes/ResourceList';

const { Header, Sider, Content } = Layout;
const { Title, Text } = Typography;
const { Option } = Select;

/**
 * EnhancedKubernetesManagement - 增强的 Kubernetes 资源管理页面
 * 
 * 支持多版本 k8s 兼容、CRD 管理、资源树视图
 */
const EnhancedKubernetesManagement = () => {
  const [clusters, setClusters] = useState([]);
  const [selectedCluster, setSelectedCluster] = useState(null);
  const [selectedResource, setSelectedResource] = useState(null);
  const [clusterVersion, setClusterVersion] = useState(null);
  const [loading, setLoading] = useState(false);

  // 加载集群列表
  useEffect(() => {
    fetchClusters();
  }, []);

  // 加载集群版本信息
  useEffect(() => {
    if (!selectedCluster) return;

    fetchClusterVersion();
  }, [selectedCluster]);

  const fetchClusters = async () => {
    setLoading(true);
    try {
      const response = await axios.get('/api/kubernetes/clusters');
      const clusterData = response.data || [];
      setClusters(clusterData);
      
      // 如果有集群，默认选择第一个
      if (clusterData.length > 0 && !selectedCluster) {
        setSelectedCluster(clusterData[0]);
      }
    } catch (err) {
      console.error('加载集群列表失败:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchClusterVersion = async () => {
    if (!selectedCluster) return;

    try {
      const response = await axios.get(
        `/api/kubernetes/clusters/${selectedCluster.id}/version`
      );
      setClusterVersion(response.data);
    } catch (err) {
      console.error('加载集群版本失败:', err);
      setClusterVersion(null);
    }
  };

  const handleClusterChange = (clusterId) => {
    const cluster = clusters.find(c => c.id === clusterId);
    setSelectedCluster(cluster);
    setSelectedResource(null); // 切换集群时重置选中的资源
    setClusterVersion(null);
  };

  const handleResourceSelect = (resourceData) => {
    setSelectedResource(resourceData);
  };

  return (
    <Layout style={{ minHeight: '100vh', background: '#f0f2f5' }}>
      {/* 顶部导航栏 */}
      <Header style={{ background: '#fff', padding: '0 24px', borderBottom: '1px solid #e8e8e8' }}>
        <Row align="middle" justify="space-between">
          <Col>
            <Space size="large">
              <Title level={3} style={{ margin: 0, display: 'inline-flex', alignItems: 'center' }}>
                <CloudServerOutlined style={{ marginRight: 8 }} />
                Kubernetes 资源管理
              </Title>

              {clusters.length > 0 && (
                <Select
                  value={selectedCluster?.id}
                  onChange={handleClusterChange}
                  style={{ width: 300 }}
                  placeholder="选择集群"
                  loading={loading}
                >
                  {clusters.map(cluster => (
                    <Option key={cluster.id} value={cluster.id}>
                      <Space>
                        <CloudServerOutlined />
                        <Text>{cluster.name}</Text>
                        {cluster.status === 'healthy' && (
                          <Tag color="green">正常</Tag>
                        )}
                      </Space>
                    </Option>
                  ))}
                </Select>
              )}
            </Space>
          </Col>

          <Col>
            <Space>
              {clusterVersion && (
                <Space>
                  <Tag color="blue" icon={<ApiOutlined />}>
                    {clusterVersion.gitVersion}
                  </Tag>
                  <Tag color="purple">
                    Kubernetes {clusterVersion.major}.{clusterVersion.minor}
                  </Tag>
                </Space>
              )}
              
              <Button
                icon={<ReloadOutlined />}
                onClick={() => {
                  fetchClusters();
                  fetchClusterVersion();
                }}
                loading={loading}
              >
                刷新
              </Button>
            </Space>
          </Col>
        </Row>
      </Header>

      {/* 主体内容 */}
      <Layout>
        {/* 左侧资源树 */}
        <Sider 
          width={400} 
          style={{ 
            background: '#fff', 
            borderRight: '1px solid #e8e8e8',
            overflow: 'auto',
            height: 'calc(100vh - 64px)',
            position: 'sticky',
            top: 64,
          }}
        >
          <Card 
            bordered={false}
            bodyStyle={{ padding: 0 }}
            title={
              <Space>
                <DatabaseOutlined />
                <Text strong>资源树</Text>
              </Space>
            }
          >
            {!selectedCluster && (
              <Alert
                message="请选择集群"
                description="在顶部下拉框中选择一个 Kubernetes 集群以查看其资源"
                type="info"
                showIcon
                style={{ margin: 16 }}
              />
            )}

            {selectedCluster && (
              <ResourceTree
                clusterId={selectedCluster.id}
                onResourceSelect={handleResourceSelect}
              />
            )}
          </Card>
        </Sider>

        {/* 右侧资源列表 */}
        <Content style={{ background: '#fff', minHeight: 'calc(100vh - 64px)' }}>
          {!selectedCluster && (
            <div style={{ padding: '100px', textAlign: 'center' }}>
              <CloudServerOutlined style={{ fontSize: 64, color: '#ccc', marginBottom: 16 }} />
              <Title level={4}>请选择集群</Title>
              <Text type="secondary">
                选择一个 Kubernetes 集群以开始管理其资源
              </Text>
            </div>
          )}

          {selectedCluster && !selectedResource && (
            <div style={{ padding: '100px', textAlign: 'center' }}>
              <DatabaseOutlined style={{ fontSize: 64, color: '#ccc', marginBottom: 16 }} />
              <Title level={4}>请选择资源类型</Title>
              <Text type="secondary">
                从左侧资源树中选择一个资源类型以查看其实例列表
              </Text>
            </div>
          )}

          {selectedCluster && selectedResource && (
            <div>
              <div style={{ 
                padding: '16px 24px', 
                background: '#fafafa', 
                borderBottom: '1px solid #e8e8e8' 
              }}>
                <Space split={<Divider type="vertical" />}>
                  <Space>
                    <Text strong>集群:</Text>
                    <Tag color="blue">{selectedCluster.name}</Tag>
                  </Space>
                  
                  {selectedResource.resource && (
                    <>
                      <Space>
                        <Text strong>资源类型:</Text>
                        <Tag color="green">{selectedResource.resource.kind}</Tag>
                      </Space>
                      <Space>
                        <Text strong>API组:</Text>
                        <Tag>{selectedResource.resource.groupVersion}</Tag>
                      </Space>
                    </>
                  )}
                  
                  {selectedResource.crd && (
                    <>
                      <Space>
                        <Text strong>CRD:</Text>
                        <Tag color="red">{selectedResource.crd.kind}</Tag>
                      </Space>
                      <Space>
                        <Text strong>组:</Text>
                        <Tag>{selectedResource.crd.group}</Tag>
                      </Space>
                      <Space>
                        <Text strong>版本:</Text>
                        <Tag>{selectedResource.crd.versions?.join(', ')}</Tag>
                      </Space>
                    </>
                  )}
                </Space>
              </div>

              <ResourceList
                clusterId={selectedCluster.id}
                resourceInfo={selectedResource}
              />
            </div>
          )}
        </Content>
      </Layout>
    </Layout>
  );
};

export default EnhancedKubernetesManagement;
