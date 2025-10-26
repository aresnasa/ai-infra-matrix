import React, { useState } from 'react';
import {
  Card, Form, Input, Upload, Button, Radio, Divider, Typography,
  Alert, Space, Tooltip, message, Spin, Result, Table, Tag
} from 'antd';
import {
  KeyOutlined, UploadOutlined, EyeInvisibleOutlined, 
  EyeTwoTone, InfoCircleOutlined, CheckCircleOutlined,
  LoadingOutlined
} from '@ant-design/icons';
import { slurmAPI } from '../services/api';

const { Text, Paragraph } = Typography;
const { TextArea } = Input;

const SSHAuthConfig = ({ 
  form, 
  initialValues = {}, 
  onAuthChange,
  showAdvanced = true,
  showTestConnection = true,
  testHost = '',
  hostFieldName = 'host',  // 添加：可配置的主机名字段名
  size = 'default' 
}) => {
  const [authType, setAuthType] = useState(initialValues.authType || 'password');
  const [keyFileContent, setKeyFileContent] = useState('');
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState(null);

  // 认证类型变化处理
  const handleAuthTypeChange = (e) => {
    const type = e.target.value;
    setAuthType(type);
    setTestResult(null); // 清除测试结果
    
    // 清除相关字段
    if (type === 'password') {
      form.setFieldsValue({ key_path: '', private_key: '' });
    } else if (type === 'key') {
      form.setFieldsValue({ password: '' });
    }
    
    if (onAuthChange) {
      onAuthChange(type);
    }
  };

  // 私钥文件上传处理
  const handleKeyUpload = (file) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target.result;
      setKeyFileContent(content);
      form.setFieldsValue({ 
        private_key: content,
        key_path: file.name 
      });
      message.success('私钥文件读取成功');
    };
    reader.readAsText(file);
    return false; // 阻止默认上传行为
  };

  // 验证私钥格式
  const validatePrivateKey = (_, value) => {
    if (!value && authType === 'key') {
      return Promise.reject(new Error('请上传或输入私钥'));
    }
    if (value && (!value.includes('BEGIN') || !value.includes('PRIVATE KEY'))) {
      return Promise.reject(new Error('私钥格式不正确'));
    }
    return Promise.resolve();
  };

  // 测试SSH连接
  const testConnection = async () => {
    try {
      setTesting(true);
      setTestResult(null);
      
      const values = form.getFieldsValue();
      // 从配置的字段名获取主机名或使用 testHost
      const host = testHost || values[hostFieldName] || values.host || 'test-ssh01';
      
      // 检查是否是多个主机（支持逗号或换行符分隔）
      const hosts = host
        .split(/[,\n]/)  // 按逗号或换行符分割
        .map(h => h.trim())
        .filter(h => h);
      const isBatchTest = hosts.length > 1;
      
      const testConfig = {
        host: host,  // 保留原始主机名用于显示
        port: values.ssh_port || 22,
        user: values.ssh_user || 'root',
        password: values.password || '',
        key_path: values.key_path || '',
        private_key: values.private_key || '',
      };

      // 验证必要字段
      if (authType === 'password' && !testConfig.password) {
        message.error('请先输入SSH密码');
        return;
      }
      
      if (authType === 'key' && !testConfig.key_path && !testConfig.private_key) {
        message.error('请先配置SSH密钥');
        return;
      }

      // 批量测试或单个测试
      if (isBatchTest) {
        message.info(`正在测试 ${hosts.length} 个节点的SSH连接...`);
        
        // 构建批量测试配置
        const nodes = hosts.map(h => ({
          host: h,
          port: testConfig.port,
          user: testConfig.user,
          password: testConfig.password,
          key_path: testConfig.key_path,
          private_key: testConfig.private_key,
        }));
        
        const response = await slurmAPI.testBatchSSHConnection(nodes);
        
        if (response.data.success) {
          setTestResult({
            success: true,
            batch: true,
            total: response.data.total,
            success_count: response.data.success_count,
            fail_count: response.data.fail_count,
            results: response.data.results,
            message: `批量测试完成: ${response.data.success_count}/${response.data.total} 成功`
          });
          message.success(`批量测试完成！成功: ${response.data.success_count}, 失败: ${response.data.fail_count}`);
        } else {
          throw new Error('批量测试失败');
        }
      } else {
        // 单个主机测试
        const response = await slurmAPI.testSSHConnection(testConfig);
        
        if (response.data.success) {
          setTestResult({
            success: true,
            batch: false,
            message: response.data.message,
            output: response.data.output,
            duration: response.data.duration
          });
          message.success('SSH连接测试成功！');
        } else {
          throw new Error(response.data.error || '连接失败');
        }
      }
      
    } catch (error) {
      const errorMsg = error.response?.data?.error || error.message || '连接测试失败';
      setTestResult({
        success: false,
        error: errorMsg,
        output: error.response?.data?.output || ''
      });
      message.error(`SSH连接测试失败: ${errorMsg}`);
    } finally {
      setTesting(false);
    }
  };

  return (
    <Card 
      title={
        <Space>
          <KeyOutlined />
          SSH认证配置
          {testResult?.success && (
            <CheckCircleOutlined style={{ color: '#52c41a' }} />
          )}
        </Space>
      }
      size={size}
      style={{ marginBottom: 16 }}
    >
      <Alert
        message="SSH认证说明"
        description={
          <div>
            • 推荐使用密钥认证以提高安全性<br/>
            • 支持批量测试：主机名用逗号或换行分隔（如：test-ssh01,test-ssh02,test-ssh03 或每行一个主机）<br/>
            • 批量测试最多支持100个节点
          </div>
        }
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />

      <Form.Item
        name="authType"
        label="认证方式"
        initialValue={authType}
      >
        <Radio.Group onChange={handleAuthTypeChange} value={authType}>
          <Radio.Button value="password">
            <Space>
              <EyeInvisibleOutlined />
              密码认证
            </Space>
          </Radio.Button>
          <Radio.Button value="key">
            <Space>
              <KeyOutlined />
              密钥认证
            </Space>
          </Radio.Button>
        </Radio.Group>
      </Form.Item>

      {/* SSH用户名 - 基础必填项 */}
      <Form.Item
        name="ssh_user"
        label="SSH用户名"
        initialValue="root"
        rules={[{ required: true, message: '请输入SSH用户名' }]}
      >
        <Input 
          placeholder="请输入SSH用户名，例如: root, ubuntu, admin" 
          style={{ width: '100%' }}
        />
      </Form.Item>

      {authType === 'password' && (
        <>
          <Form.Item
            name="password"
            label="SSH密码"
            rules={[
              { required: true, message: '请输入SSH密码' },
              { min: 1, message: '密码不能为空' }
            ]}
          >
            <Input.Password
              placeholder="请输入SSH用户密码"
              iconRender={(visible) => (visible ? <EyeTwoTone /> : <EyeInvisibleOutlined />)}
            />
          </Form.Item>
          
          <Alert
            message="安全提示"
            description="密码认证相对不够安全，建议在生产环境使用密钥认证。"
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
        </>
      )}

      {authType === 'key' && (
        <>
          <Form.Item
            name="key_path"
            label="密钥文件路径"
            tooltip="服务器上私钥文件的绝对路径，例如: /root/.ssh/id_rsa"
          >
            <Input 
              placeholder="例如: /root/.ssh/id_rsa"
              addonAfter={
                <Tooltip title="或者直接上传密钥文件">
                  <InfoCircleOutlined />
                </Tooltip>
              }
            />
          </Form.Item>

          <Divider>或者</Divider>

          <Form.Item
            name="private_key"
            label="私钥内容"
            rules={[{ validator: validatePrivateKey }]}
          >
            <TextArea
              placeholder="-----BEGIN OPENSSH PRIVATE KEY-----&#10;...&#10;-----END OPENSSH PRIVATE KEY-----"
              rows={6}
              style={{ fontFamily: 'monospace', fontSize: '12px' }}
            />
          </Form.Item>

          <Form.Item label="上传私钥文件">
            <Upload
              beforeUpload={handleKeyUpload}
              showUploadList={false}
              accept=".pem,.key,*"
            >
              <Button icon={<UploadOutlined />}>
                选择私钥文件
              </Button>
            </Upload>
            {keyFileContent && (
              <Text type="success" style={{ marginLeft: 8 }}>
                ✓ 私钥已加载
              </Text>
            )}
          </Form.Item>

          <Alert
            message="私钥安全"
            description={
              <div>
                <Paragraph>
                  • 私钥将加密存储，不会明文保存<br/>
                  • 支持RSA、ECDSA、Ed25519等格式<br/>
                  • 建议使用无密码保护的密钥，或确保密钥密码为空
                </Paragraph>
              </div>
            }
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
        </>
      )}

      {/* 连接测试 */}
      {showTestConnection && (
        <>
          <Divider>连接测试</Divider>
          
          <Space direction="vertical" style={{ width: '100%' }}>
            <Button 
              type="primary" 
              onClick={testConnection}
              loading={testing}
              icon={testing ? <LoadingOutlined /> : <CheckCircleOutlined />}
            >
              {testing ? '测试连接中...' : '测试SSH连接'}
            </Button>
            
            {testResult && (
              <Alert
                message={testResult.success ? '连接测试成功' : '连接测试失败'}
                description={
                  <div>
                    {testResult.success ? (
                      <div>
                        {testResult.batch ? (
                          // 批量测试结果
                          <div>
                            <Text type="success">
                              批量测试完成: {testResult.success_count}/{testResult.total} 成功
                            </Text>
                            {testResult.fail_count > 0 && (
                              <Text type="danger"> ({testResult.fail_count} 失败)</Text>
                            )}
                            
                            {/* 详细结果表格 */}
                            <Table
                              dataSource={testResult.results}
                              size="small"
                              pagination={{ pageSize: 5 }}
                              style={{ marginTop: 12 }}
                              columns={[
                                {
                                  title: '主机',
                                  dataIndex: 'host',
                                  key: 'host',
                                  width: 150,
                                },
                                {
                                  title: '状态',
                                  dataIndex: 'success',
                                  key: 'success',
                                  width: 80,
                                  render: (success) => (
                                    <Tag color={success ? 'success' : 'error'}>
                                      {success ? '成功' : '失败'}
                                    </Tag>
                                  ),
                                },
                                {
                                  title: '耗时',
                                  dataIndex: 'duration',
                                  key: 'duration',
                                  width: 80,
                                  render: (ms) => `${ms}ms`,
                                },
                                {
                                  title: '信息',
                                  dataIndex: 'message',
                                  key: 'message',
                                  ellipsis: true,
                                  render: (msg, record) => (
                                    <Text type={record.success ? 'secondary' : 'danger'}>
                                      {msg || record.output?.substring(0, 50) || '-'}
                                    </Text>
                                  ),
                                },
                              ]}
                            />
                          </div>
                        ) : (
                          // 单个测试结果
                          <div>
                            <Text type="success">{testResult.message}</Text>
                            {testResult.duration && (
                              <Text type="secondary"> (耗时: {testResult.duration}ms)</Text>
                            )}
                            {testResult.output && (
                              <pre style={{ 
                                marginTop: 8, 
                                fontSize: '12px', 
                                backgroundColor: '#f6f8fa',
                                padding: '8px',
                                borderRadius: '4px',
                                maxHeight: '200px',
                                overflow: 'auto'
                              }}>
                                {testResult.output}
                              </pre>
                            )}
                          </div>
                        )}
                      </div>
                    ) : (
                      <div>
                        <Text type="danger">{testResult.error}</Text>
                        {testResult.output && (
                          <pre style={{ 
                            marginTop: 8, 
                            fontSize: '12px', 
                            backgroundColor: '#fff2f0',
                            padding: '8px',
                            borderRadius: '4px',
                            maxHeight: '200px',
                            overflow: 'auto'
                          }}>
                            {testResult.output}
                          </pre>
                        )}
                      </div>
                    )}
                  </div>
                }
                type={testResult.success ? 'success' : 'error'}
                showIcon
                style={{ marginBottom: 16 }}
              />
            )}
          </Space>
        </>
      )}

      {showAdvanced && (
        <>
          <Divider>高级设置</Divider>
          
          <Form.Item
            name="ssh_port"
            label="SSH端口"
            initialValue={22}
          >
            <Input placeholder="默认: 22" type="number" />
          </Form.Item>

          <Form.Item
            name="connection_timeout"
            label="连接超时(秒)"
            initialValue={10}
          >
            <Input placeholder="默认: 10" type="number" />
          </Form.Item>
        </>
      )}
    </Card>
  );
};

export default SSHAuthConfig;