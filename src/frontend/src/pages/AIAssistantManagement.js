import React, { useState, useEffect, useMemo } from 'react';
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
  RobotOutlined,
  MessageOutlined,
  BarChartOutlined,
  SettingOutlined
} from '@ant-design/icons';
import { aiAPI } from '../services/api';

const { Option } = Select;
const { TextArea } = Input;

const AIAssistantManagement = () => {
  const [configs, setConfigs] = useState([]);
  const [conversations, setConversations] = useState([]);
  const [usage, setUsage] = useState([]);
  const [loading, setLoading] = useState(false);
  const [configModalVisible, setConfigModalVisible] = useState(false);
  const [editingConfig, setEditingConfig] = useState(null);
  const [error, setError] = useState(null);
  
  const [configForm] = Form.useForm();

  // 更强的数组确保函数
  const ensureArray = (data) => {
    try {
      // 如果已经是数组，直接返回
      if (Array.isArray(data)) {
        return data;
      }
      
      // 处理 axios 响应格式 { data: [...] }
      if (data && typeof data === 'object' && Array.isArray(data.data)) {
        return data.data;
      }
      
      // 处理单个对象的情况
      if (data && typeof data === 'object' && !Array.isArray(data)) {
        return [data];
      }
      
      // 处理 null, undefined, 或其他类型
      console.warn('ensureArray: 数据类型异常，返回空数组:', typeof data, data);
      return [];
    } catch (error) {
      console.error('ensureArray error:', error);
      return [];
    }
  };

  const calculateStats = (usageData) => {
    try {
      // 双重确保 usageData 是数组
      const safeUsageData = ensureArray(usageData);
      
      console.log('calculateStats input:', usageData, 'processed:', safeUsageData);
      
      if (safeUsageData.length === 0) {
        return {
          totalRequests: 0,
          totalTokens: 0,
          totalCost: 0,
          activeUsers: 0
        };
      }

      const stats = safeUsageData.reduce((acc, item) => {
        // 安全地获取数值，确保不是 undefined 或 null
        const requests = Number(item?.requests) || 0;
        const tokens = Number(item?.tokens) || 0;
        const cost = Number(item?.cost) || 0;
        
        acc.totalRequests += requests;
        acc.totalTokens += tokens;
        acc.totalCost += cost;
        
        return acc;
      }, { totalRequests: 0, totalTokens: 0, totalCost: 0 });

      // 计算唯一用户数 - 更安全的方式
      const uniqueUsers = safeUsageData
        .map(item => item?.user_id)
        .filter(userId => userId !== null && userId !== undefined && userId !== '');
      stats.activeUsers = new Set(uniqueUsers).size;
      
      console.log('calculateStats result:', stats);
      return stats;
    } catch (error) {
      console.error('计算统计数据失败:', error, 'input data:', usageData);
      return {
        totalRequests: 0,
        totalTokens: 0,
        totalCost: 0,
        activeUsers: 0
      };
    }
  };

  // 强制数组检查 - 在计算和渲染时确保数据类型安全
  const safeConfigs = useMemo(() => ensureArray(configs), [configs]);
  const safeConversations = useMemo(() => ensureArray(conversations), [conversations]);
  const safeUsage = useMemo(() => ensureArray(usage), [usage]);

  // 使用 useMemo 来优化 stats 计算，避免每次渲染都重新计算
  const stats = useMemo(() => calculateStats(safeUsage), [safeUsage]);

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);
      
      console.log('开始加载数据...');
      
      const [configsRes, conversationsRes, usageRes] = await Promise.all([
        aiAPI.getConfigs().catch(err => {
          console.warn('加载配置失败:', err);
          return { data: [] };
        }),
        aiAPI.getConversations().catch(err => {
          console.warn('加载对话失败:', err);
          return { data: [] };
        }),
        aiAPI.getUsage().catch(err => {
          console.warn('加载使用统计失败:', err);
          return { data: [] };
        })
      ]);

      console.log('API响应:', { configsRes, conversationsRes, usageRes });

      // 确保数据是数组格式
      const configsArray = ensureArray(configsRes);
      const conversationsArray = ensureArray(conversationsRes);
      const usageArray = ensureArray(usageRes);

      console.log('处理后的数据:', { configsArray, conversationsArray, usageArray });

      setConfigs(configsArray);
      setConversations(conversationsArray);
      setUsage(usageArray);
      
    } catch (error) {
      console.error('加载数据失败:', error);
      const errorMsg = error.response?.data?.message || error.message || '加载数据失败';
      setError(errorMsg);
      message.error(errorMsg);
      
      // 设置默认空数组防止map错误
      setConfigs([]);
      setConversations([]);
      setUsage([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    
    // 监听来自AI助手浮动窗口的配置更新事件
    const handleConfigUpdate = (e) => {
      console.log('🔄 检测到AI配置更新，重新加载数据...');
      loadData();
    };
    
    // 监听storage事件 (跨组件通信)
    window.addEventListener('storage', handleConfigUpdate);
    
    // 监听自定义事件 (同页面组件通信)
    window.addEventListener('ai-config-updated', handleConfigUpdate);
    
    return () => {
      window.removeEventListener('storage', handleConfigUpdate);
      window.removeEventListener('ai-config-updated', handleConfigUpdate);
    };
  }, []);

  const handleCreateConfig = () => {
    setEditingConfig(null);
    configForm.resetFields();
    setConfigModalVisible(true);
  };

  const handleEditConfig = (config) => {
    setEditingConfig(config);
    configForm.setFieldsValue(config);
    setConfigModalVisible(true);
  };

  const handleDeleteConfig = async (configId) => {
    try {
      setLoading(true);
      await aiAPI.deleteConfig(configId);
      message.success('配置删除成功');
      await loadData();
    } catch (error) {
      message.error('删除配置失败');
    } finally {
      setLoading(false);
    }
  };

  const handleConfigSubmit = async (values) => {
    try {
      if (editingConfig) {
        await aiAPI.updateConfig(editingConfig.id, values);
        message.success('配置更新成功');
      } else {
        await aiAPI.createConfig(values);
        message.success('配置创建成功');
      }
      setConfigModalVisible(false);
      await loadData();
    } catch (error) {
      message.error('保存配置失败');
    }
  };

  // 安全渲染函数 - 捕获任何渲染错误
  const renderSafely = (renderFn, fallback = null) => {
    try {
      return renderFn();
    } catch (error) {
      console.error('Render error caught:', error);
      return fallback || <Alert message="组件渲染错误" type="error" />;
    }
  };

  return (
    <div style={{ padding: '24px' }}>
      {error && (
        <Alert
          message="加载错误"
          description={error}
          type="error"
          showIcon
          style={{ marginBottom: '16px' }}
        />
      )}
      
      <Spin spinning={loading}>
        <Card 
          title={
            <span>
              <RobotOutlined style={{ marginRight: '8px' }} />
              AI助手管理
            </span>
          }
        >
          {/* 添加调试信息 */}
          {process.env.NODE_ENV === 'development' && (
            <div style={{ marginBottom: '16px', fontSize: '12px', color: '#666' }}>
              Debug: configs={Array.isArray(safeConfigs) ? safeConfigs.length : 'not array'}, 
              conversations={Array.isArray(safeConversations) ? safeConversations.length : 'not array'}, 
              usage={Array.isArray(safeUsage) ? safeUsage.length : 'not array'}
            </div>
          )}
          
          {renderSafely(() => (
            <Tabs 
              defaultActiveKey="configs" 
              size="large"
              items={[
                {
                  key: 'configs',
                  label: (
                    <span>
                      <SettingOutlined style={{ marginRight: '4px' }} />
                      AI配置
                    </span>
                  ),
                  children: renderSafely(() => (
                    <Card 
                      title="AI提供商配置"
                      extra={
                        <Button 
                          type="primary" 
                          icon={<PlusOutlined />}
                          onClick={handleCreateConfig}
                        >
                          新增配置
                        </Button>
                      }
                    >
                      <Table
                        columns={[
                          {
                            title: '配置名称',
                            dataIndex: 'name',
                            key: 'name'
                          },
                          {
                            title: '提供商',
                            dataIndex: 'provider',
                            key: 'provider',
                            render: (provider) => (
                              <Tag color="blue">{provider}</Tag>
                            )
                          },
                          {
                            title: '模型',
                            dataIndex: 'model',
                            key: 'model'
                          },
                          {
                            title: '状态',
                            dataIndex: 'is_enabled',
                            key: 'is_enabled',
                            render: (enabled) => (
                              <Tag color={enabled ? 'green' : 'red'}>
                                {enabled ? '启用' : '禁用'}
                              </Tag>
                            )
                          },
                          {
                            title: '操作',
                            key: 'actions',
                            render: (_, record) => (
                              <Space>
                                <Tooltip title="编辑配置">
                                  <Button 
                                    type="link" 
                                    icon={<EditOutlined />}
                                    onClick={() => handleEditConfig(record)}
                                  />
                                </Tooltip>
                                <Popconfirm
                                  title="确定删除此配置吗？"
                                  onConfirm={() => handleDeleteConfig(record.id)}
                                >
                                  <Tooltip title="删除配置">
                                    <Button 
                                      type="link" 
                                      danger 
                                      icon={<DeleteOutlined />}
                                    />
                                  </Tooltip>
                                </Popconfirm>
                              </Space>
                            )
                          }
                        ]}
                        dataSource={safeConfigs}
                        rowKey={(record) => record?.id || Math.random()}
                        loading={loading}
                        pagination={{ pageSize: 10 }}
                      />
                    </Card>
                  ))
                },
                {
                  key: 'conversations',
                  label: (
                    <span>
                      <MessageOutlined style={{ marginRight: '4px' }} />
                      对话记录
                    </span>
                  ),
                  children: renderSafely(() => (
                    <Card title="对话记录">
                      <Table
                        columns={[
                          {
                            title: '对话标题',
                            dataIndex: 'title',
                            key: 'title'
                          },
                          {
                            title: '用户ID',
                            dataIndex: 'user_id',
                            key: 'user_id'
                          },
                          {
                            title: '创建时间',
                            dataIndex: 'created_at',
                            key: 'created_at',
                            render: (time) => time ? new Date(time).toLocaleString() : ''
                          }
                        ]}
                        dataSource={safeConversations}
                        rowKey={(record) => record?.id || Math.random()}
                        loading={loading}
                        pagination={{ pageSize: 10 }}
                      />
                    </Card>
                  ))
                },
                {
                  key: 'usage',
                  label: (
                    <span>
                      <BarChartOutlined style={{ marginRight: '4px' }} />
                      使用统计
                    </span>
                  ),
                  children: renderSafely(() => (
                    <Card title="使用统计">
                      <Row gutter={16} style={{ marginBottom: 16 }}>
                        <Col span={6}>
                          <Statistic title="总请求数" value={stats?.totalRequests || 0} />
                        </Col>
                        <Col span={6}>
                          <Statistic title="总Token数" value={stats?.totalTokens || 0} />
                        </Col>
                        <Col span={6}>
                          <Statistic 
                            title="总费用" 
                            value={stats?.totalCost || 0} 
                            precision={4} 
                            prefix="$" 
                          />
                        </Col>
                        <Col span={6}>
                          <Statistic title="活跃用户" value={stats?.activeUsers || 0} />
                        </Col>
                      </Row>
                      <Table
                        columns={[
                          {
                            title: '日期',
                            dataIndex: 'date',
                            key: 'date'
                          },
                          {
                            title: '用户',
                            dataIndex: 'user_id',
                            key: 'user_id'
                          },
                          {
                            title: '请求数',
                            dataIndex: 'requests',
                            key: 'requests'
                          },
                          {
                            title: 'Token数',
                            dataIndex: 'tokens',
                            key: 'tokens'
                          }
                        ]}
                        dataSource={safeUsage}
                        rowKey={(record, index) => record?.id || index}
                        loading={loading}
                        pagination={{ pageSize: 10 }}
                      />
                    </Card>
                  ))
                }
              ]}
            />
          ))}
        </Card>
      </Spin>

      {/* 开发调试信息 */}
      {process.env.NODE_ENV === 'development' && (
        <Card 
          title="调试信息" 
          style={{ marginTop: 16 }}
          size="small"
        >
          <div style={{ fontSize: '12px', fontFamily: 'monospace', background: '#f5f5f5', padding: '8px', borderRadius: '4px' }}>
            <div><strong>原始数据类型:</strong></div>
            <div>configs: {typeof configs} ({Array.isArray(configs) ? 'array' : 'not array'})</div>
            <div>conversations: {typeof conversations} ({Array.isArray(conversations) ? 'array' : 'not array'})</div>
            <div>usage: {typeof usage} ({Array.isArray(usage) ? 'array' : 'not array'})</div>
            <div style={{ marginTop: 8 }}><strong>安全数据长度:</strong></div>
            <div>safeConfigs: {safeConfigs?.length || 0}</div>
            <div>safeConversations: {safeConversations?.length || 0}</div>
            <div>safeUsage: {safeUsage?.length || 0}</div>
            <div style={{ marginTop: 8 }}><strong>统计信息:</strong></div>
            <pre style={{ margin: 0, fontSize: '10px' }}>{JSON.stringify(stats, null, 2)}</pre>
          </div>
        </Card>
      )}

      <Modal
        title={editingConfig ? '编辑AI配置' : '新增AI配置'}
        open={configModalVisible}
        onCancel={() => setConfigModalVisible(false)}
        onOk={() => configForm.submit()}
        width={800}
      >
        <Form
          form={configForm}
          layout="vertical"
          onFinish={handleConfigSubmit}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="配置名称"
                rules={[{ required: true, message: '请输入配置名称' }]}
              >
                <Input placeholder="输入配置名称" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="provider"
                label="AI提供商"
                rules={[{ required: true, message: '请选择AI提供商' }]}
              >
                <Select placeholder="选择AI提供商">
                  <Option value="openai">OpenAI</Option>
                  <Option value="claude">Claude</Option>
                  <Option value="deepseek">DeepSeek</Option>
                  <Option value="glm">GLM</Option>
                  <Option value="qwen">Qwen</Option>
                  <Option value="local">本地模型</Option>
                  <Option value="mcp">MCP协议</Option>
                  <Option value="custom">自定义</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="model"
                label="模型名称"
                rules={[{ required: true, message: '请输入模型名称' }]}
              >
                <Input placeholder="如 gpt-3.5-turbo" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="api_endpoint"
                label="API端点"
              >
                <Input placeholder="如 https://api.openai.com/v1" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="api_key"
            label="API密钥"
          >
            <Input.Password placeholder="输入API密钥" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="max_tokens"
                label="最大Token数"
              >
                <Input type="number" placeholder="4096" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="temperature"
                label="温度"
              >
                <Input type="number" step="0.1" placeholder="0.7" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="top_p"
                label="Top P"
              >
                <Input type="number" step="0.1" placeholder="1.0" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="system_prompt"
            label="系统提示词"
          >
            <TextArea rows={4} placeholder="输入系统提示词" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="is_enabled" valuePropName="checked">
                <Switch /> 启用此配置
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="is_default" valuePropName="checked">
                <Switch /> 设为默认配置
              </Form.Item>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  );
};

export default AIAssistantManagement;