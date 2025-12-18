import React, { useState, useEffect, useMemo } from 'react';
import {
  Card,
  Form,
  Input,
  Button,
  Select,
  Space,
  message,
  Modal,
  Tabs,
  Row,
  Col,
  Table,
  Tag,
  Tooltip,
  Drawer,
  Code,
  Empty,
  Spin,
  Upload
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  DownloadOutlined,
  UploadOutlined,
  EyeOutlined,
  CopyOutlined,
  FileTextOutlined,
  AppstoreOutlined,
  ConsoleSqlOutlined
} from '@ant-design/icons';
import axios from 'axios';

/**
 * HelmTemplateManager - Helm配置模板管理组件
 * 
 * 功能：
 * 1. 管理预设的Helm配置模板
 * 2. 支持导入导出Helm配置
 * 3. 支持快速部署模板
 * 4. 支持自定义模板编辑
 */
const HelmTemplateManager = ({ cluster }) => {
  const [templates, setTemplates] = useState([]);
  const [loading, setLoading] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState(null);
  const [form] = Form.useForm();
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [previewVisible, setPreviewVisible] = useState(false);
  const [previewContent, setPreviewContent] = useState('');

  // 预设的Helm配置模板
  const presetTemplates = useMemo(() => [
    {
      id: 'nginx-ingress',
      name: 'Nginx Ingress Controller',
      description: '高性能的Nginx Ingress Controller',
      chart: 'ingress-nginx/ingress-nginx',
      namespace: 'ingress-nginx',
      values: `controller:
  service:
    type: LoadBalancer
  resources:
    requests:
      cpu: 100m
      memory: 90Mi
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10`,
      category: 'networking'
    },
    {
      id: 'prometheus',
      name: 'Prometheus Monitoring',
      description: '开源的监控和告警工具',
      chart: 'prometheus-community/kube-prometheus-stack',
      namespace: 'monitoring',
      values: `prometheus:
  prometheusSpec:
    storageSpec:
      volumeClaimTemplate:
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 50Gi
  retention: 30d

grafana:
  enabled: true
  adminPassword: "admin"

alertmanager:
  enabled: true`,
      category: 'monitoring'
    },
    {
      id: 'minio',
      name: 'MinIO Object Storage',
      description: 'S3兼容的对象存储',
      chart: 'minio/minio',
      namespace: 'minio',
      values: `auth:
  rootUser: minioadmin
  rootPassword: minioadmin
  
replicas: 4

persistence:
  enabled: true
  size: 500Gi
  storageClass: ""

defaultBucket:
  enabled: true
  name: data`,
      category: 'storage'
    },
    {
      id: 'postgresql',
      name: 'PostgreSQL Database',
      description: '可靠的关系型数据库',
      chart: 'bitnami/postgresql',
      namespace: 'database',
      values: `auth:
  username: admin
  password: admin123
  
primary:
  persistence:
    enabled: true
    size: 100Gi
    
replica:
  replicaCount: 2
  persistence:
    enabled: true
    size: 100Gi`,
      category: 'database'
    },
    {
      id: 'redis',
      name: 'Redis Cache',
      description: '高性能的内存缓存',
      chart: 'bitnami/redis',
      namespace: 'cache',
      values: `auth:
  enabled: true
  password: "redis123"

replica:
  replicaCount: 3

persistence:
  enabled: true
  size: 50Gi`,
      category: 'cache'
    },
    {
      id: 'elasticsearch',
      name: 'Elasticsearch',
      description: '分布式搜索和分析引擎',
      chart: 'elastic/elasticsearch',
      namespace: 'logging',
      values: `replicas: 3

resources:
  requests:
    cpu: 1000m
    memory: 2Gi
  limits:
    cpu: 2000m
    memory: 4Gi

volumeClaimTemplate:
  accessModes: ["ReadWriteOnce"]
  storageClassName: ""
  resources:
    requests:
      storage: 100Gi`,
      category: 'logging'
    }
  ], []);

  const categories = useMemo(() => {
    const cats = new Set();
    presetTemplates.forEach(t => cats.add(t.category));
    return Array.from(cats);
  }, [presetTemplates]);

  const loadTemplates = async () => {
    if (!cluster) return;
    setLoading(true);
    try {
      const response = await axios.get(
        `/api/kubernetes/clusters/${cluster.id}/helm/templates`
      );
      setTemplates(response.data || presetTemplates);
    } catch (error) {
      console.error('Failed to load templates:', error);
      setTemplates(presetTemplates);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadTemplates();
  }, [cluster]);

  const handleCreateTemplate = async (values) => {
    try {
      const templateData = {
        ...values,
        id: values.id || `template-${Date.now()}`,
        createdAt: new Date().toISOString()
      };

      const response = await axios.post(
        `/api/kubernetes/clusters/${cluster.id}/helm/templates`,
        templateData
      );

      setTemplates([...templates, response.data]);
      form.resetFields();
      setDrawerVisible(false);
      message.success('模板创建成功');
    } catch (error) {
      message.error('模板创建失败: ' + error.message);
    }
  };

  const handleDeleteTemplate = async (templateId) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这个模板吗？',
      okText: '删除',
      cancelText: '取消',
      okType: 'danger',
      onOk: async () => {
        try {
          await axios.delete(
            `/api/kubernetes/clusters/${cluster.id}/helm/templates/${templateId}`
          );
          setTemplates(templates.filter(t => t.id !== templateId));
          message.success('模板删除成功');
        } catch (error) {
          message.error('模板删除失败: ' + error.message);
        }
      }
    });
  };

  const handleDeployTemplate = async (template) => {
    Modal.confirm({
      title: '确认部署',
      content: `确定要部署 "${template.name}" 到命名空间 "${template.namespace}" 吗？`,
      okText: '部署',
      cancelText: '取消',
      onOk: async () => {
        try {
          const response = await axios.post(
            `/api/kubernetes/clusters/${cluster.id}/helm/deploy`,
            {
              chart: template.chart,
              namespace: template.namespace,
              values: template.values,
              name: template.id
            }
          );
          message.success('部署成功');
        } catch (error) {
          message.error('部署失败: ' + error.message);
        }
      }
    });
  };

  const handlePreviewTemplate = (template) => {
    setPreviewContent(template.values);
    setPreviewVisible(true);
  };

  // 复制模板内容（兼容 HTTP 和 HTTPS 环境）
  const handleCopyTemplate = async (template) => {
    const text = template.values;
    // 首先尝试现代 Clipboard API
    if (navigator.clipboard && window.isSecureContext) {
      try {
        await navigator.clipboard.writeText(text);
        message.success('已复制到剪贴板');
        return;
      } catch (err) {
        console.warn('Clipboard API failed, falling back to execCommand:', err);
      }
    }
    
    // 备用方案：使用传统的 execCommand 方式
    const textArea = document.createElement('textarea');
    textArea.value = text;
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
        message.success('已复制到剪贴板');
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

  const columns = [
    {
      title: '模板名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      render: (text, record) => (
        <Space>
          <FileTextOutlined />
          <span>{text}</span>
        </Space>
      )
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 250
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      key: 'namespace',
      width: 150,
      render: (text) => <Tag color="blue">{text}</Tag>
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 100,
      render: (text) => {
        const colorMap = {
          networking: 'cyan',
          monitoring: 'orange',
          storage: 'green',
          database: 'purple',
          cache: 'red',
          logging: 'magenta'
        };
        return <Tag color={colorMap[text] || 'default'}>{text}</Tag>;
      }
    },
    {
      title: '操作',
      key: 'action',
      width: 250,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="预览">
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={() => handlePreviewTemplate(record)}
            />
          </Tooltip>
          <Tooltip title="复制">
            <Button
              type="text"
              icon={<CopyOutlined />}
              onClick={() => handleCopyTemplate(record)}
            />
          </Tooltip>
          <Tooltip title="部署">
            <Button
              type="primary"
              icon={<AppstoreOutlined />}
              onClick={() => handleDeployTemplate(record)}
            />
          </Tooltip>
          <Tooltip title="删除">
            <Button
              type="text"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDeleteTemplate(record.id)}
            />
          </Tooltip>
        </Space>
      )
    }
  ];

  if (!cluster) {
    return <Empty description="请先选择一个集群" />;
  }

  return (
    <Spin spinning={loading}>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <Card
          title={
            <Space>
              <AppstoreOutlined />
              <span>Helm 配置模板管理</span>
            </Space>
          }
          extra={
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setDrawerVisible(true)}
            >
              新增模板
            </Button>
          }
        >
          <Tabs>
            <Tabs.TabPane tab={`所有模板 (${templates.length})`} key="all">
              <Table
                columns={columns}
                dataSource={templates}
                rowKey="id"
                pagination={{ pageSize: 10 }}
              />
            </Tabs.TabPane>
            {categories.map(category => (
              <Tabs.TabPane
                key={category}
                tab={`${category.charAt(0).toUpperCase() + category.slice(1)} (${templates.filter(t => t.category === category).length})`}
              >
                <Table
                  columns={columns}
                  dataSource={templates.filter(t => t.category === category)}
                  rowKey="id"
                  pagination={{ pageSize: 10 }}
                />
              </Tabs.TabPane>
            ))}
          </Tabs>
        </Card>
      </Space>

      {/* 创建/编辑模板抽屉 */}
      <Drawer
        title={editingTemplate ? '编辑模板' : '创建新模板'}
        placement="right"
        width={600}
        onClose={() => {
          setDrawerVisible(false);
          form.resetFields();
          setEditingTemplate(null);
        }}
        visible={drawerVisible}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateTemplate}
        >
          <Form.Item
            name="id"
            label="模板ID"
            rules={[{ required: true, message: '请输入模板ID' }]}
          >
            <Input placeholder="template-name" />
          </Form.Item>
          <Form.Item
            name="name"
            label="模板名称"
            rules={[{ required: true, message: '请输入模板名称' }]}
          >
            <Input placeholder="如: Nginx Ingress Controller" />
          </Form.Item>
          <Form.Item
            name="description"
            label="描述"
            rules={[{ required: true, message: '请输入描述' }]}
          >
            <Input.TextArea rows={3} placeholder="模板的简要说明" />
          </Form.Item>
          <Form.Item
            name="chart"
            label="Chart 地址"
            rules={[{ required: true, message: '请输入Chart地址' }]}
          >
            <Input placeholder="如: nginx-stable/nginx-ingress" />
          </Form.Item>
          <Form.Item
            name="namespace"
            label="命名空间"
            rules={[{ required: true, message: '请输入命名空间' }]}
          >
            <Input placeholder="如: ingress-nginx" />
          </Form.Item>
          <Form.Item
            name="category"
            label="分类"
            rules={[{ required: true, message: '请选择分类' }]}
          >
            <Select placeholder="选择分类">
              <Select.Option value="networking">网络</Select.Option>
              <Select.Option value="monitoring">监控</Select.Option>
              <Select.Option value="storage">存储</Select.Option>
              <Select.Option value="database">数据库</Select.Option>
              <Select.Option value="cache">缓存</Select.Option>
              <Select.Option value="logging">日志</Select.Option>
              <Select.Option value="other">其他</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="values"
            label="Values (YAML)"
            rules={[{ required: true, message: '请输入Values配置' }]}
          >
            <Input.TextArea
              rows={12}
              placeholder="输入Helm Values配置(YAML格式)"
              style={{ fontFamily: 'monospace', fontSize: '12px' }}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                保存模板
              </Button>
              <Button onClick={() => setDrawerVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Drawer>

      {/* 预览模板抽屉 */}
      <Drawer
        title="Values 预览"
        placement="right"
        width={700}
        onClose={() => setPreviewVisible(false)}
        visible={previewVisible}
      >
        <pre style={{
          background: '#f5f5f5',
          padding: '12px',
          borderRadius: '4px',
          fontSize: '12px',
          overflowX: 'auto',
          maxHeight: '80vh'
        }}>
          {previewContent}
        </pre>
      </Drawer>
    </Spin>
  );
};

export default HelmTemplateManager;
