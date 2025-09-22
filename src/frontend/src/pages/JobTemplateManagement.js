import React, { useState, useEffect } from 'react';
import {
  Table, Button, Modal, Form, Input, Select, message, Tag, Space,
  Card, Row, Col, Popconfirm, Tooltip, Typography, Divider
} from 'antd';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, CopyOutlined,
  EyeOutlined, CodeOutlined, FileTextOutlined, SaveOutlined,
  GlobalOutlined, LockOutlined
} from '@ant-design/icons';
import axios from 'axios';

const { Option } = Select;
const { TextArea } = Input;
const { Title, Text } = Typography;

const JobTemplateManagement = () => {
  const [templates, setTemplates] = useState([]);
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(false);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [previewModalVisible, setPreviewModalVisible] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState(null);
  const [templateContent, setTemplateContent] = useState('');
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0
  });

  const [form] = Form.useForm();
  const [editForm] = Form.useForm();
  const [filters, setFilters] = useState({
    category: '',
    isPublic: ''
  });

  // 获取模板列表
  const fetchTemplates = async () => {
    setLoading(true);
    try {
      const params = {
        page: pagination.current,
        page_size: pagination.pageSize
      };
      if (filters.category) params.category = filters.category;
      if (filters.isPublic !== '') params.is_public = filters.isPublic;

      const response = await axios.get('/api/job-templates', { params });
      if (response.data.code === 200) {
        setTemplates(response.data.data.templates || []);
        setPagination({
          ...pagination,
          total: response.data.data.total
        });
      }
    } catch (error) {
      message.error('获取模板列表失败: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  // 获取分类列表
  const fetchCategories = async () => {
    try {
      const response = await axios.get('/api/job-templates/categories');
      if (response.data.code === 200) {
        setCategories(response.data.data || []);
      }
    } catch (error) {
      console.error('获取分类失败:', error);
    }
  };

  useEffect(() => {
    fetchTemplates();
    fetchCategories();
  }, [pagination.current, pagination.pageSize, filters]);

  // 创建模板
  const handleCreate = async (values) => {
    try {
      const response = await axios.post('/api/job-templates', values);
      if (response.data.code === 200) {
        message.success('模板创建成功');
        setCreateModalVisible(false);
        form.resetFields();
        fetchTemplates();
        fetchCategories(); // 刷新分类列表
      }
    } catch (error) {
      message.error('创建模板失败: ' + error.message);
    }
  };

  // 更新模板
  const handleEdit = async (values) => {
    try {
      const response = await axios.put(`/api/job-templates/${selectedTemplate.id}`, values);
      if (response.data.code === 200) {
        message.success('模板更新成功');
        setEditModalVisible(false);
        editForm.resetFields();
        setSelectedTemplate(null);
        fetchTemplates();
        fetchCategories(); // 刷新分类列表
      }
    } catch (error) {
      message.error('更新模板失败: ' + error.message);
    }
  };

  // 删除模板
  const handleDelete = async (id) => {
    try {
      const response = await axios.delete(`/api/job-templates/${id}`);
      if (response.data.code === 200) {
        message.success('模板删除成功');
        fetchTemplates();
      }
    } catch (error) {
      message.error('删除模板失败: ' + error.message);
    }
  };

  // 预览模板内容
  const handlePreview = async (template) => {
    setSelectedTemplate(template);
    setTemplateContent(template.script_content);
    setPreviewModalVisible(true);
  };

  // 克隆模板
  const handleClone = async (template) => {
    const newName = `${template.name} - 副本`;
    try {
      const response = await axios.post(`/api/job-templates/${template.id}/clone`, {
        name: newName,
        description: `克隆自: ${template.description}`
      });
      if (response.data.code === 200) {
        message.success('模板克隆成功');
        fetchTemplates();
      }
    } catch (error) {
      message.error('克隆模板失败: ' + error.message);
    }
  };

  // 编辑模板
  const showEditModal = (template) => {
    setSelectedTemplate(template);
    editForm.setFieldsValue({
      name: template.name,
      description: template.description,
      category: template.category,
      script_content: template.script_content,
      parameters: template.parameters,
      is_public: template.is_public
    });
    setEditModalVisible(true);
  };

  const columns = [
    {
      title: '模板名称',
      dataIndex: 'name',
      key: 'name',
      render: (text) => <Text strong>{text}</Text>,
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      render: (text) => text ? <Tag color="blue">{text}</Tag> : '-',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (text) => text || '-',
    },
    {
      title: '可见性',
      dataIndex: 'is_public',
      key: 'is_public',
      render: (isPublic) => (
        <Tag icon={isPublic ? <GlobalOutlined /> : <LockOutlined />} 
             color={isPublic ? 'green' : 'orange'}>
          {isPublic ? '公开' : '私有'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (text) => text ? new Date(text).toLocaleString() : '-',
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Tooltip title="预览">
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={() => handlePreview(record)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => showEditModal(record)}
            />
          </Tooltip>
          <Tooltip title="克隆">
            <Button
              type="text"
              icon={<CopyOutlined />}
              onClick={() => handleClone(record)}
            />
          </Tooltip>
          <Tooltip title="删除">
            <Popconfirm
              title="确定要删除这个模板吗？"
              onConfirm={() => handleDelete(record.id)}
              okText="确定"
              cancelText="取消"
            >
              <Button
                type="text"
                danger
                icon={<DeleteOutlined />}
              />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  const handleTableChange = (paginationConfig) => {
    setPagination({
      ...pagination,
      current: paginationConfig.current,
      pageSize: paginationConfig.pageSize,
    });
  };

  return (
    <div>
      <Card>
        <Row justify="space-between" align="middle" style={{ marginBottom: 16 }}>
          <Col>
            <Title level={4}>作业模板管理</Title>
          </Col>
          <Col>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateModalVisible(true)}
            >
              创建模板
            </Button>
          </Col>
        </Row>

        {/* 筛选器 */}
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}>
            <Select
              placeholder="选择分类"
              allowClear
              style={{ width: '100%' }}
              value={filters.category}
              onChange={(value) => setFilters({ ...filters, category: value })}
            >
              {categories.map(category => (
                <Option key={category} value={category}>{category}</Option>
              ))}
            </Select>
          </Col>
          <Col span={8}>
            <Select
              placeholder="选择可见性"
              allowClear
              style={{ width: '100%' }}
              value={filters.isPublic}
              onChange={(value) => setFilters({ ...filters, isPublic: value })}
            >
              <Option value={true}>公开</Option>
              <Option value={false}>私有</Option>
            </Select>
          </Col>
        </Row>

        <Table
          columns={columns}
          dataSource={templates}
          rowKey="id"
          loading={loading}
          pagination={{
            ...pagination,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 个模板`,
          }}
          onChange={handleTableChange}
        />
      </Card>

      {/* 创建模板对话框 */}
      <Modal
        title="创建作业模板"
        visible={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        width={800}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreate}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="模板名称"
                name="name"
                rules={[{ required: true, message: '请输入模板名称' }]}
              >
                <Input placeholder="输入模板名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="分类"
                name="category"
              >
                <Input placeholder="输入分类，如：deep-learning, hpc, data-processing" />
              </Form.Item>
            </Col>
          </Row>
          
          <Form.Item
            label="描述"
            name="description"
          >
            <TextArea rows={2} placeholder="输入模板描述" />
          </Form.Item>

          <Form.Item
            label="脚本内容"
            name="script_content"
            rules={[{ required: true, message: '请输入脚本内容' }]}
          >
            <TextArea 
              rows={12} 
              placeholder="输入 SLURM 脚本模板内容..."
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            label="参数定义"
            name="parameters"
            tooltip="JSON 格式的参数定义，用于脚本模板中的变量替换"
          >
            <TextArea 
              rows={4} 
              placeholder='例如：{"nodes": 1, "cpus_per_task": 4, "memory": "8G"}'
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            label="可见性"
            name="is_public"
            valuePropName="checked"
            initialValue={false}
          >
            <Select>
              <Option value={false}>私有（仅自己可见）</Option>
              <Option value={true}>公开（所有用户可见）</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑模板对话框 */}
      <Modal
        title="编辑作业模板"
        visible={editModalVisible}
        onCancel={() => {
          setEditModalVisible(false);
          editForm.resetFields();
          setSelectedTemplate(null);
        }}
        onOk={() => editForm.submit()}
        width={800}
      >
        <Form
          form={editForm}
          layout="vertical"
          onFinish={handleEdit}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="模板名称"
                name="name"
                rules={[{ required: true, message: '请输入模板名称' }]}
              >
                <Input placeholder="输入模板名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="分类"
                name="category"
              >
                <Input placeholder="输入分类" />
              </Form.Item>
            </Col>
          </Row>
          
          <Form.Item
            label="描述"
            name="description"
          >
            <TextArea rows={2} placeholder="输入模板描述" />
          </Form.Item>

          <Form.Item
            label="脚本内容"
            name="script_content"
            rules={[{ required: true, message: '请输入脚本内容' }]}
          >
            <TextArea 
              rows={12} 
              placeholder="输入 SLURM 脚本模板内容..."
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            label="参数定义"
            name="parameters"
          >
            <TextArea 
              rows={4} 
              placeholder='JSON 格式的参数定义'
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item
            label="可见性"
            name="is_public"
          >
            <Select>
              <Option value={false}>私有（仅自己可见）</Option>
              <Option value={true}>公开（所有用户可见）</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 预览模板对话框 */}
      <Modal
        title={`预览模板: ${selectedTemplate?.name}`}
        visible={previewModalVisible}
        onCancel={() => {
          setPreviewModalVisible(false);
          setSelectedTemplate(null);
          setTemplateContent('');
        }}
        footer={[
          <Button key="close" onClick={() => setPreviewModalVisible(false)}>
            关闭
          </Button>
        ]}
        width={800}
      >
        <div>
          <Divider orientation="left">基本信息</Divider>
          <Row gutter={16}>
            <Col span={8}>
              <Text strong>分类：</Text> {selectedTemplate?.category || '无'}
            </Col>
            <Col span={8}>
              <Text strong>可见性：</Text> {selectedTemplate?.is_public ? '公开' : '私有'}
            </Col>
            <Col span={8}>
              <Text strong>创建时间：</Text> {selectedTemplate?.created_at ? new Date(selectedTemplate.created_at).toLocaleString() : '-'}
            </Col>
          </Row>
          
          {selectedTemplate?.description && (
            <>
              <Divider orientation="left">描述</Divider>
              <Text>{selectedTemplate.description}</Text>
            </>
          )}
          
          <Divider orientation="left">脚本内容</Divider>
          <TextArea
            value={templateContent}
            rows={15}
            readOnly
            style={{ 
              fontFamily: 'monospace',
              backgroundColor: '#f5f5f5'
            }}
          />
          
          {selectedTemplate?.parameters && (
            <>
              <Divider orientation="left">参数定义</Divider>
              <TextArea
                value={selectedTemplate.parameters}
                rows={4}
                readOnly
                style={{ 
                  fontFamily: 'monospace',
                  backgroundColor: '#f5f5f5'
                }}
              />
            </>
          )}
        </div>
      </Modal>
    </div>
  );
};

export default JobTemplateManagement;