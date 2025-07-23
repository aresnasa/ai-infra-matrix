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
  Descriptions,
  Typography,
  Alert
} from 'antd';
import { 
  PlusOutlined, 
  EditOutlined, 
  DeleteOutlined, 
  ApiOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined
} from '@ant-design/icons';
import api from '../services/api';

const { Title } = Typography;
const { TextArea } = Input;

const JupyterHubConfig = () => {
  const [configs, setConfigs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingConfig, setEditingConfig] = useState(null);
  const [testLoading, setTestLoading] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    fetchConfigs();
  }, []);

  const fetchConfigs = async () => {
    setLoading(true);
    try {
      const response = await api.get('/jupyterhub/configs');
      setConfigs(response.data.configs || []);
    } catch (error) {
      message.error('获取JupyterHub配置失败：' + error.message);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditingConfig(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (config) => {
    setEditingConfig(config);
    form.setFieldsValue({
      ...config,
      gpu_nodes: config.gpu_nodes ? JSON.stringify(JSON.parse(config.gpu_nodes), null, 2) : ''
    });
    setModalVisible(true);
  };

  const handleDelete = async (id) => {
    try {
      await api.delete(`/jupyterhub/configs/${id}`);
      message.success('删除成功');
      fetchConfigs();
    } catch (error) {
      message.error('删除失败：' + error.message);
    }
  };

  const handleSubmit = async (values) => {
    try {
      // 验证GPU节点JSON格式
      if (values.gpu_nodes) {
        try {
          JSON.parse(values.gpu_nodes);
        } catch (error) {
          message.error('GPU节点配置格式错误，请检查JSON格式');
          return;
        }
      }

      const data = {
        ...values,
        gpu_nodes: values.gpu_nodes || '[]'
      };

      if (editingConfig) {
        await api.put(`/jupyterhub/configs/${editingConfig.id}`, data);
        message.success('更新成功');
      } else {
        await api.post('/jupyterhub/configs', data);
        message.success('创建成功');
      }
      
      setModalVisible(false);
      fetchConfigs();
    } catch (error) {
      message.error('操作失败：' + error.message);
    }
  };

  const handleTestConnection = async (config) => {
    setTestLoading(true);
    try {
      const response = await api.post('/jupyterhub/test-connection', {
        url: config.url,
        token: config.token
      });
      if (response.data.connected) {
        message.success('连接测试成功');
      } else {
        message.error('连接测试失败：' + response.data.error);
      }
    } catch (error) {
      message.error('连接测试失败：' + error.message);
    } finally {
      setTestLoading(false);
    }
  };

  const toggleEnabled = async (id, enabled) => {
    try {
      await api.put(`/jupyterhub/configs/${id}`, { is_enabled: enabled });
      message.success(enabled ? '已启用' : '已禁用');
      fetchConfigs();
    } catch (error) {
      message.error('操作失败：' + error.message);
    }
  };

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'URL',
      dataIndex: 'url',
      key: 'url',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'is_enabled',
      key: 'is_enabled',
      render: (enabled, record) => (
        <Switch
          checked={enabled}
          onChange={(checked) => toggleEnabled(record.id, checked)}
          checkedChildren="启用"
          unCheckedChildren="禁用"
        />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            icon={<ApiOutlined />}
            onClick={() => handleTestConnection(record)}
            loading={testLoading}
          >
            测试连接
          </Button>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这个配置吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card 
        title={<Title level={4}>JupyterHub配置管理</Title>}
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={handleCreate}
          >
            添加配置
          </Button>
        }
      >
        <Alert
          message="JupyterHub配置说明"
          description="配置JupyterHub服务器信息，包括URL、访问Token和GPU节点信息。配置完成后可以通过此系统向远程GPU节点提交Python任务。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        
        <Table
          columns={columns}
          dataSource={configs}
          loading={loading}
          rowKey="id"
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
        />
      </Card>

      <Modal
        title={editingConfig ? '编辑JupyterHub配置' : '添加JupyterHub配置'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            is_enabled: true,
            gpu_nodes: JSON.stringify([
              {
                node_name: "gpu-node-1",
                ip_address: "192.168.1.100",
                gpu_count: 2,
                gpu_model: "RTX 4090",
                total_memory_gb: 48,
                available_gpu: 2,
                is_online: true
              }
            ], null, 2)
          }}
        >
          <Form.Item
            name="name"
            label="配置名称"
            rules={[{ required: true, message: '请输入配置名称' }]}
          >
            <Input placeholder="例如：开发环境JupyterHub" />
          </Form.Item>

          <Form.Item
            name="url"
            label="JupyterHub URL"
            rules={[
              { required: true, message: '请输入JupyterHub URL' },
              { type: 'url', message: '请输入有效的URL' }
            ]}
          >
            <Input placeholder="https://jupyterhub.example.com" />
          </Form.Item>

          <Form.Item
            name="token"
            label="API Token"
            rules={[{ required: true, message: '请输入API Token' }]}
          >
            <Input.Password placeholder="JupyterHub API Token" />
          </Form.Item>

          <Form.Item
            name="gpu_nodes"
            label="GPU节点配置"
            extra="JSON格式的GPU节点信息，包括节点名称、IP地址、GPU数量等"
          >
            <TextArea
              rows={10}
              placeholder="GPU节点配置（JSON格式）"
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            name="is_enabled"
            label="启用状态"
            valuePropName="checked"
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setModalVisible(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit">
                {editingConfig ? '更新' : '创建'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default JupyterHubConfig;
