import React, { useState, useEffect } from 'react';
import {
  Card, Table, Button, Space, Tag, Modal, Form, Input, 
  message, Tooltip, Popconfirm, Badge, Typography, Descriptions, Spin
} from 'antd';
import {
  PlusOutlined, ReloadOutlined, DeleteOutlined, EyeOutlined,
  PlayCircleOutlined, LinkOutlined, CheckCircleOutlined,
  ExclamationCircleOutlined, ClusterOutlined
} from '@ant-design/icons';
import { api } from '../../services/api';

const { Title, Text } = Typography;
const { TextArea } = Input;

const ExternalClusterManagement = () => {
  const [clusters, setClusters] = useState([]);
  const [loading, setLoading] = useState(false);
  const [connectModalVisible, setConnectModalVisible] = useState(false);
  const [detailsModalVisible, setDetailsModalVisible] = useState(false);
  const [selectedCluster, setSelectedCluster] = useState(null);
  const [form] = Form.useForm();

  // 加载集群列表
  const loadClusters = async () => {
    setLoading(true);
    try {
      const response = await api.get('/slurm/clusters');
      setClusters(response.data.clusters || []);
    } catch (error) {
      message.error('加载集群列表失败: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadClusters();
  }, []);

  // 连接外部集群
  const handleConnect = async (values) => {
    try {
      await api.post('/slurm/clusters/connect', values);
      message.success('集群连接成功');
      setConnectModalVisible(false);
      form.resetFields();
      loadClusters();
    } catch (error) {
      message.error('连接失败: ' + (error.response?.data?.error || error.message));
    }
  };

  // 删除集群
  const handleDelete = async (clusterId) => {
    try {
      await api.delete(`/slurm/clusters/${clusterId}`);
      message.success('集群已删除');
      loadClusters();
    } catch (error) {
      message.error('删除失败: ' + (error.response?.data?.error || error.message));
    }
  };

  // 查看集群详情
  const handleViewDetails = async (cluster) => {
    setLoading(true);
    try {
      const response = await api.get(`/slurm/clusters/${cluster.id}/info`);
      setSelectedCluster({ ...cluster, details: response.data });
      setDetailsModalVisible(true);
    } catch (error) {
      message.error('获取集群详情失败: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  // 表格列定义
  const columns = [
    {
      title: '集群名称',
      dataIndex: 'name',
      key: 'name',
      render: (text) => <Text strong><ClusterOutlined /> {text}</Text>
    },
    {
      title: '主机地址',
      dataIndex: 'master_host',
      key: 'master_host'
    },
    {
      title: '连接状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => {
        const statusMap = {
          'connected': { color: 'success', icon: <CheckCircleOutlined />, text: '已连接' },
          'disconnected': { color: 'error', icon: <ExclamationCircleOutlined />, text: '断开连接' },
          'connecting': { color: 'processing', icon: <LinkOutlined />, text: '连接中' }
        };
        const { color, icon, text } = statusMap[status] || statusMap['disconnected'];
        return <Badge status={color} text={<>{icon} {text}</>} />;
      }
    },
    {
      title: '节点数',
      dataIndex: 'node_count',
      key: 'node_count',
      render: (count) => <Tag color="blue">{count || 0} 个节点</Tag>
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              size="small"
              icon={<EyeOutlined />}
              onClick={() => handleViewDetails(record)}
            />
          </Tooltip>
          <Tooltip title="刷新状态">
            <Button
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => loadClusters()}
            />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个集群吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button
                size="small"
                danger
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      )
    }
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Card
        title={<Title level={4}><ClusterOutlined /> 外部 SLURM 集群管理</Title>}
        extra={
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={loadClusters}
              loading={loading}
            >
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setConnectModalVisible(true)}
            >
              连接外部集群
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={clusters}
          rowKey="id"
          loading={loading}
          pagination={{ pageSize: 10 }}
        />
      </Card>

      {/* 连接集群模态框 */}
      <Modal
        title="连接外部 SLURM 集群"
        open={connectModalVisible}
        onCancel={() => {
          setConnectModalVisible(false);
          form.resetFields();
        }}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleConnect}
        >
          <Form.Item
            label="集群名称"
            name="name"
            rules={[{ required: true, message: '请输入集群名称' }]}
          >
            <Input placeholder="例如: production-cluster" />
          </Form.Item>

          <Form.Item
            label="主节点地址"
            name="master_host"
            rules={[{ required: true, message: '请输入主节点地址' }]}
          >
            <Input placeholder="例如: 192.168.1.100" />
          </Form.Item>

          <Form.Item
            label="SSH 端口"
            name="ssh_port"
            initialValue={22}
            rules={[{ required: true, message: '请输入 SSH 端口' }]}
          >
            <Input type="number" placeholder="22" />
          </Form.Item>

          <Form.Item
            label="SSH 用户名"
            name="ssh_user"
            rules={[{ required: true, message: '请输入 SSH 用户名' }]}
          >
            <Input placeholder="例如: root" />
          </Form.Item>

          <Form.Item
            label="SSH 密码"
            name="ssh_password"
            rules={[{ required: true, message: '请输入 SSH 密码' }]}
          >
            <Input.Password placeholder="请输入密码" />
          </Form.Item>

          <Form.Item
            label="描述"
            name="description"
          >
            <TextArea rows={3} placeholder="集群描述信息（可选）" />
          </Form.Item>

          <Form.Item>
            <Space style={{ float: 'right' }}>
              <Button onClick={() => {
                setConnectModalVisible(false);
                form.resetFields();
              }}>
                取消
              </Button>
              <Button type="primary" htmlType="submit">
                连接
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 集群详情模态框 */}
      <Modal
        title={`集群详情: ${selectedCluster?.name || ''}`}
        open={detailsModalVisible}
        onCancel={() => {
          setDetailsModalVisible(false);
          setSelectedCluster(null);
        }}
        footer={[
          <Button key="close" onClick={() => setDetailsModalVisible(false)}>
            关闭
          </Button>
        ]}
        width={800}
      >
        {selectedCluster && (
          <Descriptions bordered column={2}>
            <Descriptions.Item label="集群名称" span={2}>
              {selectedCluster.name}
            </Descriptions.Item>
            <Descriptions.Item label="主节点地址">
              {selectedCluster.master_host}
            </Descriptions.Item>
            <Descriptions.Item label="SSH 端口">
              {selectedCluster.ssh_port || 22}
            </Descriptions.Item>
            <Descriptions.Item label="连接状态" span={2}>
              <Badge
                status={selectedCluster.status === 'connected' ? 'success' : 'error'}
                text={selectedCluster.status === 'connected' ? '已连接' : '断开连接'}
              />
            </Descriptions.Item>
            <Descriptions.Item label="节点数量">
              {selectedCluster.node_count || 0}
            </Descriptions.Item>
            <Descriptions.Item label="SLURM 版本">
              {selectedCluster.details?.version || 'N/A'}
            </Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>
              {selectedCluster.description || '无'}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default ExternalClusterManagement;
