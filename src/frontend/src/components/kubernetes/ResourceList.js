import React, { useState, useEffect } from 'react';
import { 
  Table, 
  Space, 
  Button, 
  Tag, 
  Input,
  Select,
  message,
  Tooltip,
  Modal
} from 'antd';
import {
  ReloadOutlined,
  EyeOutlined,
  DeleteOutlined,
  PlusOutlined,
  SearchOutlined
} from '@ant-design/icons';
import axios from 'axios';
import ResourceDetails from './ResourceDetails';

const { Search } = Input;
const { Option } = Select;

/**
 * ResourceList - Kubernetes 资源列表组件
 * 
 * 显示指定类型的所有资源实例
 */
const ResourceList = ({ 
  clusterId, 
  resourceInfo,
  namespace 
}) => {
  const [loading, setLoading] = useState(false);
  const [resources, setResources] = useState([]);
  const [filteredResources, setFilteredResources] = useState([]);
  const [searchText, setSearchText] = useState('');
  const [selectedNamespace, setSelectedNamespace] = useState(namespace || 'all');
  const [namespaces, setNamespaces] = useState([]);
  const [detailsVisible, setDetailsVisible] = useState(false);
  const [selectedResource, setSelectedResource] = useState(null);

  // 资源信息
  const resourceType = resourceInfo?.resource?.name || resourceInfo?.crd?.plural;
  const isNamespaced = resourceInfo?.resource?.namespaced || resourceInfo?.crd?.scope === 'Namespaced';
  const groupVersion = resourceInfo?.resource?.groupVersion;

  // 加载命名空间列表（如果是命名空间资源）
  useEffect(() => {
    if (!clusterId || !isNamespaced) return;

    const fetchNamespaces = async () => {
      try {
        const response = await axios.get(
          `/api/kubernetes/clusters/${clusterId}/namespaces`
        );
        const nsList = response.data.items.map(ns => ns.metadata.name);
        setNamespaces(nsList);
      } catch (err) {
        console.error('加载命名空间列表失败:', err);
      }
    };

    fetchNamespaces();
  }, [clusterId, isNamespaced]);

  // 加载资源列表
  useEffect(() => {
    if (!clusterId || !resourceType) return;

    fetchResources();
  }, [clusterId, resourceType, selectedNamespace]);

  const fetchResources = async () => {
    setLoading(true);

    try {
      let url;
      if (isNamespaced && selectedNamespace !== 'all') {
        url = `/api/kubernetes/clusters/${clusterId}/namespaces/${selectedNamespace}/resources/${resourceType}`;
      } else if (isNamespaced && selectedNamespace === 'all') {
        // 获取所有命名空间的资源需要特殊处理
        url = `/api/kubernetes/clusters/${clusterId}/namespaces/-/resources/${resourceType}`;
      } else {
        url = `/api/kubernetes/clusters/${clusterId}/cluster-resources/${resourceType}`;
      }

      const response = await axios.get(url);
      const items = response.data.items || [];
      setResources(items);
      setFilteredResources(items);
    } catch (err) {
      console.error('加载资源列表失败:', err);
      message.error(err.response?.data?.error || '加载失败');
    } finally {
      setLoading(false);
    }
  };

  // 搜索过滤
  useEffect(() => {
    if (!searchText) {
      setFilteredResources(resources);
      return;
    }

    const filtered = resources.filter(item => {
      const name = item.metadata?.name || '';
      const namespace = item.metadata?.namespace || '';
      const search = searchText.toLowerCase();
      
      return name.toLowerCase().includes(search) || 
             namespace.toLowerCase().includes(search);
    });

    setFilteredResources(filtered);
  }, [searchText, resources]);

  // 查看详情
  const handleViewDetails = (record) => {
    setSelectedResource({
      name: record.metadata.name,
      namespace: record.metadata.namespace,
    });
    setDetailsVisible(true);
  };

  // 删除资源
  const handleDelete = (record) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除资源 ${record.metadata.name} 吗？`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          let url;
          if (record.metadata.namespace) {
            url = `/api/kubernetes/clusters/${clusterId}/namespaces/${record.metadata.namespace}/resources/${resourceType}/${record.metadata.name}`;
          } else {
            url = `/api/kubernetes/clusters/${clusterId}/cluster-resources/${resourceType}/${record.metadata.name}`;
          }

          await axios.delete(url);
          message.success('删除成功');
          fetchResources();
        } catch (err) {
          console.error('删除资源失败:', err);
          message.error(err.response?.data?.error || '删除失败');
        }
      },
    });
  };

  // 表格列定义
  const columns = [
    {
      title: '名称',
      dataIndex: ['metadata', 'name'],
      key: 'name',
      fixed: 'left',
      width: 250,
      ellipsis: true,
    },
    ...(isNamespaced ? [{
      title: '命名空间',
      dataIndex: ['metadata', 'namespace'],
      key: 'namespace',
      width: 150,
      render: (ns) => <Tag color="orange">{ns}</Tag>,
    }] : []),
    {
      title: '创建时间',
      dataIndex: ['metadata', 'creationTimestamp'],
      key: 'creationTimestamp',
      width: 200,
      render: (time) => new Date(time).toLocaleString('zh-CN'),
    },
    {
      title: '标签',
      dataIndex: ['metadata', 'labels'],
      key: 'labels',
      width: 300,
      ellipsis: true,
      render: (labels) => {
        if (!labels || Object.keys(labels).length === 0) return '-';
        
        const labelTags = Object.entries(labels).slice(0, 3).map(([key, value]) => (
          <Tag key={key} color="blue" style={{ margin: '2px' }}>
            {key}: {value}
          </Tag>
        ));
        
        const remaining = Object.keys(labels).length - 3;
        if (remaining > 0) {
          labelTags.push(
            <Tooltip key="more" title={JSON.stringify(labels, null, 2)}>
              <Tag color="default">+{remaining} 更多</Tag>
            </Tooltip>
          );
        }
        
        return <Space wrap>{labelTags}</Space>;
      },
    },
    {
      title: '操作',
      key: 'actions',
      fixed: 'right',
      width: 150,
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleViewDetails(record)}
          >
            查看
          </Button>
          <Button
            type="link"
            size="small"
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '16px' }}>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        {/* 工具栏 */}
        <Space style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space>
            <Search
              placeholder="搜索资源名称..."
              allowClear
              onSearch={setSearchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 300 }}
              prefix={<SearchOutlined />}
            />
            
            {isNamespaced && (
              <Select
                value={selectedNamespace}
                onChange={setSelectedNamespace}
                style={{ width: 200 }}
                placeholder="选择命名空间"
              >
                <Option value="all">所有命名空间</Option>
                {namespaces.map(ns => (
                  <Option key={ns} value={ns}>{ns}</Option>
                ))}
              </Select>
            )}
          </Space>

          <Space>
            <Button
              icon={<ReloadOutlined />}
              onClick={fetchResources}
              loading={loading}
            >
              刷新
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
            >
              创建
            </Button>
          </Space>
        </Space>

        {/* 资源信息 */}
        <Space>
          <Tag color="green">
            资源类型: {resourceInfo?.resource?.kind || resourceInfo?.crd?.kind}
          </Tag>
          {groupVersion && (
            <Tag color="blue">API 组: {groupVersion}</Tag>
          )}
          {isNamespaced ? (
            <Tag color="orange">命名空间资源</Tag>
          ) : (
            <Tag color="purple">集群级资源</Tag>
          )}
          <Tag>总数: {filteredResources.length}</Tag>
        </Space>

        {/* 资源列表表格 */}
        <Table
          columns={columns}
          dataSource={filteredResources}
          loading={loading}
          rowKey={(record) => record.metadata.uid}
          scroll={{ x: 1200 }}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            defaultPageSize: 20,
            pageSizeOptions: ['10', '20', '50', '100'],
          }}
        />
      </Space>

      {/* 资源详情抽屉 */}
      {selectedResource && (
        <ResourceDetails
          visible={detailsVisible}
          onClose={() => setDetailsVisible(false)}
          clusterId={clusterId}
          resourceType={resourceType}
          namespace={selectedResource.namespace}
          resourceName={selectedResource.name}
          onUpdate={fetchResources}
          onDelete={() => {
            setDetailsVisible(false);
            fetchResources();
          }}
        />
      )}
    </div>
  );
};

export default ResourceList;
