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
} from 'antd';
import {
  UserOutlined,
  MailOutlined,
  LockOutlined,
  TeamOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import { authAPI } from '../services/api';

const { Title, Text } = Typography;

const UserProfile = () => {
  const [loading, setLoading] = useState(false);
  const [profileForm] = Form.useForm();
  const [passwordForm] = Form.useForm();
  const [userInfo, setUserInfo] = useState(null);
  const [passwordModalVisible, setPasswordModalVisible] = useState(false);

  useEffect(() => {
    fetchUserProfile();
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
    </div>
  );
};

export default UserProfile;
