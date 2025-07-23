import React from 'react';
import { Spin, Typography, Space } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';

const { Text } = Typography;

const LoadingFallback = ({ message = "正在加载页面...", size = "large", style = {} }) => {
  const antIcon = <LoadingOutlined style={{ fontSize: size === 'large' ? 24 : 16 }} spin />;

  return (
    <div style={{ 
      display: 'flex', 
      justifyContent: 'center', 
      alignItems: 'center', 
      minHeight: '200px',
      padding: '40px',
      ...style
    }}>
      <Space direction="vertical" align="center" size="middle">
        <Spin indicator={antIcon} size={size} />
        <Text type="secondary" style={{ fontSize: '14px' }}>
          {message}
        </Text>
      </Space>
    </div>
  );
};

// 针对管理员页面的专用加载组件
export const AdminLoadingFallback = () => (
  <LoadingFallback 
    message="正在加载管理页面..." 
    style={{ 
      minHeight: '300px',
      background: '#fafafa',
      borderRadius: '8px',
      margin: '20px'
    }} 
  />
);

// 针对项目页面的专用加载组件
export const ProjectLoadingFallback = () => (
  <LoadingFallback 
    message="正在加载项目..." 
    style={{ minHeight: '250px' }} 
  />
);

export default LoadingFallback;
