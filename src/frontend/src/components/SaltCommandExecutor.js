import React, { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Input,
  Select,
  Button,
  Table,
  Space,
  Tag,
  message,
  Modal,
  Descriptions,
  Typography,
  Collapse,
  Alert,
  Spin
} from 'antd';
import {
  ThunderboltOutlined,
  CodeOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  ReloadOutlined,
  EyeOutlined
} from '@ant-design/icons';
import { saltStackAPI } from '../services/api';

const { TextArea } = Input;
const { Option } = Select;
const { Text, Title } = Typography;
const { Panel } = Collapse;

/**
 * SaltStack 命令执行器组件
 * 功能：
 * 1. 执行 SaltStack 命令
 * 2. 显示执行结果
 * 3. 历史命令记录
 */
const SaltCommandExecutor = () => {
  const [form] = Form.useForm();
  const [executing, setExecuting] = useState(false);
  const [commandHistory, setCommandHistory] = useState([]);
  const [selectedCommand, setSelectedCommand] = useState(null);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [jobsLoading, setJobsLoading] = useState(false);
  const [recentJobs, setRecentJobs] = useState([]);
  const [lastExecutionResult, setLastExecutionResult] = useState(null);

  // 从 localStorage 加载历史记录
  useEffect(() => {
    try {
      const savedHistory = localStorage.getItem('saltstack_command_history');
      if (savedHistory) {
        const history = JSON.parse(savedHistory);
        // 转换时间戳为 Date 对象
        const parsedHistory = history.map(item => ({
          ...item,
          timestamp: new Date(item.timestamp)
        }));
        setCommandHistory(parsedHistory);
        console.log(`✅ 从 localStorage 加载了 ${parsedHistory.length} 条历史记录`);
      }
    } catch (error) {
      console.error('加载历史记录失败:', error);
    }
  }, []);

  // 保存历史记录到 localStorage
  const saveHistoryToLocalStorage = (history) => {
    try {
      // 只保存最近 100 条记录
      const limitedHistory = history.slice(0, 100);
      localStorage.setItem('saltstack_command_history', JSON.stringify(limitedHistory));
      console.log(`✅ 保存了 ${limitedHistory.length} 条历史记录到 localStorage`);
    } catch (error) {
      console.error('保存历史记录失败:', error);
    }
  };

  // 加载最近的 SaltStack 作业
  const loadRecentJobs = async () => {
    setJobsLoading(true);
    try {
      const response = await saltStackAPI.getJobs();
      let jobs = response.data?.data || response.data || [];
      
      // 按时间倒序排列（最新的在上面）
      if (Array.isArray(jobs) && jobs.length > 0) {
        jobs = jobs.sort((a, b) => {
          // 尝试多个可能的时间字段
          const timeA = new Date(a.start_time || a.StartTime || a.timestamp || a.Timestamp || 0);
          const timeB = new Date(b.start_time || b.StartTime || b.timestamp || b.Timestamp || 0);
          return timeB - timeA; // 降序排列，最新的在前
        });
        
        // 关联 TaskID：优先使用后端返回的 task_id
        jobs = jobs.map(job => {
          const taskId = job.task_id; // 后端返回的 task_id
          return taskId ? { ...job, taskId } : job;
        });
      }
      
      setRecentJobs(jobs);
    } catch (error) {
      console.error('加载 SaltStack 作业失败:', error);
    } finally {
      setJobsLoading(false);
    }
  };

  useEffect(() => {
    loadRecentJobs();
    const interval = setInterval(loadRecentJobs, 30000); // 每30秒刷新
    return () => clearInterval(interval);
  }, []);

  // 执行命令
  const handleExecute = async (values) => {
    setExecuting(true);
    const startTime = new Date();

    try {
      const response = await saltStackAPI.executeSaltCommand({
        target: values.target,
        function: values.function,
        arguments: values.arguments || ''
      });

      const endTime = new Date();
      const duration = endTime - startTime;

      const result = {
        id: Date.now(),
        timestamp: startTime,
        target: values.target,
        function: values.function,
        arguments: values.arguments,
        success: response.data?.success !== false,
        result: response.data?.data || response.data,
        duration,
        status: 'completed'
      };

      const newHistory = [result, ...commandHistory];
      setCommandHistory(newHistory);
      saveHistoryToLocalStorage(newHistory);
      setLastExecutionResult(result);
      message.success('命令执行成功');

      // 刷新作业列表
      loadRecentJobs();
    } catch (error) {
      const endTime = new Date();
      const duration = endTime - startTime;

      const result = {
        id: Date.now(),
        timestamp: startTime,
        target: values.target,
        function: values.function,
        arguments: values.arguments,
        success: false,
        error: error.response?.data?.error || error.message,
        duration,
        status: 'failed'
      };

      const newHistory = [result, ...commandHistory];
      setCommandHistory(newHistory);
      saveHistoryToLocalStorage(newHistory);
      setLastExecutionResult(result);
      message.error('命令执行失败: ' + (error.response?.data?.error || error.message));
    } finally {
      setExecuting(false);
    }
  };

  // 查看命令详情
  const showCommandDetail = (record) => {
    setSelectedCommand(record);
    setDetailModalVisible(true);
  };

  // 历史命令表格列定义
  const columns = [
    {
      title: '时间',
      dataIndex: 'timestamp',
      key: 'timestamp',
      width: 160,
      render: (timestamp) => new Date(timestamp).toLocaleString('zh-CN'),
    },
    {
      title: '目标',
      dataIndex: 'target',
      key: 'target',
      width: 120,
    },
    {
      title: '函数',
      dataIndex: 'function',
      key: 'function',
      width: 150,
      render: (text) => <Text code>{text}</Text>,
    },
    {
      title: '参数',
      dataIndex: 'arguments',
      key: 'arguments',
      ellipsis: true,
      render: (text) => text || <Text type="secondary">-</Text>,
    },
    {
      title: '耗时',
      dataIndex: 'duration',
      key: 'duration',
      width: 100,
      render: (duration) => `${duration}ms`,
    },
    {
      title: '状态',
      dataIndex: 'success',
      key: 'success',
      width: 100,
      render: (success, record) => (
        <Tag
          icon={success ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
          color={success ? 'success' : 'error'}
        >
          {record.status === 'completed' ? '成功' : '失败'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_, record) => (
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => showCommandDetail(record)}
        >
          查看详情
        </Button>
      ),
    },
  ];

  // 常用命令模板
  const commandTemplates = [
    { label: 'test.ping - 测试连接', value: 'test.ping' },
    { label: 'cmd.run - 执行Shell命令', value: 'cmd.run' },
    { label: 'state.apply - 应用状态', value: 'state.apply' },
    { label: 'service.status - 服务状态', value: 'service.status' },
    { label: 'service.restart - 重启服务', value: 'service.restart' },
    { label: 'pkg.install - 安装软件包', value: 'pkg.install' },
    { label: 'grains.items - 系统信息', value: 'grains.items' },
    { label: 'disk.usage - 磁盘使用', value: 'disk.usage' },
    { label: 'network.interfaces - 网络接口', value: 'network.interfaces' },
  ];

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      {/* 命令执行表单 */}
      <Card
        title={
          <Space>
            <ThunderboltOutlined />
            <span>执行 SaltStack 命令</span>
          </Space>
        }
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleExecute}
          initialValues={{ target: '*' }}
        >
          <Form.Item
            name="target"
            label="目标节点"
            rules={[{ required: true, message: '请选择目标节点' }]}
          >
            <Select placeholder="选择目标节点或输入模式">
              <Option value="*">所有节点 (*)</Option>
              <Option value="compute*">计算节点 (compute*)</Option>
              <Option value="test-ssh*">测试节点 (test-ssh*)</Option>
              <Option value="test-rocky*">Rocky节点 (test-rocky*)</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="function"
            label="Salt 函数"
            rules={[{ required: true, message: '请输入或选择 Salt 函数' }]}
          >
            <Select
              placeholder="选择常用命令或手动输入"
              showSearch
              allowClear
              options={commandTemplates}
              dropdownRender={(menu) => (
                <>
                  {menu}
                  <Alert
                    type="info"
                    message="提示"
                    description="可以直接输入自定义函数名"
                    style={{ margin: '8px' }}
                    showIcon
                  />
                </>
              )}
            />
          </Form.Item>

          <Form.Item name="arguments" label="参数">
            <TextArea
              placeholder="函数参数，例如：/bin/bash -c 'uptime'"
              rows={3}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button
                type="primary"
                htmlType="submit"
                loading={executing}
                icon={<CodeOutlined />}
              >
                执行命令
              </Button>
              <Button onClick={() => form.resetFields()}>重置</Button>
              <Button
                icon={<ReloadOutlined />}
                onClick={loadRecentJobs}
                loading={jobsLoading}
              >
                刷新作业列表
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      {/* 最新执行结果 - 立即显示 */}
      {lastExecutionResult && (
        <Card
          title={
            <Space>
              {lastExecutionResult.success ? (
                <CheckCircleOutlined style={{ color: '#52c41a' }} />
              ) : (
                <CloseCircleOutlined style={{ color: '#ff4d4f' }} />
              )}
              <span>最新执行结果</span>
              <Tag color={lastExecutionResult.success ? 'success' : 'error'}>
                {lastExecutionResult.success ? '成功' : '失败'}
              </Tag>
            </Space>
          }
          extra={
            <Space>
              <Text type="secondary">
                耗时: {lastExecutionResult.duration}ms
              </Text>
              <Button
                size="small"
                onClick={async () => {
                  const output = lastExecutionResult.result || lastExecutionResult.error;
                  const outputText = typeof output === 'string' ? output : JSON.stringify(output, null, 2);
                  
                  // 首先尝试现代 Clipboard API
                  if (navigator.clipboard && window.isSecureContext) {
                    try {
                      await navigator.clipboard.writeText(outputText);
                      message.success('已复制到剪贴板');
                      return;
                    } catch (err) {
                      console.warn('Clipboard API failed, falling back to execCommand:', err);
                    }
                  }
                  
                  // 备用方案：使用传统的 execCommand 方式
                  const textArea = document.createElement('textarea');
                  textArea.value = outputText;
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
                      message.success('已复制到剪贴板');
                    } else {
                      message.error('复制失败，请手动复制');
                    }
                  } catch (err) {
                    console.error('execCommand copy failed:', err);
                    message.error('复制失败，请手动复制');
                  } finally {
                    document.body.removeChild(textArea);
                  }
                }}
              >
                复制输出
              </Button>
              <Button
                size="small"
                onClick={() => setLastExecutionResult(null)}
              >
                关闭
              </Button>
            </Space>
          }
        >
          <Descriptions bordered size="small" column={2}>
            <Descriptions.Item label="执行时间" span={2}>
              {new Date(lastExecutionResult.timestamp).toLocaleString('zh-CN')}
            </Descriptions.Item>
            <Descriptions.Item label="目标节点">
              {lastExecutionResult.target}
            </Descriptions.Item>
            <Descriptions.Item label="Salt 函数">
              <Text code>{lastExecutionResult.function}</Text>
            </Descriptions.Item>
            {lastExecutionResult.arguments && (
              <Descriptions.Item label="参数" span={2}>
                {lastExecutionResult.arguments}
              </Descriptions.Item>
            )}
          </Descriptions>

          <div style={{ marginTop: '16px' }}>
            <Text strong>执行输出:</Text>
            {lastExecutionResult.success ? (
              <pre style={{
                background: '#f6ffed',
                border: '1px solid #b7eb8f',
                padding: '12px',
                borderRadius: '4px',
                maxHeight: '400px',
                overflow: 'auto',
                marginTop: '8px',
                fontFamily: 'monospace',
                fontSize: '13px'
              }}>
                {JSON.stringify(lastExecutionResult.result, null, 2)}
              </pre>
            ) : (
              <Alert
                type="error"
                message="执行失败"
                description={
                  <pre style={{
                    background: '#fff2f0',
                    padding: '8px',
                    borderRadius: '4px',
                    margin: '8px 0 0 0',
                    fontFamily: 'monospace',
                    fontSize: '13px'
                  }}>
                    {lastExecutionResult.error}
                  </pre>
                }
                showIcon
                style={{ marginTop: '8px' }}
              />
            )}
          </div>
        </Card>
      )}

      {/* 最近的 SaltStack 作业 */}
      <Card
        title={
          <Space>
            <ClockCircleOutlined />
            <span>最近的 SaltStack 作业</span>
          </Space>
        }
        extra={
          <Tag color="blue">{recentJobs.length} 个作业</Tag>
        }
      >
        {jobsLoading ? (
          <div style={{ textAlign: 'center', padding: '40px' }}>
            <Spin tip="加载作业列表..." />
          </div>
        ) : recentJobs.length > 0 ? (
          <Collapse accordion>
            {recentJobs.slice(0, 10).map((job, index) => (
              <Panel
                key={job.jid || index}
                header={
                  <Space>
                    <Tag color="blue">{job.function}</Tag>
                    <Text type="secondary">{job.target}</Text>
                    <Text type="secondary">
                      {job.start_time ? new Date(job.start_time).toLocaleString('zh-CN') : '-'}
                    </Text>
                  </Space>
                }
              >
                <Descriptions bordered size="small" column={2}>
                  <Descriptions.Item label="JID">{job.jid}</Descriptions.Item>
                  <Descriptions.Item label="函数">{job.function}</Descriptions.Item>
                  <Descriptions.Item label="目标">{job.target}</Descriptions.Item>
                  <Descriptions.Item label="开始时间">
                    {job.start_time ? new Date(job.start_time).toLocaleString('zh-CN') : '-'}
                  </Descriptions.Item>
                </Descriptions>
                {job.results && (
                  <div style={{ marginTop: '12px' }}>
                    <Text strong>执行结果:</Text>
                    <pre style={{
                      background: '#f5f5f5',
                      padding: '12px',
                      borderRadius: '4px',
                      maxHeight: '200px',
                      overflow: 'auto',
                      marginTop: '8px'
                    }}>
                      {JSON.stringify(job.results, null, 2)}
                    </pre>
                  </div>
                )}
              </Panel>
            ))}
          </Collapse>
        ) : (
          <Alert
            type="info"
            message="暂无作业记录"
            description="执行 SaltStack 命令后，作业记录将显示在这里"
            showIcon
          />
        )}
      </Card>

      {/* 命令执行历史 */}
      <Card
        title={
          <Space>
            <CodeOutlined />
            <span>命令执行历史</span>
          </Space>
        }
        extra={
          <Space>
            <Tag color="green">{commandHistory.length} 条记录</Tag>
            {commandHistory.length > 0 && (
              <Button
                size="small"
                danger
                onClick={() => {
                  Modal.confirm({
                    title: '确认清除历史记录',
                    content: `确定要清除所有 ${commandHistory.length} 条历史记录吗？此操作不可恢复。`,
                    okText: '确定清除',
                    cancelText: '取消',
                    okButtonProps: { danger: true },
                    onOk: () => {
                      setCommandHistory([]);
                      saveHistoryToLocalStorage([]);
                      message.success('历史记录已清除');
                    }
                  });
                }}
              >
                清除历史
              </Button>
            )}
          </Space>
        }
      >
        {commandHistory.length > 0 ? (
          <Table
            dataSource={commandHistory}
            columns={columns}
            rowKey="id"
            pagination={{ pageSize: 10 }}
            size="small"
          />
        ) : (
          <Alert
            type="info"
            message="暂无执行记录"
            description="执行命令后，历史记录将显示在这里"
            showIcon
          />
        )}
      </Card>

      {/* 命令详情模态框 */}
      <Modal
        title="命令执行详情"
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            关闭
          </Button>,
        ]}
        width={800}
      >
        {selectedCommand && (
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="执行时间" span={2}>
                {new Date(selectedCommand.timestamp).toLocaleString('zh-CN')}
              </Descriptions.Item>
              <Descriptions.Item label="目标节点">
                {selectedCommand.target}
              </Descriptions.Item>
              <Descriptions.Item label="Salt 函数">
                <Text code>{selectedCommand.function}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="参数" span={2}>
                {selectedCommand.arguments || <Text type="secondary">无参数</Text>}
              </Descriptions.Item>
              <Descriptions.Item label="耗时">
                {selectedCommand.duration}ms
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag
                  icon={selectedCommand.success ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
                  color={selectedCommand.success ? 'success' : 'error'}
                >
                  {selectedCommand.success ? '成功' : '失败'}
                </Tag>
              </Descriptions.Item>
            </Descriptions>

            <Card title="执行结果" size="small">
              {selectedCommand.success ? (
                <pre style={{
                  background: '#f5f5f5',
                  padding: '12px',
                  borderRadius: '4px',
                  maxHeight: '400px',
                  overflow: 'auto',
                  margin: 0
                }}>
                  {JSON.stringify(selectedCommand.result, null, 2)}
                </pre>
              ) : (
                <Alert
                  type="error"
                  message="执行失败"
                  description={selectedCommand.error}
                  showIcon
                />
              )}
            </Card>
          </Space>
        )}
      </Modal>
    </Space>
  );
};

export default SaltCommandExecutor;
