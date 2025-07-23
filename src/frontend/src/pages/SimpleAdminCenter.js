import React from 'react';

const SimpleAdminCenter = () => {
  return (
    <div style={{ padding: '20px' }}>
      <h1>简单管理中心测试</h1>
      <p>如果您能看到这个页面，说明路由正常工作。</p>
      <div style={{ marginTop: '20px' }}>
        <button onClick={() => alert('按钮点击正常')}>
          测试按钮
        </button>
      </div>
    </div>
  );
};

export default SimpleAdminCenter;
