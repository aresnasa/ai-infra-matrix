/**
 * 动态Favicon管理器
 * 根据当前页面路由自动切换对应的图标
 */

class FaviconManager {
  constructor() {
    this.defaultIcon = '/favicon.ico';
    this.iconConfig = null;
    this.currentIcon = this.defaultIcon;
    this.init();
  }

  async init() {
    try {
      // 加载图标配置
      const response = await fetch('/favicon-config.json');
      this.iconConfig = await response.json();
      
      // 设置默认图标
      this.setIcon(this.defaultIcon);
      
      // 监听路由变化
      this.setupRouteListener();
      
      console.log('🎨 FaviconManager initialized');
    } catch (error) {
      console.warn('Favicon config loading failed:', error);
      this.setIcon(this.defaultIcon);
    }
  }

  /**
   * 设置favicon图标
   * @param {string} iconPath - 图标路径
   */
  setIcon(iconPath) {
    if (this.currentIcon === iconPath) return;

    // 移除现有的favicon
    const existingLinks = document.querySelectorAll('link[rel*="icon"]');
    existingLinks.forEach(link => link.remove());

    // 添加新的favicon
    const link = document.createElement('link');
    link.rel = 'icon';
    link.type = this.getIconType(iconPath);
    link.href = iconPath;
    
    document.head.appendChild(link);
    this.currentIcon = iconPath;
    
    console.log(`🎯 Favicon updated: ${iconPath}`);
  }

  /**
   * 根据文件扩展名获取MIME类型
   * @param {string} iconPath - 图标路径
   * @returns {string} - MIME类型
   */
  getIconType(iconPath) {
    if (iconPath.endsWith('.svg')) return 'image/svg+xml';
    if (iconPath.endsWith('.png')) return 'image/png';
    if (iconPath.endsWith('.ico')) return 'image/x-icon';
    return 'image/x-icon';
  }

  /**
   * 根据路由获取对应图标
   * @param {string} pathname - 当前路径
   * @returns {string} - 图标路径
   */
  getIconForRoute(pathname) {
    if (!this.iconConfig) return this.defaultIcon;

    // 精确匹配路由
    if (this.iconConfig.routes[pathname]) {
      return this.iconConfig.routes[pathname];
    }

    // 模糊匹配
    for (const [route, icon] of Object.entries(this.iconConfig.routes)) {
      if (pathname.startsWith(route)) {
        return icon;
      }
    }

    // 检查特殊页面标识
    if (pathname.includes('jupyter') || pathname.includes('notebook')) {
      return this.iconConfig.pages?.jupyter || this.defaultIcon;
    }
    
    if (pathname.includes('admin')) {
      return this.iconConfig.pages?.admin || this.defaultIcon;
    }
    
    if (pathname.includes('kubernetes') || pathname.includes('k8s')) {
      return this.iconConfig.pages?.kubernetes || this.defaultIcon;
    }
    
    if (pathname.includes('ansible')) {
      return this.iconConfig.pages?.ansible || this.defaultIcon;
    }

    return this.defaultIcon;
  }

  /**
   * 更新当前页面的favicon
   */
  updateFavicon() {
    const pathname = window.location.pathname;
    const newIcon = this.getIconForRoute(pathname);
    this.setIcon(newIcon);
  }

  /**
   * 设置路由监听器
   */
  setupRouteListener() {
    // 监听页面加载
    this.updateFavicon();

    // 监听路由变化（用于SPA应用）
    const originalPushState = history.pushState;
    const originalReplaceState = history.replaceState;

    history.pushState = (...args) => {
      originalPushState.apply(history, args);
      setTimeout(() => this.updateFavicon(), 100);
    };

    history.replaceState = (...args) => {
      originalReplaceState.apply(history, args);
      setTimeout(() => this.updateFavicon(), 100);
    };

    // 监听popstate事件
    window.addEventListener('popstate', () => {
      setTimeout(() => this.updateFavicon(), 100);
    });

    // 监听hashchange事件
    window.addEventListener('hashchange', () => {
      setTimeout(() => this.updateFavicon(), 100);
    });

    // 为了兼容React Router，也监听URL变化
    let lastUrl = location.href;
    new MutationObserver(() => {
      const url = location.href;
      if (url !== lastUrl) {
        lastUrl = url;
        setTimeout(() => this.updateFavicon(), 100);
      }
    }).observe(document, { subtree: true, childList: true });
  }

  /**
   * 手动设置特定页面的图标
   * @param {string} pageType - 页面类型 (jupyter/admin/kubernetes/ansible)
   */
  setPageIcon(pageType) {
    if (!this.iconConfig || !this.iconConfig.pages[pageType]) {
      console.warn(`Page icon not found for: ${pageType}`);
      return;
    }
    
    this.setIcon(this.iconConfig.pages[pageType]);
  }

  /**
   * 恢复默认图标
   */
  resetToDefault() {
    this.setIcon(this.defaultIcon);
  }

  /**
   * 添加动态效果（如加载状态）
   * @param {string} effect - 效果类型
   */
  addEffect(effect) {
    switch (effect) {
      case 'loading':
        this.setLoadingIcon();
        break;
      case 'error':
        this.setErrorIcon();
        break;
      case 'success':
        this.setSuccessIcon();
        break;
      default:
        this.updateFavicon();
    }
  }

  /**
   * 设置加载状态图标
   */
  setLoadingIcon() {
    // 创建动态加载图标
    const canvas = document.createElement('canvas');
    canvas.width = 32;
    canvas.height = 32;
    const ctx = canvas.getContext('2d');
    
    // 绘制加载动画帧
    let frame = 0;
    const animate = () => {
      ctx.clearRect(0, 0, 32, 32);
      
      // 绘制旋转的圆环
      ctx.strokeStyle = '#1890ff';
      ctx.lineWidth = 3;
      ctx.lineCap = 'round';
      
      const angle = (frame * 10) * Math.PI / 180;
      ctx.beginPath();
      ctx.arc(16, 16, 12, angle, angle + Math.PI * 1.5);
      ctx.stroke();
      
      // 转换为数据URL并设置
      const dataUrl = canvas.toDataURL();
      this.setIcon(dataUrl);
      
      frame++;
      if (frame < 36) { // 一圈动画
        setTimeout(animate, 100);
      } else {
        this.updateFavicon(); // 恢复正常图标
      }
    };
    
    animate();
  }

  /**
   * 设置错误状态图标
   */
  setErrorIcon() {
    const canvas = document.createElement('canvas');
    canvas.width = 32;
    canvas.height = 32;
    const ctx = canvas.getContext('2d');
    
    // 绘制红色X
    ctx.strokeStyle = '#ff4d4f';
    ctx.lineWidth = 4;
    ctx.lineCap = 'round';
    
    ctx.beginPath();
    ctx.moveTo(8, 8);
    ctx.lineTo(24, 24);
    ctx.moveTo(24, 8);
    ctx.lineTo(8, 24);
    ctx.stroke();
    
    this.setIcon(canvas.toDataURL());
    
    // 3秒后恢复
    setTimeout(() => this.updateFavicon(), 3000);
  }

  /**
   * 设置成功状态图标
   */
  setSuccessIcon() {
    const canvas = document.createElement('canvas');
    canvas.width = 32;
    canvas.height = 32;
    const ctx = canvas.getContext('2d');
    
    // 绘制绿色勾
    ctx.strokeStyle = '#52c41a';
    ctx.lineWidth = 4;
    ctx.lineCap = 'round';
    
    ctx.beginPath();
    ctx.moveTo(8, 16);
    ctx.lineTo(14, 22);
    ctx.lineTo(24, 10);
    ctx.stroke();
    
    this.setIcon(canvas.toDataURL());
    
    // 3秒后恢复
    setTimeout(() => this.updateFavicon(), 3000);
  }
}

// 全局实例
window.faviconManager = new FaviconManager();

// 导出为ES模块
export default FaviconManager;
