import React from 'react';
import { Result, Button } from 'antd';
import { ReloadOutlined, HomeOutlined } from '@ant-design/icons';

class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error) {
    // 更新 state 使下一次渲染能够显示降级后的 UI
    return { hasError: true };
  }

  componentDidCatch(error, errorInfo) {
    // 你同样可以将错误日志上报给服务器
    console.error('ErrorBoundary caught an error:', error, errorInfo);
    console.error('Error stack:', error.stack);
    console.error('Component stack:', errorInfo.componentStack);
    
    // 也记录到localStorage以便调试
    try {
      localStorage.setItem('lastError', JSON.stringify({
        error: error.toString(),
        stack: error.stack,
        componentStack: errorInfo.componentStack,
        timestamp: new Date().toISOString()
      }));
    } catch (e) {
      console.error('Failed to save error to localStorage:', e);
    }
    
    this.setState({
      error: error,
      errorInfo: errorInfo
    });
  }

  handleReload = () => {
    window.location.reload();
  };

  handleGoHome = () => {
    window.location.href = '/';
  };

  render() {
    if (this.state.hasError) {
      // 你可以自定义降级后的 UI 并渲染
      return (
        <div style={{ 
          padding: '50px', 
          display: 'flex', 
          justifyContent: 'center', 
          alignItems: 'center',
          minHeight: '400px'
        }}>
          <Result
            status="error"
            title="页面加载失败"
            subTitle="抱歉，页面出现了一些问题。请尝试刷新页面或返回首页。"
            extra={[
              <Button type="primary" icon={<ReloadOutlined />} onClick={this.handleReload} key="reload">
                刷新页面
              </Button>,
              <Button icon={<HomeOutlined />} onClick={this.handleGoHome} key="home">
                返回首页
              </Button>,
            ]}
          >
            <div style={{ textAlign: 'left', marginTop: '20px' }}>
              <details open style={{ whiteSpace: 'pre-wrap', background: '#f5f5f5', padding: '10px', borderRadius: '4px' }}>
                <summary><strong>错误详情</strong></summary>
                <div style={{ marginTop: '10px' }}>
                  <strong>错误信息:</strong><br />
                  {this.state.error && this.state.error.toString()}<br />
                  <strong>错误堆栈:</strong><br />
                  {this.state.error && this.state.error.stack}<br />
                  <strong>组件堆栈:</strong><br />
                  {this.state.errorInfo && this.state.errorInfo.componentStack}
                </div>
              </details>
            </div>
          </Result>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
