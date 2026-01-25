import React, { useState, useEffect } from 'react';
import { Layout as AntLayout, Menu, Typography, Dropdown, Avatar, Space, Button } from 'antd';
import { ProjectOutlined, CodeOutlined, UserOutlined, LogoutOutlined, TeamOutlined, SafetyOutlined, DeleteOutlined, SecurityScanOutlined, ExperimentOutlined, DownOutlined, CloudServerOutlined, FileTextOutlined, RobotOutlined, ExperimentTwoTone, ClusterOutlined, KeyOutlined, DatabaseOutlined, DashboardOutlined, DeploymentUnitOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { useNavigate, useLocation } from 'react-router-dom';
import CustomizableNavigation from './CustomizableNavigation';
import { MainLogoSVG, CustomMenuIcons } from './CustomIcons';
import { getAvailableMenuItems, isAdmin, getUserRoleDisplayName } from '../utils/permissions';
import LanguageSwitcher from './LanguageSwitcher';
import ThemeSwitcher from './ThemeSwitcher';
import { useI18n } from '../hooks/useI18n';
import { useTheme } from '../hooks/useTheme';

const { Header, Content, Footer } = AntLayout;
const { Title } = Typography;

const Layout = ({ children, user, onLogout }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const { t } = useI18n();
  const { isDark } = useTheme();
  
  // AI 助手面板状态
  const [aiPanelWidth, setAiPanelWidth] = useState(0);

  // 监听 AI 助手面板状态变化
  useEffect(() => {
    const handleAiPanelChange = (event) => {
      const { visible, width } = event.detail;
      setAiPanelWidth(visible ? width : 0);
    };

    window.addEventListener('ai-assistant-panel-change', handleAiPanelChange);
    return () => {
      window.removeEventListener('ai-assistant-panel-change', handleAiPanelChange);
    };
  }, []);

  // 获取用户权限信息
  const userIsAdmin = isAdmin(user);
  const availableMenuItems = getAvailableMenuItems(user);
  const userRoleDisplayName = getUserRoleDisplayName(user);

  console.log('=== Layout权限检查 ===');
  console.log('用户信息:', user);
  console.log('用户角色:', user?.role);
  console.log('用户权限组:', user?.roles);
  console.log('角色模板:', user?.role_template || user?.roleTemplate);
  console.log('是否管理员:', userIsAdmin);
  console.log('可用菜单项:', availableMenuItems);
  console.log('当前路径:', location.pathname);
  console.log('========================');

  // 完整的菜单项配置
  const allMenuItems = [
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
      key: '/monitoring',
      icon: <DashboardOutlined />,
      label: '监控仪表板',
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
      label: 'SLURM',
    },
    {
      key: '/jobs',
      icon: <FileTextOutlined />,
      label: '作业管理',
    },
    {
      key: '/job-templates',
      icon: <CodeOutlined />,
      label: '作业模板',
    },
    {
      key: '/ssh-test',
      icon: <KeyOutlined />,
      label: 'SSH测试',
    },
    {
      key: '/files',
      icon: <FileTextOutlined />,
      label: '文件管理',
    },
    {
      key: '/object-storage',
      icon: <DatabaseOutlined />,
      label: '对象存储',
    },
    {
      key: '/saltstack',
      icon: <CustomMenuIcons.Menu size={16} />,
      label: 'SaltStack',
    },
    {
      key: '/argocd',
      icon: <DeploymentUnitOutlined />,
      label: 'ArgoCD',
    },
    {
      key: '/keycloak',
      icon: <SafetyCertificateOutlined />,
      label: 'Keycloak',
    },
    {
      key: '/kafka-ui',
      icon: <CloudServerOutlined />,
      label: 'Kafka UI',
    },
    {
      key: '/ai-chat',
      icon: <RobotOutlined />,
      label: 'AI 助手',
    },
  ];

  // 根据用户权限过滤菜单项
  const menuItems = allMenuItems.filter(item => {
    const menuKey = item.key.replace('/', '');
    return availableMenuItems.includes(menuKey) || availableMenuItems.includes(item.key);
  });

  // 管理中心下拉菜单项
  const adminMenuItems = [
    {
      key: '/admin/users',
      icon: <UserOutlined />,
      label: t('nav.userManagement'),
      onClick: () => navigate('/admin/users'),
    },
    {
      key: '/admin/projects',
      icon: <ProjectOutlined />,
      label: t('nav.projectManagement'),
      onClick: () => navigate('/admin/projects'),
    },
    {
      key: '/admin/ldap',
      icon: <TeamOutlined />,
      label: t('nav.ldapManagement'),
      onClick: () => navigate('/admin/ldap'),
    },
    {
      key: '/admin/test',
      icon: <ExperimentOutlined />,
      label: t('nav.systemTest'),
      onClick: () => navigate('/admin/test'),
    },
    {
      key: '/admin/trash',
      icon: <DeleteOutlined />,
      label: t('nav.trash'),
      onClick: () => navigate('/admin/trash'),
    },
    {
      key: '/admin/ai-assistant',
      icon: <CustomMenuIcons.AIAssistant />,
      label: t('nav.aiAssistant'),
      onClick: () => navigate('/admin/ai-assistant'),
    },
    {
      key: '/admin/jupyterhub',
      icon: <CloudServerOutlined />,
      label: t('nav.jupyterhubManagement'),
      onClick: () => navigate('/admin/jupyterhub'),
    },
    {
      key: '/admin/object-storage',
      icon: <DatabaseOutlined />,
      label: t('nav.objectStorageManagement'),
      onClick: () => navigate('/admin/object-storage'),
    },
    {
      key: '/admin/security',
      icon: <SafetyOutlined />,
      label: t('nav.securityManagement'),
      onClick: () => navigate('/admin/security'),
    },
  ];

  const handleMenuClick = ({ key }) => {
    // 特殊处理JupyterHub访问
    if (key === '/jupyterhub') {
      // Navigate to embedded Jupyter page for consistent UX
      navigate('/jupyter');
      return;
    }
    
    // 监控仪表板直接导航
    if (key === '/monitoring') {
      navigate('/monitoring');
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
    } else if (pathname === '/monitoring' || pathname.startsWith('/monitoring')) {
      selectedKeys = ['/monitoring'];
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
    } else if (pathname === '/jobs' || pathname.startsWith('/jobs')) {
      selectedKeys = ['/jobs'];
    } else if (pathname === '/job-templates' || pathname.startsWith('/job-templates')) {
      selectedKeys = ['/job-templates'];
    } else if (pathname === '/files' || pathname.startsWith('/files')) {
      selectedKeys = ['/files'];
    } else if (pathname === '/object-storage' || pathname.startsWith('/object-storage')) {
      selectedKeys = ['/object-storage'];
    } else if (pathname === '/saltstack' || pathname.startsWith('/saltstack')) {
      selectedKeys = ['/saltstack'];
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
      label: t('nav.profile'),
      onClick: () => navigate('/profile'),
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: t('nav.logout'),
      onClick: onLogout,
    },
  ];

  return (
    <AntLayout style={{
      marginLeft: aiPanelWidth,
      transition: 'margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
      minHeight: '100vh',
    }}>
      <Header style={{ 
        display: 'flex', 
        alignItems: 'center', 
        background: isDark ? '#141414' : '#001529',
        padding: '0 24px',
        minWidth: '1200px', // 设置最小宽度防止挤压
        overflow: 'hidden', // 防止内容溢出
        transition: 'background 0.3s ease',
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
        {userIsAdmin && (
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
              {t('nav.adminCenter')} <DownOutlined style={{ marginLeft: '4px', fontSize: '12px' }} />
            </Button>
          </Dropdown>
        )}

        {/* 主题切换器 */}
        <ThemeSwitcher size="small" showLabel={false} darkMode={true} />

        {/* 语言切换器 */}
        <LanguageSwitcher size="small" showLabel={true} darkMode={true} />

        {/* 右侧用户菜单 */}
        <Dropdown
          menu={{ items: userMenuItems }}
          placement="bottomRight"
          trigger={['click']}
        >
          <Space style={{ cursor: 'pointer', color: '#fff', marginLeft: '16px' }}>
            <Avatar icon={<UserOutlined />} />
            <span>{user?.username}</span>
            {userIsAdmin && (
              <span style={{ 
                background: '#52c41a', 
                padding: '2px 6px', 
                borderRadius: '4px',
                fontSize: '12px' 
              }}>
                {t('nav.admin')}
              </span>
            )}
            {!userIsAdmin && userRoleDisplayName !== '未知用户' && (
              <span style={{ 
                background: '#1890ff', 
                padding: '2px 6px', 
                borderRadius: '4px',
                fontSize: '12px' 
              }}>
                {userRoleDisplayName}
              </span>
            )}
          </Space>
        </Dropdown>
      </Header>
      
      <Content className={`ant-layout-content ${isDark ? 'theme-dark' : 'theme-light'}`}>
        {children}
      </Content>
      
      <Footer style={{ 
        textAlign: 'center',
        background: isDark ? '#1f1f1f' : '#f0f2f5',
        borderTop: isDark ? '1px solid #303030' : '1px solid #e8e8e8',
        color: isDark ? '#ffffff85' : 'inherit',
        transition: 'all 0.3s ease',
      }}>
        AI-Infra-Matrix ©2025 Created by DevOps Team
      </Footer>
    </AntLayout>
  );
};

export default Layout;
