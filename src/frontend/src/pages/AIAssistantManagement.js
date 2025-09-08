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
  Popconfirm,
  Spin,
  Alert
} from 'antd';
import { 
  PlusOutlined, 
  EditOutlined, 
  DeleteOutlined,
  CopyOutlined,
  RobotOutlined,
  ApiOutlined,
  MessageOutlined,
  BarChartOutlined,
  KeyOutlined
} from '@ant-design/icons';
import { aiAPI } from '../services/api';
import { CustomMenuIcons } from '../components/CustomIcons';

const { Option } = Select;
const { TabPane } = Tabs;
const { TextArea } = Input;

const AIAssistantManagement = () => {
  console.log('=== AIAssistantManagement 组件开始渲染 ===');
  
  // 状态初始化 - 确保所有数组状态都有默认值
  const [configs, setConfigs] = useState([]);
  const [conversations, setConversations] = useState([]);
  const [usage, setUsage] = useState([]);
  const [loading, setLoading] = useState(false);
  const [configModalVisible, setConfigModalVisible] = useState(false);
  const [editingConfig, setEditingConfig] = useState(null);
  const [configForm] = Form.useForm();
  const [error, setError] = useState(null);
  
  console.log('AIAssistantManagement 状态初始化完成, configs:', configs, 'conversations:', conversations, 'usage:', usage);

  // AI提供商选项 - 扩展支持更多提供商
  const aiProviders = [
    { value: 'openai', label: 'OpenAI' },
    { value: 'claude', label: 'Claude (Anthropic)' },
    { value: 'deepseek', label: 'DeepSeek' },
    { value: 'glm', label: '智谱GLM' },
    { value: 'qwen', label: '通义千问' },
    { value: 'mcp', label: 'Model Context Protocol' },
    { value: 'custom', label: '自定义' }
  ];

  // 模型类型选项
  const modelTypes = [
    { value: 'chat', label: '对话模型' },
    { value: 'completion', label: '补全模型' },
    { value: 'embedding', label: '嵌入模型' },
    { value: 'image', label: '图像生成' },
    { value: 'audio', label: '音频处理' },
    { value: 'custom', label: '自定义' }
  ];

  // 机器人分类选项
  const botCategories = [
    { value: 'general', label: '通用对话' },
    { value: 'coding', label: '代码生成' },
    { value: 'writing', label: '写作助手' },
    { value: 'analysis', label: '数据分析' },
    { value: 'translation', label: '翻译助手' },
    { value: 'research', label: '研究助手' },
    { value: 'education', label: '教育助手' },
    { value: 'business', label: '商业助手' },
    { value: 'creative', label: '创意助手' },
    { value: 'custom', label: '自定义' }
  ];

  // 安全的数组检查函数
  const ensureArray = (data, fallback = []) => {
    if (Array.isArray(data)) {
      return data;
    }
    console.warn('Data is not an array:', data);
    return fallback;
  };

  // 安全的统计计算函数
  const calculateStats = () => {
    const safeUsage = ensureArray(usage);
    console.log('计算统计，使用数据:', safeUsage);
    
    try {
      return {
        totalRequests: safeUsage.reduce((sum, item) => sum + (Number(item?.request_count) || 0), 0),
        totalTokens: safeUsage.reduce((sum, item) => sum + (Number(item?.token_used) || 0), 0),
        totalCost: safeUsage.reduce((sum, item) => sum + (Number(item?.total_cost) || 0), 0),
        activeUsers: new Set(safeUsage.map(item => item?.user_id).filter(Boolean)).size
      };
    } catch (error) {
      console.error('统计计算错误:', error);
      return {
        totalRequests: 0,
        totalTokens: 0,
        totalCost: 0,
        activeUsers: 0
      };
    }
  };

  // 加载数据
  useEffect(() => {
    console.log('=== useEffect 执行，开始加载数据 ===');
    const loadData = async () => {
      try {
        await Promise.all([
          loadConfigs(),
          loadConversations(),
          loadUsage()
        ]);
      } catch (error) {
        console.error('数据加载失败:', error);
        setError(error.message);
      }
    };
    loadData();
  }, []);

  const loadConfigs = async () => {
    console.log('开始加载 configs...');
    try {
      setLoading(true);
      const response = await aiAPI.getConfigs();
      console.log('配置响应:', response);
      
      if (!response || !response.data) {
        console.warn('Invalid response structure:', response);
        setConfigs([]);
        return;
      }
      
      const configData = response.data.data || response.data || [];
      const safeConfigs = ensureArray(configData);
      console.log('安全的配置数据:', safeConfigs);
      setConfigs(safeConfigs);
    } catch (error) {
      console.error('加载AI配置失败:', error);
      message.error('加载AI配置失败: ' + error.message);
      setConfigs([]);
    } finally {
      setLoading(false);
    }
  };

    const loadConversations = async () => {
    console.log('开始加载 conversations...');
    try {
      const response = await aiAPI.getConversations();
      console.log('对话响应:', response);
      
      if (!response || !response.data) {
        console.warn('Invalid response structure:', response);
        setConversations([]);
        return;
      }
      
      const conversationData = response.data.data || response.data || [];
      const safeConversations = ensureArray(conversationData);
      console.log('安全的对话数据:', safeConversations);
      setConversations(safeConversations);
    } catch (error) {
      console.error('加载对话记录失败:', error);
      message.error('加载对话记录失败: ' + error.message);
      setConversations([]);
    }
  };

  const loadUsage = async () => {
    console.log('开始加载 usage...');
    try {
      const response = await aiAPI.getUsage();
      console.log('使用统计响应:', response);
      
      if (!response || !response.data) {
        console.warn('Invalid response structure:', response);
        setUsage([]);
        return;
      }
      
      const usageData = response.data.data || response.data || [];
      const safeUsage = ensureArray(usageData);
      console.log('安全的使用统计数据:', safeUsage);
      setUsage(safeUsage);
    } catch (error) {
      console.error('加载使用统计失败:', error);
      message.error('加载使用统计失败: ' + error.message);
      setUsage([]);
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

  const handleCloneConfig = async (config) => {
    try {
      const clonedConfig = {
        ...config,
        name: `${config.name} (副本)`,
        is_default: false,
        enabled: false
      };
      delete clonedConfig.id;
      delete clonedConfig.created_at;
      delete clonedConfig.updated_at;
      
      await aiAPI.createConfig(clonedConfig);
      message.success('克隆成功');
      loadConfigs();
    } catch (error) {
      message.error('克隆失败');
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
      width: 120,
    },
    {
      title: '提供商',
      dataIndex: 'provider',
      key: 'provider',
      width: 100,
      render: (provider) => {
        const providerInfo = aiProviders.find(p => p.value === provider);
        return <Tag color="blue">{providerInfo?.label || provider}</Tag>;
      }
    },
    {
      title: '模型类型',
      dataIndex: 'model_type',
      key: 'model_type',
      width: 100,
      render: (modelType) => {
        const typeMap = {
          'chat': '对话',
          'completion': '补全',
          'embedding': '嵌入',
          'image': '图像',
          'audio': '音频',
          'custom': '自定义'
        };
        return <Tag color="purple">{typeMap[modelType] || modelType}</Tag>;
      }
    },
    {
      title: '模型',
      dataIndex: 'model',
      key: 'model',
      width: 120,
    },
    {
      title: '类别',
      dataIndex: 'category',
      key: 'category',
      width: 100,
      render: (category) => category ? <Tag color="orange">{category}</Tag> : '-'
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 150,
      ellipsis: true,
      render: (desc) => desc || '-'
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
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
      width: 80,
      render: (isDefault) => (
        isDefault ? <Tag color="green">默认</Tag> : null
      )
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (time) => new Date(time).toLocaleString()
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_, record) => (
        <Space>
          <Tooltip title="编辑">
            <Button 
              type="text" 
              icon={<EditOutlined />} 
              onClick={() => handleEditConfig(record)}
            />
          </Tooltip>
          <Tooltip title="克隆">
            <Button 
              type="text" 
              icon={<CopyOutlined />} 
              onClick={() => handleCloneConfig(record)}
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

  // 获取统计数据
  const stats = calculateStats();

  if (error) {
    return (
      <div style={{ padding: '24px' }}>
        <Alert
          message="加载失败"
          description={error}
          type="error"
          showIcon
          action={
            <Button onClick={() => window.location.reload()}>
              重新加载
            </Button>
          }
        />
      </div>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      <Spin spinning={loading}>
        <Card 
          title={
            <span>
              <RobotOutlined style={{ marginRight: '8px' }} />
              AI助手管理
            </span>
          }
        >
          <Tabs defaultActiveKey="configs">
        <TabPane 
          tab={
            <span>
              <CustomMenuIcons.Menu size={16} />
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
              dataSource={ensureArray(configs)}
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
              dataSource={ensureArray(conversations)}
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
                  value={stats.totalRequests}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="总Token使用"
                  value={stats.totalTokens}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="总费用"
                  value={stats.totalCost}
                  precision={4}
                  prefix="$"
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="活跃用户"
                  value={stats.activeUsers}
                />
              </Col>
            </Row>
            
            <Divider />
            
            <Table
              columns={usageColumns}
              dataSource={Array.isArray(usage) ? usage : []}
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
            name="model_type"
            label="模型类型"
            rules={[{ required: true, message: '请选择模型类型' }]}
          >
            <Select placeholder="选择模型类型">
              {modelTypes.map(type => (
                <Option key={type.value} value={type.value}>
                  {type.label}
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
            name="category"
            label="机器人分类"
          >
            <Select placeholder="选择机器人分类">
              {botCategories.map(category => (
                <Option key={category.value} value={category.value}>
                  {category.label}
                </Option>
              ))}
            </Select>
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
            name="api_secret"
            label="API密钥(备用)"
          >
            <Input.Password
              placeholder="可选，某些提供商需要额外的密钥"
              prefix={<KeyOutlined />}
            />
          </Form.Item>

          <Form.Item
            name="api_endpoint"
            label="API端点"
          >
            <Input placeholder="API基础URL，例如：https://api.openai.com/v1" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="max_tokens"
                label="最大Token数"
              >
                <Input type="number" placeholder="例如：4096" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="temperature"
                label="温度参数"
              >
                <Input type="number" step="0.1" min="0" max="2" placeholder="0.0-2.0" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="top_p"
                label="Top P"
              >
                <Input type="number" step="0.1" min="0" max="1" placeholder="0.0-1.0" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="frequency_penalty"
                label="频率惩罚"
              >
                <Input type="number" step="0.1" min="-2" max="2" placeholder="-2.0-2.0" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="presence_penalty"
                label="存在惩罚"
              >
                <Input type="number" step="0.1" min="-2" max="2" placeholder="-2.0-2.0" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="rate_limit_per_hour"
                label="每小时限额"
              >
                <Input type="number" placeholder="例如：100" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="rate_limit_per_day"
                label="每日限额"
              >
                <Input type="number" placeholder="例如：1000" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="timeout_seconds"
                label="超时时间(秒)"
              >
                <Input type="number" placeholder="例如：60" />
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

          <Form.Item
            name="description"
            label="机器人描述"
          >
            <TextArea
              rows={2}
              placeholder="描述这个机器人的用途和特点"
            />
          </Form.Item>

          <Form.Item
            name="icon_url"
            label="图标URL"
          >
            <Input placeholder="可选，自定义机器人图标URL" />
          </Form.Item>

          <Form.Item
            name="tags"
            label="标签"
          >
            <Select
              mode="tags"
              placeholder="添加标签来分类机器人"
              tokenSeparators={[',']}
            />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="enabled"
                label="启用状态"
                valuePropName="checked"
              >
                <Switch checkedChildren="启用" unCheckedChildren="禁用" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="is_default"
                label="设为默认"
                valuePropName="checked"
              >
                <Switch checkedChildren="是" unCheckedChildren="否" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="retry_attempts"
                label="重试次数"
              >
                <Input type="number" min="0" max="10" placeholder="0-10" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
        </Card>
      </Spin>
    </div>
  );
};

export default AIAssistantManagement;