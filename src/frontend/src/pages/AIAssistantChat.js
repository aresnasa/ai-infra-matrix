import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
  Layout,
  Input,
  Button,
  List,
  Avatar,
  Typography,
  Space,
  Select,
  message,
  Spin,
  Dropdown,
  Menu,
  Modal,
  Tooltip,
  Empty,
  Divider,
} from 'antd';
import {
  SendOutlined,
  PlusOutlined,
  DeleteOutlined,
  MessageOutlined,
  UserOutlined,
  MoreOutlined,
  SettingOutlined,
  StopOutlined,
  EditOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  CopyOutlined,
  CheckOutlined,
} from '@ant-design/icons';
import { aiAPI } from '../services/api';
import { useNavigate } from 'react-router-dom';
import { useTheme } from '../hooks/useTheme';
import AIRobotIcon from '../components/AIRobotIcon';
import './AIAssistantChat.css';

const { Sider, Content } = Layout;
const { TextArea } = Input;
const { Text, Title, Paragraph } = Typography;
const { Option } = Select;

// 获取当前用户信息
const getCurrentUser = () => {
  try {
    const savedUser = localStorage.getItem('user');
    if (savedUser) {
      return JSON.parse(savedUser);
    }
  } catch (error) {
    console.warn('Failed to parse user from localStorage:', error);
  }
  return null;
};

// 检查是否为管理员
const isUserAdmin = (user) => {
  if (!user) return false;
  return user.role === 'admin' || user.role === 'super-admin' ||
         (user.roles && user.roles.some(role => role.name === 'admin' || role.name === 'super-admin'));
};

// 代码块组件
const CodeBlock = ({ language, value, isDark }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="code-block-wrapper">
      <div className="code-block-header">
        <span className="code-language">{language || 'code'}</span>
        <Button
          type="text"
          size="small"
          icon={copied ? <CheckOutlined /> : <CopyOutlined />}
          onClick={handleCopy}
          className="copy-button"
        >
          {copied ? '已复制' : '复制'}
        </Button>
      </div>
      <pre className={`code-block-content ${isDark ? 'dark' : 'light'}`}>
        <code>{value}</code>
      </pre>
    </div>
  );
};

const AIAssistantChat = () => {
  const navigate = useNavigate();
  const { isDark } = useTheme();
  const [collapsed, setCollapsed] = useState(false);
  const [conversations, setConversations] = useState([]);
  const [currentConversation, setCurrentConversation] = useState(null);
  const [messages, setMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [loading, setLoading] = useState(false);
  const [sendingMessage, setSendingMessage] = useState(false);
  const [processingMessageId, setProcessingMessageId] = useState(null);
  const [configs, setConfigs] = useState([]);
  const [selectedConfig, setSelectedConfig] = useState(null);
  const [editingTitle, setEditingTitle] = useState(null);
  const [newTitle, setNewTitle] = useState('');
  const messagesEndRef = useRef(null);
  const inputRef = useRef(null);

  // 滚动到底部
  const scrollToBottom = () => {
    setTimeout(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, 100);
  };

  // 获取配置列表
  const fetchConfigs = async () => {
    try {
      const response = await aiAPI.getConfigs();
      const configData = response.data?.data || response.data || [];
      const enabledConfigs = Array.isArray(configData) 
        ? configData.filter(c => c.enabled !== false) 
        : [];
      setConfigs(enabledConfigs);
      
      // 自动选择第一个配置
      if (enabledConfigs.length > 0 && !selectedConfig) {
        const savedConfig = localStorage.getItem('ai-assistant-selected-config');
        if (savedConfig && enabledConfigs.find(c => c.id === parseInt(savedConfig))) {
          setSelectedConfig(parseInt(savedConfig));
        } else {
          setSelectedConfig(enabledConfigs[0].id);
        }
      }
    } catch (error) {
      console.error('获取配置失败:', error);
    }
  };

  // 获取对话列表
  const fetchConversations = async () => {
    try {
      setLoading(true);
      const response = await aiAPI.getConversations();
      const conversationData = response.data?.data || response.data || [];
      setConversations(conversationData);
    } catch (error) {
      console.error('获取对话列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 获取消息列表
  const fetchMessages = useCallback(async (conversationId) => {
    try {
      setLoading(true);
      const response = await aiAPI.getMessages(conversationId);
      const messageData = response.data?.data || response.data || [];
      setMessages(messageData);
      scrollToBottom();
    } catch (error) {
      console.error('获取消息失败:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  // 创建新对话
  const createConversation = async (title = '新对话') => {
    if (!selectedConfig) {
      message.error('请先选择AI模型');
      return;
    }

    try {
      const response = await aiAPI.createConversation({
        config_id: selectedConfig,
        title,
        context: window.location.pathname,
      });
      const newConversation = response.data?.data || response.data;
      setConversations(prev => [newConversation, ...prev]);
      setCurrentConversation(newConversation);
      setMessages([]);
      inputRef.current?.focus();
      return newConversation;
    } catch (error) {
      console.error('创建对话失败:', error);
      message.error('创建对话失败');
    }
  };

  // 删除对话
  const deleteConversation = async (conversationId) => {
    try {
      await aiAPI.deleteConversation(conversationId);
      setConversations(prev => prev.filter(c => c.id !== conversationId));
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

  // 更新对话标题
  const updateConversationTitle = async (conversationId, title) => {
    try {
      await aiAPI.updateConversation(conversationId, { title });
      setConversations(prev => 
        prev.map(c => c.id === conversationId ? { ...c, title } : c)
      );
      if (currentConversation?.id === conversationId) {
        setCurrentConversation(prev => ({ ...prev, title }));
      }
      setEditingTitle(null);
      message.success('标题已更新');
    } catch (error) {
      console.error('更新标题失败:', error);
      message.error('更新标题失败');
    }
  };

  // 发送消息
  const sendMessage = async () => {
    if (!inputMessage.trim()) return;

    const userMessage = inputMessage.trim();
    
    // 如果没有当前对话，创建一个
    let conversationToUse = currentConversation;
    if (!conversationToUse) {
      conversationToUse = await createConversation(userMessage.substring(0, 30));
      if (!conversationToUse) return;
    }

    setInputMessage('');
    setSendingMessage(true);

    // 添加用户消息
    const newUserMessage = {
      id: Date.now(),
      role: 'user',
      content: userMessage,
      created_at: new Date().toISOString(),
    };
    setMessages(prev => [...prev, newUserMessage]);
    scrollToBottom();

    try {
      const response = await aiAPI.sendMessage(conversationToUse.id, userMessage);
      const { message_id } = response.data;
      setProcessingMessageId(message_id);

      // 添加AI思考状态
      const thinkingMessage = {
        id: message_id,
        role: 'assistant',
        content: '',
        isThinking: true,
        created_at: new Date().toISOString(),
      };
      setMessages(prev => [...prev, thinkingMessage]);
      scrollToBottom();

      // 轮询获取AI回复
      pollMessageStatus(message_id, conversationToUse.id);
    } catch (error) {
      console.error('发送消息失败:', error);
      message.error('发送消息失败');
      setSendingMessage(false);
    }
  };

  // 轮询消息状态
  const pollMessageStatus = async (messageId, conversationId) => {
    const maxAttempts = 60;
    let attempts = 0;

    const poll = async () => {
      try {
        const response = await aiAPI.getMessageStatus(messageId);
        const { status, content, error: errorMsg } = response.data;

        if (status === 'completed') {
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { ...msg, content, isThinking: false }
              : msg
          ));
          setProcessingMessageId(null);
          setSendingMessage(false);
          scrollToBottom();
          return;
        }

        if (status === 'failed') {
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { ...msg, content: errorMsg || '处理失败', isThinking: false, isError: true }
              : msg
          ));
          setProcessingMessageId(null);
          setSendingMessage(false);
          return;
        }

        attempts++;
        if (attempts < maxAttempts) {
          setTimeout(poll, 1000);
        } else {
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { ...msg, content: '请求超时，请重试', isThinking: false, isError: true }
              : msg
          ));
          setProcessingMessageId(null);
          setSendingMessage(false);
        }
      } catch (error) {
        console.error('获取消息状态失败:', error);
        attempts++;
        if (attempts < maxAttempts) {
          setTimeout(poll, 2000);
        }
      }
    };

    poll();
  };

  // 停止消息处理
  const stopMessage = async () => {
    if (!processingMessageId) return;
    
    try {
      await aiAPI.stopMessage(processingMessageId);
      setMessages(prev => prev.map(msg => 
        msg.id === processingMessageId 
          ? { ...msg, content: msg.content || '已停止', isThinking: false }
          : msg
      ));
      setProcessingMessageId(null);
      setSendingMessage(false);
      message.info('已停止处理');
    } catch (error) {
      console.error('停止处理失败:', error);
    }
  };

  // 处理Enter键发送
  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  // 初始化
  useEffect(() => {
    fetchConfigs();
    fetchConversations();
  }, []);

  // 切换对话时获取消息
  useEffect(() => {
    if (currentConversation) {
      fetchMessages(currentConversation.id);
    }
  }, [currentConversation, fetchMessages]);

  // 对话菜单
  const getConversationMenu = (conversation) => (
    <Menu>
      <Menu.Item
        key="rename"
        icon={<EditOutlined />}
        onClick={() => {
          setEditingTitle(conversation.id);
          setNewTitle(conversation.title);
        }}
      >
        重命名
      </Menu.Item>
      <Menu.Item
        key="delete"
        icon={<DeleteOutlined />}
        onClick={() => deleteConversation(conversation.id)}
        danger
      >
        删除
      </Menu.Item>
    </Menu>
  );

  // 渲染消息内容（简单的 Markdown 渲染）
  const renderMessageContent = (content, isDark) => {
    if (!content) return null;
    
    // 处理代码块
    const codeBlockRegex = /```(\w*)\n([\s\S]*?)```/g;
    const parts = [];
    let lastIndex = 0;
    let match;
    
    while ((match = codeBlockRegex.exec(content)) !== null) {
      // 添加代码块前的文本
      if (match.index > lastIndex) {
        parts.push({
          type: 'text',
          content: content.slice(lastIndex, match.index),
        });
      }
      // 添加代码块
      parts.push({
        type: 'code',
        language: match[1] || 'text',
        content: match[2],
      });
      lastIndex = match.index + match[0].length;
    }
    
    // 添加剩余文本
    if (lastIndex < content.length) {
      parts.push({
        type: 'text',
        content: content.slice(lastIndex),
      });
    }
    
    // 如果没有代码块，直接返回普通文本
    if (parts.length === 0) {
      return <div className="message-text">{content}</div>;
    }
    
    return (
      <div className="message-markdown">
        {parts.map((part, index) => {
          if (part.type === 'code') {
            return (
              <CodeBlock
                key={index}
                language={part.language}
                value={part.content}
                isDark={isDark}
              />
            );
          }
          // 简单处理文本中的行内代码
          const textWithInlineCode = part.content.split(/`([^`]+)`/).map((segment, i) => {
            if (i % 2 === 1) {
              return <code key={i} className="inline-code">{segment}</code>;
            }
            return segment;
          });
          return (
            <div key={index} className="message-text" style={{ whiteSpace: 'pre-wrap' }}>
              {textWithInlineCode}
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <Layout className={`ai-chat-layout ${isDark ? 'dark' : 'light'}`}>
      {/* 左侧对话列表 */}
      <Sider
        width={260}
        collapsedWidth={0}
        collapsed={collapsed}
        className="ai-chat-sider"
        theme={isDark ? 'dark' : 'light'}
      >
        <div className="sider-header">
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => createConversation()}
            block
            className="new-chat-btn"
          >
            新建对话
          </Button>
        </div>

        <div className="conversation-list">
          {loading && conversations.length === 0 ? (
            <div className="loading-container">
              <Spin />
            </div>
          ) : conversations.length === 0 ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="暂无对话"
              className="empty-state"
            />
          ) : (
            <List
              dataSource={conversations}
              renderItem={conversation => (
                <div
                  className={`conversation-item ${currentConversation?.id === conversation.id ? 'active' : ''}`}
                  onClick={() => setCurrentConversation(conversation)}
                >
                  <MessageOutlined className="conv-icon" />
                  {editingTitle === conversation.id ? (
                    <Input
                      size="small"
                      value={newTitle}
                      onChange={(e) => setNewTitle(e.target.value)}
                      onPressEnter={() => updateConversationTitle(conversation.id, newTitle)}
                      onBlur={() => setEditingTitle(null)}
                      autoFocus
                      className="title-input"
                      onClick={(e) => e.stopPropagation()}
                    />
                  ) : (
                    <span className="conv-title">{conversation.title}</span>
                  )}
                  <Dropdown
                    overlay={getConversationMenu(conversation)}
                    trigger={['click']}
                    placement="bottomRight"
                  >
                    <Button
                      type="text"
                      size="small"
                      icon={<MoreOutlined />}
                      className="conv-menu-btn"
                      onClick={(e) => e.stopPropagation()}
                    />
                  </Dropdown>
                </div>
              )}
            />
          )}
        </div>

        <div className="sider-footer">
          <Divider style={{ margin: '8px 0' }} />
          {configs.length > 0 && (
            <Select
              value={selectedConfig}
              onChange={(value) => {
                setSelectedConfig(value);
                localStorage.setItem('ai-assistant-selected-config', value.toString());
                message.success('已切换AI模型');
              }}
              style={{ width: '100%' }}
              size="small"
              placeholder="选择AI模型"
            >
              {configs.map(config => (
                <Option key={config.id} value={config.id}>
                  {config.name}
                </Option>
              ))}
            </Select>
          )}
          {isUserAdmin(getCurrentUser()) && (
            <Button
              type="text"
              icon={<SettingOutlined />}
              onClick={() => navigate('/admin/ai-assistant')}
              block
              className="settings-btn"
            >
              AI设置
            </Button>
          )}
        </div>
      </Sider>

      {/* 折叠按钮 */}
      <Button
        type="text"
        icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
        onClick={() => setCollapsed(!collapsed)}
        className="collapse-btn"
      />

      {/* 主对话区域 */}
      <Content className="ai-chat-content">
        {!currentConversation && messages.length === 0 ? (
          // 欢迎界面
          <div className="welcome-container">
            <AIRobotIcon size={64} animated={true} />
            <Title level={2} className="welcome-title">
              AI 智能助手
            </Title>
            <Paragraph className="welcome-desc">
              我可以帮助您解答关于系统运维、Ansible、Kubernetes、SaltStack 等问题
            </Paragraph>
            <div className="suggestion-list">
              {[
                '如何编写一个安装 nginx 的 Ansible playbook？',
                '解释 Kubernetes 的 Pod 调度策略',
                '如何配置 SaltStack 高可用？',
                '帮我写一个监控 GPU 状态的脚本',
              ].map((suggestion, index) => (
                <Button
                  key={index}
                  className="suggestion-btn"
                  onClick={() => {
                    setInputMessage(suggestion);
                    inputRef.current?.focus();
                  }}
                >
                  {suggestion}
                </Button>
              ))}
            </div>
          </div>
        ) : (
          // 消息列表
          <div className="messages-container">
            {messages.map((msg, index) => (
              <div
                key={msg.id || index}
                className={`message-row ${msg.role}`}
              >
                <div className="message-avatar">
                  {msg.role === 'user' ? (
                    <Avatar icon={<UserOutlined />} className="user-avatar" />
                  ) : (
                    <Avatar 
                      icon={<AIRobotIcon size={20} />} 
                      className="ai-avatar"
                    />
                  )}
                </div>
                <div className="message-content">
                  {msg.isThinking ? (
                    <div className="thinking-indicator">
                      <Spin size="small" />
                      <span>AI 正在思考...</span>
                    </div>
                  ) : msg.isError ? (
                    <div className="error-content">
                      {msg.content}
                    </div>
                  ) : msg.role === 'assistant' ? (
                    renderMessageContent(msg.content, isDark)
                  ) : (
                    <div className="user-content">{msg.content}</div>
                  )}
                </div>
              </div>
            ))}
            <div ref={messagesEndRef} />
          </div>
        )}

        {/* 输入区域 */}
        <div className="input-container">
          <div className="input-wrapper">
            <TextArea
              ref={inputRef}
              value={inputMessage}
              onChange={(e) => setInputMessage(e.target.value)}
              onKeyPress={handleKeyPress}
              placeholder="输入消息... (Shift+Enter 换行)"
              autoSize={{ minRows: 1, maxRows: 6 }}
              disabled={sendingMessage}
              className="message-input"
            />
            {processingMessageId ? (
              <Button
                type="primary"
                danger
                icon={<StopOutlined />}
                onClick={stopMessage}
                className="send-btn stop"
              >
                停止
              </Button>
            ) : (
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={sendMessage}
                disabled={!inputMessage.trim() || sendingMessage}
                className="send-btn"
              >
                发送
              </Button>
            )}
          </div>
          <div className="input-hint">
            AI 可能会产生不准确的信息，请核实重要内容
          </div>
        </div>
      </Content>
    </Layout>
  );
};

export default AIAssistantChat;
