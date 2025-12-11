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
  Tabs,
  Steps,
  Alert,
  Collapse,
  Tree,
  Empty,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ReloadOutlined,
  PlayCircleOutlined,
  HistoryOutlined,
  ExperimentOutlined,
  CodeOutlined,
  CloudServerOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  FileTextOutlined,
  SettingOutlined,
  BugOutlined,
} from '@ant-design/icons';
import { ansibleAPI, kubernetesAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;
const { Option } = Select;
const { TabPane } = Tabs;
const { Step } = Steps;
const { Panel } = Collapse;

const AnsibleManagement = () => {
  const { t } = useI18n();
  const [playbooks, setPlaybooks] = useState([]);
  const [templates, setTemplates] = useState([]);
  const [tplModalOpen, setTplModalOpen] = useState(false);
  const [tplEditing, setTplEditing] = useState(null);
  const [tplForm] = Form.useForm();
  const [clusters, setClusters] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [executeModalVisible, setExecuteModalVisible] = useState(false);
  const [generateModalVisible, setGenerateModalVisible] = useState(false);
  const [editingPlaybook, setEditingPlaybook] = useState(null);
  const [selectedPlaybook, setSelectedPlaybook] = useState(null);
  const [form] = Form.useForm();
  const [executeForm] = Form.useForm();
  const [generateForm] = Form.useForm();
  const [executeLoading, setExecuteLoading] = useState(false);
  const [generateLoading, setGenerateLoading] = useState(false);
  const [executeResult, setExecuteResult] = useState(null);
  const [generatedPlaybook, setGeneratedPlaybook] = useState(null);
  const [currentStep, setCurrentStep] = useState(0);
  const [executionHistory, setExecutionHistory] = useState([]);

  // 获取playbook列表
  const fetchPlaybooks = async () => {
    setLoading(true);
    try {
      const response = await ansibleAPI.getPlaybooks();
      setPlaybooks(response.data.data || []);
    } catch (error) {
      // 优雅处理错误，不显示用户不友好的错误信息
      console.error('获取Playbook列表失败:', error);
      setPlaybooks([]);
      // 只在不是"功能暂未实现"的情况下显示错误信息
      if (!error.message.includes('功能暂未实现')) {
        message.error(t('ansible.fetchFailed') + ': ' + error.message);
      }
    } finally {
      setLoading(false);
    }
  };

  // 获取集群列表
  const fetchClusters = async () => {
    try {
      const response = await kubernetesAPI.getClusters();
      setClusters(response.data.data || []);
    } catch (error) {
      console.error('获取集群列表失败:', error);
    }
  };

  useEffect(() => {
    fetchPlaybooks();
    fetchClusters();
    initTemplates();
  }, []);

  // 模板本地降级初始化
  const initTemplates = async () => {
    try {
      const res = await ansibleAPI.getTemplates();
      setTemplates(res.data?.data || res.data || []);
    } catch (e) {
      const cached = localStorage.getItem('ansible_templates');
      if (cached) {
        setTemplates(JSON.parse(cached));
      } else {
        const defaults = [
          {
            id: 'init-k8s-control-plane',
            name: t('ansible.initK8sControlPlane'),
            description: t('ansible.initK8sControlPlaneDesc'),
            content: '# kubeadm init playbook placeholder',
            variables: { pod_network_cidr: '10.244.0.0/16' }
          },
          {
            id: 'join-k8s-worker',
            name: t('ansible.joinK8sWorker'),
            description: t('ansible.joinK8sWorkerDesc'),
            content: '# kubeadm join playbook placeholder',
            variables: { control_plane_ip: '10.0.0.10' }
          }
        ];
        setTemplates(defaults);
        localStorage.setItem('ansible_templates', JSON.stringify(defaults));
      }
    }
  };

  const saveTemplatesLocal = (list) => {
    setTemplates(list);
    localStorage.setItem('ansible_templates', JSON.stringify(list));
  };

  // 添加/编辑playbook
  const handleSubmit = async (values) => {
    try {
      if (editingPlaybook) {
        await ansibleAPI.updatePlaybook(editingPlaybook.id, values);
        message.success(t('ansible.updateSuccess'));
      } else {
        await ansibleAPI.createPlaybook(values);
        message.success(t('ansible.addSuccess'));
      }
      setModalVisible(false);
      setEditingPlaybook(null);
      form.resetFields();
      fetchPlaybooks();
    } catch (error) {
      message.error((editingPlaybook ? t('ansible.updateFailed') : t('ansible.addFailed')) + ': ' + error.message);
    }
  };

  // 删除playbook
  const handleDelete = async (id) => {
    try {
      await ansibleAPI.deletePlaybook(id);
      message.success(t('ansible.deleteSuccess'));
      fetchPlaybooks();
    } catch (error) {
      message.error(t('common.deleteFailed') + ': ' + error.message);
    }
  };

  // 执行playbook
  const handleExecute = async (playbook) => {
    setSelectedPlaybook(playbook);
  setExecuteModalVisible(true);
    setExecuteResult(null);
    executeForm.setFieldsValue({
      playbookId: playbook.id,
      playbookName: playbook.name,
    });
    
    // 获取执行历史
    try {
      const response = await ansibleAPI.getExecutionHistory(playbook.id);
      setExecutionHistory(response.data.data || []);
    } catch (error) {
      console.error('获取执行历史失败:', error);
    }
  };

  // 执行playbook提交
  const executePlaybook = async (values) => {
    setExecuteLoading(true);
    try {
      const response = await ansibleAPI.executePlaybook(selectedPlaybook.id, values);
      setExecuteResult({
        success: true,
        data: response.data,
      });
      message.success(t('ansible.executeSuccess'));
      
      // 刷新执行历史
      const historyResponse = await ansibleAPI.getExecutionHistory(selectedPlaybook.id);
      setExecutionHistory(historyResponse.data.data || []);
    } catch (error) {
      setExecuteResult({
        success: false,
        error: error.response?.data?.message || error.message,
      });
      message.error(t('ansible.executeFailed'));
    } finally {
      setExecuteLoading(false);
    }
  };

  // 模板CRUD（带后端尝试，失败则本地存储降级）
  const openTplModal = (tpl) => {
    setTplEditing(tpl || null);
    tplForm.resetFields();
    if (tpl) {
      tplForm.setFieldsValue({
        name: tpl.name,
        description: tpl.description,
        variables: typeof tpl.variables === 'string' ? tpl.variables : JSON.stringify(tpl.variables || {}, null, 2),
        content: tpl.content,
      });
    }
    setTplModalOpen(true);
  };

  const submitTpl = async (vals) => {
    const normalized = {
      name: vals.name,
      description: vals.description,
      content: vals.content,
      variables: (() => { try { return JSON.parse(vals.variables || '{}'); } catch { return vals.variables || {}; } })(),
    };
    try {
      if (tplEditing?.id) {
        try { await ansibleAPI.updateTemplate(tplEditing.id, normalized); await initTemplates(); }
        catch { const list = templates.map(t => t.id === tplEditing.id ? { ...t, ...normalized } : t); saveTemplatesLocal(list); }
        message.success(t('ansible.templateUpdated'));
      } else {
        try { await ansibleAPI.createTemplate(normalized); await initTemplates(); }
        catch { const id = `tpl-${Date.now()}`; const list = [...templates, { id, ...normalized }]; saveTemplatesLocal(list); }
        message.success(t('ansible.templateCreated'));
      }
      setTplModalOpen(false);
      setTplEditing(null);
    } catch (e) {
      message.error(t('ansible.templateSaveFailed') + ': ' + e.message);
    }
  };

  const deleteTpl = async (tpl) => {
    try {
      try { await ansibleAPI.deleteTemplate(tpl.id); await initTemplates(); }
      catch { saveTemplatesLocal(templates.filter(t => t.id !== tpl.id)); }
      message.success(t('ansible.templateDeleted'));
    } catch (e) {
      message.error(t('common.deleteFailed') + ': ' + e.message);
    }
  };

  // 生成playbook
  const handleGenerate = () => {
    setGenerateModalVisible(true);
    setGeneratedPlaybook(null);
    setCurrentStep(0);
    generateForm.resetFields();
  };

  // 生成playbook提交
  const generatePlaybook = async (values) => {
    setGenerateLoading(true);
    try {
      const response = await ansibleAPI.generatePlaybook(values);
      setGeneratedPlaybook(response.data.data);
      setCurrentStep(1);
      message.success(t('ansible.generateSuccess'));
    } catch (error) {
      message.error(t('ansible.generateFailed') + ': ' + error.message);
    } finally {
      setGenerateLoading(false);
    }
  };

  // 保存生成的playbook
  const saveGeneratedPlaybook = async (values) => {
    try {
      const playbookData = {
        name: values.name,
        description: values.description,
        content: generatedPlaybook.content,
        variables: generatedPlaybook.variables,
        target_hosts: values.target_hosts,
        cluster_id: values.cluster_id,
      };
      
      await ansibleAPI.createPlaybook(playbookData);
      message.success(t('ansible.saveSuccess'));
      setGenerateModalVisible(false);
      setGeneratedPlaybook(null);
      setCurrentStep(0);
      fetchPlaybooks();
    } catch (error) {
      message.error(t('ansible.saveFailed') + ': ' + error.message);
    }
  };

  // 编辑playbook
  const handleEdit = (playbook) => {
    setEditingPlaybook(playbook);
    form.setFieldsValue({
      ...playbook,
      variables: typeof playbook.variables === 'string' 
        ? playbook.variables 
        : JSON.stringify(playbook.variables, null, 2),
    });
    setModalVisible(true);
  };

  // 状态标签
  const getStatusTag = (status) => {
    const statusMap = {
      success: { color: 'green', icon: <CheckCircleOutlined />, text: t('common.success') },
      failed: { color: 'red', icon: <CloseCircleOutlined />, text: t('common.failed') },
      running: { color: 'blue', icon: <InfoCircleOutlined />, text: t('ansible.running') },
      pending: { color: 'orange', icon: <InfoCircleOutlined />, text: t('ansible.pending') },
    };
    const statusInfo = statusMap[status] || statusMap.pending;
    return (
      <Tag color={statusInfo.color} icon={statusInfo.icon}>
        {statusInfo.text}
      </Tag>
    );
  };

  // 表格列定义
  const columns = [
    {
      title: t('ansible.playbookName'),
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Space>
          <FileTextOutlined />
          <strong>{text}</strong>
        </Space>
      ),
    },
    {
      title: t('ansible.targetHosts'),
      dataIndex: 'target_hosts',
      key: 'target_hosts',
      render: (hosts) => (
        <Text code>{hosts || 'all'}</Text>
      ),
    },
    {
      title: t('ansible.associatedCluster'),
      dataIndex: 'cluster_id',
      key: 'cluster_id',
      render: (clusterId) => {
        const cluster = clusters.find(c => c.id === clusterId);
        return cluster ? (
          <Tag color="blue">
            <CloudServerOutlined /> {cluster.name}
          </Tag>
        ) : '-';
      },
    },
    {
      title: t('ansible.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status) => getStatusTag(status),
    },
    {
      title: t('ansible.lastExecuted'),
      dataIndex: 'last_executed',
      key: 'last_executed',
      render: (time) => time ? new Date(time).toLocaleString() : t('ansible.neverExecuted'),
    },
    {
      title: t('ansible.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('ansible.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 250,
      render: (_, record) => (
        <Space>
          <Tooltip title={t('ansible.execute')}>
            <Button
              type="primary"
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={() => handleExecute(record)}
            />
          </Tooltip>
          <Tooltip title={t('common.edit')}>
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Tooltip title={t('ansible.history')}>
            <Button
              size="small"
              icon={<HistoryOutlined />}
              onClick={() => handleExecute(record)}
            />
          </Tooltip>
          <Popconfirm
            title={t('ansible.confirmDelete')}
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
              <CodeOutlined /> {t('ansible.title')}
            </Title>
          </Col>
          <Col>
            <Space>
              <Button
                icon={<ReloadOutlined />}
                onClick={fetchPlaybooks}
                loading={loading}
              >
                {t('common.refresh')}
              </Button>
              <Button onClick={() => openTplModal()}>
                {t('ansible.newTemplate')}
              </Button>
              <Button
                icon={<BugOutlined />}
                onClick={handleGenerate}
              >
                {t('ansible.smartGenerate')}
              </Button>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => {
                  setEditingPlaybook(null);
                  form.resetFields();
                  setModalVisible(true);
                }}
              >
                {t('ansible.addPlaybook')}
              </Button>
            </Space>
          </Col>
        </Row>

        <Tabs defaultActiveKey="playbooks">
          <Tabs.TabPane tab="Playbooks" key="playbooks">
            <Table
              columns={columns}
              dataSource={playbooks}
              rowKey="id"
              loading={loading}
              locale={{
                emptyText: (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={
                      <span>
                        {t('ansible.noPlaybooks')}
                        <br />
                        <Text type="secondary">{t('ansible.noPlaybooksHint')}</Text>
                      </span>
                    }
                  >
                    <Space>
                      <Button 
                        type="primary" 
                        icon={<BugOutlined />} 
                        onClick={handleGenerate}
                      >
                        {t('ansible.smartGenerate')}
                      </Button>
                      <Button
                        icon={<PlusOutlined />}
                        onClick={() => {
                          setEditingPlaybook(null);
                          form.resetFields();
                          setModalVisible(true);
                        }}
                      >
                        {t('ansible.addPlaybook')}
                      </Button>
                    </Space>
                  </Empty>
                )
              }}
              pagination={{
                total: playbooks.length,
                pageSize: 10,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total) => t('ansible.totalPlaybooks', { total }),
              }}
            />
          </Tabs.TabPane>
          <Tabs.TabPane tab={t('ansible.templateLibrary')} key="templates">
            <Row gutter={[16,16]}>
              {(templates || []).map(tpl => (
                <Col xs={24} md={12} lg={8} key={tpl.id}>
                  <Card
                    size="small"
                    title={tpl.name}
                    extra={
                      <Space>
                        <Button size="small" onClick={() => openTplModal(tpl)}>{t('common.edit')}</Button>
                        <Popconfirm title={t('ansible.confirmDeleteTemplate')} onConfirm={() => deleteTpl(tpl)}>
                          <Button size="small" danger>{t('common.delete')}</Button>
                        </Popconfirm>
                      </Space>
                    }
                  >
                    <div style={{ minHeight: 60 }}>{tpl.description || '-'}</div>
                    <Divider style={{ margin: '8px 0' }} />
                    <Space>
                      <Button size="small" type="primary" onClick={() => {
                        // 通过模板快速创建playbook
                        setEditingPlaybook(null);
                        form.resetFields();
                        form.setFieldsValue({
                          name: tpl.name,
                          description: tpl.description,
                          content: tpl.content,
                          variables: JSON.stringify(tpl.variables || {}, null, 2),
                        });
                        setModalVisible(true);
                      }}>{t('ansible.createFromTemplate')}</Button>
                    </Space>
                  </Card>
                </Col>
              ))}
            </Row>
          </Tabs.TabPane>
        </Tabs>
      </Card>

      {/* 添加/编辑Playbook模态框 */}
      <Modal
        title={editingPlaybook ? t('ansible.editPlaybook') : t('ansible.addPlaybook')}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingPlaybook(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        width={900}
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
                label={t('ansible.playbookName')}
                rules={[{ required: true, message: t('ansible.pleaseInputPlaybookName') }]}
              >
                <Input placeholder={t('ansible.inputPlaybookName')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="cluster_id"
                label={t('ansible.associatedCluster')}
              >
                <Select placeholder={t('ansible.selectCluster')} allowClear>
                  {(clusters || []).map(cluster => (
                    <Option key={cluster.id} value={cluster.id}>
                      {cluster.name}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="target_hosts"
            label={t('ansible.targetHosts')}
            tooltip={t('ansible.targetHostsTooltip')}
          >
            <Input placeholder="all" />
          </Form.Item>

          <Form.Item
            name="content"
            label={t('ansible.playbookContent')}
            rules={[{ required: true, message: t('ansible.pleaseInputPlaybookContent') }]}
          >
            <TextArea
              rows={12}
              placeholder={t('ansible.inputPlaybookContentPlaceholder')}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            name="variables"
            label={t('ansible.variablesConfig')}
            tooltip={t('ansible.variablesConfigTooltip')}
          >
            <TextArea
              rows={4}
              placeholder='{"var1": "value1", "var2": "value2"}'
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('ansible.description')}
          >
            <TextArea
              rows={2}
              placeholder={t('ansible.descriptionOptional')}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 执行Playbook模态框 */}
      <Modal
        title={t('ansible.executePlaybook')}
        open={executeModalVisible}
        onCancel={() => {
          setExecuteModalVisible(false);
          setExecuteResult(null);
        }}
        footer={null}
        width={900}
      >
        {selectedPlaybook && (
          <Tabs defaultActiveKey="execute">
            <TabPane tab={t('ansible.execute')} key="execute">
              <Form
                form={executeForm}
                layout="vertical"
                onFinish={executePlaybook}
              >
                <Alert
                  message={`${t('ansible.aboutToExecute')}: ${selectedPlaybook.name}`}
                  type="info"
                  showIcon
                  style={{ marginBottom: 16 }}
                />

                <Form.Item
                  name="inventory"
                  label={t('ansible.inventoryConfig')}
                  tooltip={t('ansible.inventoryConfigTooltip')}
                >
                  <TextArea
                    rows={4}
                    placeholder={t('ansible.inputInventoryPlaceholder')}
                  />
                </Form.Item>

                <Form.Item
                  name="extra_vars"
                  label={t('ansible.extraVars')}
                  tooltip={t('ansible.extraVarsTooltip')}
                >
                  <TextArea
                    rows={3}
                    placeholder='{"env": "production", "version": "1.0.0"}'
                    style={{ fontFamily: 'monospace' }}
                  />
                </Form.Item>

                <Form.Item
                  name="tags"
                  label={t('ansible.executionTags')}
                  tooltip={t('ansible.executionTagsTooltip')}
                >
                  <Input placeholder="tag1,tag2" />
                </Form.Item>

                <Row>
                  <Col span={24}>
                    <Button
                      type="primary"
                      htmlType="submit"
                      loading={executeLoading}
                      icon={<PlayCircleOutlined />}
                      size="large"
                      block
                    >
                      {t('ansible.startExecution')}
                    </Button>
                  </Col>
                </Row>
              </Form>

              {executeResult && (
                <div style={{ marginTop: 24 }}>
                  <Divider />
                  {executeResult.success ? (
                    <div>
                      <Badge status="success" text={t('ansible.executeSuccess')} />
                      <div style={{ marginTop: 16, padding: 16, backgroundColor: '#f6ffed', border: '1px solid #b7eb8f', borderRadius: 4 }}>
                        <Text strong>{t('ansible.executionResult')}:</Text>
                        <pre style={{ marginTop: 8, fontSize: 12, maxHeight: 300, overflow: 'auto' }}>
                          {JSON.stringify(executeResult.data, null, 2)}
                        </pre>
                      </div>
                    </div>
                  ) : (
                    <div>
                      <Badge status="error" text={t('ansible.executeFailed')} />
                      <div style={{ marginTop: 16, padding: 16, backgroundColor: '#fff2f0', border: '1px solid #ffccc7', borderRadius: 4 }}>
                        <Text strong>{t('ansible.errorInfo')}:</Text>
                        <pre style={{ marginTop: 8, fontSize: 12, color: '#ff4d4f', maxHeight: 300, overflow: 'auto' }}>
                          {executeResult.error}
                        </pre>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </TabPane>

            <TabPane tab={t('ansible.executionHistory')} key="history">
              <Table
                dataSource={executionHistory}
                rowKey="id"
                size="small"
                pagination={false}
                columns={[
                  {
                    title: t('ansible.executedAt'),
                    dataIndex: 'executed_at',
                    render: (time) => new Date(time).toLocaleString(),
                  },
                  {
                    title: t('ansible.status'),
                    dataIndex: 'status',
                    render: (status) => getStatusTag(status),
                  },
                  {
                    title: t('ansible.executor'),
                    dataIndex: 'executor',
                  },
                  {
                    title: t('ansible.duration'),
                    dataIndex: 'duration',
                    render: (duration) => duration ? `${duration}s` : '-',
                  },
                ]}
              />
            </TabPane>
          </Tabs>
        )}
      </Modal>

      {/* 模板编辑/新增 */}
      <Modal
        title={tplEditing ? t('ansible.editTemplate') : t('ansible.newTemplate')}
        open={tplModalOpen}
        onCancel={() => { setTplModalOpen(false); setTplEditing(null); tplForm.resetFields(); }}
        onOk={() => tplForm.submit()}
        width={900}
        okText={t('common.save')}
      >
        <Form form={tplForm} layout="vertical" onFinish={submitTpl}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="name" label={t('ansible.templateName')} rules={[{ required: true, message: t('ansible.pleaseInputTemplateName') }]}>
                <Input placeholder={t('ansible.templateNamePlaceholder')} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="description" label={t('ansible.description')}>
                <Input placeholder={t('ansible.templateDescriptionPlaceholder')} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="variables" label={t('ansible.defaultVariables')} tooltip={t('ansible.defaultVariablesTooltip')}>
            <TextArea rows={6} placeholder='{"pod_network_cidr":"10.244.0.0/16"}' />
          </Form.Item>
          <Form.Item name="content" label={t('ansible.playbookContent')} rules={[{ required: true, message: t('ansible.pleaseInputPlaybookContent') }]}>
            <TextArea rows={12} placeholder={t('ansible.pastePlaybookYaml')} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 智能生成Playbook模态框 */}
      <Modal
        title={t('ansible.smartGeneratePlaybook')}
        open={generateModalVisible}
        onCancel={() => {
          setGenerateModalVisible(false);
          setGeneratedPlaybook(null);
          setCurrentStep(0);
        }}
        footer={null}
        width={1000}
      >
        <Steps current={currentStep} style={{ marginBottom: 24 }}>
          <Step title={t('ansible.configRequirements')} icon={<SettingOutlined />} />
          <Step title={t('ansible.generateResult')} icon={<CodeOutlined />} />
          <Step title={t('ansible.savePlaybook')} icon={<FileTextOutlined />} />
        </Steps>

        {currentStep === 0 && (
          <Form
            form={generateForm}
            layout="vertical"
            onFinish={generatePlaybook}
          >
            <Alert
              message={t('ansible.smartGenerateHint')}
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <Form.Item
              name="task_description"
              label={t('ansible.taskDescription')}
              rules={[{ required: true, message: t('ansible.pleaseDescribeTask') }]}
            >
              <TextArea
                rows={4}
                placeholder={t('ansible.taskDescriptionPlaceholder')}
              />
            </Form.Item>

            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="target_cluster"
                  label={t('ansible.targetCluster')}
                >
                  <Select placeholder={t('ansible.selectTargetCluster')}>
                    {(clusters || []).map(cluster => (
                      <Option key={cluster.id} value={cluster.id}>
                        {cluster.name}
                      </Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="application_type"
                  label={t('ansible.applicationType')}
                >
                  <Select placeholder={t('ansible.selectAppType')}>
                    <Option value="web">{t('ansible.appTypeWeb')}</Option>
                    <Option value="database">{t('ansible.appTypeDatabase')}</Option>
                    <Option value="microservice">{t('ansible.appTypeMicroservice')}</Option>
                    <Option value="monitoring">{t('ansible.appTypeMonitoring')}</Option>
                    <Option value="other">{t('ansible.appTypeOther')}</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              name="requirements"
              label={t('ansible.specialRequirements')}
            >
              <TextArea
                rows={3}
                placeholder={t('ansible.specialRequirementsPlaceholder')}
              />
            </Form.Item>

            <Button
              type="primary"
              htmlType="submit"
              loading={generateLoading}
              icon={<BugOutlined />}
              size="large"
              block
            >
              {t('ansible.generatePlaybook')}
            </Button>
          </Form>
        )}

        {currentStep === 1 && generatedPlaybook && (
          <div>
            <Alert
              message={t('ansible.generateSuccessCheck')}
              type="success"
              showIcon
              style={{ marginBottom: 16 }}
            />
            
            <Collapse defaultActiveKey={['1']}>
              <Panel header={t('ansible.generatedPlaybookContent')} key="1">
                <pre style={{ 
                  backgroundColor: '#f5f5f5', 
                  padding: 16, 
                  borderRadius: 4, 
                  maxHeight: 400, 
                  overflow: 'auto',
                  fontSize: 12,
                  fontFamily: 'monospace'
                }}>
                  {generatedPlaybook.content}
                </pre>
              </Panel>
              {generatedPlaybook.variables && (
                <Panel header={t('ansible.variablesConfig')} key="2">
                  <pre style={{ 
                    backgroundColor: '#f5f5f5', 
                    padding: 16, 
                    borderRadius: 4,
                    fontSize: 12,
                    fontFamily: 'monospace'
                  }}>
                    {JSON.stringify(generatedPlaybook.variables, null, 2)}
                  </pre>
                </Panel>
              )}
            </Collapse>

            <div style={{ marginTop: 16, textAlign: 'right' }}>
              <Space>
                <Button onClick={() => setCurrentStep(0)}>
                  {t('ansible.regenerate')}
                </Button>
                <Button type="primary" onClick={() => setCurrentStep(2)}>
                  {t('ansible.confirmAndSave')}
                </Button>
              </Space>
            </div>
          </div>
        )}

        {currentStep === 2 && (
          <Form
            layout="vertical"
            onFinish={saveGeneratedPlaybook}
            initialValues={{
              cluster_id: generateForm.getFieldValue('target_cluster'),
            }}
          >
            <Form.Item
              name="name"
              label={t('ansible.playbookName')}
              rules={[{ required: true, message: t('ansible.pleaseInputPlaybookName') }]}
            >
              <Input placeholder={t('ansible.inputPlaybookName')} />
            </Form.Item>

            <Form.Item
              name="target_hosts"
              label={t('ansible.targetHosts')}
              initialValue="all"
            >
              <Input placeholder="all" />
            </Form.Item>

            <Form.Item
              name="cluster_id"
              label={t('ansible.associatedCluster')}
            >
              <Select placeholder={t('ansible.selectCluster')}>
                {(clusters || []).map(cluster => (
                  <Option key={cluster.id} value={cluster.id}>
                    {cluster.name}
                  </Option>
                ))}
              </Select>
            </Form.Item>

            <Form.Item
              name="description"
              label={t('ansible.description')}
            >
              <TextArea
                rows={2}
                placeholder={t('ansible.descriptionOptional')}
              />
            </Form.Item>

            <Button type="primary" htmlType="submit" size="large" block>
              {t('ansible.savePlaybook')}
            </Button>
          </Form>
        )}
      </Modal>
    </div>
  );
};

export default AnsibleManagement;