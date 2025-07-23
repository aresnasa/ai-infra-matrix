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
  DashboardOutlined
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { usePagePreload } from '../hooks/usePagePreload';

const { Title, Paragraph } = Typography;

const AdminCenter = () => {
  const navigate = useNavigate();

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
      title: '用户管理',
      description: '管理系统用户账户、角色权限和认证源',
      icon: <UserOutlined style={{ fontSize: '24px', color: '#1890ff' }} />,
      path: '/admin/users',
      color: '#e6f7ff',
      badge: 'hot', // 标记为热门功能
      priority: 1
    },
    {
      title: '项目管理',
      description: '查看和管理所有用户项目',
      icon: <ProjectOutlined style={{ fontSize: '24px', color: '#52c41a' }} />,
      path: '/admin/projects',
      color: '#f6ffed',
      priority: 2
    },
    {
      title: 'LDAP认证设置',
      description: '配置LDAP服务器连接和用户认证',
      icon: <SafetyOutlined style={{ fontSize: '24px', color: '#fa8c16' }} />,
      path: '/admin/auth',
      color: '#fff7e6',
      badge: 'new', // 标记为新功能
      priority: 3
    },
    {
      title: 'LDAP配置',
      description: '详细的LDAP服务器配置和测试',
      icon: <SettingOutlined style={{ fontSize: '24px', color: '#722ed1' }} />,
      path: '/admin/ldap',
      color: '#f9f0ff',
      priority: 4
    },
    {
      title: '系统测试',
      description: '执行系统健康检查和功能测试',
      icon: <ExperimentOutlined style={{ fontSize: '24px', color: '#eb2f96' }} />,
      path: '/admin/test',
      color: '#fff0f6',
      priority: 5
    },
    {
      title: '回收站',
      description: '查看和恢复已删除的项目和用户',
      icon: <DeleteOutlined style={{ fontSize: '24px', color: '#fa541c' }} />,
      path: '/admin/trash',
      color: '#fff2e8',
      priority: 6
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
      hot: { color: 'red', text: '热门' },
      new: { color: 'green', text: '新功能' },
      beta: { color: 'blue', text: '测试版' }
    };

    const config = badgeConfig[badge];
    return config ? <Badge color={config.color} text={config.text} /> : null;
  };

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ marginBottom: '24px' }}>
        <Title level={2}>
          <DashboardOutlined style={{ marginRight: '8px' }} />
          管理中心
        </Title>
        <Paragraph style={{ fontSize: '16px', color: '#666' }}>
          欢迎来到系统管理中心，您可以在这里管理用户、项目和系统配置
        </Paragraph>
      </div>

      {/* 快捷操作区域 */}
      <Card 
        title="快捷操作" 
        style={{ marginBottom: '24px' }}
        extra={<TeamOutlined />}
      >
        <Space size="middle">
          <Button 
            type="primary" 
            icon={<UserOutlined />}
            onClick={() => navigate('/admin/users')}
          >
            用户管理
          </Button>
          <Button 
            icon={<SafetyOutlined />}
            onClick={() => navigate('/admin/auth')}
          >
            LDAP设置
          </Button>
          <Button 
            icon={<ExperimentOutlined />}
            onClick={() => navigate('/admin/test')}
          >
            系统测试
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
                transition: 'all 0.3s ease'
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
                e.currentTarget.style.boxShadow = '0 8px 24px rgba(0,0,0,0.12)';
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'translateY(0)';
                e.currentTarget.style.boxShadow = '0 2px 8px rgba(0,0,0,0.06)';
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
                    <Title level={4} style={{ margin: '0 0 0 12px' }}>
                      {card.title}
                    </Title>
                  </div>
                  {renderBadge(card.badge)}
                </div>
                <Paragraph style={{ 
                  color: '#666', 
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
        title="系统概览" 
        style={{ marginTop: '24px' }}
        extra={<SecurityScanOutlined />}
      >
        <Row gutter={16}>
          <Col span={6}>
            <div style={{ textAlign: 'center' }}>
              <Title level={3} style={{ color: '#1890ff', margin: '0 0 8px 0' }}>
                --
              </Title>
              <Paragraph style={{ margin: 0 }}>在线用户</Paragraph>
            </div>
          </Col>
          <Col span={6}>
            <div style={{ textAlign: 'center' }}>
              <Title level={3} style={{ color: '#52c41a', margin: '0 0 8px 0' }}>
                --
              </Title>
              <Paragraph style={{ margin: 0 }}>总项目数</Paragraph>
            </div>
          </Col>
          <Col span={6}>
            <div style={{ textAlign: 'center' }}>
              <Title level={3} style={{ color: '#fa8c16', margin: '0 0 8px 0' }}>
                --
              </Title>
              <Paragraph style={{ margin: 0 }}>今日任务</Paragraph>
            </div>
          </Col>
          <Col span={6}>
            <div style={{ textAlign: 'center' }}>
              <Title level={3} style={{ color: '#eb2f96', margin: '0 0 8px 0' }}>
                正常
              </Title>
              <Paragraph style={{ margin: 0 }}>系统状态</Paragraph>
            </div>
          </Col>
        </Row>
      </Card>
    </div>
  );
};

export default AdminCenter;
