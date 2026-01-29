import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  Card, 
  Row, 
  Col, 
  Button, 
  Table, 
  Tag, 
  Space, 
  message, 
  Modal, 
  Form, 
  Input, 
  Select,
  Tooltip,
  Empty,
  Statistic,
  Tabs,
  Alert,
  Badge,
  Typography,
  Drawer
} from 'antd';
import { 
  SyncOutlined, 
  CloudServerOutlined, 
  CheckCircleOutlined,
  CloseCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
  BranchesOutlined,
  AppstoreOutlined,
  DeploymentUnitOutlined,
  GithubOutlined,
  DesktopOutlined
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import api from '../services/api';

const { Title, Text } = Typography;
const { TabPane } = Tabs;
const { Option } = Select;

// ArgoCD 应用状态映射
const healthStatusColors = {
  Healthy: 'success',
  Progressing: 'processing',
  Degraded: 'error',
  Suspended: 'warning',
  Missing: 'default',
  Unknown: 'default'
};

const syncStatusColors = {
  Synced: 'success',
  OutOfSync: 'warning',
  Unknown: 'default'
};

const ArgoCDManagement = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [applications, setApplications] = useState([]);
  const [repositories, setRepositories] = useState([]);
  const [clusters, setClusters] = useState([]);
  const [projects, setProjects] = useState([]);
  const [statistics, setStatistics] = useState({
    totalApps: 0,
    healthyApps: 0,
    syncedApps: 0,
    repositories: 0,
    clusters: 0
  });
  const [selectedApp, setSelectedApp] = useState(null);
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [addAppModalVisible, setAddAppModalVisible] = useState(false);
  const [addRepoModalVisible, setAddRepoModalVisible] = useState(false);
  const [activeTab, setActiveTab] = useState('applications');
  const [form] = Form.useForm();
  const [repoForm] = Form.useForm();

  // 获取 ArgoCD 数据
  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const [appsRes, reposRes, clustersRes, projectsRes] = await Promise.all([
        api.get('/argocd/applications').catch(() => ({ data: { items: [] } })),
        api.get('/argocd/repositories').catch(() => ({ data: { items: [] } })),
        api.get('/argocd/clusters').catch(() => ({ data: { items: [] } })),
        api.get('/argocd/projects').catch(() => ({ data: { items: [] } }))
      ]);

      const apps = appsRes.data?.items || [];
      const repos = reposRes.data?.items || [];
      const clusterList = clustersRes.data?.items || [];
      const projectList = projectsRes.data?.items || [];

      setApplications(apps);
      setRepositories(repos);
      setClusters(clusterList);
      setProjects(projectList);

      // 计算统计信息
      setStatistics({
        totalApps: apps.length,
        healthyApps: apps.filter(a => a.status?.health?.status === 'Healthy').length,
        syncedApps: apps.filter(a => a.status?.sync?.status === 'Synced').length,
        repositories: repos.length,
        clusters: clusterList.length
      });
    } catch (error) {
      console.error('Failed to fetch ArgoCD data:', error);
      message.error(t('argocd.fetchFailed', 'Failed to fetch ArgoCD data'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  // 同步应用
  const handleSyncApp = async (appName) => {
    try {
      await api.post(`/argocd/applications/${appName}/sync`);
      message.success(t('argocd.syncStarted', 'Sync started for {app}', { app: appName }));
      fetchData();
    } catch (error) {
      message.error(t('argocd.syncFailed', 'Failed to sync application'));
    }
  };

  // 刷新应用
  const handleRefreshApp = async (appName) => {
    try {
      await api.post(`/argocd/applications/${appName}/refresh`);
      message.success(t('argocd.refreshed', 'Application refreshed'));
      fetchData();
    } catch (error) {
      message.error(t('argocd.refreshFailed', 'Failed to refresh application'));
    }
  };

  // 删除应用
  const handleDeleteApp = async (appName) => {
    Modal.confirm({
      title: t('argocd.confirmDelete', 'Delete Application'),
      content: t('argocd.confirmDeleteContent', 'Are you sure you want to delete {app}?', { app: appName }),
      okText: t('common.delete', 'Delete'),
      okType: 'danger',
      onOk: async () => {
        try {
          await api.delete(`/argocd/applications/${appName}`);
          message.success(t('argocd.deleted', 'Application deleted'));
          fetchData();
        } catch (error) {
          message.error(t('argocd.deleteFailed', 'Failed to delete application'));
        }
      }
    });
  };

  // 创建应用
  const handleCreateApp = async (values) => {
    try {
      await api.post('/argocd/applications', {
        metadata: {
          name: values.name,
          namespace: 'argocd'
        },
        spec: {
          project: values.project || 'default',
          source: {
            repoURL: values.repoURL,
            path: values.path,
            targetRevision: values.targetRevision || 'HEAD'
          },
          destination: {
            server: values.destinationServer || 'https://kubernetes.default.svc',
            namespace: values.destinationNamespace
          },
          syncPolicy: values.autoSync ? {
            automated: {
              prune: true,
              selfHeal: true
            }
          } : undefined
        }
      });
      message.success(t('argocd.created', 'Application created'));
      setAddAppModalVisible(false);
      form.resetFields();
      fetchData();
    } catch (error) {
      message.error(t('argocd.createFailed', 'Failed to create application'));
    }
  };

  // 添加仓库
  const handleAddRepo = async (values) => {
    try {
      await api.post('/argocd/repositories', {
        repo: values.url,
        username: values.username,
        password: values.password,
        type: values.type || 'git',
        insecure: values.insecure
      });
      message.success(t('argocd.repoAdded', 'Repository added'));
      setAddRepoModalVisible(false);
      repoForm.resetFields();
      fetchData();
    } catch (error) {
      message.error(t('argocd.repoAddFailed', 'Failed to add repository'));
    }
  };

  // 应用列表列配置
  const appColumns = [
    {
      title: t('argocd.appName', 'Application'),
      dataIndex: ['metadata', 'name'],
      key: 'name',
      render: (name, record) => (
        <Space>
          <AppstoreOutlined />
          <Button 
            type="link" 
            style={{ padding: 0 }}
            onClick={() => {
              setSelectedApp(record);
              setDetailDrawerVisible(true);
            }}
          >
            {name}
          </Button>
        </Space>
      )
    },
    {
      title: t('argocd.project', 'Project'),
      dataIndex: ['spec', 'project'],
      key: 'project',
      render: (project) => <Tag>{project || 'default'}</Tag>
    },
    {
      title: t('argocd.health', 'Health'),
      dataIndex: ['status', 'health', 'status'],
      key: 'health',
      render: (status) => (
        <Badge 
          status={healthStatusColors[status] || 'default'} 
          text={status || 'Unknown'} 
        />
      )
    },
    {
      title: t('argocd.sync', 'Sync Status'),
      dataIndex: ['status', 'sync', 'status'],
      key: 'sync',
      render: (status) => (
        <Tag color={syncStatusColors[status] || 'default'}>
          {status || 'Unknown'}
        </Tag>
      )
    },
    {
      title: t('argocd.repository', 'Repository'),
      dataIndex: ['spec', 'source', 'repoURL'],
      key: 'repoURL',
      ellipsis: true,
      render: (url) => (
        <Tooltip title={url}>
          <Space>
            <GithubOutlined />
            <Text ellipsis style={{ maxWidth: 200 }}>{url}</Text>
          </Space>
        </Tooltip>
      )
    },
    {
      title: t('argocd.path', 'Path'),
      dataIndex: ['spec', 'source', 'path'],
      key: 'path',
      render: (path) => <Text code>{path}</Text>
    },
    {
      title: t('argocd.destination', 'Destination'),
      dataIndex: ['spec', 'destination', 'namespace'],
      key: 'destination',
      render: (ns, record) => (
        <Tag color="blue">{ns || 'default'}</Tag>
      )
    },
    {
      title: t('common.actions', 'Actions'),
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Tooltip title={t('argocd.sync', 'Sync')}>
            <Button 
              type="primary" 
              icon={<SyncOutlined />} 
              size="small"
              onClick={() => handleSyncApp(record.metadata.name)}
            />
          </Tooltip>
          <Tooltip title={t('argocd.refresh', 'Refresh')}>
            <Button 
              icon={<ReloadOutlined />} 
              size="small"
              onClick={() => handleRefreshApp(record.metadata.name)}
            />
          </Tooltip>
          <Tooltip title={t('common.delete', 'Delete')}>
            <Button 
              danger 
              icon={<CloseCircleOutlined />} 
              size="small"
              onClick={() => handleDeleteApp(record.metadata.name)}
            />
          </Tooltip>
        </Space>
      )
    }
  ];

  // 仓库列表列配置
  const repoColumns = [
    {
      title: t('argocd.repoURL', 'Repository URL'),
      dataIndex: 'repo',
      key: 'repo',
      render: (url) => (
        <Space>
          <GithubOutlined />
          <a href={url} target="_blank" rel="noopener noreferrer">{url}</a>
        </Space>
      )
    },
    {
      title: t('argocd.type', 'Type'),
      dataIndex: 'type',
      key: 'type',
      render: (type) => <Tag>{type || 'git'}</Tag>
    },
    {
      title: t('argocd.connectionStatus', 'Connection'),
      dataIndex: 'connectionState',
      key: 'connectionState',
      render: (state) => (
        <Badge 
          status={state?.status === 'Successful' ? 'success' : 'error'}
          text={state?.status || 'Unknown'}
        />
      )
    },
    {
      title: t('common.actions', 'Actions'),
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button 
            danger 
            size="small"
            onClick={() => {
              Modal.confirm({
                title: t('argocd.deleteRepo', 'Delete Repository'),
                onOk: async () => {
                  await api.delete(`/argocd/repositories/${encodeURIComponent(record.repo)}`);
                  fetchData();
                }
              });
            }}
          >
            {t('common.delete', 'Delete')}
          </Button>
        </Space>
      )
    }
  ];

  // 集群列表列配置
  const clusterColumns = [
    {
      title: t('argocd.clusterName', 'Cluster Name'),
      dataIndex: 'name',
      key: 'name',
      render: (name) => (
        <Space>
          <CloudServerOutlined />
          {name}
        </Space>
      )
    },
    {
      title: t('argocd.server', 'Server'),
      dataIndex: 'server',
      key: 'server',
      ellipsis: true
    },
    {
      title: t('argocd.connectionStatus', 'Status'),
      dataIndex: ['connectionState', 'status'],
      key: 'status',
      render: (status) => (
        <Badge 
          status={status === 'Successful' ? 'success' : 'error'}
          text={status || 'Unknown'}
        />
      )
    },
    {
      title: t('argocd.k8sVersion', 'K8s Version'),
      dataIndex: ['serverVersion'],
      key: 'version',
      render: (version) => version || '-'
    }
  ];

  return (
    <div style={{ padding: '24px' }}>
      {/* 页面标题 */}
      <Row justify="space-between" align="middle" style={{ marginBottom: 24 }}>
        <Col>
          <Title level={2}>
            <DeploymentUnitOutlined /> {t('argocd.title', 'ArgoCD GitOps Management')}
          </Title>
          <Text type="secondary">
            {t('argocd.description', 'Manage GitOps deployments with ArgoCD')}
          </Text>
        </Col>
        <Col>
          <Space>
            <Button 
              type="primary"
              icon={<DesktopOutlined />}
              onClick={() => navigate('/argocd-ui')}
            >
              {t('argocd.openEmbedUI', '嵌入式控制台')}
            </Button>
            <Button 
              icon={<ReloadOutlined />} 
              onClick={fetchData}
              loading={loading}
            >
              {t('common.refresh', '刷新')}
            </Button>
          </Space>
        </Col>
      </Row>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title={t('argocd.totalApps', 'Total Applications')}
              value={statistics.totalApps}
              prefix={<AppstoreOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title={t('argocd.healthyApps', 'Healthy')}
              value={statistics.healthyApps}
              valueStyle={{ color: '#52c41a' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title={t('argocd.syncedApps', 'Synced')}
              value={statistics.syncedApps}
              valueStyle={{ color: '#1890ff' }}
              prefix={<SyncOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title={t('argocd.repositories', 'Repositories')}
              value={statistics.repositories}
              prefix={<BranchesOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 主要内容 */}
      <Card>
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane 
            tab={<span><AppstoreOutlined />{t('argocd.applications', 'Applications')}</span>} 
            key="applications"
          >
            <div style={{ marginBottom: 16 }}>
              <Space>
                <Button 
                  type="primary" 
                  icon={<PlusOutlined />}
                  onClick={() => setAddAppModalVisible(true)}
                >
                  {t('argocd.createApp', 'Create Application')}
                </Button>
                <Button 
                  icon={<ReloadOutlined />}
                  onClick={fetchData}
                  loading={loading}
                >
                  {t('common.refresh', 'Refresh')}
                </Button>
              </Space>
            </div>
            <Table
              dataSource={applications}
              columns={appColumns}
              rowKey={(record) => record.metadata?.name}
              loading={loading}
              pagination={{ pageSize: 10 }}
              locale={{
                emptyText: <Empty description={t('argocd.noApps', 'No applications')} />
              }}
            />
          </TabPane>

          <TabPane 
            tab={<span><BranchesOutlined />{t('argocd.repositories', 'Repositories')}</span>} 
            key="repositories"
          >
            <div style={{ marginBottom: 16 }}>
              <Space>
                <Button 
                  type="primary" 
                  icon={<PlusOutlined />}
                  onClick={() => setAddRepoModalVisible(true)}
                >
                  {t('argocd.addRepo', 'Add Repository')}
                </Button>
                <Button 
                  icon={<ReloadOutlined />}
                  onClick={fetchData}
                  loading={loading}
                >
                  {t('common.refresh', 'Refresh')}
                </Button>
              </Space>
            </div>
            <Table
              dataSource={repositories}
              columns={repoColumns}
              rowKey="repo"
              loading={loading}
              pagination={{ pageSize: 10 }}
            />
          </TabPane>

          <TabPane 
            tab={<span><CloudServerOutlined />{t('argocd.clusters', 'Clusters')}</span>} 
            key="clusters"
          >
            <Table
              dataSource={clusters}
              columns={clusterColumns}
              rowKey="server"
              loading={loading}
              pagination={{ pageSize: 10 }}
            />
          </TabPane>
        </Tabs>
      </Card>

      {/* 创建应用模态框 */}
      <Modal
        title={t('argocd.createApp', 'Create Application')}
        open={addAppModalVisible}
        onCancel={() => setAddAppModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form form={form} layout="vertical" onFinish={handleCreateApp}>
          <Form.Item
            name="name"
            label={t('argocd.appName', 'Application Name')}
            rules={[{ required: true }]}
          >
            <Input placeholder="my-app" />
          </Form.Item>
          <Form.Item
            name="project"
            label={t('argocd.project', 'Project')}
          >
            <Select placeholder="default">
              {projects.map(p => (
                <Option key={p.metadata?.name} value={p.metadata?.name}>
                  {p.metadata?.name}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item
            name="repoURL"
            label={t('argocd.repoURL', 'Repository URL')}
            rules={[{ required: true }]}
          >
            <Input placeholder="https://github.com/org/repo.git" />
          </Form.Item>
          <Form.Item
            name="path"
            label={t('argocd.path', 'Path')}
            rules={[{ required: true }]}
          >
            <Input placeholder="./manifests" />
          </Form.Item>
          <Form.Item
            name="targetRevision"
            label={t('argocd.revision', 'Target Revision')}
          >
            <Input placeholder="HEAD" />
          </Form.Item>
          <Form.Item
            name="destinationNamespace"
            label={t('argocd.namespace', 'Destination Namespace')}
            rules={[{ required: true }]}
          >
            <Input placeholder="default" />
          </Form.Item>
          <Form.Item
            name="autoSync"
            label={t('argocd.autoSync', 'Auto Sync')}
            valuePropName="checked"
          >
            <Select>
              <Option value={true}>{t('common.enabled', 'Enabled')}</Option>
              <Option value={false}>{t('common.disabled', 'Disabled')}</Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('common.create', 'Create')}
              </Button>
              <Button onClick={() => setAddAppModalVisible(false)}>
                {t('common.cancel', 'Cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 添加仓库模态框 */}
      <Modal
        title={t('argocd.addRepo', 'Add Repository')}
        open={addRepoModalVisible}
        onCancel={() => setAddRepoModalVisible(false)}
        footer={null}
      >
        <Form form={repoForm} layout="vertical" onFinish={handleAddRepo}>
          <Form.Item
            name="url"
            label={t('argocd.repoURL', 'Repository URL')}
            rules={[{ required: true }]}
          >
            <Input placeholder="https://github.com/org/repo.git" />
          </Form.Item>
          <Form.Item
            name="type"
            label={t('argocd.type', 'Type')}
          >
            <Select defaultValue="git">
              <Option value="git">Git</Option>
              <Option value="helm">Helm</Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="username"
            label={t('argocd.username', 'Username')}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="password"
            label={t('argocd.password', 'Password / Token')}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('common.add', 'Add')}
              </Button>
              <Button onClick={() => setAddRepoModalVisible(false)}>
                {t('common.cancel', 'Cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 应用详情抽屉 */}
      <Drawer
        title={selectedApp?.metadata?.name}
        placement="right"
        width={600}
        open={detailDrawerVisible}
        onClose={() => setDetailDrawerVisible(false)}
      >
        {selectedApp && (
          <div>
            <Card title={t('argocd.appInfo', 'Application Info')} style={{ marginBottom: 16 }}>
              <p><strong>{t('argocd.project', 'Project')}:</strong> {selectedApp.spec?.project}</p>
              <p><strong>{t('argocd.repoURL', 'Repository')}:</strong> {selectedApp.spec?.source?.repoURL}</p>
              <p><strong>{t('argocd.path', 'Path')}:</strong> {selectedApp.spec?.source?.path}</p>
              <p><strong>{t('argocd.revision', 'Revision')}:</strong> {selectedApp.spec?.source?.targetRevision}</p>
              <p><strong>{t('argocd.namespace', 'Namespace')}:</strong> {selectedApp.spec?.destination?.namespace}</p>
            </Card>
            <Card title={t('argocd.status', 'Status')}>
              <p>
                <strong>{t('argocd.health', 'Health')}:</strong>{' '}
                <Badge 
                  status={healthStatusColors[selectedApp.status?.health?.status]} 
                  text={selectedApp.status?.health?.status} 
                />
              </p>
              <p>
                <strong>{t('argocd.sync', 'Sync')}:</strong>{' '}
                <Tag color={syncStatusColors[selectedApp.status?.sync?.status]}>
                  {selectedApp.status?.sync?.status}
                </Tag>
              </p>
              {selectedApp.status?.health?.message && (
                <Alert 
                  type={selectedApp.status?.health?.status === 'Healthy' ? 'success' : 'warning'}
                  message={selectedApp.status?.health?.message}
                  style={{ marginTop: 16 }}
                />
              )}
            </Card>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default ArgoCDManagement;
