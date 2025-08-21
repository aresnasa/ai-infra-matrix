import React from 'react';

// 主logo图标组件
export const MainLogoIcon = ({ style = {}, ...props }) => (
  <img 
    src="/logo-main.svg" 
    alt="AI-Infra-Matrix Logo" 
    style={{ 
      width: '24px', 
      height: '24px', 
      ...style 
    }}
    {...props}
  />
);

// 内联SVG版本（更快加载）
export const MainLogoSVG = ({ style = {}, size = 24, ...props }) => (
  <svg 
    width={size} 
    height={size} 
    viewBox="0 0 32 32" 
    fill="none" 
    xmlns="http://www.w3.org/2000/svg"
    style={style}
    {...props}
  >
    {/* 背景圆形 */}
    <circle cx="16" cy="16" r="15" fill="#1890ff" stroke="#ffffff" strokeWidth="2"/>
    
    {/* AI芯片图标 */}
    <rect x="8" y="8" width="16" height="16" rx="2" fill="#ffffff" fillOpacity="0.9"/>
    
    {/* 矩阵网格 */}
    <g stroke="#1890ff" strokeWidth="1.5" fill="none">
      {/* 水平线 */}
      <line x1="10" y1="12" x2="22" y2="12"/>
      <line x1="10" y1="16" x2="22" y2="16"/>
      <line x1="10" y1="20" x2="22" y2="20"/>
      
      {/* 垂直线 */}
      <line x1="12" y1="10" x2="12" y2="22"/>
      <line x1="16" y1="10" x2="16" y2="22"/>
      <line x1="20" y1="10" x2="20" y2="22"/>
    </g>
    
    {/* 中央AI点 */}
    <circle cx="16" cy="16" r="2" fill="#1890ff"/>
    
    {/* 连接点 */}
    <circle cx="12" cy="12" r="1" fill="#52c41a"/>
    <circle cx="20" cy="12" r="1" fill="#52c41a"/>
    <circle cx="12" cy="20" r="1" fill="#52c41a"/>
    <circle cx="20" cy="20" r="1" fill="#52c41a"/>
  </svg>
);

// 自定义菜单图标组件
export const CustomMenuIcons = {
  // 仪表板图标
  Dashboard: ({ style = {}, size = 16 }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" style={style}>
      <path d="M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z"/>
    </svg>
  ),
  
  // 项目管理图标
  Projects: ({ style = {}, size = 16 }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" style={style}>
      <path d="M14,2H6A2,2 0 0,0 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2M18,20H6V4H13V9H18V20Z"/>
    </svg>
  ),
  
  // Kubernetes图标
  Kubernetes: ({ style = {}, size = 16 }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" style={style}>
      <path d="M10.204 14.35l.007.01-.999 2.413a5.171 5.171 0 0 1-2.075-2.597l2.578-.437.004.005a.44.44 0 0 1 .485.606zm-.833-2.129a.44.44 0 0 1 .173-.756l.01-.002 2.47-.881a5.188 5.188 0 0 1 1.285 3.273l-2.918.085-.003-.001a.44.44 0 0 1-.017-.718zm3.851-1.847l2.414-.998a5.171 5.171 0 0 1 .461 3.025l-2.488-.544-.005.002a.44.44 0 0 1-.382-.485z"/>
    </svg>
  ),
  
  // Gitea代码图标
  Gitea: ({ style = {}, size = 16 }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" style={style}>
      <path d="M12.017 0C5.396 0 .029 5.367.029 11.987c0 5.079 3.158 9.417 7.618 11.174-.105-.949-.199-2.403.041-3.439.219-.937 1.404-5.965 1.404-5.965s-.359-.719-.359-1.782c0-1.668.967-2.914 2.171-2.914 1.023 0 1.518.769 1.518 1.69 0 1.029-.653 2.567-.992 3.992-.282 1.193.6 2.165 1.775 2.165 2.128 0 3.768-2.245 3.768-5.487 0-2.861-2.063-4.869-5.008-4.869-3.41 0-5.409 2.562-5.409 5.199 0 1.033.394 2.143.889 2.741.099.12.112.225.085.345-.09.375-.293 1.199-.334 1.363-.053.225-.172.271-.402.165-1.495-.69-2.433-2.878-2.433-4.646 0-3.776 2.748-7.252 7.92-7.252 4.158 0 7.392 2.967 7.392 6.923 0 4.135-2.607 7.462-6.233 7.462-1.214 0-2.357-.629-2.746-1.378l-.748 2.853c-.271 1.043-1.002 2.35-1.492 3.146C9.57 23.812 10.763 24.009 12.017 24.009c6.624 0 11.99-5.367 11.99-11.988C24.007 5.367 18.641.001 12.017.001z"/>
    </svg>
  ),
  
  // JupyterHub图标
  Jupyter: ({ style = {}, size = 16 }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" style={style}>
      <path d="M7.157 22.201A1.784 1.799 0 0 1 5.374 20.4a1.784 1.799 0 0 1 1.783-1.799a1.784 1.799 0 0 1 1.784 1.799a1.784 1.799 0 0 1-1.784 1.801zM20.582 1.427a1.415 1.427 0 0 1-1.414 1.428a1.415 1.427 0 0 1-1.415-1.428A1.415 1.427 0 0 1 19.168 0a1.415 1.427 0 0 1 1.414 1.427zM16.845 6.7a1.23 1.241 0 0 1-1.23 1.241a1.23 1.241 0 0 1-1.229-1.241A1.23 1.241 0 0 1 15.615 5.46a1.23 1.241 0 0 1 1.23 1.241z"/>
    </svg>
  ),
  
  // AI助手图标
  AIAssistant: ({ style = {}, size = 16 }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" style={style}>
      <path d="M20,2A2,2 0 0,1 22,4V16A2,2 0 0,1 20,18H6L2,22V4C2,2.89 2.9,2 4,2H20M8,14H16V12H8V14M8,11H18V9H8V11M8,8H18V6H8V8Z"/>
    </svg>
  )
};

export default {
  MainLogoIcon,
  MainLogoSVG,
  CustomMenuIcons
};
