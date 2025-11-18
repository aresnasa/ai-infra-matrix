import React, { useState, useEffect } from 'react';
import { Card, Button, Modal, message, Tabs, Space, Badge, Avatar } from 'antd';
import { 
  DashboardOutlined, 
  UserOutlined, 
  SettingOutlined,
  TeamOutlined,
  CloudSyncOutlined,
  BarChartOutlined 
} from '@ant-design/icons';
import EnhancedDashboardPage from '../pages/EnhancedDashboardPage';
import { authAPI, adminAPI } from '../services/api';

const { TabPane } = Tabs;

/**
 * 仪表板集成组件 - 整合了增强仪表板和LDAP多用户管理
 * 提供统一的入口来访问新的多用户功能
 */
const DashboardIntegration = () => {
  const [activeTab, setActiveTab] = useState('dashboard');
  const [currentUser, setCurrentUser] = useState(null);
  const [userStats, setUserStats] = useState(null);
  const [ldapStatus, setLdapStatus] = useState('unknown');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadInitialData();
  }, []);

  const loadInitialData = async () => {
    try {
      setLoading(true);
      
      // 获取当前用户信息
      const userResponse = await authAPI.getCurrentUser();
      setCurrentUser(userResponse.data);
      
      // 获取用户统计信息（如果是管理员）
      if (userResponse.data?.role === 'admin') {
        try {
          const statsResponse = await adminAPI.getUserStatistics();
          setUserStats(statsResponse.data);
        } catch (error) {
          console.warn('无法获取用户统计信息:', error);
        }
        
        // 检查LDAP状态
        try {
          const ldapResponse = await adminAPI.getLDAPConfig();
          setLdapStatus(ldapResponse.data?.enabled ? 'enabled' : 'disabled');
        } catch (error) {
          console.warn('无法获取LDAP状态:', error);
          setLdapStatus('error');
        }
      }
    } catch (error) {
      console.error('加载初始数据失败:', error);
      message.error('加载数据失败');
    } finally {
      setLoading(false);
    }
  };

  const handleTabChange = (key) => {
    setActiveTab(key);
  };

  const getTabBadge = (count, type = 'primary') => {
    if (count === undefined || count === null) return null;
    return <Badge count={count} style={{ backgroundColor: type === 'warning' ? '#faad14' : '#1890ff' }} />;
  };

  const getLdapStatusColor = () => {
    switch (ldapStatus) {
      case 'enabled': return '#52c41a';
      case 'disabled': return '#faad14';
      case 'error': return '#ff4d4f';
      default: return '#d9d9d9';
    }
  };

  const getLdapStatusText = () => {
    switch (ldapStatus) {
      case 'enabled': return 'LDAP已启用';
      case 'disabled': return 'LDAP已禁用';
      case 'error': return 'LDAP配置错误';
      default: return 'LDAP状态未知';
    }
  };

  if (loading) {
    return (
      <Card loading={loading} style={{ margin: 16 }}>
        <div style={{ textAlign: 'center', padding: '50px 0' }}>
          加载中...
        </div>
      </Card>
    );
  }

  return (
    <div style={{ padding: 16 }}>
      {/* 头部信息 */}
      <Card 
        size="small" 
        style={{ marginBottom: 16 }}
        title={
          <Space>
            <DashboardOutlined />
            <span>增强仪表板与用户管理</span>
            {currentUser?.role === 'admin' && (
              <Badge 
                color={getLdapStatusColor()} 
                text={getLdapStatusText()}
                style={{ marginLeft: 16 }}
              />
            )}
          </Space>
        }
        extra={
          <Space>
            <Avatar 
              size="small" 
              icon={<UserOutlined />} 
              style={{ backgroundColor: '#1890ff' }}
            />
            <span>{currentUser?.username || '未知用户'}</span>
            {currentUser?.role && (
              <Badge 
                count={currentUser.role === 'admin' ? '管理员' : '用户'} 
                style={{ 
                  backgroundColor: currentUser.role === 'admin' ? '#52c41a' : '#1890ff' 
                }}
              />
            )}
          </Space>
        }
      >
        <div style={{ fontSize: 12, color: '#666' }}>
          <Space wrap>
            <span>🎯 支持可拖拽iframe部件重新排序</span>
            <span>👥 LDAP多用户同步与权限管理</span>
            <span>📊 用户行为统计与仪表板模板</span>
            <span>🔒 基于角色的个性化配置</span>
          </Space>
        </div>
        
        {userStats && (
          <div style={{ marginTop: 8, fontSize: 12 }}>
            <Space>
              <BarChartOutlined />
              <span>总用户: {userStats.totalUsers || 0}</span>
              <span>在线用户: {userStats.onlineUsers || 0}</span>
              <span>LDAP用户: {userStats.ldapUsers || 0}</span>
            </Space>
          </div>
        )}
      </Card>

      {/* 主要内容标签页 */}
      <Card>
        <Tabs activeKey={activeTab} onChange={handleTabChange} type="card">
          <TabPane
            tab={
              <Space>
                <DashboardOutlined />
                <span>增强仪表板</span>
              </Space>
            }
            key="dashboard"
          >
            <div style={{ background: '#fafafa', padding: 16, borderRadius: 6, marginBottom: 16 }}>
              <h4 style={{ margin: 0, marginBottom: 8 }}>📋 功能特性</h4>
              <ul style={{ margin: 0, paddingLeft: 20, fontSize: 12 }}>
                <li>🔀 支持iframe部件自由拖拽重新排序</li>
                <li>👤 基于用户角色的个性化仪表板模板</li>
                <li>💾 自动保存用户配置偏好</li>
                <li>📤 支持仪表板配置导入导出</li>
                <li>🎨 支持开发者、管理员、研究员等角色模板</li>
              </ul>
            </div>
            <EnhancedDashboardPage />
          </TabPane>

          <TabPane
            tab={
              <Space>
                <SettingOutlined />
                <span>集成说明</span>
              </Space>
            }
            key="info"
          >
            <div style={{ padding: 20 }}>
              <h3>🎯 增强仪表板与LDAP多用户集成</h3>
              
              <div style={{ marginBottom: 24 }}>
                <h4>🚀 主要功能</h4>
                <ul>
                  <li><strong>可拖拽仪表板</strong>：基于react-beautiful-dnd实现的iframe部件拖拽重排</li>
                  <li><strong>LDAP集成</strong>：完整的OpenLDAP用户同步与权限管理</li>
                  <li><strong>多用户支持</strong>：基于角色的个性化仪表板配置</li>
                  <li><strong>权限管理</strong>：细粒度的部件访问控制与角色模板</li>
                </ul>
              </div>

              <div style={{ marginBottom: 24 }}>
                <h4>🔧 技术实现</h4>
                <ul>
                  <li><strong>前端</strong>：React + Ant Design + react-beautiful-dnd</li>
                  <li><strong>后端</strong>：Go/Gin + 增强仪表板控制器</li>
                  <li><strong>认证</strong>：JWT + LDAP集成</li>
                  <li><strong>存储</strong>：用户配置持久化 + 模板系统</li>
                </ul>
              </div>

              <div style={{ marginBottom: 24 }}>
                <h4>📋 使用说明</h4>
                <ol>
                  <li>在"增强仪表板"标签页中配置和管理iframe部件</li>
                  <li>拖拽部件重新排序，系统会自动保存配置</li>
                  <li>支持导入导出仪表板配置，便于团队协作</li>
                </ol>
              </div>

              <div style={{ background: '#e6f7ff', padding: 16, borderRadius: 6, border: '1px solid #91d5ff' }}>
                <h4 style={{ color: '#1890ff', margin: 0, marginBottom: 8 }}>💡 提示</h4>
                <p style={{ margin: 0, fontSize: 13 }}>
                  此集成组件展示了完整的多用户LDAP仪表板系统。
                  包含可拖拽的iframe部件、基于角色的权限控制、LDAP用户同步等高级功能。
                  所有功能都已完全实现并可直接使用。
                </p>
              </div>
            </div>
          </TabPane>
        </Tabs>
      </Card>
    </div>
  );
};

export default DashboardIntegration;
