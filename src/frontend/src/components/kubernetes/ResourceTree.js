import React, { useState, useEffect, useMemo } from 'react';
import { Tree, Input, Spin, Alert, Tag, Space, Typography, Divider } from 'antd';
import { 
  FolderOutlined, 
  ApiOutlined, 
  FileTextOutlined,
  SearchOutlined,
  ClusterOutlined,
  CloudServerOutlined,
  DatabaseOutlined
} from '@ant-design/icons';
import axios from 'axios';

const { Search } = Input;
const { Text } = Typography;

/**
 * ResourceTree - Kubernetes 资源树组件
 * 
 * 支持按 API 组/版本/资源类型层次展示所有 k8s 资源，包括 CRD
 */
const ResourceTree = ({ clusterId, onResourceSelect }) => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [discoveryData, setDiscoveryData] = useState(null);
  const [expandedKeys, setExpandedKeys] = useState([]);
  const [searchValue, setSearchValue] = useState('');
  const [autoExpandParent, setAutoExpandParent] = useState(true);

  // 加载增强的资源发现数据
  useEffect(() => {
    if (!clusterId) return;

    const fetchDiscovery = async () => {
      setLoading(true);
      setError(null);
      
      try {
        const response = await axios.get(
          `/api/kubernetes/clusters/${clusterId}/enhanced-discovery`
        );
        setDiscoveryData(response.data);
      } catch (err) {
        console.error('加载资源发现数据失败:', err);
        setError(err.response?.data?.error || err.message);
      } finally {
        setLoading(false);
      }
    };

    fetchDiscovery();
  }, [clusterId]);

  // 构建树节点数据
  const treeData = useMemo(() => {
    if (!discoveryData) return [];

    const nodes = [];

    // 1. 添加集群版本信息节点
    if (discoveryData.version) {
      nodes.push({
        title: (
          <Space>
            <ClusterOutlined />
            <Text strong>集群版本</Text>
            <Tag color="blue">{discoveryData.version.gitVersion}</Tag>
          </Space>
        ),
        key: 'cluster-version',
        selectable: false,
        icon: <ClusterOutlined />,
      });
    }

    // 2. 添加内置资源组节点
    if (discoveryData.resourcesByGroup) {
      const builtInNode = {
        title: (
          <Space>
            <ApiOutlined />
            <Text strong>内置资源</Text>
            <Tag>{Object.keys(discoveryData.resourcesByGroup).length} 组</Tag>
          </Space>
        ),
        key: 'builtin-resources',
        icon: <ApiOutlined />,
        selectable: false,
        children: [],
      };

      // 按 GroupVersion 分组
      Object.entries(discoveryData.resourcesByGroup).forEach(([gv, resources]) => {
        const groupNode = {
          title: (
            <Space>
              <FolderOutlined />
              <Text>{gv || 'core'}</Text>
              <Tag color="green">{resources.length} 资源</Tag>
            </Space>
          ),
          key: `group-${gv}`,
          icon: <FolderOutlined />,
          selectable: false,
          children: resources.map(resource => ({
            title: (
              <Space>
                <FileTextOutlined />
                <Text>{resource.name}</Text>
                {resource.namespaced && <Tag color="orange">命名空间</Tag>}
                {!resource.namespaced && <Tag color="purple">集群级</Tag>}
                {resource.kind && <Tag>{resource.kind}</Tag>}
              </Space>
            ),
            key: `resource-${gv}-${resource.name}`,
            icon: <FileTextOutlined />,
            isLeaf: true,
            resource: {
              ...resource,
              groupVersion: gv,
            },
          })),
        };

        builtInNode.children.push(groupNode);
      });

      nodes.push(builtInNode);
    }

    // 3. 添加 CRD 节点
    if (discoveryData.crds && discoveryData.crds.length > 0) {
      const crdNode = {
        title: (
          <Space>
            <DatabaseOutlined />
            <Text strong>自定义资源 (CRD)</Text>
            <Tag color="red">{discoveryData.crds.length} 个</Tag>
          </Space>
        ),
        key: 'crds',
        icon: <DatabaseOutlined />,
        selectable: false,
        children: [],
      };

      // 按 Group 分组 CRD
      const crdsByGroup = {};
      discoveryData.crds.forEach(crd => {
        const group = crd.group || 'custom';
        if (!crdsByGroup[group]) {
          crdsByGroup[group] = [];
        }
        crdsByGroup[group].push(crd);
      });

      Object.entries(crdsByGroup).forEach(([group, crds]) => {
        const groupNode = {
          title: (
            <Space>
              <FolderOutlined />
              <Text>{group}</Text>
              <Tag color="red">{crds.length} CRD</Tag>
            </Space>
          ),
          key: `crd-group-${group}`,
          icon: <FolderOutlined />,
          selectable: false,
          children: crds.map(crd => ({
            title: (
              <Space>
                <CloudServerOutlined />
                <Text>{crd.kind}</Text>
                <Tag color="cyan">{crd.plural}</Tag>
                {crd.scope === 'Namespaced' && <Tag color="orange">命名空间</Tag>}
                {crd.scope === 'Cluster' && <Tag color="purple">集群级</Tag>}
                {crd.versions && crd.versions.length > 0 && (
                  <Tag>{crd.versions.join(', ')}</Tag>
                )}
              </Space>
            ),
            key: `crd-${crd.name}`,
            icon: <CloudServerOutlined />,
            isLeaf: true,
            crd: crd,
          })),
        };

        crdNode.children.push(groupNode);
      });

      nodes.push(crdNode);
    }

    return nodes;
  }, [discoveryData]);

  // 搜索过滤
  const filteredTreeData = useMemo(() => {
    if (!searchValue) return treeData;

    const filterTree = (nodes) => {
      return nodes
        .map(node => {
          const titleText = typeof node.title === 'string' 
            ? node.title 
            : node.title?.props?.children?.find(c => typeof c === 'string' || c?.props?.children)?.props?.children || '';
          
          const matches = titleText.toLowerCase().includes(searchValue.toLowerCase());
          
          if (node.children) {
            const filteredChildren = filterTree(node.children);
            if (filteredChildren.length > 0 || matches) {
              return { ...node, children: filteredChildren };
            }
          } else if (matches) {
            return node;
          }
          
          return null;
        })
        .filter(Boolean);
    };

    return filterTree(treeData);
  }, [treeData, searchValue]);

  // 处理节点选择
  const handleSelect = (selectedKeys, info) => {
    if (!info.node.isLeaf) return;

    const nodeData = {
      resource: info.node.resource,
      crd: info.node.crd,
    };

    if (onResourceSelect) {
      onResourceSelect(nodeData);
    }
  };

  // 处理搜索
  const handleSearch = (value) => {
    setSearchValue(value);
    if (value) {
      // 展开所有匹配的父节点
      const getAllKeys = (nodes) => {
        let keys = [];
        nodes.forEach(node => {
          keys.push(node.key);
          if (node.children) {
            keys = keys.concat(getAllKeys(node.children));
          }
        });
        return keys;
      };
      setExpandedKeys(getAllKeys(filteredTreeData));
      setAutoExpandParent(true);
    } else {
      setExpandedKeys([]);
      setAutoExpandParent(false);
    }
  };

  const handleExpand = (keys) => {
    setExpandedKeys(keys);
    setAutoExpandParent(false);
  };

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" tip="加载资源树..." />
      </div>
    );
  }

  if (error) {
    return (
      <Alert
        message="加载失败"
        description={error}
        type="error"
        showIcon
      />
    );
  }

  return (
    <div style={{ padding: '16px' }}>
      <Search
        placeholder="搜索资源..."
        onChange={(e) => handleSearch(e.target.value)}
        onSearch={handleSearch}
        style={{ marginBottom: 16 }}
        prefix={<SearchOutlined />}
        allowClear
      />

      {discoveryData && (
        <>
          <div style={{ marginBottom: 16 }}>
            <Space split={<Divider type="vertical" />}>
              <Text>
                <strong>总资源:</strong> {discoveryData.totalResources || 0}
              </Text>
              <Text>
                <strong>CRD:</strong> {discoveryData.totalCRDs || 0}
              </Text>
              <Text>
                <strong>API组:</strong> {discoveryData.groups?.groups?.length || 0}
              </Text>
            </Space>
          </div>

          <Tree
            showIcon
            treeData={filteredTreeData}
            expandedKeys={expandedKeys}
            autoExpandParent={autoExpandParent}
            onExpand={handleExpand}
            onSelect={handleSelect}
            style={{ 
              background: '#fafafa', 
              padding: '8px',
              borderRadius: '4px',
              maxHeight: 'calc(100vh - 300px)',
              overflow: 'auto',
            }}
          />
        </>
      )}
    </div>
  );
};

export default ResourceTree;
