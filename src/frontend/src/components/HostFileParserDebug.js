import React, { useState } from 'react';
import { 
  Card, 
  Upload, 
  Button, 
  Space, 
  Typography, 
  Alert, 
  Collapse, 
  Table, 
  Tag, 
  Spin,
  Input,
  Radio,
  Divider,
  Row,
  Col,
  message 
} from 'antd';
import { 
  UploadOutlined, 
  BugOutlined, 
  CheckCircleOutlined, 
  CloseCircleOutlined,
  WarningOutlined,
  FileTextOutlined,
  CodeOutlined
} from '@ant-design/icons';
import { saltStackAPI } from '../services/api';

const { Title, Text, Paragraph } = Typography;
const { Panel } = Collapse;
const { TextArea } = Input;

/**
 * HostFileParserDebug 组件
 * 用于调试主机配置文件（CSV/JSON/YAML/INI）的解析
 */
const HostFileParserDebug = () => {
  const [loading, setLoading] = useState(false);
  const [debugResult, setDebugResult] = useState(null);
  const [inputMode, setInputMode] = useState('file'); // 'file' | 'text'
  const [textContent, setTextContent] = useState('');
  const [textFilename, setTextFilename] = useState('hosts.csv');

  // 状态图标映射
  const statusIcons = {
    success: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
    failed: <CloseCircleOutlined style={{ color: '#ff4d4f' }} />,
    warning: <WarningOutlined style={{ color: '#faad14' }} />,
  };

  // 状态颜色映射
  const statusColors = {
    success: 'success',
    failed: 'error',
    warning: 'warning',
  };

  // 解析文件（调试模式）
  const handleDebugParse = async (content, filename) => {
    setLoading(true);
    setDebugResult(null);
    
    try {
      console.log('=== 调试解析开始 ===');
      console.log('文件名:', filename);
      console.log('内容长度:', content.length);
      console.log('内容预览:', content.substring(0, 500));
      
      const response = await saltStackAPI.parseHostFileDebug(content, filename);
      
      console.log('=== API 响应 ===');
      console.log('响应状态:', response.status);
      console.log('响应数据:', response.data);
      
      setDebugResult(response.data);
      
      if (response.data?.success) {
        message.success(`解析成功，共 ${response.data?.data?.count || 0} 个主机`);
      } else {
        message.error(response.data?.message || response.data?.error || '解析失败');
      }
    } catch (error) {
      console.error('=== 解析错误 ===');
      console.error('错误类型:', error.name);
      console.error('错误消息:', error.message);
      console.error('错误响应:', error.response?.data);
      
      setDebugResult({
        success: false,
        error: error.message,
        message: error.response?.data?.message || error.message,
        debug: error.response?.data?.debug || {
          steps: [{
            step: 1,
            name: '网络请求',
            status: 'failed',
            details: {
              error: error.message,
              response: error.response?.data,
            }
          }]
        }
      });
      
      message.error('解析请求失败: ' + error.message);
    } finally {
      setLoading(false);
    }
  };

  // 文件上传处理
  const handleFileUpload = async (file) => {
    try {
      const content = await file.text();
      await handleDebugParse(content, file.name);
    } catch (error) {
      message.error('读取文件失败: ' + error.message);
    }
    return false; // 阻止默认上传
  };

  // 文本输入解析
  const handleTextParse = () => {
    if (!textContent.trim()) {
      message.warning('请输入文件内容');
      return;
    }
    handleDebugParse(textContent, textFilename);
  };

  // 渲染解析步骤
  const renderSteps = (steps) => {
    if (!steps || steps.length === 0) return null;

    return (
      <Collapse defaultActiveKey={steps.map((_, i) => i.toString())}>
        {steps.map((step, index) => (
          <Panel
            key={index}
            header={
              <Space>
                {statusIcons[step.status]}
                <Text strong>步骤 {step.step}: {step.name}</Text>
                <Tag color={statusColors[step.status]}>{step.status}</Tag>
              </Space>
            }
          >
            <pre style={{ 
              background: '#f5f5f5', 
              padding: 12, 
              borderRadius: 4,
              maxHeight: 400,
              overflow: 'auto',
              fontSize: 12
            }}>
              {JSON.stringify(step.details, null, 2)}
            </pre>
          </Panel>
        ))}
      </Collapse>
    );
  };

  // 渲染解析结果的主机列表
  const renderHostsTable = (hosts) => {
    if (!hosts || hosts.length === 0) return null;

    const columns = [
      { title: '#', dataIndex: 'index', key: 'index', width: 50, render: (_, __, idx) => idx + 1 },
      { title: '主机', dataIndex: 'host', key: 'host' },
      { title: '端口', dataIndex: 'port', key: 'port', width: 80 },
      { title: '用户名', dataIndex: 'username', key: 'username' },
      { title: '密码', dataIndex: 'password', key: 'password', render: (v) => v ? '******' : '-' },
      { title: 'Sudo', dataIndex: 'use_sudo', key: 'use_sudo', width: 80, render: (v) => v ? <Tag color="blue">是</Tag> : <Tag>否</Tag> },
      { title: 'Minion ID', dataIndex: 'minion_id', key: 'minion_id' },
      { title: '分组', dataIndex: 'group', key: 'group' },
    ];

    return (
      <Table
        dataSource={hosts.map((h, i) => ({ ...h, key: i }))}
        columns={columns}
        size="small"
        pagination={false}
        scroll={{ x: 800 }}
      />
    );
  };

  // 示例内容
  const exampleContents = {
    csv: `host,port,username,password,use_sudo,minion_id,group
192.168.1.10,22,root,password123,false,minion-01,webservers
192.168.1.11,22,admin,password456,true,minion-02,databases`,
    json: `[
  {"host": "192.168.1.10", "port": 22, "username": "root", "password": "pass1", "use_sudo": false, "minion_id": "minion-01", "group": "webservers"},
  {"host": "192.168.1.11", "port": 22, "username": "admin", "password": "pass2", "use_sudo": true, "minion_id": "minion-02", "group": "databases"}
]`,
    yaml: `hosts:
  - host: 192.168.1.10
    port: 22
    username: root
    password: pass1
    use_sudo: false
    minion_id: minion-01
    group: webservers
  - host: 192.168.1.11
    port: 22
    username: admin
    password: pass2
    use_sudo: true
    minion_id: minion-02
    group: databases`,
    ini: `[webservers]
minion-01 ansible_host=192.168.1.10 ansible_port=22 ansible_user=root ansible_password=pass1 ansible_become=false

[databases]
minion-02 ansible_host=192.168.1.11 ansible_port=22 ansible_user=admin ansible_password=pass2 ansible_become=true`
  };

  return (
    <Card 
      title={
        <Space>
          <BugOutlined />
          <span>主机文件解析调试工具</span>
        </Space>
      }
    >
      {/* 输入方式选择 */}
      <Radio.Group 
        value={inputMode} 
        onChange={(e) => setInputMode(e.target.value)}
        style={{ marginBottom: 16 }}
      >
        <Radio.Button value="file">
          <UploadOutlined /> 上传文件
        </Radio.Button>
        <Radio.Button value="text">
          <CodeOutlined /> 直接输入
        </Radio.Button>
      </Radio.Group>

      {inputMode === 'file' ? (
        <Upload.Dragger
          beforeUpload={handleFileUpload}
          showUploadList={false}
          accept=".csv,.json,.yaml,.yml,.ini,.txt"
          disabled={loading}
        >
          <p className="ant-upload-drag-icon">
            <UploadOutlined />
          </p>
          <p className="ant-upload-text">点击或拖拽文件到此区域</p>
          <p className="ant-upload-hint">
            支持 CSV, JSON, YAML, INI (Ansible) 格式
          </p>
        </Upload.Dragger>
      ) : (
        <Space direction="vertical" style={{ width: '100%' }}>
          <Row gutter={16}>
            <Col span={18}>
              <Input
                addonBefore="文件名"
                value={textFilename}
                onChange={(e) => setTextFilename(e.target.value)}
                placeholder="hosts.csv"
              />
            </Col>
            <Col span={6}>
              <Button 
                type="primary" 
                icon={<BugOutlined />}
                onClick={handleTextParse}
                loading={loading}
                block
              >
                解析调试
              </Button>
            </Col>
          </Row>
          
          <TextArea
            value={textContent}
            onChange={(e) => setTextContent(e.target.value)}
            placeholder="在此输入文件内容..."
            rows={10}
            style={{ fontFamily: 'monospace' }}
          />
          
          <Divider>快速填充示例</Divider>
          <Space wrap>
            <Button size="small" onClick={() => { setTextContent(exampleContents.csv); setTextFilename('hosts.csv'); }}>
              CSV 示例
            </Button>
            <Button size="small" onClick={() => { setTextContent(exampleContents.json); setTextFilename('hosts.json'); }}>
              JSON 示例
            </Button>
            <Button size="small" onClick={() => { setTextContent(exampleContents.yaml); setTextFilename('hosts.yaml'); }}>
              YAML 示例
            </Button>
            <Button size="small" onClick={() => { setTextContent(exampleContents.ini); setTextFilename('inventory.ini'); }}>
              Ansible INI 示例
            </Button>
          </Space>
        </Space>
      )}

      {/* 加载状态 */}
      {loading && (
        <div style={{ textAlign: 'center', padding: 40 }}>
          <Spin size="large" tip="正在解析..." />
        </div>
      )}

      {/* 调试结果 */}
      {debugResult && !loading && (
        <div style={{ marginTop: 24 }}>
          <Divider>调试结果</Divider>
          
          {/* 总体状态 */}
          <Alert
            type={debugResult.success ? 'success' : 'error'}
            message={debugResult.success ? '解析成功' : '解析失败'}
            description={debugResult.message || debugResult.error}
            showIcon
            style={{ marginBottom: 16 }}
          />

          {/* 解析统计 */}
          {debugResult.success && debugResult.data && (
            <Card size="small" style={{ marginBottom: 16 }}>
              <Row gutter={16}>
                <Col span={8}>
                  <Text type="secondary">检测格式:</Text>
                  <Text strong style={{ marginLeft: 8 }}>{debugResult.data.format?.toUpperCase() || '未知'}</Text>
                </Col>
                <Col span={8}>
                  <Text type="secondary">解析主机数:</Text>
                  <Text strong style={{ marginLeft: 8 }}>{debugResult.data.count || 0}</Text>
                </Col>
                <Col span={8}>
                  <Text type="secondary">时间戳:</Text>
                  <Text strong style={{ marginLeft: 8 }}>{debugResult.debug?.timestamp || '-'}</Text>
                </Col>
              </Row>
            </Card>
          )}

          {/* 解析步骤详情 */}
          {debugResult.debug?.steps && (
            <>
              <Title level={5}>
                <FileTextOutlined /> 解析步骤详情
              </Title>
              {renderSteps(debugResult.debug.steps)}
            </>
          )}

          {/* 解析结果主机列表 */}
          {debugResult.success && debugResult.data?.hosts?.length > 0 && (
            <>
              <Divider />
              <Title level={5}>解析结果预览</Title>
              {renderHostsTable(debugResult.data.hosts)}
            </>
          )}

          {/* 原始响应数据 */}
          <Divider />
          <Collapse>
            <Panel header="原始 API 响应 (JSON)" key="raw">
              <pre style={{ 
                background: '#f5f5f5', 
                padding: 12, 
                borderRadius: 4,
                maxHeight: 400,
                overflow: 'auto',
                fontSize: 11
              }}>
                {JSON.stringify(debugResult, null, 2)}
              </pre>
            </Panel>
          </Collapse>
        </div>
      )}

      {/* 使用说明 */}
      <Divider />
      <Collapse>
        <Panel header="使用说明" key="help">
          <Paragraph>
            <Text strong>支持的文件格式：</Text>
          </Paragraph>
          <ul>
            <li><Text code>.csv</Text> - CSV 格式（逗号分隔）</li>
            <li><Text code>.json</Text> - JSON 格式（数组或对象）</li>
            <li><Text code>.yaml/.yml</Text> - YAML 格式</li>
            <li><Text code>.ini</Text> - Ansible Inventory 格式</li>
          </ul>
          
          <Paragraph>
            <Text strong>必需字段：</Text>
          </Paragraph>
          <ul>
            <li><Text code>host</Text> - 主机地址（IP 或主机名）</li>
          </ul>
          
          <Paragraph>
            <Text strong>可选字段：</Text>
          </Paragraph>
          <ul>
            <li><Text code>port</Text> - SSH 端口（默认 22）</li>
            <li><Text code>username</Text> - 用户名（默认 root）</li>
            <li><Text code>password</Text> - 密码</li>
            <li><Text code>use_sudo</Text> - 是否使用 sudo</li>
            <li><Text code>minion_id</Text> - Salt Minion ID</li>
            <li><Text code>group</Text> - 分组名称</li>
          </ul>
          
          <Paragraph>
            <Text strong>安全检查：</Text>
          </Paragraph>
          <ul>
            <li>文件大小限制：1MB</li>
            <li>最大行数：10000 行</li>
            <li>最大主机数：5000 个</li>
            <li>自动检测危险内容（命令注入、脚本等）</li>
          </ul>
        </Panel>
      </Collapse>
    </Card>
  );
};

export default HostFileParserDebug;
