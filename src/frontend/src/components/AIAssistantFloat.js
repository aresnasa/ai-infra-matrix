import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  FloatButton,
  Drawer,
  Card,
  Input,
  Button,
  List,
  Avatar,
  Typography,
  Space,
  Select,
  message,
  Tag,
  Spin,
  Dropdown,
  Menu,
} from 'antd';
import {
  RobotOutlined,
  SendOutlined,
  PlusOutlined,
  DeleteOutlined,
  MessageOutlined,
  UserOutlined,
  BulbOutlined,
  MoreOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { aiAPI } from '../services/api';
import { useNavigate } from 'react-router-dom';
import './AIAssistantFloat.css';

const { TextArea } = Input;
const { Text, Title } = Typography;
const { Option } = Select;

const AIAssistantFloat = () => {
  const navigate = useNavigate();
  const [visible, setVisible] = useState(false);
  const [conversations, setConversations] = useState([]);
  const [currentConversation, setCurrentConversation] = useState(null);
  const [messages, setMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [loading, setLoading] = useState(false);
  const [sendingMessage, setSendingMessage] = useState(false);
  const [configs, setConfigs] = useState([]);
  const [selectedConfig, setSelectedConfig] = useState(null);
  const messagesEndRef = useRef(null);

  // 获取配置列表
  const fetchConfigs = async () => {
    try {
      const response = await aiAPI.getConfigs();
      console.log('获取配置响应:', response.data);
      const configData = response.data.data || response.data || [];
      setConfigs(configData);
      const defaultConfig = configData.find(config => config.is_default);
      if (defaultConfig) {
        setSelectedConfig(defaultConfig.id);
      } else if (configData.length > 0) {
        setSelectedConfig(configData[0].id);
      }
    } catch (error) {
      console.error('获取AI配置失败:', error);
      message.error('获取AI配置失败，请检查网络连接或联系管理员');
    }
  };

  // 获取对话列表
  const fetchConversations = async () => {
    try {
      setLoading(true);
      const response = await aiAPI.getConversations();
      console.log('获取对话响应:', response.data);
      const conversationData = response.data.data || response.data || [];
      setConversations(conversationData);
    } catch (error) {
      console.error('获取对话列表失败:', error);
      message.error('获取对话列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取消息列表
  const fetchMessages = useCallback(async (conversationId) => {
    try {
      setLoading(true);
      const response = await aiAPI.getMessages(conversationId);
      console.log('获取消息响应:', response.data);
      const messageData = response.data.data || response.data || [];
      setMessages(messageData);
      scrollToBottom();
    } catch (error) {
      console.error('获取消息失败:', error);
      message.error('获取消息失败');
    } finally {
      setLoading(false);
    }
  }, []);

  // 创建新对话
  const createConversation = async (title = '新对话') => {
    if (!selectedConfig) {
      message.error('请先配置AI模型');
      return;
    }

    try {
      const response = await aiAPI.createConversation({
        config_id: selectedConfig,
        title,
        context: window.location.pathname, // 传递当前页面上下文
      });
      const newConversation = response.data.data;
      setConversations(prev => [newConversation, ...prev]);
      setCurrentConversation(newConversation);
      setMessages([]);
      return newConversation;
    } catch (error) {
      console.error('创建对话失败:', error);
      message.error('创建对话失败');
    }
  };

  // 发送消息（异步版本）
  const sendMessage = async () => {
    if (!inputMessage.trim()) return;

    let conversationToUse = currentConversation;
    
    // 如果没有当前对话，创建新对话
    if (!conversationToUse) {
      conversationToUse = await createConversation();
      if (!conversationToUse) return;
    }

    const userMessage = inputMessage.trim();
    setInputMessage('');
    setSendingMessage(true);

    // 添加用户消息到界面
    const newUserMessage = {
      id: Date.now(),
      role: 'user',
      content: userMessage,
      created_at: new Date().toISOString(),
    };
    setMessages(prev => [...prev, newUserMessage]);

    try {
      // 发送异步请求
      const response = await aiAPI.sendMessage(conversationToUse.id, userMessage);
      const { message_id, status } = response.data;
      
      // 添加状态消息
      const statusMessage = {
        id: message_id,
        role: 'system',
        content: '正在处理您的请求...',
        created_at: new Date().toISOString(),
        isStatus: true,
      };
      setMessages(prev => [...prev, statusMessage]);
      
      // 轮询消息状态
      pollMessageStatus(message_id, conversationToUse.id);
      
    } catch (error) {
      console.error('发送消息失败:', error);
      message.error('发送消息失败');
      // 移除用户消息
      setMessages(prev => prev.filter(msg => msg.id !== newUserMessage.id));
    } finally {
      setSendingMessage(false);
    }
  };

  // 轮询消息状态
  const pollMessageStatus = async (messageId, conversationId, maxAttempts = 30) => {
    let attempts = 0;
    
    const poll = async () => {
      try {
        const response = await aiAPI.getMessageStatus(messageId);
        const { status, result, error } = response.data.data;
        
        if (status === 'completed') {
          // 移除状态消息，添加AI回复
          setMessages(prev => prev.filter(msg => msg.id !== messageId));
          
          if (result) {
            const aiMessage = {
              id: `ai_${Date.now()}`,
              role: 'assistant',
              content: result,
              created_at: new Date().toISOString(),
            };
            setMessages(prev => [...prev, aiMessage]);
          }
          
          // 刷新对话列表
          fetchConversations();
          return;
        } else if (status === 'failed') {
          // 更新状态消息为错误信息
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { ...msg, content: `处理失败: ${error || '未知错误'}`, isError: true }
              : msg
          ));
          return;
        } else if (status === 'processing') {
          // 更新状态消息
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { ...msg, content: 'AI正在思考中...' }
              : msg
          ));
        }
        
        // 继续轮询
        attempts++;
        if (attempts < maxAttempts) {
          setTimeout(poll, 2000); // 2秒后再次查询
        } else {
          // 超时处理
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { ...msg, content: '处理超时，请重试', isError: true }
              : msg
          ));
        }
      } catch (error) {
        console.error('查询消息状态失败:', error);
        setMessages(prev => prev.map(msg => 
          msg.id === messageId 
            ? { ...msg, content: '状态查询失败', isError: true }
            : msg
        ));
      }
    };
    
    // 开始轮询
    setTimeout(poll, 1000); // 1秒后开始查询
  };

  // 快速聊天（异步版本）
  const quickChat = async () => {
    if (!inputMessage.trim()) return;

    const userMessage = inputMessage.trim();
    setInputMessage('');
    setSendingMessage(true);

    try {
      const response = await aiAPI.quickChat(userMessage, window.location.pathname);
      const { message_id } = response.data;
      
      message.success('快速聊天请求已提交');
      
      // 轮询状态并在完成后刷新对话列表
      pollQuickChatStatus(message_id);
      
    } catch (error) {
      console.error('快速聊天失败:', error);
      message.error('快速聊天失败');
    } finally {
      setSendingMessage(false);
    }
  };

  // 轮询快速聊天状态
  const pollQuickChatStatus = async (messageId, maxAttempts = 30) => {
    let attempts = 0;
    
    const poll = async () => {
      try {
        const response = await aiAPI.getMessageStatus(messageId);
        const { status, result } = response.data.data;
        
        if (status === 'completed') {
          // 刷新对话列表
          await fetchConversations();
          message.success('快速聊天完成');
          return;
        } else if (status === 'failed') {
          message.error('快速聊天处理失败');
          return;
        }
        
        // 继续轮询
        attempts++;
        if (attempts < maxAttempts) {
          setTimeout(poll, 2000);
        } else {
          message.warning('快速聊天处理超时');
        }
      } catch (error) {
        console.error('查询快速聊天状态失败:', error);
      }
    };
    
    setTimeout(poll, 1000);
  };

  // 删除对话
  const deleteConversation = async (conversationId) => {
    try {
      await aiAPI.deleteConversation(conversationId);
      setConversations(prev => prev.filter(conv => conv.id !== conversationId));
      if (currentConversation?.id === conversationId) {
        setCurrentConversation(null);
        setMessages([]);
      }
      message.success('对话已删除');
    } catch (error) {
      console.error('删除对话失败:', error);
      message.error('删除对话失败');
    }
  };

  // 滚动到底部
  const scrollToBottom = () => {
    setTimeout(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, 100);
  };

  // 初始化
  useEffect(() => {
    if (visible) {
      fetchConfigs();
      fetchConversations();
    }
  }, [visible]);

  // 当选择对话时，获取消息
  useEffect(() => {
    if (currentConversation) {
      fetchMessages(currentConversation.id);
    }
  }, [currentConversation, fetchMessages]);

  // 处理Enter键发送
  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (currentConversation) {
        sendMessage();
      } else {
        quickChat();
      }
    }
  };

  // 对话菜单
  const getConversationMenu = (conversation) => (
    <Menu>
      <Menu.Item
        key="delete"
        icon={<DeleteOutlined />}
        onClick={() => deleteConversation(conversation.id)}
        danger
      >
        删除对话
      </Menu.Item>
    </Menu>
  );

  return (
    <>
      {/* 悬浮按钮 */}
      <FloatButton
        icon={<RobotOutlined />}
        tooltip="AI助手"
        onClick={() => setVisible(true)}
        style={{
          right: 24,
          bottom: 24,
        }}
      />

      {/* AI助手抽屉 */}
      <Drawer
        title={
          <Space>
            <RobotOutlined />
            <span>AI助手</span>
            {configs.length > 0 && (
              <Select
                value={selectedConfig}
                onChange={setSelectedConfig}
                style={{ width: 120 }}
                size="small"
                className="config-selector"
              >
                {configs.map(config => (
                  <Option key={config.id} value={config.id}>
                    {config.name}
                  </Option>
                ))}
              </Select>
            )}
          </Space>
        }
        placement="right"
        width={400}
        open={visible}
        onClose={() => setVisible(false)}
        bodyStyle={{ padding: 0, display: 'flex', flexDirection: 'column', height: '100%' }}
        className="ai-assistant-drawer"
      >
        {configs.length === 0 ? (
          // 无配置时的提示界面
          <div style={{ padding: 24, textAlign: 'center', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <div style={{ textAlign: 'center' }}>
              <RobotOutlined style={{ fontSize: 48, color: '#1890ff', marginBottom: 16 }} />
              <Title level={4}>AI助手未配置</Title>
              <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
                需要配置AI服务后才能开始智能对话体验。
              </Text>
              <Text type="secondary" style={{ fontSize: 12, display: 'block', marginBottom: 24 }}>
                可以配置OpenAI、Claude等AI服务
              </Text>
              <Space direction="vertical" size="middle">
                <Button 
                  type="primary" 
                  icon={<SettingOutlined />}
                  onClick={() => {
                    setVisible(false);
                    navigate('/admin/ai-configs');
                  }}
                  size="large"
                >
                  配置AI模型
                </Button>
                <Button 
                  type="default"
                  onClick={() => {
                    setVisible(false);
                    navigate('/admin');
                  }}
                >
                  进入管理中心
                </Button>
              </Space>
            </div>
          </div>
        ) : (
          <>
            {/* 对话列表 */}
            <div style={{ borderBottom: '1px solid #f0f0f0', maxHeight: 200, overflow: 'auto' }}>
              <div style={{ padding: '12px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Text strong>对话历史</Text>
                <Button
                  type="text"
                  icon={<PlusOutlined />}
                  onClick={() => createConversation()}
                >
                  新对话
                </Button>
              </div>
              
              {loading && conversations.length === 0 ? (
                <div style={{ textAlign: 'center', padding: 16 }}>
                  <Spin />
                </div>
              ) : conversations.length === 0 ? (
                <div style={{ textAlign: 'center', padding: 16 }}>
                  <Text type="secondary">暂无对话</Text>
                </div>
              ) : (
                <List
                  size="small"
                  dataSource={conversations}
                  renderItem={conversation => (
                    <List.Item
                      style={{
                        padding: '8px 16px',
                        cursor: 'pointer',
                        backgroundColor: currentConversation?.id === conversation.id ? '#f6ffed' : 'transparent',
                      }}
                      onClick={() => setCurrentConversation(conversation)}
                      actions={[
                        <Dropdown
                          overlay={getConversationMenu(conversation)}
                          trigger={['click']}
                          key="more"
                        >
                          <Button type="text" icon={<MoreOutlined />} size="small" />
                        </Dropdown>
                      ]}
                    >
                      <List.Item.Meta
                        avatar={<Avatar icon={<MessageOutlined />} size="small" />}
                        title={
                          <Text ellipsis style={{ fontSize: 12 }}>
                            {conversation.title}
                          </Text>
                        }
                        description={
                          <Text type="secondary" style={{ fontSize: 11 }}>
                            {new Date(conversation.updated_at).toLocaleDateString()}
                          </Text>
                        }
                      />
                    </List.Item>
                  )}
                />
              )}
            </div>

            {/* 消息区域 */}
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              {currentConversation ? (
                <>
                  {/* 消息列表 */}
                  <div style={{ flex: 1, padding: 16, overflow: 'auto', maxHeight: 400 }}>
                    {loading && messages.length === 0 ? (
                      <div style={{ textAlign: 'center', padding: 20 }}>
                        <Spin />
                      </div>
                    ) : messages.length === 0 ? (
                      <div style={{ textAlign: 'center', padding: 20 }}>
                        <BulbOutlined style={{ fontSize: 32, color: '#1890ff', marginBottom: 8 }} />
                        <div>开始与AI对话吧！</div>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          我可以帮您解答关于Ansible、Kubernetes等问题
                        </Text>
                      </div>
                    ) : (
                      <List
                        dataSource={messages}
                        renderItem={message => (
                          <List.Item style={{ border: 'none', padding: '8px 0' }}>
                            <Card
                              size="small"
                              style={{
                                width: '100%',
                                marginLeft: message.role === 'user' ? 20 : 0,
                                marginRight: message.role === 'assistant' || message.role === 'system' ? 20 : 0,
                                backgroundColor: message.role === 'user' ? '#e6f7ff' : 
                                               message.isStatus ? '#f0f2f5' :
                                               message.isError ? '#fff2f0' : '#f6ffed',
                                border: message.isError ? '1px solid #ffccc7' : undefined,
                              }}
                              bodyStyle={{ padding: 12 }}
                            >
                              <Space direction="vertical" style={{ width: '100%' }}>
                                <Space>
                                  <Avatar
                                    icon={message.role === 'user' ? <UserOutlined /> : <RobotOutlined />}
                                    size="small"
                                  />
                                  <Text strong>
                                    {message.role === 'user' ? '我' : 
                                     message.role === 'system' ? '系统' : 'AI助手'}
                                  </Text>
                                  {message.isStatus && (
                                    <Spin size="small" />
                                  )}
                                  {message.tokens_used && (
                                    <Tag color="blue" style={{ fontSize: 10 }}>
                                      {message.tokens_used} tokens
                                    </Tag>
                                  )}
                                </Space>
                                <Text style={{ 
                                  whiteSpace: 'pre-wrap', 
                                  fontSize: 13,
                                  color: message.isError ? '#ff4d4f' : 'inherit'
                                }}>
                                  {message.content}
                                </Text>
                              </Space>
                            </Card>
                          </List.Item>
                        )}
                      />
                    )}
                    <div ref={messagesEndRef} />
                  </div>
                </>
              ) : (
                <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 20 }}>
                  <div style={{ textAlign: 'center' }}>
                    <RobotOutlined style={{ fontSize: 48, color: '#1890ff', marginBottom: 16 }} />
                    <Title level={4}>欢迎使用AI助手</Title>
                    <Text type="secondary">选择一个对话开始聊天，或创建新对话</Text>
                  </div>
                </div>
              )}

              {/* 输入区域 */}
              <div style={{ padding: 16, borderTop: '1px solid #f0f0f0' }}>
                <Space.Compact style={{ width: '100%' }}>
                  <TextArea
                    value={inputMessage}
                    onChange={(e) => setInputMessage(e.target.value)}
                    onKeyPress={handleKeyPress}
                    placeholder={currentConversation ? "输入消息..." : "快速提问..."}
                    autoSize={{ minRows: 1, maxRows: 4 }}
                    disabled={sendingMessage}
                  />
                  <Button
                    type="primary"
                    icon={<SendOutlined />}
                    onClick={currentConversation ? sendMessage : quickChat}
                    loading={sendingMessage}
                    disabled={!inputMessage.trim()}
                  />
                </Space.Compact>
                <div style={{ marginTop: 8, fontSize: 11, color: '#999' }}>
                  {currentConversation ? (
                    <Text type="secondary">当前对话：{currentConversation.title}</Text>
                  ) : (
                    <Text type="secondary">快速模式：将自动创建新对话</Text>
                  )}
                </div>
              </div>
            </div>
          </>
        )}
      </Drawer>
    </>
  );
};

export default AIAssistantFloat;