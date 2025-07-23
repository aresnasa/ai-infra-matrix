import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  message,
  Space,
  Typography,
  Tag,
  Switch
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  MenuOutlined
} from '@ant-design/icons';
import { taskAPI } from '../services/api';

const { Text } = Typography;
const { Option } = Select;

const TasksTab = ({ projectId }) => {
  const [tasks, setTasks] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTask, setEditingTask] = useState(null);
  const [form] = Form.useForm();

  // 获取任务列表
  const fetchTasks = async () => {
    setLoading(true);
    try {
      const response = await taskAPI.getTasks(projectId);
      setTasks(response.data || []);
    } catch (error) {
      message.error('获取任务列表失败');
      console.error('Error fetching tasks:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (projectId) {
      fetchTasks();
    }
  }, [projectId]);

  // 处理新增/编辑任务
  const handleSaveTask = async (values) => {
    try {
      const taskData = {
        ...values,
        project_id: projectId,
        order_index: editingTask ? editingTask.order_index : tasks.length
      };

      if (editingTask) {
        await taskAPI.updateTask(editingTask.id, taskData);
        message.success('任务更新成功');
      } else {
        await taskAPI.createTask(taskData);
        message.success('任务创建成功');
      }
      fetchTasks();
      setModalVisible(false);
      setEditingTask(null);
      form.resetFields();
    } catch (error) {
      message.error(editingTask ? '任务更新失败' : '任务创建失败');
      console.error('Error saving task:', error);
    }
  };

  // 处理删除任务
  const handleDeleteTask = async (id) => {
    try {
      await taskAPI.deleteTask(id);
      message.success('任务删除成功');
      fetchTasks();
    } catch (error) {
      message.error('任务删除失败');
      console.error('Error deleting task:', error);
    }
  };

  // 打开编辑模态框
  const handleEditTask = (task) => {
    setEditingTask(task);
    form.setFieldsValue(task);
    setModalVisible(true);
  };

  // 关闭模态框
  const handleCloseModal = () => {
    setModalVisible(false);
    setEditingTask(null);
    form.resetFields();
  };

  // 任务模块选项
  const moduleOptions = [
    { value: 'shell', label: 'Shell命令' },
    { value: 'copy', label: '文件复制' },
    { value: 'template', label: '模板文件' },
    { value: 'service', label: '服务管理' },
    { value: 'package', label: '包管理' },
    { value: 'file', label: '文件操作' },
    { value: 'user', label: '用户管理' },
    { value: 'group', label: '组管理' },
    { value: 'cron', label: '定时任务' },
    { value: 'git', label: 'Git操作' },
    { value: 'docker_container', label: 'Docker容器' },
    { value: 'docker_image', label: 'Docker镜像' },
    { value: 'apt', label: 'APT包管理' },
    { value: 'yum', label: 'YUM包管理' },
    { value: 'systemd', label: 'Systemd服务' },
    { value: 'mysql_user', label: 'MySQL用户' },
    { value: 'mysql_db', label: 'MySQL数据库' },
    { value: 'nginx', label: 'Nginx配置' }
  ];

  const columns = [
    {
      title: '排序',
      dataIndex: 'order_index',
      key: 'order_index',
      width: 60,
      render: () => <MenuOutlined style={{ color: '#999' }} />
    },
    {
      title: '任务名称',
      dataIndex: 'name',
      key: 'name',
      render: (text) => <Text strong>{text}</Text>
    },
    {
      title: '模块',
      dataIndex: 'module',
      key: 'module',
      render: (module) => {
        const moduleInfo = moduleOptions.find(opt => opt.value === module);
        return <Tag color="blue">{moduleInfo ? moduleInfo.label : module}</Tag>;
      }
    },
    {
      title: '参数',
      dataIndex: 'args',
      key: 'args',
      render: (args) => (
        <Text 
          code
          style={{ maxWidth: 200 }} 
          ellipsis={{ tooltip: args }}
        >
          {args}
        </Text>
      )
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled) => (
        <Tag color={enabled ? 'green' : 'red'}>
          {enabled ? '启用' : '禁用'}
        </Tag>
      )
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEditTask(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除这个任务吗？"
            onConfirm={() => handleDeleteTask(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button
              type="link"
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
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setModalVisible(true)}
        >
          添加任务
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={tasks}
        rowKey="id"
        loading={loading}
        pagination={{
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 个任务`
        }}
      />

      <Modal
        title={editingTask ? '编辑任务' : '添加任务'}
        open={modalVisible}
        onCancel={handleCloseModal}
        footer={null}
        width={700}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSaveTask}
          initialValues={{ enabled: true }}
        >
          <Form.Item
            name="name"
            label="任务名称"
            rules={[{ required: true, message: '请输入任务名称' }]}
          >
            <Input placeholder="例如: 安装 Nginx" />
          </Form.Item>

          <Form.Item
            name="module"
            label="Ansible 模块"
            rules={[{ required: true, message: '请选择Ansible模块' }]}
          >
            <Select 
              placeholder="选择Ansible模块"
              showSearch
              filterOption={(input, option) =>
                option.children.toLowerCase().indexOf(input.toLowerCase()) >= 0
              }
            >
              {moduleOptions.map(option => (
                <Option key={option.value} value={option.value}>
                  {option.label}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="args"
            label="模块参数"
            rules={[{ required: true, message: '请输入模块参数' }]}
          >
            <Input.TextArea 
              rows={4}
              placeholder="输入模块参数，使用YAML格式，例如：&#10;name: nginx&#10;state: present"
            />
          </Form.Item>

          <Form.Item
            name="description"
            label="任务描述"
          >
            <Input.TextArea 
              rows={2}
              placeholder="任务描述（可选）"
            />
          </Form.Item>

          <Form.Item
            name="enabled"
            label="启用状态"
            valuePropName="checked"
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={handleCloseModal}>
                取消
              </Button>
              <Button type="primary" htmlType="submit">
                {editingTask ? '更新' : '创建'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default TasksTab;
