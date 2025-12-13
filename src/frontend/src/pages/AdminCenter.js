import React, { useEffect } from 'react';
import { Card, Row, Col, Typography, Space, Button, Badge } from 'antd';
import { 
  UserOutlined, 
  ProjectOutlined, 
  SettingOutlined, 
  SecurityScanOutlined,
  DeleteOutlined,
  TeamOutlined,
  SafetyOutlined,
  ExperimentOutlined,
  DashboardOutlined,
  DatabaseOutlined
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { usePagePreload } from '../hooks/usePagePreload';
import { useI18n } from '../hooks/useI18n';
import { useTheme } from '../hooks/useTheme';

const { Title, Paragraph } = Typography;

const AdminCenter = () => {
  const navigate = useNavigate();
  const { t } = useI18n();
  const { isDark } = useTheme();

  // 针对管理中心页面的智能预加载
  usePagePreload(['admin-users', 'admin-auth', 'admin-test']);

  // 页面加载时预加载关键的管理员组件
  useEffect(() => {
    // 预加载用户管理页面（最常用的功能）
    import('./AdminUsers').catch(() => {
      // 静默处理预加载失败
    });
  }, []);

  const adminCards = [
    {
      title: t('admin.userManagement'),
      description: t('admin.userManagementDesc'),
      icon: <UserOutlined style={{ fontSize: '24px', color: '#1890ff' }} />,
      path: '/admin/users',
      color: isDark ? '#111d2c' : '#e6f7ff',
      darkColor: '#111d2c',
      badge: 'hot', // 标记为热门功能
      priority: 1
    },
    {
      title: t('admin.projectManagement'),
      description: t('admin.projectManagementDesc'),
      icon: <ProjectOutlined style={{ fontSize: '24px', color: '#52c41a' }} />,
      path: '/admin/projects',
      color: isDark ? '#162312' : '#f6ffed',
      priority: 2
    },
    {
      title: t('admin.ldapAuth'),
      description: t('admin.ldapAuthDesc'),
      icon: <SafetyOutlined style={{ fontSize: '24px', color: '#fa8c16' }} />,
      path: '/admin/auth',
      color: isDark ? '#2b1d11' : '#fff7e6',
      badge: 'new', // 标记为新功能
      priority: 3
    },
    {
      title: t('admin.ldapConfig'),
      description: t('admin.ldapConfigDesc'),
      icon: <SettingOutlined style={{ fontSize: '24px', color: '#722ed1' }} />,
      path: '/admin/ldap',
      color: isDark ? '#1a1325' : '#f9f0ff',
      priority: 4
    },
    {
      title: t('admin.systemTest'),
      description: t('admin.systemTestDesc'),
      icon: <ExperimentOutlined style={{ fontSize: '24px', color: '#eb2f96' }} />,
      path: '/admin/test',
      color: isDark ? '#291321' : '#fff0f6',
      priority: 5
    },
    {
      title: t('admin.trash'),
      description: t('admin.trashDesc'),
      icon: <DeleteOutlined style={{ fontSize: '24px', color: '#fa541c' }} />,
      path: '/admin/trash',
      color: isDark ? '#2b1611' : '#fff2e8',
      priority: 6
    },
    {
      title: t('admin.objectStorage'),
      description: t('admin.objectStorageDesc'),
      icon: <DatabaseOutlined style={{ fontSize: '24px', color: '#13c2c2' }} />,
      path: '/admin/object-storage',
      color: isDark ? '#112123' : '#e6fffb',
      badge: 'new',
      priority: 7
    }
  ];

  // 按优先级排序
  const sortedCards = [...adminCards].sort((a, b) => a.priority - b.priority);

  const handleCardClick = (path) => {
    navigate(path);
  };

  const renderBadge = (badge) => {
    if (!badge) return null;
    
    const badgeConfig = {
      hot: { color: 'red', text: t('admin.hot') },
      new: { color: 'green', text: t('admin.new') },
      beta: { color: 'blue', text: t('admin.beta') }
    };

    const config = badgeConfig[badge];
    return config ? <Badge color={config.color} text={config.text} /> : null;
  };

  return (
    <div style={{ padding: '24px', background: isDark ? '#141414' : '#f0f2f5', minHeight: '100vh' }}>
      <div style={{ marginBottom: '24px' }}>
        <Title level={2} style={{ color: isDark ? 'rgba(255, 255, 255, 0.85)' : 'inherit' }}>
          <DashboardOutlined style={{ marginRight: '8px' }} />
          {t('admin.title')}
        </Title>
        <Paragraph style={{ fontSize: '16px', color: isDark ? 'rgba(255, 255, 255, 0.45)' : '#666' }}>
          {t('admin.welcome')}，{t('admin.welcomeDesc')}
        </Paragraph>
      </div>

      {/* 快捷操作区域 */}
      <Card 
        title={t('admin.quickActions')} 
        style={{ marginBottom: '24px', background: isDark ? '#1f1f1f' : '#fff' }}
        extra={<TeamOutlined />}
      >
        <Space size="middle">
          <Button 
            type="primary" 
            icon={<UserOutlined />}
            onClick={() => navigate('/admin/users')}
          >
            {t('admin.userManagement')}
          </Button>
          <Button 
            icon={<SafetyOutlined />}
            onClick={() => navigate('/admin/auth')}
          >
            {t('admin.ldapSettings')}
          </Button>
          <Button 
            icon={<ExperimentOutlined />}
            onClick={() => navigate('/admin/test')}
          >
            {t('admin.systemTest')}
          </Button>
        </Space>
      </Card>

      {/* 功能卡片网格 */}
      <Row gutter={[16, 16]}>
        {sortedCards.map((card, index) => (
          <Col xs={24} sm={12} lg={8} key={index}>
            <Card
              hoverable
              style={{ 
                height: '180px',
                backgroundColor: card.color,
                cursor: 'pointer',
                transition: 'all 0.3s ease',
                borderColor: isDark ? '#303030' : undefined
              }}
              bodyStyle={{ 
                padding: '24px',
                height: '100%',
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'space-between'
              }}
              onClick={() => handleCardClick(card.path)}
              onMouseEnter={(e) => {
                e.currentTarget.style.transform = 'translateY(-4px)';
                e.currentTarget.style.boxShadow = isDark ? '0 8px 24px rgba(0,0,0,0.45)' : '0 8px 24px rgba(0,0,0,0.12)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)';
                e.currentTarget.style.boxShadow = isDark ? '0 2px 8px rgba(0,0,0,0.35)' : '0 2px 8px rgba(0,0,0,0.06)';
              }}
            >
              <div>
                <div style={{ 
                  display: 'flex', 
                  alignItems: 'center', 
                  justifyContent: 'space-between',
                  marginBottom: '12px' 
                }}>
                  <div style={{ display: 'flex', alignItems: 'center' }}>
                    {card.icon}
                    <Title level={4} style={{ margin: '0 0 0 12px', color: isDark ? 'rgba(255, 255, 255, 0.85)' : 'inherit' }}>
                      {card.title}
                    </Title>
                  </div>
                  {renderBadge(card.badge)}
                </div>
                <Paragraph style={{ 
                  color: isDark ? 'rgba(255, 255, 255, 0.65)' : '#666', 
                  fontSize: '14px',
                  margin: 0,
                  lineHeight: '1.4'
                }}>
                  {card.description}
                </Paragraph>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      {/* 统计信息卡片 */}
      <Card 
        title={t('admin.systemOverview')} 
        style={{ marginTop: '24px' }}
        extra={<SecurityScanOutlined />}
      >
        <Row gutter={16}>
          <Col span={6}>
            <div style={{ textAlign: 'center' }}>
              <Title level={3} style={{ color: '#1890ff', margin: '0 0 8px 0' }}>
                --
              </Title>
              <Paragraph style={{ margin: 0 }}>{t('admin.onlineUsers')}</Paragraph>
            </div>
          </Col>
          <Col span={6}>
            <div style={{ textAlign: 'center' }}>
              <Title level={3} style={{ color: '#52c41a', margin: '0 0 8px 0' }}>
                --
              </Title>
              <Paragraph style={{ margin: 0 }}>{t('admin.totalProjects')}</Paragraph>
            </div>
          </Col>
          <Col span={6}>
            <div style={{ textAlign: 'center' }}>
              <Title level={3} style={{ color: '#fa8c16', margin: '0 0 8px 0' }}>
                --
              </Title>
              <Paragraph style={{ margin: 0 }}>{t('admin.todayTasks')}</Paragraph>
            </div>
          </Col>
          <Col span={6}>
            <div style={{ textAlign: 'center' }}>
              <Title level={3} style={{ color: '#eb2f96', margin: '0 0 8px 0' }}>
                {t('admin.normal')}
              </Title>
              <Paragraph style={{ margin: 0 }}>{t('admin.systemStatus')}</Paragraph>
            </div>
          </Col>
        </Row>
      </Card>
    </div>
  );
};

export default AdminCenter;
