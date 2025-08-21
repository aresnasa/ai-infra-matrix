import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';

// 全局错误处理
window.addEventListener('error', (event) => {
  console.error('Global error caught:', event.error);
  console.error('Error details:', {
    message: event.message,
    filename: event.filename,
    lineno: event.lineno,
    colno: event.colno,
    error: event.error
  });
  
  // 保存到localStorage以便调试
  try {
    localStorage.setItem('globalError', JSON.stringify({
      message: event.message,
      filename: event.filename,
      lineno: event.lineno,
      colno: event.colno,
      stack: event.error ? event.error.stack : null,
      timestamp: new Date().toISOString()
    }));
  } catch (e) {
    console.error('Failed to save global error to localStorage:', e);
  }
});

// 捕获Promise rejection错误
window.addEventListener('unhandledrejection', (event) => {
  console.error('Unhandled promise rejection:', event.reason);
  
  try {
    localStorage.setItem('unhandledRejection', JSON.stringify({
      reason: event.reason.toString(),
      stack: event.reason.stack,
      timestamp: new Date().toISOString()
    }));
  } catch (e) {
    console.error('Failed to save promise rejection to localStorage:', e);
  }
});

const root = ReactDOM.createRoot(document.getElementById('root'));
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
