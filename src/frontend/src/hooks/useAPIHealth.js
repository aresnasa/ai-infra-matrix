import { useState, useEffect, useCallback } from 'react';
import { message } from 'antd';
import { authAPI } from '../services/api';

/**
 * API健康检查和状态监控钩子
 * 用于检测后端服务状态和及时发现变化
 */
export const useAPIHealth = (options = {}) => {
  const {
    checkInterval = 60000, // 60秒检查一次，减少对后端的压力
    enableAutoCheck = true,
    showNotifications = true
  } = options;

  const [apiHealth, setApiHealth] = useState({
    status: 'unknown', // 'healthy', 'degraded', 'down', 'unknown'
    lastCheck: null,
    errors: [],
    responseTime: null
  });

  const [isChecking, setIsChecking] = useState(false);

  const checkAPIHealth = useCallback(async () => {
    if (isChecking) return;
    
    setIsChecking(true);
    const startTime = Date.now();
    
    try {
      // 使用当前用户API作为健康检查端点
      const response = await authAPI.getCurrentUser();
      const responseTime = Date.now() - startTime;
      
      // 保存之前的状态用于通知
      const prevStatus = apiHealth.status;
      
      setApiHealth(prev => ({
        status: 'healthy',
        lastCheck: new Date(),
        errors: [],
        responseTime
      }));

      // 如果之前是错误状态，现在恢复了，显示通知
      if (prevStatus === 'down' && showNotifications) {
        message.success('后端服务已恢复正常');
      }
      
    } catch (error) {
      const responseTime = Date.now() - startTime;
      const errorInfo = {
        timestamp: new Date(),
        message: error.message,
        status: error.response?.status,
        data: error.response?.data
      };

      setApiHealth(prev => ({
        status: error.response?.status ? 'degraded' : 'down',
        lastCheck: new Date(),
        errors: [...prev.errors.slice(-4), errorInfo], // 保留最近5个错误
        responseTime
      }));

      // 显示错误通知
      if (showNotifications) {
        if (error.response?.status === 401) {
          message.warning('登录状态已过期，请重新登录');
        } else if (error.response?.status >= 500) {
          message.error('后端服务出现问题，请稍后重试');
        } else if (!error.response) {
          message.error('无法连接到后端服务');
        }
      }

      console.error('API健康检查失败:', error);
    } finally {
      setIsChecking(false);
    }
  }, [isChecking, showNotifications]);

  // 自动健康检查
  useEffect(() => {
    if (!enableAutoCheck) return;

    // 立即执行一次检查
    checkAPIHealth();

    // 设置定期检查
    const interval = setInterval(checkAPIHealth, checkInterval);

    return () => clearInterval(interval);
  }, [checkAPIHealth, checkInterval, enableAutoCheck]);

  // 监听网络状态变化
  useEffect(() => {
    const handleOnline = () => {
      console.log('网络已连接，执行健康检查');
      checkAPIHealth();
    };

    const handleOffline = () => {
      setApiHealth(prev => ({
        ...prev,
        status: 'down',
        errors: [...prev.errors, {
          timestamp: new Date(),
          message: '网络连接断开',
          status: null,
          data: null
        }]
      }));
      if (showNotifications) {
        message.warning('网络连接已断开');
      }
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, [checkAPIHealth, showNotifications]);

  return {
    apiHealth,
    isChecking,
    checkAPIHealth,
    isHealthy: apiHealth.status === 'healthy',
    isDegraded: apiHealth.status === 'degraded',
    isDown: apiHealth.status === 'down'
  };
};

/**
 * 页面级API状态钩子
 * 为单个页面提供API状态监控
 */
export const usePageAPIStatus = (apiCalls = [], dependencies = []) => {
  const [pageStatus, setPageStatus] = useState({
    loading: true,
    error: null,
    data: null,
    lastUpdate: null
  });

  const [retryCount, setRetryCount] = useState(0);

  const loadPageData = useCallback(async () => {
    if (apiCalls.length === 0) {
      setPageStatus({
        loading: false,
        error: null,
        data: null,
        lastUpdate: new Date()
      });
      return;
    }

    setPageStatus(prev => ({ ...prev, loading: true, error: null }));
    
    try {
      const results = await Promise.allSettled(apiCalls.map(call => call()));
      
      const errors = results
        .filter(result => result.status === 'rejected')
        .map(result => result.reason);
      
      const data = results
        .filter(result => result.status === 'fulfilled')
        .map(result => result.value);

      if (errors.length > 0) {
        // 如果有错误但也有成功的请求，标记为部分成功
        setPageStatus({
          loading: false,
          error: errors.length === results.length ? errors[0] : null,
          data: data.length > 0 ? data : null,
          lastUpdate: new Date(),
          partialErrors: errors.length < results.length ? errors : null
        });
      } else {
        setPageStatus({
          loading: false,
          error: null,
          data,
          lastUpdate: new Date()
        });
      }
    } catch (error) {
      setPageStatus({
        loading: false,
        error,
        data: null,
        lastUpdate: new Date()
      });
    }
  }, [apiCalls, retryCount]);

  const retry = useCallback(() => {
    setRetryCount(prev => prev + 1);
  }, []);

  useEffect(() => {
    loadPageData();
  }, [...dependencies, retryCount]);

  return {
    ...pageStatus,
    retry,
    retryCount,
    refresh: loadPageData
  };
};
