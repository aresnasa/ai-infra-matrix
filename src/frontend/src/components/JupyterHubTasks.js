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
  Select,
  InputNumber,
  Progress,
  Descriptions,
  Typography,
  Alert,
  Tooltip,
  Badge
} from 'antd';
import { 
  PlusOutlined, 
  PlayCircleOutlined, 
  StopOutlined, 
  EyeOutlined,
  ReloadOutlined,
  DeleteOutlined,
  FileTextOutlined
} from '@ant-design/icons';
import api from '../services/api';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;

const JupyterHubTasks = () => {
  const [tasks, setTasks] = useState([]);
  const [configs, setConfigs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [outputModalVisible, setOutputModalVisible] = useState(false);
  const [selectedTask, setSelectedTask] = useState(null);
  const [taskOutput, setTaskOutput] = useState('');
  const [form] = Form.useForm();
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0
  });

  useEffect(() => {
    fetchTasks();
    fetchConfigs();
  }, []);

  const fetchTasks = async (page = 1, pageSize = 10) => {
    setLoading(true);
    try {
      const response = await api.get('/jupyterhub/tasks', {
        params: { page, limit: pageSize }
      });
      setTasks(response.data.tasks || []);
      setPagination({
        current: page,
        pageSize,
        total: response.data.total
      });
    } catch (error) {
      message.error('获取任务列表失败：' + error.message);
    } finally {
      setLoading(false);
    }
  };

  const fetchConfigs = async () => {
    try {
      const response = await api.get('/jupyterhub/configs');
      setConfigs(response.data.configs || []);
    } catch (error) {
      message.error('获取配置列表失败：' + error.message);
    }
  };

  const handleCreate = () => {
    form.resetFields();
    setModalVisible(true);
  };

  const handleSubmit = async (values) => {
    try {
      await api.post('/jupyterhub/tasks', values);
      message.success('任务创建成功');
      setModalVisible(false);
      fetchTasks();
    } catch (error) {
      message.error('创建任务失败：' + error.message);
    }
  };

  const handleCancel = async (taskId) => {
    try {
      await api.post(`/jupyterhub/tasks/${taskId}/cancel`);
      message.success('任务已取消');
      fetchTasks();
    } catch (error) {
      message.error('取消任务失败：' + error.message);
    }
  };

  const handleViewOutput = async (task) => {
    setSelectedTask(task);
    try {
      const response = await api.get(`/jupyterhub/tasks/${task.id}/output`);
      setTaskOutput(response.data.output || '暂无输出');
    } catch (error) {
      setTaskOutput('获取输出失败：' + error.message);
    }
    setOutputModalVisible(true);
  };

  const getStatusColor = (status) => {
    const colors = {
      pending: 'orange',
      running: 'blue', 
      completed: 'green',
      failed: 'red',
      cancelled: 'grey'
    };
    return colors[status] || 'default';
  };

  const getStatusText = (status) => {
    const texts = {
      pending: '等待中',
      running: '运行中',
      completed: '已完成',
      failed: '失败',
      cancelled: '已取消'
    };
    return texts[status] || status;
  };

  const handleTableChange = (page, pageSize) => {
    fetchTasks(page, pageSize);
  };

  const columns = [
    {
      title: '任务名称',
      dataIndex: 'task_name',
      key: 'task_name',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Badge 
          status={status === 'completed' ? 'success' : status === 'failed' ? 'error' : 'processing'}
          text={getStatusText(status)}
        />
      ),
    },
    {
      title: 'GPU需求',
      dataIndex: 'gpu_requested',
      key: 'gpu_requested',
      render: (gpu) => gpu > 0 ? `${gpu} GPU` : '无GPU',
    },
    {
      title: '资源配置',
      key: 'resources',
      render: (_, record) => (
        <div>
          <div>内存: {record.memory_gb}GB</div>
          <div>CPU: {record.cpu_cores}核</div>
        </div>
      ),
    },
    {
      title: '配置',
      dataIndex: 'hub_config',
      key: 'hub_config',
      render: (config) => config?.name || '未知配置',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: '执行时间',
      key: 'execution_time',
      render: (_, record) => {
        if (record.started_at && record.completed_at) {
          const start = new Date(record.started_at);
          const end = new Date(record.completed_at);
          const duration = Math.round((end - start) / 1000);
          return `${duration}秒`;
        }
        if (record.started_at) {
          const start = new Date(record.started_at);
          const now = new Date();
          const duration = Math.round((now - start) / 1000);
          return `${duration}秒（运行中）`;
        }
        return '-';
      },
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Tooltip title="查看输出">
            <Button
              type="link"
              icon={<EyeOutlined />}
              onClick={() => handleViewOutput(record)}
            >
              输出
            </Button>
          </Tooltip>
          {record.status === 'running' && (
            <Tooltip title="取消任务">
              <Button
                type="link"
                danger
                icon={<StopOutlined />}
                onClick={() => handleCancel(record.id)}
              >
                取消
              </Button>
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card 
        title={<Title level={4}>JupyterHub任务管理</Title>}
        extra={
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={() => fetchTasks()}
            >
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={handleCreate}
            >
              创建任务
            </Button>
          </Space>
        }
      >
        <Alert
          message="任务执行说明"
          description="提交Python代码到远程GPU节点执行。任务会通过Ansible自动部署到配置的GPU节点上运行。"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        
        <Table
          columns={columns}
          dataSource={tasks}
          loading={loading}
          rowKey="id"
          pagination={{
            ...pagination,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            onChange: handleTableChange,
            onShowSizeChange: handleTableChange,
          }}
        />
      </Card>

      {/* 创建任务Modal */}
      <Modal
        title="创建JupyterHub任务"
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
            gpu_requested: 0,
            memory_gb: 4,
            cpu_cores: 2,
          }}
        >
          <Form.Item
            name="task_name"
            label="任务名称"
            rules={[{ required: true, message: '请输入任务名称' }]}
          >
            <Input placeholder="任务名称" />
          </Form.Item>

          <Form.Item
            name="hub_config_id"
            label="JupyterHub配置"
            rules={[{ required: true, message: '请选择JupyterHub配置' }]}
          >
            <Select placeholder="选择JupyterHub配置">
              {configs.filter(c => c.is_enabled).map(config => (
                <Option key={config.id} value={config.id}>
                  {config.name} ({config.url})
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="python_code"
            label="Python代码"
            rules={[{ required: true, message: '请输入Python代码' }]}
          >
            <TextArea
              rows={12}
              placeholder={`# 输入要执行的Python代码
import torch
import numpy as np

# 检查GPU是否可用
if torch.cuda.is_available():
    print(f"GPU数量: {torch.cuda.device_count()}")
    print(f"当前GPU: {torch.cuda.get_device_name()}")
else:
    print("未检测到GPU")

# 你的代码...
print("Hello from JupyterHub!")
`}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item label="资源配置">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Form.Item
                name="gpu_requested"
                label="GPU数量"
                style={{ marginBottom: 8 }}
              >
                <InputNumber min={0} max={8} />
              </Form.Item>
              <Form.Item
                name="memory_gb"
                label="内存(GB)"
                style={{ marginBottom: 8 }}
              >
                <InputNumber min={1} max={128} />
              </Form.Item>
              <Form.Item
                name="cpu_cores"
                label="CPU核心数"
                style={{ marginBottom: 0 }}
              >
                <InputNumber min={1} max={32} />
              </Form.Item>
            </Space>
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setModalVisible(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit">
                创建任务
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 输出查看Modal */}
      <Modal
        title={`任务输出 - ${selectedTask?.task_name}`}
        open={outputModalVisible}
        onCancel={() => setOutputModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setOutputModalVisible(false)}>
            关闭
          </Button>
        ]}
        width={1000}
      >
        {selectedTask && (
          <div style={{ marginBottom: 16 }}>
            <Descriptions size="small" column={2}>
              <Descriptions.Item label="状态">
                <Badge 
                  status={selectedTask.status === 'completed' ? 'success' : 
                          selectedTask.status === 'failed' ? 'error' : 'processing'}
                  text={getStatusText(selectedTask.status)}
                />
              </Descriptions.Item>
              <Descriptions.Item label="任务ID">{selectedTask.job_id}</Descriptions.Item>
              <Descriptions.Item label="GPU需求">{selectedTask.gpu_requested || 0}</Descriptions.Item>
              <Descriptions.Item label="内存">{selectedTask.memory_gb}GB</Descriptions.Item>
            </Descriptions>
          </div>
        )}
        <div 
          style={{ 
            backgroundColor: '#f5f5f5', 
            padding: 16, 
            borderRadius: 4,
            fontFamily: 'monospace',
            fontSize: 12,
            maxHeight: 400,
            overflow: 'auto',
            whiteSpace: 'pre-wrap'
          }}
        >
          {taskOutput}
        </div>
      </Modal>
    </div>
  );
};

export default JupyterHubTasks;
