import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Card, Row, Col, Statistic, Table, Tag, Space, Alert, Spin, Button, Layout, Typography, List, Progress, Descriptions, Badge, Tabs, Modal, Form, Input, Select, message, Skeleton } from 'antd';
import { 
  CheckCircleOutlined, 
  ExclamationCircleOutlined, 
  ClockCircleOutlined, 
  DesktopOutlined,
  SettingOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
  DatabaseOutlined,
  ApiOutlined
} from '@ant-design/icons';
import { saltStackAPI, aiAPI } from '../services/api';

const { Content } = Layout;
const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;

const { TextArea } = Input;
const { Option } = Select;

// 骨架屏组件
const StatisticSkeleton = ({ title, icon }) => (
  <Card>
    <div style={{ display: 'flex', alignItems: 'center' }}>
      {icon}
      <div style={{ marginLeft: 8, flex: 1 }}>
        <div style={{ fontSize: '14px', color: '#999', marginBottom: 4 }}>{title}</div>
        <Skeleton.Input style={{ width: 60, height: 24 }} active />
      </div>
    </div>
  </Card>
);

const SaltStackDashboard = () => {
  // 页面状态管理
  const [pageLoaded, setPageLoaded] = useState(false);
  
  // 数据状态 - 分别管理loading状态
  const [status, setStatus] = useState(null);
  const [minions, setMinions] = useState([]);
  const [jobs, setJobs] = useState([]);
  
  // 加载状态 - 分别管理每个数据块的加载状态
  const [statusLoading, setStatusLoading] = useState(false);
  const [minionsLoading, setMinionsLoading] = useState(false);
  const [jobsLoading, setJobsLoading] = useState(false);
  
  // 全局状态
  const [demo, setDemo] = useState(false);
  const [error, setError] = useState(null);
  
  // 自定义执行弹窗
  const [execVisible, setExecVisible] = useState(false);
  const [execForm] = Form.useForm();
  const [execRunning, setExecRunning] = useState(false);
  const [execOpId, setExecOpId] = useState('');
  const [execEvents, setExecEvents] = useState([]);
  const sseRef = useRef(null);
  
  // 配置管理弹窗
  const [configVisible, setConfigVisible] = useState(false);
  const [configForm] = Form.useForm();
  const [configTemplates] = useState([
    { id: 'nginx', name: 'Nginx 配置', desc: '安装和配置 Nginx Web 服务器' },
    { id: 'mysql', name: 'MySQL 配置', desc: '安装和配置 MySQL 数据库' },
    { id: 'docker', name: 'Docker 配置', desc: '安装和配置 Docker 容器引擎' },
    { id: 'firewall', name: '防火墙配置', desc: '配置系统防火墙规则' },
    { id: 'user', name: '用户管理', desc: '添加、删除和管理系统用户' },
  ]);

  const loadStatus = async () => {
    setStatusLoading(true);
    try {
      const response = await saltStackAPI.getStatus();
      setStatus(response.data?.data);
      setDemo(Boolean(response.data?.data?.demo));
      setError(null);
    } catch (e) {
      console.error('加载SaltStack状态失败', e);
      setError(e);
    } finally {
      setStatusLoading(false);
    }
  };

  const loadMinions = async () => {
    setMinionsLoading(true);
    try {
      const response = await saltStackAPI.getMinions();
      setMinions(response.data?.data || []);
      setDemo(prev => prev || Boolean(response.data?.demo));
    } catch (e) {
      console.error('加载SaltStack Minions失败', e);
    } finally {
      setMinionsLoading(false);
    }
  };

  const loadJobs = async () => {
    setJobsLoading(true);
    try {
      const response = await saltStackAPI.getJobs(10);
      setJobs(response.data?.data || []);
      setDemo(prev => prev || Boolean(response.data?.demo));
    } catch (e) {
      console.error('加载SaltStack Jobs失败', e);
    } finally {
      setJobsLoading(false);
    }
  };

  const loadAllData = async () => {
    // 并行加载所有数据，但不阻塞页面渲染
    await Promise.all([loadStatus(), loadMinions(), loadJobs()]);
  };

  // 页面初始化效果 - 立即显示静态内容
  useEffect(() => {
    // 标记页面已加载，显示静态内容
    setPageLoaded(true);
    
    // 异步加载数据（非阻塞）
    setTimeout(() => {
      loadAllData();
    }, 100); // 延迟100ms让静态内容先渲染
    
    // 设置定时刷新（60秒）
    const interval = setInterval(loadAllData, 60000);
    return () => clearInterval(interval);
  }, []);

  // 关闭SSE
  const closeSSE = () => {
    if (sseRef.current) {
      try { sseRef.current.close?.(); } catch {}
      sseRef.current = null;
    }
  };

  useEffect(() => {
    return () => closeSSE();
  }, []);

  const validateClientSide = (language, code) => {
    if (!code || !code.trim()) return '请输入要执行的代码';
    if (code.length > 20000) return '代码过长，最大20000字符';
    // 简单引号平衡检查
    let single = 0, dbl = 0;
    for (let i = 0; i < code.length; i++) {
      const ch = code[i];
      if (ch === '\'') single ^= 1; else if (ch === '"') dbl ^= 1;
    }
    if (single || dbl) return '引号不平衡，请检查代码';
    if (language === 'python') {
      const lines = code.split('\n');
      for (const ln of lines) {
        if (ln.startsWith('\t') && ln.trimStart().startsWith(' ')) return 'Python 缩进混用制表符与空格';
      }
    }
    return '';
  };

  const openExecModal = () => {
    setExecVisible(true);
    setExecEvents([]);
    setExecOpId('');
    execForm.setFieldsValue({ target: '*', language: 'bash', code: '# 例如: echo Hello\necho $(hostname)', timeout: 120 });
  };

  const handleSuggest = async () => {
    try {
      const values = await execForm.validateFields(['language', 'code']);
      const lang = values.language;
      const prompt = `为 Salt 下发的${lang}脚本提供补全建议，仅给出代码片段，不要解释。`;
      await aiAPI.quickChat(prompt, 'salt-exec-suggest'); // 预留：后端应返回异步消息ID，这里仅调用以示占位
      message.info('已发送智能补全请求（占位），后端实现后将展示建议');
    } catch (e) {
      // 忽略
    }
  };

  const startSSE = (opId) => {
    closeSSE();
    const url = saltStackAPI.streamProgressUrl(opId);
    const es = new EventSource(url, { withCredentials: false });
    sseRef.current = es;
    es.onmessage = (evt) => {
      try {
        const data = JSON.parse(evt.data);
        console.log('[SSE事件]', data.type, data);
        setExecEvents((prev) => [...prev, data]);
        
        // 检查是否执行完成 - 只在收到 complete 或 error 事件时停止
        if (data.type === 'complete' || data.type === 'error') {
          console.log('[SSE] 收到完成事件，准备停止');
          // 延迟一点点以确保UI更新
          setTimeout(() => {
            console.log('[SSE] 设置 execRunning = false');
            setExecRunning(false);
            closeSSE();
          }, 300);
        }
      } catch (err) {
        console.error('[SSE] 解析消息失败:', err);
      }
    };
    es.onerror = (err) => {
      console.error('[SSE] 连接错误:', err);
      // 自动关闭，避免内存泄漏
      closeSSE();
      setExecRunning(false);
    };
  };

  const handleExecute = async () => {
    try {
      const values = await execForm.validateFields();
      const err = validateClientSide(values.language, values.code);
      if (err) {
        message.error(err);
        return;
      }
      setExecRunning(true);
      setExecEvents([]);
      const resp = await saltStackAPI.executeCustomAsync({
        target: values.target,
        language: values.language,
        code: values.code,
        timeout: values.timeout || 120,
      });
      const opId = resp.data?.opId || resp.data?.data?.opId || resp.data?.id || resp.data?.op_id;
      if (!opId) {
        message.error('未返回操作ID');
        setExecRunning(false);
        return;
      }
      setExecOpId(opId);
      startSSE(opId);
    } catch (e) {
      message.error('提交执行失败: ' + (e?.response?.data?.error || e.message));
      setExecRunning(false);
    }
  };

  const execFooter = (
    <Space>
      <Button onClick={() => setExecVisible(false)} disabled={execRunning}>关闭</Button>
      <Button onClick={handleSuggest} disabled={execRunning}>智能补全（预留）</Button>
      <Button type="primary" onClick={handleExecute} loading={execRunning}>执行</Button>
    </Space>
  );

  const getStatusColor = (state) => {
    switch (state?.toLowerCase()) {
      case 'up': case 'online': case 'running': return 'success';
      case 'down': case 'offline': case 'stopped': return 'error';
      case 'pending': case 'starting': return 'processing';
      default: return 'default';
    }
  };

  const getJobStatusColor = (status) => {
    switch (status?.toLowerCase()) {
      case 'success': case 'completed': return 'green';
      case 'failed': case 'error': return 'red';
      case 'running': case 'in_progress': return 'blue';
      case 'pending': case 'queued': return 'orange';
      default: return 'default';
    }
  };

  // 如果页面还未初始化，显示简单加载提示
  if (!pageLoaded) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" />
        <div style={{ marginTop: 16 }}>初始化SaltStack界面...</div>
      </div>
    );
  }

  return (
    <Layout style={{ minHeight: '100vh', background: '#f0f2f5' }}>
      <Content style={{ padding: 24 }}>
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div>
            <Title level={2}>
              <ThunderboltOutlined style={{ marginRight: 8, color: '#1890ff' }} />
              SaltStack 配置管理
            </Title>
            <Paragraph type="secondary">
              基于 SaltStack 的基础设施自动化配置管理系统
            </Paragraph>
          </div>

          {error && (
            <Alert 
              type="error" 
              showIcon 
              message="无法连接到SaltStack"
              description={
                <Space>
                  <span>请确认SaltStack服务正在运行且后端API可达。</span>
                  <Button size="small" onClick={loadAllData}>重试</Button>
                </Space>
              }
            />
          )}

          {demo && (
            <Alert 
              type="info" 
              showIcon 
              message="演示模式" 
              description="显示SaltStack演示数据"
            />
          )}

          {/* 数据加载进度提示 */}
          {(statusLoading || minionsLoading || jobsLoading) && (
            <Alert 
              type="info" 
              showIcon 
              message="正在加载数据" 
              description={
                <Space>
                  <span>
                    状态数据: {statusLoading ? '加载中...' : '✓'} | 
                    Minions: {minionsLoading ? '加载中...' : '✓'} | 
                    作业历史: {jobsLoading ? '加载中...' : '✓'}
                  </span>
                </Space>
              }
            />
          )}

          {/* 状态概览 - 每个卡片独立加载 */}
          <Row gutter={16}>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="Master状态" 
                  value={status?.master_status || (statusLoading ? '加载中...' : '未知')} 
                  prefix={<SettingOutlined />}
                  valueStyle={{ 
                    color: statusLoading ? '#999' : (status?.master_status === 'running' ? '#3f8600' : '#cf1322') 
                  }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="在线Minions" 
                  value={status?.minions_up || (statusLoading ? '...' : 0)} 
                  prefix={<DesktopOutlined />}
                  valueStyle={{ color: statusLoading ? '#999' : '#3f8600' }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="离线Minions" 
                  value={status?.minions_down || (statusLoading ? '...' : 0)} 
                  prefix={<ExclamationCircleOutlined />}
                  valueStyle={{ color: statusLoading ? '#999' : '#cf1322' }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic 
                  title="API状态" 
                  value={status?.api_status || (statusLoading ? '检测中...' : '未知')} 
                  prefix={<ApiOutlined />}
                  valueStyle={{ 
                    color: statusLoading ? '#999' : (status?.api_status === 'running' ? '#3f8600' : '#cf1322') 
                  }}
                  loading={statusLoading}
                />
              </Card>
            </Col>
          </Row>

          {/* 详细信息选项卡 */}
          <Card>
            <Tabs defaultActiveKey="overview" size="large">
              <TabPane tab="系统概览" key="overview" icon={<DatabaseOutlined />}>
                <Row gutter={16}>
                  <Col span={12}>
                    <Card title="Master信息" size="small" loading={statusLoading}>
                      <Descriptions size="small" column={1}>
                        <Descriptions.Item label="版本">
                          {status?.salt_version || (statusLoading ? '加载中...' : '未知')}
                        </Descriptions.Item>
                        <Descriptions.Item label="启动时间">
                          {status?.uptime || (statusLoading ? '获取中...' : '未知')}
                        </Descriptions.Item>
                        <Descriptions.Item label="配置文件">
                          {status?.config_file || '/etc/salt/master'}
                        </Descriptions.Item>
                        <Descriptions.Item label="日志级别">
                          <Tag color="blue">{status?.log_level || 'info'}</Tag>
                        </Descriptions.Item>
                      </Descriptions>
                    </Card>
                  </Col>
                  <Col span={12}>
                    <Card title="性能指标" size="small" loading={statusLoading}>
                      {statusLoading ? (
                        <div style={{ textAlign: 'center', padding: '20px 0' }}>
                          <Spin size="small" />
                          <div style={{ marginTop: 8 }}>正在获取性能数据...</div>
                        </div>
                      ) : (
                        <Space direction="vertical" style={{ width: '100%' }}>
                          <div>
                            <Text>CPU使用率</Text>
                            <Progress 
                              percent={status?.cpu_usage || 0} 
                              size="small" 
                              status={status?.cpu_usage > 80 ? 'exception' : 'active'}
                            />
                          </div>
                          <div>
                            <Text>内存使用率</Text>
                            <Progress 
                              percent={status?.memory_usage || 0} 
                              size="small" 
                              status={status?.memory_usage > 85 ? 'exception' : 'active'}
                            />
                          </div>
                          <div>
                            <Text>活跃连接数</Text>
                            <Progress 
                              percent={Math.min((status?.active_connections || 0) / 100 * 100, 100)} 
                              size="small"
                              showInfo={false}
                            />
                            <Text type="secondary"> {status?.active_connections || 0}/100</Text>
                          </div>
                        </Space>
                      )}
                    </Card>
                  </Col>
                </Row>
              </TabPane>

              <TabPane tab="Minions管理" key="minions" icon={<DesktopOutlined />}>
                {minionsLoading ? (
                  <div style={{ textAlign: 'center', padding: '60px 0' }}>
                    <Spin size="large" />
                    <div style={{ marginTop: 16 }}>正在加载Minions数据...</div>
                  </div>
                ) : (
                  <>
                    <List
                      grid={{ gutter: 16, column: 2 }}
                      dataSource={minions}
                      renderItem={minion => (
                        <List.Item>
                          <Card 
                            size="small" 
                            title={
                              <Space>
                                <Badge status={getStatusColor(minion.status)} />
                                {minion.id || minion.name}
                              </Space>
                            }
                            extra={
                              <Tag color={getStatusColor(minion.status)}>
                                {minion.status || '未知'}
                              </Tag>
                            }
                          >
                            <Descriptions size="small" column={1}>
                              <Descriptions.Item label="操作系统">
                                {minion.os || '未知'}
                              </Descriptions.Item>
                              <Descriptions.Item label="架构">
                                {minion.arch || '未知'}
                              </Descriptions.Item>
                              <Descriptions.Item label="Salt版本">
                                {minion.salt_version || '未知'}
                              </Descriptions.Item>
                              <Descriptions.Item label="最后响应">
                                {minion.last_seen || '未知'}
                              </Descriptions.Item>
                            </Descriptions>
                          </Card>
                        </List.Item>
                      )}
                    />
                    {minions.length === 0 && (
                      <div style={{ textAlign: 'center', padding: '40px 0' }}>
                        <Text type="secondary">暂无Minion数据</Text>
                      </div>
                    )}
                  </>
                )}
              </TabPane>

              <TabPane tab="作业历史" key="jobs" icon={<PlayCircleOutlined />}>
                {jobsLoading ? (
                  <div style={{ textAlign: 'center', padding: '60px 0' }}>
                    <Spin size="large" />
                    <div style={{ marginTop: 16 }}>正在加载作业历史...</div>
                  </div>
                ) : (
                  <>
                    <List
                      dataSource={jobs}
                      renderItem={job => (
                        <List.Item>
                          <Card 
                            size="small" 
                            style={{ width: '100%' }}
                            title={
                              <Space>
                                <Tag color={getJobStatusColor(job.status)}>
                                  {job.status || '未知'}
                                </Tag>
                                <Text strong>{job.function || job.command}</Text>
                              </Space>
                            }
                            extra={
                              <Text type="secondary">
                                {job.timestamp || job.start_time}
                              </Text>
                            }
                          >
                            <Descriptions size="small" column={2}>
                              <Descriptions.Item label="目标">
                                {job.target || '所有节点'}
                              </Descriptions.Item>
                              <Descriptions.Item label="用户">
                                {job.user || 'root'}
                              </Descriptions.Item>
                              <Descriptions.Item label="持续时间">
                                {job.duration || '未知'}
                              </Descriptions.Item>
                              <Descriptions.Item label="返回码">
                                <Tag color={job.return_code === 0 ? 'green' : 'red'}>
                                  {job.return_code ?? '未知'}
                                </Tag>
                              </Descriptions.Item>
                            </Descriptions>
                            {job.result && (
                              <div style={{ marginTop: 8 }}>
                                <Text type="secondary">结果:</Text>
                                <Paragraph 
                                  code 
                                  style={{ 
                                    marginTop: 4, 
                                    marginBottom: 0, 
                                    maxHeight: 100, 
                                    overflow: 'auto' 
                                  }}
                                >
                                  {typeof job.result === 'string' ? job.result : JSON.stringify(job.result, null, 2)}
                                </Paragraph>
                              </div>
                            )}
                          </Card>
                        </List.Item>
                      )}
                    />
                    {jobs.length === 0 && (
                      <div style={{ textAlign: 'center', padding: '40px 0' }}>
                        <Text type="secondary">暂无作业历史</Text>
                      </div>
                    )}
                  </>
                )}
              </TabPane>
            </Tabs>
          </Card>

          {/* 操作按钮 */}
          <Card>
            <Space>
              <Button 
                type="primary" 
                icon={<ReloadOutlined />} 
                onClick={loadAllData}
                loading={statusLoading || minionsLoading || jobsLoading}
              >
                刷新数据
              </Button>
              <Button 
                icon={<PlayCircleOutlined />}
                onClick={openExecModal}
              >
                执行命令
              </Button>
              <Button 
                icon={<SettingOutlined />}
                onClick={() => {
                  setConfigVisible(true);
                  configForm.setFieldsValue({ target: '*' });
                }}
              >
                配置管理
              </Button>
            </Space>
          </Card>

          {/* 执行命令弹窗 */}
          <Modal
            title="执行自定义命令（Bash / Python）"
            open={execVisible}
            onCancel={() => { setExecVisible(false); closeSSE(); setExecRunning(false); }}
            footer={execFooter}
            width={900}
          >
            <Form form={execForm} layout="vertical">
              <Form.Item name="target" label="目标节点" rules={[{ required: true, message: '请输入目标，例如 * 或 compute* 或 列表' }]}>
                <Input placeholder="例如: * 或 compute* 或 ai-infra-web-01" />
              </Form.Item>
              <Form.Item name="language" label="语言" rules={[{ required: true }]}> 
                <Select>
                  <Option value="bash">Bash</Option>
                  <Option value="python">Python</Option>
                </Select>
              </Form.Item>
              <Form.Item name="code" label="代码" rules={[{ required: true, message: '请输入要执行的代码' }]}>
                <TextArea rows={10} placeholder="# 在此粘贴脚本..." style={{ fontFamily: 'monospace' }} />
              </Form.Item>
              <Form.Item name="timeout" label="超时（秒）">
                <Input type="number" min={10} max={3600} placeholder="120" />
              </Form.Item>
            </Form>

            <Card size="small" title="执行进度" style={{ marginTop: 12 }}>
              <div style={{ maxHeight: 240, overflow: 'auto', background: '#0b1021', color: '#e6e6e6', padding: 8, borderRadius: 6 }}>
                {execEvents.length === 0 ? (
                  <Text type="secondary">等待执行或无日志...</Text>
                ) : (
                  execEvents.map((ev, idx) => (
                    <div key={idx} style={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
                      <span style={{ color: '#7aa2f7' }}>[{new Date(ev.ts || Date.now()).toLocaleTimeString()}]</span>
                      <span style={{ color: ev.type === 'error' ? '#f7768e' : '#9ece6a' }}> {ev.type} </span>
                      {ev.host ? <span style={{ color: '#bb9af7' }}>({ev.host})</span> : null}
                      <span> - {ev.message}</span>
                      {ev.data && (
                        <pre style={{ margin: 0, color: '#e0af68' }}>{typeof ev.data === 'string' ? ev.data : JSON.stringify(ev.data, null, 2)}</pre>
                      )}
                    </div>
                  ))
                )}
              </div>
            </Card>
          </Modal>

          {/* 配置管理弹窗 */}
          <Modal
            title="Salt 配置模板管理"
            open={configVisible}
            onCancel={() => setConfigVisible(false)}
            footer={[
              <Button key="cancel" onClick={() => setConfigVisible(false)}>取消</Button>,
              <Button 
                key="apply" 
                type="primary" 
                onClick={() => {
                  configForm.validateFields().then(values => {
                    message.info(`将应用配置模板: ${values.template} 到目标: ${values.target}`);
                    // TODO: 调用后端 API 应用配置模板
                    // saltStackAPI.applyTemplate({ template: values.template, target: values.target });
                    setConfigVisible(false);
                  });
                }}
              >
                应用配置
              </Button>,
            ]}
            width={700}
          >
            <Form form={configForm} layout="vertical">
              <Form.Item 
                name="target" 
                label="目标节点" 
                rules={[{ required: true, message: '请输入目标节点' }]}
              >
                <Input placeholder="例如: * 或 web* 或 db01" />
              </Form.Item>
              <Form.Item 
                name="template" 
                label="配置模板" 
                rules={[{ required: true, message: '请选择配置模板' }]}
              >
                <Select placeholder="选择要应用的配置模板">
                  {configTemplates.map(t => (
                    <Option key={t.id} value={t.id}>
                      {t.name} - {t.desc}
                    </Option>
                  ))}
                </Select>
              </Form.Item>
              <Alert
                message="提示"
                description="选择配置模板后，将通过 Salt State 在目标节点上应用相应的配置。此功能需要后端 API 支持。"
                type="info"
                showIcon
                style={{ marginTop: 16 }}
              />
            </Form>
          </Modal>
        </Space>
      </Content>
    </Layout>
  );
};

export default SaltStackDashboard;
