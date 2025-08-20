import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, List, Typography, Button, Space, Spin } from 'antd';
import { 
  ProjectOutlined, 
  UserOutlined, 
  ClusterOutlined, 
  BranchesOutlined,
  RightOutlined 
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { projectAPI, enhancedUserAPI, kubernetesAPI, authAPI } from '../services/api';

const { Title, Text } = Typography;

const OverviewPage = ({ user }) => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({
    projects: 0,
    users: 0,
    clusters: 0,
    repositories: 0
  });
  const [recentProjects, setRecentProjects] = useState([]);
  const [recentActivity, setRecentActivity] = useState([]);

  useEffect(() => {
    loadOverviewData();
  }, []);

  const loadOverviewData = async () => {
    try {
      setLoading(true);
      
      // 并行获取各种统计数据
      const [projectsRes, usersRes, clustersRes] = await Promise.allSettled([
        projectAPI.getProjects(),
        user?.role === 'admin' || user?.role === 'super-admin' ? enhancedUserAPI.getUsers() : Promise.resolve({ data: [] }),
        kubernetesAPI.getClusters().catch(() => ({ data: [] }))
      ]);

      const projectCount = projectsRes.status === 'fulfilled' ? projectsRes.value.data.length : 0;
      const userCount = usersRes.status === 'fulfilled' ? usersRes.value.data.length : 0;
      const clusterCount = clustersRes.status === 'fulfilled' ? clustersRes.value.data.length : 0;

      setStats({
        projects: projectCount,
        users: userCount,
        clusters: clusterCount,
        repositories: Math.floor(Math.random() * 50) + 10 // 模拟数据
      });

      // 设置最近项目
      if (projectsRes.status === 'fulfilled') {
        setRecentProjects(projectsRes.value.data.slice(0, 5));
      }

      // 模拟最近活动
      setRecentActivity([
        { id: 1, action: '创建了新项目', target: 'AI训练平台', time: '2小时前' },
        { id: 2, action: '部署了集群', target: 'kubernetes-prod', time: '4小时前' },
        { id: 3, action: '同步了用户', target: 'LDAP', time: '6小时前' },
        { id: 4, action: '执行了任务', target: 'Ansible playbook', time: '8小时前' }
      ]);

    } catch (error) {
      console.error('加载概览数据失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const quickActions = [
    {
      title: '创建项目',
      icon: <ProjectOutlined />,
      onClick: () => navigate('/projects'),
      description: '开始新的项目'
    },
    {
      title: '管理集群',
      icon: <ClusterOutlined />,
      onClick: () => navigate('/kubernetes'),
      description: '查看Kubernetes集群',
      adminOnly: true
    },
    {
      title: '用户管理',
      icon: <UserOutlined />,
      onClick: () => navigate('/ldap-management'),
      description: '管理系统用户',
      adminOnly: true
    },
    {
      title: '代码仓库',
      icon: <BranchesOutlined />,
      onClick: () => navigate('/gitea'),
      description: '访问Git仓库'
    }
  ];

  const isAdmin = user?.role === 'admin' || user?.role === 'super-admin';
  const availableActions = quickActions.filter(action => !action.adminOnly || isAdmin);

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '400px' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>系统概览</Title>
      <Text type="secondary">欢迎回来，{user?.username}！以下是您的系统概览信息。</Text>
      
      <Row gutter={[16, 16]} style={{ marginTop: '24px' }}>
        {/* 统计卡片 */}
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="项目总数"
              value={stats.projects}
              prefix={<ProjectOutlined style={{ color: '#1890ff' }} />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="代码仓库"
              value={stats.repositories}
              prefix={<BranchesOutlined style={{ color: '#52c41a' }} />}
            />
          </Card>
        </Col>
        {isAdmin && (
          <>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title="系统用户"
                  value={stats.users}
                  prefix={<UserOutlined style={{ color: '#722ed1' }} />}
                />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={6}>
              <Card>
                <Statistic
                  title="Kubernetes集群"
                  value={stats.clusters}
                  prefix={<ClusterOutlined style={{ color: '#fa8c16' }} />}
                />
              </Card>
            </Col>
          </>
        )}
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: '24px' }}>
        {/* 快速操作 */}
        <Col xs={24} lg={12}>
          <Card title="快速操作" size="small">
            <Row gutter={[8, 8]}>
              {availableActions.map((action, index) => (
                <Col xs={24} sm={12} key={index}>
                  <Card
                    hoverable
                    size="small"
                    onClick={action.onClick}
                    style={{ cursor: 'pointer' }}
                  >
                    <Space direction="vertical" size={4} style={{ width: '100%' }}>
                      <Space>
                        {action.icon}
                        <Text strong>{action.title}</Text>
                      </Space>
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        {action.description}
                      </Text>
                    </Space>
                  </Card>
                </Col>
              ))}
            </Row>
          </Card>
        </Col>

        {/* 最近项目 */}
        <Col xs={24} lg={12}>
          <Card 
            title="最近项目" 
            size="small"
            extra={
              <Button 
                type="link" 
                size="small" 
                onClick={() => navigate('/projects')}
                icon={<RightOutlined />}
              >
                查看全部
              </Button>
            }
          >
            <List
              size="small"
              dataSource={recentProjects}
              renderItem={project => (
                <List.Item
                  onClick={() => navigate(`/projects/${project.id}`)}
                  style={{ cursor: 'pointer', padding: '8px 0' }}
                >
                  <List.Item.Meta
                    avatar={<ProjectOutlined />}
                    title={<Text strong>{project.name}</Text>}
                    description={
                      <Text type="secondary" style={{ fontSize: '12px' }}>
                        {project.description || '暂无描述'}
                      </Text>
                    }
                  />
                </List.Item>
              )}
              locale={{ emptyText: '暂无项目' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 最近活动 */}
      <Row gutter={[16, 16]} style={{ marginTop: '16px' }}>
        <Col xs={24}>
          <Card title="最近活动" size="small">
            <List
              size="small"
              dataSource={recentActivity}
              renderItem={activity => (
                <List.Item>
                  <List.Item.Meta
                    title={
                      <Space>
                        <Text>{user?.username}</Text>
                        <Text type="secondary">{activity.action}</Text>
                        <Text strong>{activity.target}</Text>
                      </Space>
                    }
                    description={<Text type="secondary">{activity.time}</Text>}
                  />
                </List.Item>
              )}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default OverviewPage;
