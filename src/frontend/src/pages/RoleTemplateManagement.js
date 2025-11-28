import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  message,
  Space,
  Tag,
  Tooltip,
  Popconfirm,
  Typography,
  Row,
  Col,
  Divider,
  InputNumber,
  List,
  Badge,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SyncOutlined,
  SafetyCertificateOutlined,
  CrownOutlined,
  ToolOutlined,
  DatabaseOutlined,
  ExperimentOutlined,
  CodeOutlined,
  UserOutlined,
  LockOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { roleTemplateAPI } from '../services/api';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TextArea } = Input;

// 图标映射
const iconMap = {
  crown: <CrownOutlined />,
  tool: <ToolOutlined />,
  database: <DatabaseOutlined />,
  experiment: <ExperimentOutlined />,
  code: <CodeOutlined />,
  user: <UserOutlined />,
  lock: <LockOutlined />,
  safety: <SafetyCertificateOutlined />,
};

// 颜色选项
const colorOptions = [
  { value: 'red', label: '红色', color: '#f5222d' },
  { value: 'orange', label: '橙色', color: '#fa8c16' },
  { value: 'gold', label: '金色', color: '#faad14' },
  { value: 'green', label: '绿色', color: '#52c41a' },
  { value: 'blue', label: '蓝色', color: '#1890ff' },
  { value: 'purple', label: '紫色', color: '#722ed1' },
  { value: 'cyan', label: '青色', color: '#13c2c2' },
  { value: 'magenta', label: '洋红', color: '#eb2f96' },
];

// 图标选项
const iconOptions = [
  { value: 'crown', label: '皇冠', icon: <CrownOutlined /> },
  { value: 'tool', label: '工具', icon: <ToolOutlined /> },
  { value: 'database', label: '数据库', icon: <DatabaseOutlined /> },
  { value: 'experiment', label: '实验', icon: <ExperimentOutlined /> },
  { value: 'code', label: '代码', icon: <CodeOutlined /> },
  { value: 'user', label: '用户', icon: <UserOutlined /> },
  { value: 'lock', label: '锁', icon: <LockOutlined /> },
  { value: 'safety', label: '安全', icon: <SafetyCertificateOutlined /> },
];

const RoleTemplateManagement = () => {
  const [templates, setTemplates] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState(null);
  const [resources, setResources] = useState([]);
  const [verbs, setVerbs] = useState([]);
  const [form] = Form.useForm();
  const [syncLoading, setSyncLoading] = useState(false);

  // 获取角色模板列表
  const fetchTemplates = useCallback(async () => {
    setLoading(true);
    try {
      const response = await roleTemplateAPI.list();
      setTemplates(response.data || []);
    } catch (error) {
      message.error('获取角色模板列表失败');
      console.error('Error fetching templates:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  // 获取资源和操作列表
  const fetchResourcesAndVerbs = useCallback(async () => {
    try {
      const [resourcesRes, verbsRes] = await Promise.all([
        roleTemplateAPI.getResources(),
        roleTemplateAPI.getVerbs(),
      ]);
      setResources(resourcesRes.data || []);
      setVerbs(verbsRes.data || []);
    } catch (error) {
      console.error('Error fetching resources and verbs:', error);
    }
  }, []);

  useEffect(() => {
    fetchTemplates();
    fetchResourcesAndVerbs();
  }, [fetchTemplates, fetchResourcesAndVerbs]);

  // 打开模态框
  const openModal = (template = null) => {
    setEditingTemplate(template);
    if (template) {
      const permissions = template.permissions?.map(p => ({
        resource: p.resource,
        verb: p.verb,
        scope: p.scope || '*',
      })) || [];
      form.setFieldsValue({
        ...template,
        permissions,
      });
    } else {
      form.resetFields();
      form.setFieldsValue({
        is_active: true,
        priority: 50,
        color: 'blue',
        permissions: [],
      });
    }
    setModalVisible(true);
  };

  // 保存角色模板
  const handleSave = async (values) => {
    try {
      const data = {
        ...values,
        permissions: values.permissions || [],
      };

      if (editingTemplate) {
        await roleTemplateAPI.update(editingTemplate.id, data);
        message.success('角色模板更新成功');
      } else {
        await roleTemplateAPI.create(data);
        message.success('角色模板创建成功');
      }
      setModalVisible(false);
      fetchTemplates();
    } catch (error) {
      message.error(editingTemplate ? '更新失败' : '创建失败');
      console.error('Error saving template:', error);
    }
  };

  // 删除角色模板
  const handleDelete = async (id) => {
    try {
      await roleTemplateAPI.delete(id);
      message.success('角色模板删除成功');
      fetchTemplates();
    } catch (error) {
      message.error('删除失败: ' + (error.response?.data?.error || error.message));
      console.error('Error deleting template:', error);
    }
  };

  // 同步角色模板到角色
  const handleSync = async () => {
    setSyncLoading(true);
    try {
      await roleTemplateAPI.sync();
      message.success('角色模板同步成功');
    } catch (error) {
      message.error('同步失败');
      console.error('Error syncing templates:', error);
    } finally {
      setSyncLoading(false);
    }
  };

  // 表格列定义
  const columns = [
    {
      title: '模板名称',
      dataIndex: 'name',
      key: 'name',
      width: 150,
      render: (text, record) => (
        <Space>
          <span style={{ color: colorOptions.find(c => c.value === record.color)?.color || '#1890ff' }}>
            {iconMap[record.icon] || <SafetyCertificateOutlined />}
          </span>
          <Text strong>{record.display_name || text}</Text>
          {record.is_system && <Tag color="volcano">系统</Tag>}
        </Space>
      ),
    },
    {
      title: '标识',
      dataIndex: 'name',
      key: 'identifier',
      width: 120,
      render: (text) => <Tag color="default">{text}</Tag>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '权限数量',
      dataIndex: 'permissions',
      key: 'permissions_count',
      width: 100,
      render: (permissions) => (
        <Badge count={permissions?.length || 0} showZero style={{ backgroundColor: '#52c41a' }} />
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      sorter: (a, b) => (b.priority || 0) - (a.priority || 0),
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      width: 80,
      render: (isActive) => (
        <Tag color={isActive ? 'success' : 'default'}>
          {isActive ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => openModal(record)}
            />
          </Tooltip>
          {!record.is_system && (
            <Popconfirm
              title="确定要删除这个角色模板吗？"
              onConfirm={() => handleDelete(record.id)}
              okText="删除"
              cancelText="取消"
            >
              <Tooltip title="删除">
                <Button type="text" danger icon={<DeleteOutlined />} />
              </Tooltip>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  // 展开行显示权限详情
  const expandedRowRender = (record) => {
    const permissions = record.permissions || [];
    return (
      <div style={{ padding: '8px 0' }}>
        <Text strong>权限列表：</Text>
        {permissions.length === 0 ? (
          <Text type="secondary" style={{ marginLeft: 8 }}>无权限配置</Text>
        ) : (
          <div style={{ marginTop: 8 }}>
            {permissions.map((perm, index) => (
              <Tag key={index} color="blue" style={{ marginBottom: 4 }}>
                {perm.resource}:{perm.verb}
                {perm.scope && perm.scope !== '*' && `:${perm.scope}`}
              </Tag>
            ))}
          </div>
        )}
      </div>
    );
  };

  return (
    <div style={{ padding: '24px' }}>
      <Row justify="space-between" align="middle" style={{ marginBottom: 24 }}>
        <Col>
          <Title level={2} style={{ margin: 0 }}>
            <SafetyCertificateOutlined style={{ marginRight: 8 }} />
            角色模板管理
          </Title>
          <Paragraph type="secondary" style={{ marginTop: 8, marginBottom: 0 }}>
            管理系统角色模板，配置不同角色的权限策略。模板可以被分配给用户，控制其访问权限。
          </Paragraph>
        </Col>
        <Col>
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={fetchTemplates}
              loading={loading}
            >
              刷新
            </Button>
            <Button
              icon={<SyncOutlined />}
              onClick={handleSync}
              loading={syncLoading}
            >
              同步到角色
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => openModal()}
            >
              创建模板
            </Button>
          </Space>
        </Col>
      </Row>

      <Card>
        <Table
          columns={columns}
          dataSource={templates}
          rowKey="id"
          loading={loading}
          expandable={{
            expandedRowRender,
            rowExpandable: () => true,
          }}
          pagination={{
            pageSize: 10,
            showTotal: (total) => `共 ${total} 个模板`,
          }}
        />
      </Card>

      {/* 创建/编辑模态框 */}
      <Modal
        title={editingTemplate ? '编辑角色模板' : '创建角色模板'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
        okText="保存"
        cancelText="取消"
        width={800}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="模板标识"
                rules={[
                  { required: true, message: '请输入模板标识' },
                  { pattern: /^[a-z][a-z0-9-]*$/, message: '只能包含小写字母、数字和连字符，且以字母开头' },
                ]}
                extra="用于系统内部标识，如 data-developer"
              >
                <Input
                  placeholder="输入模板标识"
                  disabled={editingTemplate?.is_system}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="display_name"
                label="显示名称"
              >
                <Input placeholder="输入显示名称" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="description"
            label="描述"
          >
            <TextArea rows={2} placeholder="输入模板描述" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="color"
                label="颜色"
              >
                <Select placeholder="选择颜色">
                  {colorOptions.map(option => (
                    <Option key={option.value} value={option.value}>
                      <Space>
                        <span style={{
                          display: 'inline-block',
                          width: 12,
                          height: 12,
                          backgroundColor: option.color,
                          borderRadius: 2,
                        }} />
                        {option.label}
                      </Space>
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="icon"
                label="图标"
              >
                <Select placeholder="选择图标">
                  {iconOptions.map(option => (
                    <Option key={option.value} value={option.value}>
                      <Space>
                        {option.icon}
                        {option.label}
                      </Space>
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="priority"
                label="优先级"
                extra="数值越大优先级越高"
              >
                <InputNumber min={0} max={100} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="is_active"
            label="启用状态"
            valuePropName="checked"
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>

          <Divider>权限配置</Divider>

          <Form.List name="permissions">
            {(fields, { add, remove }) => (
              <>
                {fields.map(({ key, name, ...restField }) => (
                  <Row key={key} gutter={8} align="middle" style={{ marginBottom: 8 }}>
                    <Col span={8}>
                      <Form.Item
                        {...restField}
                        name={[name, 'resource']}
                        rules={[{ required: true, message: '请选择资源' }]}
                        style={{ marginBottom: 0 }}
                      >
                        <Select placeholder="选择资源" showSearch>
                          {resources.map(resource => (
                            <Option key={resource} value={resource}>
                              {resource === '*' ? '所有资源' : resource}
                            </Option>
                          ))}
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item
                        {...restField}
                        name={[name, 'verb']}
                        rules={[{ required: true, message: '请选择操作' }]}
                        style={{ marginBottom: 0 }}
                      >
                        <Select placeholder="选择操作" showSearch>
                          {verbs.map(verb => (
                            <Option key={verb} value={verb}>
                              {verb === '*' ? '所有操作' : verb}
                            </Option>
                          ))}
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={6}>
                      <Form.Item
                        {...restField}
                        name={[name, 'scope']}
                        initialValue="*"
                        style={{ marginBottom: 0 }}
                      >
                        <Select placeholder="选择作用域">
                          <Option value="*">所有</Option>
                          <Option value="own">仅自己</Option>
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={2}>
                      <Button
                        type="text"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => remove(name)}
                      />
                    </Col>
                  </Row>
                ))}
                <Form.Item>
                  <Button
                    type="dashed"
                    onClick={() => add({ resource: '', verb: '', scope: '*' })}
                    block
                    icon={<PlusOutlined />}
                  >
                    添加权限
                  </Button>
                </Form.Item>
              </>
            )}
          </Form.List>

          {/* 权限快捷模板 */}
          <div style={{ marginTop: 16 }}>
            <Text type="secondary">快捷添加：</Text>
            <Space style={{ marginTop: 8 }}>
              <Button
                size="small"
                onClick={() => {
                  const current = form.getFieldValue('permissions') || [];
                  form.setFieldsValue({
                    permissions: [...current, { resource: '*', verb: '*', scope: '*' }],
                  });
                }}
              >
                超级管理员
              </Button>
              <Button
                size="small"
                onClick={() => {
                  const current = form.getFieldValue('permissions') || [];
                  const newPerms = [
                    { resource: 'projects', verb: '*', scope: '*' },
                    { resource: 'hosts', verb: 'read', scope: '*' },
                    { resource: 'variables', verb: '*', scope: '*' },
                  ];
                  form.setFieldsValue({
                    permissions: [...current, ...newPerms],
                  });
                }}
              >
                数据开发
              </Button>
              <Button
                size="small"
                onClick={() => {
                  const current = form.getFieldValue('permissions') || [];
                  const newPerms = [
                    { resource: 'saltstack', verb: '*', scope: '*' },
                    { resource: 'ansible', verb: '*', scope: '*' },
                    { resource: 'kubernetes', verb: '*', scope: '*' },
                    { resource: 'hosts', verb: '*', scope: '*' },
                  ];
                  form.setFieldsValue({
                    permissions: [...current, ...newPerms],
                  });
                }}
              >
                SRE
              </Button>
            </Space>
          </div>
        </Form>
      </Modal>
    </div>
  );
};

export default RoleTemplateManagement;
