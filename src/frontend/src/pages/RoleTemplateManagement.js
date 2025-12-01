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
import { useI18n } from '../hooks/useI18n';

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

const RoleTemplateManagement = () => {
  const { t, isEnUS } = useI18n();

  // 获取本地化的显示名称
  const getLocalizedDisplayName = (record) => {
    if (isEnUS && record.display_name_en) {
      return record.display_name_en;
    }
    return record.display_name || record.name;
  };

  // 获取本地化的描述
  const getLocalizedDescription = (record) => {
    if (isEnUS && record.description_en) {
      return record.description_en;
    }
    return record.description || '';
  };

  // 颜色选项
  const colorOptions = [
    { value: 'red', label: t('roleTemplate.red'), color: '#f5222d' },
    { value: 'orange', label: t('roleTemplate.orange'), color: '#fa8c16' },
    { value: 'gold', label: t('roleTemplate.gold'), color: '#faad14' },
    { value: 'green', label: t('roleTemplate.green'), color: '#52c41a' },
    { value: 'blue', label: t('roleTemplate.blue'), color: '#1890ff' },
    { value: 'purple', label: t('roleTemplate.purple'), color: '#722ed1' },
    { value: 'cyan', label: t('roleTemplate.cyan'), color: '#13c2c2' },
    { value: 'magenta', label: t('roleTemplate.magenta'), color: '#eb2f96' },
  ];

  // 图标选项
  const iconOptions = [
    { value: 'crown', label: t('roleTemplate.crown'), icon: <CrownOutlined /> },
    { value: 'tool', label: t('roleTemplate.tool'), icon: <ToolOutlined /> },
    { value: 'database', label: t('roleTemplate.database'), icon: <DatabaseOutlined /> },
    { value: 'experiment', label: t('roleTemplate.experiment'), icon: <ExperimentOutlined /> },
    { value: 'code', label: t('roleTemplate.code'), icon: <CodeOutlined /> },
    { value: 'user', label: t('roleTemplate.user'), icon: <UserOutlined /> },
    { value: 'lock', label: t('roleTemplate.lock'), icon: <LockOutlined /> },
    { value: 'safety', label: t('roleTemplate.safety'), icon: <SafetyCertificateOutlined /> },
  ];
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
      message.error(t('roleTemplate.fetchFailed'));
      console.error('Error fetching templates:', error);
    } finally {
      setLoading(false);
    }
  }, [t]);

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
        message.success(t('roleTemplate.updateSuccess'));
      } else {
        await roleTemplateAPI.create(data);
        message.success(t('roleTemplate.createSuccess'));
      }
      setModalVisible(false);
      fetchTemplates();
    } catch (error) {
      message.error(editingTemplate ? t('roleTemplate.updateFailed') : t('roleTemplate.createFailed'));
      console.error('Error saving template:', error);
    }
  };

  // 删除角色模板
  const handleDelete = async (id) => {
    try {
      await roleTemplateAPI.delete(id);
      message.success(t('roleTemplate.deleteSuccess'));
      fetchTemplates();
    } catch (error) {
      message.error(t('roleTemplate.deleteFailed') + ': ' + (error.response?.data?.error || error.message));
      console.error('Error deleting template:', error);
    }
  };

  // 同步角色模板到角色
  const handleSync = async () => {
    setSyncLoading(true);
    try {
      await roleTemplateAPI.sync();
      message.success(t('roleTemplate.syncSuccess'));
    } catch (error) {
      message.error(t('roleTemplate.syncFailed'));
      console.error('Error syncing templates:', error);
    } finally {
      setSyncLoading(false);
    }
  };

  // 表格列定义
  const columns = [
    {
      title: t('roleTemplate.columns.name'),
      dataIndex: 'name',
      key: 'name',
      width: 150,
      render: (text, record) => (
        <Space>
          <span style={{ color: colorOptions.find(c => c.value === record.color)?.color || '#1890ff' }}>
            {iconMap[record.icon] || <SafetyCertificateOutlined />}
          </span>
          <Text strong>{getLocalizedDisplayName(record)}</Text>
          {record.is_system && <Tag color="volcano">{t('roleTemplate.system')}</Tag>}
        </Space>
      ),
    },
    {
      title: t('roleTemplate.columns.identifier'),
      dataIndex: 'name',
      key: 'identifier',
      width: 120,
      render: (text) => <Tag color="default">{text}</Tag>,
    },
    {
      title: t('roleTemplate.columns.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (text, record) => getLocalizedDescription(record),
    },
    {
      title: t('roleTemplate.columns.permissionCount'),
      dataIndex: 'permissions',
      key: 'permissions_count',
      width: 100,
      render: (permissions) => (
        <Badge count={permissions?.length || 0} showZero style={{ backgroundColor: '#52c41a' }} />
      ),
    },
    {
      title: t('roleTemplate.columns.priority'),
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      sorter: (a, b) => (b.priority || 0) - (a.priority || 0),
    },
    {
      title: t('roleTemplate.columns.status'),
      dataIndex: 'is_active',
      key: 'is_active',
      width: 80,
      render: (isActive) => (
        <Tag color={isActive ? 'success' : 'default'}>
          {isActive ? t('roleTemplate.enabled') : t('roleTemplate.disabled')}
        </Tag>
      ),
    },
    {
      title: t('roleTemplate.columns.actions'),
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space>
          <Tooltip title={t('common.edit')}>
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => openModal(record)}
            />
          </Tooltip>
          {!record.is_system && (
            <Popconfirm
              title={t('roleTemplate.deleteConfirm')}
              onConfirm={() => handleDelete(record.id)}
              okText={t('common.delete')}
              cancelText={t('common.cancel')}
            >
              <Tooltip title={t('common.delete')}>
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
        <Text strong>{t('roleTemplate.permissionList')}：</Text>
        {permissions.length === 0 ? (
          <Text type="secondary" style={{ marginLeft: 8 }}>{t('roleTemplate.noPermissions')}</Text>
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
            {t('roleTemplate.title')}
          </Title>
          <Paragraph type="secondary" style={{ marginTop: 8, marginBottom: 0 }}>
            {t('roleTemplate.description')}
          </Paragraph>
        </Col>
        <Col>
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={fetchTemplates}
              loading={loading}
            >
              {t('common.refresh')}
            </Button>
            <Button
              icon={<SyncOutlined />}
              onClick={handleSync}
              loading={syncLoading}
            >
              {t('roleTemplate.syncToRoles')}
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => openModal()}
            >
              {t('roleTemplate.createTemplate')}
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
            showTotal: (total) => t('roleTemplate.totalTemplates', { count: total }),
          }}
        />
      </Card>

      {/* 创建/编辑模态框 */}
      <Modal
        title={editingTemplate ? t('roleTemplate.editTemplate') : t('roleTemplate.createTemplate')}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
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
                label={t('roleTemplate.form.identifier')}
                rules={[
                  { required: true, message: t('roleTemplate.form.identifierRequired') },
                  { pattern: /^[a-z][a-z0-9-]*$/, message: t('roleTemplate.form.identifierPattern') },
                ]}
                extra={t('roleTemplate.form.identifierHint')}
              >
                <Input
                  placeholder={t('roleTemplate.form.identifierPlaceholder')}
                  disabled={editingTemplate?.is_system}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="display_name"
                label={t('roleTemplate.form.displayName')}
              >
                <Input placeholder={t('roleTemplate.form.displayNamePlaceholder')} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="description"
            label={t('roleTemplate.form.description')}
          >
            <TextArea rows={2} placeholder={t('roleTemplate.form.descriptionPlaceholder')} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="color"
                label={t('roleTemplate.form.color')}
              >
                <Select placeholder={t('roleTemplate.form.colorPlaceholder')}>
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
                label={t('roleTemplate.form.icon')}
              >
                <Select placeholder={t('roleTemplate.form.iconPlaceholder')}>
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
                label={t('roleTemplate.form.priority')}
                extra={t('roleTemplate.form.priorityHint')}
              >
                <InputNumber min={0} max={100} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="is_active"
            label={t('roleTemplate.form.status')}
            valuePropName="checked"
          >
            <Switch checkedChildren={t('roleTemplate.enabled')} unCheckedChildren={t('roleTemplate.disabled')} />
          </Form.Item>

          <Divider>{t('roleTemplate.permissionConfig')}</Divider>

          <Form.List name="permissions">
            {(fields, { add, remove }) => (
              <>
                {fields.map(({ key, name, ...restField }) => (
                  <Row key={key} gutter={8} align="middle" style={{ marginBottom: 8 }}>
                    <Col span={8}>
                      <Form.Item
                        {...restField}
                        name={[name, 'resource']}
                        rules={[{ required: true, message: t('roleTemplate.form.resourceRequired') }]}
                        style={{ marginBottom: 0 }}
                      >
                        <Select placeholder={t('roleTemplate.form.resourcePlaceholder')} showSearch>
                          {resources.map(resource => (
                            <Option key={resource} value={resource}>
                              {resource === '*' ? t('roleTemplate.allResources') : resource}
                            </Option>
                          ))}
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={8}>
                      <Form.Item
                        {...restField}
                        name={[name, 'verb']}
                        rules={[{ required: true, message: t('roleTemplate.form.verbRequired') }]}
                        style={{ marginBottom: 0 }}
                      >
                        <Select placeholder={t('roleTemplate.form.verbPlaceholder')} showSearch>
                          {verbs.map(verb => (
                            <Option key={verb} value={verb}>
                              {verb === '*' ? t('roleTemplate.allOperations') : verb}
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
                        <Select placeholder={t('roleTemplate.form.scopePlaceholder')}>
                          <Option value="*">{t('roleTemplate.scopeAll')}</Option>
                          <Option value="own">{t('roleTemplate.scopeOwn')}</Option>
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
                    {t('roleTemplate.addPermission')}
                  </Button>
                </Form.Item>
              </>
            )}
          </Form.List>

          {/* 权限快捷模板 */}
          <div style={{ marginTop: 16 }}>
            <Text type="secondary">{t('roleTemplate.quickAdd')}：</Text>
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
                {t('roleTemplate.superAdmin')}
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
                {t('roleTemplate.dataDeveloper')}
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
                {t('roleTemplate.sre')}
              </Button>
            </Space>
          </div>
        </Form>
      </Modal>
    </div>
  );
};

export default RoleTemplateManagement;
