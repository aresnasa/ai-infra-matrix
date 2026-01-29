import React, { useState, useEffect, useRef, useCallback } from 'react';
import {
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
  Modal,
  Tooltip,
  Divider,
  Form,
  Radio,
  Collapse,
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
  LockOutlined,
  UnlockOutlined,
  ApiOutlined,
  ThunderboltOutlined,
  EditOutlined,
  ExpandOutlined,
  KeyOutlined,
  LinkOutlined,
} from '@ant-design/icons';
import { aiAPI } from '../services/api';
import { useNavigate } from 'react-router-dom';
import { useTheme } from '../hooks/useTheme';
import AIRobotIcon from './AIRobotIcon';
import './AIAssistantFloat.css';

const { TextArea } = Input;
const { Text, Title } = Typography;
const { Option } = Select;

// 获取当前用户信息的工具函数
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

// 检查是否为管理员的工具函数
const isUserAdmin = (user) => {
  if (!user) return false;
  return user.role === 'admin' || user.role === 'super-admin' ||
         (user.roles && user.roles.some(role => role.name === 'admin' || role.name === 'super-admin'));
};

const AIAssistantFloat = () => {
  const navigate = useNavigate();
  const { isDark } = useTheme();
  const [visible, setVisible] = useState(false); // 控制面板显示/隐藏
  const [locked, setLocked] = useState(false); // 控制面板锁定状态，锁定后不会自动关闭
  const [conversations, setConversations] = useState([]);
  const [currentConversation, setCurrentConversation] = useState(null);
  const [messages, setMessages] = useState([]);
  const [inputMessage, setInputMessage] = useState('');
  const [loading, setLoading] = useState(false);
  const [sendingMessage, setSendingMessage] = useState(false);
  const [processingMessageId, setProcessingMessageId] = useState(null);
  const [configs, setConfigs] = useState([]);
  const [selectedConfig, setSelectedConfig] = useState(null);
  const [showModelConfig, setShowModelConfig] = useState(false); // 控制模型配置弹窗
  const [modelSearchText, setModelSearchText] = useState(''); // 模型配置搜索文本
  const [customModelUrl, setCustomModelUrl] = useState(''); // 自定义模型地址
  const [customRestfulConfig, setCustomRestfulConfig] = useState({ // RESTful接口配置
    name: '',
    apiUrl: '',
    method: 'POST',
    headers: {},
    requestFormat: 'openai', // openai, custom
    authType: 'bearer', // bearer, apikey, none
    authValue: ''
  });
  const [panelWidth, setPanelWidth] = useState(400); // 面板宽度
  const [dragWidth, setDragWidth] = useState(400); // 拖拽时的临时宽度
  const [isResizing, setIsResizing] = useState(false); // 是否正在调整大小
  const [isDragging, setIsDragging] = useState(false); // 是否正在拖拽状态
  const [dragStarted, setDragStarted] = useState(false); // 是否已开始拖拽
  
  // 悬浮按钮位置状态
  const [floatButtonPos, setFloatButtonPos] = useState(() => {
    // 从 localStorage 读取保存的位置
    const saved = localStorage.getItem('ai-float-button-pos');
    if (saved) {
      try {
        return JSON.parse(saved);
      } catch (e) {
        console.warn('Failed to parse float button position');
      }
    }
    return { left: 24, bottom: 24 };
  });
  const [isButtonDragging, setIsButtonDragging] = useState(false);
  const buttonDragRef = useRef({ startX: 0, startY: 0, startLeft: 0, startBottom: 0 });
  
  const messagesEndRef = useRef(null);
  const resizeRef = useRef(null);
  const dragTimeoutRef = useRef(null);
  const dragStateRef = useRef({
    isDragging: false,
    startX: 0,
    startWidth: 0,
    rafId: null
  });

  // 悬浮按钮拖拽处理
  const handleButtonMouseDown = useCallback((e) => {
    if (e.button !== 0) return; // 只处理左键
    
    e.preventDefault();
    e.stopPropagation();
    
    buttonDragRef.current = {
      startX: e.clientX,
      startY: e.clientY,
      startLeft: floatButtonPos.left,
      startBottom: floatButtonPos.bottom,
      hasMoved: false
    };
    
    setIsButtonDragging(true);
    document.body.style.cursor = 'grabbing';
    document.body.style.userSelect = 'none';
  }, [floatButtonPos]);

  const handleButtonMouseMove = useCallback((e) => {
    if (!isButtonDragging) return;
    
    const deltaX = e.clientX - buttonDragRef.current.startX;
    const deltaY = buttonDragRef.current.startY - e.clientY; // 注意：bottom 是从下往上计算的
    
    // 检测是否真正移动了
    if (Math.abs(deltaX) > 3 || Math.abs(deltaY) > 3) {
      buttonDragRef.current.hasMoved = true;
    }
    
    const newLeft = Math.max(10, Math.min(window.innerWidth - 74, buttonDragRef.current.startLeft + deltaX));
    const newBottom = Math.max(10, Math.min(window.innerHeight - 74, buttonDragRef.current.startBottom + deltaY));
    
    setFloatButtonPos({ left: newLeft, bottom: newBottom });
  }, [isButtonDragging]);

  const handleButtonMouseUp = useCallback((e) => {
    if (!isButtonDragging) return;
    
    setIsButtonDragging(false);
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    
    // 保存位置到 localStorage
    const currentPos = { 
      left: buttonDragRef.current.startLeft + (e.clientX - buttonDragRef.current.startX),
      bottom: buttonDragRef.current.startBottom + (buttonDragRef.current.startY - e.clientY)
    };
    localStorage.setItem('ai-float-button-pos', JSON.stringify(currentPos));
    
    // 如果没有移动，则视为点击，标记需要打开面板
    if (!buttonDragRef.current.hasMoved) {
      buttonDragRef.current.shouldOpen = true;
    }
  }, [isButtonDragging]);

  // 处理点击打开面板（在拖拽结束后）
  useEffect(() => {
    if (!isButtonDragging && buttonDragRef.current.shouldOpen) {
      buttonDragRef.current.shouldOpen = false;
      console.log('🔄 打开AI助手，刷新配置列表...');
      setVisible(true);
    }
  }, [isButtonDragging]);

  // 监听悬浮按钮拖拽的全局事件
  useEffect(() => {
    if (isButtonDragging) {
      window.addEventListener('mousemove', handleButtonMouseMove);
      window.addEventListener('mouseup', handleButtonMouseUp);
      return () => {
        window.removeEventListener('mousemove', handleButtonMouseMove);
        window.removeEventListener('mouseup', handleButtonMouseUp);
      };
    }
  }, [isButtonDragging, handleButtonMouseMove, handleButtonMouseUp]);

  // 通知布局组件 AI 助手面板状态变化
  useEffect(() => {
    // 派发自定义事件，通知布局组件面板状态变化
    const event = new CustomEvent('ai-assistant-panel-change', {
      detail: {
        visible,
        width: visible ? panelWidth : 0
      }
    });
    window.dispatchEvent(event);
    
    // 更新 CSS 变量，用于布局偏移
    document.documentElement.style.setProperty(
      '--ai-panel-width',
      visible ? `${panelWidth}px` : '0px'
    );
  }, [visible, panelWidth]);

  // 获取模型图标
  const getModelIcon = (model) => {
    if (!model) return <RobotOutlined />;
    
    const modelName = (model.name || '').toLowerCase();
    const modelType = (model.model_type || '').toLowerCase();
    const provider = (model.provider || '').toLowerCase();
    
    if (modelName.includes('gpt') || modelType.includes('openai') || provider.includes('openai')) {
      return <ThunderboltOutlined style={{ color: '#10B981' }} />;
    } else if (modelName.includes('claude') || modelType.includes('anthropic') || provider.includes('claude')) {
      return <BulbOutlined style={{ color: '#F59E0B' }} />;
    } else if (modelName.includes('gemini') || modelType.includes('google') || provider.includes('google')) {
      return <ApiOutlined style={{ color: '#3B82F6' }} />;
    } else {
      return <RobotOutlined style={{ color: '#8B5CF6' }} />;
    }
  };

  // 获取模型状态标签
  const getModelStatusTag = (model) => {
    if (!model) return null;
    
    if (model.is_default) {
      return <Tag color="green" size="small">默认</Tag>;
    } else if (model.api_endpoint && model.api_endpoint !== '') {
      return <Tag color="blue" size="small">自定义</Tag>;
    }
    return null;
  };

  // 优化的拖拽处理逻辑 - 必须点击左键才能开始拖拽
  const handleResizeMouseDown = useCallback((e) => {
    // 只处理左键点击
    if (e.button !== 0) return;
    
    e.preventDefault();
    e.stopPropagation();
    
    // 设置拖拽初始状态
    dragStateRef.current = {
      isDragging: true,
      startX: e.clientX,
      startWidth: panelWidth,
      rafId: null
    };
    
    setIsDragging(true);
    setIsResizing(true);
    
    // 设置拖拽样式
    document.body.style.cursor = 'ew-resize';
    document.body.style.userSelect = 'none';
    document.body.style.pointerEvents = 'none';
    
    console.log('🖱️ 开始拖拽调整面板大小');
  }, [panelWidth]);

  // 鼠标移动处理 - 只在拖拽状态下生效
  const handleGlobalMouseMove = useCallback((e) => {
    if (!dragStateRef.current.isDragging) return;
    
    e.preventDefault();
    
    // 取消之前的动画帧
    if (dragStateRef.current.rafId) {
      cancelAnimationFrame(dragStateRef.current.rafId);
    }
    
    // 使用 RAF 优化性能
    dragStateRef.current.rafId = requestAnimationFrame(() => {
      // 面板在左侧，向右拖拽增加宽度
      const deltaX = e.clientX - dragStateRef.current.startX;
      const newWidth = Math.max(320, Math.min(800, dragStateRef.current.startWidth + deltaX));
      
      // 使用临时宽度避免频繁更新状态
      setDragWidth(newWidth);
      setPanelWidth(newWidth);
      
      dragStateRef.current.rafId = null;
    });
  }, []);

  // 鼠标释放处理 - 结束拖拽
  const handleGlobalMouseUp = useCallback((e) => {
    if (!dragStateRef.current.isDragging) return;
    
    console.log('🖱️ 结束拖拽调整');
    
    // 清理拖拽状态
    dragStateRef.current.isDragging = false;
    setIsDragging(false);
    setIsResizing(false);
    
    // 取消动画帧
    if (dragStateRef.current.rafId) {
      cancelAnimationFrame(dragStateRef.current.rafId);
      dragStateRef.current.rafId = null;
    }
    
    // 计算最终宽度 - 面板在左侧，向右拖拽增加宽度
    const finalDelta = e.clientX - dragStateRef.current.startX;
    const finalWidth = Math.max(320, Math.min(800, dragStateRef.current.startWidth + finalDelta));
    
    // 设置最终宽度
    setPanelWidth(finalWidth);
    setDragWidth(finalWidth);
    
    // 重置样式
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    document.body.style.pointerEvents = '';
  }, []);

  // 全局鼠标事件监听
  useEffect(() => {
    // 添加全局事件监听器
    document.addEventListener('mousemove', handleGlobalMouseMove, { passive: false });
    document.addEventListener('mouseup', handleGlobalMouseUp, { passive: false });
    
    return () => {
      // 清理事件监听器
      document.removeEventListener('mousemove', handleGlobalMouseMove);
      document.removeEventListener('mouseup', handleGlobalMouseUp);
      
      // 清理拖拽状态
      if (dragStateRef.current.rafId) {
        cancelAnimationFrame(dragStateRef.current.rafId);
      }
      dragStateRef.current.isDragging = false;
    };
  }, [handleGlobalMouseMove, handleGlobalMouseUp]);

  // 保存自定义RESTful配置
  const saveCustomRestfulConfig = async () => {
    try {
      // 检查管理员权限
      const currentUser = getCurrentUser();
      if (!isUserAdmin(currentUser)) {
        message.error('只有管理员才能创建AI模型配置，请联系管理员');
        return;
      }
      
      // 验证必填字段
      if (!customRestfulConfig.apiUrl) {
        message.error('请填写API地址');
        return;
      }
      
      if (!customRestfulConfig.name) {
        message.error('请填写配置名称');
        return;
      }

      console.log('🔐 管理员权限验证通过，创建配置...');

      // 构造配置数据，映射到后端字段格式
      const configData = {
        name: customRestfulConfig.name,
        provider: 'custom',
        model: 'custom-restful-model',
        api_endpoint: customRestfulConfig.apiUrl,
        api_key: customRestfulConfig.authValue,
        headers: JSON.stringify(customRestfulConfig.headers), // 转换为JSON字符串
        parameters: JSON.stringify({
          method: customRestfulConfig.method,
          requestFormat: customRestfulConfig.requestFormat,
          authType: customRestfulConfig.authType
        }),
        description: `通过AI助手浮窗创建的RESTful配置`,
        category: '自定义接口',
        max_tokens: 4096,
        temperature: 0.7,
        top_p: 1.0,
        is_enabled: true,
        is_default: false
      };

      console.log('📡 保存RESTful配置到后端:', configData);
      
      // 调用API创建配置
      await aiAPI.createConfig(configData);
      
      message.success('RESTful配置保存成功，已同步到AI助手管理');
      setShowModelConfig(false);
      
      // 重置表单
      setCustomRestfulConfig({
        name: '',
        apiUrl: '',
        method: 'POST',
        headers: {},
        requestFormat: 'openai',
        authType: 'bearer',
        authValue: ''
      });
      
      // 刷新配置列表以显示新增的配置
      await fetchConfigs();
      
      // 通知其他组件配置已更新 (用于AI助手管理页面同步)
      console.log('🔔 通知AI助手管理页面配置已更新');
      localStorage.setItem('ai-config-updated', Date.now().toString());
      
      // 使用 setTimeout 确保 localStorage 事件能被正确触发
      setTimeout(() => {
        window.dispatchEvent(new Event('storage'));
        
        // 触发自定义事件 (同页面组件通信)
        window.dispatchEvent(new CustomEvent('ai-config-updated', {
          detail: { action: 'created', config: configData, timestamp: Date.now() }
        }));
        console.log('🔔 配置更新事件已发送');
      }, 100);
    } catch (error) {
      console.error('❌ 保存RESTful配置失败:', error);
      const errorMsg = error.response?.data?.message || error.message || '保存配置失败';
      message.error(`保存配置失败: ${errorMsg}`);
    }
  };

  // 过滤模型配置的函数
  const getFilteredConfigs = () => {
    if (!modelSearchText.trim()) {
      return configs; // 没有搜索文本时返回所有配置
    }
    
    const searchText = modelSearchText.toLowerCase();
    return configs.filter(config => {
      const modelName = (config.name || '').toLowerCase();
      const modelType = (config.model_type || '').toLowerCase();
      const apiUrl = (config.api_endpoint || '').toLowerCase();
      const description = (config.description || '').toLowerCase();
      const provider = (config.provider || '').toLowerCase();
      
      // 支持按名称、类型、API地址、描述、提供商进行模糊搜索
      return modelName.includes(searchText) || 
             modelType.includes(searchText) || 
             apiUrl.includes(searchText) ||
             description.includes(searchText) ||
             provider.includes(searchText);
    });
  };

  // 获取配置列表
  const fetchConfigs = async () => {
    try {
      console.log('📡 开始获取AI配置列表...');
      const response = await aiAPI.getConfigs();
      console.log('✅ 获取配置响应:', response);
      console.log('✅ 原始响应数据:', response.data);
      
      let configData = [];
      
      // 处理不同的响应格式
      if (response.data) {
        if (response.data.data && Array.isArray(response.data.data)) {
          configData = response.data.data;
        } else if (Array.isArray(response.data)) {
          configData = response.data;
        } else if (response.data.configs && Array.isArray(response.data.configs)) {
          configData = response.data.configs;
        } else {
          console.warn('⚠️ 未知的响应格式，尝试作为数组处理:', response.data);
          configData = [];
        }
      }
      
      console.log('📋 处理后的配置数据:', configData);
      console.log('📋 配置数据类型:', Array.isArray(configData) ? 'Array' : typeof configData);
      console.log('📋 配置数量:', configData.length);
      
      if (!Array.isArray(configData)) {
        console.error('❌ 配置数据不是数组:', configData);
        setConfigs([]);
        message.warning('配置数据格式错误，请联系管理员检查');
        return;
      }
      
      // 确保每个配置都有有效的ID
      const validConfigs = configData.filter(config => {
        const hasId = config.id && typeof config.id === 'number';
        const hasName = config.name && typeof config.name === 'string';
        if (!hasId) console.warn('⚠️ 配置缺少ID:', config);
        if (!hasName) console.warn('⚠️ 配置缺少名称:', config);
        return hasId && hasName;
      });
      
      console.log('✅ 有效配置数量:', validConfigs.length);
      
      // 如果没有有效配置，尝试创建一个默认配置
      if (validConfigs.length === 0) {
        console.warn('⚠️ 没有有效配置，尝试创建默认配置...');
        try {
          const defaultConfig = {
            name: '默认 OpenAI GPT-4',
            provider: 'openai',
            model: 'gpt-4',
            api_endpoint: 'https://api.openai.com/v1',
            max_tokens: 4096,
            temperature: 0.7,
            system_prompt: '你是一个智能的AI助手，请提供准确、有用的回答。',
            is_enabled: true,
            is_default: true,
            description: '默认的OpenAI GPT-4模型配置',
            category: '通用对话'
          };
          
          console.log('🔧 尝试创建默认配置:', defaultConfig);
          const createResponse = await aiAPI.createConfig(defaultConfig);
          console.log('✅ 默认配置创建成功:', createResponse);
          
          // 重新获取配置列表
          setTimeout(() => {
            fetchConfigs();
          }, 1000);
          
          message.info('正在创建默认配置，请稍候...');
          return;
        } catch (createError) {
          console.error('❌ 创建默认配置失败:', createError);
          message.error('无法创建默认配置，请联系管理员');
          setConfigs([]);
          return;
        }
      }
      
      setConfigs(validConfigs);
      
      // 获取用户之前保存的选择
      const savedConfigId = localStorage.getItem('ai-assistant-selected-config');
      let targetConfigId = null;
      
      if (savedConfigId) {
        // 检查保存的配置是否仍然存在
        const savedConfig = validConfigs.find(config => config.id === parseInt(savedConfigId));
        if (savedConfig) {
          console.log('📋 恢复用户之前的选择:', savedConfig.name);
          targetConfigId = savedConfig.id;
        } else {
          console.log('⚠️ 保存的配置不存在，清除localStorage');
          localStorage.removeItem('ai-assistant-selected-config');
        }
      }
      
      // 如果没有保存的配置或配置不存在，则选择默认配置
      if (!targetConfigId) {
        const defaultConfig = validConfigs.find(config => config.is_default);
        if (defaultConfig) {
          console.log('🎯 使用默认配置:', defaultConfig.name);
          targetConfigId = defaultConfig.id;
        } else if (validConfigs.length > 0) {
          console.log('🎯 使用第一个配置:', validConfigs[0].name);
          targetConfigId = validConfigs[0].id;
        }
      }
      
      // 只有在当前没有选择配置时才设置，避免覆盖用户当前的选择
      if (targetConfigId && !selectedConfig) {
        console.log('🎯 设置选中配置:', targetConfigId);
        setSelectedConfig(targetConfigId);
      } else if (targetConfigId && selectedConfig && selectedConfig !== targetConfigId) {
        console.log('🎯 更新选中配置:', targetConfigId, '(之前:', selectedConfig, ')');
        setSelectedConfig(targetConfigId);
      }
    } catch (error) {
      console.error('❌ 获取AI配置失败:', error);
      console.error('错误详情:', error.response?.data || error.message);
      console.error('错误堆栈:', error.stack);
      
      // 根据错误类型提供不同的处理
      if (error.response) {
        const status = error.response.status;
        const errorData = error.response.data;
        
        if (status === 401) {
          message.error('认证失败，请重新登录');
          console.log('🔑 认证失败，可能需要登录');
        } else if (status === 403) {
          message.error('权限不足，无法访问AI配置');
          console.log('🚫 权限不足');
        } else if (status === 404) {
          message.error('API端点不存在，请检查配置');
          console.log('🔍 API端点未找到');
        } else if (status >= 500) {
          message.error('服务器内部错误，请稍后重试');
          console.log('🔥 服务器错误');
        } else {
          message.error(`获取配置失败: ${errorData?.message || error.message}`);
        }
      } else if (error.request) {
        message.error('网络连接失败，请检查网络连接');
        console.log('🌐 网络连接问题');
      } else {
        message.error('未知错误，请联系管理员');
        console.log('❓ 未知错误');
      }
      
      setConfigs([]);
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

    // 停止消息处理（增强版本）
  const stopMessage = async () => {
    if (!processingMessageId) {
      console.log('⚠️ 没有正在处理的消息');
      return;
    }
    
    console.log('⏹️ 正在停止消息处理:', processingMessageId);
    
    try {
      // 调用API停止消息
      const response = await aiAPI.stopMessage(processingMessageId);
      console.log('✅ 停止消息API响应:', response.data);
      
      // 更新消息状态为已停止
      setMessages(prev => prev.map(msg => 
        msg.id === processingMessageId 
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
      
    } catch (error) {
      console.error('❌ 停止消息失败:', error);
      
      // 即使API调用失败，也更新本地状态
      setMessages(prev => prev.map(msg => 
        msg.id === processingMessageId 
          ? { 
              ...msg, 
              content: '停止请求已发送', 
              isError: false,
              status: 'stopping',
              isStopped: true
            }
          : msg
      ));
      
      // 清除正在处理的消息ID
      setProcessingMessageId(null);
      setSendingMessage(false);
      
      message.warning('停止请求已发送，但可能需要等待AI处理完成');
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
        isProcessing: true, // 添加处理标识
      };
      
      console.log('📝 添加状态消息:', statusMessage);
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

  // 轮询消息状态（增强版本，修复状态显示问题）
  const pollMessageStatus = async (messageId, conversationId, maxAttempts = 30) => {
    let attempts = 0;
    let lastStatus = 'processing';
    
    console.log('🔄 开始轮询消息状态:', messageId);
    
    const poll = async () => {
      try {
        attempts++;
        console.log(`📊 轮询尝试 ${attempts}/${maxAttempts}, 消息ID: ${messageId}`);
        
        const response = await aiAPI.getMessageStatus(messageId);
        const { status, result, error, tokens_used } = response.data.data;
        
        console.log('📋 消息状态响应:', { status, result: result?.substring(0, 100), error, tokens_used });
        
        // 更新最后状态
        lastStatus = status;
        
        if (status === 'completed') {
          console.log('✅ 消息处理完成');
          
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
          console.log('❌ 消息处理失败:', error);
          
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
          console.log('⏹️ 消息已被停止');
          
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
          console.log('🔄 消息正在处理中...');
          
          // 更新状态消息内容
          const processingMessages = [
            'AI正在思考中...',
            'AI正在分析您的请求...',
            'AI正在生成回复...',
            'AI正在优化回答...',
          ];
          
          const messageIndex = Math.floor(attempts / 3) % processingMessages.length;
          const currentMessage = processingMessages[messageIndex];
          
          console.log(`📝 更新状态消息: "${currentMessage}"`);
          
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { 
                  ...msg, 
                  content: currentMessage,
                  status: 'processing'
                }
              : msg
          ));
          
          // 继续轮询
          if (attempts < maxAttempts) {
            console.log(`⏰ ${attempts}/${maxAttempts} 轮询继续...`);
            setTimeout(poll, 2000);
          } else {
            console.log('⏰ 轮询超时');
            
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
        } else {
          console.log('⚠️ 未知状态:', status);
          
          // 未知状态，继续轮询
          if (attempts < maxAttempts) {
            setTimeout(poll, 2000);
          } else {
            setMessages(prev => prev.map(msg => 
              msg.id === messageId 
                ? { 
                    ...msg, 
                    content: '处理状态未知，请稍后重试', 
                    isError: true,
                    status: 'unknown'
                  }
                : msg
            ));
            setProcessingMessageId(null);
            setSendingMessage(false);
            message.warning('处理状态未知，请稍后重试');
          }
        }
        
      } catch (error) {
        console.error('❌ 查询消息状态失败:', error);
        
        if (attempts < maxAttempts) {
          console.log(`🔄 网络错误重试 ${attempts}/${maxAttempts}`);
          setTimeout(poll, 3000); // 增加重试间隔
        } else {
          console.log('❌ 网络错误重试次数用尽');
          
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
    
    console.log('🚀 启动轮询...');
    setTimeout(poll, 1000);
  };

  // 快速聊天（改进版本 - 立即创建对话并同步消息）
  const quickChat = async () => {
    if (!inputMessage.trim()) return;

    const userMessage = inputMessage.trim();
    setInputMessage('');
    setSendingMessage(true);

    try {
      // 首先创建一个新对话
      const newConversation = await createConversation('新对话');
      if (!newConversation) {
        setSendingMessage(false);
        return;
      }

      // 确保面板打开以显示对话
      if (!visible) {
        setVisible(true);
      }

      // 添加用户消息到界面
      const newUserMessage = {
        id: Date.now(),
        role: 'user',
        content: userMessage,
        created_at: new Date().toISOString(),
        status: 'sent',
      };
      setMessages([newUserMessage]);

      // 发送快速聊天请求
      const response = await aiAPI.quickChat(userMessage, window.location.pathname);
      const { message_id } = response.data;
      
      // 设置正在处理的消息ID
      setProcessingMessageId(message_id);
      
      // 添加状态消息
      const statusMessage = {
        id: message_id,
        role: 'system',
        content: 'AI正在思考中...',
        created_at: new Date().toISOString(),
        isStatus: true,
        status: 'processing',
        isProcessing: true,
      };
      
      setMessages(prev => [...prev, statusMessage]);
      
      message.success('快速聊天已创建新对话');
      
      // 轮询状态并同步消息到当前对话
      pollQuickChatStatus(message_id, newConversation.id);
      
    } catch (error) {
      console.error('快速聊天失败:', error);
      message.error('快速聊天失败');
      setSendingMessage(false);
    }
  };

  // 轮询快速聊天状态（改进版本 - 实时更新消息）
  const pollQuickChatStatus = async (messageId, conversationId, maxAttempts = 30) => {
    let attempts = 0;
    
    const poll = async () => {
      try {
        attempts++;
        console.log(`🔄 轮询快速聊天状态 ${attempts}/${maxAttempts}, 消息ID: ${messageId}`);
        
        const response = await aiAPI.getMessageStatus(messageId);
        const { status, result, error, tokens_used } = response.data.data;
        
        if (status === 'completed') {
          console.log('✅ 快速聊天处理完成');
          
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
              message.success(`快速聊天完成，使用了 ${tokens_used} 个tokens`);
            } else {
              message.success('快速聊天完成');
            }
          }
          
          // 刷新对话列表以更新统计信息
          await fetchConversations();
          return;
          
        } else if (status === 'failed') {
          console.log('❌ 快速聊天处理失败:', error);
          
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
          
          message.error(`快速聊天失败: ${error || '未知错误'}`);
          return;
          
        } else if (status === 'stopped') {
          console.log('⏹️ 快速聊天已被停止');
          
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { 
                  ...msg, 
                  content: '快速聊天已停止', 
                  isError: false,
                  status: 'stopped',
                  isStopped: true
                }
              : msg
          ));
          
          setProcessingMessageId(null);
          setSendingMessage(false);
          
          message.info('快速聊天已停止');
          return;
          
        } else if (status === 'processing') {
          console.log('🔄 快速聊天正在处理中...');
          
          // 更新状态消息内容
          const processingMessages = [
            'AI正在分析您的快速提问...',
            'AI正在生成回复...',
            'AI正在优化答案...',
            '即将完成...',
          ];
          
          const messageIndex = Math.floor(attempts / 3) % processingMessages.length;
          const currentMessage = processingMessages[messageIndex];
          
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { 
                  ...msg, 
                  content: currentMessage,
                  status: 'processing'
                }
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
                    content: '快速聊天处理超时，请稍后重试', 
                    isError: true,
                    status: 'timeout'
                  }
                : msg
            ));
            setProcessingMessageId(null);
            setSendingMessage(false);
            message.warning('快速聊天处理超时，请稍后重试');
          }
        } else {
          // 未知状态，继续轮询
          if (attempts < maxAttempts) {
            setTimeout(poll, 2000);
          } else {
            setMessages(prev => prev.map(msg => 
              msg.id === messageId 
                ? { 
                    ...msg, 
                    content: '快速聊天状态未知，请稍后重试', 
                    isError: true,
                    status: 'unknown'
                  }
                : msg
            ));
            setProcessingMessageId(null);
            setSendingMessage(false);
            message.warning('快速聊天状态未知，请稍后重试');
          }
        }
        
      } catch (error) {
        console.error('❌ 查询快速聊天状态失败:', error);
        
        if (attempts < maxAttempts) {
          setTimeout(poll, 3000); // 增加重试间隔
        } else {
          setMessages(prev => prev.map(msg => 
            msg.id === messageId 
              ? { 
                  ...msg, 
                  content: '网络错误，无法获取快速聊天回复', 
                  isError: true,
                  status: 'network_error'
                }
              : msg
          ));
          setProcessingMessageId(null);
          setSendingMessage(false);
          message.error('网络错误，无法获取快速聊天回复');
        }
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

  // 初始化和配置同步
  useEffect(() => {
    if (visible) {
      fetchConfigs();
      fetchConversations();
      
      // 设置定时同步配置（每30秒检查一次）
      const syncInterval = setInterval(() => {
        console.log('🔄 定时同步AI配置列表...');
        fetchConfigs();
      }, 30000);

      // 清理定时器
      return () => {
        clearInterval(syncInterval);
      };
    }
  }, [visible]);

  // 当选择对话时，获取消息
  useEffect(() => {
    if (currentConversation) {
      fetchMessages(currentConversation.id);
    }
  }, [currentConversation, fetchMessages]);

  // 处理点击外部区域关闭面板
  useEffect(() => {
    const handleClickOutside = (event) => {
      if (visible && !locked && !event.target.closest('.ai-assistant-panel') && !event.target.closest('.ant-float-btn')) {
        setVisible(false);
      }
    };

    const handleEscapeKey = (event) => {
      if (event.key === 'Escape' && visible && !locked) {
        setVisible(false);
      }
    };

    // 键盘快捷键调整面板宽度
    const handleKeyDown = (event) => {
      if (!visible) return;
      
      if (event.ctrlKey || event.metaKey) {
        switch (event.key) {
          case '+':
          case '=':
            event.preventDefault();
            setPanelWidth(prev => Math.min(800, prev + 50));
            break;
          case '-':
            event.preventDefault();
            setPanelWidth(prev => Math.max(320, prev - 50));
            break;
          case '0':
            event.preventDefault();
            setPanelWidth(400); // 重置为默认宽度
            break;
          default:
            break;
        }
      }
    };

    if (visible) {
      document.addEventListener('mousedown', handleClickOutside);
      document.addEventListener('keydown', handleEscapeKey);
      document.addEventListener('keydown', handleKeyDown);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscapeKey);
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [visible, locked]);

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
      {/* 可拖拽悬浮按钮 - 面板打开时隐藏 */}
      {!visible && (
        <div
          className={`ai-float-button ${isButtonDragging ? 'dragging' : ''}`}
          onMouseDown={handleButtonMouseDown}
          style={{
            position: 'fixed',
            left: floatButtonPos.left,
            bottom: floatButtonPos.bottom,
            width: 56,
            height: 56,
            borderRadius: '50%',
            background: isDark 
              ? 'linear-gradient(135deg, #1f1f1f 0%, #2d2d2d 100%)' 
              : 'linear-gradient(135deg, #e6f7ff 0%, #bae7ff 100%)',
            border: isDark ? '2px solid #177ddc' : '2px solid #1890ff',
            boxShadow: isButtonDragging 
              ? '0 8px 24px rgba(24, 144, 255, 0.4)' 
              : '0 4px 16px rgba(24, 144, 255, 0.25)',
            cursor: isButtonDragging ? 'grabbing' : 'grab',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 9998,
            transition: isButtonDragging ? 'none' : 'box-shadow 0.3s, transform 0.2s',
            transform: isButtonDragging ? 'scale(1.1)' : 'scale(1)',
          }}
          title="AI助手 (可拖动)"
        >
          <AIRobotIcon size={28} animated={!isButtonDragging} />
        </div>
      )}

      {/* AI助手侧边面板 - 左侧 */}
      <div
        className={`ai-assistant-panel ${visible ? 'ai-assistant-panel-visible' : ''} ${locked ? 'ai-assistant-panel-locked' : ''} ${isResizing ? 'ai-assistant-panel-resizing' : ''} ${isDark ? 'ai-assistant-panel-dark' : ''}`}
        style={{
          position: 'fixed',
          top: 0,
          left: visible ? 0 : -(panelWidth + 20),
          width: isDragging ? dragWidth : panelWidth, // 拖拽时使用dragWidth获得更好的实时反馈
          height: '100vh',
          background: isDark ? '#141414' : '#ffffff',
          boxShadow: isDark ? '2px 0 8px rgba(0, 0, 0, 0.45)' : '2px 0 8px rgba(0, 0, 0, 0.15)',
          zIndex: 9999,
          display: 'flex',
          flexDirection: 'column',
          transition: isResizing ? 'none' : 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
          borderRight: isDark ? '1px solid #303030' : '1px solid #e8e8e8',
        }}
      >
        {/* 拖拽调整大小的手柄 - 右侧 */}
        <div
          className="resize-handle resize-handle-right"
          onMouseDown={handleResizeMouseDown}
          title={`点击并拖拽调整面板宽度 (当前: ${isDragging ? dragWidth : panelWidth}px)`}
          style={{
            position: 'absolute',
            right: 0,
            top: 0,
            width: 6,
            height: '100%',
            cursor: 'ew-resize',
            zIndex: 10000,
          }}
        >
          <div
            className="resize-indicator"
            style={{
              position: 'absolute',
              right: 0,
              top: '50%',
              transform: 'translateY(-50%)',
              width: 3,
              height: 50,
              borderRadius: 2,
              background: isResizing ? '#1890ff' : '#d9d9d9',
              opacity: isResizing ? 1 : 0.6,
            }}
          />
        </div>
        {/* 面板头部 */}
        <div
          style={{
            padding: '16px 20px',
            borderBottom: isDark ? '1px solid #303030' : '1px solid #f0f0f0',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            background: isDark ? '#1f1f1f' : '#fafafa',
          }}
        >
          <Space>
            <AIRobotIcon size={20} animated={false} />
            <span style={{ fontSize: 16, fontWeight: 500 }}>AI助手</span>
            {/* 模型选择器 */}
            <Space size={4}>
              {configs.length > 0 ? (
                <>
                  <Select
                    value={selectedConfig}
                    onChange={(value) => {
                      console.log('🔄 切换模型配置:', value, 'from:', selectedConfig);
                      setSelectedConfig(value);
                      
                      // 保存用户选择到localStorage
                      localStorage.setItem('ai-assistant-selected-config', value.toString());
                      
                      const selected = configs.find(c => c.id === value);
                      if (selected) {
                        console.log('✅ 已选择模型:', selected.name);
                        message.success(`已切换到模型: ${selected.name}`);
                        
                        // 触发配置更新事件
                        window.dispatchEvent(new CustomEvent('ai-config-updated', {
                          detail: { configId: value, config: selected }
                        }));
                      } else {
                        console.warn('⚠️ 未找到选中的配置:', value);
                      }
                    }}
                    onDropdownVisibleChange={(open) => {
                      console.log('📋 下拉框状态变化:', open, '配置数量:', configs.length);
                    }}
                    onClick={() => {
                      console.log('🖱️ Select组件被点击，当前配置:', configs);
                    }}
                    style={{ width: Math.min(220, panelWidth * 0.4) }} // 动态宽度
                    size="small"
                    className="config-selector"
                    placeholder="选择AI模型"
                    optionLabelProp="label"
                    dropdownStyle={{ minWidth: 280, zIndex: 10001 }} // 增加z-index确保显示在最前面
                    notFoundContent="暂无可用配置"
                    loading={loading}
                    allowClear={false}
                    disabled={configs.length === 0}
                    getPopupContainer={(trigger) => trigger.parentNode} // 确保下拉框在正确的容器中渲染
                  >
                    {configs.map(config => {
                      console.log('🎨 渲染配置选项:', config.id, config.name, config);
                      return (
                        <Option 
                          key={config.id} 
                          value={config.id}
                          label={
                            <Space size={4}>
                              {getModelIcon(config)}
                              <span>{config.name}</span>
                              {getModelStatusTag(config)}
                            </Space>
                          }
                        >
                          <Space size={8}>
                            {getModelIcon(config)}
                            <div>
                              <div style={{ fontWeight: 500 }}>{config.name}</div>
                              <div style={{ fontSize: 11, color: '#999' }}>
                                {config.model || 'AI模型'} • {config.api_endpoint ? '自定义地址' : '默认配置'}
                              </div>
                            </div>
                            {getModelStatusTag(config)}
                          </Space>
                        </Option>
                      );
                    })}
                  </Select>
                  <Tooltip title={isUserAdmin(getCurrentUser()) ? "配置自定义模型地址" : "只有管理员能配置AI模型"}>
                    <Button
                      type="text"
                      size="small"
                      icon={<EditOutlined />}
                      onClick={() => {
                        const currentUser = getCurrentUser();
                        if (!isUserAdmin(currentUser)) {
                          message.warning('只有管理员才能配置AI模型，请联系管理员');
                          return;
                        }
                        setShowModelConfig(true);
                        // 打开配置弹窗时刷新配置列表
                        fetchConfigs();
                      }}
                      style={{ color: isUserAdmin(getCurrentUser()) ? '#1890ff' : '#d9d9d9' }}
                    />
                  </Tooltip>
                </>
              ) : (
                <>
                  <span style={{ color: '#999', fontSize: 12 }}>
                    {loading ? '加载配置中...' : '暂无配置'}
                  </span>
                  <Tooltip title={isUserAdmin(getCurrentUser()) ? "配置自定义模型地址" : "只有管理员能配置AI模型"}>
                    <Button
                      type="text"
                      size="small"
                      icon={<EditOutlined />}
                      onClick={() => {
                        const currentUser = getCurrentUser();
                        if (!isUserAdmin(currentUser)) {
                          message.warning('只有管理员才能配置AI模型，请联系管理员');
                          return;
                        }
                        setShowModelConfig(true);
                        // 打开配置弹窗时刷新配置列表
                        fetchConfigs();
                      }}
                      style={{ color: isUserAdmin(getCurrentUser()) ? '#1890ff' : '#d9d9d9' }}
                    />
                  </Tooltip>
                </>
              )}
            </Space>
          </Space>
          <Space>
            <Tooltip title="锁定面板">
              <Button
                type="text"
                icon={locked ? <UnlockOutlined /> : <LockOutlined />}
                onClick={() => setLocked(!locked)}
                title={locked ? '解除锁定' : '锁定面板'}
                style={{
                  color: locked ? '#52c41a' : '#8c8c8c',
                }}
              />
            </Tooltip>
            <Tooltip title={
              <div>
                <div>面板宽度: {panelWidth}px</div>
                <div style={{ fontSize: 11, marginTop: 4, color: '#bfbfbf' }}>
                  Ctrl/Cmd + / - 调整宽度<br/>
                  Ctrl/Cmd + 0 重置宽度
                </div>
              </div>
            }>
              <Button
                type="text"
                icon={<ExpandOutlined />}
                style={{ color: '#8c8c8c', fontSize: 12 }}
                size="small"
              />
            </Tooltip>
            {/* 关闭按钮 - 始终显示 */}
            <Button
              type="text"
              onClick={() => setVisible(false)}
              style={{ fontSize: 18 }}
              title={locked ? '关闭面板（锁定状态）' : '关闭面板'}
            >
              ×
            </Button>
          </Space>
        </div>
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
                {isUserAdmin(getCurrentUser()) ? (
                  <Button 
                    type="primary" 
                    icon={<SettingOutlined />}
                    onClick={() => {
                      setVisible(false);
                      navigate('/admin/ai-assistant');
                    }}
                    size="large"
                  >
                    配置AI模型
                  </Button>
                ) : (
                  <Button 
                    type="primary" 
                    icon={<MessageOutlined />}
                    disabled
                    size="large"
                    title="联系管理员配置AI模型"
                  >
                    AI模型未配置
                  </Button>
                )}
                <Button 
                  type="default"
                  onClick={() => {
                    setVisible(false);
                    navigate(isUserAdmin(getCurrentUser()) ? '/admin' : '/dashboard');
                  }}
                >
                  {isUserAdmin(getCurrentUser()) ? '进入管理中心' : '返回主页面'}
                </Button>
              </Space>
            </div>
          </div>
        ) : (
          <>
            {/* 对话列表 */}
            <div style={{ borderBottom: isDark ? '1px solid #303030' : '1px solid #f0f0f0', maxHeight: 200, overflow: 'auto', background: isDark ? '#1f1f1f' : 'transparent' }}>
              <div style={{ padding: '12px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Text strong style={{ color: isDark ? '#fff' : undefined }}>对话历史</Text>
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
                        backgroundColor: currentConversation?.id === conversation.id 
                          ? (isDark ? '#162312' : '#f6ffed') 
                          : 'transparent',
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
                          <Text ellipsis style={{ fontSize: 12, color: isDark ? '#fff' : undefined }}>
                            {conversation.title}
                          </Text>
                        }
                        description={
                          <Text type="secondary" style={{ fontSize: 11, color: isDark ? '#8c8c8c' : undefined }}>
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
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', background: isDark ? '#141414' : 'transparent' }}>
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
                        <div style={{ color: isDark ? '#fff' : undefined }}>开始与AI对话吧！</div>
                        <Text type="secondary" style={{ fontSize: 12, color: isDark ? '#8c8c8c' : undefined }}>
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
                                backgroundColor: message.role === 'user' 
                                  ? (isDark ? '#111d2c' : '#e6f7ff')
                                  : message.isStatus 
                                    ? (isDark ? '#1f1f1f' : '#f0f2f5')
                                    : message.isError 
                                      ? (isDark ? '#2a1215' : '#fff2f0') 
                                      : (isDark ? '#162312' : '#f6ffed'),
                                border: message.isError 
                                  ? (isDark ? '1px solid #58181c' : '1px solid #ffccc7') 
                                  : (isDark ? '1px solid #303030' : undefined),
                              }}
                              bodyStyle={{ padding: 12 }}
                            >
                              <Space direction="vertical" style={{ width: '100%' }}>
                                <Space>
                                  <Avatar
                                    icon={message.role === 'user' ? <UserOutlined /> : 
                                          message.role === 'system' ? <SettingOutlined /> : 
                                          <AIRobotIcon size={16} animated={message.isProcessing || false} />}
                                    size="small"
                                  />
                                  <Text strong style={{ color: isDark ? '#fff' : undefined }}>
                                    {message.role === 'user' ? '我' : 
                                     message.role === 'system' ? '系统' : 'AI助手'}
                                  </Text>
                                  {message.isStatus && (
                                    <Spin size="small" />
                                  )}
                                  {message.status === 'processing' && (
                                    <Tag color="processing" style={{ fontSize: 10 }}>
                                      处理中
                                    </Tag>
                                  )}
                                  {message.status === 'timeout' && (
                                    <Tag color="warning" style={{ fontSize: 10 }}>
                                      超时
                                    </Tag>
                                  )}
                                  {message.status === 'failed' && (
                                    <Tag color="error" style={{ fontSize: 10 }}>
                                      失败
                                    </Tag>
                                  )}
                                  {message.status === 'stopped' && (
                                    <Tag color="default" style={{ fontSize: 10 }}>
                                      已停止
                                    </Tag>
                                  )}
                                  {message.status === 'completed' && (
                                    <Tag color="success" style={{ fontSize: 10 }}>
                                      完成
                                    </Tag>
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
                                  color: message.isError ? '#ff4d4f' : 
                                         message.isStatus ? '#1890ff' : 
                                         isDark ? '#d9d9d9' : 'inherit',
                                  fontStyle: message.isStatus ? 'italic' : 'normal',
                                }}>
                                  {message.content}
                                  {message.isProcessing && (
                                    <span style={{ marginLeft: 8 }}>
                                      <Spin size="small" />
                                    </span>
                                  )}
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
                    <Title level={4} style={{ color: isDark ? '#fff' : undefined }}>欢迎使用AI助手</Title>
                    <Text type="secondary" style={{ color: isDark ? '#8c8c8c' : undefined }}>选择一个对话开始聊天，或创建新对话</Text>
                  </div>
                </div>
              )}

              {/* 输入区域 */}
              <div style={{ padding: 16, borderTop: isDark ? '1px solid #303030' : '1px solid #f0f0f0', background: isDark ? '#1f1f1f' : 'transparent' }}>
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
                    // 显示停止按钮（增强版本）
                    <Button
                      type="primary"
                      danger
                      icon={<StopOutlined />}
                      onClick={() => {
                        console.log('🛑 用户点击停止按钮，处理消息ID:', processingMessageId);
                        stopMessage();
                      }}
                      loading={false}
                      style={{
                        backgroundColor: '#ff4d4f',
                        borderColor: '#ff4d4f',
                        boxShadow: '0 2px 8px rgba(255, 77, 79, 0.3)',
                      }}
                    >
                      停止
                    </Button>
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
                <div style={{ marginTop: 8, fontSize: 11, color: isDark ? '#8c8c8c' : '#999' }}>
                  {processingMessageId ? (
                    <Space>
                      <Spin size="small" />
                      <Text type="secondary" style={{ color: '#1890ff' }}>
                        AI正在处理中，点击停止按钮可中断...
                      </Text>
                    </Space>
                  ) : currentConversation ? (
                    <Text type="secondary" style={{ color: isDark ? '#8c8c8c' : undefined }}>当前对话：{currentConversation.title}</Text>
                  ) : (
                    <Text type="secondary" style={{ color: isDark ? '#8c8c8c' : undefined }}>快速模式：将自动创建新对话</Text>
                  )}
                </div>
              </div>
            </div>
          </>
        )}
      </div>

      {/* 模型配置弹窗 */}
      <Modal
        title={
          <Space>
            <ApiOutlined />
            <span style={{ color: isDark ? '#d9d9d9' : undefined }}>模型配置</span>
          </Space>
        }
        open={showModelConfig}
        onCancel={() => {
          setShowModelConfig(false);
          setCustomModelUrl('');
        }}
        onOk={() => {
          if (customModelUrl.trim() || customRestfulConfig.apiUrl.trim()) {
            saveCustomRestfulConfig();
          } else {
            setShowModelConfig(false);
          }
        }}
        width={720}
        className={`ai-model-config-modal ${isDark ? 'ai-model-config-modal-dark' : ''}`}
        styles={{
          header: isDark ? { background: '#1f1f1f', borderBottom: '1px solid #303030' } : {},
          body: isDark ? { background: '#1f1f1f', color: '#d9d9d9' } : {},
          footer: isDark ? { background: '#1f1f1f', borderTop: '1px solid #303030' } : {},
          content: isDark ? { background: '#1f1f1f' } : {},
          mask: isDark ? { background: 'rgba(0, 0, 0, 0.65)' } : {},
        }}
      >
        <div style={{ padding: '16px 0' }}>
          {/* 当前选择的模型信息 */}
          {selectedConfig && (
            <Card size="small" style={{ marginBottom: 16, backgroundColor: isDark ? '#1f1f1f' : undefined, borderColor: isDark ? '#434343' : undefined }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Space size={12}>
                  {getModelIcon(configs.find(c => c.id === selectedConfig))}
                  <div>
                    <div style={{ fontWeight: 500, marginBottom: 4, color: isDark ? '#d9d9d9' : undefined }}>
                      当前模型: {configs.find(c => c.id === selectedConfig)?.name}
                    </div>
                    <div style={{ fontSize: 12, color: isDark ? '#8c8c8c' : '#666' }}>
                      类型: {configs.find(c => c.id === selectedConfig)?.model_type || 'AI模型'}
                    </div>
                    {configs.find(c => c.id === selectedConfig)?.api_endpoint && (
                      <div style={{ fontSize: 12, color: isDark ? '#8c8c8c' : '#666' }}>
                        地址: {configs.find(c => c.id === selectedConfig)?.api_endpoint}
                      </div>
                    )}
                  </div>
                </Space>
                <div>
                  {getModelStatusTag(configs.find(c => c.id === selectedConfig))}
                </div>
              </div>
            </Card>
          )}

          <Divider orientation="left" orientationMargin="0">
            <span style={{ fontSize: 14, fontWeight: 500, color: isDark ? '#d9d9d9' : undefined }}>可用模型列表</span>
          </Divider>

          {/* 模型搜索框 */}
          <Input
            prefix={<MessageOutlined style={{ color: '#1890ff' }} />}
            placeholder="搜索模型名称、类型或API地址（忽略大小写）"
            value={modelSearchText}
            onChange={(e) => setModelSearchText(e.target.value)}
            allowClear
            size="small"
            style={{ marginBottom: 16, backgroundColor: isDark ? '#141414' : undefined, borderColor: isDark ? '#434343' : undefined, color: isDark ? '#d9d9d9' : undefined }}
          />

          {/* 模型列表 */}
          <div style={{ maxHeight: 300, overflowY: 'auto' }}>
            {(() => {
              const filteredConfigs = getFilteredConfigs();
              
              if (configs.length === 0) {
                return (
                  <div style={{ textAlign: 'center', padding: 20 }}>
                    <RobotOutlined style={{ fontSize: 32, color: '#d9d9d9', marginBottom: 8 }} />
                    <div style={{ color: isDark ? '#8c8c8c' : '#999' }}>暂无可用的AI模型配置</div>
                    <Button 
                      type="link" 
                      onClick={() => {
                        setShowModelConfig(false);
                        navigate('/admin/ai-configs');
                      }}
                    >
                      前往配置
                    </Button>
                  </div>
                );
              }
              
              if (filteredConfigs.length === 0 && modelSearchText.trim()) {
                return (
                  <div style={{ textAlign: 'center', padding: 20 }}>
                    <MessageOutlined style={{ fontSize: 32, color: isDark ? '#6b7280' : '#d9d9d9', marginBottom: 8 }} />
                    <div style={{ color: isDark ? '#8c8c8c' : '#999' }}>没有找到匹配 "{modelSearchText}" 的模型配置</div>
                    <Button 
                      type="link" 
                      onClick={() => setModelSearchText('')}
                      size="small"
                    >
                      清除搜索
                    </Button>
                  </div>
                );
              }
              
              return (
                <List
                  dataSource={filteredConfigs}
                  renderItem={(config) => (
                    <List.Item
                      style={{
                        padding: '12px 16px',
                        cursor: 'pointer',
                        borderRadius: 8,
                        marginBottom: 8,
                        border: isDark ? '1px solid #303030' : '1px solid #f0f0f0',
                        backgroundColor: config.id === selectedConfig 
                          ? (isDark ? '#162312' : '#f6ffed') 
                          : (isDark ? '#1f1f1f' : '#fafafa'),
                      }}
                      onClick={() => {
                        setSelectedConfig(config.id);
                        message.success(`已选择模型: ${config.name}`);
                      }}
                    >
                      <List.Item.Meta
                        avatar={<Avatar icon={getModelIcon(config)} />}
                        title={
                          <Space>
                            <span style={{ fontWeight: 500, color: isDark ? '#d9d9d9' : undefined }}>{config.name}</span>
                            {getModelStatusTag(config)}
                            {config.id === selectedConfig && (
                              <Tag color="green" size="small">当前使用</Tag>
                            )}
                          </Space>
                        }
                        description={
                          <div style={{ color: isDark ? '#8c8c8c' : undefined }}>
                            <div style={{ marginBottom: 4 }}>
                              类型: {config.model_type || 'AI模型'} • 提供商: {config.provider || '未知'}
                            </div>
                            {config.api_endpoint && (
                              <div style={{ fontSize: 12, color: isDark ? '#6b7280' : '#666' }}>
                                API地址: {config.api_endpoint}
                              </div>
                            )}
                            {config.description && (
                              <div style={{ fontSize: 12, color: isDark ? '#6b7280' : '#999' }}>
                                {config.description}
                              </div>
                            )}
                          </div>
                        }
                      />
                    </List.Item>
                  )}
                />
              );
            })()}
          </div>

          <Divider orientation="left" orientationMargin="0">
            <Space>
              <ApiOutlined style={{ color: isDark ? '#d9d9d9' : undefined }} />
              <span style={{ fontSize: 14, fontWeight: 500, color: isDark ? '#d9d9d9' : undefined }}>自定义RESTful接口配置</span>
            </Space>
          </Divider>

          {/* RESTful接口配置 */}
          <Collapse
            defaultActiveKey={['1']}
            size="small"
            items={[
              {
                key: '1',
                label: (
                  <Space>
                    <LinkOutlined style={{ color: isDark ? '#d9d9d9' : undefined }} />
                    <span style={{ color: isDark ? '#d9d9d9' : undefined }}>RESTful API配置</span>
                  </Space>
                ),
                children: (
                  <Form layout="vertical" size="small">
                    <Form.Item label="配置名称" required>
                      <Input
                        value={customRestfulConfig.name || ''}
                        onChange={(e) => setCustomRestfulConfig(prev => ({ ...prev, name: e.target.value }))}
                        placeholder="为您的自定义配置起个名字"
                        prefix={<EditOutlined />}
                      />
                    </Form.Item>

                    <Form.Item label="API地址" required>
                      <Input
                        value={customRestfulConfig.apiUrl}
                        onChange={(e) => setCustomRestfulConfig(prev => ({ ...prev, apiUrl: e.target.value }))}
                        placeholder="https://api.example.com/v1/chat/completions"
                        prefix={<LinkOutlined />}
                      />
                    </Form.Item>

                    <Form.Item label="请求方法">
                      <Radio.Group
                        value={customRestfulConfig.method}
                        onChange={(e) => setCustomRestfulConfig(prev => ({ ...prev, method: e.target.value }))}
                        size="small"
                      >
                        <Radio value="POST">POST</Radio>
                        <Radio value="GET">GET</Radio>
                        <Radio value="PUT">PUT</Radio>
                      </Radio.Group>
                    </Form.Item>

                    <Form.Item label="请求格式">
                      <Select
                        value={customRestfulConfig.requestFormat}
                        onChange={(value) => setCustomRestfulConfig(prev => ({ ...prev, requestFormat: value }))}
                        placeholder="选择请求格式"
                        style={{ width: '100%' }}
                      >
                        <Select.Option value="openai">OpenAI格式</Select.Option>
                        <Select.Option value="anthropic">Anthropic格式</Select.Option>
                        <Select.Option value="google">Google格式</Select.Option>
                        <Select.Option value="custom">自定义格式</Select.Option>
                      </Select>
                    </Form.Item>

                    <Form.Item label="认证方式">
                      <Select
                        value={customRestfulConfig.authType}
                        onChange={(value) => setCustomRestfulConfig(prev => ({ ...prev, authType: value }))}
                        placeholder="选择认证方式"
                        style={{ width: '100%' }}
                      >
                        <Select.Option value="bearer">Bearer Token</Select.Option>
                        <Select.Option value="apikey">API Key</Select.Option>
                        <Select.Option value="basic">Basic Auth</Select.Option>
                        <Select.Option value="none">无认证</Select.Option>
                      </Select>
                    </Form.Item>

                    {customRestfulConfig.authType !== 'none' && (
                      <Form.Item label="认证信息">
                        <Input.Password
                          value={customRestfulConfig.authValue}
                          onChange={(e) => setCustomRestfulConfig(prev => ({ ...prev, authValue: e.target.value }))}
                          placeholder={`请输入${customRestfulConfig.authType === 'bearer' ? 'Bearer Token' : 
                                     customRestfulConfig.authType === 'apikey' ? 'API Key' : 'Basic Auth'}`}
                          prefix={<KeyOutlined />}
                        />
                      </Form.Item>
                    )}

                    <Form.Item label="自定义请求头 (JSON格式)" help="例: {&quot;Content-Type&quot;: &quot;application/json&quot;}">
                      <Input.TextArea
                        value={JSON.stringify(customRestfulConfig.headers, null, 2)}
                        onChange={(e) => {
                          try {
                            const headers = JSON.parse(e.target.value || '{}');
                            setCustomRestfulConfig(prev => ({ ...prev, headers }));
                          } catch (error) {
                            // 忽略JSON解析错误，继续编辑
                          }
                        }}
                        placeholder='{"Authorization": "Bearer your-token"}'
                        rows={3}
                      />
                    </Form.Item>
                  </Form>
                ),
              },
            ]}
          />
        </div>
      </Modal>
    </>
  );
};

export default AIAssistantFloat;