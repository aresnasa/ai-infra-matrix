import { useState, useEffect, useCallback, useRef } from 'react';
import { message } from 'antd';
import { authAPI } from '../services/api';
import { authCache } from '../utils/authCache';

/**
 * API健康检查和状态监控钩子
 * 优化版本 - 减少频繁检查，智能缓存
 */
export const useAPIHealth = (options = {}) => {
  const {
    checkInterval = 5 * 60 * 1000, // 5分钟检查一次，大幅减少频率
    enableAutoCheck = true,
    showNotifications = false, // 默认关闭通知，减少干扰
    onlyCheckOnFocus = true // 只在窗口获得焦点时检查
  } = options;

  const [apiHealth, setApiHealth] = useState({
    status: 'unknown', // 'healthy', 'degraded', 'down', 'unknown'
    lastCheck: null,
    errors: [],
    responseTime: null
  });

  const [isChecking, setIsChecking] = useState(false);
  const intervalRef = useRef(null);
  const lastCheckRef = useRef(null);
  const notificationShownRef = useRef(false);

  // 检查是否需要进行健康检查
  const shouldCheck = useCallback(() => {
    if (!enableAutoCheck) return false;
    if (isChecking) return false;
    
    // 如果设置了只在焦点时检查，且窗口没有焦点，则跳过
    if (onlyCheckOnFocus && document.hidden) return false;
    
    // 检查上次检查时间，避免过于频繁
    if (lastCheckRef.current) {
      const timeSinceLastCheck = Date.now() - lastCheckRef.current;
      if (timeSinceLastCheck < checkInterval) return false;
    }
    
    return true;
  }, [enableAutoCheck, isChecking, onlyCheckOnFocus, checkInterval]);

  const checkAPIHealth = useCallback(async () => {
    if (!shouldCheck()) return;
    
    setIsChecking(true);
    lastCheckRef.current = Date.now();
    const startTime = Date.now();
    
    try {
      // 使用缓存的用户信息，避免频繁API调用
      const cachedUser = authCache.getCachedUser();
      if (cachedUser && !authCache.shouldRefreshAuth()) {
        // 使用缓存数据，跳过API调用
        const responseTime = 50; // 模拟快速响应
        
        setApiHealth(prev => ({
          status: 'healthy',
          lastCheck: new Date(),
          errors: [],
          responseTime
        }));
        
        setIsChecking(false);
        return;
      }
      
      // 真正的API检查
      await authAPI.getCurrentUser();
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
      if (prevStatus === 'down' && showNotifications && !notificationShownRef.current) {
        message.success('后端服务已恢复正常');
        notificationShownRef.current = true;
        // 5秒后重置通知状态
        setTimeout(() => {
          notificationShownRef.current = false;
        }, 5000);
      }
      
    } catch (error) {
      const responseTime = Date.now() - startTime;
      const prevStatus = apiHealth.status;
      
      console.warn('API健康检查失败:', error.message);
      
      setApiHealth(prev => ({
        status: 'down',
        lastCheck: new Date(),
        errors: [error.message],
        responseTime
      }));

      // 只在状态从正常变为异常时显示通知
      if (prevStatus === 'healthy' && showNotifications && !notificationShownRef.current) {
        message.warning('后端服务连接异常');
        notificationShownRef.current = true;
        // 30秒后重置通知状态
        setTimeout(() => {
          notificationShownRef.current = false;
        }, 30000);
      }
    } finally {
      setIsChecking(false);
    }
  }, [shouldCheck, apiHealth.status, showNotifications]);

  // 监听窗口焦点变化，在获得焦点时检查
  useEffect(() => {
    if (!onlyCheckOnFocus) return;
    
    const handleFocus = () => {
      // 延迟检查，避免频繁切换
      setTimeout(checkAPIHealth, 1000);
    };
    
    const handleVisibilityChange = () => {
      if (!document.hidden) {
        setTimeout(checkAPIHealth, 1000);
      }
    };
    
    window.addEventListener('focus', handleFocus);
    document.addEventListener('visibilitychange', handleVisibilityChange);
    
    return () => {
      window.removeEventListener('focus', handleFocus);
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [checkAPIHealth, onlyCheckOnFocus]);

  // 设置定时检查
  useEffect(() => {
    if (!enableAutoCheck) return;
    
    // 立即执行一次检查
    checkAPIHealth();
    
    // 设置定时器
    intervalRef.current = setInterval(checkAPIHealth, checkInterval);
    
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [enableAutoCheck, checkInterval, checkAPIHealth]);

  // 手动触发健康检查
  const manualCheck = useCallback(() => {
    lastCheckRef.current = null; // 重置检查时间，强制执行
    checkAPIHealth();
  }, [checkAPIHealth]);

  // 计算状态
  const isHealthy = apiHealth.status === 'healthy';
  const isDegraded = apiHealth.status === 'degraded';
  const isDown = apiHealth.status === 'down';

  return {
    apiHealth,
    isHealthy,
    isDegraded,
    isDown,
    isChecking,
    manualCheck
  };
};
