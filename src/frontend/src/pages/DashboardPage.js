import React, { useState, useEffect, useCallback } from 'react';
import { Card, Row, Col, Button, Modal, Form, Input, Select, message, Typography, Space, Switch, Tooltip } from 'antd';
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
  EyeInvisibleOutlined
} from '@ant-design/icons';
import { DragDropContext, Droppable, Draggable } from 'react-beautiful-dnd';
import { dashboardAPI } from '../services/api';

const { Title } = Typography;
const { Option } = Select;

// 预定义的iframe类型
const IFRAME_TYPES = {
  JUPYTERHUB: {
    name: 'JupyterHub',
    url: '/jupyter',
    icon: '🚀',
    description: 'Jupyter Notebook 环境',
    defaultSize: { width: 12, height: 600 }
  },
  GITEA: {
    name: 'Gitea',
    url: '/gitea',
    icon: '📚',
    description: 'Git 代码仓库',
    defaultSize: { width: 12, height: 600 }
  },
  KUBERNETES: {
    name: 'Kubernetes',
    url: '/kubernetes',
    icon: '☸️',
    description: 'Kubernetes 集群管理',
    defaultSize: { width: 12, height: 600 }
  },
  ANSIBLE: {
    name: 'Ansible',
    url: '/ansible',
    icon: '🔧',
    description: 'Ansible 自动化',
    defaultSize: { width: 12, height: 600 }
  },
  SLURM: {
    name: 'Slurm',
    url: '/slurm',
    icon: '🖥️',
    description: 'Slurm 计算集群',
    defaultSize: { width: 12, height: 600 }
  },
  SALTSTACK: {
    name: 'SaltStack',
    url: '/saltstack',
    icon: '⚡',
    description: 'SaltStack 配置管理',
    defaultSize: { width: 12, height: 600 }
  },
  CUSTOM: {
    name: '自定义',
    url: '',
    icon: '🔗',
    description: '自定义 URL',
    defaultSize: { width: 12, height: 600 }
  }
};

const DashboardPage = ({ user }) => {
  const [widgets, setWidgets] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingWidget, setEditingWidget] = useState(null);
  const [form] = Form.useForm();
  const [fullscreenWidget, setFullscreenWidget] = useState(null);

  // 加载用户的dashboard配置
  const loadDashboard = useCallback(async () => {
    setLoading(true);
    try {
      const response = await dashboardAPI.getUserDashboard();
      setWidgets(response.data.widgets || []);
    } catch (error) {
      console.error('加载仪表板失败:', error);
      // 如果没有配置，使用默认配置
      const defaultWidgets = [
        {
          id: 'widget-1',
          type: 'JUPYTERHUB',
          title: 'JupyterHub',
          url: '/jupyter',
          size: { width: 12, height: 600 },
          position: 0,
          visible: true,
          settings: {}
        },
        {
          id: 'widget-2',
          type: 'GITEA',
          title: 'Gitea',
          url: '/gitea',
          size: { width: 12, height: 600 },
          position: 1,
          visible: true,
          settings: {}
        }
      ];
      setWidgets(defaultWidgets);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadDashboard();
  }, [loadDashboard]);

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

    // 保存到后端
    try {
      await dashboardAPI.updateDashboard({ widgets: updatedItems });
      message.success('布局已保存');
    } catch (error) {
      message.error('保存布局失败');
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
      await dashboardAPI.updateDashboard({ widgets: updatedWidgets });
      
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
          await dashboardAPI.updateDashboard({ widgets: updatedWidgets });
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
      await dashboardAPI.updateDashboard({ widgets: updatedWidgets });
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

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <Title level={2}>我的工作台</Title>
        <Button 
          type="primary" 
          icon={<PlusOutlined />} 
          onClick={() => openModal()}
        >
          添加Widget
        </Button>
      </div>

      <DragDropContext onDragEnd={onDragEnd}>
        <Droppable droppableId="dashboard">
          {(provided) => (
            <div {...provided.droppableProps} ref={provided.innerRef}>
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
                              transition: 'all 0.3s'
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
                              />
                            )}
                          </Card>
                        </Col>
                      )}
                    </Draggable>
                  ))}
              </Row>
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
              {Object.entries(IFRAME_TYPES).map(([key, value]) => (
                <Option key={key} value={key}>
                  {value.icon} {value.name} - {value.description}
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
                  rules={[{ required: true, message: '请输入URL' }]}
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
    </div>
  );
};

export default DashboardPage;
