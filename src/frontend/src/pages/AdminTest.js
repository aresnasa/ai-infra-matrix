import React from 'react';
import { Card, Typography } from 'antd';

const { Title } = Typography;

const AdminTest = () => {
  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <Title level={2}>管理中心测试页面</Title>
        <p>如果您看到这个页面，说明路由工作正常！</p>
        <p>当前时间: {new Date().toLocaleString()}</p>
      </Card>
    </div>
  );
};

export default AdminTest;
