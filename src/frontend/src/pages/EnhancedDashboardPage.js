import React, { useState, useEffect, useCallback } from 'react';
import { 
  Card, Row, Col, Button, Modal, Form, Input, Select, message, Typography, Space, Switch, 
  Tooltip, Avatar, Badge, Dropdown, Menu, Alert, Divider, Progress, Spin, Tag, Empty 
} from 'antd';
import { 
  DragOutlined, 
  SettingOutlined, 
  PlusOutlined, 
  EditOutlined, 
  DeleteOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  ReloadOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  UserOutlined,
  TeamOutlined,
  ShareAltOutlined,
  SaveOutlined,
  ImportOutlined,
  ExportOutlined,
  CopyOutlined,
  BulbOutlined,
  CloudSyncOutlined
} from '@ant-design/icons';
import { DragDropContext, Droppable, Draggable } from 'react-beautiful-dnd';
import { dashboardAPI, userAPI, adminAPI } from '../services/api';
import { useAuth } from '../hooks/useAuth';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;

// 扩展的iframe类型，包含权限控制
const IFRAME_TYPES = {
  JUPYTERHUB: {
    name: 'JupyterHub',
    url: '/jupyter',
    icon: '🚀',
    description: 'Jupyter Notebook 环境',
    defaultSize: { width: 12, height: 600 },
    requiresAuth: true,
    minRole: 'user',
    category: 'development'
  },
  GITEA: {
    name: 'Gitea',
    url: '/gitea',
    icon: '📚',
    description: 'Git 代码仓库',
    defaultSize: { width: 12, height: 600 },
    requiresAuth: true,
    minRole: 'user',
    category: 'development'
  },
  KUBERNETES: {
    name: 'Kubernetes',
    url: '/kubernetes',
    icon: '☸️',
    description: 'Kubernetes 集群管理',
    defaultSize: { width: 12, height: 600 },
    requiresAuth: true,
    minRole: 'admin',
    category: 'infrastructure'
  },
  ANSIBLE: {
    name: 'Ansible',
    url: '/ansible',
    icon: '🔧',
    description: 'Ansible 自动化',
    defaultSize: { width: 12, height: 600 },
    requiresAuth: true,
    minRole: 'operator',
    category: 'automation'
  },
  SLURM: {
    name: 'Slurm',
    url: '/slurm',
    icon: '🖥️',
    description: 'Slurm 计算集群',
    defaultSize: { width: 12, height: 600 },
    requiresAuth: true,
    minRole: 'user',
    category: 'compute'
  },
  SALTSTACK: {
    name: 'SaltStack',
    url: '/saltstack',
    icon: '⚡',
    description: 'SaltStack 配置管理',
    defaultSize: { width: 12, height: 600 },
    requiresAuth: true,
    minRole: 'admin',
    category: 'infrastructure'
  },
  MONITORING: {
    name: '监控面板',
    url: '/grafana',
    icon: '📊',
    description: 'Grafana 监控面板',
    defaultSize: { width: 12, height: 600 },
    requiresAuth: true,
    minRole: 'user',
    category: 'monitoring'
  },
  CUSTOM: {
    name: '自定义',
    url: '',
    icon: '🔗',
    description: '自定义 URL',
    defaultSize: { width: 12, height: 600 },
    requiresAuth: false,
    minRole: 'user',
    category: 'custom'
  }
};

// 仪表板模板
const DASHBOARD_TEMPLATES = {
  developer: {
    name: '开发者模板',
    description: '适合开发人员的工作环境',
    widgets: [
      { type: 'JUPYTERHUB', title: 'Jupyter开发环境', position: 0 },
      { type: 'GITEA', title: 'Git代码仓库', position: 1 },
      { type: 'MONITORING', title: '性能监控', position: 2 }
    ]
  },
  admin: {
    name: '管理员模板',
    description: '系统管理员全功能面板',
    widgets: [
      { type: 'KUBERNETES', title: 'K8s集群管理', position: 0 },
      { type: 'SALTSTACK', title: 'SaltStack配置', position: 1 },
      { type: 'ANSIBLE', title: 'Ansible自动化', position: 2 },
      { type: 'MONITORING', title: '系统监控', position: 3 }
    ]
  },
  researcher: {
    name: '研究员模板',
    description: '科研计算环境',
    widgets: [
      { type: 'JUPYTERHUB', title: 'Jupyter研究环境', position: 0 },
      { type: 'SLURM', title: 'Slurm计算集群', position: 1 },
      { type: 'MONITORING', title: '计算资源监控', position: 2 }
    ]
  }
};

const EnhancedDashboardPage = () => {
  const { user, isAdmin } = useAuth();
  const [widgets, setWidgets] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [templateModalVisible, setTemplateModalVisible] = useState(false);
  const [shareModalVisible, setShareModalVisible] = useState(false);
  const [editingWidget, setEditingWidget] = useState(null);
  const [form] = Form.useForm();
  const [fullscreenWidget, setFullscreenWidget] = useState(null);
  const [dashboardStats, setDashboardStats] = useState({});
  const [ldapSyncStatus, setLdapSyncStatus] = useState(null);
  const [availableTemplates, setAvailableTemplates] = useState([]);
  const [sharedDashboards, setSharedDashboards] = useState([]);
  const [autoSaveEnabled, setAutoSaveEnabled] = useState(true);

  // 加载用户的dashboard配置
  const loadDashboard = useCallback(async () => {
    setLoading(true);
    try {
      const response = await dashboardAPI.getUserDashboard();
      setWidgets(response.data.widgets || []);
      
      // 加载仪表板统计信息
      const statsResponse = await dashboardAPI.getDashboardStats();
      setDashboardStats(statsResponse.data);
      
    } catch (error) {
      console.error('加载仪表板失败:', error);
      message.warning('加载用户配置失败，使用默认模板');
      
      // 根据用户角色加载默认模板
      const userRole = getUserPrimaryRole();
      const template = getTemplateByRole(userRole);
      setWidgets(template.widgets.map((w, index) => ({
        id: `widget-${Date.now()}-${index}`,
        type: w.type,
        title: w.title,
        url: IFRAME_TYPES[w.type]?.url || '',
        size: IFRAME_TYPES[w.type]?.defaultSize || { width: 12, height: 600 },
        position: w.position,
        visible: true,
        settings: {}
      })));
    } finally {
      setLoading(false);
    }
  }, []);

  // 获取用户主要角色
  const getUserPrimaryRole = () => {
    if (!user?.roles) return 'user';
    
    const roleHierarchy = ['admin', 'operator', 'user'];
    for (const role of roleHierarchy) {
      if (user.roles.includes(role)) {
        return role;
      }
    }
    return 'user';
  };

  // 根据角色获取模板
  const getTemplateByRole = (role) => {
    if (role === 'admin') return DASHBOARD_TEMPLATES.admin;
    if (role === 'operator') return DASHBOARD_TEMPLATES.developer;
    return DASHBOARD_TEMPLATES.researcher;
  };

  // 检查用户是否有权限使用特定widget
  const hasWidgetPermission = (widgetType) => {
    const widgetInfo = IFRAME_TYPES[widgetType];
    if (!widgetInfo?.requiresAuth) return true;
    
    const userRole = getUserPrimaryRole();
    const roleLevel = { user: 1, operator: 2, admin: 3 };
    const minLevel = roleLevel[widgetInfo.minRole] || 1;
    const userLevel = roleLevel[userRole] || 1;
    
    return userLevel >= minLevel;
  };

  // 加载LDAP同步状态
  const loadLdapSyncStatus = useCallback(async () => {
    if (!isAdmin) return;
    
    try {
      const response = await adminAPI.getLDAPSyncStatus();
      setLdapSyncStatus(response.data);
    } catch (error) {
      console.error('获取LDAP同步状态失败:', error);
    }
  }, [isAdmin]);

  useEffect(() => {
    loadDashboard();
    loadLdapSyncStatus();
    
    // 设置自动刷新
    const interval = setInterval(() => {
      if (isAdmin) {
        loadLdapSyncStatus();
      }
    }, 30000);
    
    return () => clearInterval(interval);
  }, [loadDashboard, loadLdapSyncStatus, isAdmin]);

  // 拖拽结束处理
  const onDragEnd = async (result) => {
    if (!result.destination) return;

    const items = Array.from(widgets);
    const [reorderedItem] = items.splice(result.source.index, 1);
    items.splice(result.destination.index, 0, reorderedItem);

    // 更新position
    const updatedItems = items.map((item, index) => ({
      ...item,
      position: index
    }));

    setWidgets(updatedItems);

    // 自动保存（如果启用）
    if (autoSaveEnabled) {
      try {
        await dashboardAPI.updateDashboard({ widgets: updatedItems });
        message.success('布局已自动保存', 1);
      } catch (error) {
        message.error('自动保存失败');
        console.error('保存失败:', error);
      }
    }
  };

  // 应用模板
  const applyTemplate = async (templateKey) => {
    const template = DASHBOARD_TEMPLATES[templateKey];
    if (!template) return;

    Modal.confirm({
      title: '应用模板',
      content: `确定要应用"${template.name}"模板吗？这将替换当前的所有Widget配置。`,
      okText: '应用',
      cancelText: '取消',
      onOk: async () => {
        const templateWidgets = template.widgets
          .filter(w => hasWidgetPermission(w.type))
          .map((w, index) => ({
            id: `widget-${Date.now()}-${index}`,
            type: w.type,
            title: w.title,
            url: IFRAME_TYPES[w.type]?.url || '',
            size: IFRAME_TYPES[w.type]?.defaultSize || { width: 12, height: 600 },
            position: w.position,
            visible: true,
            settings: {}
          }));

        setWidgets(templateWidgets);
        
        try {
          await dashboardAPI.updateDashboard({ widgets: templateWidgets });
          message.success('模板应用成功');
          setTemplateModalVisible(false);
        } catch (error) {
          message.error('保存模板失败');
          console.error('保存失败:', error);
        }
      }
    });
  };

  // 导出配置
  const exportDashboard = () => {
    const config = {
      version: '1.0',
      user: user.username,
      exportTime: new Date().toISOString(),
      widgets: widgets
    };
    
    const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `dashboard-${user.username}-${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    
    message.success('仪表板配置已导出');
  };

  // 导入配置
  const importDashboard = (file) => {
    const reader = new FileReader();
    reader.onload = async (e) => {
      try {
        const config = JSON.parse(e.target.result);
        
        if (!config.widgets || !Array.isArray(config.widgets)) {
          throw new Error('无效的配置文件格式');
        }

        // 过滤用户无权限的widget
        const validWidgets = config.widgets.filter(w => hasWidgetPermission(w.type));
        
        Modal.confirm({
          title: '导入配置',
          content: `确定要导入配置吗？将导入 ${validWidgets.length} 个Widget，当前配置将被替换。`,
          okText: '导入',
          cancelText: '取消',
          onOk: async () => {
            setWidgets(validWidgets);
            
            try {
              await dashboardAPI.updateDashboard({ widgets: validWidgets });
              message.success('配置导入成功');
            } catch (error) {
              message.error('保存导入配置失败');
              console.error('保存失败:', error);
            }
          }
        });
        
      } catch (error) {
        message.error('解析配置文件失败');
        console.error('解析失败:', error);
      }
    };
    reader.readAsText(file);
    return false; // 阻止自动上传
  };

  // 手动保存
  const saveConfig = async () => {
    try {
      await dashboardAPI.updateDashboard({ widgets });
      message.success('配置保存成功');
    } catch (error) {
      message.error('保存失败');
      console.error('保存失败:', error);
    }
  };

  // 打开添加/编辑模态框
  const openModal = (widget = null) => {
    setEditingWidget(widget);
    if (widget) {
      form.setFieldsValue({
        type: widget.type,
        title: widget.title,
        url: widget.type === 'CUSTOM' ? widget.url : '',
        width: widget.size?.width || 12,
        height: widget.size?.height || 600,
        visible: widget.visible
      });
    } else {
      form.resetFields();
      form.setFieldsValue({
        type: 'JUPYTERHUB',
        width: 12,
        height: 600,
        visible: true
      });
    }
    setModalVisible(true);
  };

  // 保存widget
  const handleSave = async (values) => {
    // 检查权限
    if (!hasWidgetPermission(values.type)) {
      message.error('您没有权限添加此类型的Widget');
      return;
    }

    try {
      const widgetData = {
        id: editingWidget?.id || `widget-${Date.now()}`,
        type: values.type,
        title: values.title || IFRAME_TYPES[values.type]?.name,
        url: values.type === 'CUSTOM' ? values.url : IFRAME_TYPES[values.type]?.url,
        size: {
          width: values.width,
          height: values.height
        },
        position: editingWidget?.position ?? widgets.length,
        visible: values.visible,
        settings: editingWidget?.settings || {}
      };

      let updatedWidgets;
      if (editingWidget) {
        updatedWidgets = widgets.map(w => 
          w.id === editingWidget.id ? widgetData : w
        );
      } else {
        updatedWidgets = [...widgets, widgetData];
      }

      setWidgets(updatedWidgets);
      
      if (autoSaveEnabled) {
        await dashboardAPI.updateDashboard({ widgets: updatedWidgets });
      }
      
      setModalVisible(false);
      message.success(editingWidget ? 'Widget更新成功' : 'Widget添加成功');
    } catch (error) {
      message.error('保存失败');
      console.error('保存Widget失败:', error);
    }
  };

  // 删除widget
  const handleDelete = async (widgetId) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这个Widget吗？',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          const updatedWidgets = widgets.filter(w => w.id !== widgetId);
          setWidgets(updatedWidgets);
          
          if (autoSaveEnabled) {
            await dashboardAPI.updateDashboard({ widgets: updatedWidgets });
          }
          
          message.success('Widget删除成功');
        } catch (error) {
          message.error('删除失败');
          console.error('删除Widget失败:', error);
        }
      }
    });
  };

  // 切换widget可见性
  const toggleVisibility = async (widgetId) => {
    try {
      const updatedWidgets = widgets.map(w => 
        w.id === widgetId ? { ...w, visible: !w.visible } : w
      );
      setWidgets(updatedWidgets);
      
      if (autoSaveEnabled) {
        await dashboardAPI.updateDashboard({ widgets: updatedWidgets });
      }
    } catch (error) {
      message.error('更新失败');
      console.error('更新可见性失败:', error);
    }
  };

  // 刷新iframe
  const refreshIframe = (widgetId) => {
    const iframe = document.querySelector(`#iframe-${widgetId}`);
    if (iframe) {
      iframe.src = iframe.src;
      message.success('页面已刷新', 1);
    }
  };

  // 全屏切换
  const toggleFullscreen = (widget) => {
    if (fullscreenWidget?.id === widget.id) {
      setFullscreenWidget(null);
    } else {
      setFullscreenWidget(widget);
    }
  };

  // 获取用户信息显示
  const getUserDisplay = () => {
    if (!user) return null;

    const authSourceColors = {
      local: 'blue',
      ldap: 'green'
    };

    return (
      <Space>
        <Avatar icon={<UserOutlined />} size="small" />
        <span>{user.username}</span>
        <Tag color={authSourceColors[user.auth_source] || 'default'}>
          {user.auth_source === 'ldap' ? 'LDAP' : '本地'}
        </Tag>
        {user.roles && user.roles.map(role => (
          <Tag key={role} color="purple">{role}</Tag>
        ))}
      </Space>
    );
  };

  // 用户菜单
  const userMenu = (
    <Menu>
      <Menu.Item key="templates" icon={<BulbOutlined />} onClick={() => setTemplateModalVisible(true)}>
        应用模板
      </Menu.Item>
      <Menu.Item key="export" icon={<ExportOutlined />} onClick={exportDashboard}>
        导出配置
      </Menu.Item>
      <Menu.Item key="share" icon={<ShareAltOutlined />} onClick={() => setShareModalVisible(true)}>
        分享配置
      </Menu.Item>
      <Menu.Divider />
      <Menu.Item key="autoSave" icon={<CloudSyncOutlined />}>
        <Space>
          自动保存
          <Switch 
            size="small" 
            checked={autoSaveEnabled} 
            onChange={setAutoSaveEnabled}
          />
        </Space>
      </Menu.Item>
    </Menu>
  );

  // 获取可用的widget类型（基于权限）
  const getAvailableWidgetTypes = () => {
    return Object.entries(IFRAME_TYPES)
      .filter(([key, value]) => hasWidgetPermission(key))
      .map(([key, value]) => ({ key, ...value }));
  };

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '400px' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      {/* 页头 */}
      <div style={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center', 
        marginBottom: '24px',
        flexWrap: 'wrap',
        gap: '16px'
      }}>
        <div>
          <Title level={2} style={{ margin: 0 }}>
            我的工作台
          </Title>
          <Text type="secondary">
            {getUserDisplay()}
          </Text>
        </div>
        
        <Space wrap>
          {/* LDAP同步状态 */}
          {isAdmin && ldapSyncStatus && (
            <Badge 
              status={ldapSyncStatus.status === 'running' ? 'processing' : 'success'} 
              text={`LDAP: ${ldapSyncStatus.status === 'running' ? '同步中' : '就绪'}`}
            />
          )}
          
          {/* 仪表板统计 */}
          {dashboardStats.totalWidgets && (
            <Badge count={dashboardStats.totalWidgets} color="blue" title="总Widget数量" />
          )}
          
          {!autoSaveEnabled && (
            <Button icon={<SaveOutlined />} onClick={saveConfig}>
              保存配置
            </Button>
          )}
          
          <Dropdown overlay={userMenu} trigger={['click']}>
            <Button icon={<SettingOutlined />}>
              更多操作
            </Button>
          </Dropdown>
          
          <Button 
            type="primary" 
            icon={<PlusOutlined />} 
            onClick={() => openModal()}
          >
            添加Widget
          </Button>
        </Space>
      </div>

      {/* 权限提示 */}
      {user?.auth_source === 'ldap' && (
        <Alert
          message="LDAP用户提示"
          description="您正在使用LDAP账户，某些高级功能可能需要管理员权限。如需更多权限，请联系系统管理员。"
          type="info"
          showIcon
          style={{ marginBottom: '24px' }}
          closable
        />
      )}

      {/* 拖拽区域 */}
      <DragDropContext onDragEnd={onDragEnd}>
        <Droppable droppableId="dashboard">
          {(provided) => (
            <div {...provided.droppableProps} ref={provided.innerRef}>
              {widgets.length === 0 ? (
                <Empty
                  description="暂无Widget，点击右上角添加按钮开始配置您的工作台"
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                >
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
                    添加第一个Widget
                  </Button>
                </Empty>
              ) : (
                <Row gutter={[16, 16]}>
                  {widgets
                    .sort((a, b) => (a.position || 0) - (b.position || 0))
                    .map((widget, index) => (
                      <Draggable key={widget.id} draggableId={widget.id} index={index}>
                        {(provided, snapshot) => (
                          <Col
                            span={widget.size?.width || 12}
                            ref={provided.innerRef}
                            {...provided.draggableProps}
                            style={{
                              ...provided.draggableProps.style,
                              opacity: widget.visible ? 1 : 0.6
                            }}
                          >
                            <Card
                              title={
                                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                                  <div style={{ display: 'flex', alignItems: 'center' }}>
                                    <span {...provided.dragHandleProps} style={{ marginRight: '8px', cursor: 'grab' }}>
                                      <DragOutlined />
                                    </span>
                                    <span>
                                      {IFRAME_TYPES[widget.type]?.icon} {widget.title}
                                    </span>
                                    {IFRAME_TYPES[widget.type]?.requiresAuth && (
                                      <Tag size="small" color="orange" style={{ marginLeft: '8px' }}>
                                        需要权限
                                      </Tag>
                                    )}
                                  </div>
                                  <Space>
                                    <Tooltip title={widget.visible ? '隐藏' : '显示'}>
                                      <Button 
                                        type="text" 
                                        size="small"
                                        icon={widget.visible ? <EyeOutlined /> : <EyeInvisibleOutlined />}
                                        onClick={() => toggleVisibility(widget.id)}
                                      />
                                    </Tooltip>
                                    <Tooltip title="刷新">
                                      <Button 
                                        type="text" 
                                        size="small"
                                        icon={<ReloadOutlined />}
                                        onClick={() => refreshIframe(widget.id)}
                                      />
                                    </Tooltip>
                                    <Tooltip title="全屏">
                                      <Button 
                                        type="text" 
                                        size="small"
                                        icon={<FullscreenOutlined />}
                                        onClick={() => toggleFullscreen(widget)}
                                      />
                                    </Tooltip>
                                    <Tooltip title="编辑">
                                      <Button 
                                        type="text" 
                                        size="small"
                                        icon={<EditOutlined />}
                                        onClick={() => openModal(widget)}
                                      />
                                    </Tooltip>
                                    <Tooltip title="删除">
                                      <Button 
                                        type="text" 
                                        size="small"
                                        danger
                                        icon={<DeleteOutlined />}
                                        onClick={() => handleDelete(widget.id)}
                                      />
                                    </Tooltip>
                                  </Space>
                                </div>
                              }
                              style={{
                                height: widget.visible ? 'auto' : '60px',
                                overflow: 'hidden',
                                transition: 'all 0.3s',
                                border: snapshot.isDragging ? '2px solid #1890ff' : undefined
                              }}
                              bodyStyle={{ 
                                padding: widget.visible ? '24px' : '0',
                                height: widget.visible ? `${widget.size?.height || 600}px` : '0'
                              }}
                            >
                              {widget.visible && (
                                <iframe
                                  id={`iframe-${widget.id}`}
                                  src={widget.url}
                                  style={{
                                    width: '100%',
                                    height: '100%',
                                    border: 'none',
                                    borderRadius: '6px'
                                  }}
                                  title={widget.title}
                                  onLoad={() => console.log(`Widget ${widget.title} loaded`)}
                                />
                              )}
                            </Card>
                          </Col>
                        )}
                      </Draggable>
                    ))}
                </Row>
              )}
              {provided.placeholder}
            </div>
          )}
        </Droppable>
      </DragDropContext>

      {/* 全屏模态框 */}
      <Modal
        title={
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <span>{fullscreenWidget?.title}</span>
            <Button 
              type="text" 
              icon={<FullscreenExitOutlined />}
              onClick={() => setFullscreenWidget(null)}
            />
          </div>
        }
        open={!!fullscreenWidget}
        onCancel={() => setFullscreenWidget(null)}
        footer={null}
        width="95vw"
        style={{ top: 20 }}
        bodyStyle={{ height: '85vh', padding: 0 }}
      >
        {fullscreenWidget && (
          <iframe
            src={fullscreenWidget.url}
            style={{
              width: '100%',
              height: '100%',
              border: 'none'
            }}
            title={fullscreenWidget.title}
          />
        )}
      </Modal>

      {/* 添加/编辑Widget模态框 */}
      <Modal
        title={editingWidget ? '编辑Widget' : '添加Widget'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
        okText="保存"
        cancelText="取消"
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
        >
          <Form.Item
            name="type"
            label="类型"
            rules={[{ required: true, message: '请选择Widget类型' }]}
          >
            <Select 
              placeholder="选择Widget类型"
              onChange={(value) => {
                const typeInfo = IFRAME_TYPES[value];
                if (typeInfo && value !== 'CUSTOM') {
                  form.setFieldsValue({
                    title: typeInfo.name,
                    url: ''
                  });
                }
              }}
            >
              {getAvailableWidgetTypes().map(({ key, name, icon, description, requiresAuth, minRole }) => (
                <Option key={key} value={key}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <span>{icon} {name} - {description}</span>
                    {requiresAuth && (
                      <Tag size="small" color="orange">需要{minRole}权限</Tag>
                    )}
                  </div>
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="title"
            label="标题"
            rules={[{ required: true, message: '请输入Widget标题' }]}
          >
            <Input placeholder="Widget标题" />
          </Form.Item>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) => prevValues.type !== currentValues.type}
          >
            {({ getFieldValue }) => {
              return getFieldValue('type') === 'CUSTOM' ? (
                <Form.Item
                  name="url"
                  label="自定义URL"
                  rules={[
                    { required: true, message: '请输入URL' },
                    { type: 'url', message: '请输入有效的URL' }
                  ]}
                >
                  <Input placeholder="https://example.com" />
                </Form.Item>
              ) : null;
            }}
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="width"
                label="宽度 (1-24)"
                rules={[{ required: true, message: '请输入宽度' }]}
              >
                <Input type="number" min={1} max={24} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="height"
                label="高度 (px)"
                rules={[{ required: true, message: '请输入高度' }]}
              >
                <Input type="number" min={300} max={1200} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="visible"
            label="默认显示"
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      {/* 模板选择模态框 */}
      <Modal
        title="选择仪表板模板"
        open={templateModalVisible}
        onCancel={() => setTemplateModalVisible(false)}
        footer={null}
        width={800}
      >
        <Row gutter={[16, 16]}>
          {Object.entries(DASHBOARD_TEMPLATES).map(([key, template]) => (
            <Col span={8} key={key}>
              <Card
                hoverable
                onClick={() => applyTemplate(key)}
                title={template.name}
              >
                <Paragraph type="secondary">{template.description}</Paragraph>
                <Divider />
                <Text strong>包含组件：</Text>
                <ul>
                  {template.widgets.map((widget, index) => (
                    <li key={index}>
                      {IFRAME_TYPES[widget.type]?.icon} {widget.title}
                      {!hasWidgetPermission(widget.type) && (
                        <Tag size="small" color="red" style={{ marginLeft: '8px' }}>
                          权限不足
                        </Tag>
                      )}
                    </li>
                  ))}
                </ul>
              </Card>
            </Col>
          ))}
        </Row>
      </Modal>
    </div>
  );
};

export default EnhancedDashboardPage;
