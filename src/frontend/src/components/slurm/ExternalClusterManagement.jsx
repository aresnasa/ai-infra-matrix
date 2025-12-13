import React, { useState, useEffect } from 'react';
import {
  Card, Table, Button, Space, Tag, Modal, Form, Input, 
  message, Tooltip, Popconfirm, Badge, Typography, Descriptions, Spin, Radio, Upload
} from 'antd';
import {
  PlusOutlined, ReloadOutlined, DeleteOutlined, EyeOutlined,
  PlayCircleOutlined, LinkOutlined, CheckCircleOutlined,
  ExclamationCircleOutlined, ClusterOutlined, UploadOutlined
} from '@ant-design/icons';
import { api } from '../../services/api';
import { useI18n } from '../../hooks/useI18n';

const { Title, Text } = Typography;
const { TextArea } = Input;

const ExternalClusterManagement = () => {
  const { t } = useI18n();
  const [clusters, setClusters] = useState([]);
  const [loading, setLoading] = useState(false);
  const [connectModalVisible, setConnectModalVisible] = useState(false);
  const [detailsModalVisible, setDetailsModalVisible] = useState(false);
  const [selectedCluster, setSelectedCluster] = useState(null);
  const [authType, setAuthType] = useState('password'); // 'password' 或 'key'
  const [sshKeyFile, setSshKeyFile] = useState(null);
  const [form] = Form.useForm();

  // 加载集群列表
  const loadClusters = async () => {
    setLoading(true);
    try {
      const response = await api.get('/slurm/clusters');
      setClusters(response.data.clusters || []);
    } catch (error) {
      message.error(t('externalCluster.messages.loadFailed') + ': ' + (error.response?.data?.error || error.message));
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
      const formData = new FormData();
      
      // 基本信息
      formData.append('name', values.name);
      formData.append('master_host', values.master_host);
      formData.append('ssh_port', values.ssh_port || 22);
      formData.append('ssh_user', values.ssh_user);
      if (values.description) {
        formData.append('description', values.description);
      }
      
      // 认证信息
      formData.append('auth_type', authType);
      if (authType === 'password') {
        formData.append('ssh_password', values.ssh_password);
      } else if (authType === 'key' && sshKeyFile) {
        formData.append('ssh_key', sshKeyFile);
      }
      
      await api.post('/slurm/clusters/connect', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });
      
      message.success(t('externalCluster.messages.connectSuccess'));
      setConnectModalVisible(false);
      form.resetFields();
      setAuthType('password');
      setSshKeyFile(null);
      loadClusters();
    } catch (error) {
      message.error(t('externalCluster.messages.connectFailed') + ': ' + (error.response?.data?.error || error.message));
    }
  };

  // 删除集群
  const handleDelete = async (clusterId) => {
    try {
      await api.delete(`/slurm/clusters/${clusterId}`);
      message.success(t('externalCluster.messages.deleteSuccess'));
      loadClusters();
    } catch (error) {
      message.error(t('externalCluster.messages.deleteFailed') + ': ' + (error.response?.data?.error || error.message));
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
      message.error(t('externalCluster.messages.detailFailed') + ': ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  // 表格列定义
  const columns = [
    {
      title: t('externalCluster.columns.clusterName'),
      dataIndex: 'name',
      key: 'name',
      render: (text) => <Text strong><ClusterOutlined /> {text}</Text>
    },
    {
      title: t('externalCluster.columns.hostAddress'),
      dataIndex: 'master_host',
      key: 'master_host'
    },
    {
      title: t('externalCluster.columns.connectionStatus'),
      dataIndex: 'status',
      key: 'status',
      render: (status) => {
        const statusMap = {
          'connected': { color: 'success', icon: <CheckCircleOutlined />, text: t('externalCluster.status.connected') },
          'disconnected': { color: 'error', icon: <ExclamationCircleOutlined />, text: t('externalCluster.status.disconnected') },
          'connecting': { color: 'processing', icon: <LinkOutlined />, text: t('externalCluster.status.connecting') }
        };
        const { color, icon, text: statusText } = statusMap[status] || statusMap['disconnected'];
        return <Badge status={color} text={<>{icon} {statusText}</>} />;
      }
    },
    {
      title: t('externalCluster.columns.nodeCount'),
      dataIndex: 'node_count',
      key: 'node_count',
      render: (count) => <Tag color="blue">{count || 0} {t('externalCluster.nodes')}</Tag>
    },
    {
      title: t('externalCluster.columns.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true
    },
    {
      title: t('externalCluster.columns.actions'),
      key: 'action',
      render: (_, record) => (
        <Space>
          <Tooltip title={t('externalCluster.actions.viewDetail')}>
            <Button
              size="small"
              icon={<EyeOutlined />}
              onClick={() => handleViewDetails(record)}
            />
          </Tooltip>
          <Tooltip title={t('externalCluster.actions.refreshStatus')}>
            <Button
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => loadClusters()}
            />
          </Tooltip>
          <Popconfirm
            title={t('externalCluster.messages.confirmDelete')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('externalCluster.confirm')}
            cancelText={t('externalCluster.cancel')}
          >
            <Tooltip title={t('externalCluster.actions.delete')}>
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
        title={<Title level={4}><ClusterOutlined /> {t('externalCluster.title')}</Title>}
        extra={
          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={loadClusters}
              loading={loading}
            >
              {t('externalCluster.actions.refresh')}
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setConnectModalVisible(true)}
            >
              {t('externalCluster.actions.connectCluster')}
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
        title={t('externalCluster.connectModal.title')}
        open={connectModalVisible}
        onCancel={() => {
          setConnectModalVisible(false);
          form.resetFields();
          setAuthType('password');
          setSshKeyFile(null);
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
            label={t('externalCluster.connectModal.clusterName')}
            name="name"
            rules={[{ required: true, message: t('externalCluster.connectModal.clusterNameRequired') }]}
          >
            <Input placeholder={t('externalCluster.connectModal.clusterNamePlaceholder')} />
          </Form.Item>

          <Form.Item
            label={t('externalCluster.connectModal.masterHost')}
            name="master_host"
            rules={[{ required: true, message: t('externalCluster.connectModal.masterHostRequired') }]}
          >
            <Input placeholder={t('externalCluster.connectModal.masterHostPlaceholder')} />
          </Form.Item>

          <Form.Item
            label={t('externalCluster.connectModal.sshPort')}
            name="ssh_port"
            initialValue={22}
            rules={[{ required: true, message: t('externalCluster.connectModal.sshPortRequired') }]}
          >
            <Input type="number" placeholder="22" />
          </Form.Item>

          <Form.Item
            label={t('externalCluster.connectModal.sshUser')}
            name="ssh_user"
            rules={[{ required: true, message: t('externalCluster.connectModal.sshUserRequired') }]}
          >
            <Input placeholder={t('externalCluster.connectModal.sshUserPlaceholder')} />
          </Form.Item>

          <Form.Item label={t('externalCluster.connectModal.authType')}>
            <Radio.Group 
              value={authType} 
              onChange={(e) => setAuthType(e.target.value)}
            >
              <Radio value="password">{t('externalCluster.connectModal.passwordAuth')}</Radio>
              <Radio value="key">{t('externalCluster.connectModal.keyAuth')}</Radio>
            </Radio.Group>
          </Form.Item>

          {authType === 'password' ? (
            <Form.Item
              label={t('externalCluster.connectModal.sshPassword')}
              name="ssh_password"
              rules={[{ required: true, message: t('externalCluster.connectModal.sshPasswordRequired') }]}
            >
              <Input.Password placeholder={t('externalCluster.connectModal.sshPasswordPlaceholder')} />
            </Form.Item>
          ) : (
            <Form.Item
              label={t('externalCluster.connectModal.sshKey')}
              rules={[{ required: !sshKeyFile, message: t('externalCluster.connectModal.sshKeyRequired') }]}
            >
              <Upload
                beforeUpload={(file) => {
                  setSshKeyFile(file);
                  message.success(`${t('externalCluster.messages.fileSelected')}: ${file.name}`);
                  return false; // 阻止自动上传
                }}
                onRemove={() => {
                  setSshKeyFile(null);
                }}
                maxCount={1}
                accept=".pem,.key,id_rsa,id_ed25519"
              >
                <Button icon={<UploadOutlined />}>
                  {sshKeyFile ? `${t('externalCluster.connectModal.keyFileSelected')}: ${sshKeyFile.name}` : t('externalCluster.connectModal.selectKeyFile')}
                </Button>
              </Upload>
              <div style={{ marginTop: 8, fontSize: 12, color: '#999' }}>
                {t('externalCluster.connectModal.supportedFormats')}
              </div>
            </Form.Item>
          )}

          <Form.Item
            label={t('externalCluster.connectModal.description')}
            name="description"
          >
            <TextArea rows={3} placeholder={t('externalCluster.connectModal.descriptionPlaceholder')} />
          </Form.Item>

          <Form.Item>
            <Space style={{ float: 'right' }}>
              <Button onClick={() => {
                setConnectModalVisible(false);
                form.resetFields();
                setAuthType('password');
                setSshKeyFile(null);
              }}>
                {t('externalCluster.cancel')}
              </Button>
              <Button type="primary" htmlType="submit">
                {t('externalCluster.actions.connect')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 集群详情模态框 */}
      <Modal
        title={`${t('externalCluster.detailModal.title')}: ${selectedCluster?.name || ''}`}
        open={detailsModalVisible}
        onCancel={() => {
          setDetailsModalVisible(false);
          setSelectedCluster(null);
        }}
        footer={[
          <Button key="close" onClick={() => setDetailsModalVisible(false)}>
            {t('externalCluster.close')}
          </Button>
        ]}
        width={800}
      >
        {selectedCluster && (
          <Descriptions bordered column={2}>
            <Descriptions.Item label={t('externalCluster.detailModal.clusterName')} span={2}>
              {selectedCluster.name}
            </Descriptions.Item>
            <Descriptions.Item label={t('externalCluster.detailModal.masterHost')}>
              {selectedCluster.master_host}
            </Descriptions.Item>
            <Descriptions.Item label={t('externalCluster.detailModal.sshPort')}>
              {selectedCluster.ssh_port || 22}
            </Descriptions.Item>
            <Descriptions.Item label={t('externalCluster.detailModal.connectionStatus')} span={2}>
              <Badge
                status={selectedCluster.status === 'connected' ? 'success' : 'error'}
                text={selectedCluster.status === 'connected' ? t('externalCluster.status.connected') : t('externalCluster.status.disconnected')}
              />
            </Descriptions.Item>
            <Descriptions.Item label={t('externalCluster.detailModal.nodeCount')}>
              {selectedCluster.node_count || 0}
            </Descriptions.Item>
            <Descriptions.Item label={t('externalCluster.detailModal.slurmVersion')}>
              {selectedCluster.details?.version || 'N/A'}
            </Descriptions.Item>
            <Descriptions.Item label={t('externalCluster.detailModal.description')} span={2}>
              {selectedCluster.description || t('externalCluster.detailModal.noDescription')}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default ExternalClusterManagement;
