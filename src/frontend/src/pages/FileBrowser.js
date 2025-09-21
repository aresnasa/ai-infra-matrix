import React, { useEffect, useState } from 'react';
import { Card, Space, Input, Button, Table, Breadcrumb, Upload, Select, message } from 'antd';
import { FolderOpenOutlined, FileOutlined, UploadOutlined, DownloadOutlined, ArrowUpOutlined } from '@ant-design/icons';
import { filesAPI } from '../services/api';
import { api as rawApi } from '../services/api';

// Minimal File Browser to integrate OpenSCOW-like file management
const FileBrowser = () => {
  const [cluster, setCluster] = useState('');
  const [clusters, setClusters] = useState([]);
  const [path, setPath] = useState('/home');
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState([]);

  const loadClusters = async () => {
    try {
      const res = await rawApi.get('/jobs/clusters');
      const list = res?.data?.data || [];
      setClusters(list);
      if (!cluster && list.length > 0) {
        setCluster(list[0].id || list[0].ID || '');
      }
    } catch (e) {
      // silent
    }
  };

  const load = async (c = cluster, p = path) => {
    if (!c) return message.warning('请选择集群');
    setLoading(true);
    try {
      const { data } = await filesAPI.list(c, p);
      setItems(data?.data || []);
    } catch (e) {
      message.error('加载目录失败: ' + (e?.response?.data?.message || e.message));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { loadClusters(); /* eslint-disable-next-line */ }, []);
  useEffect(() => { if (cluster) load(cluster, path); /* eslint-disable-next-line */ }, [cluster]);

  const into = (name, isDir) => {
    if (!isDir) return;
    const next = path.endsWith('/') ? path + name : path + '/' + name;
    setPath(next);
    setTimeout(() => load(cluster, next), 0);
  };

  const goUp = () => {
    if (path === '/' || path === '') return;
    const parts = path.split('/').filter(Boolean);
    parts.pop();
    const parent = '/' + parts.join('/');
    setPath(parent || '/');
    setTimeout(() => load(cluster, parent || '/'), 0);
  };

  const download = async (record) => {
    try {
      const res = await filesAPI.download(cluster, record.path);
      const blob = new Blob([res.data], { type: 'application/octet-stream' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = record.name;
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (e) {
      message.error('下载失败: ' + (e?.response?.data?.message || e.message));
    }
  };

  const uploadProps = {
    name: 'file',
    multiple: false,
    showUploadList: false,
    customRequest: async ({ file, onSuccess, onError }) => {
      try {
        await filesAPI.upload(cluster, path + (path.endsWith('/') ? '' : '/') + file.name, file);
        onSuccess && onSuccess();
        message.success('上传成功');
        load();
      } catch (e) {
        onError && onError(e);
        message.error('上传失败: ' + (e?.response?.data?.message || e.message));
      }
    },
  };

  const columns = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Space>
          {record.is_dir ? <FolderOpenOutlined /> : <FileOutlined />}
          <a onClick={() => into(record.name, record.is_dir)}>{text}</a>
        </Space>
      ),
    },
    {
      title: '大小',
      dataIndex: 'size',
      key: 'size',
      width: 120,
      render: (s, r) => (r.is_dir ? '-' : s),
    },
    {
      title: '修改时间',
      dataIndex: 'mod_time',
      key: 'mod_time',
      width: 200,
      render: (t) => (t ? new Date(t).toLocaleString() : ''),
    },
    {
      title: '操作',
      key: 'action',
      width: 140,
      render: (_, record) => (
        <Space>
          {!record.is_dir && (
            <Button size="small" icon={<DownloadOutlined />} onClick={() => download(record)}>
              下载
            </Button>
          )}
        </Space>
      ),
    },
  ];

  const clusterOptions = clusters.map(c => ({ value: c.id || c.ID, label: c.name || c.Name || (c.id || '') }));

  const breadcrumbs = path.split('/').filter(Boolean);

  return (
    <Card title="文件浏览器 (SSH)" extra={
      <Space>
        <Select
          value={cluster}
          placeholder="选择集群"
          style={{ width: 200 }}
          options={clusterOptions}
          onChange={(v) => setCluster(v)}
        />
        <Input value={path} onChange={(e) => setPath(e.target.value)} style={{ width: 360 }} />
        <Button icon={<ArrowUpOutlined />} onClick={goUp}>上一级</Button>
        <Button onClick={() => load()}>刷新</Button>
        <Upload {...uploadProps}>
          <Button type="primary" icon={<UploadOutlined />}>上传</Button>
        </Upload>
      </Space>
    }>
      <Breadcrumb>
        <Breadcrumb.Item onClick={() => { setPath('/'); load(cluster, '/'); }} style={{ cursor: 'pointer' }}>/</Breadcrumb.Item>
        {breadcrumbs.map((seg, idx) => (
          <Breadcrumb.Item key={idx}>{seg}</Breadcrumb.Item>
        ))}
      </Breadcrumb>
      <Table
        rowKey={(r) => r.path}
        loading={loading}
        dataSource={items}
        columns={columns}
        size="small"
        pagination={false}
        style={{ marginTop: 12 }}
      />
    </Card>
  );
};

export default FileBrowser;
