import React, { useState } from 'react';
import { Card, Tabs, Typography } from 'antd';
import JupyterHubConfig from '../components/JupyterHubConfig';
import JupyterHubTasks from '../components/JupyterHubTasks';

const { Title } = Typography;
const { TabPane } = Tabs;

const JupyterHubManagement = () => {
  const [activeTab, setActiveTab] = useState('config');

  return (
    <div style={{ padding: 24 }}>
      <Title level={2}>JupyterHub管理</Title>
      
      <Card>
        <Tabs 
          activeKey={activeTab} 
          onChange={setActiveTab}
          type="card"
        >
          <TabPane tab="配置管理" key="config">
            <JupyterHubConfig />
          </TabPane>
          <TabPane tab="任务管理" key="tasks">
            <JupyterHubTasks />
          </TabPane>
        </Tabs>
      </Card>
    </div>
  );
};

export default JupyterHubManagement;
