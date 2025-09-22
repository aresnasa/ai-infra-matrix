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
  
  console.log('SSHConnectionTest v2.1 - 修复 test-host 问题');

  const validateHostInput = (input) => {
    const errors = [];
    const lines = input.split('\n').filter(line => line.trim());
    
    lines.forEach((line, index) => {
      const trimmedLine = line.trim();
      if (!trimmedLine) return;
      
      const lineNumber = index + 1;
      const hasAt = trimmedLine.includes('@');
      const hasColon = trimmedLine.includes(':');
      
      // 检测IPv6格式
      const isIPv6 = trimmedLine.startsWith('[') && trimmedLine.includes(']:');
      
      if (hasAt) {
        const atIndex = trimmedLine.indexOf('@');
        const userPart = trimmedLine.substring(0, atIndex);
        const hostPart = trimmedLine.substring(atIndex + 1);
        
        if (!userPart.trim()) {
          errors.push(`第${lineNumber}行: 用户名不能为空`);
        }
        
        if (!hostPart.trim()) {
          errors.push(`第${lineNumber}行: 主机地址不能为空`);
        } else {
          validateHostPart(hostPart, lineNumber, errors, isIPv6);
        }
      } else {
        validateHostPart(trimmedLine, lineNumber, errors, isIPv6);
      }
    });
    
    return errors;
  };

  const validateHostPart = (hostPart, lineNumber, errors, isIPv6) => {
    if (isIPv6) {
      // IPv6格式验证 [host]:port
      const match = hostPart.match(/^\[(.+)\]:(\d+)$/);
      if (!match) {
        errors.push(`第${lineNumber}行: IPv6格式应为 [host]:port`);
        return;
      }
      const port = parseInt(match[2], 10);
      if (port < 1 || port > 65535) {
        errors.push(`第${lineNumber}行: 端口号必须在1-65535之间`);
      }
    } else if (hostPart.includes(':')) {
      // IPv4或主机名:端口格式
      const lastColonIndex = hostPart.lastIndexOf(':');
      const hostName = hostPart.substring(0, lastColonIndex);
      const portPart = hostPart.substring(lastColonIndex + 1);
      
      if (!hostName.trim()) {
        errors.push(`第${lineNumber}行: 主机名不能为空`);
      } else {
        validateHostName(hostName, lineNumber, errors);
      }
      
      if (!portPart.trim()) {
        errors.push(`第${lineNumber}行: 端口号不能为空`);
      } else {
        const port = parseInt(portPart, 10);
        if (isNaN(port) || port < 1 || port > 65535) {
          errors.push(`第${lineNumber}行: 端口号必须是1-65535之间的数字`);
        }
      }
    } else {
      // 只有主机名，没有端口
      validateHostName(hostPart, lineNumber, errors);
    }
  };

  const validateHostName = (hostName, lineNumber, errors) => {
    if (!hostName.trim()) {
      errors.push(`第${lineNumber}行: 主机名不能为空`);
      return;
    }
    
    // IPv4格式验证
    const ipv4Regex = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/;
    const ipv4Match = hostName.match(ipv4Regex);
    
    if (ipv4Match) {
      // 验证IPv4各段
      for (let i = 1; i <= 4; i++) {
        const octet = parseInt(ipv4Match[i], 10);
        if (octet > 255) {
          errors.push(`第${lineNumber}行: IPv4地址格式不正确`);
          break;
        }
      }
    } else {
      // 主机名格式验证
      const hostnameRegex = /^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$/;
      if (!hostnameRegex.test(hostName)) {
        errors.push(`第${lineNumber}行: 主机名格式不正确`);
      }
    }
  };

  const handleQuickTest = () => {
    console.log('=== 🧪 调试解析逻辑开始 ===');
    const inputValue = form.getFieldValue('hosts') || '';
    console.log('表单输入值:', JSON.stringify(inputValue));
    
    const lines = inputValue.split('\n').filter(line => line.trim());
    console.log('解析的行数:', lines.length);
    
    lines.forEach((line, index) => {
      console.log(`行 ${index + 1}: "${line}"`);
      
      const config = parseHostLine(line.trim());
      console.log(`解析结果 ${index + 1}:`, config);
    });
    
    console.log('=== 🧪 调试解析逻辑结束 ===');
    message.success('调试输出已打印到控制台，请查看开发者工具');
  };

  const parseHostLine = (line) => {
    console.log('输入行:', JSON.stringify(line));
    
    if (!line || !line.trim()) {
      return null;
    }
    
    let user = 'root';
    let host = '';
    let port = 22;
    
    const trimmedLine = line.trim();
    console.log('清理后的行:', JSON.stringify(trimmedLine));
    
    let workingLine = trimmedLine;
    
    // 处理 user@host:port 格式
    if (workingLine.includes('@')) {
      const atIndex = workingLine.indexOf('@');
      user = workingLine.substring(0, atIndex).trim();
      workingLine = workingLine.substring(atIndex + 1).trim();
      console.log('提取的用户名:', JSON.stringify(user));
      console.log('剩余部分:', JSON.stringify(workingLine));
    }
    
    // 处理 host:port 格式
    if (workingLine.includes(':') && !workingLine.startsWith('[')) {
      const lastColonIndex = workingLine.lastIndexOf(':');
      host = workingLine.substring(0, lastColonIndex).trim();
      const portStr = workingLine.substring(lastColonIndex + 1).trim();
      const parsedPort = parseInt(portStr, 10);
      if (!isNaN(parsedPort) && parsedPort > 0 && parsedPort <= 65535) {
        port = parsedPort;
      }
      console.log('提取的主机:', JSON.stringify(host));
      console.log('提取的端口:', port);
    } else if (workingLine.startsWith('[') && workingLine.includes(']:')) {
      // IPv6 格式 [host]:port
      const match = workingLine.match(/^\[(.+)\]:(\d+)$/);
      if (match) {
        host = match[1].trim();
        const parsedPort = parseInt(match[2], 10);
        if (!isNaN(parsedPort) && parsedPort > 0 && parsedPort <= 65535) {
          port = parsedPort;
        }
      } else {
        host = workingLine;
      }
    } else {
      // 只有主机名
      host = workingLine;
      console.log('仅主机名:', JSON.stringify(host));
    }
    
    const result = { user, host, port };
    console.log('最终解析结果:', result);
    return result;
  };

  const handleTest = async () => {
    try {
      const formData = await form.validateFields();
      console.log('=== 开始SSH连接测试 ===');
      console.log('表单数据:', formData);
      console.log('主机输入原始值:', JSON.stringify(formData.hosts));
      
      const validationErrors = validateHostInput(formData.hosts);
      if (validationErrors.length > 0) {
        message.error('输入格式错误，请检查主机配置');
        setTestResults(validationErrors.map(error => ({
          host: 'validation-error',
          success: false,
          message: error,
          duration: 0
        })));
        return;
      }
      
      setTesting(true);
      setTestResults([]);
      
      // 首先初始化主机
      console.log('开始主机初始化...');
      setInitializing(true);
      try {
        const initResponse = await slurmAPI.initializeTestHosts({
          hosts: formData.hosts.split('\n').filter(line => line.trim())
        });
        console.log('主机初始化响应:', initResponse);
        setInitResults(initResponse.results || []);
      } catch (error) {
        console.error('主机初始化失败:', error);
        message.warning('主机初始化失败，但仍尝试连接测试');
        setInitResults([]);
      } finally {
        setInitializing(false);
      }
      
      const lines = formData.hosts.split('\n').filter(line => line.trim());
      console.log('处理的主机行:', lines);
      
      const results = [];
      
      for (let i = 0; i < lines.length; i++) {
        const line = lines[i].trim();
        console.log(`处理行 ${i + 1}:`, JSON.stringify(line));
        
        const config = parseHostLine(line);
        console.log(`行 ${i + 1} 解析结果:`, config);
        
        if (!config || !config.host) {
          console.warn(`跳过无效行 ${i + 1}:`, line);
          continue;
        }
        
        console.log(`准备测试连接 ${i + 1}:`, {
          host: config.host,
          port: config.port,
          user: config.user,
          password: formData.password
        });
        
        try {
          const result = await slurmAPI.testSSHConnection({
            host: config.host,
            port: config.port,
            user: config.user,
            password: formData.password
          });
          console.log(`连接结果 ${i + 1}:`, result);
          
          const processedResult = {
            host: config.host,
            port: config.port,
            user: config.user,
            success: result.success || false,
            message: result.error || result.message || (result.success ? '连接成功' : '连接失败'),
            duration: result.duration || 0,
            canQuickFix: !result.success && (
              result.error?.includes('test-host') || 
              result.error?.includes('no such host') ||
              result.message?.includes('test-host')
            )
          };
          
          results.push(processedResult);
        } catch (error) {
          console.error(`连接测试失败 ${i + 1}:`, error);
          results.push({
            host: config.host,
            port: config.port,
            user: config.user,
            success: false,
            message: `连接失败: ${error.message}`,
            duration: 0,
            canQuickFix: error.message?.includes('test-host')
          });
        }
      }
      
      console.log('所有连接测试完成:', results);
      setTestResults(results);
      
      const successCount = results.filter(r => r.success).length;
      if (successCount === results.length) {
        message.success(`全部${results.length}台主机连接成功！`);
      } else if (successCount > 0) {
        message.warning(`${successCount}/${results.length}台主机连接成功`);
      } else {
        message.error('所有主机连接失败，请检查配置');
      }
      
    } catch (error) {
      console.error('测试过程失败:', error);
      message.error('测试失败: ' + error.message);
    } finally {
      setTesting(false);
    }
  };

  const handleQuickFix = () => {
    console.log('执行快速修复...');
    form.setFieldsValue({
      hosts: 'test-ssh01\ntest-ssh02\ntest-ssh03'
    });
    setTestResults([]);
    message.success('已重置为预设的测试容器配置');
  };

  const renderInitResult = (result, index) => {
    return (
      <Alert
        key={`init-${index}`}
        message={`主机初始化: ${result.host || '未知'}`}
        description={result.message || result.error || '初始化完成'}
        type={result.success ? "success" : "warning"}
        showIcon
        style={{ marginBottom: 8 }}
      />
    );
  };

  const renderTestResult = (result, index) => {
    const getStatusIcon = () => {
      if (result.success) {
        return <CheckCircleOutlined style={{ color: '#52c41a' }} />;
      } else {
        return <ExclamationCircleOutlined style={{ color: '#ff4d4f' }} />;
      }
    };

    const getStatusColor = () => {
      return result.success ? '#f6ffed' : '#fff2f0';
    };

    return (
      <div key={index} style={{ 
        marginBottom: 16, 
        padding: 16, 
        border: `1px solid ${result.success ? '#b7eb8f' : '#ffb3b3'}`,
        borderRadius: 8,
        backgroundColor: getStatusColor()
      }}>
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 8 }}>
          {getStatusIcon()}
          <Text strong style={{ marginLeft: 8, fontSize: 16 }}>
            {result.user}@{result.host}:{result.port}
          </Text>
          <Tag color={result.success ? 'success' : 'error'} style={{ marginLeft: 12 }}>
            {result.success ? '连接成功' : '连接失败'}
          </Tag>
        </div>
        
        <div style={{ marginLeft: 24 }}>
          <Text type={result.success ? "success" : "danger"}>
            {result.message}
          </Text>
          
          {result.duration > 0 && (
            <div style={{ marginTop: 4 }}>
              <Text type="secondary">耗时: {result.duration}ms</Text>
            </div>
          )}
          
          {result.canQuickFix && (
            <div style={{ marginTop: 8 }}>
              <Button 
                size="small" 
                type="link" 
                onClick={handleQuickFix}
                style={{ padding: 0, height: 'auto' }}
              >
                🔧 点击快速修复
              </Button>
              <Text type="secondary" style={{ marginLeft: 8 }}>
                (将重置为预设测试容器)
              </Text>
            </div>
          )}
        </div>
      </div>
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
                <Title level={4} style={{ margin: 0 }}>SSH连接测试工具</Title>
              </Space>
            }
          >
            <Alert
              message="SSH连接测试工具"
              description="输入主机地址进行批量SSH连接测试。支持多种格式：主机名、IP地址、用户@主机:端口等。"
              type="info"
              showIcon
              style={{ marginBottom: 24 }}
            />
            
            <Form
              form={form}
              layout="vertical"
              initialValues={{
                hosts: 'root@test-ssh01:22\nroot@test-ssh02:22\nroot@test-ssh03:22',
                password: 'rootpass123'
              }}
            >
              <Form.Item
                label="主机列表"
                name="hosts"
                rules={[{ required: true, message: '请输入要测试的主机列表' }]}
                extra="每行一个主机，支持格式：主机名、IP地址、用户@主机:端口"
              >
                <Input.TextArea
                  rows={8}
                  placeholder={`示例格式：
test-ssh01
192.168.1.100:22
root@test-ssh02:22
user@192.168.1.101:2222
[::1]:22`}
                />
              </Form.Item>

              <SSHAuthConfig />

              <Form.Item>
                <Space>
                  <Button 
                    type="primary" 
                    onClick={handleTest} 
                    loading={testing}
                    icon={testing ? <LoadingOutlined /> : <KeyOutlined />}
                  >
                    {initializing ? '正在初始化...' : (testing ? '测试中...' : '开始批量测试')}
                  </Button>
                  
                  <Button onClick={handleQuickTest}>
                    🧪 调试解析逻辑
                  </Button>
                </Space>
              </Form.Item>
            </Form>

            {/* 初始化结果 */}
            {initResults.length > 0 && (
              <>
                <Divider>主机初始化结果</Divider>
                <div style={{ maxHeight: '200px', overflow: 'auto' }}>
                  {initResults.map((result, index) => renderInitResult(result, index))}
                </div>
              </>
            )}

            {/* 测试结果 */}
            {testResults.length > 0 && (
              <>
                <Divider>测试结果</Divider>
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