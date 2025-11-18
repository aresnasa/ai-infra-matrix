/**
 * API请求去重和缓存管理器
 * 防止重复API调用，减少对后端的压力
 */
class APIRequestManager {
  constructor() {
    this.pendingRequests = new Map();
    this.responseCache = new Map();
    this.CACHE_DURATION = 2 * 60 * 1000; // 2分钟缓存
    this.MAX_CACHE_SIZE = 100;
  }

  // 生成请求的缓存键
  generateCacheKey(url, method = 'GET', params = {}) {
    const sortedParams = Object.keys(params)
      .sort()
      .map(key => `${key}=${params[key]}`)
      .join('&');
    return `${method}:${url}${sortedParams ? '?' + sortedParams : ''}`;
  }

  // 检查缓存是否有效
  isCacheValid(cacheEntry) {
    if (!cacheEntry) return false;
    const age = Date.now() - cacheEntry.timestamp;
    return age < this.CACHE_DURATION;
  }

  // 清理过期缓存
  cleanExpiredCache() {
    const now = Date.now();
    for (const [key, entry] of this.responseCache.entries()) {
      if (now - entry.timestamp > this.CACHE_DURATION) {
        this.responseCache.delete(key);
      }
    }
  }

  // 限制缓存大小
  limitCacheSize() {
    if (this.responseCache.size > this.MAX_CACHE_SIZE) {
      // 删除最旧的缓存条目
      const entries = Array.from(this.responseCache.entries());
      entries.sort((a, b) => a[1].timestamp - b[1].timestamp);
      
      const toDelete = entries.slice(0, entries.length - this.MAX_CACHE_SIZE + 10);
      toDelete.forEach(([key]) => this.responseCache.delete(key));
    }
  }

  // 包装API请求，提供去重和缓存
  async wrapRequest(requestFn, cacheKey, enableCache = true) {
    // 检查缓存
    if (enableCache) {
      const cachedResponse = this.responseCache.get(cacheKey);
      if (this.isCacheValid(cachedResponse)) {
        console.debug(`Using cached response for: ${cacheKey}`);
        return cachedResponse.data;
      }
    }

    // 检查是否有相同请求正在进行
    if (this.pendingRequests.has(cacheKey)) {
      console.debug(`Deduplicating request: ${cacheKey}`);
      return await this.pendingRequests.get(cacheKey);
    }

    // 创建新请求
    const requestPromise = this.executeRequest(requestFn, cacheKey, enableCache);
    this.pendingRequests.set(cacheKey, requestPromise);

    try {
      return await requestPromise;
    } finally {
      this.pendingRequests.delete(cacheKey);
    }
  }

  // 执行实际请求
  async executeRequest(requestFn, cacheKey, enableCache) {
    try {
      const response = await requestFn();
      
      // 缓存响应
      if (enableCache && response) {
        this.responseCache.set(cacheKey, {
          data: response,
          timestamp: Date.now()
        });
        
        // 定期清理缓存
        this.cleanExpiredCache();
        this.limitCacheSize();
      }
      
      return response;
    } catch (error) {
      // 不缓存错误响应
      throw error;
    }
  }

  // 清除特定缓存
  clearCache(pattern) {
    if (pattern) {
      // 清除匹配模式的缓存
      for (const key of this.responseCache.keys()) {
        if (key.includes(pattern)) {
          this.responseCache.delete(key);
        }
      }
    } else {
      // 清除所有缓存
      this.responseCache.clear();
    }
  }

  // 清除所有待请求
  clearPendingRequests() {
    this.pendingRequests.clear();
  }

  // 获取缓存统计信息
  getCacheStats() {
    return {
      cacheSize: this.responseCache.size,
      pendingRequests: this.pendingRequests.size,
      cacheHitRate: this.calculateHitRate()
    };
  }

  // 计算缓存命中率（简化版本）
  calculateHitRate() {
    // 这里可以实现更复杂的统计逻辑
    return 'N/A';
  }
}

// 导出单例实例
export const apiRequestManager = new APIRequestManager();
export default apiRequestManager;
