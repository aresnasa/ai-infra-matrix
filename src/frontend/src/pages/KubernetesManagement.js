import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  message,
  Popconfirm,
  Tag,
  Space,
  Card,
  Row,
  Col,
  Typography,
  Divider,
  Tooltip,
  Badge,
  Upload,
  Tabs,
  Alert,
  Spin,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ReloadOutlined,
  ExperimentOutlined,
  CloudServerOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  UploadOutlined,
  CopyOutlined,
  RocketOutlined,
  DownloadOutlined,
  RollbackOutlined,
} from '@ant-design/icons';
import { kubernetesAPI, ansibleAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;
const { TabPane } = Tabs;

const KubernetesManagement = () => {
  const { t } = useI18n();
  const [clusters, setClusters] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [testModalVisible, setTestModalVisible] = useState(false);
  const [editingCluster, setEditingCluster] = useState(null);
  const [form] = Form.useForm();
  const [testForm] = Form.useForm();
  const [testLoading, setTestLoading] = useState(false);
  const [testResult, setTestResult] = useState(null);
  const [selectedCluster, setSelectedCluster] = useState(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [namespace, setNamespace] = useState('default');
  const [namespaces, setNamespaces] = useState([]);
  const [pods, setPods] = useState([]);
  const [deployments, setDeployments] = useState([]);
  const [services, setServices] = useState([]);
  const [nodes, setNodes] = useState([]);
  const [events, setEvents] = useState([]);
  const [resourceLoading, setResourceLoading] = useState(false);
  const [execModalOpen, setExecModalOpen] = useState(false);
  const [execTarget, setExecTarget] = useState(null);
  
  // Helm 相关状态
  const [helmReleases, setHelmReleases] = useState([]);
  const [helmLoading, setHelmLoading] = useState(false);
  const [helmInstallModalVisible, setHelmInstallModalVisible] = useState(false);
  const [helmImportModalVisible, setHelmImportModalVisible] = useState(false);
  const [helmForm] = Form.useForm();
  const [helmImportContent, setHelmImportContent] = useState('');

  // 获取集群列表
  const fetchClusters = async () => {
    setLoading(true);
    try {
      const response = await kubernetesAPI.getClusters();
      // 修复：后端直接返回数组，不是嵌套在data.data中
      const clusterData = response.data || [];
      
      // 清理数据，确保所有字段都是有效的
      const cleanedClusters = clusterData.map(cluster => ({
        ...cluster,
        // 确保字符串字段不为null
        name: cluster.name || '',
        description: cluster.description || '',
        api_server: cluster.api_server || '',
        kube_config: cluster.kube_config || '',
        namespace: cluster.namespace || 'default',
        status: cluster.status || 'unknown',
        version: cluster.version || '',
        cluster_type: cluster.cluster_type || 'kubernetes',
      }));
      
      setClusters(cleanedClusters);
      console.log('Fetched clusters:', cleanedClusters.length); // 调试日志
    } catch (error) {
      console.error('Failed to fetch clusters:', error); // 调试日志
      message.error(t('kubernetes.fetchFailed') + ': ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    console.log('KubernetesManagement component mounted, fetching clusters...');
    fetchClusters();
  }, []);

  // 进入集群详情
  const enterCluster = async (cluster) => {
    setSelectedCluster(cluster);
    setDetailOpen(true);
    setNamespace(cluster.namespace || 'default');
    await Promise.all([
      loadNamespaces(cluster.id), 
      loadResources(cluster.id, cluster.namespace || 'default'),
      loadHelmReleases(cluster.id, cluster.namespace || 'default'),
    ]);
  };

  const loadNamespaces = async (clusterId) => {
    try {
      const res = await kubernetesAPI.getClusterNamespaces(clusterId);
      setNamespaces(res.data?.items || res.data || []);
    } catch (e) {
      setNamespaces(['default']);
    }
  };

  const loadResources = async (clusterId, ns) => {
    setResourceLoading(true);
    try {
      const [podsRes, depRes, svcRes, nodesRes, eventsRes] = await Promise.allSettled([
        kubernetesAPI.getPods(clusterId, ns),
        kubernetesAPI.getDeployments(clusterId, ns),
        kubernetesAPI.getServices(clusterId, ns),
        kubernetesAPI.getNodesDetail(clusterId),
        kubernetesAPI.getEvents(clusterId, ns)
      ]);

      setPods(podsRes.status === 'fulfilled' ? (podsRes.value.data?.items || podsRes.value.data || []) : []);
      setDeployments(depRes.status === 'fulfilled' ? (depRes.value.data?.items || depRes.value.data || []) : []);
      setServices(svcRes.status === 'fulfilled' ? (svcRes.value.data?.items || svcRes.value.data || []) : []);
      setNodes(nodesRes.status === 'fulfilled' ? (nodesRes.value.data?.items || nodesRes.value.data || []) : []);
      setEvents(eventsRes.status === 'fulfilled' ? (eventsRes.value.data?.items || eventsRes.value.data || []) : []);
    } finally {
      setResourceLoading(false);
    }
  };

  const onNamespaceChange = async (ns) => {
    setNamespace(ns);
    if (selectedCluster) {
      await Promise.all([
        loadResources(selectedCluster.id, ns),
        loadHelmReleases(selectedCluster.id, ns),
      ]);
    }
  };

  // 操作：伸缩Deployment
  const handleScaleDeployment = async (d) => {
    try {
      const replicas = Number(prompt(t('kubernetes.setReplicas'), d?.spec?.replicas ?? 1));
      if (Number.isNaN(replicas)) return;
      await kubernetesAPI.scaleDeployment(selectedCluster.id, namespace, d.metadata.name, replicas);
      message.success(t('kubernetes.scaleSubmitted'));
      await loadResources(selectedCluster.id, namespace);
    } catch (e) {
      message.error(t('kubernetes.scaleFailed') + ': ' + (e.response?.data?.message || e.message));
    }
  };

  // 操作：删除资源（示意）
  const handleDeleteResource = async (kind, name) => {
    try {
      await kubernetesAPI.deleteResource(selectedCluster.id, namespace, kind, name);
      message.success(t('common.deleteSuccess'));
      await loadResources(selectedCluster.id, namespace);
    } catch (e) {
      message.error(t('common.deleteFailed') + ': ' + (e.response?.data?.message || e.message));
    }
  };

  // 操作：进入容器（占位，真实需后端ws代理）
  const openExec = (pod) => {
    setExecTarget({ pod: pod.metadata.name, containers: pod.spec?.containers?.map(c => c.name) || [] });
    setExecModalOpen(true);
  };

  // ---- Helm 相关函数 ----
  // 加载 Helm releases
  const loadHelmReleases = async (clusterId, ns) => {
    setHelmLoading(true);
    try {
      const res = await kubernetesAPI.getHelmReleases(clusterId, ns);
      setHelmReleases(res.data?.releases || res.data || []);
    } catch (e) {
      console.error('Failed to load Helm releases:', e);
      setHelmReleases([]);
    } finally {
      setHelmLoading(false);
    }
  };

  // 安装 Helm chart
  const handleInstallHelmChart = async (values) => {
    try {
      await kubernetesAPI.installHelmChart(selectedCluster.id, {
        ...values,
        namespace: namespace,
      });
      message.success(t('kubernetes.installSuccess') || 'Install successful');
      setHelmInstallModalVisible(false);
      helmForm.resetFields();
      await loadHelmReleases(selectedCluster.id, namespace);
    } catch (e) {
      message.error(t('kubernetes.installFailed') + ': ' + (e.response?.data?.message || e.message));
    }
  };

  // 卸载 Helm release
  const handleUninstallHelmRelease = async (releaseName) => {
    try {
      await kubernetesAPI.uninstallHelmRelease(selectedCluster.id, namespace, releaseName);
      message.success(t('kubernetes.uninstallSuccess') || 'Uninstall successful');
      await loadHelmReleases(selectedCluster.id, namespace);
    } catch (e) {
      message.error(t('kubernetes.uninstallFailed') + ': ' + (e.response?.data?.message || e.message));
    }
  };

  // 回滚 Helm release
  const handleRollbackHelmRelease = async (releaseName) => {
    const revision = prompt(t('kubernetes.inputRevision') || 'Enter revision number', '1');
    if (!revision) return;
    try {
      await kubernetesAPI.rollbackHelmRelease(selectedCluster.id, namespace, releaseName, parseInt(revision));
      message.success(t('kubernetes.rollbackSuccess') || 'Rollback successful');
      await loadHelmReleases(selectedCluster.id, namespace);
    } catch (e) {
      message.error(t('kubernetes.rollbackFailed') + ': ' + (e.response?.data?.message || e.message));
    }
  };

  // 导入 Helm 配置
  const handleImportHelmConfig = async () => {
    try {
      let config;
      try {
        config = JSON.parse(helmImportContent);
      } catch {
        // 尝试解析为 YAML
        message.error(t('kubernetes.invalidConfig') || 'Invalid config format');
        return;
      }
      await kubernetesAPI.importHelmConfig(selectedCluster.id, config);
      message.success(t('kubernetes.helmConfigImported'));
      setHelmImportModalVisible(false);
      setHelmImportContent('');
      await loadHelmReleases(selectedCluster.id, namespace);
    } catch (e) {
      message.error(t('kubernetes.importFailed') + ': ' + (e.response?.data?.message || e.message));
    }
  };

  // 导出 Helm 配置
  const handleExportHelmConfig = async (releaseName) => {
    try {
      const res = await kubernetesAPI.exportHelmConfig(selectedCluster.id, namespace, releaseName);
      const blob = new Blob([JSON.stringify(res.data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${releaseName}-helm-config.json`;
      a.click();
      URL.revokeObjectURL(url);
      message.success(t('kubernetes.helmConfigExported'));
    } catch (e) {
      message.error(t('kubernetes.exportFailed') + ': ' + (e.response?.data?.message || e.message));
    }
  };

  // Helm release 状态颜色
  const getHelmStatusColor = (status) => {
    const colors = {
      deployed: 'green',
      failed: 'red',
      'pending-install': 'orange',
      'pending-upgrade': 'orange',
      'pending-rollback': 'orange',
      superseded: 'default',
      uninstalling: 'processing',
      uninstalled: 'default',
    };
    return colors[status?.toLowerCase()] || 'default';
  };

  // 添加/编辑集群
  const handleSubmit = async (values) => {
    console.log('Submitting cluster data:', values);
    try {
      // 修复字段映射：将config字段映射为kube_config
      const payload = {
        ...values,
        kube_config: values.config || ''
      };
      delete payload.config; // 删除原config字段
      
      console.log('Payload to be sent:', payload);
      
      if (editingCluster) {
        await kubernetesAPI.updateCluster(editingCluster.id, payload);
        message.success(t('kubernetes.updateSuccess'));
      } else {
        await kubernetesAPI.createCluster(payload);
        message.success(t('kubernetes.addSuccess'));
      }
      setModalVisible(false);
      setEditingCluster(null);
      form.resetFields();
      fetchClusters();
    } catch (error) {
      console.error('Submit error:', error);
      message.error((editingCluster ? t('kubernetes.updateFailed') : t('kubernetes.addFailed')) + ': ' + error.message);
    }
  };

  // 删除集群
  const handleDelete = async (id) => {
    try {
      await kubernetesAPI.deleteCluster(id);
      message.success(t('kubernetes.deleteSuccess'));
      fetchClusters();
    } catch (error) {
      message.error(t('common.deleteFailed') + ': ' + error.message);
    }
  };

  // 测试连接
  const handleTestConnection = async (cluster) => {
    setSelectedCluster(cluster);
    setTestModalVisible(true);
    setTestResult(null);
    testForm.setFieldsValue({
      clusterId: cluster.id,
      clusterName: cluster.name,
    });
  };

  // 执行连接测试
  const executeTest = async () => {
    setTestLoading(true);
    try {
      const response = await kubernetesAPI.testConnection(selectedCluster.id);
      setTestResult({
        success: true,
        data: response.data,
      });
      message.success(t('kubernetes.testSuccess'));
      // 测试成功后刷新集群列表以更新状态
      fetchClusters();
    } catch (error) {
      setTestResult({
        success: false,
        error: error.response?.data?.message || error.message,
      });
      message.error(t('kubernetes.testFailed'));
      // 测试失败后也刷新集群列表以更新状态
      fetchClusters();
    } finally {
      setTestLoading(false);
    }
  };

  // 编辑集群
  const handleEdit = (cluster) => {
    console.log('Editing cluster:', cluster);
    setEditingCluster(cluster);
    form.setFieldsValue({
      ...cluster,
      // 修复字段映射：将kube_config映射到config字段用于表单显示
      config: cluster.kube_config || '',
    });
    setModalVisible(true);
  };

  // 复制kubeconfig（兼容 HTTP 和 HTTPS 环境）
  const handleCopyConfig = async (config) => {
    // 首先尝试现代 Clipboard API
    if (navigator.clipboard && window.isSecureContext) {
      try {
        await navigator.clipboard.writeText(config);
        message.success(t('kubernetes.copiedToClipboard'));
        return;
      } catch (err) {
        console.warn('Clipboard API failed, falling back to execCommand:', err);
      }
    }
    
    // 备用方案：使用传统的 execCommand 方式
    const textArea = document.createElement('textarea');
    textArea.value = config;
    textArea.style.position = 'fixed';
    textArea.style.left = '-9999px';
    textArea.style.top = '-9999px';
    textArea.style.opacity = '0';
    
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    
    try {
      const successful = document.execCommand('copy');
      if (successful) {
        message.success(t('kubernetes.copiedToClipboard'));
      } else {
        message.error('复制失败，请手动复制');
      }
    } catch (err) {
      console.error('execCommand copy failed:', err);
      message.error('复制失败，请手动复制');
    } finally {
      document.body.removeChild(textArea);
    }
  };

  // 状态标签
  const getStatusTag = (status) => {
    const statusMap = {
      connected: { color: 'green', icon: <CheckCircleOutlined />, text: t('kubernetes.connected') },
      disconnected: { color: 'red', icon: <CloseCircleOutlined />, text: t('kubernetes.disconnected') },
      active: { color: 'green', icon: <CheckCircleOutlined />, text: t('kubernetes.active') },
      inactive: { color: 'red', icon: <CloseCircleOutlined />, text: t('kubernetes.inactive') },
      unknown: { color: 'orange', icon: <InfoCircleOutlined />, text: t('kubernetes.notTested') },
    };
    const statusInfo = statusMap[status] || statusMap.unknown;
    return (
      <Tag color={statusInfo.color} icon={statusInfo.icon}>
        {statusInfo.text}
      </Tag>
    );
  };

  // 表格列定义
  const columns = [
    {
      title: t('kubernetes.clusterName'),
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Space>
          <CloudServerOutlined />
          <strong>{text}</strong>
        </Space>
      ),
    },
    {
      title: t('kubernetes.clusterType'),
      dataIndex: 'cluster_type',
      key: 'cluster_type',
      render: (type) => (
        <Tag color="blue">{type || 'kubernetes'}</Tag>
      ),
    },
    {
      title: t('kubernetes.apiServer'),
      dataIndex: 'api_server',
      key: 'api_server',
      ellipsis: true,
      render: (text) => (
        <Tooltip title={text}>
          <Text code>{text}</Text>
        </Tooltip>
      ),
    },
    {
      title: t('kubernetes.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status) => getStatusTag(status),
    },
    {
      title: t('kubernetes.version'),
      dataIndex: 'version',
      key: 'version',
      render: (version) => version && <Tag>{version}</Tag>,
    },
    {
      title: t('kubernetes.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('kubernetes.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 200,
      render: (_, record) => (
        <Space>
          <Tooltip title={t('kubernetes.testConnection')}>
            <Button
              type="primary"
              size="small"
              icon={<ExperimentOutlined />}
              onClick={() => handleTestConnection(record)}
            />
          </Tooltip>
          <Tooltip title={t('kubernetes.enterCluster')}>
            <Button size="small" onClick={() => enterCluster(record)}>{t('kubernetes.enter')}</Button>
          </Tooltip>
          <Tooltip title={t('common.edit')}>
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Popconfirm
            title={t('kubernetes.confirmDelete')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
          >
            <Tooltip title={t('common.delete')}>
              <Button
                size="small"
                danger
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
          <Col>
            <Title level={2} style={{ margin: 0 }}>
              <CloudServerOutlined /> {t('kubernetes.title')}
            </Title>
          </Col>
          <Col>
            <Space>
              <Button
                icon={<ReloadOutlined />}
                onClick={fetchClusters}
                loading={loading}
              >
                {t('common.refresh')}
              </Button>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  setEditingCluster(null);
                  form.resetFields();
                  setModalVisible(true);
                }}
              >
                {t('kubernetes.addCluster')}
              </Button>
            </Space>
          </Col>
        </Row>

  <Table
          columns={columns}
          dataSource={clusters}
          rowKey="id"
          loading={loading}
          pagination={{
            total: clusters.length,
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => t('kubernetes.totalClusters', { total }),
          }}
          locale={{
            emptyText: loading ? t('common.loading') : clusters.length === 0 ? t('kubernetes.noClusters') : t('common.noData'),
          }}
        />
        
        {/* 调试信息 */}
        {process.env.NODE_ENV === 'development' && (
          <div style={{ marginTop: 16, padding: 8, backgroundColor: '#f5f5f5', fontSize: 12 }}>
            <strong>调试信息:</strong> 当前集群数量: {clusters.length}, 
            加载状态: {loading ? '加载中' : '完成'}
            {clusters.length > 0 && (
              <div>集群列表: {clusters.map(c => c.name).join(', ')}</div>
            )}
          </div>
        )}
      </Card>

      {/* 添加/编辑集群模态框 */}
      <Modal
        title={editingCluster ? t('kubernetes.editCluster') : t('kubernetes.addCluster')}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingCluster(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        width={800}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label={t('kubernetes.clusterName')}
                rules={[{ required: true, message: t('kubernetes.pleaseInputClusterName') }]}
              >
                <Input placeholder={t('kubernetes.inputClusterName')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="cluster_type"
                label={t('kubernetes.clusterType')}
                initialValue="kubernetes"
              >
                <Select>
                  <Option value="kubernetes">Kubernetes</Option>
                  <Option value="openshift">OpenShift</Option>
                  <Option value="k3s">K3s</Option>
                  <Option value="microk8s">MicroK8s</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="api_server"
            label={t('kubernetes.apiServerUrl')}
            rules={[
              { required: true, message: t('kubernetes.pleaseInputApiServer') },
              { type: 'url', message: t('kubernetes.invalidUrl') },
            ]}
          >
            <Input placeholder="https://kubernetes.example.com:6443" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="username"
                label={t('kubernetes.username')}
              >
                <Input placeholder={t('kubernetes.usernameOptional')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="password"
                label={t('kubernetes.password')}
              >
                <Input.Password placeholder={t('kubernetes.passwordOptional')} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="token"
            label={t('kubernetes.accessToken')}
          >
            <TextArea
              rows={3}
              placeholder={t('kubernetes.tokenOptional')}
            />
          </Form.Item>

          <Form.Item
            name="config"
            label={t('kubernetes.kubeconfig')}
            tooltip={t('kubernetes.kubeconfigTooltip')}
          >
            <TextArea
              rows={8}
              placeholder={t('kubernetes.pasteKubeconfig')}
            />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('kubernetes.description')}
          >
            <TextArea
              rows={2}
              placeholder={t('kubernetes.descriptionOptional')}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 集群详情抽屉 */}
      <Modal
        title={`${t('kubernetes.clusterDetail')} - ${selectedCluster?.name || ''}`}
        open={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
        width={1000}
      >
        {selectedCluster && (
          <>
            <Space style={{ marginBottom: 12 }}>
              <Text strong>{t('kubernetes.namespace')}:</Text>
              <Select size="small" value={namespace} style={{ minWidth: 180 }} onChange={onNamespaceChange}>
                {(Array.isArray(namespaces) ? namespaces : []).map(ns => (
                  <Select.Option key={typeof ns === 'string' ? ns : ns.metadata?.name || ns.name} value={typeof ns === 'string' ? ns : ns.metadata?.name || ns.name}>
                    {typeof ns === 'string' ? ns : ns.metadata?.name || ns.name}
                  </Select.Option>
                ))}
              </Select>
              <Button size="small" icon={<ReloadOutlined />} onClick={() => loadResources(selectedCluster.id, namespace)} loading={resourceLoading}>{t('common.refresh')}</Button>
            </Space>

            <Tabs defaultActiveKey="overview">
              <Tabs.TabPane tab={t('kubernetes.overview')} key="overview">
                <Row gutter={12}>
                  <Col span={12}>
                    <Card size="small" title={t('kubernetes.nodes')}>
                      <Table
                        rowKey={(r) => r.metadata?.name || r.name}
                        size="small"
                        dataSource={nodes}
                        pagination={{ pageSize: 5 }}
                        columns={[
                          { title: t('kubernetes.nodeName'), dataIndex: ['metadata','name'], key: 'name', render: (_,r)=>(r.metadata?.name||r.name) },
                          { title: t('kubernetes.role'), key: 'roles', render:(_,r)=> (r.metadata?.labels?.['kubernetes.io/role'] || r.metadata?.labels?.['node-role.kubernetes.io/control-plane'] ? 'control-plane' : 'worker') },
                          { title: t('kubernetes.version'), dataIndex: ['status','nodeInfo','kubeletVersion'], key: 'ver', render:(_,r)=> r.status?.nodeInfo?.kubeletVersion || '-' },
                        ]}
                      />
                    </Card>
                  </Col>
                  <Col span={12}>
                    <Card size="small" title={t('kubernetes.events')}>
                      <Table
                        rowKey={(r,i)=> i}
                        size="small"
                        dataSource={events}
                        pagination={{ pageSize: 5 }}
                        columns={[
                          { title: t('kubernetes.eventType'), dataIndex: 'type', key: 'type' },
                          { title: t('kubernetes.reason'), dataIndex: 'reason', key: 'reason' },
                          { title: t('kubernetes.object'), key: 'obj', render:(_,r)=> r.involvedObject?.name },
                          { title: t('kubernetes.time'), key: 'time', render:(_,r)=> r.lastTimestamp || r.eventTime || '-' },
                        ]}
                      />
                    </Card>
                  </Col>
                </Row>
              </Tabs.TabPane>

              <Tabs.TabPane tab={t('kubernetes.workloads')} key="workloads">
                <Row gutter={12}>
                  <Col span={12}>
                    <Card size="small" title="Deployments">
                      <Table
                        rowKey={(r)=> r.metadata?.name}
                        size="small"
                        dataSource={deployments}
                        pagination={{ pageSize: 5 }}
                        columns={[
                          { title: t('kubernetes.deploymentName'), dataIndex: ['metadata','name'], key: 'name' },
                          { title: t('kubernetes.replicas'), key: 'replicas', render:(_,r)=> `${r.status?.readyReplicas || 0}/${r.spec?.replicas || 0}` },
                          { title: t('common.actions'), key: 'actions', render:(_,r)=> (
                            <Space>
                              <Button size="small" onClick={()=>handleScaleDeployment(r)}>{t('kubernetes.scale')}</Button>
                              <Popconfirm title={t('kubernetes.confirmDeleteResource')} onConfirm={()=>handleDeleteResource('deployment', r.metadata.name)}>
                                <Button size="small" danger>{t('common.delete')}</Button>
                              </Popconfirm>
                            </Space>
                          ) },
                        ]}
                      />
                    </Card>
                  </Col>
                  <Col span={12}>
                    <Card size="small" title="Services">
                      <Table
                        rowKey={(r)=> r.metadata?.name}
                        size="small"
                        dataSource={services}
                        pagination={{ pageSize: 5 }}
                        columns={[
                          { title: t('kubernetes.serviceName'), dataIndex: ['metadata','name'], key: 'name' },
                          { title: t('kubernetes.serviceType'), dataIndex: ['spec','type'], key: 'type' },
                          { title: 'ClusterIP', dataIndex: ['spec','clusterIP'], key: 'cip' },
                          { title: t('kubernetes.ports'), key: 'ports', render:(_,r)=> (r.spec?.ports||[]).map(p=>`${p.port}/${p.protocol}`).join(', ') },
                          { title: t('common.actions'), key: 'actions', render:(_,r)=> (
                            <Popconfirm title={t('kubernetes.confirmDeleteResource')} onConfirm={()=>handleDeleteResource('service', r.metadata.name)}>
                              <Button size="small" danger>{t('common.delete')}</Button>
                            </Popconfirm>
                          ) },
                        ]}
                      />
                    </Card>
                  </Col>
                </Row>
              </Tabs.TabPane>

              <Tabs.TabPane tab="Pods" key="pods">
                <Card size="small">
                  <Table
                    rowKey={(r)=> r.metadata?.name}
                    size="small"
                    dataSource={pods}
                    pagination={{ pageSize: 10 }}
                    columns={[
                      { title: t('kubernetes.podName'), dataIndex: ['metadata','name'], key: 'name' },
                      { title: t('kubernetes.status'), key: 'phase', render:(_,r)=> r.status?.phase },
                      { title: t('kubernetes.node'), dataIndex: ['spec','nodeName'], key: 'node' },
                      { title: t('kubernetes.restarts'), key: 'restarts', render:(_,r)=> (r.status?.containerStatuses||[]).reduce((a,c)=> a + (c.restartCount||0), 0) },
                      { title: t('kubernetes.images'), key: 'images', ellipsis: true, render:(_,r)=> (r.spec?.containers||[]).map(c=>c.image).join(', ') },
                      { title: t('common.actions'), key: 'actions', render:(_,r)=> (
                        <Space>
                          <Button size="small" onClick={()=>openExec(r)}>{t('kubernetes.enter')}</Button>
                          <Popconfirm title={t('kubernetes.confirmDeleteResource')} onConfirm={()=>handleDeleteResource('pod', r.metadata.name)}>
                            <Button size="small" danger>{t('common.delete')}</Button>
                          </Popconfirm>
                        </Space>
                      ) },
                    ]}
                  />
                </Card>
              </Tabs.TabPane>

              {/* Helm Tab */}
              <Tabs.TabPane tab={<span><RocketOutlined /> {t('kubernetes.helm')}</span>} key="helm">
                <Card 
                  size="small" 
                  title={t('kubernetes.helmReleases')}
                  extra={
                    <Space>
                      <Button 
                        size="small" 
                        icon={<UploadOutlined />}
                        onClick={() => setHelmImportModalVisible(true)}
                      >
                        {t('kubernetes.importHelmConfig')}
                      </Button>
                      <Button 
                        size="small" 
                        type="primary" 
                        icon={<PlusOutlined />}
                        onClick={() => setHelmInstallModalVisible(true)}
                      >
                        {t('kubernetes.installChart')}
                      </Button>
                      <Button 
                        size="small" 
                        icon={<ReloadOutlined />} 
                        onClick={() => loadHelmReleases(selectedCluster.id, namespace)}
                        loading={helmLoading}
                      >
                        {t('common.refresh')}
                      </Button>
                    </Space>
                  }
                >
                  <Spin spinning={helmLoading}>
                    <Table
                      rowKey={(r) => r.name || r.release_name}
                      size="small"
                      dataSource={helmReleases}
                      pagination={{ pageSize: 10 }}
                      locale={{
                        emptyText: t('kubernetes.noReleases'),
                      }}
                      columns={[
                        { 
                          title: t('kubernetes.releaseName'), 
                          dataIndex: 'name',
                          key: 'name',
                          render: (name, r) => name || r.release_name,
                        },
                        { 
                          title: t('kubernetes.chartName'), 
                          dataIndex: 'chart',
                          key: 'chart',
                          render: (chart, r) => chart || r.chart_name || '-',
                        },
                        { 
                          title: t('kubernetes.chartVersion'), 
                          dataIndex: 'chart_version',
                          key: 'version',
                          render: (v, r) => v || r.app_version || r.version || '-',
                        },
                        { 
                          title: t('kubernetes.releaseStatus'), 
                          dataIndex: 'status',
                          key: 'status',
                          render: (status) => (
                            <Tag color={getHelmStatusColor(status)}>
                              {status || 'unknown'}
                            </Tag>
                          ),
                        },
                        { 
                          title: t('kubernetes.releaseRevision'), 
                          dataIndex: 'revision',
                          key: 'revision',
                          render: (v) => v || '-',
                        },
                        { 
                          title: t('kubernetes.releaseNamespace'), 
                          dataIndex: 'namespace',
                          key: 'namespace',
                          render: (ns) => ns || namespace,
                        },
                        {
                          title: t('common.actions'),
                          key: 'actions',
                          width: 200,
                          render: (_, r) => (
                            <Space>
                              <Tooltip title={t('kubernetes.exportHelmConfig')}>
                                <Button 
                                  size="small" 
                                  icon={<DownloadOutlined />}
                                  onClick={() => handleExportHelmConfig(r.name || r.release_name)}
                                />
                              </Tooltip>
                              <Tooltip title={t('kubernetes.rollbackRelease')}>
                                <Button 
                                  size="small" 
                                  icon={<RollbackOutlined />}
                                  onClick={() => handleRollbackHelmRelease(r.name || r.release_name)}
                                />
                              </Tooltip>
                              <Popconfirm 
                                title={t('kubernetes.confirmDelete')}
                                onConfirm={() => handleUninstallHelmRelease(r.name || r.release_name)}
                              >
                                <Button size="small" danger icon={<DeleteOutlined />} />
                              </Popconfirm>
                            </Space>
                          ),
                        },
                      ]}
                    />
                  </Spin>
                </Card>
              </Tabs.TabPane>
            </Tabs>
          </>
        )}
      </Modal>

      {/* 进入容器（占位） */}
      <Modal
        title={t('kubernetes.containerTerminal')}
        open={execModalOpen}
        onCancel={()=> setExecModalOpen(false)}
        footer={[<Button key="close" onClick={()=> setExecModalOpen(false)}>{t('common.close')}</Button>]}
        width={800}
      >
        {execTarget ? (
          <div>
            <div style={{ marginBottom: 8 }}>
              <Text strong>Pod:</Text> <Text code>{execTarget.pod}</Text>
            </div>
            <div style={{ marginBottom: 8 }}>
              <Text strong>{t('kubernetes.containers')}:</Text> {(execTarget.containers||[]).map(c => <Tag key={c}>{c}</Tag>)}
            </div>
            <Alert
              type="info"
              message={t('kubernetes.terminalPlaceholder')}
            />
          </div>
        ) : null}
      </Modal>

      {/* 连接测试模态框 */}
      <Modal
        title={t('kubernetes.connectionTest')}
        open={testModalVisible}
        onCancel={() => {
          setTestModalVisible(false);
          setTestResult(null);
        }}
        footer={[
          <Button key="cancel" onClick={() => setTestModalVisible(false)}>
            {t('common.close')}
          </Button>,
          <Button
            key="test"
            type="primary"
            loading={testLoading}
            onClick={executeTest}
          >
            {t('kubernetes.startTest')}
          </Button>,
        ]}
        width={700}
      >
        {selectedCluster && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Text strong>{t('kubernetes.cluster')}: </Text>
              <Text>{selectedCluster.name}</Text>
              <br />
              <Text strong>{t('kubernetes.apiServer')}: </Text>
              <Text code>{selectedCluster.api_server}</Text>
            </div>

            <Divider />

            {testResult && (
              <div>
                {testResult.success ? (
                  <div>
                    <Badge status="success" text={t('kubernetes.connectionSuccess')} />
                    <div style={{ marginTop: 16, padding: 16, backgroundColor: '#f6ffed', border: '1px solid #b7eb8f', borderRadius: 4 }}>
                      <Text strong>{t('kubernetes.clusterInfo')}:</Text>
                      <pre style={{ marginTop: 8, fontSize: 12 }}>
                        {JSON.stringify(testResult.data, null, 2)}
                      </pre>
                    </div>
                  </div>
                ) : (
                  <div>
                    <Badge status="error" text={t('kubernetes.connectionFailed')} />
                    <div style={{ marginTop: 16, padding: 16, backgroundColor: '#fff2f0', border: '1px solid #ffccc7', borderRadius: 4 }}>
                      <Text strong>{t('kubernetes.errorInfo')}:</Text>
                      <pre style={{ marginTop: 8, fontSize: 12, color: '#ff4d4f' }}>
                        {testResult.error}
                      </pre>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </Modal>

      {/* Helm Install Modal */}
      <Modal
        title={t('kubernetes.installChart')}
        open={helmInstallModalVisible}
        onCancel={() => {
          setHelmInstallModalVisible(false);
          helmForm.resetFields();
        }}
        onOk={() => helmForm.submit()}
        confirmLoading={helmLoading}
        width={600}
      >
        <Form
          form={helmForm}
          layout="vertical"
          onFinish={handleInstallHelmChart}
        >
          <Form.Item
            name="releaseName"
            label={t('kubernetes.releaseName')}
            rules={[{ required: true, message: t('kubernetes.pleaseInputReleaseName') }]}
          >
            <Input placeholder={t('kubernetes.pleaseInputReleaseName')} />
          </Form.Item>
          <Form.Item
            name="chartName"
            label={t('kubernetes.chartName')}
            rules={[{ required: true, message: t('kubernetes.pleaseInputChartName') }]}
          >
            <Input placeholder="repo/chart-name" />
          </Form.Item>
          <Form.Item
            name="chartVersion"
            label={t('kubernetes.chartVersion')}
          >
            <Input placeholder={t('kubernetes.latestVersion')} />
          </Form.Item>
          <Form.Item
            name="values"
            label={t('kubernetes.valuesYaml')}
          >
            <Input.TextArea 
              rows={10} 
              placeholder={`# ${t('kubernetes.valuesYamlPlaceholder')}\n# replicaCount: 1\n# image:\n#   repository: nginx\n#   tag: latest`}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Helm Import Modal */}
      <Modal
        title={t('kubernetes.importHelmConfig')}
        open={helmImportModalVisible}
        onCancel={() => {
          setHelmImportModalVisible(false);
          setHelmImportContent('');
        }}
        onOk={handleImportHelmConfig}
        confirmLoading={helmLoading}
        width={700}
      >
        <Alert 
          message={t('kubernetes.importHelmConfigTip')} 
          type="info" 
          showIcon 
          style={{ marginBottom: 16 }}
        />
        <Input.TextArea
          rows={15}
          value={helmImportContent}
          onChange={(e) => setHelmImportContent(e.target.value)}
          placeholder={`{
  "releases": [
    {
      "name": "my-release",
      "chart": "bitnami/nginx",
      "version": "1.0.0",
      "values": {
        "replicaCount": 2
      }
    }
  ]
}`}
          style={{ fontFamily: 'monospace' }}
        />
      </Modal>
    </div>
  );
};

export default KubernetesManagement;