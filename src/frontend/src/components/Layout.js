import React from 'react';
import { Layout as AntLayout, Menu, Typography, Dropdown, Avatar, Space, Button } from 'antd';
import { ProjectOutlined, CodeOutlined, UserOutlined, LogoutOutlined, TeamOutlined, SafetyOutlined, DeleteOutlined, SecurityScanOutlined, ExperimentOutlined, DownOutlined, CloudServerOutlined, FileTextOutlined, RobotOutlined, ExperimentTwoTone, ClusterOutlined } from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import CustomizableNavigation from './CustomizableNavigation';
import { MainLogoSVG, CustomMenuIcons } from './CustomIcons';

const { Header, Content, Footer } = AntLayout;
const { Title } = Typography;

const Layout = ({ children, user, onLogout }) => {
  const navigate = useNavigate();
  const location = useLocation();

  // 检查用户是否为管理员（支持 admin 和 super-admin 角色）
  const isAdmin = user?.role === 'admin' || user?.role === 'super-admin' || 
    (user?.roles && user.roles.some(role => role.name === 'admin' || role.name === 'super-admin'));

  console.log('=== Layout权限检查 ===');
  console.log('用户信息:', user);
  console.log('用户角色:', user?.role);
  console.log('用户权限组:', user?.roles);
  console.log('是否管理员:', isAdmin);
  console.log('当前路径:', location.pathname);
  console.log('========================');

  const menuItems = [
    {
      key: '/dashboard',
      icon: <CustomMenuIcons.Dashboard />,
      label: '我的工作台',
    },
    {
      key: '/enhanced-dashboard',
      icon: <ExperimentOutlined />,
      label: '增强仪表板',
    },
    {
      key: '/projects',
      icon: <CustomMenuIcons.Projects />,
      label: '项目管理',
    },
    {
      key: '/gitea',
      icon: <CustomMenuIcons.Gitea />,
      label: 'Gitea',
    },
    {
      key: '/kubernetes',
      icon: <CustomMenuIcons.Kubernetes />,
      label: 'Kubernetes',
    },
    {
      key: '/ansible',
      icon: <FileTextOutlined />,
      label: 'Ansible',
    },
    {
      key: '/jupyterhub',
      icon: <CustomMenuIcons.Jupyter />,
      label: 'JupyterHub',
    },
    {
      key: '/slurm',
      icon: <ClusterOutlined />,
      label: 'Slurm',
    },
    {
      key: '/saltstack',
      icon: <CustomMenuIcons.Menu size={16} />,
      label: 'SaltStack',
    },
  ];

  // 管理中心下拉菜单项
  const adminMenuItems = [
    {
      key: '/admin/users',
      icon: <UserOutlined />,
      label: '用户管理',
      onClick: () => navigate('/admin/users'),
    },
    {
      key: '/admin/projects',
      icon: <ProjectOutlined />,
      label: '项目管理',
      onClick: () => navigate('/admin/projects'),
    },
    {
      key: '/admin/auth',
      icon: <SafetyOutlined />,
      label: 'LDAP认证设置',
      onClick: () => navigate('/admin/auth'),
    },
    {
      key: '/admin/ldap',
      icon: <TeamOutlined />,
      label: 'LDAP管理',
      onClick: () => navigate('/admin/ldap'),
    },
    {
      key: '/admin/test',
      icon: <ExperimentOutlined />,
      label: '系统测试',
      onClick: () => navigate('/admin/test'),
    },
    {
      key: '/admin/trash',
      icon: <DeleteOutlined />,
      label: '回收站',
      onClick: () => navigate('/admin/trash'),
    },
    {
      key: '/admin/ai-assistant',
      icon: <CustomMenuIcons.AIAssistant />,
      label: 'AI助手管理',
      onClick: () => navigate('/admin/ai-assistant'),
    },
    {
      key: '/admin/jupyterhub',
      icon: <CloudServerOutlined />,
      label: 'JupyterHub管理',
      onClick: () => navigate('/admin/jupyterhub'),
    },
  ];

  const handleMenuClick = ({ key }) => {
    // 特殊处理JupyterHub访问
    if (key === '/jupyterhub') {
      // Navigate to embedded Jupyter page for consistent UX
  navigate('/jupyter');
      return;
    }
    
    // 其他菜单项正常导航
    navigate(key);
  };

  // 获取当前选中的菜单key和打开的子菜单
  const getCurrentMenuKeys = () => {
    const pathname = location.pathname;
    let selectedKeys = [pathname];
    const openKeys = [];
    
    // 特殊路径映射
    if (pathname === '/jupyter' || pathname.startsWith('/jupyter')) {
      selectedKeys = ['/jupyterhub'];
    } else if (pathname === '/gitea' || pathname.startsWith('/gitea')) {
      selectedKeys = ['/gitea'];
    } else if (pathname === '/projects' || pathname.startsWith('/projects')) {
      selectedKeys = ['/projects'];
    } else if (pathname === '/kubernetes' || pathname.startsWith('/kubernetes')) {
      selectedKeys = ['/kubernetes'];
    } else if (pathname === '/ansible' || pathname.startsWith('/ansible')) {
      selectedKeys = ['/ansible'];
    } else if (pathname === '/slurm' || pathname.startsWith('/slurm')) {
      selectedKeys = ['/slurm'];
    } else if (pathname === '/saltstack' || pathname.startsWith('/saltstack')) {
      selectedKeys = ['/saltstack'];
    } else if (pathname === '/ldap-management' || pathname.startsWith('/ldap-management')) {
      selectedKeys = ['/ldap-management'];
    }
    
    // 如果是管理员页面，确保管理中心子菜单展开
    if (pathname.startsWith('/admin') && pathname !== '/admin') {
      openKeys.push('/admin');
    }
    
    return { selectedKeys, openKeys };
  };

  const { selectedKeys, openKeys } = getCurrentMenuKeys();

  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人信息',
      onClick: () => navigate('/profile'),
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: onLogout,
    },
  ];

  return (
    <AntLayout>
      <Header style={{ 
        display: 'flex', 
        alignItems: 'center', 
        background: '#001529',
        padding: '0 24px',
        minWidth: '1200px', // 设置最小宽度防止挤压
        overflow: 'hidden' // 防止内容溢出
      }}>
        {/* 左侧标题区域 - 固定宽度不会被挤压 */}
        <div style={{ 
          display: 'flex', 
          alignItems: 'center', 
          minWidth: '200px', // 固定最小宽度
          flexShrink: 0 // 不允许收缩
        }}>
          <MainLogoSVG style={{ 
            fontSize: '24px', 
            color: '#1890ff', 
            marginRight: '12px' 
          }} />
          <Title 
            level={3} 
            style={{ 
              color: '#fff', 
              margin: 0,
              fontWeight: 600,
              whiteSpace: 'nowrap' // 防止标题换行
            }}
          >
            AI-Infra-Matrix
          </Title>
        </div>
        
        {/* 中间导航区域 - 可伸缩 */}
        <div style={{ 
          display: 'flex', 
          alignItems: 'center', 
          flex: 1,
          minWidth: 0, // 允许收缩
          overflow: 'hidden' // 防止溢出
        }}>
          <CustomizableNavigation
            user={user}
            selectedKeys={selectedKeys}
            onMenuClick={handleMenuClick}
          />
        </div>
          
        {/* 右侧管理员菜单 */}
        {isAdmin && (
          <Dropdown
            menu={{ items: adminMenuItems }}
            placement="bottomRight"
            trigger={['hover']}
          >
            <Button 
              type="text" 
              style={{ 
                color: '#fff',
                height: '64px',
                padding: '0 16px',
                marginLeft: '8px',
                backgroundColor: location.pathname.startsWith('/admin') ? '#1890ff' : 'transparent',
                borderRadius: '0'
              }}
              icon={<CustomMenuIcons.Menu size={16} />}
              onClick={() => navigate('/admin')}
            >
              管理中心 <DownOutlined style={{ marginLeft: '4px', fontSize: '12px' }} />
            </Button>
          </Dropdown>
        )}

        {/* 右侧用户菜单 */}
        <Dropdown
          menu={{ items: userMenuItems }}
          placement="bottomRight"
          trigger={['click']}
        >
          <Space style={{ cursor: 'pointer', color: '#fff', marginLeft: '16px' }}>
            <Avatar icon={<UserOutlined />} />
            <span>{user?.username}</span>
            {isAdmin && (
              <span style={{ 
                background: '#52c41a', 
                padding: '2px 6px', 
                borderRadius: '4px',
                fontSize: '12px' 
              }}>
                管理员
              </span>
            )}
          </Space>
        </Dropdown>
      </Header>
      
      <Content className="ant-layout-content">
        {children}
      </Content>
      
      <Footer style={{ 
        textAlign: 'center',
        background: '#f0f2f5',
        borderTop: '1px solid #e8e8e8'
      }}>
        AI-Infra-Matrix ©2025 Created by DevOps Team
      </Footer>
    </AntLayout>
  );
};

export default Layout;
