import React, { useState, useEffect } from 'react';
import {
  Card, Form, Input, Select, Button, Switch, Space, message,
  Modal, Table, Typography, Popconfirm, Tag, Alert, Row, Col,
  Tabs, InputNumber, Tooltip, Divider
} from 'antd';
import {
  PlusOutlined, EditOutlined, DeleteOutlined, CheckCircleOutlined,
  ExclamationCircleOutlined, DatabaseOutlined, KeyOutlined,
  LinkOutlined, ExperimentOutlined, SaveOutlined, ReloadOutlined
} from '@ant-design/icons';
import { objectStorageAPI } from '../../services/api';

const { Title, Text } = Typography;
const { Option } = Select;
const { TabPane } = Tabs;
const { TextArea } = Input;

const ObjectStorageConfigPage = () => {
  const [form] = Form.useForm();
  const [configs, setConfigs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingConfig, setEditingConfig] = useState(null);
  const [testingConnection, setTestingConnection] = useState(false);

  // 存储类型配置
  const storageTypes = [
    {
      value: 'minio',
      label: 'MinIO',
      icon: <DatabaseOutlined />,
      description: '高性能分布式对象存储，兼容S3 API',
      requiresWebUrl: true
    },
    {
      value: 'aws_s3',
      label: 'Amazon S3',
      icon: <DatabaseOutlined />,
      description: 'AWS原生对象存储服务',
      requiresWebUrl: false
    },
    {
      value: 'aliyun_oss',
      label: '阿里云OSS',
      icon: <DatabaseOutlined />,
      description: '阿里云对象存储服务',
      requiresWebUrl: false
    },
    {
      value: 'tencent_cos',
      label: '腾讯云COS',
      icon: <DatabaseOutlined />,
      description: '腾讯云对象存储',
      requiresWebUrl: false
    }
  ];

  // 加载配置列表
  const loadConfigs = async () => {
    setLoading(true);
    try {
      const response = await objectStorageAPI.getConfigs();
      setConfigs(response.data?.data || []);
    } catch (error) {
      console.error('加载配置失败:', error);
      message.error('加载配置失败: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  // 测试连接
  const testConnection = async (config) => {
    setTestingConnection(true);
    try {
      const response = await objectStorageAPI.testConnection(config);
      if (response.data?.success) {
        message.success('连接测试成功');
        return true;
      } else {
        message.error('连接测试失败: ' + (response.data?.error || '未知错误'));
        return false;
      }
    } catch (error) {
      console.error('连接测试失败:', error);
      message.error('连接测试失败: ' + (error.response?.data?.error || error.message));
      return false;
    } finally {
      setTestingConnection(false);
    }
  };

  // 保存配置
  const saveConfig = async (values) => {
    try {
      if (editingConfig) {
        // 更新配置
        await objectStorageAPI.updateConfig(editingConfig.id, values);
        message.success('配置更新成功');
      } else {
        // 创建新配置
        await objectStorageAPI.createConfig(values);
        message.success('配置创建成功');
      }
      
      setModalVisible(false);
      setEditingConfig(null);
      form.resetFields();
      await loadConfigs();
    } catch (error) {
      console.error('保存配置失败:', error);
      message.error('保存配置失败: ' + (error.response?.data?.error || error.message));
    }
  };

  // 删除配置
  const deleteConfig = async (id) => {
    try {
      await objectStorageAPI.deleteConfig(id);
      message.success('配置删除成功');
      await loadConfigs();
    } catch (error) {
      console.error('删除配置失败:', error);
      message.error('删除配置失败: ' + (error.response?.data?.error || error.message));
    }
  };

  // 设置为激活配置
  const setActiveConfig = async (id) => {
    try {
      await objectStorageAPI.setActiveConfig(id);
      message.success('已设置为激活配置');
      await loadConfigs();
    } catch (error) {
      console.error('设置激活配置失败:', error);
      message.error('设置激活配置失败: ' + (error.response?.data?.error || error.message));
    }
  };

  // 打开配置模态框
  const openConfigModal = (config = null) => {
    setEditingConfig(config);
    setModalVisible(true);
    
    if (config) {
      form.setFieldsValue(config);
    } else {
      form.resetFields();
      form.setFieldsValue({
        type: 'minio',
        is_active: false,
        ssl_enabled: false,
        timeout: 30
      });
    }
  };

  // 获取存储类型配置
  const getStorageTypeConfig = (type) => {
    return storageTypes.find(t => t.value === type) || storageTypes[0];
  };

  // 处理表单提交
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      
      // 如果是测试并保存，先测试连接
      const shouldTest = document.getElementById('test-before-save')?.checked;
      if (shouldTest) {
        const testResult = await testConnection(values);
        if (!testResult) {
          Modal.confirm({
            title: '连接测试失败',
            content: '连接测试失败，是否仍要保存配置？',
            okText: '仍要保存',
            cancelText: '取消',
            onOk: () => saveConfig(values)
          });
          return;
        }
      }
      
      await saveConfig(values);
    } catch (error) {
      console.log('表单验证失败:', error);
    }
  };

  useEffect(() => {
    loadConfigs();
  }, []);

  // 表格列定义
  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Space>
          {getStorageTypeConfig(record.type).icon}
          <Text strong>{text}</Text>
          {record.is_active && (
            <Tag color="green" size="small">激活</Tag>
          )}
        </Space>
      )
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type) => {
        const config = getStorageTypeConfig(type);
        return (
          <Tag color="blue">
            {config.label}
          </Tag>
        );
      }
    },
    {
      title: '端点',
      dataIndex: 'endpoint',
      key: 'endpoint',
      ellipsis: true
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag 
          icon={status === 'connected' ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}
          color={status === 'connected' ? 'success' : 'error'}
        >
          {status === 'connected' ? '已连接' : '未连接'}
        </Tag>
      )
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
      width: 200,
      render: (_, record) => (
        <Space>
          <Tooltip title="编辑配置">
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => openConfigModal(record)}
            />
          </Tooltip>
          
          <Tooltip title="测试连接">
            <Button
              size="small"
              icon={<ExperimentOutlined />}
              loading={testingConnection}
              onClick={() => testConnection(record)}
            />
          </Tooltip>
          
          {!record.is_active && (
            <Tooltip title="设为激活">
              <Button
                size="small"
                icon={<CheckCircleOutlined />}
                onClick={() => setActiveConfig(record.id)}
              />
            </Tooltip>
          )}
          
          <Popconfirm
            title="确认删除此配置？"
            description="删除后无法恢复，请确认操作"
            onConfirm={() => deleteConfig(record.id)}
            okText="删除"
            cancelText="取消"
            okType="danger"
          >
            <Tooltip title="删除配置">
              <Button
                size="small"
                danger
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      )
    }
  ];

  return (
    <div>
      <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>
            <DatabaseOutlined style={{ marginRight: '8px', color: '#1890ff' }} />
            对象存储配置
          </Title>
          <Text type="secondary">管理MinIO、S3等对象存储服务的连接配置</Text>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadConfigs}>
            刷新
          </Button>
          <Button 
            type="primary" 
            icon={<PlusOutlined />}
            onClick={() => openConfigModal()}
          >
            添加配置
          </Button>
        </Space>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={configs}
          rowKey="id"
          loading={loading}
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 项，共 ${total} 项`
          }}
        />
      </Card>

      {/* 配置模态框 */}
      <Modal
        title={
          <Space>
            <DatabaseOutlined />
            {editingConfig ? '编辑存储配置' : '添加存储配置'}
          </Space>
        }
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingConfig(null);
          form.resetFields();
        }}
        footer={[
          <Button key="cancel" onClick={() => setModalVisible(false)}>
            取消
          </Button>,
          <Button 
            key="test" 
            icon={<ExperimentOutlined />}
            loading={testingConnection}
            onClick={async () => {
              try {
                const values = await form.validateFields();
                await testConnection(values);
              } catch (error) {
                console.log('表单验证失败:', error);
              }
            }}
          >
            测试连接
          </Button>,
          <Button 
            key="submit" 
            type="primary" 
            icon={<SaveOutlined />}
            onClick={handleSubmit}
          >
            保存
          </Button>
        ]}
        width={800}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            type: 'minio',
            is_active: false,
            ssl_enabled: false,
            timeout: 30
          }}
        >
          <Tabs defaultActiveKey="basic">
            <TabPane tab="基本配置" key="basic">
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    name="name"
                    label="配置名称"
                    rules={[{ required: true, message: '请输入配置名称' }]}
                  >
                    <Input placeholder="例如: 生产环境MinIO" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    name="type"
                    label="存储类型"
                    rules={[{ required: true, message: '请选择存储类型' }]}
                  >
                    <Select>
                      {storageTypes.map(type => (
                        <Option key={type.value} value={type.value}>
                          <Space>
                            {type.icon}
                            {type.label}
                          </Space>
                        </Option>
                      ))}
                    </Select>
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item
                name="endpoint"
                label="服务端点"
                rules={[{ required: true, message: '请输入服务端点' }]}
                extra="例如: localhost:9000 或 s3.amazonaws.com"
              >
                <Input placeholder="服务端点地址" prefix={<LinkOutlined />} />
              </Form.Item>

              <Form.Item noStyle shouldUpdate={(prevValues, currentValues) => prevValues.type !== currentValues.type}>
                {({ getFieldValue }) => {
                  const currentType = getFieldValue('type');
                  const typeConfig = getStorageTypeConfig(currentType);
                  
                  return typeConfig.requiresWebUrl ? (
                    <Form.Item
                      name="web_url"
                      label="Web控制台地址"
                      rules={[{ required: true, message: '请输入Web控制台地址' }]}
                      extra="MinIO控制台访问地址，例如: localhost:9001"
                    >
                      <Input placeholder="Web控制台地址" prefix={<LinkOutlined />} />
                    </Form.Item>
                  ) : null;
                }}
              </Form.Item>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    name="access_key"
                    label="Access Key"
                    rules={[{ required: true, message: '请输入Access Key' }]}
                  >
                    <Input placeholder="访问密钥ID" prefix={<KeyOutlined />} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    name="secret_key"
                    label="Secret Key"
                    rules={[{ required: true, message: '请输入Secret Key' }]}
                  >
                    <Input.Password placeholder="访问密钥Secret" prefix={<KeyOutlined />} />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item
                name="description"
                label="描述"
              >
                <TextArea 
                  rows={3} 
                  placeholder="配置描述信息（可选）" 
                />
              </Form.Item>
            </TabPane>

            <TabPane tab="高级配置" key="advanced">
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    name="region"
                    label="区域"
                  >
                    <Input placeholder="存储区域（可选）" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    name="timeout"
                    label="超时时间（秒）"
                  >
                    <InputNumber 
                      min={1} 
                      max={300} 
                      style={{ width: '100%' }}
                      placeholder="请求超时时间"
                    />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item
                    name="ssl_enabled"
                    label="启用SSL"
                    valuePropName="checked"
                  >
                    <Switch checkedChildren="启用" unCheckedChildren="禁用" />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item
                    name="is_active"
                    label="设为激活配置"
                    valuePropName="checked"
                    extra="激活配置将作为默认存储"
                  >
                    <Switch checkedChildren="是" unCheckedChildren="否" />
                  </Form.Item>
                </Col>
              </Row>

              <Divider />
              
              <div>
                <input 
                  type="checkbox" 
                  id="test-before-save" 
                  defaultChecked 
                  style={{ marginRight: '8px' }} 
                />
                <label htmlFor="test-before-save">
                  保存前测试连接
                </label>
              </div>
            </TabPane>
          </Tabs>
        </Form>
      </Modal>
    </div>
  );
};

export default ObjectStorageConfigPage;