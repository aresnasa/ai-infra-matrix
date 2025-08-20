import React, { useState, useEffect, useCallback } from 'react';
import { 
  Table, Card, Button, Modal, Form, Input, Select, message, Tag, Space, 
  Drawer, Descriptions, Badge, Alert, Typography, Row, Col, Statistic,
  Tooltip, Progress, Spin, Empty, Switch, Timeline, List, Avatar,
  Popconfirm, Divider, Tabs, Upload
} from 'antd';
import {
  UserOutlined, 
  TeamOutlined, 
  SyncOutlined, 
  SettingOutlined,
  EditOutlined, 
  DeleteOutlined, 
  PlusOutlined,
  ExclamationCircleOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloudSyncOutlined,
  SecurityScanOutlined,
  UserAddOutlined,
  ImportOutlined,
  ExportOutlined,
  ReloadOutlined,
  FilterOutlined,
  SearchOutlined,
  InfoCircleOutlined,
  LockOutlined,
  UnlockOutlined,
  CrownOutlined,
  GroupOutlined
} from '@ant-design/icons';
import { adminAPI, userAPI } from '../services/api';
import { useAuth } from '../hooks/useAuth';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TabPane } = Tabs;
const { Search } = Input;

const MultiUserLDAPManagement = () => {
  const { user, isAdmin } = useAuth();
  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState([]);
  const [userGroups, setUserGroups] = useState([]);
  const [roles, setRoles] = useState([]);
  const [ldapConfig, setLdapConfig] = useState(null);
  const [syncStatus, setSyncStatus] = useState(null);
  const [syncHistory, setSyncHistory] = useState([]);
  const [selectedUser, setSelectedUser] = useState(null);
  const [userDetailVisible, setUserDetailVisible] = useState(false);
  const [userModalVisible, setUserModalVisible] = useState(false);
  const [groupModalVisible, setGroupModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState(null);
  const [editingGroup, setEditingGroup] = useState(null);
  const [form] = Form.useForm();
  const [groupForm] = Form.useForm();
  const [statistics, setStatistics] = useState({});
  const [filters, setFilters] = useState({
    authSource: 'all',
    status: 'all',
    role: 'all',
    searchText: ''
  });
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0
  });

  // 加载所有数据
  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [usersRes, groupsRes, rolesRes, configRes, statsRes] = await Promise.all([
        adminAPI.getUsers({ 
          page: pagination.current,
          pageSize: pagination.pageSize,
          ...filters 
        }),
        adminAPI.getUserGroups(),
        adminAPI.getRoles(),
        adminAPI.getLDAPConfig(),
        adminAPI.getUserStatistics()
      ]);

      setUsers(usersRes.data.users || []);
      setPagination(prev => ({ ...prev, total: usersRes.data.total || 0 }));
      setUserGroups(groupsRes.data || []);
      setRoles(rolesRes.data || []);
      setLdapConfig(configRes.data);
      setStatistics(statsRes.data || {});
    } catch (error) {
      console.error('加载数据失败:', error);
      message.error('加载数据失败');
    } finally {
      setLoading(false);
    }
  }, [pagination.current, pagination.pageSize, filters]);

  // 加载同步状态和历史
  const loadSyncData = useCallback(async () => {
    if (!isAdmin) return;
    
    try {
      const [statusRes, historyRes] = await Promise.all([
        adminAPI.getLDAPSyncStatus(),
        adminAPI.getLDAPSyncHistory(10)
      ]);

      setSyncStatus(statusRes.data);
      setSyncHistory(historyRes.data.history || []);
    } catch (error) {
      console.error('加载同步数据失败:', error);
    }
  }, [isAdmin]);

  useEffect(() => {
    loadData();
    loadSyncData();
  }, [loadData, loadSyncData]);

  // 启动LDAP同步
  const handleLDAPSync = async (options = {}) => {
    if (!ldapConfig?.enabled) {
      message.error('LDAP未启用，无法同步');
      return;
    }

    Modal.confirm({
      title: '确认同步',
      content: '确定要从LDAP同步用户和组信息吗？这可能需要几分钟时间。',
      okText: '开始同步',
      cancelText: '取消',
      onOk: async () => {
        try {
          const response = await adminAPI.syncLDAPUsers(options);
          message.success('LDAP同步已启动');
          
          // 开始轮询同步状态
          const pollSync = async () => {
            try {
              const statusRes = await adminAPI.getLDAPSyncStatus(response.data.sync_id);
              setSyncStatus(statusRes.data);
              
              if (statusRes.data.status === 'completed' || statusRes.data.status === 'failed') {
                loadData();
                loadSyncData();
                return;
              }
              
              setTimeout(pollSync, 2000);
            } catch (error) {
              console.error('检查同步状态失败:', error);
            }
          };
          
          setTimeout(pollSync, 1000);
        } catch (error) {
          message.error('启动同步失败: ' + (error.response?.data?.error || error.message));
        }
      }
    });
  };

  // 测试LDAP连接
  const testLDAPConnection = async () => {
    try {
      setLoading(true);
      const response = await adminAPI.testLDAPConnection();
      
      if (response.data.success) {
        message.success('LDAP连接测试成功');
      } else {
        message.error('LDAP连接测试失败: ' + response.data.message);
      }
    } catch (error) {
      message.error('LDAP连接测试失败: ' + (error.response?.data?.error || error.message));
    } finally {
      setLoading(false);
    }
  };

  // 查看用户详情
  const viewUserDetail = async (userId) => {
    try {
      const response = await adminAPI.getUserWithAuthSource(userId);
      setSelectedUser(response.data);
      setUserDetailVisible(true);
    } catch (error) {
      message.error('获取用户详情失败');
      console.error('获取用户详情失败:', error);
    }
  };

  // 编辑用户
  const editUser = (user) => {
    setEditingUser(user);
    form.setFieldsValue({
      username: user.username,
      email: user.email,
      is_active: user.is_active,
      roles: user.roles?.map(role => role.name) || [],
      user_groups: user.user_groups?.map(group => group.id) || []
    });
    setUserModalVisible(true);
  };

  // 保存用户
  const handleUserSave = async (values) => {
    try {
      if (editingUser) {
        await adminAPI.updateUser(editingUser.id, values);
        message.success('用户更新成功');
      } else {
        await adminAPI.createUser(values);
        message.success('用户创建成功');
      }
      
      setUserModalVisible(false);
      setEditingUser(null);
      form.resetFields();
      loadData();
    } catch (error) {
      message.error('保存用户失败: ' + (error.response?.data?.error || error.message));
    }
  };

  // 删除用户
  const deleteUser = async (userId) => {
    try {
      await adminAPI.deleteUser(userId);
      message.success('用户删除成功');
      loadData();
    } catch (error) {
      message.error('删除用户失败: ' + (error.response?.data?.error || error.message));
    }
  };

  // 重置用户密码
  const resetUserPassword = async (userId) => {
    Modal.confirm({
      title: '重置密码',
      content: '确定要重置此用户的密码吗？新密码将通过邮件发送给用户。',
      okText: '重置',
      cancelText: '取消',
      onOk: async () => {
        try {
          await adminAPI.resetUserPassword(userId);
          message.success('密码重置成功');
        } catch (error) {
          message.error('密码重置失败: ' + (error.response?.data?.error || error.message));
        }
      }
    });
  };

  // 用户表格列定义
  const userColumns = [
    {
      title: '用户',
      key: 'user',
      render: (_, record) => (
        <Space>
          <Avatar icon={<UserOutlined />} size="small" />
          <div>
            <div>{record.username}</div>
            <Text type="secondary" style={{ fontSize: '12px' }}>{record.email}</Text>
          </div>
        </Space>
      ),
    },
    {
      title: '认证源',
      dataIndex: 'auth_source',
      key: 'auth_source',
      filters: [
        { text: 'LDAP', value: 'ldap' },
        { text: '本地', value: 'local' }
      ],
      render: (authSource) => (
        <Tag color={authSource === 'ldap' ? 'green' : 'blue'}>
          {authSource === 'ldap' ? 'LDAP' : '本地'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      filters: [
        { text: '活跃', value: true },
        { text: '禁用', value: false }
      ],
      render: (isActive) => (
        <Badge 
          status={isActive ? 'success' : 'error'} 
          text={isActive ? '活跃' : '禁用'} 
        />
      ),
    },
    {
      title: '角色',
      dataIndex: 'roles',
      key: 'roles',
      render: (roles) => (
        <Space wrap>
          {roles?.map(role => (
            <Tag key={role.name} color="purple">
              {role.name === 'admin' && <CrownOutlined style={{ marginRight: 4 }} />}
              {role.name}
            </Tag>
          )) || '-'}
        </Space>
      ),
    },
    {
      title: '用户组',
      dataIndex: 'user_groups',
      key: 'user_groups',
      render: (groups) => (
        <Space wrap>
          {groups?.slice(0, 2).map(group => (
            <Tag key={group.id} color="orange">
              <GroupOutlined style={{ marginRight: 4 }} />
              {group.name}
            </Tag>
          ))}
          {groups?.length > 2 && (
            <Tag color="orange">+{groups.length - 2}</Tag>
          )}
        </Space>
      ),
    },
    {
      title: '最后登录',
      dataIndex: 'last_login',
      key: 'last_login',
      render: (lastLogin) => (
        lastLogin ? new Date(lastLogin).toLocaleString() : '从未登录'
      ),
    },
    {
      title: '操作',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Tooltip title="查看详情">
            <Button 
              type="text" 
              size="small" 
              icon={<InfoCircleOutlined />}
              onClick={() => viewUserDetail(record.id)}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button 
              type="text" 
              size="small" 
              icon={<EditOutlined />}
              onClick={() => editUser(record)}
            />
          </Tooltip>
          {record.auth_source === 'local' && (
            <Tooltip title="重置密码">
              <Button 
                type="text" 
                size="small" 
                icon={<LockOutlined />}
                onClick={() => resetUserPassword(record.id)}
              />
            </Tooltip>
          )}
          <Popconfirm
            title="确定要删除此用户吗？"
            onConfirm={() => deleteUser(record.id)}
            disabled={record.username === user?.username}
          >
            <Tooltip title={record.username === user?.username ? '不能删除自己' : '删除'}>
              <Button 
                type="text" 
                size="small" 
                danger
                disabled={record.username === user?.username}
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // 过滤用户数据
  const filteredUsers = users.filter(user => {
    const matchesAuth = filters.authSource === 'all' || user.auth_source === filters.authSource;
    const matchesStatus = filters.status === 'all' || 
      (filters.status === 'active' && user.is_active) ||
      (filters.status === 'inactive' && !user.is_active);
    const matchesRole = filters.role === 'all' || 
      user.roles?.some(role => role.name === filters.role);
    const matchesSearch = !filters.searchText || 
      user.username.toLowerCase().includes(filters.searchText.toLowerCase()) ||
      user.email.toLowerCase().includes(filters.searchText.toLowerCase());
    
    return matchesAuth && matchesStatus && matchesRole && matchesSearch;
  });

  // 同步状态组件
  const SyncStatusCard = () => (
    <Card 
      title={
        <Space>
          <CloudSyncOutlined />
          LDAP同步状态
        </Space>
      }
      extra={
        <Space>
          <Button 
            size="small" 
            icon={<ReloadOutlined />}
            onClick={loadSyncData}
          >
            刷新
          </Button>
          <Button 
            type="primary"
            size="small"
            icon={<SyncOutlined />}
            onClick={() => handleLDAPSync()}
            disabled={!ldapConfig?.enabled || syncStatus?.status === 'running'}
            loading={syncStatus?.status === 'running'}
          >
            同步用户
          </Button>
        </Space>
      }
      size="small"
    >
      {syncStatus ? (
        <Space direction="vertical" style={{ width: '100%' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Text>状态：</Text>
            <Badge 
              status={
                syncStatus.status === 'running' ? 'processing' :
                syncStatus.status === 'completed' ? 'success' : 'error'
              } 
              text={
                syncStatus.status === 'running' ? '同步中' :
                syncStatus.status === 'completed' ? '已完成' : '失败'
              }
            />
          </div>
          {syncStatus.status === 'running' && syncStatus.progress > 0 && (
            <Progress 
              percent={syncStatus.progress} 
              size="small"
              status="active"
            />
          )}
          {syncStatus.result && (
            <div style={{ fontSize: '12px' }}>
              <Text type="secondary">
                创建: {syncStatus.result.users_created} | 
                更新: {syncStatus.result.users_updated} | 
                组: {syncStatus.result.groups_created}
              </Text>
            </div>
          )}
        </Space>
      ) : (
        <Text type="secondary">暂无同步记录</Text>
      )}
    </Card>
  );

  // 统计卡片
  const StatisticsCards = () => (
    <Row gutter={16} style={{ marginBottom: 24 }}>
      <Col span={6}>
        <Card>
          <Statistic
            title="总用户数"
            value={statistics.totalUsers || 0}
            prefix={<UserOutlined />}
            valueStyle={{ color: '#1890ff' }}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="LDAP用户"
            value={statistics.ldapUsers || 0}
            prefix={<SecurityScanOutlined />}
            valueStyle={{ color: '#52c41a' }}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="活跃用户"
            value={statistics.activeUsers || 0}
            prefix={<CheckCircleOutlined />}
            valueStyle={{ color: '#722ed1' }}
          />
        </Card>
      </Col>
      <Col span={6}>
        <Card>
          <Statistic
            title="用户组"
            value={userGroups.length}
            prefix={<TeamOutlined />}
            valueStyle={{ color: '#fa8c16' }}
          />
        </Card>
      </Col>
    </Row>
  );

  if (!isAdmin) {
    return (
      <div style={{ padding: '24px' }}>
        <Alert
          message="权限不足"
          description="您需要管理员权限才能访问用户管理功能"
          type="error"
          showIcon
        />
      </div>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title level={2}>多用户LDAP管理</Title>
        <Space>
          <Button icon={<ImportOutlined />}>
            批量导入
          </Button>
          <Button icon={<ExportOutlined />}>
            导出用户
          </Button>
          <Button 
            type="primary" 
            icon={<UserAddOutlined />}
            onClick={() => {
              setEditingUser(null);
              form.resetFields();
              setUserModalVisible(true);
            }}
          >
            新增用户
          </Button>
        </Space>
      </div>

      {/* LDAP配置状态 */}
      {ldapConfig && (
        <Alert
          message={`LDAP配置${ldapConfig.enabled ? '已启用' : '未启用'}`}
          description={
            ldapConfig.enabled ? 
            `服务器: ${ldapConfig.host}:${ldapConfig.port} | 基础DN: ${ldapConfig.base_dn}` :
            '请先在系统设置中配置并启用LDAP'
          }
          type={ldapConfig.enabled ? 'success' : 'warning'}
          showIcon
          style={{ marginBottom: 24 }}
          action={
            <Space>
              {ldapConfig.enabled && (
                <Button size="small" onClick={testLDAPConnection}>
                  测试连接
                </Button>
              )}
              <Button size="small" type="link" href="/admin/settings">
                配置LDAP
              </Button>
            </Space>
          }
        />
      )}

      <StatisticsCards />

      <Row gutter={16}>
        <Col span={18}>
          <Card 
            title="用户列表"
            extra={
              <Space>
                <Search
                  placeholder="搜索用户名或邮箱"
                  style={{ width: 200 }}
                  value={filters.searchText}
                  onChange={(e) => setFilters(prev => ({ ...prev, searchText: e.target.value }))}
                  allowClear
                />
                <Select
                  placeholder="认证源"
                  style={{ width: 100 }}
                  value={filters.authSource}
                  onChange={(value) => setFilters(prev => ({ ...prev, authSource: value }))}
                >
                  <Option value="all">全部</Option>
                  <Option value="ldap">LDAP</Option>
                  <Option value="local">本地</Option>
                </Select>
                <Select
                  placeholder="状态"
                  style={{ width: 100 }}
                  value={filters.status}
                  onChange={(value) => setFilters(prev => ({ ...prev, status: value }))}
                >
                  <Option value="all">全部</Option>
                  <Option value="active">活跃</Option>
                  <Option value="inactive">禁用</Option>
                </Select>
              </Space>
            }
          >
            <Table
              columns={userColumns}
              dataSource={filteredUsers}
              rowKey="id"
              loading={loading}
              pagination={{
                ...pagination,
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
                onChange: (page, pageSize) => {
                  setPagination(prev => ({ ...prev, current: page, pageSize }));
                }
              }}
              size="small"
            />
          </Card>
        </Col>
        
        <Col span={6}>
          <Space direction="vertical" style={{ width: '100%' }}>
            <SyncStatusCard />
            
            <Card 
              title="同步历史" 
              size="small"
              bodyStyle={{ maxHeight: 300, overflowY: 'auto' }}
            >
              {syncHistory.length > 0 ? (
                <Timeline size="small">
                  {syncHistory.map((record, index) => (
                    <Timeline.Item
                      key={index}
                      color={record.status === 'completed' ? 'green' : 'red'}
                      dot={
                        record.status === 'completed' ? 
                        <CheckCircleOutlined /> : 
                        <ExclamationCircleOutlined />
                      }
                    >
                      <div style={{ fontSize: '12px' }}>
                        <div>{new Date(record.start_time).toLocaleString()}</div>
                        <Text type="secondary">
                          {record.status === 'completed' ? '同步成功' : '同步失败'}
                          {record.result && ` - 创建${record.result.users_created}个用户`}
                        </Text>
                      </div>
                    </Timeline.Item>
                  ))}
                </Timeline>
              ) : (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无历史记录" />
              )}
            </Card>
          </Space>
        </Col>
      </Row>

      {/* 用户详情抽屉 */}
      <Drawer
        title="用户详情"
        width={600}
        onClose={() => setUserDetailVisible(false)}
        open={userDetailVisible}
      >
        {selectedUser && (
          <Tabs defaultActiveKey="basic">
            <TabPane tab="基本信息" key="basic">
              <Descriptions bordered column={1}>
                <Descriptions.Item label="用户名">{selectedUser.user.username}</Descriptions.Item>
                <Descriptions.Item label="邮箱">{selectedUser.user.email}</Descriptions.Item>
                <Descriptions.Item label="认证源">
                  <Tag color={selectedUser.auth_source === 'ldap' ? 'green' : 'blue'}>
                    {selectedUser.auth_source === 'ldap' ? 'LDAP' : '本地'}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="状态">
                  <Badge 
                    status={selectedUser.user.is_active ? 'success' : 'error'} 
                    text={selectedUser.user.is_active ? '活跃' : '禁用'} 
                  />
                </Descriptions.Item>
                <Descriptions.Item label="管理员">
                  {selectedUser.is_admin ? '是' : '否'}
                </Descriptions.Item>
                <Descriptions.Item label="创建时间">
                  {new Date(selectedUser.user.created_at).toLocaleString()}
                </Descriptions.Item>
                <Descriptions.Item label="最后登录">
                  {selectedUser.user.last_login ? 
                    new Date(selectedUser.user.last_login).toLocaleString() : 
                    '从未登录'
                  }
                </Descriptions.Item>
                {selectedUser.user.ldap_dn && (
                  <Descriptions.Item label="LDAP DN">
                    <Text code>{selectedUser.user.ldap_dn}</Text>
                  </Descriptions.Item>
                )}
              </Descriptions>
            </TabPane>
            
            <TabPane tab="角色权限" key="roles">
              <Space direction="vertical" style={{ width: '100%' }}>
                <div>
                  <Text strong>用户角色：</Text>
                  <div style={{ marginTop: 8 }}>
                    {selectedUser.user.roles?.length > 0 ? (
                      selectedUser.user.roles.map(role => (
                        <Tag key={role.name} color="purple" style={{ marginBottom: 8 }}>
                          {role.name === 'admin' && <CrownOutlined style={{ marginRight: 4 }} />}
                          {role.name}
                        </Tag>
                      ))
                    ) : (
                      <Text type="secondary">暂无角色</Text>
                    )}
                  </div>
                </div>
                
                <Divider />
                
                <div>
                  <Text strong>用户组：</Text>
                  <div style={{ marginTop: 8 }}>
                    {selectedUser.user.user_groups?.length > 0 ? (
                      selectedUser.user.user_groups.map(group => (
                        <Tag key={group.id} color="orange" style={{ marginBottom: 8 }}>
                          <GroupOutlined style={{ marginRight: 4 }} />
                          {group.name}
                        </Tag>
                      ))
                    ) : (
                      <Text type="secondary">暂无用户组</Text>
                    )}
                  </div>
                </div>
              </Space>
            </TabPane>
          </Tabs>
        )}
      </Drawer>

      {/* 用户编辑模态框 */}
      <Modal
        title={editingUser ? '编辑用户' : '新增用户'}
        open={userModalVisible}
        onCancel={() => {
          setUserModalVisible(false);
          setEditingUser(null);
          form.resetFields();
        }}
        onOk={() => form.submit()}
        okText="保存"
        cancelText="取消"
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleUserSave}
        >
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="用户名" disabled={!!editingUser} />
          </Form.Item>

          <Form.Item
            name="email"
            label="邮箱"
            rules={[
              { required: true, message: '请输入邮箱' },
              { type: 'email', message: '请输入有效的邮箱地址' }
            ]}
          >
            <Input placeholder="邮箱地址" />
          </Form.Item>

          {!editingUser && (
            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true, message: '请输入密码' }]}
            >
              <Input.Password placeholder="密码" />
            </Form.Item>
          )}

          <Form.Item
            name="roles"
            label="角色"
          >
            <Select
              mode="multiple"
              placeholder="选择角色"
              options={roles.map(role => ({ label: role.name, value: role.name }))}
            />
          </Form.Item>

          <Form.Item
            name="user_groups"
            label="用户组"
          >
            <Select
              mode="multiple"
              placeholder="选择用户组"
              options={userGroups.map(group => ({ label: group.name, value: group.id }))}
            />
          </Form.Item>

          <Form.Item
            name="is_active"
            label="启用用户"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default MultiUserLDAPManagement;
