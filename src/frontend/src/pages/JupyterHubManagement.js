import React, { useState } from 'react';
import { Card, Tabs, Typography } from 'antd';
import JupyterHubConfig from '../components/JupyterHubConfig';
import JupyterHubTasks from '../components/JupyterHubTasks';
import JupyterLabTemplateManager from '../components/JupyterLabTemplateManager';

const { Title } = Typography;

const JupyterHubManagement = () => {
  const [activeTab, setActiveTab] = useState('templates');

  const tabItems = [
    {
      key: 'templates',
      label: 'JupyterLab模板',
      children: <JupyterLabTemplateManager />
    },
    {
      key: 'config',
      label: '配置管理',
      children: <JupyterHubConfig />
    },
    {
      key: 'tasks',
      label: '任务管理',
      children: <JupyterHubTasks />
    }
  ];

  return (
    <div style={{ padding: 24 }}>
      <Title level={2}>JupyterHub管理</Title>
      
      <Card>
        <Tabs 
          activeKey={activeTab} 
          onChange={setActiveTab}
          type="card"
          items={tabItems}
        />
      </Card>
    </div>
  );
};

export default JupyterHubManagement;
