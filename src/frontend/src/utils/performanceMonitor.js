import { useEffect, useRef } from 'react';

/**
 * 性能监控和优化工具
 */
class PerformanceMonitor {
  constructor() {
    this.metrics = {
      apiCalls: 0,
      slowApiCalls: 0,
      memoryUsage: [],
      renderTimes: [],
      errorCount: 0
    };
    
    this.observers = [];
    this.isMonitoring = false;
    this.logInterval = null;
  }

  // 开始监控
  startMonitoring() {
    if (this.isMonitoring) return;
    
    this.isMonitoring = true;
    
    // 监控API调用
    this.interceptXHR();
    this.interceptFetch();
    
    // 监控内存使用
    this.startMemoryMonitoring();
    
    // 监控渲染性能
    this.startRenderMonitoring();
    
    // 定期输出性能报告
    this.logInterval = setInterval(() => {
      this.logPerformanceReport();
    }, 60000); // 每分钟输出一次
    
    console.log('🚀 Performance monitoring started');
  }

  // 停止监控
  stopMonitoring() {
    if (!this.isMonitoring) return;
    
    this.isMonitoring = false;
    
    // 清理监控
    this.observers.forEach(observer => observer.disconnect());
    this.observers = [];
    
    if (this.logInterval) {
      clearInterval(this.logInterval);
      this.logInterval = null;
    }
    
    console.log('📊 Performance monitoring stopped');
  }

  // 拦截XMLHttpRequest
  interceptXHR() {
    const originalOpen = XMLHttpRequest.prototype.open;
    const originalSend = XMLHttpRequest.prototype.send;
    
    XMLHttpRequest.prototype.open = function(method, url, ...args) {
      this._startTime = Date.now();
      this._method = method;
      this._url = url;
      return originalOpen.apply(this, [method, url, ...args]);
    };
    
    XMLHttpRequest.prototype.send = function(...args) {
      this.addEventListener('loadend', () => {
        const responseTime = Date.now() - this._startTime;
        performanceMonitor.recordApiCall(this._method, this._url, responseTime, this.status);
      });
      
      return originalSend.apply(this, args);
    };
  }

  // 拦截fetch
  interceptFetch() {
    const originalFetch = window.fetch;
    
    window.fetch = async function(url, options = {}) {
      const startTime = Date.now();
      const method = options.method || 'GET';
      
      try {
        const response = await originalFetch(url, options);
        const responseTime = Date.now() - startTime;
        
        performanceMonitor.recordApiCall(method, url, responseTime, response.status);
        
        return response;
      } catch (error) {
        const responseTime = Date.now() - startTime;
        performanceMonitor.recordApiCall(method, url, responseTime, 0, error);
        throw error;
      }
    };
  }

  // 记录API调用
  recordApiCall(method, url, responseTime, status, error = null) {
    this.metrics.apiCalls++;
    
    if (responseTime > 3000) { // 超过3秒算慢请求
      this.metrics.slowApiCalls++;
      console.warn(`🐌 Slow API call: ${method} ${url} took ${responseTime}ms`);
    }
    
    if (error || status >= 400) {
      this.metrics.errorCount++;
      console.error(`❌ API error: ${method} ${url} - Status: ${status}`, error);
    }
  }

  // 监控内存使用
  startMemoryMonitoring() {
    if (!performance.memory) return;
    
    const checkMemory = () => {
      if (this.isMonitoring) {
        const memory = {
          used: performance.memory.usedJSHeapSize,
          total: performance.memory.totalJSHeapSize,
          limit: performance.memory.jsHeapSizeLimit,
          timestamp: Date.now()
        };
        
        this.metrics.memoryUsage.push(memory);
        
        // 只保留最近10分钟的数据
        const tenMinutesAgo = Date.now() - 10 * 60 * 1000;
        this.metrics.memoryUsage = this.metrics.memoryUsage.filter(
          m => m.timestamp > tenMinutesAgo
        );
        
        // 检查内存使用过高
        const usagePercent = (memory.used / memory.limit) * 100;
        if (usagePercent > 80) {
          console.warn(`🧠 High memory usage: ${usagePercent.toFixed(1)}%`);
        }
        
        setTimeout(checkMemory, 30000); // 每30秒检查一次
      }
    };
    
    checkMemory();
  }

  // 监控渲染性能
  startRenderMonitoring() {
    if (!window.PerformanceObserver) return;
    
    try {
      // 监控首次内容绘制
      const paintObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries();
        entries.forEach(entry => {
          if (entry.name === 'first-contentful-paint') {
            console.log(`🎨 First Contentful Paint: ${entry.startTime.toFixed(2)}ms`);
          }
        });
      });
      
      paintObserver.observe({ entryTypes: ['paint'] });
      this.observers.push(paintObserver);
      
      // 监控长任务
      const longTaskObserver = new PerformanceObserver((list) => {
        const entries = list.getEntries();
        entries.forEach(entry => {
          if (entry.duration > 50) { // 超过50ms的任务
            console.warn(`⏱️ Long task detected: ${entry.duration.toFixed(2)}ms`);
          }
        });
      });
      
      longTaskObserver.observe({ entryTypes: ['longtask'] });
      this.observers.push(longTaskObserver);
      
    } catch (error) {
      console.debug('Performance Observer not fully supported:', error);
    }
  }

  // 输出性能报告
  logPerformanceReport() {
    const report = {
      timestamp: new Date().toISOString(),
      apiCalls: this.metrics.apiCalls,
      slowApiCalls: this.metrics.slowApiCalls,
      errorCount: this.metrics.errorCount,
      memoryUsage: this.getLatestMemoryUsage(),
      performance: this.getPerformanceMetrics()
    };
    
    console.group('📊 Performance Report');
    console.table(report);
    console.groupEnd();
    
    // 重置计数器
    this.metrics.apiCalls = 0;
    this.metrics.slowApiCalls = 0;
    this.metrics.errorCount = 0;
  }

  // 获取最新内存使用情况
  getLatestMemoryUsage() {
    if (!this.metrics.memoryUsage.length) return null;
    
    const latest = this.metrics.memoryUsage[this.metrics.memoryUsage.length - 1];
    return {
      used: `${(latest.used / 1024 / 1024).toFixed(2)}MB`,
      total: `${(latest.total / 1024 / 1024).toFixed(2)}MB`,
      usagePercent: `${((latest.used / latest.limit) * 100).toFixed(1)}%`
    };
  }

  // 获取性能指标
  getPerformanceMetrics() {
    if (!performance.getEntriesByType) return null;
    
    const navigation = performance.getEntriesByType('navigation')[0];
    if (!navigation) return null;
    
    return {
      domContentLoaded: `${navigation.domContentLoadedEventEnd.toFixed(2)}ms`,
      loadComplete: `${navigation.loadEventEnd.toFixed(2)}ms`,
      firstByte: `${navigation.responseStart.toFixed(2)}ms`
    };
  }

  // 强制垃圾回收（如果支持）
  forceGC() {
    if (window.gc) {
      window.gc();
      console.log('🗑️ Forced garbage collection');
    } else {
      console.log('Garbage collection not available in this environment');
    }
  }
}

// 全局性能监控实例
const performanceMonitor = new PerformanceMonitor();

// React Hook
export const usePerformanceMonitor = (enabled = true) => {
  const mountedRef = useRef(false);
  
  useEffect(() => {
    if (!enabled) return;
    
    mountedRef.current = true;
    
    // 延迟启动监控，避免影响初始渲染
    const timer = setTimeout(() => {
      if (mountedRef.current) {
        performanceMonitor.startMonitoring();
      }
    }, 2000);
    
    return () => {
      clearTimeout(timer);
      mountedRef.current = false;
      if (enabled) {
        performanceMonitor.stopMonitoring();
      }
    };
  }, [enabled]);
  
  return {
    forceGC: () => performanceMonitor.forceGC(),
    getReport: () => performanceMonitor.logPerformanceReport()
  };
};

export default performanceMonitor;
