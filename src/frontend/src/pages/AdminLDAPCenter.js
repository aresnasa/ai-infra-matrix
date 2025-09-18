import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Form,
  Input,
  Button,
  Switch,
  Select,
  message,
  Alert,
  Space,
  Typography,
  Spin,
  Tooltip,
  Row,
  Col,
  Tabs,
  Table,
  Tag,
  Modal,
  Descriptions,
  Statistic,
  Progress,
  Empty,
  Timeline,
  Badge,
  Divider,
  Popconfirm
} from 'antd';
import {
  SafetyOutlined,
  ExperimentOutlined,
  SaveOutlined,
  InfoCircleOutlined,
  BookOutlined,
  UserOutlined,
  TeamOutlined,
  SyncOutlined,
  SettingOutlined,
  LockOutlined,
  UnlockOutlined,
  EyeOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined
} from '@ant-design/icons';
import { adminAPI } from '../services/api';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TextArea } = Input;
const { TabPane } = Tabs;

/**
 * 统一的LDAP管理中心
 * 整合了LDAP配置、用户管理、同步等功能
 * 采用只读LDAP策略，保证数据安全
 */
const AdminLDAPCenter = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [saving, setSaving] = useState(false);
  
  // LDAP配置相关状态
  const [config, setConfig] = useState(null);
  const [testResult, setTestResult] = useState(null);
  const [ldapEnabled, setLdapEnabled] = useState(false);
  
  // 用户管理相关状态
  const [users, setUsers] = useState([]);
  const [ldapUsers, setLdapUsers] = useState([]);
  const [syncStatus, setSyncStatus] = useState(null);
  const [syncHistory, setSyncHistory] = useState([]);
  const [selectedUser, setSelectedUser] = useState(null);
  const [userModalVisible, setUserModalVisible] = useState(false);
  
  // 活动Tab
  const [activeTab, setActiveTab] = useState('config');

  useEffect(() => {
    loadLDAPConfig();
    loadUsers();
    loadSyncHistory();
  }, []);

  // 加载LDAP配置
  const loadLDAPConfig = async () => {
    setLoading(true);
    try {
      const response = await adminAPI.getLDAPConfig();
      const ldapConfig = response.data;
      setConfig(ldapConfig);
      form.setFieldsValue(ldapConfig);
      setLdapEnabled(ldapConfig.enabled || ldapConfig.is_enabled || false);
    } catch (error) {
      if (error.response?.status !== 404) {
        message.error('加载LDAP配置失败');
      }
    } finally {
      setLoading(false);
    }
  };

  // 加载用户列表
  const loadUsers = async () => {
    try {
      const [localResponse, ldapResponse] = await Promise.all([
        adminAPI.getUsers(),
        adminAPI.getLDAPUsers().catch(() => ({ data: { users: [] } }))
      ]);
      
      setUsers(localResponse.data.users || localResponse.data || []);
      setLdapUsers(ldapResponse.data.users || []);
    } catch (error) {
      message.error('加载用户列表失败');
    }
  };

  // 加载同步历史
  const loadSyncHistory = async () => {
    try {
      const response = await adminAPI.getLDAPSyncHistory();
      setSyncHistory(response.data.history || []);
    } catch (error) {
      // 同步历史可能不存在，忽略错误
    }
  };

  // 保存LDAP配置
  const handleSave = async (values) => {
    setSaving(true);
    try {
      await adminAPI.updateLDAPConfig(values);
      message.success('LDAP配置保存成功');
      await loadLDAPConfig();
      setTestResult(null); // 清除之前的测试结果
    } catch (error) {
      message.error(error.response?.data?.message || '保存LDAP配置失败');
    } finally {
      setSaving(false);
    }
  };

  // 测试LDAP连接
  const handleTest = async () => {
    try {
      const values = await form.validateFields();
      if (!values.enabled && !values.is_enabled) {
        message.warning('请先启用LDAP认证');
        return;
      }
      
      setTesting(true);
      const response = await adminAPI.testLDAPConnection(values);
      setTestResult({
        success: true,
        message: response.data.message || '连接测试成功'
      });
      message.success('LDAP连接测试成功');
    } catch (error) {
      if (error.errorFields) {
        message.error('请先完善表单信息');
        return;
      }
      setTestResult({
        success: false,
        message: error.response?.data?.message || error.response?.data?.error || '连接测试失败'
      });
      message.error('LDAP连接测试失败');
    } finally {
      setTesting(false);
    }
  };

  // 同步LDAP用户
  const handleSyncUsers = async () => {
    if (!ldapEnabled) {
      message.warning('请先启用LDAP认证');
      return;
    }

    setSyncing(true);
    try {
      const response = await adminAPI.syncLDAPUsers();
      message.success('LDAP用户同步完成');
      setSyncStatus(response.data);
      await loadUsers();
      await loadSyncHistory();
    } catch (error) {
      message.error(error.response?.data?.message || '同步LDAP用户失败');
    } finally {
      setSyncing(false);
    }
  };

  // 切换用户启用状态
  const toggleUserStatus = async (userId, currentStatus) => {
    try {
      await adminAPI.toggleUserStatus(userId, !currentStatus);
      message.success(currentStatus ? '用户已禁用' : '用户已启用');
      await loadUsers();
    } catch (error) {
      message.error('切换用户状态失败');
    }
  };

  // 重置表单
  const handleReset = () => {
    if (config) {
      form.setFieldsValue(config);
      setLdapEnabled(config.enabled || config.is_enabled || false);
    } else {
      form.resetFields();
      setLdapEnabled(false);
    }
    setTestResult(null);
  };

  // LDAP开关切换
  const handleLdapToggle = (enabled) => {
    setLdapEnabled(enabled);
    if (!enabled) {
      setTestResult(null);
    }
  };

  // 查看用户详情
  const showUserDetails = (user) => {
    setSelectedUser(user);
    setUserModalVisible(true);
  };

  // 用户表格列定义
  const userColumns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (text, record) => (
        <Space>
          <Text strong>{text}</Text>
          {record.auth_source === 'ldap' && (
            <Tag color="blue">LDAP</Tag>
          )}
          {!record.is_active && (
            <Tag color="red">已禁用</Tag>
          )}
        </Space>
      )
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email'
    },
    {
      title: '姓名',
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: '认证源',
      dataIndex: 'auth_source',
      key: 'auth_source',
      render: (source) => (
        <Tag color={source === 'ldap' ? 'blue' : 'green'}>
          {source === 'ldap' ? 'LDAP' : '本地'}
        </Tag>
      )
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (active) => (
        <Badge 
          status={active ? 'success' : 'error'} 
          text={active ? '启用' : '禁用'} 
        />
      )
    },
    {
      title: '最后登录',
      dataIndex: 'last_login',
      key: 'last_login',
      render: (lastLogin) => lastLogin ? new Date(lastLogin).toLocaleString() : '从未登录'
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button
            size="small"
            icon={<EyeOutlined />}
            onClick={() => showUserDetails(record)}
          >
            详情
          </Button>
          {record.auth_source === 'ldap' && (
            <Popconfirm
              title={`确定要${record.is_active ? '禁用' : '启用'}该用户吗？`}
              description="这只会影响本系统的访问权限，不会修改LDAP数据"
              onConfirm={() => toggleUserStatus(record.id, record.is_active)}
              okText="确定"
              cancelText="取消"
            >
              <Button
                size="small"
                icon={record.is_active ? <LockOutlined /> : <UnlockOutlined />}
                danger={record.is_active}
              >
                {record.is_active ? '禁用' : '启用'}
              </Button>
            </Popconfirm>
          )}
        </Space>
      )
    }
  ];

  // 渲染LDAP配置Tab
  const renderConfigTab = () => (
    <div>
      {/* 安全提示 */}
      <Alert
        message="LDAP只读模式"
        description={
          <div>
            <p>✅ <strong>安全策略</strong>: 本系统采用LDAP只读模式，仅用于认证和用户同步</p>
            <p>🔒 <strong>用户管理</strong>: 所有用户的创建、修改、删除需要通过企业LDAP系统进行</p>
            <p>📋 <strong>本地管理</strong>: 仅支持禁用/启用本系统的用户访问权限</p>
          </div>
        }
        type="info"
        showIcon
        style={{ marginBottom: 24 }}
      />

      {/* 测试结果显示 */}
      {testResult && (
        <Alert
          message={testResult.success ? '连接测试成功' : '连接测试失败'}
          description={testResult.message}
          type={testResult.success ? 'success' : 'error'}
          showIcon
          closable
          style={{ marginBottom: 24 }}
        />
      )}

      <Form
        form={form}
        layout="vertical"
        onFinish={handleSave}
        initialValues={{
          enabled: false,
          is_enabled: false,
          port: 389,
          use_ssl: false,
          skip_verify: false,
          user_filter: "(objectClass=person)",
          username_attr: "uid",
          name_attr: "cn",
          email_attr: "mail"
        }}
      >
        {/* 基本配置 */}
        <Card size="small" title="基本配置" style={{ marginBottom: 16 }}>
          <Form.Item
            name={['enabled', 'is_enabled']}
            label="启用LDAP认证"
            valuePropName="checked"
            extra={ldapEnabled ? "LDAP认证已启用，用户可通过企业账户登录" : "LDAP认证已禁用，仅本地账户可登录"}
          >
            <Switch 
              checked={ldapEnabled}
              onChange={handleLdapToggle}
              checkedChildren="启用" 
              unCheckedChildren="禁用"
            />
          </Form.Item>

          <Row gutter={16}>
            <Col span={16}>
              <Form.Item
                name="server"
                label="LDAP服务器"
                rules={[{ required: true, message: '请输入LDAP服务器地址' }]}
              >
                <Input placeholder="ldap.company.com 或 192.168.1.100" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="port"
                label="端口"
                rules={[{ required: true, message: '请输入端口号' }]}
              >
                <Input placeholder="389" type="number" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name={['use_ssl', 'useSSL']}
                label="使用SSL/TLS"
                valuePropName="checked"
              >
                <Switch checkedChildren="启用" unCheckedChildren="禁用" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name={['skip_verify', 'skipVerify']}
                label="跳过证书验证"
                valuePropName="checked"
              >
                <Switch checkedChildren="跳过" unCheckedChildren="验证" />
              </Form.Item>
            </Col>
          </Row>
        </Card>

        {/* 绑定配置 */}
        <Card size="small" title="绑定认证" style={{ marginBottom: 16 }}>
          <Form.Item
            name={['bind_dn', 'bindDN']}
            label="绑定DN"
            rules={[{ required: true, message: '请输入绑定DN' }]}
          >
            <Input placeholder="cn=admin,dc=company,dc=com" />
          </Form.Item>

          <Form.Item
            name={['bind_password', 'bindPassword']}
            label="绑定密码"
            rules={[{ required: true, message: '请输入绑定密码' }]}
          >
            <Input.Password placeholder="管理员密码" />
          </Form.Item>

          <Form.Item
            name={['base_dn', 'baseDN']}
            label="基准DN"
            rules={[{ required: true, message: '请输入基准DN' }]}
          >
            <Input placeholder="dc=company,dc=com" />
          </Form.Item>
        </Card>

        {/* 用户配置 */}
        <Card size="small" title="用户属性映射" style={{ marginBottom: 16 }}>
          <Form.Item
            name={['user_filter', 'userFilter']}
            label="用户过滤器"
            extra="使用{username}作为用户名占位符"
          >
            <Input placeholder="(uid={username})" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name={['username_attr', 'usernameAttr']}
                label="用户名属性"
              >
                <Input placeholder="uid" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name={['name_attr', 'nameAttr']}
                label="姓名属性"
              >
                <Input placeholder="cn" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name={['email_attr', 'emailAttr']}
                label="邮箱属性"
              >
                <Input placeholder="mail" />
              </Form.Item>
            </Col>
          </Row>
        </Card>

        {/* 操作按钮 */}
        <Card size="small">
          <Space>
            <Button 
              type="primary" 
              icon={<SaveOutlined />}
              htmlType="submit"
              loading={saving}
              disabled={!ldapEnabled}
            >
              保存配置
            </Button>
            
            <Button 
              icon={<ExperimentOutlined />} 
              onClick={handleTest}
              loading={testing}
              disabled={!ldapEnabled}
            >
              测试连接
            </Button>
            
            <Button 
              icon={<ReloadOutlined />} 
              onClick={handleReset}
            >
              重置
            </Button>
          </Space>
        </Card>
      </Form>
    </div>
  );

  // 渲染用户管理Tab
  const renderUsersTab = () => (
    <div>
      {/* 用户统计 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="总用户数"
              value={users.length}
              prefix={<UserOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="LDAP用户"
              value={users.filter(u => u.auth_source === 'ldap').length}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="启用用户"
              value={users.filter(u => u.is_active).length}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="禁用用户"
              value={users.filter(u => !u.is_active).length}
              prefix={<LockOutlined />}
              valueStyle={{ color: '#cf1322' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 操作区 */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space>
          <Button
            type="primary"
            icon={<SyncOutlined />}
            onClick={handleSyncUsers}
            loading={syncing}
            disabled={!ldapEnabled}
          >
            同步LDAP用户
          </Button>
          
          <Button
            icon={<ReloadOutlined />}
            onClick={loadUsers}
          >
            刷新列表
          </Button>
        </Space>
      </Card>

      {/* 用户列表 */}
      <Card title="用户列表">
        <Table
          columns={userColumns}
          dataSource={users}
          rowKey="id"
          loading={loading}
          pagination={{
            total: users.length,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 个用户`
          }}
        />
      </Card>
    </div>
  );

  // 渲染同步历史Tab
  const renderSyncTab = () => (
    <div>
      {syncStatus && (
        <Alert
          message="最新同步结果"
          description={
            <div>
              <p>同步时间: {new Date(syncStatus.start_time).toLocaleString()}</p>
              <p>处理用户: {syncStatus.total_users} | 新增: {syncStatus.created_users} | 更新: {syncStatus.updated_users}</p>
            </div>
          }
          type="success"
          style={{ marginBottom: 16 }}
        />
      )}

      <Card title="同步历史">
        {syncHistory.length > 0 ? (
          <Timeline>
            {syncHistory.map((record, index) => (
              <Timeline.Item
                key={index}
                color={record.status === 'success' ? 'green' : 'red'}
                dot={record.status === 'success' ? <CheckCircleOutlined /> : <ExclamationCircleOutlined />}
              >
                <div>
                  <Text strong>{new Date(record.start_time).toLocaleString()}</Text>
                  <br />
                  <Text type="secondary">
                    处理 {record.total_users} 个用户，新增 {record.created_users}，更新 {record.updated_users}
                  </Text>
                  {record.error_message && (
                    <div>
                      <Text type="danger">错误: {record.error_message}</Text>
                    </div>
                  )}
                </div>
              </Timeline.Item>
            ))}
          </Timeline>
        ) : (
          <Empty description="暂无同步历史" />
        )}
      </Card>
    </div>
  );

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Title level={2}>
          <SettingOutlined style={{ marginRight: 8 }} />
          LDAP管理中心
        </Title>
        <Paragraph type="secondary">
          统一的LDAP配置和用户管理中心，采用只读策略确保数据安全
        </Paragraph>
      </div>

      <Spin spinning={loading}>
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane 
            tab={
              <span>
                <SettingOutlined />
                LDAP配置
              </span>
            } 
            key="config"
          >
            {renderConfigTab()}
          </TabPane>
          
          <TabPane 
            tab={
              <span>
                <UserOutlined />
                用户管理
              </span>
            } 
            key="users"
          >
            {renderUsersTab()}
          </TabPane>
          
          <TabPane 
            tab={
              <span>
                <SyncOutlined />
                同步历史
              </span>
            } 
            key="sync"
          >
            {renderSyncTab()}
          </TabPane>
        </Tabs>
      </Spin>

      {/* 用户详情模态框 */}
      <Modal
        title="用户详情"
        visible={userModalVisible}
        onCancel={() => setUserModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setUserModalVisible(false)}>
            关闭
          </Button>
        ]}
        width={600}
      >
        {selectedUser && (
          <Descriptions column={2} bordered>
            <Descriptions.Item label="用户名">{selectedUser.username}</Descriptions.Item>
            <Descriptions.Item label="邮箱">{selectedUser.email}</Descriptions.Item>
            <Descriptions.Item label="姓名">{selectedUser.name || '未设置'}</Descriptions.Item>
            <Descriptions.Item label="认证源">
              <Tag color={selectedUser.auth_source === 'ldap' ? 'blue' : 'green'}>
                {selectedUser.auth_source === 'ldap' ? 'LDAP' : '本地'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <Badge 
                status={selectedUser.is_active ? 'success' : 'error'} 
                text={selectedUser.is_active ? '启用' : '禁用'} 
              />
            </Descriptions.Item>
            <Descriptions.Item label="LDAP DN">
              {selectedUser.ldap_dn || '无'}
            </Descriptions.Item>
            <Descriptions.Item label="最后登录">
              {selectedUser.last_login ? new Date(selectedUser.last_login).toLocaleString() : '从未登录'}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">
              {new Date(selectedUser.created_at).toLocaleString()}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
};

export default AdminLDAPCenter;