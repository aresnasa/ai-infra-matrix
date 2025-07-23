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

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;
const { Option } = Select;
const { TabPane } = Tabs;
const { Step } = Steps;
const { Panel } = Collapse;

const AnsibleManagement = () => {
  const [playbooks, setPlaybooks] = useState([]);
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
        message.error('获取Playbook列表失败: ' + error.message);
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
  }, []);

  // 添加/编辑playbook
  const handleSubmit = async (values) => {
    try {
      if (editingPlaybook) {
        await ansibleAPI.updatePlaybook(editingPlaybook.id, values);
        message.success('Playbook更新成功');
      } else {
        await ansibleAPI.createPlaybook(values);
        message.success('Playbook添加成功');
      }
      setModalVisible(false);
      setEditingPlaybook(null);
      form.resetFields();
      fetchPlaybooks();
    } catch (error) {
      message.error(editingPlaybook ? '更新失败: ' : '添加失败: ' + error.message);
    }
  };

  // 删除playbook
  const handleDelete = async (id) => {
    try {
      await ansibleAPI.deletePlaybook(id);
      message.success('Playbook删除成功');
      fetchPlaybooks();
    } catch (error) {
      message.error('删除失败: ' + error.message);
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
      message.success('Playbook执行成功');
      
      // 刷新执行历史
      const historyResponse = await ansibleAPI.getExecutionHistory(selectedPlaybook.id);
      setExecutionHistory(historyResponse.data.data || []);
    } catch (error) {
      setExecuteResult({
        success: false,
        error: error.response?.data?.message || error.message,
      });
      message.error('Playbook执行失败');
    } finally {
      setExecuteLoading(false);
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
      message.success('Playbook生成成功');
    } catch (error) {
      message.error('生成失败: ' + error.message);
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
      message.success('Playbook保存成功');
      setGenerateModalVisible(false);
      setGeneratedPlaybook(null);
      setCurrentStep(0);
      fetchPlaybooks();
    } catch (error) {
      message.error('保存失败: ' + error.message);
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
      success: { color: 'green', icon: <CheckCircleOutlined />, text: '成功' },
      failed: { color: 'red', icon: <CloseCircleOutlined />, text: '失败' },
      running: { color: 'blue', icon: <InfoCircleOutlined />, text: '运行中' },
      pending: { color: 'orange', icon: <InfoCircleOutlined />, text: '等待中' },
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
      title: 'Playbook名称',
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
      title: '目标主机',
      dataIndex: 'target_hosts',
      key: 'target_hosts',
      render: (hosts) => (
        <Text code>{hosts || 'all'}</Text>
      ),
    },
    {
      title: '关联集群',
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
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => getStatusTag(status),
    },
    {
      title: '最后执行',
      dataIndex: 'last_executed',
      key: 'last_executed',
      render: (time) => time ? new Date(time).toLocaleString() : '从未执行',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '操作',
      key: 'actions',
      width: 250,
      render: (_, record) => (
        <Space>
          <Tooltip title="执行">
            <Button
              type="primary"
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={() => handleExecute(record)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Tooltip title="历史">
            <Button
              size="small"
              icon={<HistoryOutlined />}
              onClick={() => handleExecute(record)}
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除此Playbook吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除">
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
              <CodeOutlined /> Ansible Playbook 管理
            </Title>
          </Col>
          <Col>
            <Space>
              <Button
                icon={<ReloadOutlined />}
                onClick={fetchPlaybooks}
                loading={loading}
              >
                刷新
              </Button>
              <Button
                icon={<BugOutlined />}
                onClick={handleGenerate}
              >
                智能生成
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
                添加Playbook
              </Button>
            </Space>
          </Col>
        </Row>

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
                    暂无Playbook数据
                    <br />
                    <Text type="secondary">您可以点击"智能生成"或"添加Playbook"开始创建</Text>
                  </span>
                }
              >
                <Space>
                  <Button 
                    type="primary" 
                    icon={<BugOutlined />} 
                    onClick={handleGenerate}
                  >
                    智能生成
                  </Button>
                  <Button
                    icon={<PlusOutlined />}
                    onClick={() => {
                      setEditingPlaybook(null);
                      form.resetFields();
                      setModalVisible(true);
                    }}
                  >
                    添加Playbook
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
            showTotal: (total) => `共 ${total} 个Playbook`,
          }}
        />
      </Card>

      {/* 添加/编辑Playbook模态框 */}
      <Modal
        title={editingPlaybook ? '编辑Playbook' : '添加Playbook'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingPlaybook(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        width={900}
        okText="确定"
        cancelText="取消"
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
                label="Playbook名称"
                rules={[{ required: true, message: '请输入Playbook名称' }]}
              >
                <Input placeholder="输入Playbook名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="cluster_id"
                label="关联集群"
              >
                <Select placeholder="选择关联的Kubernetes集群" allowClear>
                  {clusters.map(cluster => (
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
            label="目标主机"
            tooltip="Ansible inventory中的主机组或主机名，默认为all"
          >
            <Input placeholder="all" />
          </Form.Item>

          <Form.Item
            name="content"
            label="Playbook内容"
            rules={[{ required: true, message: '请输入Playbook内容' }]}
          >
            <TextArea
              rows={12}
              placeholder="输入YAML格式的Ansible Playbook内容..."
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            name="variables"
            label="变量配置"
            tooltip="JSON格式的变量配置"
          >
            <TextArea
              rows={4}
              placeholder='{"var1": "value1", "var2": "value2"}'
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <TextArea
              rows={2}
              placeholder="Playbook描述（可选）"
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 执行Playbook模态框 */}
      <Modal
        title="执行Playbook"
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
            <TabPane tab="执行" key="execute">
              <Form
                form={executeForm}
                layout="vertical"
                onFinish={executePlaybook}
              >
                <Alert
                  message={`即将执行: ${selectedPlaybook.name}`}
                  type="info"
                  showIcon
                  style={{ marginBottom: 16 }}
                />

                <Form.Item
                  name="inventory"
                  label="Inventory配置"
                  tooltip="指定目标主机的inventory配置"
                >
                  <TextArea
                    rows={4}
                    placeholder="输入inventory配置或使用默认配置"
                  />
                </Form.Item>

                <Form.Item
                  name="extra_vars"
                  label="额外变量"
                  tooltip="运行时传递的额外变量，JSON格式"
                >
                  <TextArea
                    rows={3}
                    placeholder='{"env": "production", "version": "1.0.0"}'
                    style={{ fontFamily: 'monospace' }}
                  />
                </Form.Item>

                <Form.Item
                  name="tags"
                  label="执行标签"
                  tooltip="只执行指定标签的任务"
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
                      开始执行
                    </Button>
                  </Col>
                </Row>
              </Form>

              {executeResult && (
                <div style={{ marginTop: 24 }}>
                  <Divider />
                  {executeResult.success ? (
                    <div>
                      <Badge status="success" text="执行成功" />
                      <div style={{ marginTop: 16, padding: 16, backgroundColor: '#f6ffed', border: '1px solid #b7eb8f', borderRadius: 4 }}>
                        <Text strong>执行结果:</Text>
                        <pre style={{ marginTop: 8, fontSize: 12, maxHeight: 300, overflow: 'auto' }}>
                          {JSON.stringify(executeResult.data, null, 2)}
                        </pre>
                      </div>
                    </div>
                  ) : (
                    <div>
                      <Badge status="error" text="执行失败" />
                      <div style={{ marginTop: 16, padding: 16, backgroundColor: '#fff2f0', border: '1px solid #ffccc7', borderRadius: 4 }}>
                        <Text strong>错误信息:</Text>
                        <pre style={{ marginTop: 8, fontSize: 12, color: '#ff4d4f', maxHeight: 300, overflow: 'auto' }}>
                          {executeResult.error}
                        </pre>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </TabPane>

            <TabPane tab="执行历史" key="history">
              <Table
                dataSource={executionHistory}
                rowKey="id"
                size="small"
                pagination={false}
                columns={[
                  {
                    title: '执行时间',
                    dataIndex: 'executed_at',
                    render: (time) => new Date(time).toLocaleString(),
                  },
                  {
                    title: '状态',
                    dataIndex: 'status',
                    render: (status) => getStatusTag(status),
                  },
                  {
                    title: '执行者',
                    dataIndex: 'executor',
                  },
                  {
                    title: '耗时',
                    dataIndex: 'duration',
                    render: (duration) => duration ? `${duration}s` : '-',
                  },
                ]}
              />
            </TabPane>
          </Tabs>
        )}
      </Modal>

      {/* 智能生成Playbook模态框 */}
      <Modal
        title="智能生成Playbook"
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
          <Step title="配置需求" icon={<SettingOutlined />} />
          <Step title="生成结果" icon={<CodeOutlined />} />
          <Step title="保存Playbook" icon={<FileTextOutlined />} />
        </Steps>

        {currentStep === 0 && (
          <Form
            form={generateForm}
            layout="vertical"
            onFinish={generatePlaybook}
          >
            <Alert
              message="描述您希望Ansible Playbook执行的任务，AI将为您生成相应的配置"
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />

            <Form.Item
              name="task_description"
              label="任务描述"
              rules={[{ required: true, message: '请描述您要执行的任务' }]}
            >
              <TextArea
                rows={4}
                placeholder="例如：在Kubernetes集群上部署Nginx应用，配置负载均衡和存储..."
              />
            </Form.Item>

            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="target_cluster"
                  label="目标集群"
                >
                  <Select placeholder="选择目标Kubernetes集群">
                    {clusters.map(cluster => (
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
                  label="应用类型"
                >
                  <Select placeholder="选择应用类型">
                    <Option value="web">Web应用</Option>
                    <Option value="database">数据库</Option>
                    <Option value="microservice">微服务</Option>
                    <Option value="monitoring">监控工具</Option>
                    <Option value="other">其他</Option>
                  </Select>
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              name="requirements"
              label="特殊要求"
            >
              <TextArea
                rows={3}
                placeholder="例如：需要持久化存储、高可用部署、特定的网络配置等..."
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
              生成Playbook
            </Button>
          </Form>
        )}

        {currentStep === 1 && generatedPlaybook && (
          <div>
            <Alert
              message="Playbook生成成功！请检查生成的内容，确认后可以保存"
              type="success"
              showIcon
              style={{ marginBottom: 16 }}
            />
            
            <Collapse defaultActiveKey={['1']}>
              <Panel header="生成的Playbook内容" key="1">
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
                <Panel header="变量配置" key="2">
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
                  重新生成
                </Button>
                <Button type="primary" onClick={() => setCurrentStep(2)}>
                  确认并保存
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
              label="Playbook名称"
              rules={[{ required: true, message: '请输入Playbook名称' }]}
            >
              <Input placeholder="输入Playbook名称" />
            </Form.Item>

            <Form.Item
              name="target_hosts"
              label="目标主机"
              initialValue="all"
            >
              <Input placeholder="all" />
            </Form.Item>

            <Form.Item
              name="cluster_id"
              label="关联集群"
            >
              <Select placeholder="选择关联的Kubernetes集群">
                {clusters.map(cluster => (
                  <Option key={cluster.id} value={cluster.id}>
                    {cluster.name}
                  </Option>
                ))}
              </Select>
            </Form.Item>

            <Form.Item
              name="description"
              label="描述"
            >
              <TextArea
                rows={2}
                placeholder="Playbook描述"
              />
            </Form.Item>

            <Button type="primary" htmlType="submit" size="large" block>
              保存Playbook
            </Button>
          </Form>
        )}
      </Modal>
    </div>
  );
};

export default AnsibleManagement;