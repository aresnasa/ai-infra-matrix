import React, { useState, useEffect } from 'react';
import { 
  Card, 
  Button, 
  Form, 
  Input, 
  Select, 
  Switch, 
  Table, 
  Modal, 
  message,
  Tabs,
  Row,
  Col,
  Statistic,
  Divider,
  Space,
  Tag,
  Tooltip,
  Popconfirm
} from 'antd';
import { 
  PlusOutlined, 
  EditOutlined, 
  DeleteOutlined,
  RobotOutlined,
  ApiOutlined,
  MessageOutlined,
  BarChartOutlined,
  SettingOutlined,
  KeyOutlined
} from '@ant-design/icons';
import { aiAPI } from '../services/api';

const { Option } = Select;
const { TabPane } = Tabs;
const { TextArea } = Input;

const AIAssistantManagement = () => {
  const [configs, setConfigs] = useState([]);
  const [conversations, setConversations] = useState([]);
  const [usage, setUsage] = useState([]);
  const [loading, setLoading] = useState(false);
  const [configModalVisible, setConfigModalVisible] = useState(false);
  const [editingConfig, setEditingConfig] = useState(null);
  const [configForm] = Form.useForm();

  // AI提供商选项
  const aiProviders = [
    { value: 'openai', label: 'OpenAI' },
    { value: 'claude', label: 'Claude (Anthropic)' },
    { value: 'mcp', label: 'Model Context Protocol' },
    { value: 'custom', label: '自定义' }
  ];

  // 加载数据
  useEffect(() => {
    loadConfigs();
    loadConversations();
    loadUsage();
  }, []);

  const loadConfigs = async () => {
    try {
      setLoading(true);
      const response = await aiAPI.getConfigs();
      console.log('管理页面获取配置响应:', response.data);
      const configData = response.data.data || response.data || [];
      setConfigs(configData);
    } catch (error) {
      console.error('加载AI配置失败:', error);
      message.error('加载AI配置失败');
    } finally {
      setLoading(false);
    }
  };

  const loadConversations = async () => {
    try {
      const response = await aiAPI.getConversations();
      const conversationData = response.data.data || response.data || [];
      setConversations(conversationData);
    } catch (error) {
      message.error('加载对话记录失败');
    }
  };

  const loadUsage = async () => {
    try {
      const response = await aiAPI.getUsage();
      const usageData = response.data.data || response.data || [];
      setUsage(usageData);
    } catch (error) {
      message.error('加载使用统计失败');
    }
  };

  // 配置管理
  const handleCreateConfig = () => {
    setEditingConfig(null);
    configForm.resetFields();
    setConfigModalVisible(true);
  };

  const handleEditConfig = (config) => {
    setEditingConfig(config);
    configForm.setFieldsValue({
      ...config,
      api_key: '********' // 隐藏API密钥
    });
    setConfigModalVisible(true);
  };

  const handleDeleteConfig = async (id) => {
    try {
      await aiAPI.deleteConfig(id);
      message.success('删除成功');
      loadConfigs();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleConfigSubmit = async (values) => {
    try {
      setLoading(true);
      if (editingConfig) {
        // 如果API密钥没有改变，不发送
        if (values.api_key === '********') {
          delete values.api_key;
        }
        await aiAPI.updateConfig(editingConfig.id, values);
        message.success('更新成功');
      } else {
        await aiAPI.createConfig(values);
        message.success('创建成功');
      }
      setConfigModalVisible(false);
      loadConfigs();
    } catch (error) {
      message.error(editingConfig ? '更新失败' : '创建失败');
    } finally {
      setLoading(false);
    }
  };

  const handleToggleConfig = async (id, enabled) => {
    try {
      await aiAPI.updateConfig(id, { enabled });
      message.success(enabled ? '已启用' : '已禁用');
      loadConfigs();
    } catch (error) {
      message.error('操作失败');
    }
  };

  // 清理对话历史
  const handleClearConversations = async () => {
    try {
      await aiAPI.clearConversations();
      message.success('清理成功');
      loadConversations();
    } catch (error) {
      message.error('清理失败');
    }
  };

  // 配置表格列
  const configColumns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '提供商',
      dataIndex: 'provider',
      key: 'provider',
      render: (provider) => {
        const providerInfo = aiProviders.find(p => p.value === provider);
        return <Tag color="blue">{providerInfo?.label || provider}</Tag>;
      }
    },
    {
      title: '模型',
      dataIndex: 'model',
      key: 'model',
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled, record) => (
        <Switch
          checked={enabled}
          onChange={(checked) => handleToggleConfig(record.id, checked)}
          checkedChildren="启用"
          unCheckedChildren="禁用"
        />
      )
    },
    {
      title: '默认',
      dataIndex: 'is_default',
      key: 'is_default',
      render: (isDefault) => (
        isDefault ? <Tag color="green">默认</Tag> : null
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
      key: 'action',
      render: (_, record) => (
        <Space>
          <Tooltip title="编辑">
            <Button 
              type="text" 
              icon={<EditOutlined />} 
              onClick={() => handleEditConfig(record)}
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个配置吗？"
            onConfirm={() => handleDeleteConfig(record.id)}
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
        </Space>
      ),
    },
  ];

  // 对话表格列
  const conversationColumns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '用户',
      dataIndex: 'user_id',
      key: 'user_id',
      width: 100,
    },
    {
      title: '消息数',
      dataIndex: 'message_count',
      key: 'message_count',
      width: 100,
    },
    {
      title: '最后消息',
      dataIndex: 'last_message_at',
      key: 'last_message_at',
      render: (time) => time ? new Date(time).toLocaleString() : '-'
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => new Date(time).toLocaleString()
    },
  ];

  // 使用统计表格列
  const usageColumns = [
    {
      title: '用户ID',
      dataIndex: 'user_id',
      key: 'user_id',
    },
    {
      title: '配置',
      dataIndex: 'config_name',
      key: 'config_name',
    },
    {
      title: '请求数',
      dataIndex: 'request_count',
      key: 'request_count',
    },
    {
      title: 'Token使用',
      dataIndex: 'token_used',
      key: 'token_used',
    },
    {
      title: '总费用',
      dataIndex: 'total_cost',
      key: 'total_cost',
      render: (cost) => `$${(cost || 0).toFixed(4)}`
    },
    {
      title: '日期',
      dataIndex: 'date',
      key: 'date',
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ marginBottom: '24px' }}>
        <h1>
          <RobotOutlined style={{ marginRight: '8px' }} />
          AI助手管理
        </h1>
      </div>

      <Tabs defaultActiveKey="configs">
        <TabPane 
          tab={
            <span>
              <SettingOutlined />
              AI配置
            </span>
          } 
          key="configs"
        >
          <Card
            title="AI提供商配置"
            extra={
              <Button 
                type="primary" 
                icon={<PlusOutlined />}
                onClick={handleCreateConfig}
              >
                添加配置
              </Button>
            }
          >
            <Table
              columns={configColumns}
              dataSource={configs}
              rowKey="id"
              loading={loading}
              pagination={{ pageSize: 10 }}
            />
          </Card>
        </TabPane>

        <TabPane 
          tab={
            <span>
              <MessageOutlined />
              对话管理
            </span>
          } 
          key="conversations"
        >
          <Card
            title="对话记录"
            extra={
              <Popconfirm
                title="确定要清理所有对话记录吗？"
                onConfirm={handleClearConversations}
                okText="确定"
                cancelText="取消"
              >
                <Button danger>
                  清理对话记录
                </Button>
              </Popconfirm>
            }
          >
            <Table
              columns={conversationColumns}
              dataSource={conversations}
              rowKey="id"
              pagination={{ pageSize: 10 }}
            />
          </Card>
        </TabPane>

        <TabPane 
          tab={
            <span>
              <BarChartOutlined />
              使用统计
            </span>
          } 
          key="usage"
        >
          <Card title="使用统计">
            <Row gutter={16} style={{ marginBottom: '24px' }}>
              <Col span={6}>
                <Statistic
                  title="总请求数"
                  value={usage.reduce((sum, item) => sum + (item.request_count || 0), 0)}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="总Token使用"
                  value={usage.reduce((sum, item) => sum + (item.token_used || 0), 0)}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="总费用"
                  value={usage.reduce((sum, item) => sum + (item.total_cost || 0), 0)}
                  precision={4}
                  prefix="$"
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="活跃用户"
                  value={new Set(usage.map(item => item.user_id)).size}
                />
              </Col>
            </Row>
            
            <Divider />
            
            <Table
              columns={usageColumns}
              dataSource={usage}
              rowKey={(record) => `${record.user_id}-${record.date}`}
              pagination={{ pageSize: 10 }}
            />
          </Card>
        </TabPane>
      </Tabs>

      {/* 配置编辑模态框 */}
      <Modal
        title={editingConfig ? '编辑AI配置' : '添加AI配置'}
        open={configModalVisible}
        onCancel={() => setConfigModalVisible(false)}
        onOk={() => configForm.submit()}
        confirmLoading={loading}
        width={600}
      >
        <Form
          form={configForm}
          layout="vertical"
          onFinish={handleConfigSubmit}
        >
          <Form.Item
            name="name"
            label="配置名称"
            rules={[{ required: true, message: '请输入配置名称' }]}
          >
            <Input placeholder="例如：GPT-4配置" />
          </Form.Item>

          <Form.Item
            name="provider"
            label="AI提供商"
            rules={[{ required: true, message: '请选择AI提供商' }]}
          >
            <Select placeholder="选择AI提供商">
              {aiProviders.map(provider => (
                <Option key={provider.value} value={provider.value}>
                  {provider.label}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="model"
            label="模型名称"
            rules={[{ required: true, message: '请输入模型名称' }]}
          >
            <Input placeholder="例如：gpt-4、claude-3-opus" />
          </Form.Item>

          <Form.Item
            name="api_key"
            label="API密钥"
            rules={[{ required: !editingConfig, message: '请输入API密钥' }]}
          >
            <Input.Password 
              placeholder={editingConfig ? "留空表示不修改" : "请输入API密钥"}
              prefix={<KeyOutlined />}
            />
          </Form.Item>

          <Form.Item
            name="api_base"
            label="API基础URL"
          >
            <Input placeholder="可选，自定义API基础URL" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="max_tokens"
                label="最大Token数"
              >
                <Input type="number" placeholder="例如：4096" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="temperature"
                label="温度参数"
              >
                <Input type="number" step="0.1" placeholder="0.0-2.0" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="system_prompt"
            label="系统提示词"
          >
            <TextArea 
              rows={3}
              placeholder="可选，自定义系统提示词"
            />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="enabled"
                label="启用状态"
                valuePropName="checked"
              >
                <Switch checkedChildren="启用" unCheckedChildren="禁用" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="is_default"
                label="设为默认"
                valuePropName="checked"
              >
                <Switch checkedChildren="是" unCheckedChildren="否" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  );
};

export default AIAssistantManagement;