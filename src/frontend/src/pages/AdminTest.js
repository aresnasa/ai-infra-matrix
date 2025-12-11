import React from 'react';
import { Card, Typography } from 'antd';
import { useI18n } from '../hooks/useI18n';

const { Title } = Typography;

const AdminTest = () => {
  const { t } = useI18n();

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <Title level={2}>{t('admin.testPage')}</Title>
        <p>{t('admin.testPageInfo')}</p>
        <p>{t('admin.currentTime')}: {new Date().toLocaleString()}</p>
      </Card>
    </div>
  );
};

export default AdminTest;
