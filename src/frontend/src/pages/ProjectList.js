import React, { useState, useEffect } from 'react';
import { 
  Card, 
  Button, 
  Modal, 
  Form, 
  Input, 
  message, 
  Row, 
  Col,
  Typography,
  Spin,
  Empty,
  Alert
} from 'antd';
import { 
  PlusOutlined, 
  EditOutlined, 
  DeleteOutlined,
  DesktopOutlined,
  SettingOutlined,
  PlayCircleOutlined,
  EyeOutlined,
  ReloadOutlined
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { projectAPI } from '../services/api';
import { usePageAPIStatus } from '../hooks/usePageAPIStatus';
import EnhancedLoading from '../components/EnhancedLoading';
import { useI18n } from '../hooks/useI18n';

const { Title, Paragraph } = Typography;
const { TextArea } = Input;

const ProjectList = ({ onError, retryCount }) => {
  const { t, locale } = useI18n();
  const [localLoading, setLocalLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingProject, setEditingProject] = useState(null);
  const [form] = Form.useForm();
  const navigate = useNavigate();

  // 使用API状态监控
  const {
    loading,
    error,
    data: projectsData,
    retry,
    refresh,
    lastUpdate
  } = usePageAPIStatus([
    () => projectAPI.getProjects()
  ], [retryCount]);

  const projects = projectsData?.[0]?.data || [];

  // 处理错误
  useEffect(() => {
    if (error && onError) {
      onError(error);
    }
  }, [error, onError]);

  // 打开创建/编辑模态框
  const openModal = (project = null) => {
    setEditingProject(project);
    if (project) {
      form.setFieldsValue(project);
    } else {
      form.resetFields();
    }
    setModalVisible(true);
  };

  // 保存项目
  const handleSave = async (values) => {
    setLocalLoading(true);
    try {
      if (editingProject) {
        await projectAPI.updateProject(editingProject.id, values);
        message.success(t('projects.updateSuccess'));
      } else {
        await projectAPI.createProject(values);
        message.success(t('projects.createSuccess'));
      }
      setModalVisible(false);
      form.resetFields();
      setEditingProject(null);
      refresh(); // 使用新的refresh方法
    } catch (error) {
      message.error(editingProject ? t('projects.updateFailed') : t('projects.createFailed'));
      console.error('Error saving project:', error);
    } finally {
      setLocalLoading(false);
    }
  };

  // 删除项目
  const handleDelete = async (id) => {
    Modal.confirm({
      title: t('projects.confirmDelete'),
      content: t('projects.confirmDeleteDesc'),
      okText: t('common.delete'),
      okType: 'danger',
      cancelText: t('common.cancel'),
      onOk: async () => {
        setLocalLoading(true);
        try {
          await projectAPI.deleteProject(id);
          message.success(t('projects.deleteSuccess'));
          refresh(); // 使用新的refresh方法
        } catch (error) {
          message.error(t('projects.deleteFailed'));
          console.error('Error deleting project:', error);
        } finally {
          setLocalLoading(false);
        }
      },
    });
  };

  // 查看项目详情
  const viewProject = (id) => {
    navigate(`/projects/${id}`);
  };

  const formatDate = (dateString) => {
    if (!dateString) return '-';
    return new Date(dateString).toLocaleDateString(locale === 'zh-CN' ? 'zh-CN' : 'en-US');
  };

  return (
    <div className="content-container">
      <div className="page-header">
        <Title level={2}>{t('projects.title')}</Title>
        <Paragraph type="secondary">
          {t('projects.subtitle')}
        </Paragraph>
      </div>

      <div className="action-buttons">
        <Button 
          type="primary" 
          icon={<PlusOutlined />}
          onClick={() => openModal()}
          size="large"
          disabled={loading || localLoading}
        >
          {t('projects.createProject')}
        </Button>
        {error && (
          <Button 
            type="default" 
            icon={<ReloadOutlined />}
            onClick={retry}
            size="large"
          >
            {t('common.refresh')}
          </Button>
        )}
      </div>

      {/* 显示API状态 */}
      {loading && (
        <EnhancedLoading 
          text={t('projects.loadingProjects')}
          showAPIStatus={true}
        />
      )}

      {/* 显示错误信息 */}
      {error && !loading && (
        <Alert
          message={t('projects.loadFailed')}
          description={error.message}
          type="error"
          showIcon
          action={
            <Button size="small" onClick={retry}>
              {t('common.refresh')}
            </Button>
          }
          style={{ marginBottom: 16 }}
        />
      )}

      {/* 内容区域 */}
      {!loading && !error && (
        <Spin spinning={localLoading}>
          {projects.length === 0 ? (
            <Empty
              description={t('projects.noProjects')}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            >
              <Button type="primary" onClick={() => openModal()}>
                {t('projects.createFirstProject')}
              </Button>
            </Empty>
          ) : (
            <Row gutter={[16, 16]}>
              {(projects || []).map(project => (
                <Col xs={24} sm={12} lg={8} xl={6} key={project.id}>
                  <Card
                    className="project-card"
                    hoverable
                    actions={[
                      <EyeOutlined 
                        key="view" 
                        onClick={() => viewProject(project.id)}
                      />,
                      <EditOutlined 
                        key="edit" 
                        onClick={() => openModal(project)}
                      />,
                      <DeleteOutlined 
                        key="delete" 
                        onClick={() => handleDelete(project.id)}
                      />
                    ]}
                  >
                    <Card.Meta
                      avatar={<DesktopOutlined style={{ fontSize: '24px', color: '#1890ff' }} />}
                      title={project.name}
                      description={project.description || t('projects.noDescription')}
                    />
                    <div className="project-stats">
                      <div className="stat-item">
                        <SettingOutlined />
                        {t('projects.createdAt')}: {formatDate(project.created_at)}
                      </div>
                    </div>
                  </Card>
                </Col>
              ))}
            </Row>
          )}
        </Spin>
      )}

      <Modal
        title={editingProject ? t('projects.editProject') : t('projects.createProject')}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
        >
          <Form.Item
            name="name"
            label={t('projects.projectName')}
            rules={[
              { required: true, message: t('projects.projectNameRequired') },
              { max: 255, message: t('projects.projectNameMaxLength') }
            ]}
          >
            <Input placeholder={t('projects.projectNamePlaceholder')} />
          </Form.Item>
          
          <Form.Item
            name="description"
            label={t('projects.projectDescription')}
            rules={[
              { max: 1000, message: t('projects.projectDescMaxLength') }
            ]}
          >
            <TextArea 
              placeholder={t('projects.projectDescPlaceholder')} 
              rows={4}
              showCount
              maxLength={1000}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ProjectList;
