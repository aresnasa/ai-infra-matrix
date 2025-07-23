import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Modal,
  Form,
  Input,
  InputNumber,
  message,
  Popconfirm,
  Space
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { hostAPI } from '../services/api';

const HostsTab = ({ projectId }) => {
  const [hosts, setHosts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingHost, setEditingHost] = useState(null);
  const [form] = Form.useForm();

  // 获取主机列表
  const fetchHosts = async () => {
    if (!projectId) return;
    
    setLoading(true);
    try {
      const response = await hostAPI.getHosts(projectId);
      setHosts(response.data || []);
    } catch (error) {
      message.error('获取主机列表失败');
      console.error('Error fetching hosts:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHosts();
  }, [projectId]);

  // 打开创建/编辑模态框
  const openModal = (host = null) => {
    setEditingHost(host);
    if (host) {
      form.setFieldsValue(host);
    } else {
      form.resetFields();
      form.setFieldsValue({ project_id: projectId, port: 22 });
    }
    setModalVisible(true);
  };

  // 保存主机
  const handleSave = async (values) => {
    try {
      values.project_id = projectId;
      
      if (editingHost) {
        await hostAPI.updateHost(editingHost.id, values);
        message.success('主机更新成功');
      } else {
        await hostAPI.createHost(values);
        message.success('主机创建成功');
      }
      setModalVisible(false);
      fetchHosts();
    } catch (error) {
      message.error(editingHost ? '主机更新失败' : '主机创建失败');
      console.error('Error saving host:', error);
    }
  };

  // 删除主机
  const handleDelete = async (id) => {
    try {
      await hostAPI.deleteHost(id);
      message.success('主机删除成功');
      fetchHosts();
    } catch (error) {
      message.error('主机删除失败');
      console.error('Error deleting host:', error);
    }
  };

  const columns = [
    {
      title: '主机名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
    },
    {
      title: '端口',
      dataIndex: 'port',
      key: 'port',
      width: 80,
    },
    {
      title: '用户名',
      dataIndex: 'user',
      key: 'user',
    },
    {
      title: '主机组',
      dataIndex: 'group',
      key: 'group',
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => openModal(record)}
            size="small"
          />
          <Popconfirm
            title="确定要删除这个主机吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="删除"
            cancelText="取消"
          >
            <Button
              type="link"
              danger
              icon={<DeleteOutlined />}
              size="small"
            />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div className="action-buttons">
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => openModal()}
        >
          添加主机
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={hosts}
        rowKey="id"
        loading={loading}
        pagination={false}
        locale={{ emptyText: '暂无主机配置' }}
      />

      <Modal
        title={editingHost ? '编辑主机' : '添加主机'}
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
            label="主机名称"
            rules={[
              { required: true, message: '请输入主机名称' },
              { max: 255, message: '主机名称不能超过255个字符' }
            ]}
          >
            <Input placeholder="例如: web-server-01" />
          </Form.Item>
          
          <Form.Item
            name="ip"
            label="IP地址"
            rules={[
              { required: true, message: '请输入IP地址' },
              { 
                pattern: /^(\d{1,3}\.){3}\d{1,3}$/, 
                message: '请输入正确的IP地址格式' 
              }
            ]}
          >
            <Input placeholder="例如: 192.168.1.100" />
          </Form.Item>
          
          <Form.Item
            name="port"
            label="SSH端口"
            rules={[
              { required: true, message: '请输入SSH端口' },
              { 
                type: 'number', 
                min: 1, 
                max: 65535, 
                message: '端口范围: 1-65535' 
              }
            ]}
          >
            <InputNumber 
              placeholder="22" 
              style={{ width: '100%' }}
              min={1}
              max={65535}
            />
          </Form.Item>
          
          <Form.Item
            name="user"
            label="登录用户"
            rules={[
              { required: true, message: '请输入登录用户名' },
              { max: 100, message: '用户名不能超过100个字符' }
            ]}
          >
            <Input placeholder="例如: root, ubuntu" />
          </Form.Item>
          
          <Form.Item
            name="group"
            label="主机组"
            rules={[
              { max: 100, message: '主机组名不能超过100个字符' }
            ]}
          >
            <Input placeholder="例如: webservers, databases (可选)" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default HostsTab;
