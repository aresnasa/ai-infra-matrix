import React, { useState, useEffect } from 'react';
import { Menu, Button, Modal, Space, Switch, Card, Typography, message, Tooltip } from 'antd';
import { 
  SettingOutlined, 
  DragOutlined, 
  EyeOutlined, 
  EyeInvisibleOutlined,
  SaveOutlined,
  UndoOutlined,
  ProjectOutlined,
  ExperimentOutlined,
  CodeOutlined,
  CloudServerOutlined,
  FileTextOutlined,
  ExperimentTwoTone,
  ClusterOutlined,
  TeamOutlined,
  ControlOutlined,
  ApiOutlined,
  MenuOutlined
} from '@ant-design/icons';
import { DragDropContext, Droppable, Draggable } from 'react-beautiful-dnd';
import { navigationAPI } from '../services/api';

const { Title, Text } = Typography;

// 默认导航项配置
const DEFAULT_NAV_ITEMS = [
  {
    id: 'projects',
    key: '/projects',
    label: '项目管理',
    icon: 'ProjectOutlined',
    visible: true,
    order: 0,
    roles: ['user', 'admin', 'super-admin']
  },
  {
    id: 'gitea',
    key: '/gitea',
    label: 'Gitea',
    icon: 'CodeOutlined',
    visible: true,
    order: 1,
    roles: ['user', 'admin', 'super-admin']
  },
  {
    id: 'kubernetes',
    key: '/kubernetes',
    label: 'Kubernetes',
    icon: 'CloudServerOutlined',
    visible: true,
    order: 2,
    roles: ['admin', 'super-admin']
  },
  {
    id: 'ansible',
    key: '/ansible',
    label: 'Ansible',
    icon: 'FileTextOutlined',
    visible: true,
    order: 3,
    roles: ['admin', 'super-admin']
  },
  {
    id: 'jupyterhub',
    key: '/jupyterhub',
    label: 'JupyterHub',
    icon: 'ExperimentTwoTone',
    visible: true,
    order: 4,
    roles: ['user', 'admin', 'super-admin']
  },
  {
    id: 'slurm',
    key: '/slurm',
    label: 'Slurm',
    icon: 'ClusterOutlined',
    visible: true,
    order: 5,
    roles: ['admin', 'super-admin']
  },
  {
    id: 'saltstack',
    key: '/saltstack',
    label: 'SaltStack',
    icon: 'ControlOutlined',
    visible: true,
    order: 6,
    roles: ['admin', 'super-admin']
  }
];

const CustomizableNavigation = ({ user, selectedKeys, onMenuClick, children }) => {
  const [navItems, setNavItems] = useState(DEFAULT_NAV_ITEMS);
  const [configModalVisible, setConfigModalVisible] = useState(false);
  const [loading, setLoading] = useState(false);
  const [isDragging, setIsDragging] = useState(false);

  // 检查用户权限
  const hasRole = (requiredRoles, userRoles = []) => {
    if (!Array.isArray(requiredRoles) || requiredRoles.length === 0) return true;
    if (!Array.isArray(userRoles) || userRoles.length === 0) return false;
    return requiredRoles.some(role => userRoles.includes(role));
  };  // 加载用户自定义导航配置
  const loadNavigationConfig = async () => {
    try {
      setLoading(true);
      const response = await navigationAPI.getUserNavigationConfig();
      if (response.data && response.data.data && response.data.data.length > 0) {
        console.log('加载用户自定义导航配置:', response.data.data);
        // 确保每个导航项的roles是数组格式
        const dataArray = Array.isArray(response.data.data) ? response.data.data : [];
        const formattedItems = dataArray.map(item => ({
          ...item,
          roles: Array.isArray(item.roles) ? item.roles : (item.roles ? [item.roles] : [])
        }));
        setNavItems(formattedItems);
      } else {
        console.log('使用默认导航配置');
        setNavItems(DEFAULT_NAV_ITEMS);
      }
    } catch (error) {
      console.error('加载导航配置失败:', error);
      console.log('使用默认导航配置');
      setNavItems(DEFAULT_NAV_ITEMS);
    } finally {
      setLoading(false);
    }
  };

  // 保存导航配置
  const saveNavigationConfig = async () => {
    try {
      setLoading(true);
      console.log('保存导航配置:', navItems);
      const response = await navigationAPI.saveUserNavigationConfig(navItems);
      console.log('保存响应:', response);
      message.success('导航配置已保存');
      setConfigModalVisible(false);
    } catch (error) {
      console.error('保存配置失败:', error);
      message.error('保存配置失败: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  // 重置为默认配置
  const resetToDefault = () => {
    setNavItems(DEFAULT_NAV_ITEMS);
    message.info('已重置为默认配置');
  };

  // 处理拖拽结束
  const handleDragEnd = (result) => {
    setIsDragging(false);
    
    if (!result.destination) {
      return;
    }

    const items = Array.from(navItems);
    const [reorderedItem] = items.splice(result.source.index, 1);
    items.splice(result.destination.index, 0, reorderedItem);

    // 更新order字段
    const updatedItems = items.map((item, index) => ({
      ...item,
      order: index
    }));

    setNavItems(updatedItems);
  };

  // 切换导航项可见性
  const toggleItemVisibility = (itemId) => {
    setNavItems(prev => prev.map(item => 
      item.id === itemId ? { ...item, visible: !item.visible } : item
    ));
  };

  // 获取可见且有权限的导航项
  const getVisibleNavItems = () => {
    const userRoles = user?.roles?.map(r => r.name) || [];
    return navItems
      .filter(item => item.visible && hasRole(item.roles, userRoles))
      .sort((a, b) => a.order - b.order);
  };

  // 渲染配置模态框
  const renderConfigModal = () => (
    <Modal
      title="自定义导航栏"
      open={configModalVisible}
      onCancel={() => setConfigModalVisible(false)}
      width={800}
      footer={[
        <Button key="reset" onClick={resetToDefault}>
          <UndoOutlined /> 重置默认
        </Button>,
        <Button key="cancel" onClick={() => setConfigModalVisible(false)}>
          取消
        </Button>,
        <Button 
          key="save" 
          type="primary" 
          loading={loading}
          onClick={saveNavigationConfig}
        >
          <SaveOutlined /> 保存配置
        </Button>
      ]}
    >
      <div style={{ marginBottom: 16 }}>
        <Text type="secondary">
          拖拽下方卡片可重新排序导航项，点击眼睛图标可显示/隐藏导航项
        </Text>
      </div>
      
      <DragDropContext onDragEnd={handleDragEnd} onDragStart={() => setIsDragging(true)}>
        <Droppable droppableId="navigation-items">
          {(provided) => (
            <div {...provided.droppableProps} ref={provided.innerRef}>
              {navItems.map((item, index) => (
                <Draggable key={item.id} draggableId={item.id} index={index}>
                  {(provided, snapshot) => (
                    <Card
                      ref={provided.innerRef}
                      {...provided.draggableProps}
                      size="small"
                      style={{
                        marginBottom: 8,
                        opacity: !hasRole(item.roles, user?.roles?.map(r => r.name) || []) ? 0.5 : 1,
                        backgroundColor: snapshot.isDragging ? '#f0f0f0' : '#fff',
                        transform: snapshot.isDragging ? 'rotate(5deg)' : 'none',
                        ...provided.draggableProps.style
                      }}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <div style={{ display: 'flex', alignItems: 'center' }}>
                          <div {...provided.dragHandleProps} style={{ marginRight: 12, cursor: 'grab' }}>
                            <DragOutlined style={{ color: '#999' }} />
                          </div>
                          <span>{item.label}</span>
                          {!hasRole(item.roles, user?.roles?.map(r => r.name) || []) && (
                            <Text type="secondary" style={{ marginLeft: 8 }}>
                              (无权限)
                            </Text>
                          )}
                        </div>
                        <div>
                          <Tooltip title={item.visible ? "点击隐藏" : "点击显示"}>
                            <Button
                              type="text"
                              size="small"
                              disabled={!hasRole(item.roles, user?.roles?.map(r => r.name) || [])}
                              icon={item.visible ? <EyeOutlined /> : <EyeInvisibleOutlined />}
                              onClick={() => toggleItemVisibility(item.id)}
                            />
                          </Tooltip>
                        </div>
                      </div>
                    </Card>
                  )}
                </Draggable>
              ))}
              {provided.placeholder}
            </div>
          )}
        </Droppable>
      </DragDropContext>
    </Modal>
  );

  // 图标映射
  const iconMap = {
    'ProjectOutlined': <ProjectOutlined />,
    'ExperimentOutlined': <ExperimentOutlined />,
    'CodeOutlined': <CodeOutlined />,
    'CloudServerOutlined': <CloudServerOutlined />,
    'FileTextOutlined': <FileTextOutlined />,
    'ExperimentTwoTone': <ExperimentTwoTone />,
    'ClusterOutlined': <ClusterOutlined />,
    'SettingOutlined': <SettingOutlined />,
    'TeamOutlined': <TeamOutlined />,
    'ControlOutlined': <ControlOutlined />,
    'ApiOutlined': <ApiOutlined />,
    'MenuOutlined': <MenuOutlined />
  };

  useEffect(() => {
    if (user && user.id) {
      console.log('用户信息已加载，开始加载导航配置:', user);
      loadNavigationConfig();
    } else {
      console.log('等待用户信息加载...');
    }
  }, [user]);

  const visibleItems = getVisibleNavItems();

  return (
    <>
      <div style={{ 
        display: 'flex', 
        alignItems: 'center', 
        flex: 1,
        overflow: 'hidden' // 防止溢出
      }}>
        <Menu
          theme="dark"
          mode="horizontal"
          selectedKeys={selectedKeys}
          items={visibleItems.map(item => ({
            key: item.key,
            label: item.label,
            icon: iconMap[item.icon] || <ProjectOutlined />
          }))}
          onClick={onMenuClick}
          style={{ 
            minWidth: 0, // 允许收缩
            borderBottom: 'none',
            flex: 1,
            overflow: 'hidden' // 防止Menu项溢出
          }}
        />
        
        {/* 导航配置按钮 */}
        <Tooltip title="自定义导航栏">
          <Button
            type="text"
            size="small"
            icon={<SettingOutlined />}
            onClick={() => setConfigModalVisible(true)}
            style={{
              color: '#fff',
              marginLeft: 8,
              opacity: 0.7,
              flexShrink: 0 // 不允许收缩
            }}
          />
        </Tooltip>
      </div>
      
      {children}
      {renderConfigModal()}
    </>
  );
};

export default CustomizableNavigation;
