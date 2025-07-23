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
  Empty
} from 'antd';
import { 
  PlusOutlined, 
  EditOutlined, 
  DeleteOutlined,
  DesktopOutlined,
  SettingOutlined,
  PlayCircleOutlined,
  EyeOutlined
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { projectAPI } from '../services/api';

const { Title, Paragraph } = Typography;
const { TextArea } = Input;

const ProjectList = () => {
  const [projects, setProjects] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingProject, setEditingProject] = useState(null);
  const [form] = Form.useForm();
  const navigate = useNavigate();

  // 获取项目列表
  const fetchProjects = async () => {
    setLoading(true);
    try {
      const response = await projectAPI.getProjects();
      setProjects(response.data || []);
    } catch (error) {
      message.error('获取项目列表失败');
      console.error('Error fetching projects:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProjects();
  }, []);

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
    try {
      if (editingProject) {
        await projectAPI.updateProject(editingProject.id, values);
        message.success('项目更新成功');
      } else {
        await projectAPI.createProject(values);
        message.success('项目创建成功');
      }
      setModalVisible(false);
      fetchProjects();
    } catch (error) {
      message.error(editingProject ? '项目更新失败' : '项目创建失败');
      console.error('Error saving project:', error);
    }
  };

  // 删除项目
  const handleDelete = async (id) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除这个项目吗？此操作不可恢复。',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await projectAPI.deleteProject(id);
          message.success('项目删除成功');
          fetchProjects();
        } catch (error) {
          message.error('项目删除失败');
          console.error('Error deleting project:', error);
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
    return new Date(dateString).toLocaleDateString('zh-CN');
  };

  return (
    <div className="content-container">
      <div className="page-header">
        <Title level={2}>项目管理</Title>
        <Paragraph type="secondary">
          管理您的Ansible项目，配置主机、变量和任务，然后生成playbook文件
        </Paragraph>
      </div>

      <div className="action-buttons">
        <Button 
          type="primary" 
          icon={<PlusOutlined />}
          onClick={() => openModal()}
          size="large"
        >
          创建新项目
        </Button>
      </div>

      <Spin spinning={loading}>
        {projects.length === 0 ? (
          <Empty
            description="暂无项目"
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          >
            <Button type="primary" onClick={() => openModal()}>
              创建第一个项目
            </Button>
          </Empty>
        ) : (
          <Row gutter={[16, 16]}>
            {projects.map(project => (
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
                    description={project.description || '暂无描述'}
                  />
                  <div className="project-stats">
                    <div className="stat-item">
                      <SettingOutlined />
                      创建时间: {formatDate(project.created_at)}
                    </div>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        )}
      </Spin>

      <Modal
        title={editingProject ? '编辑项目' : '创建项目'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
        okText="保存"
        cancelText="取消"
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSave}
        >
          <Form.Item
            name="name"
            label="项目名称"
            rules={[
              { required: true, message: '请输入项目名称' },
              { max: 255, message: '项目名称不能超过255个字符' }
            ]}
          >
            <Input placeholder="输入项目名称" />
          </Form.Item>
          
          <Form.Item
            name="description"
            label="项目描述"
            rules={[
              { max: 1000, message: '项目描述不能超过1000个字符' }
            ]}
          >
            <TextArea 
              placeholder="输入项目描述" 
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
