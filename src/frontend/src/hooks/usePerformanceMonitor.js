import { useState, useEffect, useRef } from 'react';

/**
 * 性能监控Hook
 * @param {boolean} enabled - 是否启用性能监控
 * @returns {Object} 性能监控工具
 */
export const usePerformanceMonitor = (enabled = false) => {
  const [metrics, setMetrics] = useState({
    loadTime: 0,
    renderTime: 0,
    memoryUsage: 0,
    networkRequests: 0
  });

  const startTimeRef = useRef(performance.now());
  const renderCountRef = useRef(0);

  useEffect(() => {
    if (!enabled) return;

    // 记录页面加载时间
    const loadTime = performance.now() - startTimeRef.current;
    setMetrics(prev => ({ ...prev, loadTime }));

    // 监控内存使用情况
    const updateMemoryUsage = () => {
      if (performance.memory) {
        setMetrics(prev => ({
          ...prev,
          memoryUsage: performance.memory.usedJSHeapSize
        }));
      }
    };

    // 监控渲染性能
    const observer = new PerformanceObserver((list) => {
      const entries = list.getEntries();
      entries.forEach((entry) => {
        if (entry.entryType === 'measure') {
          setMetrics(prev => ({
            ...prev,
            renderTime: entry.duration
          }));
        }
      });
    });

    observer.observe({ entryTypes: ['measure'] });

    // 定期更新内存使用情况
    const memoryInterval = setInterval(updateMemoryUsage, 5000);

    return () => {
      observer.disconnect();
      clearInterval(memoryInterval);
    };
  }, [enabled]);

  // 记录渲染次数
  useEffect(() => {
    renderCountRef.current += 1;
  });

  const startMeasure = (name) => {
    if (!enabled) return;
    performance.mark(`${name}-start`);
  };

  const endMeasure = (name) => {
    if (!enabled) return;
    performance.mark(`${name}-end`);
    performance.measure(name, `${name}-start`, `${name}-end`);
  };

  const logMetrics = () => {
    if (!enabled) return;
    console.log('Performance Metrics:', {
      ...metrics,
      renderCount: renderCountRef.current,
      timestamp: new Date().toISOString()
    });
  };

  return {
    metrics,
    startMeasure,
    endMeasure,
    logMetrics,
    renderCount: renderCountRef.current
  };
};