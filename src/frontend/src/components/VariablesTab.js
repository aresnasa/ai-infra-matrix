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
  Typography
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined
} from '@ant-design/icons';
import { variableAPI } from '../services/api';

const { Text } = Typography;
const { Option } = Select;

const VariablesTab = ({ projectId }) => {
  const [variables, setVariables] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingVariable, setEditingVariable] = useState(null);
  const [form] = Form.useForm();

  // 获取变量列表
  const fetchVariables = async () => {
    setLoading(true);
    try {
      const response = await variableAPI.getVariables(projectId);
      setVariables(response.data || []);
    } catch (error) {
      message.error('获取变量列表失败');
      console.error('Error fetching variables:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (projectId) {
      fetchVariables();
    }
  }, [projectId]);

  // 处理新增/编辑变量
  const handleSaveVariable = async (values) => {
    try {
      if (editingVariable) {
        await variableAPI.updateVariable(editingVariable.id, values);
        message.success('变量更新成功');
      } else {
        await variableAPI.createVariable({ ...values, project_id: projectId });
        message.success('变量创建成功');
      }
      fetchVariables();
      setModalVisible(false);
      setEditingVariable(null);
      form.resetFields();
    } catch (error) {
      message.error(editingVariable ? '变量更新失败' : '变量创建失败');
      console.error('Error saving variable:', error);
    }
  };

  // 处理删除变量
  const handleDeleteVariable = async (id) => {
    try {
      await variableAPI.deleteVariable(id);
      message.success('变量删除成功');
      fetchVariables();
    } catch (error) {
      message.error('变量删除失败');
      console.error('Error deleting variable:', error);
    }
  };

  // 打开编辑模态框
  const handleEditVariable = (variable) => {
    setEditingVariable(variable);
    form.setFieldsValue(variable);
    setModalVisible(true);
  };

  // 关闭模态框
  const handleCloseModal = () => {
    setModalVisible(false);
    setEditingVariable(null);
    form.resetFields();
  };

  const columns = [
    {
      title: '变量名',
      dataIndex: 'name',
      key: 'name',
      render: (text) => <Text code>{text}</Text>
    },
    {
      title: '变量值',
      dataIndex: 'value',
      key: 'value',
      render: (text) => (
        <Text 
          style={{ maxWidth: 200 }} 
          ellipsis={{ tooltip: text }}
        >
          {text}
        </Text>
      )
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type) => {
        const typeMap = {
          string: '字符串',
          number: '数字',
          boolean: '布尔值',
          list: '列表',
          dict: '字典'
        };
        return typeMap[type] || type;
      }
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      render: (text) => (
        <Text 
          style={{ maxWidth: 150 }} 
          ellipsis={{ tooltip: text }}
        >
          {text || '-'}
        </Text>
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
            onClick={() => handleEditVariable(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除这个变量吗？"
            onConfirm={() => handleDeleteVariable(record.id)}
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
          添加变量
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={variables}
        rowKey="id"
        loading={loading}
        pagination={{
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 个变量`
        }}
      />

      <Modal
        title={editingVariable ? '编辑变量' : '添加变量'}
        open={modalVisible}
        onCancel={handleCloseModal}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSaveVariable}
        >
          <Form.Item
            name="name"
            label="变量名"
            rules={[
              { required: true, message: '请输入变量名' },
              { pattern: /^[a-zA-Z_][a-zA-Z0-9_]*$/, message: '变量名只能包含字母、数字和下划线，且不能以数字开头' }
            ]}
          >
            <Input placeholder="例如: app_port" />
          </Form.Item>

          <Form.Item
            name="type"
            label="变量类型"
            rules={[{ required: true, message: '请选择变量类型' }]}
          >
            <Select placeholder="选择变量类型">
              <Option value="string">字符串</Option>
              <Option value="number">数字</Option>
              <Option value="boolean">布尔值</Option>
              <Option value="list">列表</Option>
              <Option value="dict">字典</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="value"
            label="变量值"
            rules={[{ required: true, message: '请输入变量值' }]}
          >
            <Input.TextArea 
              rows={3}
              placeholder="输入变量值，列表和字典请使用YAML格式"
            />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
          >
            <Input.TextArea 
              rows={2}
              placeholder="变量描述（可选）"
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={handleCloseModal}>
                取消
              </Button>
              <Button type="primary" htmlType="submit">
                {editingVariable ? '更新' : '创建'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default VariablesTab;
