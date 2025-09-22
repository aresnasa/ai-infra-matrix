import React, { useState } from 'react';
import {
  Card, Form, Input, Button, message, Alert, Space, Typography,
  Row, Col, Divider, Tag, Result
} from 'antd';
import {
  ExperimentOutlined, CheckCircleOutlined, ExclamationCircleOutlined,
  LoadingOutlined, KeyOutlined
} from '@ant-design/icons';
import SSHAuthConfig from '../components/SSHAuthConfig';
import { slurmAPI } from '../services/api';

const { Title, Text } = Typography;

const SSHConnectionTest = () => {
  const [form] = Form.useForm();
  const [testing, setTesting] = useState(false);
  const [testResults, setTestResults] = useState([]);
  const [initializing, setInitializing] = useState(false);
  const [initResults, setInitResults] = useState([]);
  
  console.log('SSHConnectionTest 组件版本: v3.0 - 添加主机初始化');

  const validateHostInput = (input) => {
    const errors = [];
    const lines = input.split('\n').filter(line => line.trim());
    
    lines.forEach((line, index) => {
      const trimmedLine = line.trim();
      if (!trimmedLine) return;
      
      // 检查基本格式
      const hasAt = trimmedLine.includes('@');
      const hasColon = trimmedLine.includes(':');
      
      // 用于IPv6地址检查
      const isIPv6 = trimmedLine.startsWith('[') && trimmedLine.includes(']:');
      
      if (hasAt) {
        const atIndex = trimmedLine.indexOf('@');
        const userPart = trimmedLine.substring(0, atIndex);
        const hostPart = trimmedLine.substring(atIndex + 1);
        
        // 验证用户名部分
        if (!userPart || userPart.includes(' ') || userPart.includes('\t')) {
          errors.push(`第${index + 1}行：用户名格式错误 "${userPart}"`);
        }
        
        // 验证主机部分
        if (!hostPart) {
          errors.push(`第${index + 1}行：主机名不能为空`);
        } else {
          validateHostPart(hostPart, index + 1, errors, isIPv6);
        }
      } else {
        // 没有用户名，整行都是主机部分
        validateHostPart(trimmedLine, index + 1, errors, isIPv6);
      }
    });
    
    return errors;
  };
  
  const validateHostPart = (hostPart, lineNumber, errors, isIPv6) => {
    if (isIPv6) {
      // IPv6格式验证 [address]:port
      const match = hostPart.match(/^\[(.+)\]:(\d+)$/);
      if (!match) {
        errors.push(`第${lineNumber}行：IPv6格式错误，应为 [地址]:端口 格式`);
        return;
      }
      
      const port = parseInt(match[2], 10);
      if (port < 1 || port > 65535) {
        errors.push(`第${lineNumber}行：端口号 ${port} 不在有效范围 (1-65535)`);
      }
    } else if (hostPart.includes(':')) {
      // IPv4或主机名带端口
      const lastColonIndex = hostPart.lastIndexOf(':');
      const hostName = hostPart.substring(0, lastColonIndex);
      const portPart = hostPart.substring(lastColonIndex + 1);
      
      if (!hostName) {
        errors.push(`第${lineNumber}行：主机名不能为空`);
      }
      
      if (!/^\d+$/.test(portPart)) {
        errors.push(`第${lineNumber}行：端口 "${portPart}" 必须是数字`);
      } else {
        const port = parseInt(portPart, 10);
        if (port < 1 || port > 65535) {
          errors.push(`第${lineNumber}行：端口号 ${port} 不在有效范围 (1-65535)`);
        }
      }
      
      validateHostName(hostName, lineNumber, errors);
    } else {
      // 只有主机名或IP，没有端口
      validateHostName(hostPart, lineNumber, errors);
    }
  };
  
  const validateHostName = (hostName, lineNumber, errors) => {
    if (!hostName) {
      errors.push(`第${lineNumber}行：主机名不能为空`);
      return;
    }
    
    // 检查主机名是否包含空格或制表符
    if (hostName.includes(' ') || hostName.includes('\t')) {
      errors.push(`第${lineNumber}行：主机名 "${hostName}" 不能包含空格`);
      return;
    }
    
    // IPv4地址格式检查
    const ipv4Regex = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/;
    const ipv4Match = hostName.match(ipv4Regex);
    if (ipv4Match) {
      // 验证IPv4地址的每个数字段
      const octets = ipv4Match.slice(1, 5).map(Number);
      if (octets.some(octet => octet > 255)) {
        errors.push(`第${lineNumber}行：IP地址 "${hostName}" 格式错误，每段不能大于255`);
      }
      return;
    }
    
    // 主机名格式检查（允许字母、数字、点、连字符）
    const hostnameRegex = /^[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$/;
    if (!hostnameRegex.test(hostName)) {
      errors.push(`第${lineNumber}行：主机名 "${hostName}" 格式错误，只能包含字母、数字、点和连字符`);
    }
  };

  const handleInputChange = (e) => {
    const value = e.target.value;
    const errors = validateHostInput(value);
    
    if (errors.length > 0) {
      // 显示前3个错误
      const displayErrors = errors.slice(0, 3);
      const moreCount = errors.length - 3;
      let errorMessage = displayErrors.join('\n');
      if (moreCount > 0) {
        errorMessage += `\n... 还有 ${moreCount} 个错误`;
      }
      
      form.setFields([{
        name: 'hosts',
        errors: [errorMessage]
      }]);
    } else {
      form.setFields([{
        name: 'hosts',
        errors: []
      }]);
    }
  };

  const handleQuickTest = () => {
    // 直接测试解析逻辑，不依赖表单状态
    const testInput = "root@test-ssh01:22\nroot@test-ssh02:22\nroot@test-ssh03:22";
    const lines = testInput.split('\n').filter(line => line.trim());
    
    lines.forEach(line => {
      line = line.trim();
      let user = 'root';
      let host = '';
      let port = 22;
      
      if (line.includes('@')) {
        const atIndex = line.indexOf('@');
        user = line.substring(0, atIndex).trim();
        host = line.substring(atIndex + 1).trim();
      } else {
        host = line;
      }
      
      if (host.includes(':')) {
        const lastColonIndex = host.lastIndexOf(':');
        const portPart = host.substring(lastColonIndex + 1);
        
        if (/^\d+$/.test(portPart)) {
          const parsedPort = parseInt(portPart, 10);
          if (parsedPort > 0 && parsedPort <= 65535) {
            host = host.substring(0, lastColonIndex);
            port = parsedPort;
          }
        }
      }
      
      console.log('快速测试解析结果:', {
        originalLine: line,
        user: user,
        host: host,
        port: port
      });
    });
    
    message.info('解析测试完成，请查看控制台输出');
  };

  const handleQuickFix = () => {
    const correctHosts = "test-ssh01\ntest-ssh02\ntest-ssh03";
    
    // 强制清除任何可能的缓存状态
    form.resetFields();
    
    // 设置新的值
    form.setFieldsValue({ 
      hosts: correctHosts,
      ssh_user: 'root',
      ssh_port: 22,
      password: 'rootpass123'
    });
    
    form.setFields([{
      name: 'hosts',
      errors: []
    }]);
    
    message.success('已重置表单并设置正确的测试容器配置');
  };

  const handleTest = async (values) => {
    try {
      setTesting(true);
      setTestResults([]);
      setInitResults([]);
      
      console.log('=== SSH 连接测试调试信息 ===');
      console.log('表单输入值:', values);
      console.log('原始 hosts 字段:', values.hosts);
      
      // 增强的主机列表解析逻辑
      const hosts = values.hosts
        .split('\n')
        .filter(line => line.trim())
        .map(line => {
          line = line.trim();
          let user = values.ssh_user || 'root';
          let host = '';
          let port = values.ssh_port || 22;
          
          console.log('处理行:', line);
          
          // 解析用户名@主机:端口格式 (user@host:port)
          if (line.includes('@')) {
            const atIndex = line.indexOf('@');
            user = line.substring(0, atIndex).trim();
            host = line.substring(atIndex + 1).trim();
            console.log('解析 @ 格式 - 用户:', user, '主机部分:', host);
          } else {
            // 没有用户名，整行都是主机部分
            host = line;
            console.log('无用户名格式 - 主机部分:', host);
          }
          
          // 解析主机:端口格式 (支持IPv4, IPv6, 主机名)
          if (host.includes(':')) {
            // 处理IPv6地址 [::1]:22 格式
            if (host.startsWith('[') && host.includes(']:')) {
              const match = host.match(/^\[(.+)\]:(\d+)$/);
              if (match) {
                host = match[1];
                port = parseInt(match[2], 10);
                console.log('IPv6 格式 - 主机:', host, '端口:', port);
              }
            } 
            // 处理IPv4和主机名 host:port 格式
            else {
              const lastColonIndex = host.lastIndexOf(':');
              const portPart = host.substring(lastColonIndex + 1);
              
              // 验证端口是否为数字
              if (/^\d+$/.test(portPart)) {
                const parsedPort = parseInt(portPart, 10);
                if (parsedPort > 0 && parsedPort <= 65535) {
                  host = host.substring(0, lastColonIndex);
                  port = parsedPort;
                  console.log('主机:端口 格式 - 主机:', host, '端口:', port);
                }
              }
            }
          }
          
          const result = { 
            host: host.trim(), 
            user: user.trim(), 
            port: port,
            originalInput: line
          };
          
          console.log('解析结果:', result);
          return result;
        })
        .filter(item => item.host && item.user); // 过滤无效条目

      console.log('最终主机列表:', hosts);

      if (hosts.length === 0) {
        message.warning('请至少输入一个有效的主机地址');
        return;
      }

      // 第一步：主机初始化
      console.log('开始主机初始化...');
      setInitializing(true);
      
      const hostList = hosts.map(h => h.host);
      console.log('需要初始化的主机:', hostList);
      
      try {
        const initResponse = await slurmAPI.initializeHosts(hostList);
        console.log('主机初始化响应:', initResponse);
        setInitResults(initResponse.data.results || []);
        
        if (!initResponse.data.success) {
          message.error(`主机初始化失败：${initResponse.data.failed}/${initResponse.data.total} 个主机初始化失败`);
          return;
        } else {
          message.success(`主机初始化成功：${initResponse.data.successful}/${initResponse.data.total} 个主机已就绪`);
        }
      } catch (error) {
        console.error('主机初始化错误:', error);
        message.error('主机初始化失败: ' + (error.response?.data?.error || error.message));
        return;
      } finally {
        setInitializing(false);
      }

      // 等待一小段时间让容器完全启动
      console.log('等待容器完全启动...');
      await new Promise(resolve => setTimeout(resolve, 2000));

      // 第二步：SSH连接测试
      console.log('开始SSH连接测试...');
      const results = [];
      
      // 并发测试所有主机
      const testPromises = hosts.map(async ({ host, user, port, originalInput }) => {
        const testConfig = {
          host: host,
          port: port,
          user: user,
          password: values.password || '',
          key_path: values.key_path || '',
          private_key: values.private_key || '',
        };

        console.log('发送到后端的配置:', testConfig);

        try {
          const response = await slurmAPI.testSSHConnection(testConfig);
          return {
            host: `${host}:${port}`,
            user,
            success: response.data.success,
            message: response.data.message,
            output: response.data.output,
            duration: response.data.duration,
            error: null,
            originalInput
          };
        } catch (error) {
          const errorMessage = error.response?.data?.error || error.message || '未知错误';
          let enhancedError = errorMessage;
          
          // 增强DNS解析错误提示
          if (errorMessage.includes('no such host') || errorMessage.includes('server misbehaving')) {
            if (errorMessage.includes('test-host')) {
              enhancedError = `主机名 'test-host' 不存在。请使用正确的测试容器名称：test-ssh01, test-ssh02, test-ssh03`;
            } else {
              enhancedError = `DNS解析失败：${errorMessage}。请检查主机名是否正确，或使用IP地址。如果是测试容器，请确保已正确初始化。`;
            }
          }
          
          return {
            host: `${host}:${port}`,
            user,
            success: false,
            message: '连接失败',
            output: error.response?.data?.output || '',
            duration: 0,
            error: enhancedError,
            originalInput,
            canQuickFix: errorMessage.includes('test-host')
          };
        }
      });

      const testResults = await Promise.all(testPromises);
      setTestResults(testResults);
      
      const successCount = testResults.filter(r => r.success).length;
      const totalCount = testResults.length;
      
      if (successCount === totalCount) {
        message.success(`所有 ${totalCount} 个主机连接测试成功！`);
      } else {
        message.warning(`${successCount}/${totalCount} 个主机连接成功`);
      }

    } catch (error) {
      message.error('测试过程出错: ' + error.message);
    } finally {
      setTesting(false);
      setInitializing(false);
    }
  };

  const renderInitResult = (result, index) => {
    const { Host, Success, Output, Error, Duration } = result;
    
    return (
      <Card
        key={index}
        size="small"
        style={{ marginBottom: 8 }}
        title={
          <Space>
            {Success ? (
              <CheckCircleOutlined style={{ color: '#52c41a' }} />
            ) : (
              <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
            )}
            <Text strong>{Host}</Text>
            <Tag color={Success ? 'success' : 'error'}>
              {Success ? '已就绪' : '初始化失败'}
            </Tag>
            {Duration > 0 && (
              <Tag color="blue">{Duration}ms</Tag>
            )}
          </Space>
        }
      >
        {Success ? (
          <div>
            <Text type="success">{Output || '主机初始化成功'}</Text>
          </div>
        ) : (
          <div>
            <Text type="danger">{Error || '初始化失败'}</Text>
            {Output && (
              <pre style={{
                marginTop: 8,
                fontSize: '11px',
                backgroundColor: '#fff2f0',
                padding: '8px',
                borderRadius: '4px',
                maxHeight: '120px',
                overflow: 'auto'
              }}>
                {Output}
              </pre>
            )}
          </div>
        )}
      </Card>
    );
  };

  const renderTestResult = (result, index) => {
    const { host, user, success, message: msg, output, duration, error, originalInput, canQuickFix } = result;
    
    return (
      <Card
        key={index}
        size="small"
        style={{ marginBottom: 8 }}
        title={
          <Space>
            {success ? (
              <CheckCircleOutlined style={{ color: '#52c41a' }} />
            ) : (
              <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />
            )}
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-start' }}>
              <Text strong>{user}@{host}</Text>
              {originalInput && originalInput !== `${user}@${host}` && (
                <Text style={{ fontSize: '11px', color: '#888' }}>
                  原始输入: {originalInput}
                </Text>
              )}
            </div>
            <Tag color={success ? 'success' : 'error'}>
              {success ? '成功' : '失败'}
            </Tag>
            {success && duration && (
              <Tag color="blue">{duration}ms</Tag>
            )}
          </Space>
        }
      >
        {success ? (
          <div>
            <Text type="success">{msg}</Text>
            {output && (
              <pre style={{
                marginTop: 8,
                fontSize: '11px',
                backgroundColor: '#f6f8fa',
                padding: '8px',
                borderRadius: '4px',
                maxHeight: '120px',
                overflow: 'auto'
              }}>
                {output}
              </pre>
            )}
          </div>
        ) : (
          <div>
            <Text type="danger">{error}</Text>
            {canQuickFix && (
              <div style={{ marginTop: 8 }}>
                <Button 
                  type="primary" 
                  size="small" 
                  onClick={handleQuickFix}
                >
                  🔧 快速修复：使用正确的测试容器名称
                </Button>
              </div>
            )}
            {output && (
              <pre style={{
                marginTop: 8,
                fontSize: '11px',
                backgroundColor: '#fff2f0',
                padding: '8px',
                borderRadius: '4px',
                maxHeight: '120px',
                overflow: 'auto'
              }}>
                {output}
              </pre>
            )}
          </div>
        )}
      </Card>
    );
  };

  const getOverallStatus = () => {
    if (testResults.length === 0) return null;
    
    const successCount = testResults.filter(r => r.success).length;
    const totalCount = testResults.length;
    
    if (successCount === totalCount) {
      return (
        <Alert
          message="所有主机连接测试成功！"
          description={`成功连接 ${totalCount} 台主机，可以进行后续操作。`}
          type="success"
          showIcon
          style={{ marginBottom: 16 }}
        />
      );
    } else if (successCount > 0) {
      return (
        <Alert
          message="部分主机连接成功"
          description={`${successCount}/${totalCount} 台主机连接成功，请检查失败的主机配置。`}
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
        />
      );
    } else {
      const hasQuickFixableErrors = testResults.some(r => r.canQuickFix);
      return (
        <Alert
          message="所有主机连接失败"
          description={
            <div>
              <p>请检查SSH认证配置和网络连接，确保主机地址正确且SSH服务正常运行。</p>
              {hasQuickFixableErrors && (
                <p style={{ marginTop: 8, marginBottom: 0 }}>
                  💡 检测到主机名错误，请查看下方的快速修复按钮。
                </p>
              )}
            </div>
          }
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
        />
      );
    }
  };

  return (
    <div style={{ padding: '24px' }}>
      <Row gutter={24}>
        <Col span={24}>
          <Card
            title={
              <Space>
                <ExperimentOutlined />
                <Title level={4} style={{ margin: 0 }}>SSH连接测试工具 v3.0</Title>
              </Space>
            }
          >
            <Alert
              message="SSH连接测试工具（带主机初始化）"
              description={
                <div>
                  <p>支持批量测试多个主机的SSH连接，自动初始化测试容器并验证连接。</p>
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: '16px' }}>
                    <div>
                      <p><strong>📝 基本格式：</strong></p>
                      <ul style={{ paddingLeft: '16px' }}>
                        <li><code>test-ssh01</code> - 主机名（使用默认端口）</li>
                        <li><code>192.168.1.100</code> - IP地址（使用默认端口）</li>
                        <li><code>user@host</code> - 指定用户名</li>
                      </ul>
                    </div>
                    <div>
                      <p><strong>🔌 端口格式：</strong></p>
                      <ul style={{ paddingLeft: '16px' }}>
                        <li><code>host:2222</code> - 主机名 + 端口</li>
                        <li><code>192.168.1.100:22</code> - IP + 端口</li>
                        <li><code>user@host:port</code> - 完整格式</li>
                      </ul>
                    </div>
                    <div>
                      <p><strong>🌐 IPv6支持：</strong></p>
                      <ul style={{ paddingLeft: '16px' }}>
                        <li><code>[::1]:22</code> - IPv6 + 端口</li>
                        <li><code>user@[::1]:22</code> - IPv6完整格式</li>
                      </ul>
                    </div>
                  </div>
                  <div style={{ marginTop: '16px', padding: '12px', backgroundColor: '#f0f2f5', borderRadius: '6px' }}>
                    <p><strong>🧪 可用测试容器：</strong></p>
                    <div style={{ fontFamily: 'monospace', fontSize: '13px' }}>
                      <span style={{ color: '#1890ff' }}>test-ssh01</span>, <span style={{ color: '#1890ff' }}>test-ssh02</span>, <span style={{ color: '#1890ff' }}>test-ssh03</span> 
                      <span style={{ marginLeft: '12px', color: '#666' }}>（默认用户: <code>root</code>，密码: <code>rootpass123</code>）</span>
                    </div>
                  </div>
                  <p style={{ marginTop: '12px', marginBottom: 0, fontSize: '13px', color: '#666' }}>
                    ⚡ 测试容器将自动初始化启动，支持的端口范围：1-65535
                  </p>
                </div>
              }
              type="info"
              showIcon
              style={{ marginBottom: 24 }}
            />

            <Form
              form={form}
              layout="vertical"
              onFinish={handleTest}
              initialValues={{
                hosts: "test-ssh01\ntest-ssh02\ntest-ssh03"
              }}
            >
              <Form.Item
                name="hosts"
                label="目标主机列表"
                rules={[{ required: true, message: '请输入要测试的主机列表' }]}
                validateStatus="validating"
              >
                <Input.TextArea
                  placeholder="支持多种格式，每行一个地址:&#10;test-ssh01 (主机名)&#10;192.168.1.100 (IP地址)&#10;test-ssh02:2222 (主机名:端口)&#10;192.168.1.101:22 (IP:端口)&#10;root@test-ssh03:22 (用户@主机:端口)&#10;admin@192.168.1.102 (用户@IP)&#10;[::1]:22 (IPv6)&#10;user@[2001:db8::1]:2222 (用户@IPv6)"
                  rows={8}
                  style={{ fontFamily: 'monospace' }}
                  onChange={handleInputChange}
                />
              </Form.Item>

              {/* SSH认证配置 */}
              <SSHAuthConfig
                form={form}
                initialValues={{
                  authType: 'password',
                  ssh_user: 'root',
                  ssh_port: 22
                }}
                showAdvanced={true}
                showTestConnection={false}
                size="default"
              />

              <Form.Item style={{ textAlign: 'center' }}>
                <Space size="middle">
                  <Button
                    type="default"
                    onClick={handleQuickTest}
                    size="small"
                  >
                    🧪 调试解析逻辑
                  </Button>
                  <Button
                    type="primary"
                    htmlType="submit"
                    loading={testing || initializing}
                    size="large"
                    icon={testing || initializing ? <LoadingOutlined /> : <ExperimentOutlined />}
                  >
                    {initializing ? '正在初始化主机...' : testing ? '正在测试连接...' : '开始批量测试'}
                  </Button>
                </Space>
              </Form.Item>
            </Form>

            {/* 初始化结果 */}
            {initResults.length > 0 && (
              <>
                <Divider>主机初始化结果</Divider>
                <div style={{ maxHeight: '300px', overflow: 'auto', marginBottom: 16 }}>
                  {initResults.map((result, index) => renderInitResult(result, index))}
                </div>
              </>
            )}

            {/* 测试结果 */}
            {testResults.length > 0 && (
              <>
                <Divider>SSH连接测试结果</Divider>
                {getOverallStatus()}
                <div style={{ maxHeight: '400px', overflow: 'auto' }}>
                  {testResults.map((result, index) => renderTestResult(result, index))}
                </div>
              </>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default SSHConnectionTest;