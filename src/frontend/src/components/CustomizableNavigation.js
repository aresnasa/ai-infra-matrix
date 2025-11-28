import React, { useState, useEffect, useMemo } from 'react';
import { Menu, Button, Modal, Space, Switch, Card, Typography, message, Tooltip } from 'antd';
import { 
  SettingOutlined, 
  DragOutlined, 
  EyeOutlined, 
  EyeInvisibleOutlined,
  SaveOutlined,
  UndoOutlined,
  ProjectOutlined,
  DashboardOutlined,
  ExperimentOutlined,
  CodeOutlined,
  CloudServerOutlined,
  FileTextOutlined,
  ExperimentTwoTone,
  ClusterOutlined,
  TeamOutlined,
  ControlOutlined,
  ApiOutlined,
  MenuOutlined,
  DatabaseOutlined
} from '@ant-design/icons';
import { DragDropContext, Droppable, Draggable } from 'react-beautiful-dnd';
import { navigationAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';

const { Title, Text } = Typography;

// 默认导航项配置 - 使用 labelKey 代替 label
const DEFAULT_NAV_ITEMS = [
  {
    id: 'projects',
    key: '/projects',
    labelKey: 'nav.projects',
    icon: 'ProjectOutlined',
    visible: true,
    order: 0,
    roles: ['user', 'admin', 'super-admin']
  },
  {
    id: 'monitoring',
    key: '/monitoring',
    labelKey: 'nav.monitoring',
    icon: 'DashboardOutlined',
    visible: true,
    order: 1,
    roles: ['admin', 'super-admin']
  },
  {
    id: 'gitea',
    key: '/gitea',
    labelKey: 'nav.gitea',
    icon: 'CodeOutlined',
    visible: true,
    order: 1,
    roles: ['user', 'admin', 'super-admin']
  },
  {
    id: 'kubernetes',
    key: '/kubernetes',
    labelKey: 'nav.kubernetes',
    icon: 'CloudServerOutlined',
    visible: true,
    order: 2,
    roles: ['admin', 'super-admin']
  },
  {
    id: 'ansible',
    key: '/ansible',
    labelKey: 'nav.ansible',
    icon: 'FileTextOutlined',
    visible: true,
    order: 3,
    roles: ['admin', 'super-admin']
  },
  {
    id: 'jupyterhub',
    key: '/jupyterhub',
    labelKey: 'nav.jupyterhub',
    icon: 'ExperimentTwoTone',
    visible: true,
    order: 4,
    roles: ['user', 'admin', 'super-admin']
  },
  {
    id: 'slurm',
    key: '/slurm',
    labelKey: 'nav.slurm',
    icon: 'ClusterOutlined',
    visible: true,
    order: 5,
    roles: ['sre', 'admin', 'super-admin']
  },
  {
    id: 'object-storage',
    key: '/object-storage',
    labelKey: 'nav.objectStorage',
    icon: 'DatabaseOutlined',
    visible: true,
    order: 6,
    roles: ['data-developer', 'sre', 'admin', 'super-admin']
  },
  {
    id: 'saltstack',
    key: '/saltstack',
    labelKey: 'nav.saltstack',
    icon: 'ControlOutlined',
    visible: true,
    order: 7,
    roles: ['admin', 'super-admin']
  },
  {
    id: 'role-templates',
    key: '/admin/role-templates',
    labelKey: 'nav.roleTemplates',
    icon: 'TeamOutlined',
    visible: true,
    order: 8,
    roles: ['admin', 'super-admin']
  }
];

const CustomizableNavigation = ({ user, selectedKeys, onMenuClick, children }) => {
  const { t } = useI18n();
  const [navItems, setNavItems] = useState(DEFAULT_NAV_ITEMS);
  const [configModalVisible, setConfigModalVisible] = useState(false);
  const [loading, setLoading] = useState(false);
  const [isDragging, setIsDragging] = useState(false);

  // 获取导航项的显示标签 - 支持 labelKey 和 label 两种方式
  const getItemLabel = (item) => {
    if (item.labelKey) {
      return t(item.labelKey);
    }
    return item.label || item.id;
  };

  // 检查用户权限 - 支持检查 roles 数组和 role_template 字段
  const hasRole = (requiredRoles, userRoles = [], roleTemplate = null) => {
    if (!Array.isArray(requiredRoles) || requiredRoles.length === 0) return true;
    
    // 检查 role_template 是否匹配
    if (roleTemplate && requiredRoles.includes(roleTemplate)) {
      return true;
    }
    
    // 检查 roles 数组是否匹配
    if (!Array.isArray(userRoles) || userRoles.length === 0) return false;
    return requiredRoles.some(role => userRoles.includes(role));
  };  // 加载用户自定义导航配置
  const loadNavigationConfig = async () => {
    try {
      setLoading(true);
      const response = await navigationAPI.getUserNavigationConfig();
      // 添加更安全的数据访问，防止解构失败
      const responseData = response?.data?.data;
      if (responseData && Array.isArray(responseData) && responseData.length > 0) {
        console.log('Loading user navigation config:', responseData);
        // 确保每个导航项的roles是数组格式，并转换 label 为 labelKey
        let formattedItems = responseData.map(item => {
          const defaultItem = DEFAULT_NAV_ITEMS.find(d => d.id === item.id);
          return {
            ...item,
            labelKey: item.labelKey || defaultItem?.labelKey || `nav.${item.id}`,
            roles: Array.isArray(item.roles) ? item.roles : (item.roles ? [item.roles] : [])
          };
        });
        // 将默认项中新增的导航合并进用户配置（保持用户顺序在前）
        const existingIds = new Set(formattedItems.map(i => i.id));
        const missingDefaults = DEFAULT_NAV_ITEMS.filter(d => !existingIds.has(d.id));
        // 追加缺失的默认项到末尾，并为其分配顺序
        formattedItems = [
          ...formattedItems,
          ...missingDefaults.map((d, idx) => ({ ...d, order: (formattedItems.length + idx) }))
        ];
        setNavItems(formattedItems);
      } else {
        console.log(t('nav.usingDefaultConfig'));
        setNavItems(DEFAULT_NAV_ITEMS);
      }
    } catch (error) {
      console.error('Failed to load navigation config:', error);
      console.log(t('nav.usingDefaultConfig'));
      setNavItems(DEFAULT_NAV_ITEMS);
    } finally {
      setLoading(false);
    }
  };

  // 保存导航配置
  const saveNavigationConfig = async () => {
    try {
      setLoading(true);
      console.log('Saving navigation config:', navItems);
      const response = await navigationAPI.saveUserNavigationConfig(navItems);
      console.log('Save response:', response);
      message.success(t('nav.configSaved'));
      setConfigModalVisible(false);
    } catch (error) {
      console.error('Failed to save config:', error);
      message.error(t('nav.configSaveFailed') + ': ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  // 重置为默认配置
  const resetToDefault = () => {
    setNavItems(DEFAULT_NAV_ITEMS);
    message.info(t('nav.resetToDefault'));
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
    const roleTemplate = user?.role_template || user?.roleTemplate;
    return navItems
      .filter(item => item.visible && hasRole(item.roles, userRoles, roleTemplate))
      .sort((a, b) => a.order - b.order);
  };

  // 渲染配置模态框
  const renderConfigModal = () => (
    <Modal
      title={t('nav.customizeNav')}
      open={configModalVisible}
      onCancel={() => setConfigModalVisible(false)}
      width={800}
      footer={[
        <Button key="reset" onClick={resetToDefault}>
          <UndoOutlined /> {t('nav.resetDefault')}
        </Button>,
        <Button key="cancel" onClick={() => setConfigModalVisible(false)}>
          {t('common.cancel')}
        </Button>,
        <Button 
          key="save" 
          type="primary" 
          loading={loading}
          onClick={saveNavigationConfig}
        >
          <SaveOutlined /> {t('nav.saveConfig')}
        </Button>
      ]}
    >
      <div style={{ marginBottom: 16 }}>
        <Text type="secondary">
          {t('nav.customizeNavDesc')}
        </Text>
      </div>
      
      <DragDropContext onDragEnd={handleDragEnd} onDragStart={() => setIsDragging(true)}>
        <Droppable droppableId="navigation-items">
          {(provided) => {
            const userRoles = user?.roles?.map(r => r.name) || [];
            const roleTemplate = user?.role_template || user?.roleTemplate;
            return (
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
                        opacity: !hasRole(item.roles, userRoles, roleTemplate) ? 0.5 : 1,
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
                          <span>{getItemLabel(item)}</span>
                          {!hasRole(item.roles, userRoles, roleTemplate) && (
                            <Text type="secondary" style={{ marginLeft: 8 }}>
                              {t('nav.noPermission')}
                            </Text>
                          )}
                        </div>
                        <div>
                          <Tooltip title={item.visible ? t('nav.clickToHide') : t('nav.clickToShow')}>
                            <Button
                              type="text"
                              size="small"
                              disabled={!hasRole(item.roles, userRoles, roleTemplate)}
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
          )}}
        </Droppable>
      </DragDropContext>
    </Modal>
  );

  // 图标映射
  const iconMap = {
    'ProjectOutlined': <ProjectOutlined />,
    'DashboardOutlined': <DashboardOutlined />,
    'ExperimentOutlined': <ExperimentOutlined />,
    'CodeOutlined': <CodeOutlined />,
    'CloudServerOutlined': <CloudServerOutlined />,
    'FileTextOutlined': <FileTextOutlined />,
    'ExperimentTwoTone': <ExperimentTwoTone />,
    'ClusterOutlined': <ClusterOutlined />,
    'DatabaseOutlined': <DatabaseOutlined />,
    'SettingOutlined': <SettingOutlined />,
    'TeamOutlined': <TeamOutlined />,
    'ControlOutlined': <ControlOutlined />,
    'ApiOutlined': <ApiOutlined />,
    'MenuOutlined': <MenuOutlined />
  };

  useEffect(() => {
    if (user && user.id) {
      console.log('User info loaded, loading navigation config:', user);
      loadNavigationConfig();
    } else {
      console.log('Waiting for user info...');
    }
  }, [user]);

  // 使用 useMemo 确保导航项标签随语言变化而更新
  const visibleItems = useMemo(() => {
    return getVisibleNavItems();
  }, [navItems, user, t]);

  // 菜单项列表，使用翻译后的标签
  const menuItems = useMemo(() => {
    return visibleItems.map(item => ({
      key: item.key,
      label: getItemLabel(item),
      icon: iconMap[item.icon] || <ProjectOutlined />
    }));
  }, [visibleItems, t]);

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
          items={menuItems}
          onClick={onMenuClick}
          style={{ 
            minWidth: 0, // 允许收缩
            borderBottom: 'none',
            flex: 1,
            overflow: 'hidden' // 防止Menu项溢出
          }}
        />
        
        {/* 导航配置按钮 */}
        <Tooltip title={t('nav.customizeNav')}>
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
