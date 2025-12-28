import React, { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Input,
  Button,
  message,
  Row,
  Col,
  Typography,
  Divider,
  Space,
  Modal,
  Tag,
  List,
  Alert,
  Spin,
} from 'antd';
import {
  UserOutlined,
  MailOutlined,
  LockOutlined,
  TeamOutlined,
  SafetyCertificateOutlined,
  QrcodeOutlined,
  CopyOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { QRCodeSVG } from 'qrcode.react';
import { authAPI, securityAPI } from '../services/api';

const { Title, Text } = Typography;

const UserProfile = () => {
  const [loading, setLoading] = useState(false);
  const [profileForm] = Form.useForm();
  const [passwordForm] = Form.useForm();
  const [userInfo, setUserInfo] = useState(null);
  const [passwordModalVisible, setPasswordModalVisible] = useState(false);
  
  // 2FA状态
  const [twoFAStatus, setTwoFAStatus] = useState({
    enabled: false,
    loading: true,
  });
  const [twoFASetupData, setTwoFASetupData] = useState(null);
  const [setupModalVisible, setSetupModalVisible] = useState(false);
  const [verifyCode, setVerifyCode] = useState('');
  const [verifying, setVerifying] = useState(false);
  const [disableModalVisible, setDisableModalVisible] = useState(false);
  const [disableCode, setDisableCode] = useState('');
  const [disabling, setDisabling] = useState(false);

  useEffect(() => {
    fetchUserProfile();
    fetch2FAStatus();
  }, []);

  const fetchUserProfile = async () => {
    try {
      setLoading(true);
      const response = await authAPI.getProfile();
      setUserInfo(response.data);
      profileForm.setFieldsValue({
        username: response.data.username,
        email: response.data.email,
      });
    } catch (error) {
      message.error('获取用户信息失败');
      console.error('Error fetching user profile:', error);
    } finally {
      setLoading(false);
    }
  };

  // 获取2FA状态
  const fetch2FAStatus = async () => {
    try {
      setTwoFAStatus(prev => ({ ...prev, loading: true }));
      const response = await securityAPI.get2FAStatus();
      // 后端返回格式: { success: true, data: { enabled: boolean, ... } }
      const data = response.data?.data || response.data;
      setTwoFAStatus({
        enabled: data?.enabled || false,
        loading: false,
      });
    } catch (error) {
      console.error('Error fetching 2FA status:', error);
      setTwoFAStatus({ enabled: false, loading: false });
    }
  };

  // 初始化2FA设置（获取二维码）
  const handleSetup2FA = async () => {
    try {
      setTwoFAStatus(prev => ({ ...prev, loading: true }));
      const response = await securityAPI.setup2FA();
      // 后端返回格式: { success: true, data: { secret, qr_code, recovery_codes, ... } }
      const data = response.data?.data || response.data;
      setTwoFASetupData({
        secret: data.secret,
        otpauth_url: data.qr_code,
        qr_code_url: data.qr_code,
        recovery_codes: data.recovery_codes || [],
        issuer: data.issuer,
        account: data.account,
      });
      setSetupModalVisible(true);
      setVerifyCode('');
    } catch (error) {
      message.error('获取2FA配置失败: ' + (error.response?.data?.error || error.message));
    } finally {
      setTwoFAStatus(prev => ({ ...prev, loading: false }));
    }
  };

  // 验证并启用2FA
  const handleEnable2FA = async () => {
    if (!verifyCode || verifyCode.length !== 6) {
      message.warning('请输入6位验证码');
      return;
    }
    try {
      setVerifying(true);
      await securityAPI.enable2FA({ code: verifyCode });
      message.success('2FA已成功启用！');
      setSetupModalVisible(false);
      setTwoFASetupData(null);
      setVerifyCode('');
      fetch2FAStatus();
    } catch (error) {
      message.error('验证失败: ' + (error.response?.data?.error || error.message));
    } finally {
      setVerifying(false);
    }
  };

  // 禁用2FA
  const handleDisable2FA = async () => {
    if (!disableCode || disableCode.length !== 6) {
      message.warning('请输入6位验证码或恢复码');
      return;
    }
    try {
      setDisabling(true);
      await securityAPI.disable2FA({ code: disableCode });
      message.success('2FA已禁用');
      setDisableModalVisible(false);
      setDisableCode('');
      fetch2FAStatus();
    } catch (error) {
      message.error('禁用失败: ' + (error.response?.data?.error || error.message));
    } finally {
      setDisabling(false);
    }
  };

  // 重新生成恢复码
  const handleRegenerateRecoveryCodes = async () => {
    try {
      const response = await securityAPI.regenerateRecoveryCodes();
      // 后端返回格式: { success: true, data: { recovery_codes: [...] } }
      const data = response.data?.data || response.data;
      if (data?.recovery_codes) {
        const recoveryCodes = data.recovery_codes;
        Modal.info({
          title: '新的恢复码',
          width: 500,
          content: (
            <div>
              <Alert
                message="重要提示"
                description="这些是您的新恢复码，旧恢复码已失效。请妥善保存！"
                type="warning"
                showIcon
                style={{ marginBottom: 16 }}
              />
              <div style={{ 
                background: '#f5f5f5', 
                padding: 16, 
                borderRadius: 8,
                fontFamily: 'monospace' 
              }}>
                {recoveryCodes.map((code, index) => (
                  <div key={index} style={{ marginBottom: 4 }}>{code}</div>
                ))}
              </div>
            </div>
          ),
          okText: '我已保存',
        });
      }
    } catch (error) {
      message.error('重新生成恢复码失败: ' + (error.response?.data?.error || error.message));
    }
  };

  // 复制到剪贴板（兼容 HTTP 和 HTTPS 环境）
  const copyToClipboard = async (text, label) => {
    // 首先尝试现代 Clipboard API
    if (navigator.clipboard && window.isSecureContext) {
      try {
        await navigator.clipboard.writeText(text);
        message.success(`${label}已复制到剪贴板`);
        return;
      } catch (err) {
        console.warn('Clipboard API failed, falling back to execCommand:', err);
      }
    }
    
    // 备用方案：使用传统的 execCommand 方式
    const textArea = document.createElement('textarea');
    textArea.value = text;
    
    // 避免滚动到底部
    textArea.style.position = 'fixed';
    textArea.style.left = '-9999px';
    textArea.style.top = '-9999px';
    textArea.style.opacity = '0';
    
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    
    try {
      const successful = document.execCommand('copy');
      if (successful) {
        message.success(`${label}已复制到剪贴板`);
      } else {
        message.error('复制失败，请手动复制');
      }
    } catch (err) {
      console.error('execCommand copy failed:', err);
      message.error('复制失败，请手动复制');
    } finally {
      document.body.removeChild(textArea);
    }
  };

  const handleUpdateProfile = async (values) => {
    try {
      setLoading(true);
      await authAPI.updateProfile(values);
      message.success('个人信息更新成功');
      fetchUserProfile();
    } catch (error) {
      message.error('更新失败');
      console.error('Error updating profile:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleChangePassword = async (values) => {
    try {
      setLoading(true);
      await authAPI.changePassword(values);
      message.success('密码修改成功');
      setPasswordModalVisible(false);
      passwordForm.resetFields();
    } catch (error) {
      message.error(error.response?.data?.error || '密码修改失败');
      console.error('Error changing password:', error);
    } finally {
      setLoading(false);
    }
  };

  const formatDate = (dateString) => {
    if (!dateString) return '未知';
    return new Date(dateString).toLocaleString('zh-CN');
  };

  const getRoleColor = (role) => {
    const roleColors = {
      'super-admin': 'red',
      'admin': 'orange',
      'user': 'blue',
      'viewer': 'green',
    };
    return roleColors[role] || 'default';
  };

  const getRoleText = (role) => {
    const roleTexts = {
      'super-admin': '超级管理员',
      'admin': '管理员',
      'user': '普通用户',
      'viewer': '查看者',
    };
    return roleTexts[role] || role;
  };

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>
        <UserOutlined /> 个人资料
      </Title>

      <Row gutter={24}>
        {/* 基本信息 */}
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <UserOutlined />
                基本信息
              </Space>
            }
            loading={loading}
          >
            <Form
              form={profileForm}
              layout="vertical"
              onFinish={handleUpdateProfile}
            >
              <Form.Item
                label="用户名"
                name="username"
                rules={[
                  { required: true, message: '请输入用户名' },
                  { min: 3, max: 50, message: '用户名长度为3-50个字符' },
                ]}
              >
                <Input prefix={<UserOutlined />} />
              </Form.Item>

              <Form.Item
                label="邮箱"
                name="email"
                rules={[
                  { required: true, message: '请输入邮箱' },
                  { type: 'email', message: '请输入有效的邮箱地址' },
                ]}
              >
                <Input prefix={<MailOutlined />} />
              </Form.Item>

              <Form.Item>
                <Space>
                  <Button type="primary" htmlType="submit" loading={loading}>
                    更新信息
                  </Button>
                  <Button
                    icon={<LockOutlined />}
                    onClick={() => setPasswordModalVisible(true)}
                  >
                    修改密码
                  </Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>
        </Col>

        {/* 账户详情 */}
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <SafetyCertificateOutlined />
                账户详情
              </Space>
            }
            loading={loading}
          >
            {userInfo && (
              <div>
                <div style={{ marginBottom: 16 }}>
                  <Text strong>用户ID：</Text>
                  <Text>{userInfo.id}</Text>
                </div>

                <div style={{ marginBottom: 16 }}>
                  <Text strong>账户状态：</Text>
                  <Tag color={userInfo.is_active ? 'success' : 'error'}>
                    {userInfo.is_active ? '活跃' : '禁用'}
                  </Tag>
                </div>

                <div style={{ marginBottom: 16 }}>
                  <Text strong>注册时间：</Text>
                  <Text>{formatDate(userInfo.created_at)}</Text>
                </div>

                <div style={{ marginBottom: 16 }}>
                  <Text strong>最后登录：</Text>
                  <Text>{formatDate(userInfo.last_login)}</Text>
                </div>

                <Divider />

                <div style={{ marginBottom: 16 }}>
                  <Text strong>角色：</Text>
                  <div style={{ marginTop: 8 }}>
                    {userInfo.roles?.map((role) => (
                      <Tag key={role.id} color={getRoleColor(role.name)}>
                        {getRoleText(role.name)}
                      </Tag>
                    )) || <Text type="secondary">暂无角色</Text>}
                  </div>
                </div>

                <div style={{ marginBottom: 16 }}>
                  <Text strong>用户组：</Text>
                  <div style={{ marginTop: 8 }}>
                    {userInfo.user_groups?.map((group) => (
                      <Tag key={group.id} icon={<TeamOutlined />}>
                        {group.name}
                      </Tag>
                    )) || <Text type="secondary">暂无用户组</Text>}
                  </div>
                </div>
              </div>
            )}
          </Card>
        </Col>
      </Row>

      {/* 二次认证 (2FA) 管理 */}
      <Row style={{ marginTop: 24 }}>
        <Col span={24}>
          <Card
            title={
              <Space>
                <SafetyCertificateOutlined />
                二次认证 (2FA) 安全设置
              </Space>
            }
          >
            <Spin spinning={twoFAStatus.loading}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 16 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                  <Text strong>2FA状态：</Text>
                  {twoFAStatus.enabled ? (
                    <Tag icon={<CheckCircleOutlined />} color="success">已启用</Tag>
                  ) : (
                    <Tag icon={<ExclamationCircleOutlined />} color="warning">未启用</Tag>
                  )}
                </div>
                <Space wrap>
                  {twoFAStatus.enabled ? (
                    <>
                      <Button
                        icon={<ReloadOutlined />}
                        onClick={handleSetup2FA}
                      >
                        重新绑定
                      </Button>
                      <Button
                        icon={<ReloadOutlined />}
                        onClick={handleRegenerateRecoveryCodes}
                      >
                        重新生成恢复码
                      </Button>
                      <Button
                        danger
                        onClick={() => setDisableModalVisible(true)}
                      >
                        禁用2FA
                      </Button>
                    </>
                  ) : (
                    <Button
                      type="primary"
                      icon={<QrcodeOutlined />}
                      onClick={handleSetup2FA}
                    >
                      启用2FA
                    </Button>
                  )}
                </Space>
              </div>

              <Divider />

              <Alert
                message="什么是二次认证 (2FA)？"
                description={
                  <div>
                    <p>二次认证是一种额外的安全层，在登录时除了密码外还需要输入动态验证码。</p>
                    <p>启用2FA后，您需要使用认证器APP（如 Google Authenticator、Microsoft Authenticator 或 Authy）扫描二维码并获取验证码。</p>
                    <p><Text type="warning"><strong>重要：</strong>关闭或修改危险命令黑名单等高危操作需要先启用2FA。</Text></p>
                  </div>
                }
                type="info"
                showIcon
              />
            </Spin>
          </Card>
        </Col>
      </Row>

      {/* 项目统计 */}
      {userInfo && (
        <Row style={{ marginTop: 24 }}>
          <Col span={24}>
            <Card title="项目统计">
              <div>
                <Text strong>拥有项目数：</Text>
                <Text style={{ fontSize: '24px', color: '#1890ff', marginLeft: 8 }}>
                  {userInfo.projects?.length || 0}
                </Text>
              </div>
              {userInfo.projects && userInfo.projects.length > 0 && (
                <div style={{ marginTop: 16 }}>
                  <Text strong>最近项目：</Text>
                  <List
                    size="small"
                    dataSource={userInfo.projects.slice(0, 5)}
                    renderItem={(project) => (
                      <List.Item>
                        <Text>{project.name}</Text>
                        <Text type="secondary" style={{ fontSize: '12px' }}>
                          {formatDate(project.updated_at)}
                        </Text>
                      </List.Item>
                    )}
                  />
                </div>
              )}
            </Card>
          </Col>
        </Row>
      )}

      {/* 修改密码模态框 */}
      <Modal
        title="修改密码"
        open={passwordModalVisible}
        onCancel={() => {
          setPasswordModalVisible(false);
          passwordForm.resetFields();
        }}
        footer={null}
      >
        <Form
          form={passwordForm}
          layout="vertical"
          onFinish={handleChangePassword}
        >
          <Form.Item
            label="当前密码"
            name="old_password"
            rules={[{ required: true, message: '请输入当前密码' }]}
          >
            <Input.Password prefix={<LockOutlined />} />
          </Form.Item>

          <Form.Item
            label="新密码"
            name="new_password"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码长度至少6个字符' },
            ]}
          >
            <Input.Password prefix={<LockOutlined />} />
          </Form.Item>

          <Form.Item
            label="确认新密码"
            name="confirm_password"
            dependencies={['new_password']}
            rules={[
              { required: true, message: '请确认新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('new_password') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password prefix={<LockOutlined />} />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading}>
                确认修改
              </Button>
              <Button
                onClick={() => {
                  setPasswordModalVisible(false);
                  passwordForm.resetFields();
                }}
              >
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 2FA设置/重新绑定模态框 */}
      <Modal
        title={
          <Space>
            <QrcodeOutlined />
            {twoFAStatus.enabled ? '重新绑定2FA' : '设置二次认证 (2FA)'}
          </Space>
        }
        open={setupModalVisible}
        onCancel={() => {
          setSetupModalVisible(false);
          setTwoFASetupData(null);
          setVerifyCode('');
        }}
        footer={null}
        width={520}
      >
        {twoFASetupData && (
          <div>
            {twoFAStatus.enabled && (
              <Alert
                message="警告：重新绑定将使旧的认证器失效"
                description="完成验证后，您之前绑定的认证器将无法使用，需要使用新的二维码重新配置。"
                type="warning"
                showIcon
                style={{ marginBottom: 16 }}
              />
            )}

            <div style={{ marginBottom: 16 }}>
              <Text strong>步骤 1：</Text>
              <Text>使用认证器APP（如 Google Authenticator、Authy）扫描二维码</Text>
            </div>

            <div style={{ 
              display: 'flex', 
              justifyContent: 'center', 
              padding: 24, 
              background: '#fafafa', 
              borderRadius: 8,
              marginBottom: 16 
            }}>
              <QRCodeSVG
                value={twoFASetupData.otpauth_url || twoFASetupData.qr_code_url}
                size={200}
                level="M"
                includeMargin
              />
            </div>

            <div style={{ marginBottom: 16 }}>
              <Text strong>步骤 2：</Text>
              <Text>或手动输入密钥</Text>
              <div style={{ 
                display: 'flex', 
                alignItems: 'center', 
                marginTop: 8,
                background: '#ffffff',
                border: '1px solid #d9d9d9',
                padding: '8px 12px',
                borderRadius: 4,
                gap: 8
              }}>
                <Text code style={{ flex: 1, letterSpacing: 2, color: '#000000', background: 'transparent' }}>
                  {twoFASetupData.secret}
                </Text>
                <Button
                  type="text"
                  icon={<CopyOutlined />}
                  onClick={() => copyToClipboard(twoFASetupData.secret, '密钥')}
                />
              </div>
            </div>

            {twoFASetupData.recovery_codes && twoFASetupData.recovery_codes.length > 0 && (
              <div style={{ marginBottom: 16 }}>
                <Alert
                  message="重要：请保存恢复码"
                  description="当您无法使用认证器时，可以使用恢复码登录。每个恢复码只能使用一次。"
                  type="warning"
                  showIcon
                  style={{ marginBottom: 8 }}
                />
                <div style={{ 
                  background: '#f5f5f5', 
                  padding: 12, 
                  borderRadius: 8,
                  fontFamily: 'monospace'
                }}>
                  {twoFASetupData.recovery_codes.map((code, index) => (
                    <div key={index} style={{ marginBottom: 4 }}>{code}</div>
                  ))}
                </div>
                <Button
                  type="link"
                  icon={<CopyOutlined />}
                  onClick={() => copyToClipboard(twoFASetupData.recovery_codes.join('\n'), '恢复码')}
                  style={{ paddingLeft: 0, marginTop: 8 }}
                >
                  复制所有恢复码
                </Button>
              </div>
            )}

            <Divider />

            <div style={{ marginBottom: 16 }}>
              <Text strong>步骤 3：</Text>
              <Text>输入认证器显示的6位验证码以完成设置</Text>
            </div>

            <Space.Compact style={{ width: '100%', marginBottom: 16 }}>
              <Input
                placeholder="请输入6位验证码"
                value={verifyCode}
                onChange={(e) => setVerifyCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                maxLength={6}
                style={{ textAlign: 'center', letterSpacing: 8, fontSize: 18 }}
                onPressEnter={handleEnable2FA}
              />
            </Space.Compact>

            <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
              <Button onClick={() => {
                setSetupModalVisible(false);
                setTwoFASetupData(null);
                setVerifyCode('');
              }}>
                取消
              </Button>
              <Button
                type="primary"
                onClick={handleEnable2FA}
                loading={verifying}
                disabled={verifyCode.length !== 6}
              >
                验证并{twoFAStatus.enabled ? '重新绑定' : '启用'}
              </Button>
            </Space>
          </div>
        )}
      </Modal>

      {/* 禁用2FA确认模态框 */}
      <Modal
        title={
          <Space>
            <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
            禁用二次认证
          </Space>
        }
        open={disableModalVisible}
        onCancel={() => {
          setDisableModalVisible(false);
          setDisableCode('');
        }}
        footer={null}
        width={420}
      >
        <Alert
          message="安全警告"
          description="禁用2FA会降低账户安全性。请确保这是您本人操作。"
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <div style={{ marginBottom: 16 }}>
          <Text>请输入当前认证器显示的验证码或恢复码以确认：</Text>
        </div>

        <Input
          placeholder="6位验证码或恢复码"
          value={disableCode}
          onChange={(e) => setDisableCode(e.target.value)}
          style={{ marginBottom: 16 }}
          onPressEnter={handleDisable2FA}
        />

        <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
          <Button onClick={() => {
            setDisableModalVisible(false);
            setDisableCode('');
          }}>
            取消
          </Button>
          <Button
            danger
            type="primary"
            onClick={handleDisable2FA}
            loading={disabling}
            disabled={!disableCode}
          >
            确认禁用
          </Button>
        </Space>
      </Modal>
    </div>
  );
};

export default UserProfile;
