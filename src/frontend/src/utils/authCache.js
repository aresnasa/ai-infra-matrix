/**
 * 认证缓存管理器
 * 减少频繁的API调用，提供智能缓存机制
 */
class AuthCache {
  constructor() {
    this.cache = new Map();
    this.requestQueue = new Map();
    this.lastAuthCheck = null;
    this.AUTH_CACHE_DURATION = 5 * 60 * 1000; // 5分钟缓存
    this.TOKEN_REFRESH_BUFFER = 10 * 60 * 1000; // 10分钟缓冲
  }

  // 检查token是否需要刷新（本地检查）
  isTokenNearExpiry() {
    const expires_at = localStorage.getItem('token_expires');
    if (!expires_at) return true;
    
    const expiryTime = new Date(expires_at).getTime();
    const currentTime = new Date().getTime();
    
    return currentTime + this.TOKEN_REFRESH_BUFFER >= expiryTime;
  }

  // 检查是否需要重新验证用户（避免频繁API调用）
  shouldRefreshAuth() {
    if (!this.lastAuthCheck) return true;
    
    const timeSinceLastCheck = Date.now() - this.lastAuthCheck;
    return timeSinceLastCheck > this.AUTH_CACHE_DURATION;
  }

  // 获取缓存的用户信息
  getCachedUser() {
    try {
      const savedUser = localStorage.getItem('user');
      const cacheTime = localStorage.getItem('user_cache_time');
      
      if (!savedUser || !cacheTime) return null;
      
      const cacheAge = Date.now() - parseInt(cacheTime);
      if (cacheAge > this.AUTH_CACHE_DURATION) {
        // 缓存过期
        this.clearUserCache();
        return null;
      }
      
      return JSON.parse(savedUser);
    } catch (error) {
      console.warn('Failed to parse cached user:', error);
      this.clearUserCache();
      return null;
    }
  }

  // 缓存用户信息
  cacheUser(userData) {
    try {
      localStorage.setItem('user', JSON.stringify(userData));
      localStorage.setItem('user_cache_time', Date.now().toString());
      this.lastAuthCheck = Date.now();
    } catch (error) {
      console.warn('Failed to cache user data:', error);
    }
  }

  // 清除用户缓存
  clearUserCache() {
    localStorage.removeItem('user');
    localStorage.removeItem('user_cache_time');
    this.lastAuthCheck = null;
  }

  // 防重复请求机制
  async withDeduplication(key, asyncFunction) {
    // 如果已有相同请求在进行中，等待结果
    if (this.requestQueue.has(key)) {
      return await this.requestQueue.get(key);
    }

    // 创建新的请求Promise
    const requestPromise = asyncFunction();
    this.requestQueue.set(key, requestPromise);

    try {
      const result = await requestPromise;
      return result;
    } finally {
      // 清理请求队列
      this.requestQueue.delete(key);
    }
  }

  // 清理所有缓存
  clearAll() {
    this.cache.clear();
    this.requestQueue.clear();
    this.clearUserCache();
    localStorage.removeItem('token');
    localStorage.removeItem('token_expires');
  }
}

// 导出单例实例
export const authCache = new AuthCache();
export default authCache;
