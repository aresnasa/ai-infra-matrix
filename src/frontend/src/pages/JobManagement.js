import React, { useState, useEffect } from 'react';
import {
  Table, Button, Modal, Form, Input, Select, message, Tag, Space,
  Card, Row, Col, Statistic, Progress, Descriptions, Tabs, Typography, Tooltip
} from 'antd';
import {
  PlusOutlined, PlayCircleOutlined, StopOutlined,
  EyeOutlined, FileTextOutlined, ClusterOutlined,
  CheckCircleOutlined, ClockCircleOutlined, CloseCircleOutlined,
  LoadingOutlined, SyncOutlined
} from '@ant-design/icons';
import axios from 'axios';

const { Option } = Select;
const { TextArea } = Input;
const { TabPane } = Tabs;
const { Text } = Typography;

const JobManagement = () => {
  const [jobs, setJobs] = useState([]);
  const [clusters, setClusters] = useState([]);
  const [templates, setTemplates] = useState([]);
  const [loading, setLoading] = useState(false);
  const [submitModalVisible, setSubmitModalVisible] = useState(false);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [outputModalVisible, setOutputModalVisible] = useState(false);
  const [selectedJob, setSelectedJob] = useState(null);
  const [jobOutput, setJobOutput] = useState(null);
  const [selectedTemplate, setSelectedTemplate] = useState(null);
  const [stats, setStats] = useState({
    totalJobs: 0,
    runningJobs: 0,
    pendingJobs: 0,
    completedJobs: 0,
    failedJobs: 0,
    totalClusters: 0,
    activeClusters: 0
  });

  const [form] = Form.useForm();
  const [filters, setFilters] = useState({
    cluster: '',
    status: ''
  });

  // 获取作业列表
  const fetchJobs = async () => {
    setLoading(true);
    try {
      const params = {};
      if (filters.cluster) params.cluster = filters.cluster;
      if (filters.status) params.status = filters.status;

      const response = await axios.get('/api/jobs', { params });
      setJobs(response.data.data.jobs);
    } catch (error) {
      message.error('获取作业列表失败: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  // 获取集群列表
  const fetchClusters = async () => {
    try {
      const response = await axios.get('/api/clusters');
      setClusters(response.data.data);
    } catch (error) {
      message.error('获取集群列表失败: ' + error.message);
    }
  };

  // 获取模板列表
  const fetchTemplates = async () => {
    try {
      const response = await axios.get('/api/job-templates', {
        params: { page: 1, page_size: 100, is_public: true }
      });
      if (response.data.code === 200) {
        setTemplates(response.data.data.templates || []);
      }
    } catch (error) {
      console.error('获取模板列表失败:', error);
    }
  };

  // 获取统计信息
  const fetchStats = async () => {
    try {
      const response = await axios.get('/api/dashboard/stats');
      setStats(response.data.data);
    } catch (error) {
      console.error('获取统计信息失败:', error);
    }
  };

  useEffect(() => {
    fetchClusters();
    fetchTemplates();
    fetchStats();
  }, []);

  useEffect(() => {
    fetchJobs();
  }, [filters]);

  // 提交作业
  const handleSubmitJob = async (values) => {
    try {
      await axios.post('/api/jobs', values);
      message.success('作业提交成功');
      setSubmitModalVisible(false);
      form.resetFields();
      setSelectedTemplate(null);
      fetchJobs();
      fetchStats();
    } catch (error) {
      message.error('提交作业失败: ' + error.message);
    }
  };

  // 应用模板
  const applyTemplate = (templateId) => {
    const template = templates.find(t => t.id === templateId);
    if (template) {
      setSelectedTemplate(template);
      form.setFieldsValue({
        name: template.name,
        command: template.command,
        partition: template.partition,
        nodes: template.nodes,
        cpus: template.cpus,
        memory: template.memory,
        time_limit: template.time_limit,
      });
      message.success('模板应用成功');
    }
  };

  // 清除模板
  const clearTemplate = () => {
    setSelectedTemplate(null);
    form.resetFields();
  };

  // 取消作业
  const handleCancelJob = async (jobId, clusterId) => {
    try {
      await axios.post(`/api/jobs/${jobId}/cancel?cluster=${clusterId}`);
      message.success('作业取消成功');
      fetchJobs();
      fetchStats();
    } catch (error) {
      message.error('取消作业失败: ' + error.message);
    }
  };

  // 查看作业详情
  const handleViewDetail = async (jobId, clusterId) => {
    try {
      const response = await axios.get(`/api/jobs/${jobId}?cluster=${clusterId}`);
      setSelectedJob(response.data.data);
      setDetailModalVisible(true);
    } catch (error) {
      message.error('获取作业详情失败: ' + error.message);
    }
  };

  // 查看作业输出
  const handleViewOutput = async (jobId, clusterId) => {
    try {
      const response = await axios.get(`/api/jobs/${jobId}/output?cluster=${clusterId}`);
      setJobOutput(response.data.data);
      setOutputModalVisible(true);
    } catch (error) {
      message.error('获取作业输出失败: ' + error.message);
    }
  };

  // 获取状态标签
  const getStatusTag = (status) => {
    const statusConfig = {
      'PENDING': { color: 'orange', icon: <ClockCircleOutlined />, text: '等待中' },
      'RUNNING': { color: 'blue', icon: <SyncOutlined spin />, text: '运行中' },
      'COMPLETED': { color: 'green', icon: <CheckCircleOutlined />, text: '已完成' },
      'FAILED': { color: 'red', icon: <CloseCircleOutlined />, text: '失败' },
      'CANCELLED': { color: 'gray', icon: <StopOutlined />, text: '已取消' }
    };

    const config = statusConfig[status] || { color: 'default', icon: null, text: status };
    return <Tag color={config.color} icon={config.icon}>{config.text}</Tag>;
  };

  // 表格列配置
  const columns = [
    {
      title: '作业ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: '作业名称',
      dataIndex: 'name',
      key: 'name',
      ellipsis: true,
    },
    {
      title: '集群',
      dataIndex: 'cluster_id',
      key: 'cluster_id',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => getStatusTag(status),
    },
    {
      title: '分区',
      dataIndex: 'partition',
      key: 'partition',
    },
    {
      title: '节点数',
      dataIndex: 'nodes',
      key: 'nodes',
      width: 80,
    },
    {
      title: 'CPU数',
      dataIndex: 'cpus',
      key: 'cpus',
      width: 80,
    },
    {
      title: '提交时间',
      dataIndex: 'submit_time',
      key: 'submit_time',
      render: (time) => new Date(time).toLocaleString(),
      width: 160,
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => handleViewDetail(record.id, record.cluster_id)}
          >
            详情
          </Button>
          {record.status === 'RUNNING' && (
            <Button
              type="link"
              danger
              icon={<StopOutlined />}
              onClick={() => handleCancelJob(record.id, record.cluster_id)}
            >
              取消
            </Button>
          )}
          {(record.status === 'COMPLETED' || record.status === 'FAILED') && (
            <Button
              type="link"
              icon={<FileTextOutlined />}
              onClick={() => handleViewOutput(record.id, record.cluster_id)}
            >
              输出
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '20px' }}>
      <Row gutter={[16, 16]}>
        {/* 统计卡片 */}
        <Col span={24}>
          <Row gutter={16}>
            <Col span={4}>
              <Card>
                <Statistic
                  title="总作业数"
                  value={stats.totalJobs}
                  prefix={<FileTextOutlined />}
                />
              </Card>
            </Col>
            <Col span={4}>
              <Card>
                <Statistic
                  title="运行中"
                  value={stats.runningJobs}
                  prefix={<SyncOutlined />}
                  valueStyle={{ color: '#1890ff' }}
                />
              </Card>
            </Col>
            <Col span={4}>
              <Card>
                <Statistic
                  title="等待中"
                  value={stats.pendingJobs}
                  prefix={<ClockCircleOutlined />}
                  valueStyle={{ color: '#faad14' }}
                />
              </Card>
            </Col>
            <Col span={4}>
              <Card>
                <Statistic
                  title="已完成"
                  value={stats.completedJobs}
                  prefix={<CheckCircleOutlined />}
                  valueStyle={{ color: '#52c41a' }}
                />
              </Card>
            </Col>
            <Col span={4}>
              <Card>
                <Statistic
                  title="失败"
                  value={stats.failedJobs}
                  prefix={<CloseCircleOutlined />}
                  valueStyle={{ color: '#ff4d4f' }}
                />
              </Card>
            </Col>
            <Col span={4}>
              <Card>
                <Statistic
                  title="活跃集群"
                  value={stats.activeClusters}
                  prefix={<ClusterOutlined />}
                />
              </Card>
            </Col>
          </Row>
        </Col>

        {/* 作业管理 */}
        <Col span={24}>
          <Card
            title="作业管理"
            extra={
              <Space>
                <Select
                  placeholder="选择集群"
                  style={{ width: 120 }}
                  allowClear
                  onChange={(value) => setFilters({...filters, cluster: value})}
                >
                  {clusters.map(cluster => (
                    <Option key={cluster.id} value={cluster.id}>
                      {cluster.name}
                    </Option>
                  ))}
                </Select>
                <Select
                  placeholder="选择状态"
                  style={{ width: 120 }}
                  allowClear
                  onChange={(value) => setFilters({...filters, status: value})}
                >
                  <Option value="PENDING">等待中</Option>
                  <Option value="RUNNING">运行中</Option>
                  <Option value="COMPLETED">已完成</Option>
                  <Option value="FAILED">失败</Option>
                  <Option value="CANCELLED">已取消</Option>
                </Select>
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => setSubmitModalVisible(true)}
                >
                  提交作业
                </Button>
              </Space>
            }
          >
            <Table
              columns={columns}
              dataSource={jobs}
              loading={loading}
              rowKey="id"
              pagination={{
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total, range) =>
                  `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
              }}
            />
          </Card>
        </Col>
      </Row>

      {/* 提交作业模态框 */}
      <Modal
        title="提交作业"
        open={submitModalVisible}
        onCancel={() => {
          setSubmitModalVisible(false);
          form.resetFields();
          setSelectedTemplate(null);
        }}
        footer={null}
        width={800}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmitJob}
        >
          {/* 模板选择区域 */}
          <Card style={{ marginBottom: 16, backgroundColor: '#f9f9f9' }} size="small">
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <div style={{ flex: 1 }}>
                <Text strong>模板选择：</Text>
                <Select
                  placeholder="选择一个模板（可选）"
                  allowClear
                  style={{ width: '100%', marginLeft: 8 }}
                  value={selectedTemplate?.id}
                  onChange={(templateId) => {
                    if (templateId) {
                      applyTemplate(templateId);
                    } else {
                      clearTemplate();
                    }
                  }}
                >
                  {templates.map(template => (
                    <Select.Option key={template.id} value={template.id}>
                      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <span>{template.name}</span>
                        <Tag color="blue" size="small">{template.category}</Tag>
                      </div>
                    </Select.Option>
                  ))}
                </Select>
              </div>
              {selectedTemplate && (
                <Button 
                  size="small" 
                  type="link" 
                  onClick={clearTemplate}
                  style={{ marginLeft: 8 }}
                >
                  清除模板
                </Button>
              )}
            </div>
            {selectedTemplate && (
              <div style={{ marginTop: 8, padding: 8, backgroundColor: '#e6f7ff', borderRadius: 4 }}>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  已应用模板: <Text strong>{selectedTemplate.name}</Text>
                  {selectedTemplate.description && (
                    <span> - {selectedTemplate.description}</span>
                  )}
                </Text>
              </div>
            )}
          </Card>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="cluster_id"
                label="集群"
                rules={[{ required: true, message: '请选择集群' }]}
              >
                <Select placeholder="选择集群">
                  {clusters.map(cluster => (
                    <Option key={cluster.id} value={cluster.id}>
                      {cluster.name}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="name"
                label="作业名称"
                rules={[{ required: true, message: '请输入作业名称' }]}
              >
                <Input placeholder="输入作业名称" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="command"
            label="执行命令"
            rules={[{ required: true, message: '请输入执行命令' }]}
          >
            <TextArea
              placeholder="输入要执行的命令或脚本"
              rows={4}
            />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="partition" label="分区">
                <Input placeholder="计算分区" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="nodes" label="节点数">
                <Input type="number" placeholder="1" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="cpus" label="CPU数">
                <Input type="number" placeholder="1" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="memory" label="内存">
                <Input placeholder="4G" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="time_limit" label="时间限制">
                <Input placeholder="01:00:00" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="working_dir" label="工作目录">
            <Input placeholder="工作目录路径" />
          </Form.Item>

          <Form.Item style={{ textAlign: 'right', marginBottom: 0 }}>
            <Space>
              <Button onClick={() => {
                setSubmitModalVisible(false);
                form.resetFields();
              }}>
                取消
              </Button>
              <Button type="primary" htmlType="submit">
                提交
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 作业详情模态框 */}
      <Modal
        title="作业详情"
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={null}
        width={800}
      >
        {selectedJob && (
          <Descriptions bordered column={2}>
            <Descriptions.Item label="作业ID">{selectedJob.id}</Descriptions.Item>
            <Descriptions.Item label="作业名称">{selectedJob.name}</Descriptions.Item>
            <Descriptions.Item label="集群">{selectedJob.cluster_id}</Descriptions.Item>
            <Descriptions.Item label="状态">{getStatusTag(selectedJob.status)}</Descriptions.Item>
            <Descriptions.Item label="分区">{selectedJob.partition || '-'}</Descriptions.Item>
            <Descriptions.Item label="节点数">{selectedJob.nodes}</Descriptions.Item>
            <Descriptions.Item label="CPU数">{selectedJob.cpus}</Descriptions.Item>
            <Descriptions.Item label="内存">{selectedJob.memory || '-'}</Descriptions.Item>
            <Descriptions.Item label="时间限制">{selectedJob.time_limit || '-'}</Descriptions.Item>
            <Descriptions.Item label="工作目录">{selectedJob.working_dir || '-'}</Descriptions.Item>
            <Descriptions.Item label="提交时间">
              {new Date(selectedJob.submit_time).toLocaleString()}
            </Descriptions.Item>
            {selectedJob.start_time && (
              <Descriptions.Item label="开始时间">
                {new Date(selectedJob.start_time).toLocaleString()}
              </Descriptions.Item>
            )}
            {selectedJob.end_time && (
              <Descriptions.Item label="结束时间">
                {new Date(selectedJob.end_time).toLocaleString()}
              </Descriptions.Item>
            )}
            {selectedJob.exit_code !== null && (
              <Descriptions.Item label="退出码">{selectedJob.exit_code}</Descriptions.Item>
            )}
            <Descriptions.Item label="标准输出">{selectedJob.std_out || '-'}</Descriptions.Item>
            <Descriptions.Item label="标准错误">{selectedJob.std_err || '-'}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>

      {/* 作业输出模态框 */}
      <Modal
        title="作业输出"
        open={outputModalVisible}
        onCancel={() => setOutputModalVisible(false)}
        footer={null}
        width={800}
      >
        {jobOutput && (
          <Tabs defaultActiveKey="stdout">
            <TabPane tab="标准输出" key="stdout">
              <pre style={{
                background: '#f5f5f5',
                padding: '10px',
                borderRadius: '4px',
                maxHeight: '400px',
                overflow: 'auto',
                whiteSpace: 'pre-wrap'
              }}>
                {jobOutput.stdout || '无输出'}
              </pre>
            </TabPane>
            <TabPane tab="标准错误" key="stderr">
              <pre style={{
                background: '#fff2f0',
                padding: '10px',
                borderRadius: '4px',
                maxHeight: '400px',
                overflow: 'auto',
                whiteSpace: 'pre-wrap'
              }}>
                {jobOutput.stderr || '无错误输出'}
              </pre>
            </TabPane>
          </Tabs>
        )}
      </Modal>
    </div>
  );
};

export default JobManagement;