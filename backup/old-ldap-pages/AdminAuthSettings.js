import React, { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Input,
  Button,
  Switch,
  Select,
  message,
  Alert,
  Divider,
  Space,
  Typography,
  Spin,
  Tooltip,
  Radio,
  Row,
  Col,
  Modal
} from 'antd';
import {
  SafetyOutlined,
  ExperimentOutlined,
  SaveOutlined,
  InfoCircleOutlined,
  UserOutlined,
  LockOutlined,
  SettingOutlined,
  SyncOutlined
} from '@ant-design/icons';
import { adminAPI } from '../services/api';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TextArea } = Input;

const AdminAuthSettings = () => {
  const [form] = Form.useForm();
  const [ldapForm] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [authMode, setAuthMode] = useState('local');
  const [ldapConfig, setLdapConfig] = useState(null);
  const [testResult, setTestResult] = useState(null);
  const [testModalVisible, setTestModalVisible] = useState(false);
  const [syncResult, setSyncResult] = useState(null);
  const [syncModalVisible, setSyncModalVisible] = useState(false);
  const [syncHistory, setSyncHistory] = useState([]);

  useEffect(() => {
    loadAuthSettings();
    loadSyncHistory();
  }, []);

  const loadAuthSettings = async () => {
    setLoading(true);
    try {
      // 加载当前认证模式和LDAP配置
      const ldapResponse = await adminAPI.getLDAPConfig();
      setLdapConfig(ldapResponse.data);
      setAuthMode(ldapResponse.data.enabled ? 'ldap' : 'local');
      ldapForm.setFieldsValue(ldapResponse.data);
    } catch (error) {
      if (error.response?.status !== 404) {
        message.error('加载认证设置失败');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleAuthModeChange = (e) => {
    setAuthMode(e.target.value);
  };

  const handleSaveSettings = async () => {
    setSaving(true);
    try {
      if (authMode === 'ldap') {
        // 验证LDAP表单
        const ldapValues = await ldapForm.validateFields();
        
        // 保存LDAP配置并启用
        await adminAPI.updateLDAPConfig({
          ...ldapValues,
          enabled: true
        });
        
        message.success('LDAP认证配置已保存并启用');
      } else {
        // 禁用LDAP，使用本地认证
        if (ldapConfig) {
          await adminAPI.updateLDAPConfig({
            ...ldapConfig,
            enabled: false
          });
        }
        
        message.success('已切换到本地数据库认证');
      }
      
      await loadAuthSettings();
    } catch (error) {
      message.error('保存认证设置失败');
    } finally {
      setSaving(false);
    }
  };

  const handleTestLDAP = async () => {
    setTesting(true);
    setTestResult(null);
    
    try {
      const values = await ldapForm.validateFields();
      const response = await adminAPI.testLDAPConnection(values);
      
      setTestResult({
        success: true,
        message: response.data.message || 'LDAP连接测试成功',
        details: response.data
      });
    } catch (error) {
      setTestResult({
        success: false,
        message: error.response?.data?.error || 'LDAP连接测试失败',
        details: error.response?.data
      });
    } finally {
      setTesting(false);
    }
  };

  const showTestModal = () => {
    setTestModalVisible(true);
    setTestResult(null);
  };

  const handleSyncLDAP = async () => {
    if (!ldapConfig || !ldapConfig.enabled) {
      message.error('请先启用并保存LDAP配置');
      return;
    }

    setSyncing(true);
    setSyncResult(null);
    
    try {
      const response = await adminAPI.syncLDAPUsers();
      const syncId = response.data.sync_id;
      
      // 显示同步模态框
      setSyncModalVisible(true);
      setSyncResult({
        syncId: syncId,
        status: 'running',
        message: '正在同步LDAP用户和用户组...',
        progress: 0
      });

      // 轮询检查同步状态
      const checkSyncStatus = async () => {
        try {
          const statusResponse = await adminAPI.getLDAPSyncStatus(syncId);
          const status = statusResponse.data;
          
          setSyncResult(status);
          
          if (status.status === 'completed' || status.status === 'failed') {
            // 同步完成，刷新同步历史
            loadSyncHistory();
            clearInterval(statusInterval);
          }
        } catch (error) {
          console.error('检查同步状态失败:', error);
          clearInterval(statusInterval);
        }
      };

      // 每2秒检查一次状态
      const statusInterval = setInterval(checkSyncStatus, 2000);
      
      // 初始状态检查
      setTimeout(checkSyncStatus, 1000);
      
      message.success('LDAP同步已启动');
    } catch (error) {
      message.error('启动LDAP同步失败: ' + (error.response?.data?.error || error.message));
      setSyncResult({
        status: 'failed',
        message: error.response?.data?.error || '同步启动失败',
        error: error.message
      });
    } finally {
      setSyncing(false);
    }
  };

  const loadSyncHistory = async () => {
    try {
      const response = await adminAPI.getLDAPSyncHistory(5);
      setSyncHistory(response.data.history || []);
    } catch (error) {
      console.error('加载同步历史失败:', error);
    }
  };

  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '400px' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>
        <SettingOutlined style={{ marginRight: '8px' }} />
        认证设置
      </Title>
      
      <Alert
        message="认证模式说明"
        description="系统支持两种认证模式：本地数据库认证和LDAP认证。切换认证模式后，用户需要使用对应的认证方式登录。"
        type="info"
        showIcon
        style={{ marginBottom: '24px' }}
      />

      <Card title="认证模式选择" style={{ marginBottom: '24px' }}>
        <Radio.Group value={authMode} onChange={handleAuthModeChange} size="large">
          <Space direction="vertical" size="large">
            <Radio value="local">
              <UserOutlined style={{ marginRight: '8px' }} />
              本地数据库认证
              <Paragraph type="secondary" style={{ marginLeft: '24px', marginBottom: 0 }}>
                使用系统内置的用户数据库进行认证，适合小型团队或独立部署。
              </Paragraph>
            </Radio>
            <Radio value="ldap">
              <LockOutlined style={{ marginRight: '8px' }} />
              LDAP认证
              <Paragraph type="secondary" style={{ marginLeft: '24px', marginBottom: 0 }}>
                集成企业LDAP/Active Directory，支持统一身份认证，适合大型组织。
              </Paragraph>
            </Radio>
          </Space>
        </Radio.Group>
      </Card>

      {authMode === 'ldap' && (
        <Card 
          title="LDAP配置" 
          extra={
            <Space>
              <Button 
                icon={<ExperimentOutlined />} 
                onClick={showTestModal}
                disabled={testing}
              >
                测试连接
              </Button>
              {ldapConfig?.enabled && (
                <Button 
                  type="primary"
                  icon={<UserOutlined />} 
                  onClick={handleSyncLDAP}
                  loading={syncing}
                  disabled={testing || saving}
                >
                  同步用户
                </Button>
              )}
            </Space>
          }
        >
          <Form form={ldapForm} layout="vertical">
            <Row gutter={[16, 0]}>
              <Col span={12}>
                <Form.Item
                  label={
                    <span>
                      LDAP服务器地址
                      <Tooltip title="LDAP服务器的地址，格式：ldap://域名:端口 或 ldaps://域名:端口">
                        <InfoCircleOutlined style={{ marginLeft: '4px' }} />
                      </Tooltip>
                    </span>
                  }
                  name="server"
                  rules={[
                    { required: true, message: '请输入LDAP服务器地址' },
                    { pattern: /^ldaps?:\/\/.+/, message: '请输入有效的LDAP地址' }
                  ]}
                >
                  <Input placeholder="ldap://ldap.company.com:389" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label="端口"
                  name="port"
                  rules={[
                    { required: true, message: '请输入端口号' },
                    { type: 'number', min: 1, max: 65535, message: '端口号范围：1-65535' }
                  ]}
                >
                  <Input type="number" placeholder="389" />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={[16, 0]}>
              <Col span={12}>
                <Form.Item
                  label={
                    <span>
                      Base DN
                      <Tooltip title="LDAP搜索的根目录，例如：dc=company,dc=com">
                        <InfoCircleOutlined style={{ marginLeft: '4px' }} />
                      </Tooltip>
                    </span>
                  }
                  name="base_dn"
                  rules={[{ required: true, message: '请输入Base DN' }]}
                >
                  <Input placeholder="dc=company,dc=com" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label={
                    <span>
                      用户DN模板
                      <Tooltip title="用户登录时的DN模板，{username}会被替换为实际用户名">
                        <InfoCircleOutlined style={{ marginLeft: '4px' }} />
                      </Tooltip>
                    </span>
                  }
                  name="user_dn_template"
                  rules={[{ required: true, message: '请输入用户DN模板' }]}
                >
                  <Input placeholder="uid={username},ou=users,dc=company,dc=com" />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={[16, 0]}>
              <Col span={12}>
                <Form.Item
                  label="管理员DN"
                  name="bind_dn"
                  rules={[{ required: true, message: '请输入管理员DN' }]}
                >
                  <Input placeholder="cn=admin,dc=company,dc=com" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label="管理员密码"
                  name="bind_password"
                  rules={[{ required: true, message: '请输入管理员密码' }]}
                >
                  <Input.Password placeholder="管理员密码" />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={[16, 0]}>
              <Col span={8}>
                <Form.Item
                  label="用户名属性"
                  name="username_attribute"
                  initialValue="uid"
                  rules={[{ required: true, message: '请输入用户名属性' }]}
                >
                  <Input placeholder="uid" />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item
                  label="邮箱属性"
                  name="email_attribute"
                  initialValue="mail"
                  rules={[{ required: true, message: '请输入邮箱属性' }]}
                >
                  <Input placeholder="mail" />
                </Form.Item>
              </Col>
              <Col span={8}>
                <Form.Item
                  label="显示名称属性"
                  name="display_name_attribute"
                  initialValue="cn"
                  rules={[{ required: true, message: '请输入显示名称属性' }]}
                >
                  <Input placeholder="cn" />
                </Form.Item>
              </Col>
            </Row>

            <Row gutter={[16, 0]}>
              <Col span={12}>
                <Form.Item
                  label={
                    <span>
                      用户搜索过滤器
                      <Tooltip title="搜索用户的LDAP过滤器，{username}会被替换为实际用户名">
                        <InfoCircleOutlined style={{ marginLeft: '4px' }} />
                      </Tooltip>
                    </span>
                  }
                  name="user_filter"
                  initialValue="(&(objectClass=inetOrgPerson)(uid={username}))"
                  rules={[{ required: true, message: '请输入用户搜索过滤器' }]}
                >
                  <Input placeholder="(&(objectClass=inetOrgPerson)(uid={username}))" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label="用户搜索基础DN"
                  name="user_search_base"
                  initialValue="ou=users"
                >
                  <Input placeholder="ou=users" />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item
              label="连接超时(秒)"
              name="timeout"
              initialValue={30}
              rules={[
                { required: true, message: '请输入连接超时时间' },
                { type: 'number', min: 1, max: 300, message: '超时时间范围：1-300秒' }
              ]}
            >
              <Input type="number" placeholder="30" />
            </Form.Item>

            <Form.Item label="启用TLS" name="enable_tls" valuePropName="checked">
              <Switch />
            </Form.Item>

            <Form.Item label="跳过TLS验证" name="skip_tls_verify" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Form>
        </Card>
      )}

      {authMode === 'ldap' && ldapConfig?.enabled && (
        <Card 
          title="同步历史" 
          style={{ marginTop: '24px' }}
          extra={
            <Button 
              size="small"
              onClick={loadSyncHistory}
            >
              刷新
            </Button>
          }
        >
          {syncHistory.length > 0 ? (
            <div>
              {syncHistory.map((sync, index) => (
                <div key={sync.id || index} style={{ 
                  padding: '12px', 
                  border: '1px solid #f0f0f0', 
                  borderRadius: '6px',
                  marginBottom: '8px',
                  backgroundColor: sync.status === 'completed' ? '#f6ffed' : 
                                   sync.status === 'failed' ? '#fff2f0' : '#f0f0f0'
                }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                      <Text strong>
                        {sync.status === 'completed' && '✅ 同步完成'}
                        {sync.status === 'failed' && '❌ 同步失败'}
                        {sync.status === 'running' && '🔄 同步中'}
                      </Text>
                      <div style={{ marginTop: '4px' }}>
                        <Text type="secondary" style={{ fontSize: '12px' }}>
                          {new Date(sync.start_time).toLocaleString()}
                        </Text>
                        {sync.duration && (
                          <Text type="secondary" style={{ fontSize: '12px', marginLeft: '12px' }}>
                            耗时: {Math.round(sync.duration / 1000000000)}秒
                          </Text>
                        )}
                      </div>
                    </div>
                    {sync.result && (
                      <div style={{ textAlign: 'right' }}>
                        <div style={{ fontSize: '12px' }}>
                          <Text type="secondary">用户: </Text>
                          <Text>{sync.result.users_created + sync.result.users_updated}</Text>
                          <Text type="secondary" style={{ marginLeft: '8px' }}>组: </Text>
                          <Text>{sync.result.groups_created + sync.result.groups_updated}</Text>
                        </div>
                      </div>
                    )}
                  </div>
                  {sync.message && (
                    <div style={{ marginTop: '8px' }}>
                      <Text style={{ fontSize: '12px' }}>{sync.message}</Text>
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div style={{ textAlign: 'center', padding: '24px' }}>
              <Text type="secondary">暂无同步记录</Text>
            </div>
          )}
        </Card>
      )}

      <Card style={{ marginTop: '24px' }}>
        <Space>
          <Button 
            type="primary" 
            icon={<SaveOutlined />} 
            onClick={handleSaveSettings}
            loading={saving}
            size="large"
          >
            保存设置
          </Button>
          <Button onClick={loadAuthSettings} disabled={saving}>
            重置
          </Button>
        </Space>
      </Card>

      {/* LDAP测试连接模态框 */}
      <Modal
        title="LDAP连接测试"
        open={testModalVisible}
        onCancel={() => setTestModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setTestModalVisible(false)}>
            关闭
          </Button>,
          <Button 
            key="test" 
            type="primary" 
            icon={<ExperimentOutlined />}
            onClick={handleTestLDAP}
            loading={testing}
          >
            测试连接
          </Button>
        ]}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Text>点击"测试连接"按钮验证LDAP服务器配置是否正确。</Text>
          
          {testing && (
            <div style={{ textAlign: 'center', padding: '20px' }}>
              <Spin />
              <div style={{ marginTop: '8px' }}>正在测试连接...</div>
            </div>
          )}
          
          {testResult && (
            <Alert
              type={testResult.success ? 'success' : 'error'}
              message={testResult.message}
              description={testResult.details && (
                <div>
                  {testResult.details.server_info && (
                    <div>服务器信息: {testResult.details.server_info}</div>
                  )}
                  {testResult.details.bind_result && (
                    <div>绑定结果: {testResult.details.bind_result}</div>
                  )}
                  {testResult.details.search_result && (
                    <div>搜索结果: {testResult.details.search_result}</div>
                  )}
                </div>
              )}
              showIcon
            />
          )}
        </Space>
      </Modal>

      {/* LDAP同步状态模态框 */}
      <Modal
        title="LDAP用户同步"
        open={syncModalVisible}
        onCancel={() => setSyncModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setSyncModalVisible(false)}>
            关闭
          </Button>
        ]}
        width={600}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          {syncResult && (
            <div>
              {syncResult.status === 'running' && (
                <div style={{ textAlign: 'center', padding: '20px' }}>
                  <Spin size="large" />
                  <div style={{ marginTop: '16px' }}>
                    <Text strong>{syncResult.message}</Text>
                  </div>
                  {syncResult.progress > 0 && (
                    <div style={{ marginTop: '8px' }}>
                      <Text type="secondary">进度: {syncResult.progress.toFixed(1)}%</Text>
                    </div>
                  )}
                </div>
              )}
              
              {syncResult.status === 'completed' && (
                <Alert
                  type="success"
                  message="同步完成"
                  description={
                    <div>
                      <div>同步ID: {syncResult.id}</div>
                      <div>开始时间: {new Date(syncResult.start_time).toLocaleString()}</div>
                      {syncResult.end_time && (
                        <div>结束时间: {new Date(syncResult.end_time).toLocaleString()}</div>
                      )}
                      {syncResult.duration && (
                        <div>耗时: {Math.round(syncResult.duration / 1000000000)}秒</div>
                      )}
                      {syncResult.result && (
                        <div style={{ marginTop: '12px' }}>
                          <Text strong>同步结果:</Text>
                          <ul style={{ marginTop: '8px', paddingLeft: '20px' }}>
                            <li>创建用户: {syncResult.result.users_created}</li>
                            <li>更新用户: {syncResult.result.users_updated}</li>
                            <li>创建用户组: {syncResult.result.groups_created}</li>
                            <li>更新用户组: {syncResult.result.groups_updated}</li>
                            <li>分配角色: {syncResult.result.roles_assigned}</li>
                            <li>总用户数: {syncResult.result.total_users}</li>
                            <li>总用户组数: {syncResult.result.total_groups}</li>
                          </ul>
                          {syncResult.result.errors && syncResult.result.errors.length > 0 && (
                            <div style={{ marginTop: '12px' }}>
                              <Text type="danger">错误信息:</Text>
                              <ul style={{ marginTop: '8px', paddingLeft: '20px' }}>
                                {syncResult.result.errors.map((error, index) => (
                                  <li key={index} style={{ color: '#ff4d4f' }}>{error}</li>
                                ))}
                              </ul>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  }
                  showIcon
                />
              )}
              
              {syncResult.status === 'failed' && (
                <Alert
                  type="error"
                  message="同步失败"
                  description={
                    <div>
                      <div>同步ID: {syncResult.id}</div>
                      <div>开始时间: {new Date(syncResult.start_time).toLocaleString()}</div>
                      {syncResult.end_time && (
                        <div>结束时间: {new Date(syncResult.end_time).toLocaleString()}</div>
                      )}
                      <div style={{ marginTop: '12px' }}>
                        <Text type="danger">错误信息: {syncResult.error || syncResult.message}</Text>
                      </div>
                    </div>
                  }
                  showIcon
                />
              )}
            </div>
          )}
        </Space>
      </Modal>
    </div>
  );
};

export default AdminAuthSettings;
