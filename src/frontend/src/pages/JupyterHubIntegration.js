import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Button,
  Tabs,
  Typography,
  Space,
  Alert,
  Spin,
  Badge,
  Statistic,
  Table,
  Tag,
  Modal,
  Form,
  Input,
  Select,
  message,
  Tooltip,
  Switch,
  Progress
} from 'antd';
import {
  ExperimentOutlined,
  RocketOutlined,
  MonitorOutlined,
  SettingOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  ReloadOutlined,
  EyeOutlined,
  LinkOutlined,
  CodeOutlined,
  DatabaseOutlined,
  CloudServerOutlined,
  ThunderboltOutlined
} from '@ant-design/icons';

const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;
const { Option } = Select;

const JupyterHubIntegration = () => {
  const [loading, setLoading] = useState(false);
  const [jupyterHubStatus, setJupyterHubStatus] = useState('unknown');
  const [gpuStatus, setGpuStatus] = useState({});
  const [activeJobs, setActiveJobs] = useState([]);
  const [systemStats, setSystemStats] = useState({});
  const [showJobModal, setShowJobModal] = useState(false);
  const [selectedNotebook, setSelectedNotebook] = useState(null);

  // JupyterHub 配置
  const jupyterHubConfig = {
    url: process.env.REACT_APP_JUPYTERHUB_URL || 'http://localhost:8088',
    apiUrl: process.env.REACT_APP_API_URL || 'http://localhost:8080',
    namespace: 'jupyterhub-jobs'
  };

  useEffect(() => {
    checkJupyterHubStatus();
    fetchGPUStatus();
    fetchActiveJobs();
    fetchSystemStats();
  }, []);

  const checkJupyterHubStatus = async () => {
    setLoading(true);
    try {
      const response = await fetch(`${jupyterHubConfig.url}/hub/api/info`);
      if (response.ok) {
        setJupyterHubStatus('running');
      } else {
        setJupyterHubStatus('error');
      }
    } catch (error) {
      setJupyterHubStatus('stopped');
    }
    setLoading(false);
  };

  const fetchGPUStatus = async () => {
    try {
      const response = await fetch(`${jupyterHubConfig.apiUrl}/api/k8s/gpu-status`);
      if (response.ok) {
        const data = await response.json();
        setGpuStatus(data);
      }
    } catch (error) {
      console.error('获取GPU状态失败:', error);
    }
  };

  const fetchActiveJobs = async () => {
    try {
      const response = await fetch(`${jupyterHubConfig.apiUrl}/api/k8s/jobs`);
      if (response.ok) {
        const data = await response.json();
        setActiveJobs(data.jobs || []);
      }
    } catch (error) {
      console.error('获取活动作业失败:', error);
    }
  };

  const fetchSystemStats = async () => {
    try {
      const response = await fetch(`${jupyterHubConfig.apiUrl}/api/system/stats`);
      if (response.ok) {
        const data = await response.json();
        setSystemStats(data);
      }
    } catch (error) {
      console.error('获取系统统计失败:', error);
    }
  };

  const handleJupyterHubLaunch = async () => {
    setLoading(true);
    try {
      // 获取当前用户的认证令牌
      const token = localStorage.getItem('token');
      if (!token) {
        message.error('请先登录');
        return;
      }

      // 通过后端API生成JupyterHub认证令牌
      const response = await fetch(`${jupyterHubConfig.apiUrl}/api/auth/jupyterhub-token`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        if (data.success) {
          // 使用生成的token直接跳转到JupyterHub
          const jupyterUrl = `${jupyterHubConfig.url}/hub/login?next=%2Fhub%2F&token=${data.token}`;
          window.open(jupyterUrl, '_blank');
          message.success('正在跳转到JupyterHub...');
        } else {
          message.error('生成JupyterHub令牌失败');
        }
      } else {
        // 如果令牌生成失败，尝试直接跳转
        message.warning('使用传统方式跳转到JupyterHub');
        window.open(jupyterHubConfig.url, '_blank');
      }
    } catch (error) {
      console.error('JupyterHub跳转失败:', error);
      message.warning('使用传统方式跳转到JupyterHub');
      window.open(jupyterHubConfig.url, '_blank');
    } finally {
      setLoading(false);
    }
  };

  const handleNotebookLaunch = async (notebookPath) => {
    setLoading(true);
    try {
      const token = localStorage.getItem('token');
      if (!token) {
        message.error('请先登录');
        return;
      }

      // 通过后端API生成JupyterHub认证令牌
      const response = await fetch(`${jupyterHubConfig.apiUrl}/api/auth/jupyterhub-token`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        if (data.success) {
          // 直接跳转到特定的notebook，包含认证信息
          const notebookUrl = `${jupyterHubConfig.url}/hub/login?next=%2Fhub%2Fuser-redirect%2Flab%2Ftree%2F${encodeURIComponent(notebookPath)}&token=${data.token}`;
          window.open(notebookUrl, '_blank');
          message.success('正在跳转到Notebook...');
        } else {
          message.error('生成认证令牌失败');
        }
      } else {
        // 如果令牌生成失败，使用传统方式
        const notebookUrl = `${jupyterHubConfig.url}/hub/user-redirect/lab/tree/${notebookPath}`;
        window.open(notebookUrl, '_blank');
        message.warning('使用传统方式跳转到Notebook');
      }
    } catch (error) {
      console.error('Notebook跳转失败:', error);
      const notebookUrl = `${jupyterHubConfig.url}/hub/user-redirect/lab/tree/${notebookPath}`;
      window.open(notebookUrl, '_blank');
      message.warning('使用传统方式跳转到Notebook');
    } finally {
      setLoading(false);
    }
  };

  const handleJobSubmit = async (values) => {
    try {
      const response = await fetch(`${jupyterHubConfig.apiUrl}/api/k8s/submit-job`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(values),
      });

      if (response.ok) {
        message.success('作业提交成功');
        setShowJobModal(false);
        fetchActiveJobs();
      } else {
        message.error('作业提交失败');
      }
    } catch (error) {
      message.error('作业提交错误');
    }
  };

  const getStatusColor = (status) => {
    switch (status) {
      case 'running': return 'success';
      case 'stopped': return 'error';
      case 'error': return 'warning';
      default: return 'default';
    }
  };

  const getStatusText = (status) => {
    switch (status) {
      case 'running': return '运行中';
      case 'stopped': return '已停止';
      case 'error': return '错误';
      default: return '未知';
    }
  };

  const predefinedNotebooks = [
    {
      name: 'K8s GPU 集成初始化',
      path: 'notebooks/02-k8s-gpu-integration-init.ipynb',
      description: '初始化 Kubernetes GPU 集成环境，包含 GPU 监控和作业提交功能',
      icon: <ThunderboltOutlined />,
      category: 'gpu'
    },
    {
      name: 'JupyterHub K8s 集成',
      path: 'notebooks/01-jupyterhub-k8s-gpu-integration.ipynb',
      description: '完整的 JupyterHub 与 Kubernetes 集成示例',
      icon: <CloudServerOutlined />,
      category: 'integration'
    },
    {
      name: 'API 共享示例',
      path: 'notebooks/03-share-api.ipynb',
      description: '演示如何使用共享 API 进行协作',
      icon: <DatabaseOutlined />,
      category: 'api'
    },
    {
      name: 'JupyterLab 共享',
      path: 'notebooks/04-share-jupyterlab.ipynb',
      description: 'JupyterLab 环境共享和协作功能',
      icon: <CodeOutlined />,
      category: 'collaboration'
    }
  ];

  const jobColumns = [
    {
      title: '作业名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={status === 'Succeeded' ? 'green' : status === 'Failed' ? 'red' : 'blue'}>
          {status}
        </Tag>
      ),
    },
    {
      title: 'GPU',
      dataIndex: 'gpu_required',
      key: 'gpu_required',
      render: (gpu) => gpu ? <Tag color="gold">GPU</Tag> : <Tag>CPU</Tag>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button size="small" icon={<EyeOutlined />}>
            查看日志
          </Button>
        </Space>
      ),
    },
  ];

  const JobSubmitModal = () => {
    const [form] = Form.useForm();

    return (
      <Modal
        title="提交 K8s GPU 作业"
        open={showJobModal}
        onCancel={() => setShowJobModal(false)}
        onOk={() => form.submit()}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleJobSubmit}
        >
          <Form.Item
            name="name"
            label="作业名称"
            rules={[{ required: true, message: '请输入作业名称' }]}
          >
            <Input placeholder="输入作业名称" />
          </Form.Item>

          <Form.Item
            name="script_content"
            label="Python 脚本内容"
            rules={[{ required: true, message: '请输入脚本内容' }]}
          >
            <Input.TextArea
              rows={8}
              placeholder="输入要执行的 Python 脚本内容..."
            />
          </Form.Item>

          <Form.Item
            name="gpu_required"
            label="需要 GPU"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch />
          </Form.Item>

          <Form.Item
            name="gpu_type"
            label="GPU 类型"
            initialValue="any"
          >
            <Select>
              <Option value="any">任意</Option>
              <Option value="tesla-v100">Tesla V100</Option>
              <Option value="tesla-p100">Tesla P100</Option>
              <Option value="tesla-t4">Tesla T4</Option>
            </Select>
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="cpu_limit"
                label="CPU 限制"
                initialValue="4"
              >
                <Input addonAfter="cores" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="memory_limit"
                label="内存限制"
                initialValue="8Gi"
              >
                <Input addonAfter="GB" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    );
  };

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={[16, 16]} align="middle" style={{ marginBottom: 24 }}>
        <Col>
          <Title level={2} style={{ margin: 0 }}>
            <ExperimentOutlined /> JupyterHub K8s GPU 集成平台
          </Title>
        </Col>
        <Col flex="auto">
          <Text type="secondary">
            连接 JupyterHub 和 Kubernetes GPU 集群的统一管理平台
          </Text>
        </Col>
        <Col>
          <Space>
            <Button
              type="primary"
              size="large"
              icon={<LinkOutlined />}
              onClick={handleJupyterHubLaunch}
              loading={loading}
            >
              启动 JupyterHub
            </Button>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => {
                checkJupyterHubStatus();
                fetchGPUStatus();
                fetchActiveJobs();
              }}
            />
          </Space>
        </Col>
      </Row>

      {/* 系统状态卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="JupyterHub 状态"
              value={getStatusText(jupyterHubStatus)}
              prefix={
                <Badge
                  status={getStatusColor(jupyterHubStatus)}
                  style={{ marginRight: 8 }}
                />
              }
              suffix={
                <Tooltip title="点击刷新状态">
                  <Button
                    type="text"
                    size="small"
                    icon={<ReloadOutlined />}
                    onClick={checkJupyterHubStatus}
                  />
                </Tooltip>
              }
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="可用 GPU"
              value={gpuStatus.available_gpus || 0}
              suffix={`/ ${gpuStatus.total_gpus || 0}`}
              prefix={<ThunderboltOutlined style={{ color: '#faad14' }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="活动作业"
              value={activeJobs.length}
              prefix={<RocketOutlined style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="GPU 使用率"
              value={Math.round((gpuStatus.used_gpus || 0) / (gpuStatus.total_gpus || 1) * 100)}
              suffix="%"
              prefix={<MonitorOutlined style={{ color: '#52c41a' }} />}
            />
            <Progress
              percent={Math.round((gpuStatus.used_gpus || 0) / (gpuStatus.total_gpus || 1) * 100)}
              size="small"
              showInfo={false}
              style={{ marginTop: 8 }}
            />
          </Card>
        </Col>
      </Row>

      <Tabs defaultActiveKey="notebooks" type="card">
        {/* Notebook 启动面板 */}
        <TabPane tab={<span><CodeOutlined />预配置 Notebook</span>} key="notebooks">
          <Row gutter={[16, 16]}>
            {predefinedNotebooks.map((notebook, index) => (
              <Col span={12} key={index}>
                <Card
                  hoverable
                  actions={[
                    <Button
                      type="primary"
                      icon={<PlayCircleOutlined />}
                      onClick={() => handleNotebookLaunch(notebook.path)}
                    >
                      启动 Notebook
                    </Button>
                  ]}
                >
                  <Card.Meta
                    avatar={
                      <div style={{ 
                        fontSize: 24, 
                        color: '#1890ff',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        width: 48,
                        height: 48,
                        backgroundColor: '#f0f8ff',
                        borderRadius: 8
                      }}>
                        {notebook.icon}
                      </div>
                    }
                    title={notebook.name}
                    description={
                      <div>
                        <Paragraph ellipsis={{ rows: 2 }}>
                          {notebook.description}
                        </Paragraph>
                        <Tag color="blue">{notebook.category}</Tag>
                      </div>
                    }
                  />
                </Card>
              </Col>
            ))}
          </Row>

          <Card 
            style={{ marginTop: 16 }}
            title="快速启动选项"
            extra={
              <Button
                type="primary"
                icon={<RocketOutlined />}
                onClick={() => setShowJobModal(true)}
              >
                提交自定义作业
              </Button>
            }
          >
            <Row gutter={16}>
              <Col span={8}>
                <Button
                  block
                  size="large"
                  icon={<LinkOutlined />}
                  onClick={() => window.open(`${jupyterHubConfig.url}/hub/spawn`, '_blank')}
                >
                  新建 JupyterLab 实例
                </Button>
              </Col>
              <Col span={8}>
                <Button
                  block
                  size="large"
                  icon={<MonitorOutlined />}
                  onClick={() => window.open(`${jupyterHubConfig.url}/hub/admin`, '_blank')}
                >
                  管理面板
                </Button>
              </Col>
              <Col span={8}>
                <Button
                  block
                  size="large"
                  icon={<DatabaseOutlined />}
                  onClick={() => window.open(`${jupyterHubConfig.apiUrl}/docs`, '_blank')}
                >
                  API 文档
                </Button>
              </Col>
            </Row>
          </Card>
        </TabPane>

        {/* GPU 资源监控 */}
        <TabPane tab={<span><ThunderboltOutlined />GPU 资源</span>} key="gpu">
          <Row gutter={[16, 16]}>
            <Col span={24}>
              <Card title="GPU 节点状态" extra={<Button icon={<ReloadOutlined />} onClick={fetchGPUStatus} />}>
                {gpuStatus.nodes && gpuStatus.nodes.length > 0 ? (
                  <Row gutter={[16, 16]}>
                    {gpuStatus.nodes.map((node, index) => (
                      <Col span={8} key={index}>
                        <Card size="small">
                          <Statistic
                            title={node.name}
                            value={node.available_gpus}
                            suffix={`/ ${node.total_gpus} 可用`}
                            prefix={
                              <Badge
                                status={node.status === 'Ready' ? 'success' : 'error'}
                              />
                            }
                          />
                          <div style={{ marginTop: 8 }}>
                            <Text type="secondary">类型: {node.gpu_type}</Text>
                          </div>
                        </Card>
                      </Col>
                    ))}
                  </Row>
                ) : (
                  <Alert
                    message="未检测到 GPU 节点"
                    description="请确保集群中有正确标记的 GPU 节点"
                    type="warning"
                    showIcon
                  />
                )}
              </Card>
            </Col>
          </Row>
        </TabPane>

        {/* 作业管理 */}
        <TabPane tab={<span><RocketOutlined />作业管理</span>} key="jobs">
          <Card
            title="K8s GPU 作业"
            extra={
              <Space>
                <Button
                  type="primary"
                  icon={<RocketOutlined />}
                  onClick={() => setShowJobModal(true)}
                >
                  提交新作业
                </Button>
                <Button icon={<ReloadOutlined />} onClick={fetchActiveJobs} />
              </Space>
            }
          >
            <Table
              columns={jobColumns}
              dataSource={activeJobs}
              rowKey="name"
              pagination={{ pageSize: 10 }}
            />
          </Card>
        </TabPane>

        {/* 系统配置 */}
        <TabPane tab={<span><SettingOutlined />系统配置</span>} key="config">
          <Row gutter={[16, 16]}>
            <Col span={12}>
              <Card title="JupyterHub 配置">
                <Space direction="vertical" style={{ width: '100%' }}>
                  <div>
                    <Text strong>服务地址:</Text>
                    <Text copyable={{ text: jupyterHubConfig.url }}>
                      {jupyterHubConfig.url}
                    </Text>
                  </div>
                  <div>
                    <Text strong>API 地址:</Text>
                    <Text copyable={{ text: jupyterHubConfig.apiUrl }}>
                      {jupyterHubConfig.apiUrl}
                    </Text>
                  </div>
                  <div>
                    <Text strong>K8s 命名空间:</Text>
                    <Text code>{jupyterHubConfig.namespace}</Text>
                  </div>
                </Space>
              </Card>
            </Col>
            <Col span={12}>
              <Card title="快速链接">
                <Space direction="vertical" style={{ width: '100%' }}>
                  <Button
                    block
                    icon={<LinkOutlined />}
                    onClick={() => window.open(`${jupyterHubConfig.url}/hub/token`, '_blank')}
                  >
                    API Token 管理
                  </Button>
                  <Button
                    block
                    icon={<MonitorOutlined />}
                    onClick={() => window.open(`${jupyterHubConfig.url}/hub/admin`, '_blank')}
                  >
                    用户管理
                  </Button>
                  <Button
                    block
                    icon={<SettingOutlined />}
                    onClick={() => window.open(`${jupyterHubConfig.apiUrl}/docs`, '_blank')}
                  >
                    API 文档
                  </Button>
                </Space>
              </Card>
            </Col>
          </Row>
        </TabPane>
      </Tabs>

      <JobSubmitModal />
    </div>
  );
};

export default JupyterHubIntegration;
