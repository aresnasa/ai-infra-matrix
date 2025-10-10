import React, { useState, useEffect } from 'react';
import { 
  Drawer, 
  Tabs, 
  Descriptions, 
  Table, 
  Tag, 
  Space, 
  Button,
  Alert,
  Spin,
  message,
  Modal,
  Typography
} from 'antd';
import {
  ReloadOutlined,
  DeleteOutlined,
  EditOutlined,
  DownloadOutlined,
  ExclamationCircleOutlined
} from '@ant-design/icons';
import axios from 'axios';
import MonacoEditor from '@monaco-editor/react';
import yaml from 'js-yaml';

const { TabPane } = Tabs;
const { Text, Paragraph } = Typography;
const { confirm } = Modal;

/**
 * ResourceDetails - Kubernetes 资源详情查看器
 * 
 * 展示资源的元数据、规格、状态等信息
 */
const ResourceDetails = ({ 
  visible, 
  onClose, 
  clusterId, 
  resourceType,
  namespace,
  resourceName,
  onUpdate,
  onDelete 
}) => {
  const [loading, setLoading] = useState(false);
  const [resourceData, setResourceData] = useState(null);
  const [error, setError] = useState(null);
  const [editMode, setEditMode] = useState(false);
  const [yamlContent, setYamlContent] = useState('');

  // 加载资源详情
  useEffect(() => {
    if (!visible || !clusterId || !resourceType || !resourceName) {
      return;
    }

    fetchResourceDetails();
  }, [visible, clusterId, resourceType, namespace, resourceName]);

  const fetchResourceDetails = async () => {
    setLoading(true);
    setError(null);

    try {
      let url;
      if (namespace) {
        url = `/api/kubernetes/clusters/${clusterId}/namespaces/${namespace}/resources/${resourceType}/${resourceName}`;
      } else {
        url = `/api/kubernetes/clusters/${clusterId}/cluster-resources/${resourceType}/${resourceName}`;
      }

      const response = await axios.get(url);
      setResourceData(response.data);
      setYamlContent(yaml.dump(response.data, { indent: 2, lineWidth: -1 }));
    } catch (err) {
      console.error('加载资源详情失败:', err);
      setError(err.response?.data?.error || err.message);
    } finally {
      setLoading(false);
    }
  };

  // 处理编辑
  const handleEdit = () => {
    setEditMode(true);
  };

  // 处理保存
  const handleSave = async () => {
    try {
      const updatedData = yaml.load(yamlContent);
      
      let url;
      if (namespace) {
        url = `/api/kubernetes/clusters/${clusterId}/namespaces/${namespace}/resources/${resourceType}/${resourceName}`;
      } else {
        url = `/api/kubernetes/clusters/${clusterId}/cluster-resources/${resourceType}/${resourceName}`;
      }

      await axios.put(url, updatedData);
      message.success('资源更新成功');
      setEditMode(false);
      
      if (onUpdate) {
        onUpdate();
      }
      
      fetchResourceDetails();
    } catch (err) {
      console.error('更新资源失败:', err);
      message.error(err.response?.data?.error || '更新失败');
    }
  };

  // 处理删除
  const handleDelete = () => {
    confirm({
      title: '确认删除',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除资源 ${resourceName} 吗？此操作不可撤销。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          let url;
          if (namespace) {
            url = `/api/kubernetes/clusters/${clusterId}/namespaces/${namespace}/resources/${resourceType}/${resourceName}`;
          } else {
            url = `/api/kubernetes/clusters/${clusterId}/cluster-resources/${resourceType}/${resourceName}`;
          }

          await axios.delete(url);
          message.success('资源删除成功');
          onClose();
          
          if (onDelete) {
            onDelete();
          }
        } catch (err) {
          console.error('删除资源失败:', err);
          message.error(err.response?.data?.error || '删除失败');
        }
      },
    });
  };

  // 下载 YAML
  const handleDownload = () => {
    const blob = new Blob([yamlContent], { type: 'text/yaml' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${resourceName}.yaml`;
    a.click();
    window.URL.revokeObjectURL(url);
  };

  // 渲染元数据
  const renderMetadata = () => {
    if (!resourceData?.metadata) return null;

    const { metadata } = resourceData;

    return (
      <Descriptions bordered column={2} size="small">
        <Descriptions.Item label="名称">{metadata.name}</Descriptions.Item>
        <Descriptions.Item label="命名空间">{metadata.namespace || 'N/A'}</Descriptions.Item>
        <Descriptions.Item label="UID">{metadata.uid}</Descriptions.Item>
        <Descriptions.Item label="创建时间">{metadata.creationTimestamp}</Descriptions.Item>
        <Descriptions.Item label="资源版本">{metadata.resourceVersion}</Descriptions.Item>
        <Descriptions.Item label="代数">{metadata.generation || 'N/A'}</Descriptions.Item>
        
        {metadata.labels && Object.keys(metadata.labels).length > 0 && (
          <Descriptions.Item label="标签" span={2}>
            <Space wrap>
              {Object.entries(metadata.labels).map(([key, value]) => (
                <Tag key={key} color="blue">
                  {key}: {value}
                </Tag>
              ))}
            </Space>
          </Descriptions.Item>
        )}
        
        {metadata.annotations && Object.keys(metadata.annotations).length > 0 && (
          <Descriptions.Item label="注解" span={2}>
            <Space direction="vertical" style={{ width: '100%' }}>
              {Object.entries(metadata.annotations).map(([key, value]) => (
                <Text key={key} code style={{ fontSize: '12px' }}>
                  {key}: {value}
                </Text>
              ))}
            </Space>
          </Descriptions.Item>
        )}
      </Descriptions>
    );
  };

  // 渲染规格（Spec）
  const renderSpec = () => {
    if (!resourceData?.spec) return <Alert message="此资源没有 spec 字段" type="info" />;

    return (
      <MonacoEditor
        height="500px"
        language="yaml"
        value={yaml.dump(resourceData.spec, { indent: 2, lineWidth: -1 })}
        options={{
          readOnly: true,
          minimap: { enabled: false },
          lineNumbers: 'on',
          scrollBeyondLastLine: false,
        }}
        theme="vs-light"
      />
    );
  };

  // 渲染状态（Status）
  const renderStatus = () => {
    if (!resourceData?.status) return <Alert message="此资源没有 status 字段" type="info" />;

    return (
      <MonacoEditor
        height="500px"
        language="yaml"
        value={yaml.dump(resourceData.status, { indent: 2, lineWidth: -1 })}
        options={{
          readOnly: true,
          minimap: { enabled: false },
          lineNumbers: 'on',
          scrollBeyondLastLine: false,
        }}
        theme="vs-light"
      />
    );
  };

  // 渲染完整 YAML
  const renderYAML = () => {
    return (
      <MonacoEditor
        height="600px"
        language="yaml"
        value={yamlContent}
        onChange={(value) => setYamlContent(value)}
        options={{
          readOnly: !editMode,
          minimap: { enabled: true },
          lineNumbers: 'on',
          scrollBeyondLastLine: false,
        }}
        theme="vs-dark"
      />
    );
  };

  return (
    <Drawer
      title={
        <Space>
          <Text strong>{resourceName}</Text>
          <Tag color="green">{resourceType}</Tag>
          {namespace && <Tag color="orange">{namespace}</Tag>}
        </Space>
      }
      width="80%"
      onClose={onClose}
      visible={visible}
      extra={
        <Space>
          <Button 
            icon={<ReloadOutlined />} 
            onClick={fetchResourceDetails}
            loading={loading}
          >
            刷新
          </Button>
          <Button 
            icon={<DownloadOutlined />} 
            onClick={handleDownload}
          >
            下载 YAML
          </Button>
          {!editMode && (
            <Button 
              type="primary"
              icon={<EditOutlined />} 
              onClick={handleEdit}
            >
              编辑
            </Button>
          )}
          {editMode && (
            <>
              <Button onClick={() => setEditMode(false)}>取消</Button>
              <Button type="primary" onClick={handleSave}>保存</Button>
            </>
          )}
          <Button 
            danger
            icon={<DeleteOutlined />} 
            onClick={handleDelete}
          >
            删除
          </Button>
        </Space>
      }
    >
      {loading && (
        <div style={{ textAlign: 'center', padding: '100px' }}>
          <Spin size="large" tip="加载中..." />
        </div>
      )}

      {error && (
        <Alert
          message="加载失败"
          description={error}
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      {!loading && !error && resourceData && (
        <Tabs defaultActiveKey="metadata">
          <TabPane tab="元数据" key="metadata">
            {renderMetadata()}
          </TabPane>
          
          <TabPane tab="规格 (Spec)" key="spec">
            {renderSpec()}
          </TabPane>
          
          <TabPane tab="状态 (Status)" key="status">
            {renderStatus()}
          </TabPane>
          
          <TabPane tab="完整 YAML" key="yaml">
            {renderYAML()}
          </TabPane>
        </Tabs>
      )}
    </Drawer>
  );
};

export default ResourceDetails;
