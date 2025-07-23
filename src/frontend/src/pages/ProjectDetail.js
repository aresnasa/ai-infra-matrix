import React, { useState, useEffect } from 'react';
import { 
  Typography, 
  Tabs, 
  Button, 
  message, 
  Card,
  Spin,
  Alert,
  Modal,
  Divider,
  Row,
  Col
} from 'antd';
import { 
  ArrowLeftOutlined, 
  PlayCircleOutlined,
  DownloadOutlined,
  EyeOutlined
} from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { projectAPI, playbookAPI } from '../services/api';
import HostsTab from '../components/HostsTab';
import VariablesTab from '../components/VariablesTab';
import TasksTab from '../components/TasksTab';

const { Title, Paragraph } = Typography;

const ProjectDetail = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const [project, setProject] = useState(null);
  const [loading, setLoading] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [generationResult, setGenerationResult] = useState(null);
  
  // 预览相关状态
  const [previewVisible, setPreviewVisible] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewData, setPreviewData] = useState(null);
  
  // 下载相关状态
  const [downloadLoading, setDownloadLoading] = useState(false);

  // 获取项目详情
  const fetchProject = async () => {
    setLoading(true);
    try {
      const response = await projectAPI.getProject(id);
      setProject(response.data);
    } catch (error) {
      message.error('获取项目详情失败');
      console.error('Error fetching project:', error);
      // 设置project为false来表示加载失败，区别于初始状态的null
      setProject(false);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProject();
  }, [id]);

  // 生成Playbook
  const handleGeneratePlaybook = async () => {
    setGenerating(true);
    try {
      const response = await playbookAPI.generatePlaybook(parseInt(id));
      setGenerationResult(response.data);
      message.success('Playbook生成成功！');
    } catch (error) {
      message.error('Playbook生成失败');
      console.error('Error generating playbook:', error);
    } finally {
      setGenerating(false);
    }
  };

  // 预览Playbook文件
  const handlePreviewPlaybook = async () => {
    setPreviewLoading(true);
    try {
      const response = await playbookAPI.previewPlaybook(parseInt(id));
      setPreviewData(response.data);
      setPreviewVisible(true);
    } catch (error) {
      message.error('预览失败，请先生成Playbook');
      console.error('Error previewing playbook:', error);
    } finally {
      setPreviewLoading(false);
    }
  };

  // 下载ZIP包
  const handleDownloadZip = async () => {
    setDownloadLoading(true);
    try {
      // 先生成ZIP包
      const packageResponse = await playbookAPI.generateZipPackage(parseInt(id));
      const zipPath = packageResponse.data.zip_path;
      
      // 下载ZIP包
      const downloadResponse = await playbookAPI.downloadZipPackage(zipPath);
      
      // 创建下载链接
      const url = window.URL.createObjectURL(new Blob([downloadResponse.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `${project.name}-playbook.zip`);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
      
      message.success('ZIP包下载成功！');
    } catch (error) {
      message.error('ZIP包下载失败');
      console.error('Error downloading zip:', error);
    } finally {
      setDownloadLoading(false);
    }
  };

  const tabItems = [
    {
      key: 'hosts',
      label: '主机配置',
      children: project ? <HostsTab projectId={project.id} /> : null,
    },
    {
      key: 'variables',
      label: '变量配置',
      children: project ? <VariablesTab projectId={project.id} /> : null,
    },
    {
      key: 'tasks',
      label: '任务配置',
      children: project ? <TasksTab projectId={project.id} /> : null,
    },
  ];

  if (loading) {
    return (
      <div className="content-container">
        <Spin size="large" style={{ 
          display: 'flex', 
          justifyContent: 'center', 
          marginTop: '100px' 
        }} />
      </div>
    );
  }

  if (project === false) {
    return (
      <div className="content-container">
        <Alert
          message="项目未找到"
          description="请检查项目ID是否正确"
          type="error"
          showIcon
        />
      </div>
    );
  }

  if (!project) {
    // 加载中或未初始化时不渲染任何内容
    return null;
  }

  return (
    <div className="content-container">
      <div className="page-header">
        <Button 
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate('/projects')}
          style={{ marginBottom: '16px' }}
        >
          返回项目列表
        </Button>
        
        <Title level={2}>{project.name}</Title>
        <Paragraph type="secondary">
          {project.description || '暂无描述'}
        </Paragraph>
      </div>

      <div className="tabs-content">
        <Tabs items={tabItems} />
      </div>

      <Card className="generate-section" title="生成 Ansible Playbook">
        <Paragraph>
          配置完成后，点击下方按钮生成Ansible Playbook文件。
          系统将根据您的配置自动生成playbook.yml和inventory.ini文件。
        </Paragraph>
        
        <Button
          type="primary"
          size="large"
          icon={<PlayCircleOutlined />}
          onClick={handleGeneratePlaybook}
          loading={generating}
          className="generate-button"
        >
          {generating ? '正在生成...' : '生成 Playbook'}
        </Button>

        {generationResult && (
          <div className="success-message">
            <div style={{ marginBottom: '16px' }}>
              <strong>✅ 生成成功！</strong>
            </div>
            <div style={{ marginBottom: '8px' }}>
              文件名: {generationResult.file_name}
            </div>
            <div style={{ marginBottom: '16px' }}>
              生成时间: {new Date(generationResult.created_at).toLocaleString('zh-CN')}
            </div>
            
            <Row gutter={16}>
              <Col>
                <Button
                  type="primary"
                  icon={<EyeOutlined />}
                  onClick={handlePreviewPlaybook}
                  loading={previewLoading}
                >
                  预览文件
                </Button>
              </Col>
              <Col>
                <Button
                  type="default"
                  icon={<DownloadOutlined />}
                  onClick={handleDownloadZip}
                  loading={downloadLoading}
                >
                  下载ZIP包
                </Button>
              </Col>
            </Row>
          </div>
        )}
      </Card>

      {/* 预览模态框 */}
      <Modal
        title="Playbook 文件预览"
        open={previewVisible}
        onCancel={() => setPreviewVisible(false)}
        footer={[
          <Button key="close" onClick={() => setPreviewVisible(false)}>
            关闭
          </Button>,
        ]}
        width="80%"
        style={{ minWidth: '800px' }}
      >
        {previewData && (
          <div>
            {/* Playbook YAML */}
            {previewData.playbook_yaml && (
              <div style={{ marginBottom: '24px' }}>
                <Typography.Title level={4}>playbook.yml</Typography.Title>
                <pre style={{
                  backgroundColor: '#f5f5f5',
                  padding: '16px',
                  borderRadius: '4px',
                  overflow: 'auto',
                  maxHeight: '300px',
                  fontSize: '12px',
                  lineHeight: '1.4'
                }}>
                  {previewData.playbook_yaml}
                </pre>
              </div>
            )}

            {/* Inventory INI */}
            {previewData.inventory_ini && (
              <div style={{ marginBottom: '24px' }}>
                <Typography.Title level={4}>inventory.ini</Typography.Title>
                <pre style={{
                  backgroundColor: '#f5f5f5',
                  padding: '16px',
                  borderRadius: '4px',
                  overflow: 'auto',
                  maxHeight: '300px',
                  fontSize: '12px',
                  lineHeight: '1.4'
                }}>
                  {previewData.inventory_ini}
                </pre>
              </div>
            )}

            {/* Variables YAML */}
            {previewData.variables_yaml && (
              <div style={{ marginBottom: '24px' }}>
                <Typography.Title level={4}>variables.yml</Typography.Title>
                <pre style={{
                  backgroundColor: '#f5f5f5',
                  padding: '16px',
                  borderRadius: '4px',
                  overflow: 'auto',
                  maxHeight: '300px',
                  fontSize: '12px',
                  lineHeight: '1.4'
                }}>
                  {previewData.variables_yaml}
                </pre>
              </div>
            )}

            {/* README */}
            {previewData.readme_md && (
              <div style={{ marginBottom: '24px' }}>
                <Typography.Title level={4}>README.md</Typography.Title>
                <pre style={{
                  backgroundColor: '#f5f5f5',
                  padding: '16px',
                  borderRadius: '4px',
                  overflow: 'auto',
                  maxHeight: '300px',
                  fontSize: '12px',
                  lineHeight: '1.4'
                }}>
                  {previewData.readme_md}
                </pre>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default ProjectDetail;
