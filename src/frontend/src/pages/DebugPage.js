import React from 'react';
import { Card, Typography, Button } from 'antd';

const { Title, Paragraph, Text } = Typography;

const DebugPage = () => {
  const checkErrors = () => {
    const lastError = localStorage.getItem('lastError');
    const globalError = localStorage.getItem('globalError');
    const unhandledRejection = localStorage.getItem('unhandledRejection');
    
    console.log('=== 错误调试信息 ===');
    console.log('Last Error:', lastError ? JSON.parse(lastError) : 'None');
    console.log('Global Error:', globalError ? JSON.parse(globalError) : 'None');
    console.log('Unhandled Rejection:', unhandledRejection ? JSON.parse(unhandledRejection) : 'None');
    
    alert('请查看浏览器控制台获取错误详情');
  };
  
  const clearErrors = () => {
    localStorage.removeItem('lastError');
    localStorage.removeItem('globalError');
    localStorage.removeItem('unhandledRejection');
    alert('错误记录已清除');
  };
  
  const testAIAssistant = () => {
    try {
      // 尝试加载AI助手组件
      window.location.href = '/admin/ai-assistant';
    } catch (error) {
      console.error('Navigation error:', error);
    }
  };

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <Title level={2}>AI助手页面调试</Title>
        <Paragraph>
          当前页面用于调试AI助手管理页面的加载问题。
        </Paragraph>
        
        <div style={{ marginBottom: '16px' }}>
          <Button type="primary" onClick={checkErrors} style={{ marginRight: '8px' }}>
            检查错误日志
          </Button>
          <Button onClick={clearErrors} style={{ marginRight: '8px' }}>
            清除错误记录
          </Button>
          <Button type="default" onClick={testAIAssistant}>
            测试AI助手页面
          </Button>
        </div>
        
        <div>
          <Text strong>调试步骤：</Text>
          <ol>
            <li>首先点击"检查错误日志"查看是否有记录的错误</li>
            <li>打开浏览器开发者工具(F12)查看控制台</li>
            <li>点击"测试AI助手页面"尝试访问</li>
            <li>如果出现错误，返回此页面再次检查错误日志</li>
          </ol>
        </div>
      </Card>
    </div>
  );
};

export default DebugPage;
