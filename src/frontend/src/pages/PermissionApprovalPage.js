import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Select,
  Input,
  Tag,
  Space,
  Tabs,
  Checkbox,
  InputNumber,
  message,
  Badge,
  Tooltip,
  Typography,
  Row,
  Col,
  Statistic,
  Timeline,
  Divider,
  Alert,
  Popconfirm,
  Switch,
} from 'antd';
import {
  PlusOutlined,
  CheckOutlined,
  CloseOutlined,
  EyeOutlined,
  DeleteOutlined,
  HistoryOutlined,
  SafetyOutlined,
  UserOutlined,
  TeamOutlined,
  AppstoreOutlined,
  SettingOutlined,
  FileTextOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import { approvalAPI } from '../services/api';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;
const { Option } = Select;
const { TabPane } = Tabs;

// 状态颜色映射
const statusColors = {
  pending: 'gold',
  approved: 'green',
  rejected: 'red',
  canceled: 'gray',
  expired: 'purple',
};

// 状态中文映射
const statusLabels = {
  pending: '待审批',
  approved: '已批准',
  rejected: '已拒绝',
  canceled: '已取消',
  expired: '已过期',
};

// 优先级颜色映射
const priorityColors = {
  1: 'blue',
  2: 'cyan',
  3: 'orange',
  4: 'red',
};

const priorityLabels = {
  1: '低',
  2: '普通',
  3: '高',
  4: '紧急',
};

const PermissionApprovalPage = () => {
  const [activeTab, setActiveTab] = useState('requests');
  const [loading, setLoading] = useState(false);
  const [requests, setRequests] = useState([]);
  const [grants, setGrants] = useState([]);
  const [rules, setRules] = useState([]);
  const [modules, setModules] = useState([]);
  const [verbs, setVerbs] = useState([]);
  const [stats, setStats] = useState({});
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  
  // 弹窗状态
  const [requestModalVisible, setRequestModalVisible] = useState(false);
  const [approveModalVisible, setApproveModalVisible] = useState(false);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [grantModalVisible, setGrantModalVisible] = useState(false);
  const [ruleModalVisible, setRuleModalVisible] = useState(false);
  
  // 当前选中的数据
  const [selectedRequest, setSelectedRequest] = useState(null);
  const [requestDetail, setRequestDetail] = useState(null);
  
  // 表单
  const [requestForm] = Form.useForm();
  const [approveForm] = Form.useForm();
  const [grantForm] = Form.useForm();
  const [ruleForm] = Form.useForm();

  // 获取可用模块列表
  const fetchModules = useCallback(async () => {
    try {
      const response = await approvalAPI.getAvailableModules();
      if (response.success) {
        setModules(response.data.modules || []);
      }
    } catch (error) {
      console.error('获取模块列表失败:', error);
    }
  }, []);

  // 获取可用操作权限列表
  const fetchVerbs = useCallback(async () => {
    try {
      const response = await approvalAPI.getAvailableVerbs();
      if (response.success) {
        setVerbs(response.data || []);
      }
    } catch (error) {
      console.error('获取操作权限列表失败:', error);
    }
  }, []);

  // 获取统计信息
  const fetchStats = useCallback(async () => {
    try {
      const response = await approvalAPI.getStats();
      if (response.success) {
        setStats(response.data || {});
      }
    } catch (error) {
      console.error('获取统计信息失败:', error);
    }
  }, []);

  // 获取申请列表
  const fetchRequests = useCallback(async (params = {}) => {
    setLoading(true);
    try {
      const response = await approvalAPI.listRequests({
        page: pagination.current,
        page_size: pagination.pageSize,
        ...params,
      });
      if (response.success) {
        setRequests(response.data.items || []);
        setPagination(prev => ({
          ...prev,
          total: response.data.total || 0,
        }));
      }
    } catch (error) {
      message.error('获取申请列表失败');
    } finally {
      setLoading(false);
    }
  }, [pagination.current, pagination.pageSize]);

  // 获取我的授权列表
  const fetchMyGrants = useCallback(async () => {
    setLoading(true);
    try {
      const response = await approvalAPI.getMyGrants();
      if (response.success) {
        setGrants(response.data.grants || []);
      }
    } catch (error) {
      message.error('获取授权列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  // 获取审批规则列表
  const fetchRules = useCallback(async () => {
    setLoading(true);
    try {
      const response = await approvalAPI.getRules();
      if (response.success) {
        setRules(response.data || []);
      }
    } catch (error) {
      message.error('获取审批规则失败');
    } finally {
      setLoading(false);
    }
  }, []);

  // 初始化加载
  useEffect(() => {
    fetchModules();
    fetchVerbs();
    fetchStats();
  }, [fetchModules, fetchVerbs, fetchStats]);

  // 根据 Tab 加载数据
  useEffect(() => {
    if (activeTab === 'requests') {
      fetchRequests();
    } else if (activeTab === 'my-grants') {
      fetchMyGrants();
    } else if (activeTab === 'rules') {
      fetchRules();
    }
  }, [activeTab, fetchRequests, fetchMyGrants, fetchRules]);

  // 创建权限申请
  const handleCreateRequest = async (values) => {
    try {
      const response = await approvalAPI.createRequest(values);
      if (response.success) {
        message.success('权限申请已提交');
        setRequestModalVisible(false);
        requestForm.resetFields();
        fetchRequests();
        fetchStats();
      } else {
        message.error(response.message || '提交失败');
      }
    } catch (error) {
      message.error('提交申请失败');
    }
  };

  // 审批权限申请
  const handleApprove = async (values) => {
    if (!selectedRequest) return;
    try {
      const response = await approvalAPI.approveRequest(selectedRequest.ID, values);
      if (response.success) {
        message.success(values.approved ? '已批准' : '已拒绝');
        setApproveModalVisible(false);
        approveForm.resetFields();
        setSelectedRequest(null);
        fetchRequests();
        fetchStats();
      } else {
        message.error(response.message || '审批失败');
      }
    } catch (error) {
      message.error('审批操作失败');
    }
  };

  // 取消权限申请
  const handleCancelRequest = async (id) => {
    try {
      const response = await approvalAPI.cancelRequest(id);
      if (response.success) {
        message.success('申请已取消');
        fetchRequests();
        fetchStats();
      } else {
        message.error(response.message || '取消失败');
      }
    } catch (error) {
      message.error('取消申请失败');
    }
  };

  // 查看申请详情
  const handleViewDetail = async (record) => {
    try {
      const response = await approvalAPI.getRequest(record.ID);
      if (response.success) {
        setRequestDetail(response.data);
        setDetailModalVisible(true);
      }
    } catch (error) {
      message.error('获取详情失败');
    }
  };

  // 手动授权
  const handleGrantPermission = async (values) => {
    try {
      const response = await approvalAPI.grantPermission(values);
      if (response.success) {
        message.success('权限已授予');
        setGrantModalVisible(false);
        grantForm.resetFields();
        fetchMyGrants();
        fetchStats();
      } else {
        message.error(response.message || '授权失败');
      }
    } catch (error) {
      message.error('授权操作失败');
    }
  };

  // 撤销权限
  const handleRevokePermission = async (grant) => {
    try {
      const response = await approvalAPI.revokePermission({
        user_id: grant.UserID,
        module: grant.Module,
        reason: '管理员撤销',
      });
      if (response.success) {
        message.success('权限已撤销');
        fetchMyGrants();
        fetchStats();
      } else {
        message.error(response.message || '撤销失败');
      }
    } catch (error) {
      message.error('撤销操作失败');
    }
  };

  // 创建审批规则
  const handleCreateRule = async (values) => {
    try {
      const response = await approvalAPI.createRule(values);
      if (response.success) {
        message.success('审批规则已创建');
        setRuleModalVisible(false);
        ruleForm.resetFields();
        fetchRules();
      } else {
        message.error(response.message || '创建失败');
      }
    } catch (error) {
      message.error('创建规则失败');
    }
  };

  // 删除审批规则
  const handleDeleteRule = async (id) => {
    try {
      const response = await approvalAPI.deleteRule(id);
      if (response.success) {
        message.success('规则已删除');
        fetchRules();
      } else {
        message.error(response.message || '删除失败');
      }
    } catch (error) {
      message.error('删除规则失败');
    }
  };

  // 申请列表表格列
  const requestColumns = [
    {
      title: 'ID',
      dataIndex: 'ID',
      width: 60,
    },
    {
      title: '申请人',
      dataIndex: 'Requester',
      render: (requester) => requester?.Username || '-',
    },
    {
      title: '目标用户',
      dataIndex: 'TargetUser',
      render: (user) => user?.Username || '-',
    },
    {
      title: '申请模块',
      dataIndex: 'RequestedModules',
      render: (modules) => {
        const moduleList = modules ? JSON.parse(modules) : [];
        return (
          <Space wrap>
            {moduleList.slice(0, 3).map(m => (
              <Tag key={m} color="blue">{m}</Tag>
            ))}
            {moduleList.length > 3 && (
              <Tag>+{moduleList.length - 3}</Tag>
            )}
          </Space>
        );
      },
    },
    {
      title: '申请权限',
      dataIndex: 'RequestedVerbs',
      render: (verbs) => {
        const verbList = verbs ? JSON.parse(verbs) : [];
        return (
          <Space wrap>
            {verbList.map(v => (
              <Tag key={v} color="cyan">{v}</Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: '优先级',
      dataIndex: 'Priority',
      render: (priority) => (
        <Tag color={priorityColors[priority] || 'default'}>
          {priorityLabels[priority] || '普通'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'Status',
      render: (status) => (
        <Badge 
          status={status === 'pending' ? 'processing' : (status === 'approved' ? 'success' : 'default')}
          text={
            <Tag color={statusColors[status]}>
              {statusLabels[status] || status}
            </Tag>
          }
        />
      ),
    },
    {
      title: '申请时间',
      dataIndex: 'CreatedAt',
      render: (time) => time ? new Date(time).toLocaleString() : '-',
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Tooltip title="查看详情">
            <Button 
              type="link" 
              icon={<EyeOutlined />} 
              onClick={() => handleViewDetail(record)}
            />
          </Tooltip>
          {record.Status === 'pending' && (
            <>
              <Tooltip title="审批">
                <Button 
                  type="link" 
                  icon={<CheckOutlined />} 
                  style={{ color: 'green' }}
                  onClick={() => {
                    setSelectedRequest(record);
                    setApproveModalVisible(true);
                  }}
                />
              </Tooltip>
              <Tooltip title="取消">
                <Popconfirm
                  title="确定要取消这个申请吗？"
                  onConfirm={() => handleCancelRequest(record.ID)}
                >
                  <Button type="link" icon={<CloseOutlined />} danger />
                </Popconfirm>
              </Tooltip>
            </>
          )}
        </Space>
      ),
    },
  ];

  // 授权列表表格列
  const grantColumns = [
    {
      title: '模块',
      dataIndex: 'Module',
      render: (module) => <Tag color="blue">{module}</Tag>,
    },
    {
      title: '操作权限',
      dataIndex: 'Verb',
      render: (verb) => <Tag color="cyan">{verb}</Tag>,
    },
    {
      title: '授权类型',
      dataIndex: 'GrantType',
      render: (type) => (
        <Tag color={type === 'approval' ? 'green' : 'orange'}>
          {type === 'approval' ? '审批授权' : '手动授权'}
        </Tag>
      ),
    },
    {
      title: '授权原因',
      dataIndex: 'Reason',
      ellipsis: true,
    },
    {
      title: '过期时间',
      dataIndex: 'ExpiresAt',
      render: (time) => time ? new Date(time).toLocaleString() : '永久',
    },
    {
      title: '状态',
      dataIndex: 'IsActive',
      render: (isActive) => (
        <Tag color={isActive ? 'green' : 'red'}>
          {isActive ? '有效' : '已失效'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        record.IsActive && (
          <Popconfirm
            title="确定要撤销这个权限吗？"
            onConfirm={() => handleRevokePermission(record)}
          >
            <Button type="link" danger>撤销</Button>
          </Popconfirm>
        )
      ),
    },
  ];

  // 规则列表表格列
  const ruleColumns = [
    {
      title: '规则名称',
      dataIndex: 'Name',
    },
    {
      title: '描述',
      dataIndex: 'Description',
      ellipsis: true,
    },
    {
      title: '条件类型',
      dataIndex: 'ConditionType',
      render: (type) => {
        const labels = {
          role_template: '角色模板',
          module: '模块',
          user_group: '用户组',
        };
        return labels[type] || type;
      },
    },
    {
      title: '自动审批',
      dataIndex: 'AutoApprove',
      render: (auto) => (
        <Tag color={auto ? 'green' : 'default'}>
          {auto ? '是' : '否'}
        </Tag>
      ),
    },
    {
      title: '最大有效期(天)',
      dataIndex: 'MaxValidDays',
      render: (days) => days || '无限制',
    },
    {
      title: '状态',
      dataIndex: 'IsActive',
      render: (isActive) => (
        <Tag color={isActive ? 'green' : 'red'}>
          {isActive ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Popconfirm
            title="确定要删除这个规则吗？"
            onConfirm={() => handleDeleteRule(record.ID)}
          >
            <Button type="link" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // 按类别分组的模块选择器
  const renderModuleSelector = () => {
    const grouped = modules.reduce((acc, m) => {
      if (!acc[m.category]) {
        acc[m.category] = [];
      }
      acc[m.category].push(m);
      return acc;
    }, {});

    const categoryLabels = {
      infrastructure: '基础设施',
      compute: '计算资源',
      devops: 'DevOps',
      data: '数据服务',
      monitoring: '监控告警',
      management: '项目管理',
      admin: '系统管理',
      storage: '存储服务',
      tools: '工具服务',
    };

    return (
      <div>
        {Object.entries(grouped).map(([category, categoryModules]) => (
          <div key={category} style={{ marginBottom: 16 }}>
            <Text strong style={{ display: 'block', marginBottom: 8 }}>
              {categoryLabels[category] || category}
            </Text>
            <Checkbox.Group style={{ width: '100%' }}>
              <Row>
                {categoryModules.map(m => (
                  <Col span={8} key={m.name}>
                    <Checkbox value={m.name}>
                      {m.displayName || m.name}
                    </Checkbox>
                  </Col>
                ))}
              </Row>
            </Checkbox.Group>
          </div>
        ))}
      </div>
    );
  };

  return (
    <div style={{ padding: 24 }}>
      <Title level={3}>
        <SafetyOutlined /> 权限审批管理
      </Title>
      
      <Paragraph type="secondary">
        管理用户权限申请、授权审批和自动化规则配置
      </Paragraph>

      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="待审批申请"
              value={stats.PendingRequests || 0}
              valueStyle={{ color: '#faad14' }}
              prefix={<HistoryOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已批准申请"
              value={stats.ApprovedRequests || 0}
              valueStyle={{ color: '#52c41a' }}
              prefix={<CheckOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="有效授权"
              value={stats.ActiveGrants || 0}
              valueStyle={{ color: '#1890ff' }}
              prefix={<SafetyOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="总申请数"
              value={stats.TotalRequests || 0}
              prefix={<FileTextOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Tabs 
          activeKey={activeTab} 
          onChange={setActiveTab}
          tabBarExtraContent={
            <Space>
              <Button 
                icon={<ReloadOutlined />}
                onClick={() => {
                  if (activeTab === 'requests') fetchRequests();
                  else if (activeTab === 'my-grants') fetchMyGrants();
                  else if (activeTab === 'rules') fetchRules();
                  fetchStats();
                }}
              >
                刷新
              </Button>
              {activeTab === 'requests' && (
                <Button 
                  type="primary" 
                  icon={<PlusOutlined />}
                  onClick={() => setRequestModalVisible(true)}
                >
                  申请权限
                </Button>
              )}
              {activeTab === 'my-grants' && (
                <Button 
                  type="primary" 
                  icon={<PlusOutlined />}
                  onClick={() => setGrantModalVisible(true)}
                >
                  手动授权
                </Button>
              )}
              {activeTab === 'rules' && (
                <Button 
                  type="primary" 
                  icon={<PlusOutlined />}
                  onClick={() => setRuleModalVisible(true)}
                >
                  创建规则
                </Button>
              )}
            </Space>
          }
        >
          <TabPane 
            tab={
              <span>
                <FileTextOutlined /> 权限申请
                {stats.PendingRequests > 0 && (
                  <Badge count={stats.PendingRequests} style={{ marginLeft: 8 }} />
                )}
              </span>
            } 
            key="requests"
          >
            <Table
              columns={requestColumns}
              dataSource={requests}
              rowKey="ID"
              loading={loading}
              pagination={{
                ...pagination,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total) => `共 ${total} 条`,
                onChange: (page, pageSize) => {
                  setPagination(prev => ({ ...prev, current: page, pageSize }));
                },
              }}
            />
          </TabPane>

          <TabPane 
            tab={<span><SafetyOutlined /> 我的授权</span>} 
            key="my-grants"
          >
            <Table
              columns={grantColumns}
              dataSource={grants}
              rowKey="ID"
              loading={loading}
              pagination={false}
            />
          </TabPane>

          <TabPane 
            tab={<span><SettingOutlined /> 审批规则</span>} 
            key="rules"
          >
            <Alert
              message="审批规则说明"
              description="配置自动审批规则，满足条件的申请将自动批准，无需人工审核。规则按优先级从高到低匹配。"
              type="info"
              showIcon
              style={{ marginBottom: 16 }}
            />
            <Table
              columns={ruleColumns}
              dataSource={rules}
              rowKey="ID"
              loading={loading}
              pagination={false}
            />
          </TabPane>
        </Tabs>
      </Card>

      {/* 创建申请弹窗 */}
      <Modal
        title="申请权限"
        open={requestModalVisible}
        onCancel={() => {
          setRequestModalVisible(false);
          requestForm.resetFields();
        }}
        footer={null}
        width={700}
      >
        <Form
          form={requestForm}
          layout="vertical"
          onFinish={handleCreateRequest}
          initialValues={{
            priority: 2,
            valid_days: 30,
            notify_email: true,
          }}
        >
          <Form.Item
            name="requested_modules"
            label="申请模块"
            rules={[{ required: true, message: '请选择申请的模块' }]}
          >
            <Select
              mode="multiple"
              placeholder="选择需要权限的模块"
              optionLabelProp="label"
            >
              {modules.map(m => (
                <Option key={m.name} value={m.name} label={m.displayName}>
                  <Space>
                    <Tag color="blue">{m.category}</Tag>
                    {m.displayName}
                  </Space>
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="requested_verbs"
            label="申请权限"
            rules={[{ required: true, message: '请选择操作权限' }]}
          >
            <Checkbox.Group>
              <Row>
                {verbs.map(v => (
                  <Col span={6} key={v}>
                    <Checkbox value={v}>{v}</Checkbox>
                  </Col>
                ))}
              </Row>
            </Checkbox.Group>
          </Form.Item>

          <Form.Item
            name="reason"
            label="申请原因"
            rules={[{ required: true, message: '请填写申请原因' }]}
          >
            <TextArea rows={3} placeholder="请详细说明申请权限的原因和用途" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="priority" label="优先级">
                <Select>
                  <Option value={1}>低</Option>
                  <Option value={2}>普通</Option>
                  <Option value={3}>高</Option>
                  <Option value={4}>紧急</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="valid_days" label="有效期(天)">
                <InputNumber min={1} max={365} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="related_ticket" label="关联工单">
                <Input placeholder="可选" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="notify_email" valuePropName="checked">
                <Checkbox>邮件通知</Checkbox>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="notify_dingtalk" valuePropName="checked">
                <Checkbox>钉钉通知</Checkbox>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                提交申请
              </Button>
              <Button onClick={() => {
                setRequestModalVisible(false);
                requestForm.resetFields();
              }}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 审批弹窗 */}
      <Modal
        title="审批权限申请"
        open={approveModalVisible}
        onCancel={() => {
          setApproveModalVisible(false);
          approveForm.resetFields();
          setSelectedRequest(null);
        }}
        footer={null}
      >
        {selectedRequest && (
          <div style={{ marginBottom: 16 }}>
            <Paragraph>
              <Text strong>申请人：</Text>{selectedRequest.Requester?.Username}
            </Paragraph>
            <Paragraph>
              <Text strong>申请模块：</Text>
              {JSON.parse(selectedRequest.RequestedModules || '[]').map(m => (
                <Tag key={m} color="blue">{m}</Tag>
              ))}
            </Paragraph>
            <Paragraph>
              <Text strong>申请权限：</Text>
              {JSON.parse(selectedRequest.RequestedVerbs || '[]').map(v => (
                <Tag key={v} color="cyan">{v}</Tag>
              ))}
            </Paragraph>
            <Paragraph>
              <Text strong>申请原因：</Text>{selectedRequest.Reason}
            </Paragraph>
          </div>
        )}
        <Divider />
        <Form
          form={approveForm}
          layout="vertical"
          onFinish={handleApprove}
        >
          <Form.Item
            name="approved"
            label="审批决定"
            rules={[{ required: true }]}
          >
            <Select placeholder="选择审批结果">
              <Option value={true}>
                <Text type="success"><CheckOutlined /> 批准</Text>
              </Option>
              <Option value={false}>
                <Text type="danger"><CloseOutlined /> 拒绝</Text>
              </Option>
            </Select>
          </Form.Item>

          <Form.Item name="comment" label="审批意见">
            <TextArea rows={3} placeholder="请填写审批意见（可选）" />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                确认审批
              </Button>
              <Button onClick={() => {
                setApproveModalVisible(false);
                approveForm.resetFields();
                setSelectedRequest(null);
              }}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情弹窗 */}
      <Modal
        title="申请详情"
        open={detailModalVisible}
        onCancel={() => {
          setDetailModalVisible(false);
          setRequestDetail(null);
        }}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            关闭
          </Button>
        ]}
        width={700}
      >
        {requestDetail && (
          <div>
            <Row gutter={16}>
              <Col span={12}>
                <Paragraph>
                  <Text strong>申请ID：</Text>{requestDetail.request?.ID}
                </Paragraph>
                <Paragraph>
                  <Text strong>申请人：</Text>{requestDetail.request?.Requester?.Username}
                </Paragraph>
                <Paragraph>
                  <Text strong>目标用户：</Text>{requestDetail.request?.TargetUser?.Username}
                </Paragraph>
                <Paragraph>
                  <Text strong>状态：</Text>
                  <Tag color={statusColors[requestDetail.request?.Status]}>
                    {statusLabels[requestDetail.request?.Status]}
                  </Tag>
                </Paragraph>
              </Col>
              <Col span={12}>
                <Paragraph>
                  <Text strong>优先级：</Text>
                  <Tag color={priorityColors[requestDetail.request?.Priority]}>
                    {priorityLabels[requestDetail.request?.Priority]}
                  </Tag>
                </Paragraph>
                <Paragraph>
                  <Text strong>有效期：</Text>{requestDetail.request?.ValidDays || '未设置'} 天
                </Paragraph>
                <Paragraph>
                  <Text strong>申请时间：</Text>
                  {requestDetail.request?.CreatedAt ? new Date(requestDetail.request.CreatedAt).toLocaleString() : '-'}
                </Paragraph>
              </Col>
            </Row>

            <Divider />

            <Paragraph>
              <Text strong>申请模块：</Text>
            </Paragraph>
            <Space wrap style={{ marginBottom: 16 }}>
              {JSON.parse(requestDetail.request?.RequestedModules || '[]').map(m => (
                <Tag key={m} color="blue">{m}</Tag>
              ))}
            </Space>

            <Paragraph>
              <Text strong>申请权限：</Text>
            </Paragraph>
            <Space wrap style={{ marginBottom: 16 }}>
              {JSON.parse(requestDetail.request?.RequestedVerbs || '[]').map(v => (
                <Tag key={v} color="cyan">{v}</Tag>
              ))}
            </Space>

            <Paragraph>
              <Text strong>申请原因：</Text>
            </Paragraph>
            <Paragraph>{requestDetail.request?.Reason}</Paragraph>

            {requestDetail.request?.ApproveComment && (
              <>
                <Paragraph>
                  <Text strong>审批意见：</Text>
                </Paragraph>
                <Paragraph>{requestDetail.request?.ApproveComment}</Paragraph>
              </>
            )}

            <Divider />

            <Paragraph>
              <Text strong>审批日志：</Text>
            </Paragraph>
            <Timeline>
              {(requestDetail.logs || []).map((log, index) => (
                <Timeline.Item 
                  key={index}
                  color={
                    log.Action === 'approve' ? 'green' : 
                    log.Action === 'reject' ? 'red' : 
                    log.Action === 'cancel' ? 'gray' : 'blue'
                  }
                >
                  <p>
                    <Text strong>{log.Operator?.Username || '系统'}</Text>
                    {' '}
                    {log.Action === 'submit' && '提交了申请'}
                    {log.Action === 'approve' && '批准了申请'}
                    {log.Action === 'reject' && '拒绝了申请'}
                    {log.Action === 'cancel' && '取消了申请'}
                    {log.Action === 'auto_approve' && '自动批准了申请'}
                  </p>
                  {log.Comment && <p><Text type="secondary">{log.Comment}</Text></p>}
                  <p><Text type="secondary">{new Date(log.CreatedAt).toLocaleString()}</Text></p>
                </Timeline.Item>
              ))}
            </Timeline>
          </div>
        )}
      </Modal>

      {/* 手动授权弹窗 */}
      <Modal
        title="手动授权"
        open={grantModalVisible}
        onCancel={() => {
          setGrantModalVisible(false);
          grantForm.resetFields();
        }}
        footer={null}
        width={600}
      >
        <Form
          form={grantForm}
          layout="vertical"
          onFinish={handleGrantPermission}
          initialValues={{
            valid_days: 30,
          }}
        >
          <Form.Item
            name="user_id"
            label="目标用户ID"
            rules={[{ required: true, message: '请输入用户ID' }]}
          >
            <InputNumber min={1} style={{ width: '100%' }} placeholder="输入用户ID" />
          </Form.Item>

          <Form.Item
            name="modules"
            label="授权模块"
            rules={[{ required: true, message: '请选择授权模块' }]}
          >
            <Select
              mode="multiple"
              placeholder="选择授权模块"
            >
              {modules.map(m => (
                <Option key={m.name} value={m.name}>
                  {m.displayName}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="verbs"
            label="授权权限"
            rules={[{ required: true, message: '请选择操作权限' }]}
          >
            <Checkbox.Group>
              <Row>
                {verbs.map(v => (
                  <Col span={6} key={v}>
                    <Checkbox value={v}>{v}</Checkbox>
                  </Col>
                ))}
              </Row>
            </Checkbox.Group>
          </Form.Item>

          <Form.Item
            name="reason"
            label="授权原因"
            rules={[{ required: true, message: '请填写授权原因' }]}
          >
            <TextArea rows={2} placeholder="请说明授权原因" />
          </Form.Item>

          <Form.Item name="valid_days" label="有效期(天)">
            <InputNumber min={1} max={365} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                确认授权
              </Button>
              <Button onClick={() => {
                setGrantModalVisible(false);
                grantForm.resetFields();
              }}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 创建规则弹窗 */}
      <Modal
        title="创建审批规则"
        open={ruleModalVisible}
        onCancel={() => {
          setRuleModalVisible(false);
          ruleForm.resetFields();
        }}
        footer={null}
        width={600}
      >
        <Form
          form={ruleForm}
          layout="vertical"
          onFinish={handleCreateRule}
          initialValues={{
            is_active: true,
            auto_approve: false,
            priority: 10,
            max_valid_days: 30,
          }}
        >
          <Form.Item
            name="name"
            label="规则名称"
            rules={[{ required: true, message: '请输入规则名称' }]}
          >
            <Input placeholder="输入规则名称" />
          </Form.Item>

          <Form.Item name="description" label="规则描述">
            <TextArea rows={2} placeholder="描述规则的用途和匹配条件" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="condition_type"
                label="条件类型"
                rules={[{ required: true }]}
              >
                <Select placeholder="选择条件类型">
                  <Option value="role_template">角色模板</Option>
                  <Option value="module">模块</Option>
                  <Option value="user_group">用户组</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="condition_value" label="条件值">
                <Select mode="tags" placeholder="输入条件值" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="priority" label="优先级">
                <InputNumber min={1} max={100} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="max_valid_days" label="最大有效期(天)">
                <InputNumber min={1} max={365} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="min_approvals" label="最少审批人数">
                <InputNumber min={0} max={10} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="auto_approve" valuePropName="checked">
                <Checkbox>自动审批</Checkbox>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="is_active" valuePropName="checked">
                <Checkbox>启用规则</Checkbox>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="notify_admins" valuePropName="checked">
                <Checkbox>通知管理员</Checkbox>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="allowed_modules" label="允许的模块（仅对模块条件类型有效）">
            <Select
              mode="multiple"
              placeholder="选择允许的模块，留空表示全部"
            >
              {modules.map(m => (
                <Option key={m.name} value={m.name}>
                  {m.displayName}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                创建规则
              </Button>
              <Button onClick={() => {
                setRuleModalVisible(false);
                ruleForm.resetFields();
              }}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default PermissionApprovalPage;
