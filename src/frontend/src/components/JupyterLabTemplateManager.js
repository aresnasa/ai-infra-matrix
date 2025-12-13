import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Form,
  Input,
  Modal,
  message,
  Space,
  Tag,
  Popconfirm,
  Switch,
  Select,
  InputNumber,
  Divider,
  Row,
  Col,
  Typography,
  Tabs,
  List,
  Tooltip,
  Upload,
  Alert
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  CopyOutlined,
  DownloadOutlined,
  UploadOutlined,
  StarOutlined,
  StarFilled,
  PlayCircleOutlined,
  StopOutlined
} from '@ant-design/icons';
import { CustomMenuIcons } from './CustomIcons';

const { Title, Text } = Typography;
const { Option } = Select;
const { TextArea } = Input;
const { TabPane } = Tabs;

const JupyterLabTemplateManager = () => {
  const [templates, setTemplates] = useState([]);
  const [instances, setInstances] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [instanceModalVisible, setInstanceModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState(null);
  const [activeTab, setActiveTab] = useState('templates');
  const [form] = Form.useForm();
  const [instanceForm] = Form.useForm();

  useEffect(() => {
    fetchTemplates();
    fetchInstances();
  }, []);

  // 获取模板列表
  const fetchTemplates = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/jupyterlab/templates', {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      const data = await response.json();
      setTemplates(data.data || []);
    } catch (error) {
      message.error('获取模板列表失败：' + error.message);
    } finally {
      setLoading(false);
    }
  };

  // 获取实例列表
  const fetchInstances = async () => {
    try {
      const response = await fetch('/api/jupyterlab/instances', {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });
      const data = await response.json();
      setInstances(data.data || []);
    } catch (error) {
      message.error('获取实例列表失败：' + error.message);
    }
  };

  // 创建/更新模板
  const handleSubmit = async (values) => {
    try {
      const templateData = {
        ...values,
        requirements: values.requirements ? values.requirements.split('\n').filter(r => r.trim()) : [],
        conda_packages: values.conda_packages ? values.conda_packages.split('\n').filter(p => p.trim()) : [],
        system_packages: values.system_packages ? values.system_packages.split('\n').filter(p => p.trim()) : [],
        environment_vars: values.environment_vars || [],
        resource_quota: {
          cpu_limit: values.cpu_limit || '2',
          cpu_request: values.cpu_request || '1',
          memory_limit: values.memory_limit || '4Gi',
          memory_request: values.memory_request || '2Gi',
          disk_limit: values.disk_limit || '10Gi',
          gpu_limit: values.gpu_limit || 0,
          gpu_type: values.gpu_type || '',
          max_replicas: values.max_replicas || 1,
          max_lifetime: values.max_lifetime || 7200
        }
      };

      const url = editingTemplate 
        ? `/api/jupyterlab/templates/${editingTemplate.id}`
        : '/api/jupyterlab/templates';
      
      const method = editingTemplate ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify(templateData)
      });

      if (response.ok) {
        message.success(editingTemplate ? '更新成功' : '创建成功');
        setModalVisible(false);
        form.resetFields();
        setEditingTemplate(null);
        fetchTemplates();
      } else {
        const error = await response.json();
        message.error(error.error || '操作失败');
      }
    } catch (error) {
      message.error('操作失败：' + error.message);
    }
  };

  // 删除模板
  const handleDelete = async (id) => {
    try {
      const response = await fetch(`/api/jupyterlab/templates/${id}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      if (response.ok) {
        message.success('删除成功');
        fetchTemplates();
      } else {
        const error = await response.json();
        message.error(error.error || '删除失败');
      }
    } catch (error) {
      message.error('删除失败：' + error.message);
    }
  };

  // 克隆模板
  const handleClone = async (template) => {
    try {
      const response = await fetch(`/api/jupyterlab/templates/${template.id}/clone`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify({
          name: `${template.name} - 副本`
        })
      });

      if (response.ok) {
        message.success('克隆成功');
        fetchTemplates();
      } else {
        const error = await response.json();
        message.error(error.error || '克隆失败');
      }
    } catch (error) {
      message.error('克隆失败：' + error.message);
    }
  };

  // 设置默认模板
  const handleSetDefault = async (id) => {
    try {
      const response = await fetch(`/api/jupyterlab/templates/${id}/default`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      if (response.ok) {
        message.success('设置成功');
        fetchTemplates();
      } else {
        const error = await response.json();
        message.error(error.error || '设置失败');
      }
    } catch (error) {
      message.error('设置失败：' + error.message);
    }
  };

  // 创建实例
  const handleCreateInstance = async (values) => {
    try {
      const response = await fetch('/api/jupyterlab/instances', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        },
        body: JSON.stringify(values)
      });

      if (response.ok) {
        message.success('实例创建成功');
        setInstanceModalVisible(false);
        instanceForm.resetFields();
        fetchInstances();
      } else {
        const error = await response.json();
        message.error(error.error || '创建失败');
      }
    } catch (error) {
      message.error('创建失败：' + error.message);
    }
  };

  // 删除实例
  const handleDeleteInstance = async (id) => {
    try {
      const response = await fetch(`/api/jupyterlab/instances/${id}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`
        }
      });

      if (response.ok) {
        message.success('删除成功');
        fetchInstances();
      } else {
        const error = await response.json();
        message.error(error.error || '删除失败');
      }
    } catch (error) {
      message.error('删除失败：' + error.message);
    }
  };

  // 安全解析JSON字段
  const safeParseJSON = (value, defaultValue = []) => {
    if (!value) return defaultValue;
    if (Array.isArray(value)) return value;
    if (typeof value === 'string') {
      try {
        const parsed = JSON.parse(value);
        return Array.isArray(parsed) ? parsed : defaultValue;
      } catch (e) {
        console.warn('JSON parse failed:', e);
        return defaultValue;
      }
    }
    return defaultValue;
  };

  // 编辑模板
  const handleEdit = (template) => {
    setEditingTemplate(template);
    
    // 设置表单值 - 安全解析JSON字段
    const requirements = safeParseJSON(template.requirements, []);
    const conda_packages = safeParseJSON(template.conda_packages, []);
    const system_packages = safeParseJSON(template.system_packages, []);
    const environment_vars = safeParseJSON(template.environment_vars, []);
    
    const formValues = {
      ...template,
      requirements: requirements.join('\n'),
      conda_packages: conda_packages.join('\n'),
      system_packages: system_packages.join('\n'),
      environment_vars: environment_vars
    };

    // 如果有资源配额，设置配额字段
    if (template.resource_quota) {
      Object.assign(formValues, {
        cpu_limit: template.resource_quota.cpu_limit,
        cpu_request: template.resource_quota.cpu_request,
        memory_limit: template.resource_quota.memory_limit,
        memory_request: template.resource_quota.memory_request,
        disk_limit: template.resource_quota.disk_limit,
        gpu_limit: template.resource_quota.gpu_limit,
        gpu_type: template.resource_quota.gpu_type,
        max_replicas: template.resource_quota.max_replicas,
        max_lifetime: template.resource_quota.max_lifetime
      });
    }

    form.setFieldsValue(formValues);
    setModalVisible(true);
  };

  // 模板表格列
  const templateColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Space>
          {text}
          {record.is_default && <StarFilled style={{ color: '#faad14' }} />}
        </Space>
      )
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true
    },
    {
      title: 'Python版本',
      dataIndex: 'python_version',
      key: 'python_version',
      width: 100
    },
    {
      title: '基础镜像',
      dataIndex: 'base_image',
      key: 'base_image',
      ellipsis: true,
      width: 200
    },
    {
      title: '资源配额',
      key: 'resources',
      width: 150,
      render: (_, record) => {
        const quota = record.resource_quota;
        if (!quota) return '-';
        return (
          <div>
            <div>CPU: {quota.cpu_request}~{quota.cpu_limit}</div>
            <div>内存: {quota.memory_request}~{quota.memory_limit}</div>
            {quota.gpu_limit > 0 && <div>GPU: {quota.gpu_limit}</div>}
          </div>
        );
      }
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      width: 80,
      render: (active) => (
        <Tag color={active ? 'green' : 'red'}>
          {active ? '激活' : '禁用'}
        </Tag>
      )
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (time) => new Date(time).toLocaleString()
    },
    {
      title: '操作',
      key: 'actions',
      width: 200,
      render: (_, record) => (
        <Space>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Tooltip title="克隆">
            <Button
              type="text"
              icon={<CopyOutlined />}
              onClick={() => handleClone(record)}
            />
          </Tooltip>
          {!record.is_default && (
            <Tooltip title="设为默认">
              <Button
                type="text"
                icon={<StarOutlined />}
                onClick={() => handleSetDefault(record.id)}
              />
            </Tooltip>
          )}
          <Tooltip title="启动实例">
            <Button
              type="text"
              icon={<PlayCircleOutlined />}
              onClick={() => {
                instanceForm.setFieldsValue({ template_id: record.id });
                setInstanceModalVisible(true);
              }}
            />
          </Tooltip>
          {record.created_by !== 0 && (
            <Popconfirm
              title="确定要删除这个模板吗？"
              onConfirm={() => handleDelete(record.id)}
              okText="确定"
              cancelText="取消"
            >
              <Tooltip title="删除">
                <Button
                  type="text"
                  danger
                  icon={<DeleteOutlined />}
                />
              </Tooltip>
            </Popconfirm>
          )}
        </Space>
      )
    }
  ];

  // 实例表格列
  const instanceColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: '模板',
      dataIndex: ['template', 'name'],
      key: 'template_name'
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => {
        const statusColors = {
          pending: 'orange',
          running: 'green',
          stopped: 'red',
          failed: 'red'
        };
        return <Tag color={statusColors[status]}>{status}</Tag>;
      }
    },
    {
      title: 'URL',
      dataIndex: 'url',
      key: 'url',
      render: (url) => url ? (
        <a href={url} target="_blank" rel="noopener noreferrer">
          访问JupyterLab
        </a>
      ) : '-'
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => new Date(time).toLocaleString()
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, record) => (
        <Space>
          {record.status === 'running' && record.url && (
            <Button
              type="link"
              href={record.url}
              target="_blank"
              icon={<PlayCircleOutlined />}
            >
              访问
            </Button>
          )}
          <Popconfirm
            title="确定要删除这个实例吗？"
            onConfirm={() => handleDeleteInstance(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      )
    }
  ];

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ marginBottom: '24px' }}>
        <Title level={2}>
          <CustomMenuIcons.Menu size={20} style={{ marginRight: '8px' }} />
          JupyterLab 模板管理
        </Title>
      </div>

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab="模板管理" key="templates">
          <Card
            title="模板列表"
            extra={
              <Space>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => {
                    setEditingTemplate(null);
                    form.resetFields();
                    setModalVisible(true);
                  }}
                >
                  创建模板
                </Button>
              </Space>
            }
          >
            <Table
              columns={templateColumns}
              dataSource={templates}
              rowKey="id"
              loading={loading}
              pagination={{ pageSize: 10 }}
              scroll={{ x: 1200 }}
            />
          </Card>
        </TabPane>

        <TabPane tab="实例管理" key="instances">
          <Card
            title="JupyterLab 实例"
            extra={
              <Button
                type="primary"
                icon={<PlayCircleOutlined />}
                onClick={() => {
                  instanceForm.resetFields();
                  setInstanceModalVisible(true);
                }}
              >
                启动实例
              </Button>
            }
          >
            <Table
              columns={instanceColumns}
              dataSource={instances}
              rowKey="id"
              pagination={{ pageSize: 10 }}
            />
          </Card>
        </TabPane>
      </Tabs>

      {/* 模板编辑模态框 */}
      <Modal
        title={editingTemplate ? '编辑模板' : '创建模板'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingTemplate(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        width={800}
        destroyOnClose
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
                label="模板名称"
                rules={[{ required: true, message: '请输入模板名称' }]}
              >
                <Input placeholder="请输入模板名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="python_version"
                label="Python版本"
                initialValue="3.11"
              >
                <Select>
                  <Option value="3.8">Python 3.8</Option>
                  <Option value="3.9">Python 3.9</Option>
                  <Option value="3.10">Python 3.10</Option>
                  <Option value="3.11">Python 3.11</Option>
                  <Option value="3.12">Python 3.12</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="description"
            label="描述"
          >
            <TextArea rows={2} placeholder="请输入模板描述" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="conda_version"
                label="Conda版本"
                initialValue="23.7.0"
              >
                <Input placeholder="Conda版本" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="base_image"
                label="基础镜像"
                initialValue="jupyter/scipy-notebook:latest"
              >
                <Input placeholder="Docker基础镜像" />
              </Form.Item>
            </Col>
          </Row>

          <Divider>软件包配置</Divider>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="requirements"
                label="Pip包 (每行一个)"
              >
                <TextArea rows={4} placeholder="numpy\npandas\nscikit-learn" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="conda_packages"
                label="Conda包 (每行一个)"
              >
                <TextArea rows={4} placeholder="pytorch\ntensorflow\nxgboost" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="system_packages"
                label="系统包 (每行一个)"
              >
                <TextArea rows={4} placeholder="git\nvim\ncurl" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="startup_script"
            label="启动脚本"
          >
            <TextArea rows={3} placeholder="#!/bin/bash\n# 启动脚本内容" />
          </Form.Item>

          <Divider>资源配额</Divider>

          <Row gutter={16}>
            <Col span={6}>
              <Form.Item
                name="cpu_request"
                label="CPU请求"
                initialValue="1"
              >
                <Input placeholder="1" />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item
                name="cpu_limit"
                label="CPU限制"
                initialValue="2"
              >
                <Input placeholder="2" />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item
                name="memory_request"
                label="内存请求"
                initialValue="2Gi"
              >
                <Input placeholder="2Gi" />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item
                name="memory_limit"
                label="内存限制"
                initialValue="4Gi"
              >
                <Input placeholder="4Gi" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={6}>
              <Form.Item
                name="disk_limit"
                label="磁盘限制"
                initialValue="10Gi"
              >
                <Input placeholder="10Gi" />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item
                name="gpu_limit"
                label="GPU数量"
                initialValue={0}
              >
                <InputNumber min={0} max={8} />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item
                name="gpu_type"
                label="GPU类型"
              >
                <Select placeholder="选择GPU类型" allowClear>
                  <Option value="nvidia.com/gpu">NVIDIA GPU</Option>
                  <Option value="amd.com/gpu">AMD GPU</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item
                name="max_lifetime"
                label="最大运行时间(秒)"
                initialValue={7200}
              >
                <InputNumber min={600} max={86400} />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>

      {/* 实例创建模态框 */}
      <Modal
        title="启动JupyterLab实例"
        open={instanceModalVisible}
        onCancel={() => {
          setInstanceModalVisible(false);
          instanceForm.resetFields();
        }}
        onOk={() => instanceForm.submit()}
        destroyOnClose
      >
        <Form
          form={instanceForm}
          layout="vertical"
          onFinish={handleCreateInstance}
        >
          <Form.Item
            name="template_id"
            label="选择模板"
            rules={[{ required: true, message: '请选择模板' }]}
          >
            <Select placeholder="选择JupyterLab模板">
              {templates.filter(t => t.is_active).map(template => (
                <Option key={template.id} value={template.id}>
                  {template.name}
                  {template.is_default && <StarFilled style={{ color: '#faad14', marginLeft: 8 }} />}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="name"
            label="实例名称"
            rules={[{ required: true, message: '请输入实例名称' }]}
          >
            <Input placeholder="请输入实例名称" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default JupyterLabTemplateManager;
