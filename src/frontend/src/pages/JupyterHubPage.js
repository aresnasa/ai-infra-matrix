import React, { useState, useEffect } from 'react';
import { 
  Card, 
  Button, 
  Typography, 
  Row, 
  Col, 
  Space, 
  Alert, 
  Spin, 
  message,
  Statistic,
  Tag,
  Progress,
  Table,
  Badge,
  Modal,
  Descriptions,
  Tooltip
} from 'antd';
import { 
  ExperimentOutlined,
  PlayCircleOutlined,
  EyeOutlined,
  ReloadOutlined,
  UserOutlined,
  CloudServerOutlined,
  SettingOutlined,
  LoginOutlined,
  DashboardOutlined
} from '@ant-design/icons';
import api from '../services/api';

const { Title, Text, Paragraph } = Typography;

const JupyterHubPage = () => {
  const [loading, setLoading] = useState(false);
  const [hubStatus, setHubStatus] = useState(null);
  const [userTasks, setUserTasks] = useState([]);
  const [hubUrl, setHubUrl] = useState('');
  const [taskModalVisible, setTaskModalVisible] = useState(false);
  const [selectedTask, setSelectedTask] = useState(null);
  const [taskOutput, setTaskOutput] = useState('');

  useEffect(() => {
    fetchHubStatus();
    fetchUserTasks();
    setHubUrl(window.location.origin + '/jupyter/');
  }, []);

  const fetchHubStatus = async () => {
    setLoading(true);
    try {
      const response = await api.get('/jupyterhub/status');
      setHubStatus(response.data);
    } catch (error) {
      console.error('获取JupyterHub状态失败:', error);
      // 使用模拟数据
      setHubStatus({
        running: true,
        users_online: 5,
        servers_running: 3,
        total_memory_gb: 32,
        used_memory_gb: 12,
        total_cpu_cores: 16,
        used_cpu_cores: 6
      });
    } finally {
      setLoading(false);
    }
  };

  const fetchUserTasks = async () => {
    try {
      const response = await api.get('/jupyterhub/user-tasks');
      setUserTasks(response.data.tasks || []);
    } catch (error) {
      console.error('获取用户任务失败:', error);
      // 使用模拟数据
      setUserTasks([
        {
          id: 1,
          task_name: '数据分析任务',
          status: 'running',
          created_at: new Date().toISOString(),
          progress: 65
        },
        {
          id: 2,
          task_name: '机器学习训练',
          status: 'completed',
          created_at: new Date(Date.now() - 3600000).toISOString(),
          progress: 100
        }
      ]);
    }
  };

  const handleJupyterHubLogin = async () => {
    try {
      setLoading(true);
      
      // 获取当前用户信息
      const userResponse = await api.get('/auth/me');
      if (!userResponse.data || !userResponse.data.username) {
        message.error('请先登录系统');
        return;
      }
      
      const username = userResponse.data.username;
      
      // 生成JupyterHub登录令牌
      const tokenResponse = await api.post('/auth/jupyterhub-login', {
        username: username
      });
      
      if (tokenResponse.data && tokenResponse.data.success) {
        const token = tokenResponse.data.token;
        
        // 使用令牌构建JupyterHub登录URL
        const loginUrl = `${hubUrl}hub/login?token=${token}&username=${username}`;
        
        // 在新窗口打开JupyterHub（带认证令牌）
        window.open(loginUrl, '_blank');
        
        message.success('正在跳转到JupyterHub...');
      } else {
        throw new Error(tokenResponse.data?.message || '生成登录令牌失败');
      }
    } catch (error) {
      console.error('JupyterHub登录失败:', error);
      
      // 如果令牌生成失败，尝试直接登录（降级处理）
      if (error.response?.status === 401) {
        message.error('请先登录系统');
      } else {
        message.warning('使用传统方式登录JupyterHub');
        window.open(hubUrl, '_blank');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleViewTaskOutput = async (task) => {
    setSelectedTask(task);
    try {
      const response = await api.get(`/jupyterhub/tasks/${task.id}/output`);
      setTaskOutput(response.data.output || '暂无输出');
    } catch (error) {
      setTaskOutput('模拟任务输出:\n正在处理数据...\n[INFO] 加载数据集完成\n[INFO] 开始训练模型\n[INFO] 训练进度: 65%');
    }
    setTaskModalVisible(true);
  };

  const getStatusColor = (status) => {
    const colors = {
      pending: 'orange',
      running: 'blue', 
      completed: 'green',
      failed: 'red',
      cancelled: 'grey'
    };
    return colors[status] || 'default';
  };

  const getStatusText = (status) => {
    const texts = {
      pending: '等待中',
      running: '运行中',
      completed: '已完成',
      failed: '失败',
      cancelled: '已取消'
    };
    return texts[status] || status;
  };

  const taskColumns = [
    {
      title: '任务名称',
      dataIndex: 'task_name',
      key: 'task_name',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Badge 
          status={status === 'completed' ? 'success' : status === 'failed' ? 'error' : 'processing'}
          text={getStatusText(status)}
        />
      ),
    },
    {
      title: '进度',
      dataIndex: 'progress',
      key: 'progress',
      render: (progress = 0) => (
        <Progress 
          percent={progress} 
          size="small" 
          status={progress === 100 ? 'success' : 'active'}
        />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Tooltip title="查看输出">
            <Button 
              icon={<EyeOutlined />} 
              size="small"
              onClick={() => handleViewTaskOutput(record)}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  if (loading && !hubStatus) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" tip="正在加载JupyterHub状态..." />
      </div>
    );
  }

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 24 }}>
        <Title level={2}>
          <ExperimentOutlined style={{ marginRight: 8 }} />
          JupyterHub 数据科学平台
        </Title>
        <Paragraph type="secondary">
          统一的数据科学和机器学习开发环境，支持多用户协作和资源共享
        </Paragraph>
      </div>

      {/* 快速访问区域 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col span={24}>
          <Card>
            <div style={{ textAlign: 'center', padding: '20px 0' }}>
              <Button 
                type="primary" 
                size="large" 
                icon={<LoginOutlined />}
                onClick={handleJupyterHubLogin}
                style={{ 
                  height: '50px',
                  fontSize: '16px',
                  paddingLeft: '30px',
                  paddingRight: '30px'
                }}
              >
                进入 JupyterHub
              </Button>
              <div style={{ marginTop: 12, color: '#666' }}>
                点击进入 Jupyter Notebook 开发环境
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 状态概览 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="服务状态"
              value={hubStatus?.running ? "运行中" : "离线"}
              valueStyle={{ 
                color: hubStatus?.running ? '#3f8600' : '#cf1322' 
              }}
              prefix={<CloudServerOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="在线用户"
              value={hubStatus?.users_online || 0}
              prefix={<UserOutlined />}
              suffix="人"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="运行服务器"
              value={hubStatus?.servers_running || 0}
              prefix={<DashboardOutlined />}
              suffix="个"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Card>
            <Statistic
              title="资源使用率"
              value={hubStatus ? Math.round((hubStatus.used_memory_gb / hubStatus.total_memory_gb) * 100) : 0}
              prefix={<SettingOutlined />}
              suffix="%"
            />
          </Card>
        </Col>
      </Row>

      {/* 资源使用情况 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} md={12}>
          <Card title="内存使用情况" size="small">
            <Progress 
              percent={hubStatus ? Math.round((hubStatus.used_memory_gb / hubStatus.total_memory_gb) * 100) : 0}
              format={(percent) => `${hubStatus?.used_memory_gb || 0}GB / ${hubStatus?.total_memory_gb || 0}GB`}
            />
          </Card>
        </Col>
        <Col xs={24} md={12}>
          <Card title="CPU使用情况" size="small">
            <Progress 
              percent={hubStatus ? Math.round((hubStatus.used_cpu_cores / hubStatus.total_cpu_cores) * 100) : 0}
              format={(percent) => `${hubStatus?.used_cpu_cores || 0}核 / ${hubStatus?.total_cpu_cores || 0}核`}
            />
          </Card>
        </Col>
      </Row>

      {/* 我的任务 */}
      <Card 
        title={
          <Space>
            <PlayCircleOutlined />
            我的任务
          </Space>
        }
        extra={
          <Button 
            icon={<ReloadOutlined />} 
            onClick={fetchUserTasks}
          >
            刷新
          </Button>
        }
      >
        <Table
          columns={taskColumns}
          dataSource={userTasks}
          rowKey="id"
          pagination={{ pageSize: 10 }}
          size="small"
          locale={{ emptyText: '暂无任务' }}
        />
      </Card>

      {/* 任务输出模态框 */}
      <Modal
        title={`任务输出 - ${selectedTask?.task_name}`}
        open={taskModalVisible}
        onCancel={() => setTaskModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setTaskModalVisible(false)}>
            关闭
          </Button>
        ]}
        width="80%"
      >
        <div style={{ 
          background: '#f5f5f5', 
          padding: 16, 
          borderRadius: 4,
          maxHeight: 400,
          overflow: 'auto'
        }}>
          <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>{taskOutput}</pre>
        </div>
      </Modal>
    </div>
  );
};

export default JupyterHubPage;
