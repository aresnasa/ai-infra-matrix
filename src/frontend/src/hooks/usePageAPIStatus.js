import { useState, useEffect, useCallback } from 'react';
import { useAPIHealth } from './useAPIHealth';

/**
 * 页面级API状态管理Hook
 * 专门用于管理页面初始化时的数据加载
 */
export const usePageAPIStatus = (apiCalls = [], dependencies = []) => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [data, setData] = useState(null);
  const [lastUpdate, setLastUpdate] = useState(null);
  
  const { isHealthy, checkAPIHealth, lastCheck } = useAPIHealth();

  // 执行API调用
  const executeAPICalls = useCallback(async () => {
    if (!apiCalls.length) {
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      // 直接执行API调用，不做预先的健康检查
      // 因为健康检查会增加不必要的复杂性
      const results = await Promise.all(
        apiCalls.map(apiCall => apiCall())
      );

      setData(results);
      setLastUpdate(new Date());
    } catch (err) {
      console.error('API calls failed:', err);
      setError({
        message: err.message || '数据加载失败',
        type: 'api_error',
        timestamp: new Date(),
        isNetworkError: !navigator.onLine || err.name === 'NetworkError'
      });
    } finally {
      setLoading(false);
    }
  }, [apiCalls, checkAPIHealth]);

  // 重试功能
  const retry = useCallback(() => {
    executeAPICalls();
  }, [executeAPICalls]);

  // 刷新数据
  const refresh = useCallback(() => {
    executeAPICalls();
  }, [executeAPICalls]);

  // 初始化和依赖项变化时重新加载
  useEffect(() => {
    executeAPICalls();
  }, dependencies);

  return {
    loading,
    error,
    data,
    retry,
    refresh,
    lastUpdate,
    isHealthy,
    lastCheck
  };
};

export default usePageAPIStatus;
