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
  StopOutlined,
} from '@ant-design/icons';
import { aiAPI } from '../services/api';
import { useNavigate } from 'react-router-dom';
import AIRobotIcon from './AIRobotIcon';
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
  const [processingMessageId, setProcessingMessageId] = useState(null);
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

  // 停止消息处理
  const stopMessage = async (messageId) => {
    try {
      await aiAPI.stopMessage(messageId);
      
      // 更新消息状态为已停止
      setMessages(prev => prev.map(msg => 
        msg.id === messageId 
          ? { 
              ...msg, 
              content: '消息处理已停止', 
              isError: false,
              status: 'stopped',
              isStopped: true
            }
          : msg
      ));
      
      // 清除处理中的消息ID
      setProcessingMessageId(null);
      setSendingMessage(false);
      
      message.info('消息处理已停止');
    } catch (error) {
      console.error('停止消息失败:', error);
      message.error('停止消息失败');
    }
  };

  // 发送消息（增强版本，包含更好的错误处理和状态管理）
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

    // 添加用户消息到界面（带时间戳）
    const newUserMessage = {
      id: Date.now(),
      role: 'user',
      content: userMessage,
      created_at: new Date().toISOString(),
      status: 'sent', // 添加消息状态
    };
    setMessages(prev => [...prev, newUserMessage]);

    try {
      // 发送异步请求
      const response = await aiAPI.sendMessage(conversationToUse.id, userMessage);
      const { message_id, status } = response.data;
      
      // 设置正在处理的消息ID
      setProcessingMessageId(message_id);
      
      // 添加状态消息（带加载动画）
      const statusMessage = {
        id: message_id,
        role: 'system',
        content: 'AI正在思考中...',
        created_at: new Date().toISOString(),
        isStatus: true,
        status: 'processing',
      };
      setMessages(prev => [...prev, statusMessage]);
      
      // 轮询消息状态（增强版本）
      pollMessageStatus(message_id, conversationToUse.id);
      
    } catch (error) {
      console.error('发送消息失败:', error);
      
      // 更新用户消息状态为失败
      setMessages(prev => prev.map(msg => 
        msg.id === newUserMessage.id 
          ? { ...msg, status: 'failed', error: '发送失败' }
          : msg
      ));
      
      message.error('发送消息失败，请重试');
    } finally {
      setSendingMessage(false);
    }
  };

  // 轮询消息状态（增强版本）
  const pollMessageStatus = async (messageId, conversationId, maxAttempts = 30) => {
    let attempts = 0;
    
    const poll = async () => {
      try {
        attempts++;
        const response = await aiAPI.getMessageStatus(messageId);
        const { status, result, error, tokens_used } = response.data.data;
        
        if (status === 'completed') {
          // 移除状态消息，添加AI回复
          setMessages(prev => prev.filter(msg => msg.id !== messageId));
          
          // 清除正在处理的消息ID
          setProcessingMessageId(null);
          setSendingMessage(false);
          
          if (result) {
            const aiMessage = {
              id: `ai_${Date.now()}`,
              role: 'assistant',
              content: result,
              created_at: new Date().toISOString(),
              tokens_used: tokens_used,
              status: 'completed',
            };
            setMessages(prev => [...prev, aiMessage]);
            
            // 显示token使用信息
            if (tokens_used) {
              message.success(`AI回复完成，使用了 ${tokens_used} 个tokens`);
            }
          }
          
          // 刷新对话列表以更新统计信息
          fetchConversations();
          return;
          
        } else if (status === 'failed') {
          // 更新状态消息为错误信息
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { 
                  ...msg, 
                  content: `处理失败: ${error || '未知错误'}`, 
                  isError: true,
                  status: 'failed'
                }
              : msg
          ));
          
          // 清除正在处理的消息ID
          setProcessingMessageId(null);
          setSendingMessage(false);
          
          message.error(`AI处理失败: ${error || '未知错误'}`);
          return;
          
        } else if (status === 'stopped') {
          // 消息已被停止
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { 
                  ...msg, 
                  content: '消息处理已停止', 
                  isError: false,
                  status: 'stopped',
                  isStopped: true
                }
              : msg
          ));
          
          // 清除正在处理的消息ID
          setProcessingMessageId(null);
          setSendingMessage(false);
          
          message.info('消息处理已停止');
          return;
          
        } else if (status === 'processing') {
          // 更新状态消息内容
          const processingMessages = [
            'AI正在思考中...',
            'AI正在分析您的请求...',
            'AI正在生成回复...',
            'AI正在优化回答...',
          ];
          
          const messageIndex = Math.floor(attempts / 3) % processingMessages.length;
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { ...msg, content: processingMessages[messageIndex] }
              : msg
          ));
          
          // 继续轮询
          if (attempts < maxAttempts) {
            setTimeout(poll, 2000);
          } else {
            // 超时处理
            setMessages(prev => prev.map(msg => 
              msg.id === messageId 
                ? { 
                    ...msg, 
                    content: '处理超时，请稍后重试', 
                    isError: true,
                    status: 'timeout'
                  }
                : msg
            ));
            // 清除正在处理的消息ID
            setProcessingMessageId(null);
            setSendingMessage(false);
            message.warning('AI处理超时，请稍后重试');
          }
        }
        
      } catch (error) {
        console.error('查询消息状态失败:', error);
        
        if (attempts < maxAttempts) {
          setTimeout(poll, 3000); // 增加重试间隔
        } else {
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { 
                  ...msg, 
                  content: '网络错误，请检查连接后重试', 
                  isError: true,
                  status: 'network_error'
                }
              : msg
          ));
          // 清除正在处理的消息ID
          setProcessingMessageId(null);
          setSendingMessage(false);
          message.error('网络错误，无法获取AI回复');
        }
      }
    };
    
    setTimeout(poll, 1000);
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
        icon={<AIRobotIcon size={28} animated={true} />}
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
            <AIRobotIcon size={20} animated={false} />
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
              <AIRobotIcon size={48} animated={true} style={{ marginBottom: 16 }} />
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
                                    icon={message.role === 'user' ? <UserOutlined /> : <AIRobotIcon size={16} animated={false} />}
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
                    <AIRobotIcon size={48} animated={true} style={{ marginBottom: 16 }} />
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
                  {processingMessageId ? (
                    // 显示停止按钮
                    <Button
                      type="primary"
                      danger
                      icon={<StopOutlined />}
                      onClick={() => stopMessage(processingMessageId)}
                      loading={false}
                    />
                  ) : (
                    // 显示发送按钮
                    <Button
                      type="primary"
                      icon={<SendOutlined />}
                      onClick={currentConversation ? sendMessage : quickChat}
                      loading={sendingMessage}
                      disabled={!inputMessage.trim()}
                    />
                  )}
                </Space.Compact>
                <div style={{ marginTop: 8, fontSize: 11, color: '#999' }}>
                  {processingMessageId ? (
                    <Text type="secondary" style={{ color: '#ff4d4f' }}>AI正在处理中，点击停止按钮可中断...</Text>
                  ) : currentConversation ? (
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