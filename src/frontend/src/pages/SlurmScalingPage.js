import React, { useEffect, useState, useCallback } from 'react';
import {
  Card, Row, Col, Statistic, Table, Tag, Space, Alert, Spin, Button,
  Typography, Modal, Form, Input, Select, message, Progress, List,
  Descriptions, Badge, Tabs, Divider, Tooltip, Popconfirm, Checkbox, Skeleton, Dropdown
} from 'antd';
import {
  PlusOutlined, MinusOutlined, ReloadOutlined, ThunderboltOutlined,
  DesktopOutlined, ClusterOutlined, NodeIndexOutlined, ApiOutlined,
  CheckCircleOutlined, ExclamationCircleOutlined, ClockCircleOutlined,
  PlayCircleOutlined, StopOutlined, SettingOutlined, EyeOutlined,
  BarChartOutlined, PauseCircleOutlined, DownOutlined, CloseCircleOutlined,
  HourglassOutlined, SyncOutlined, LinkOutlined, DisconnectOutlined,
  WarningOutlined, UnorderedListOutlined
} from '@ant-design/icons';
import { slurmAPI, saltStackAPI } from '../services/api';
import { useNavigate } from 'react-router-dom';
import SSHAuthConfig from '../components/SSHAuthConfig';
import SaltCommandExecutor from '../components/SaltCommandExecutor';
import SlurmTaskBar from '../components/SlurmTaskBar';
import ExternalClusterManagement from '../components/slurm/ExternalClusterManagement';
import { useI18n } from '../hooks/useI18n';
import { useTheme } from '../hooks/useTheme';

const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;
const { Option } = Select;
const { TextArea } = Input;

// 扩展的 SLURM API
const extendedSlurmAPI = {
  ...slurmAPI,
  // 扩缩容相关 API（直接复用已在 services/api.js 中定义的方法）
  getScalingStatus: () => slurmAPI.getScalingStatus(),
  scaleUp: (nodes) => slurmAPI.scaleUp(nodes),
  scaleDown: (nodeIds) => slurmAPI.scaleDown(nodeIds),
  getNodeTemplates: () => slurmAPI.getNodeTemplates(),
  createNodeTemplate: (template) => slurmAPI.createNodeTemplate(template),
  deleteNodeTemplate: (id) => slurmAPI.deleteNodeTemplate(id),
  // SaltStack 联动 API（使用 saltStackAPI 封装）
  getSaltStackIntegration: () => saltStackAPI.getSaltStackIntegration(),
  deploySaltMinion: (nodeConfig) => saltStackAPI.deploySaltMinion(nodeConfig),
  executeSaltCommand: (command) => saltStackAPI.executeSaltCommand(command),
  getSaltJobs: () => saltStackAPI.getSaltJobs(),
};

const SlurmScalingPage = () => {
  const navigate = useNavigate();
  const { t } = useI18n();
  const { isDark } = useTheme();
  // 基础状态
  const [summary, setSummary] = useState(null);
  const [nodes, setNodes] = useState([]);
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  // 分阶段加载状态
  const [loadingStages, setLoadingStages] = useState({
    summary: true,
    nodes: true,
    jobs: true,
    scaling: true,
    templates: true,
    salt: true
  });

  // 扩缩容相关状态
  const [scalingStatus, setScalingStatus] = useState(null);
  const [nodeTemplates, setNodeTemplates] = useState([]);
  const [saltIntegration, setSaltIntegration] = useState(null);
  const [saltJobs, setSaltJobs] = useState([]);

  // 模态框状态
  const [scaleUpModal, setScaleUpModal] = useState(false);
  const [scaleDownModal, setScaleDownModal] = useState(false);
  const [templateModal, setTemplateModal] = useState(false);
  const [saltCommandModal, setSaltCommandModal] = useState(false);

  // 节点选择状态
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [selectedJobKeys, setSelectedJobKeys] = useState([]);
  const [operationLoading, setOperationLoading] = useState(false);

  // 表单
  const [scaleUpForm] = Form.useForm();
  const [templateForm] = Form.useForm();
  const [saltCommandForm] = Form.useForm();

  // 表格列定义
  const nodeColumns = [
    { title: t('slurmScaling.nodeColumns.name'), dataIndex: 'name', key: 'name' },
    { title: t('slurmScaling.nodeColumns.partition'), dataIndex: 'partition', key: 'partition' },
    { title: t('slurmScaling.nodeColumns.state'), dataIndex: 'state', key: 'state',
      render: (state) => <Tag color={getNodeStateColor(state)}>{state}</Tag> },
    { title: t('slurmScaling.nodeColumns.cpu'), dataIndex: 'cpus', key: 'cpus' },
    { title: t('slurmScaling.nodeColumns.memory'), dataIndex: 'memory_mb', key: 'memory_mb' },
    { title: t('slurmScaling.nodeColumns.saltStatus'), dataIndex: 'salt_status', key: 'salt_status',
      render: (status) => <Badge status={getSaltStatus(status)} text={status || t('slurmScaling.status.notConfigured')} /> },
    {
      title: t('slurmScaling.nodeColumns.actions'),
      key: 'action',
      render: (_, record) => (
        <Space size="small">
          <Tooltip title={t('slurmScaling.reinitialize')}>
            <Button size="small" icon={<ReloadOutlined />} />
          </Tooltip>
          <Popconfirm
            title={
              <div>
                <div>{t('slurmScaling.confirmDeleteNode')} <strong>{record.name}</strong>?</div>
                <div style={{ marginTop: 8, fontSize: '12px', color: isDark ? '#999' : '#666' }}>
                  {t('slurmScaling.confirmDeleteNodeDesc')}
                </div>
              </div>
            }
            onConfirm={() => handleDeleteNode(record, false)}
            okText={t('slurmScaling.ok')}
            cancelText={t('slurmScaling.cancel')}
            okButtonProps={{ danger: true }}
          >
            <Button size="small" danger icon={<MinusOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const templateColumns = [
    { title: t('slurmScaling.templateColumns.name'), dataIndex: 'name', key: 'name' },
    { title: t('slurmScaling.templateColumns.cpus'), dataIndex: 'cpus', key: 'cpus' },
    { title: t('slurmScaling.templateColumns.memoryGb'), dataIndex: 'memory_gb', key: 'memory_gb' },
    { title: t('slurmScaling.templateColumns.diskGb'), dataIndex: 'disk_gb', key: 'disk_gb' },
    { title: t('slurmScaling.templateColumns.os'), dataIndex: 'os', key: 'os' },
    {
      title: t('slurmScaling.templateColumns.actions'),
      key: 'action',
      render: (_, record) => (
        <Space size="small">
          <Button size="small" onClick={() => handleUseTemplate(record)}>
            {t('slurmScaling.useTemplate')}
          </Button>
          <Button size="small" danger onClick={() => handleDeleteTemplate(record.id)}>
            {t('slurmScaling.deleteTemplate')}
          </Button>
        </Space>
      ),
    },
  ];

  // 工具函数
  const getNodeStateColor = (state) => {
    if (!state) return 'default';
    const stateStr = state.toLowerCase();
    if (stateStr.includes('idle')) return 'green';
    if (stateStr.includes('alloc')) return 'blue';
    if (stateStr.includes('down')) return 'red';
    if (stateStr.includes('maint')) return 'orange';
    return 'default';
  };

  const getSaltStatus = (status) => {
    if (!status) return 'default';
    const statusStr = status.toLowerCase();
    if (statusStr.includes('up') || statusStr.includes('online')) return 'success';
    if (statusStr.includes('down') || statusStr.includes('offline')) return 'error';
    if (statusStr.includes('pending')) return 'processing';
    return 'default';
  };

  // 更新加载阶段状态
  const updateLoadingStage = useCallback((stage, isLoading) => {
    setLoadingStages(prev => ({
      ...prev,
      [stage]: isLoading
    }));
  }, []);

  // 分阶段异步加载数据
  const loadDataAsync = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      // 第一阶段：加载核心数据（优先显示）
      // 并行加载摘要和节点信息
      Promise.all([
        slurmAPI.getSummary()
          .then(res => {
            setSummary(res.data?.data);
            updateLoadingStage('summary', false);
          })
          .catch(e => {
            console.error('加载摘要失败:', e);
            updateLoadingStage('summary', false);
          }),
        
        slurmAPI.getNodes()
          .then(res => {
            setNodes(res.data?.data || []);
            updateLoadingStage('nodes', false);
          })
          .catch(e => {
            console.error('加载节点失败:', e);
            updateLoadingStage('nodes', false);
          })
      ]);

      // 第二阶段：加载作业信息（稍后加载）
      setTimeout(() => {
        slurmAPI.getJobs()
          .then(res => {
            setJobs(res.data?.data || []);
            updateLoadingStage('jobs', false);
          })
          .catch(e => {
            console.error('加载作业失败:', e);
            updateLoadingStage('jobs', false);
          });
      }, 100);

      // 第三阶段：加载扩展功能数据（延迟加载）
      setTimeout(() => {
        Promise.all([
          extendedSlurmAPI.getScalingStatus()
            .then(res => {
              setScalingStatus(res.data?.data);
              updateLoadingStage('scaling', false);
            })
            .catch(e => {
              console.error('加载扩缩容状态失败:', e);
              updateLoadingStage('scaling', false);
            }),
          
          extendedSlurmAPI.getNodeTemplates()
            .then(res => {
              try {
                const templateData = res.data?.data || [];
                setNodeTemplates(Array.isArray(templateData) ? templateData : []);
              } catch (templateError) {
                console.warn('处理模板数据失败:', templateError);
                setNodeTemplates([]);
              }
              updateLoadingStage('templates', false);
            })
            .catch(e => {
              console.error('加载节点模板失败:', e);
              updateLoadingStage('templates', false);
            }),
          
          Promise.all([
            extendedSlurmAPI.getSaltStackIntegration()
              .then(res => {
                setSaltIntegration(res.data?.data);
              })
              .catch(e => {
                console.error('加载SaltStack集成失败:', e);
                // 设置默认的不可用状态
                setSaltIntegration({
                  enabled: false,
                  master_status: 'unavailable',
                  api_status: 'unavailable',
                  minions: { total: 0, online: 0, offline: 0 }
                });
              }),
            
            extendedSlurmAPI.getSaltJobs()
              .then(res => {
                setSaltJobs(res.data?.data || []);
              })
              .catch(e => {
                console.error('加载Salt作业失败:', e);
                setSaltJobs([]);
              })
          ]).finally(() => {
            updateLoadingStage('salt', false);
          })
        ]);
      }, 300);

      setLoading(false);
    } catch (e) {
      console.error('加载数据失败', e);
      setError(e);
      setLoading(false);
      
      // 设置所有阶段为加载完成
      Object.keys(loadingStages).forEach(stage => {
        updateLoadingStage(stage, false);
      });
    }
  }, [updateLoadingStage]);

  // 数据加载函数（兼容旧版本）
  const loadData = async () => {
    await loadDataAsync();
  };

  // 扩缩容处理函数
  const handleScaleUp = async (values) => {
    try {
      // 将多行文本或逗号分隔文本解析为 NodeConfig 数组（与后端契约一致）
      // 支持格式：
      // 1. 每行一个节点（换行分隔）
      // 2. 逗号分隔的节点列表
      // 3. 混合使用换行和逗号
      const nodes = String(values.nodes || '')
        .split(/[\n,]+/)  // 同时支持换行符和逗号作为分隔符
        .map((l) => l.trim())
        .filter(Boolean)
        .map((line) => {
          // 支持 user@host 形式，或仅 host
          let user = values.ssh_user || 'root';
          let host = line;
          if (line.includes('@')) {
            const [u, h] = line.split('@');
            if (u && h) {
              user = u;
              host = h.split(/\s+/)[0];
            }
          } else {
            host = line.split(/\s+/)[0];
          }
          
          // 构建SSH认证信息
          const nodeConfig = {
            host,
            port: values.ssh_port || 22,
            user,
            minion_id: host,
          };

          // 根据认证类型添加认证信息
          if (values.authType === 'password' && values.password) {
            nodeConfig.password = values.password;
            nodeConfig.key_path = ''; // 确保密钥路径为空
          } else if (values.authType === 'key') {
            if (values.private_key) {
              // 如果提供了私钥内容，使用内联私钥
              nodeConfig.private_key = values.private_key;
              nodeConfig.key_path = ''; // 内联私钥时路径为空
            } else if (values.key_path) {
              // 如果只提供了路径，使用文件路径
              nodeConfig.key_path = values.key_path;
            }
            nodeConfig.password = ''; // 确保密码为空
          }

          return nodeConfig;
        });

      if (!nodes.length) {
        message.warning(t('slurmScaling.atLeastOneNode'));
        return;
      }

      // 验证SSH认证信息
      const hasValidAuth = nodes.every(node => 
        node.password || node.key_path || node.private_key
      );
      
      if (!hasValidAuth) {
        message.error(t('slurmScaling.configureSSHAuth'));
        return;
      }

      const response = await extendedSlurmAPI.scaleUp(nodes);
      const opId = response.data?.opId || response.data?.data?.task_id;
      
      if (opId) {
        // 显示带有导航按钮的成功消息
        message.success({
          content: (
            <div>
              <div>{t('slurmScaling.scaleUpSubmitted')}（ID: {opId}）</div>
              <Button 
                size="small" 
                type="link" 
                onClick={() => navigate(`/slurm-tasks?taskId=${opId}&status=running`)}
                style={{ padding: 0, height: 'auto' }}
              >
                {t('slurmScaling.viewTaskProgress')} →
              </Button>
            </div>
          ),
          duration: 6, // 延长显示时间
        });
      } else {
        message.success(t('slurmScaling.scaleUpSubmitted'));
      }
      setScaleUpModal(false);
      scaleUpForm.resetFields();
      loadData();
    } catch (e) {
      const errMsg = e?.response?.data?.error || e.message || t('slurmScaling.unknownError');
      message.error(t('slurmScaling.scaleUpFailed') + ': ' + errMsg);
    }
  };

  const handleScaleDown = async (nodeIds) => {
    try {
      const response = await extendedSlurmAPI.scaleDown(nodeIds);
      const opId = response.data?.opId || response.data?.data?.task_id;
      
      if (opId) {
        // 显示带有导航按钮的成功消息
        message.success({
          content: (
            <div>
              <div>{t('slurmScaling.scaleDownSubmitted')}（ID: {opId}）</div>
              <Button 
                size="small" 
                type="link" 
                onClick={() => navigate(`/slurm-tasks?taskId=${opId}&status=running`)}
                style={{ padding: 0, height: 'auto' }}
              >
                {t('slurmScaling.viewTaskProgress')} →
              </Button>
            </div>
          ),
          duration: 6,
        });
      } else {
        message.success(t('slurmScaling.scaleDownSubmitted'));
      }
      loadData();
    } catch (e) {
      message.error(t('slurmScaling.scaleDownFailed') + ': ' + e.message);
    }
  };

  // 删除节点
  const handleDeleteNode = async (record, force = false) => {
    try {
      await slurmAPI.deleteNode(record.id, force);
      message.success(t('slurmScaling.deleteSuccess', { name: record.name }));
      loadData();
    } catch (e) {
      message.error(t('slurmScaling.deleteFailed') + ': ' + e.message);
    }
  };

  // 批量删除节点
  const handleBatchDeleteNodes = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('slurmScaling.selectNodesFirst'));
      return;
    }

    Modal.confirm({
      title: t('slurmScaling.confirmDeleteNodes'),
      content: (
        <div>
          <p>{t('slurmScaling.confirmDeleteNodesDesc', { count: selectedRowKeys.length })}</p>
          <p style={{ color: '#ff4d4f', marginTop: 8 }}>
            {t('slurmScaling.deleteWarning')}
          </p>
        </div>
      ),
      okText: t('slurmScaling.confirmDelete'),
      cancelText: t('slurmScaling.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        setOperationLoading(true);
        let successCount = 0;
        let failCount = 0;
        const errors = [];

        try {
          // 逐个删除节点并收集结果
          for (const nodeName of selectedRowKeys) {
            try {
              console.log(`Deleting node: ${nodeName}`);
              
              // 优先尝试通过名称删除（适用于所有节点，包括未在数据库中注册的）
              const response = await slurmAPI.deleteNodeByName(nodeName, false);
              console.log(`Node ${nodeName} delete response:`, response);
              
              // 检查响应状态
              if (response.data?.success !== false) {
                successCount++;
                console.log(`✓ Node ${nodeName} deleted successfully`);
              } else {
                failCount++;
                const errorMsg = response.data?.error || t('slurmScaling.unknownError');
                errors.push(`${nodeName}: ${errorMsg}`);
                console.error(`✗ Node ${nodeName} delete failed:`, errorMsg);
              }
            } catch (error) {
              failCount++;
              const errorMsg = error.response?.data?.error || error.message || t('slurmScaling.unknownError');
              errors.push(`${nodeName}: ${errorMsg}`);
              console.error(`✗ Error deleting node ${nodeName}:`, error);
            }
          }

          // 显示结果消息
          if (successCount > 0 && failCount === 0) {
            message.success(t('slurmScaling.batchDeleteSuccess', { count: successCount }));
          } else if (successCount > 0 && failCount > 0) {
            message.warning({
              content: (
                <div>
                  <div>{t('slurmScaling.batchDeletePartial', { success: successCount, fail: failCount })}</div>
                  {errors.length > 0 && (
                    <div style={{ marginTop: 8, fontSize: '12px', color: isDark ? '#999' : '#666' }}>
                      {errors.slice(0, 3).map((err, idx) => (
                        <div key={idx}>• {err}</div>
                      ))}
                      {errors.length > 3 && <div>...+{errors.length - 3}</div>}
                    </div>
                  )}
                </div>
              ),
              duration: 6,
            });
          } else {
            message.error({
              content: (
                <div>
                  <div>{t('slurmScaling.deleteFailed')}</div>
                  {errors.length > 0 && (
                    <div style={{ marginTop: 8, fontSize: '12px' }}>
                      {errors.slice(0, 3).map((err, idx) => (
                        <div key={idx}>• {err}</div>
                      ))}
                      {errors.length > 3 && <div>...+{errors.length - 3}</div>}
                    </div>
                  )}
                </div>
              ),
              duration: 6,
            });
          }

          setSelectedRowKeys([]);
          await loadData();
        } catch (error) {
          console.error('Batch delete nodes error:', error);
          message.error(t('slurmScaling.deleteError'));
        } finally {
          setOperationLoading(false);
        }
      },
    });
  };

  // 节点操作函数
  const handleNodeOperation = async (action, actionLabel, reason = '') => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('slurmScaling.selectNodesFirst'));
      return;
    }

    Modal.confirm({
      title: t('slurmScaling.confirmOperation', { action: actionLabel }),
      content: t('slurmScaling.confirmOperationDesc', { count: selectedRowKeys.length, action: actionLabel }),
      okText: t('slurmScaling.confirm'),
      cancelText: t('slurmScaling.cancel'),
      onOk: async () => {
        setOperationLoading(true);
        try {
          const response = await slurmAPI.manageNodes(selectedRowKeys, action, reason);
          if (response.data?.success) {
            message.success(response.data.message || `成功${actionLabel} ${selectedRowKeys.length} 个节点`);
            setSelectedRowKeys([]);
            // 重新加载节点列表
            await loadData();
          } else {
            message.error(response.data?.error || `${actionLabel}节点失败`);
          }
        } catch (error) {
          console.error(`${actionLabel}节点失败:`, error);
          message.error(error.response?.data?.error || `${actionLabel}节点失败，请稍后重试`);
        } finally {
          setOperationLoading(false);
        }
      },
    });
  };

  // 节点操作菜单项
  const nodeOperationMenuItems = [
    {
      key: 'resume',
      label: t('slurmScaling.nodeOperations.resume'),
      icon: <PlayCircleOutlined />,
      onClick: () => handleNodeOperation('resume', t('slurmScaling.resumeLabel')),
    },
    {
      key: 'drain',
      label: t('slurmScaling.nodeOperations.drain'),
      icon: <PauseCircleOutlined />,
      onClick: () => handleNodeOperation('drain', t('slurmScaling.drainLabel')),
    },
    {
      key: 'down',
      label: t('slurmScaling.nodeOperations.down'),
      icon: <StopOutlined />,
      onClick: () => handleNodeOperation('down', t('slurmScaling.downLabel')),
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: t('slurmScaling.deleteNodes'),
      icon: <MinusOutlined />,
      danger: true,
      onClick: () => handleBatchDeleteNodes(),
    },
  ];

  // 作业操作函数
  const handleJobOperation = async (action, actionLabel, signal = '') => {
    if (selectedJobKeys.length === 0) {
      message.warning(t('slurmScaling.selectJobsFirst'));
      return;
    }

    Modal.confirm({
      title: t('slurmScaling.confirmJobOperation', { action: actionLabel }),
      content: t('slurmScaling.confirmJobOperationDesc', { count: selectedJobKeys.length, action: actionLabel }),
      okText: t('slurmScaling.confirm'),
      cancelText: t('slurmScaling.cancel'),
      onOk: async () => {
        setOperationLoading(true);
        try {
          const response = await slurmAPI.manageJobs(selectedJobKeys, action, signal);
          if (response.data?.success) {
            message.success(response.data.message || `${actionLabel} ${selectedJobKeys.length} jobs`);
            setSelectedJobKeys([]);
            // 重新加载作业列表
            await loadData();
          } else {
            message.error(response.data?.error || `${actionLabel} failed`);
          }
        } catch (error) {
          console.error(`${actionLabel} jobs failed:`, error);
          message.error(error.response?.data?.error || `${actionLabel} failed`);
        } finally {
          setOperationLoading(false);
        }
      },
    });
  };

  // 作业操作菜单项
  const jobOperationMenuItems = [
    {
      key: 'cancel',
      label: t('slurmScaling.jobOperations.cancel'),
      icon: <CloseCircleOutlined />,
      onClick: () => handleJobOperation('cancel', t('slurmScaling.cancelLabel')),
    },
    {
      key: 'hold',
      label: t('slurmScaling.jobOperations.hold'),
      icon: <PauseCircleOutlined />,
      onClick: () => handleJobOperation('hold', t('slurmScaling.holdLabel')),
    },
    {
      key: 'release',
      label: t('slurmScaling.jobOperations.release'),
      icon: <PlayCircleOutlined />,
      onClick: () => handleJobOperation('release', t('slurmScaling.releaseLabel')),
    },
    {
      key: 'suspend',
      label: t('slurmScaling.jobOperations.suspend'),
      icon: <HourglassOutlined />,
      onClick: () => handleJobOperation('suspend', t('slurmScaling.suspendLabel')),
    },
    {
      key: 'resume',
      label: t('slurmScaling.jobOperations.resume'),
      icon: <SyncOutlined />,
      onClick: () => handleJobOperation('resume', t('slurmScaling.resumeLabel2')),
    },
    {
      key: 'requeue',
      label: t('slurmScaling.jobOperations.requeue'),
      icon: <ReloadOutlined />,
      onClick: () => handleJobOperation('requeue', t('slurmScaling.requeueLabel')),
    },
  ];

  const handleCreateTemplate = async (values) => {
    try {
      await extendedSlurmAPI.createNodeTemplate(values);
      message.success(t('slurmScaling.templateCreated'));
      setTemplateModal(false);
      templateForm.resetFields();
      loadData();
    } catch (e) {
      message.error(t('slurmScaling.templateCreateFailed') + ': ' + e.message);
    }
  };

  const handleDeleteTemplate = async (templateId) => {
    try {
      await extendedSlurmAPI.deleteNodeTemplate(templateId);
      message.success(t('slurmScaling.templateDeleted'));
      loadData();
    } catch (e) {
      message.error(t('slurmScaling.templateDeleteFailed') + ': ' + e.message);
    }
  };

  const handleUseTemplate = (template) => {
    scaleUpForm.setFieldsValue({
      cpus: template.cpus,
      memory_gb: template.memory_gb,
      disk_gb: template.disk_gb,
      os: template.os,
    });
    setScaleUpModal(true);
  };

  const handleExecuteSaltCommand = async (values) => {
    try {
      await extendedSlurmAPI.executeSaltCommand(values);
      message.success(t('slurmScaling.saltCommandExecuted'));
      setSaltCommandModal(false);
      saltCommandForm.resetFields();
      loadData();
    } catch (e) {
      message.error(t('slurmScaling.saltCommandFailed') + ': ' + e.message);
    }
  };

  useEffect(() => {
    // 立即加载数据（异步方式）
    loadDataAsync();
    
    // 定时刷新（每30秒）
    const interval = setInterval(() => {
      loadDataAsync();
    }, 30000);
    
    return () => clearInterval(interval);
  }, [loadDataAsync]);

  // 首次加载时不显示全屏加载，而是直接显示页面框架
  // 刷新函数 - 同时刷新页面数据和任务
  const handleRefresh = async () => {
    await loadData();
    // 触发任务栏刷新的方法将通过ref传递
    if (taskBarRef.current && taskBarRef.current.refresh) {
      taskBarRef.current.refresh();
    }
  };

  // 任务栏的ref
  const taskBarRef = React.useRef(null);

  // 数据通过骨架屏逐步加载
  return (
    <div style={{ padding: 24, position: 'relative' }}>
      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        {/* 标题行 */}
        <Row gutter={16} align="middle">
          <Col flex="auto">
            <Title level={2} style={{ marginBottom: 0 }}>
              <ClusterOutlined /> SLURM 集群管理
            </Title>
          </Col>
        </Row>

        {/* 任务栏和操作按钮并排显示 */}
        <Row gutter={16} align="middle">
          <Col flex="450px">
            <div style={{
              boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
              borderRadius: '4px'
            }}>
              <SlurmTaskBar ref={taskBarRef} maxItems={5} refreshInterval={15000} />
            </div>
          </Col>
          <Col flex="auto">
            <Space>
              <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={loading}>
                刷新
              </Button>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => setScaleUpModal(true)}
              >
                扩容节点
              </Button>
              <Button
                icon={<ThunderboltOutlined />}
                onClick={() => setSaltCommandModal(true)}
              >
                SaltStack 命令
              </Button>
            </Space>
          </Col>
        </Row>

        {error && (
          <Alert
            type="error"
            showIcon
            message={t('slurmScaling.dataLoadFailed')}
            description={t('slurmScaling.checkBackendService')}
          />
        )}

        {/* 集群概览 - 支持骨架屏 */}
        <Row gutter={16}>
          <Col span={4}>
            <Card>
              {loadingStages.summary ? (
                <Skeleton active paragraph={{ rows: 1 }} />
              ) : (
                <Statistic
                  title={t('slurmScaling.stats.totalNodes')}
                  value={summary?.nodes_total || 0}
                  prefix={<NodeIndexOutlined />}
                />
              )}
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              {loadingStages.summary ? (
                <Skeleton active paragraph={{ rows: 1 }} />
              ) : (
                <Statistic
                  title={t('slurmScaling.stats.idleNodes')}
                  value={summary?.nodes_idle || 0}
                  valueStyle={{ color: '#3f8600' }}
                />
              )}
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              {loadingStages.summary ? (
                <Skeleton active paragraph={{ rows: 1 }} />
              ) : (
                <Statistic
                  title={t('slurmScaling.stats.runningNodes')}
                  value={summary?.nodes_alloc || 0}
                  valueStyle={{ color: '#1890ff' }}
                />
              )}
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              {loadingStages.jobs ? (
                <Skeleton active paragraph={{ rows: 1 }} />
              ) : (
                <Statistic
                  title={t('slurmScaling.stats.runningJobs')}
                  value={summary?.jobs_running || 0}
                  prefix={<PlayCircleOutlined />}
                />
              )}
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              {loadingStages.jobs ? (
                <Skeleton active paragraph={{ rows: 1 }} />
              ) : (
                <Statistic
                  title={t('slurmScaling.stats.pendingJobs')}
                  value={summary?.jobs_pending || 0}
                  valueStyle={{ color: '#faad14' }}
                />
              )}
            </Card>
          </Col>
          <Col span={4}>
            <Card>
              {loadingStages.salt ? (
                <Skeleton active paragraph={{ rows: 1 }} />
              ) : (
                (() => {
                  const minions = saltIntegration?.minions || {};
                  const total = minions.total || 0;
                  const online = minions.online || 0;
                  const offline = minions.offline || 0;
                  
                  let icon, valueStyle, suffix, displayValue;
                  
                  // 全部在线：绿色链接图标
                  if (total > 0 && online === total && offline === 0) {
                    icon = <LinkOutlined style={{ color: '#52c41a' }} />;
                    valueStyle = { color: '#52c41a' };
                    suffix = t('slurmScaling.status.online');
                    displayValue = online;
                  }
                  // 部分在线：黄色警告图标
                  else if (total > 0 && online > 0 && offline > 0) {
                    icon = <WarningOutlined style={{ color: '#faad14' }} />;
                    valueStyle = { color: '#faad14' };
                    suffix = `${t('slurmScaling.status.online')} / ${total}`;
                    displayValue = online;
                  }
                  // 全部离线或无连接：红色断开图标
                  else {
                    icon = <DisconnectOutlined style={{ color: '#ff4d4f' }} />;
                    valueStyle = { color: '#ff4d4f' };
                    suffix = total > 0 ? t('slurmScaling.status.offline') : '';
                    displayValue = total;
                  }
                  
                  return (
                    <Statistic
                      title="SaltStack Minions"
                      value={displayValue}
                      prefix={icon}
                      suffix={suffix}
                      valueStyle={valueStyle}
                    />
                  );
                })()
              )}
            </Card>
          </Col>
        </Row>

        {/* 扩缩容状态 - 始终显示 */}
        <Card 
          title={t('slurmScaling.scalingStatus')} 
          size="small"
          extra={
            <Badge 
              status={
                loadingStages.scaling ? 'processing' : 
                (scalingStatus?.active ? 'processing' : 'default')
              } 
              text={
                loadingStages.scaling ? t('slurmScaling.status.loading') :
                (scalingStatus?.active ? t('slurmScaling.status.active') : t('slurmScaling.status.idle'))
              }
            />
          }
        >
          {loadingStages.scaling ? (
            <Skeleton active paragraph={{ rows: 1 }} />
          ) : (
            <Row gutter={[16, 16]} align="middle">
              <Col xs={24} sm={6}>
                <Statistic 
                  title={t('slurmScaling.stats.activeTasks')} 
                  value={scalingStatus?.active_tasks || 0}
                  prefix={<ThunderboltOutlined />}
                  valueStyle={{ color: scalingStatus?.active_tasks > 0 ? '#1890ff' : undefined }}
                />
              </Col>
              <Col xs={24} sm={6}>
                <Statistic 
                  title={t('slurmScaling.stats.successNodes')} 
                  value={scalingStatus?.success_nodes || 0}
                  prefix={<CheckCircleOutlined />}
                  valueStyle={{ color: scalingStatus?.success_nodes > 0 ? '#52c41a' : undefined }}
                />
              </Col>
              <Col xs={24} sm={6}>
                <Statistic 
                  title={t('slurmScaling.stats.failedNodes')} 
                  value={scalingStatus?.failed_nodes || 0}
                  prefix={<CloseCircleOutlined />}
                  valueStyle={{ color: scalingStatus?.failed_nodes > 0 ? '#ff4d4f' : undefined }}
                />
              </Col>
              <Col xs={24} sm={6}>
                <div style={{ textAlign: 'center' }}>
                  <div style={{ marginBottom: 8, color: isDark ? '#999' : '#666', fontSize: '14px' }}>{t('common.progress', 'Progress')}</div>
                  <Progress
                    type="circle"
                    percent={Math.round(scalingStatus?.progress || 0)}
                    status={
                      scalingStatus?.active ? 'active' : 
                      (scalingStatus?.failed_nodes > 0 ? 'exception' : 'success')
                    }
                    width={80}
                  />
                </div>
              </Col>
            </Row>
          )}
        </Card>

        <Tabs defaultActiveKey="nodes" type="card">
          <TabPane tab={<span><DesktopOutlined />{t('slurm.nodeManagement', 'Node Management')}</span>} key="nodes">
            <Card title={t('slurmScaling.clusterNodes')} extra={
              <Space>
                {selectedRowKeys.length > 0 && (
                  <>
                    <Text type="secondary">{t('common.selected', { count: selectedRowKeys.length })}</Text>
                    <Dropdown 
                      menu={{ items: nodeOperationMenuItems }}
                      placement="bottomRight"
                      disabled={operationLoading}
                    >
                      <Button 
                        type="primary" 
                        loading={operationLoading}
                        icon={<DownOutlined />}
                      >
                        {t('slurmScaling.nodeActions')}
                      </Button>
                    </Dropdown>
                  </>
                )}
                <Button icon={<PlusOutlined />} onClick={() => setScaleUpModal(true)}>
                  {t('slurm.addNode', 'Add Node')}
                </Button>
                <Button icon={<SettingOutlined />} onClick={() => setTemplateModal(true)}>
                  {t('slurm.manageTemplates', 'Manage Templates')}
                </Button>
              </Space>
            }>
              {loadingStages.nodes ? (
                <Skeleton active paragraph={{ rows: 5 }} />
              ) : (
                <Table
                  rowKey="name"
                  dataSource={nodes}
                  columns={nodeColumns}
                  size="small"
                  pagination={{ pageSize: 10 }}
                  rowSelection={{
                    selectedRowKeys,
                    onChange: setSelectedRowKeys,
                    selections: [
                      Table.SELECTION_ALL,
                      Table.SELECTION_INVERT,
                      Table.SELECTION_NONE,
                    ],
                  }}
                />
              )}
            </Card>
          </TabPane>

          <TabPane tab={<span><PlayCircleOutlined />{t('slurm.jobQueue', 'Job Queue')}</span>} key="jobs">
            <Card title={t('slurmScaling.jobStatus')} extra={
              <Space>
                {selectedJobKeys.length > 0 && (
                  <>
                    <Text type="secondary">{t('common.selected', { count: selectedJobKeys.length })}</Text>
                    <Dropdown 
                      menu={{ items: jobOperationMenuItems }}
                      placement="bottomRight"
                      disabled={operationLoading}
                    >
                      <Button 
                        type="primary" 
                        loading={operationLoading}
                        icon={<DownOutlined />}
                      >
                        {t('slurmScaling.jobActions')}
                      </Button>
                    </Dropdown>
                  </>
                )}
              </Space>
            }>
              {loadingStages.jobs ? (
                <Skeleton active paragraph={{ rows: 5 }} />
              ) : (
                <Table
                  rowKey="id"
                  dataSource={jobs}
                  columns={[
                    { title: t('slurmScaling.jobColumns.id'), dataIndex: 'id', key: 'id' },
                    { title: t('slurmScaling.jobColumns.name'), dataIndex: 'name', key: 'name' },
                    { title: t('slurmScaling.jobColumns.user'), dataIndex: 'user', key: 'user' },
                    { title: t('slurmScaling.jobColumns.state'), dataIndex: 'state', key: 'state',
                      render: (state) => <Tag color={state === 'RUNNING' ? 'blue' : state === 'PENDING' ? 'orange' : 'default'}>{state}</Tag> },
                    { title: t('slurmScaling.jobColumns.elapsed'), dataIndex: 'elapsed', key: 'elapsed' },
                    { title: t('slurmScaling.jobColumns.nodes'), dataIndex: 'nodes', key: 'nodes' },
                  ]}
                  size="small"
                  pagination={{ pageSize: 10 }}
                  rowSelection={{
                    selectedRowKeys: selectedJobKeys,
                    onChange: setSelectedJobKeys,
                    selections: [
                      Table.SELECTION_ALL,
                      Table.SELECTION_INVERT,
                      Table.SELECTION_NONE,
                    ],
                  }}
                />
              )}
            </Card>
          </TabPane>

          <TabPane tab={<span><ThunderboltOutlined />SaltStack</span>} key="saltstack">
            <Row gutter={16}>
              <Col span={12}>
                <Card title={t('slurmScaling.saltStackStatus')}>
                  <Descriptions column={2} size="small">
                    <Descriptions.Item label={t('slurmScaling.saltMasterStatus')}>
                      <span style={{
                        color: ['running', 'available'].includes(saltIntegration?.master_status) ? '#52c41a' : '#ff4d4f',
                        fontSize: '16px',
                        fontWeight: 500
                      }}>
                        {saltIntegration?.master_status || t('slurmScaling.status.unknown')}
                      </span>
                    </Descriptions.Item>
                    <Descriptions.Item label={t('slurmScaling.saltApiStatus')}>
                      <span style={{
                        color: ['running', 'available', 'connected'].includes(saltIntegration?.api_status) ? '#52c41a' : '#ff4d4f',
                        fontSize: '16px',
                        fontWeight: 500
                      }}>
                        {saltIntegration?.api_status || t('slurmScaling.status.unknown')}
                      </span>
                    </Descriptions.Item>
                    <Descriptions.Item label={t('slurmCluster.salt.totalMinions')}>
                      <Space>
                        {(() => {
                          const minions = saltIntegration?.minions || {};
                          const total = minions.total || 0;
                          const online = minions.online || 0;
                          const offline = minions.offline || 0;
                          
                          console.log('Minions data:', { total, online, offline, minions });
                          
                          // 全部在线：绿色链接图标
                          if (total > 0 && online === total && offline === 0) {
                            return (
                              <>
                                <LinkOutlined style={{ color: '#52c41a', fontSize: '16px' }} />
                                <span style={{ fontSize: '16px', fontWeight: 500, color: '#52c41a' }}>
                                  {online} {t('slurmScaling.status.online')}
                                </span>
                              </>
                            );
                          }
                          // 部分在线：黄色警告图标
                          else if (total > 0 && online > 0 && offline > 0) {
                            return (
                              <>
                                <WarningOutlined style={{ color: '#faad14', fontSize: '16px' }} />
                                <span style={{ fontSize: '16px', fontWeight: 500, color: '#faad14' }}>
                                  {online} {t('slurmScaling.status.online')} / {total}
                                </span>
                              </>
                            );
                          }
                          // 全部离线或无连接：红色断开图标
                          else {
                            return (
                              <>
                                <DisconnectOutlined style={{ color: '#ff4d4f', fontSize: '16px' }} />
                                <span style={{ fontSize: '16px', fontWeight: 500, color: '#ff4d4f' }}>
                                  {total > 0 ? `${total} ${t('slurmScaling.status.offline')}` : t('common.noConnection', 'No Connection')}
                                </span>
                              </>
                            );
                          }
                        })()}
                      </Space>
                    </Descriptions.Item>
                    <Descriptions.Item label={t('slurmCluster.salt.recentJobs')}>
                      {saltJobs?.length || 0}
                    </Descriptions.Item>
                  </Descriptions>
                </Card>
              </Col>
              <Col span={12}>
                <Card title={t('slurmCluster.salt.recentJobs')}>
                  <List
                    size="small"
                    dataSource={saltJobs?.slice(0, 5) || []}
                    renderItem={(job) => {
                      // 计算任务状态：根据 results 判断成功/失败
                      const successCount = Object.values(job.results || {}).filter(v => v === true).length;
                      const totalCount = Object.keys(job.results || {}).length;
                      const status = totalCount > 0 && successCount === totalCount ? 'success' : 
                                     totalCount > 0 && successCount > 0 ? 'warning' : 'error';
                      const statusText = totalCount > 0 ? `${successCount}/${totalCount}` : t('common.noResponse', 'No Response');
                      
                      return (
                        <List.Item>
                          <List.Item.Meta
                            title={<Text strong>{job.function}</Text>}
                            description={
                              <Space size="small">
                                <Text type="secondary">{job.target}</Text>
                                <Text>•</Text>
                                <Text type="secondary">{statusText}</Text>
                                {job.start_time && (
                                  <>
                                    <Text>•</Text>
                                    <Text type="secondary">
                                      {new Date(job.start_time).toLocaleString(undefined, {
                                        month: '2-digit',
                                        day: '2-digit',
                                        hour: '2-digit',
                                        minute: '2-digit'
                                      })}
                                    </Text>
                                  </>
                                )}
                              </Space>
                            }
                          />
                          <Badge status={status === 'success' ? 'success' : status === 'warning' ? 'warning' : 'error'} />
                        </List.Item>
                      );
                    }}
                  />
                </Card>
              </Col>
            </Row>
            
            {/* SaltStack 命令执行器 */}
            <Divider />
            <SaltCommandExecutor />
          </TabPane>

          <TabPane tab={<span><SettingOutlined />节点模板</span>} key="templates">
            <Card title="节点配置模板" extra={
              <Button icon={<PlusOutlined />} onClick={() => setTemplateModal(true)}>
                新建模板
              </Button>
            }>
              <Table
                rowKey="id"
                dataSource={nodeTemplates}
                columns={templateColumns}
                size="small"
                pagination={{ pageSize: 10 }}
                loading={loading}
              />
            </Card>
          </TabPane>

          <TabPane tab={<span><BarChartOutlined />监控仪表板</span>} key="dashboard">
            <Card 
              title={
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <BarChartOutlined />
                  <span>SLURM 集群监控</span>
                  <Badge status="processing" text="实时" />
                </div>
              } 
              style={{ height: '600px' }}
            >
              {/* 监控iframe容器 - 使用 Nightingale 监控系统 */}
              <div style={{ height: '520px', border: '1px solid #d9d9d9', borderRadius: '6px', position: 'relative' }}>
                <iframe
                  id="slurm-dashboard-iframe"
                  src={`${window.location.protocol}//${window.location.hostname}:${window.location.port}/nightingale/`}
                  style={{
                    width: '100%',
                    height: '100%',
                    border: 'none',
                    borderRadius: '6px'
                  }}
                  title="SLURM 集群监控"
                  onLoad={(e) => {
                    console.log('Nightingale 监控仪表板加载完成');
                  }}
                />
              </div>
              <div style={{ marginTop: '8px', textAlign: 'center' }}>
                <Space>
                  <Text type="secondary">使用 Nightingale 实时监控集群状态和任务进度</Text>
                  <Button 
                    size="small" 
                    icon={<ReloadOutlined />}
                    onClick={() => {
                      const iframe = document.querySelector('iframe[title="SLURM Dashboard"]');
                      if (iframe) {
                        iframe.src = iframe.src;
                      }
                    }}
                  >
                    刷新
                  </Button>
                </Space>
              </div>
            </Card>
          </TabPane>

          <TabPane tab={<span><ClusterOutlined />外部集群管理</span>} key="external-clusters">
            <ExternalClusterManagement />
          </TabPane>

          <TabPane tab={<span><UnorderedListOutlined />任务监控</span>} key="tasks">
            <Card 
              title="SLURM 任务状态" 
              extra={
                <Button 
                  type="link" 
                  onClick={() => navigate('/slurm-tasks')}
                >
                  查看全部 →
                </Button>
              }
            >
              <SlurmTaskBar maxItems={10} />
            </Card>
          </TabPane>
        </Tabs>
      </Space>

      {/* 扩容模态框 */}
      <Modal
        title="扩容 SLURM 节点"
        open={scaleUpModal}
        onCancel={() => setScaleUpModal(false)}
        footer={null}
        width={800}
      >
        <Form
          form={scaleUpForm}
          layout="vertical"
          onFinish={handleScaleUp}
        >
          <Form.Item
            name="nodes"
            label="节点配置"
            rules={[{ required: true, message: '请配置要添加的节点' }]}
          >
            <TextArea
              placeholder="每行一个节点配置，或使用逗号分隔多个节点&#10;格式: hostname 或 user@hostname&#10;例如:&#10;worker01&#10;worker02,worker03&#10;root@worker04"
              rows={6}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          {/* SSH认证配置 */}
          <SSHAuthConfig 
            form={scaleUpForm}
            hostFieldName="nodes"
            initialValues={{
              authType: 'password',
              ssh_user: 'root',
              ssh_port: 22
            }}
            showAdvanced={true}
            size="small"
          />

          <Divider>节点规格</Divider>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="cpus" label="CPU核心数">
                <Input placeholder="例如: 4" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="memory_gb" label="内存(GB)">
                <Input placeholder="例如: 8" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="disk_gb" label="磁盘(GB)">
                <Input placeholder="例如: 100" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="os" label="操作系统">
            <Select placeholder="选择操作系统">
              <Option value="ubuntu20.04">Ubuntu 20.04</Option>
              <Option value="ubuntu22.04">Ubuntu 22.04</Option>
              <Option value="centos7">CentOS 7</Option>
              <Option value="centos8">CentOS 8</Option>
              <Option value="rocky8">Rocky Linux 8</Option>
              <Option value="alpine3.18">Alpine 3.18</Option>
            </Select>
          </Form.Item>

          <Form.Item name="auto_deploy_salt" valuePropName="checked">
            <Checkbox>自动部署 SaltStack Minion</Checkbox>
          </Form.Item>

          <Form.Item name="install_singularity" valuePropName="checked">
            <Checkbox>安装 Singularity 容器运行时</Checkbox>
          </Form.Item>

          <Form.Item style={{ textAlign: 'right', marginBottom: 0 }}>
            <Space>
              <Button onClick={() => setScaleUpModal(false)}>取消</Button>
              <Button type="primary" htmlType="submit">
                开始扩容
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 缩容模态框 */}
      <Modal
        title="缩容 SLURM 节点"
        open={scaleDownModal}
        onCancel={() => setScaleDownModal(false)}
        footer={null}
        width={500}
      >
        <Alert
          message="警告"
          description="缩容操作将永久移除选中的节点，请确认这些节点上的作业已完成或已迁移。"
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form layout="vertical">
          <Form.Item label="选择要移除的节点">
            <Select
              mode="multiple"
              placeholder="选择节点"
              style={{ width: '100%' }}
            >
              {nodes.filter(node => node.state?.toLowerCase().includes('idle')).map(node => (
                <Option key={node.name} value={node.name}>
                  {node.name} ({node.state})
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item style={{ textAlign: 'right', marginBottom: 0 }}>
            <Space>
              <Button onClick={() => setScaleDownModal(false)}>取消</Button>
              <Button danger htmlType="submit">
                确认缩容
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 节点模板模态框 */}
      <Modal
        title="节点配置模板"
        open={templateModal}
        onCancel={() => setTemplateModal(false)}
        footer={null}
        width={600}
      >
        <Form
          form={templateForm}
          layout="vertical"
          onFinish={handleCreateTemplate}
        >
          <Form.Item
            name="name"
            label="模板名称"
            rules={[{ required: true, message: '请输入模板名称' }]}
          >
            <Input placeholder="例如: compute-node-medium" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item
                name="cpus"
                label="CPU核心数"
                rules={[{ required: true, message: '请输入CPU核心数' }]}
              >
                <Input type="number" placeholder="4" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="memory_gb"
                label="内存(GB)"
                rules={[{ required: true, message: '请输入内存大小' }]}
              >
                <Input type="number" placeholder="8" />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="disk_gb"
                label="磁盘(GB)"
                rules={[{ required: true, message: '请输入磁盘大小' }]}
              >
                <Input type="number" placeholder="100" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="os"
            label="操作系统"
            rules={[{ required: true, message: '请选择操作系统' }]}
          >
            <Select placeholder="选择操作系统">
              <Option value="ubuntu20.04">Ubuntu 20.04</Option>
              <Option value="ubuntu22.04">Ubuntu 22.04</Option>
              <Option value="centos7">CentOS 7</Option>
              <Option value="centos8">CentOS 8</Option>
              <Option value="rocky8">Rocky Linux 8</Option>
              <Option value="alpine3.18">Alpine 3.18</Option>
            </Select>
          </Form.Item>

          <Form.Item name="description" label="描述">
            <TextArea placeholder="模板描述信息" rows={2} />
          </Form.Item>

          <Form.Item style={{ textAlign: 'right', marginBottom: 0 }}>
            <Space>
              <Button onClick={() => setTemplateModal(false)}>取消</Button>
              <Button type="primary" htmlType="submit">
                创建模板
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* SaltStack 命令模态框 */}
      <Modal
        title="执行 SaltStack 命令"
        open={saltCommandModal}
        onCancel={() => setSaltCommandModal(false)}
        footer={null}
        width={700}
      >
        <Form
          form={saltCommandForm}
          layout="vertical"
          onFinish={handleExecuteSaltCommand}
        >
          <Form.Item
            name="target"
            label="目标节点"
            rules={[{ required: true, message: '请选择目标节点' }]}
          >
            <Select placeholder="选择目标节点或输入模式">
              <Option value="*">所有节点 (*)</Option>
              <Option value="compute*">计算节点 (compute*)</Option>
              <Option value="login*">登录节点 (login*)</Option>
              <Option value="storage*">存储节点 (storage*)</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="function"
            label="Salt 函数"
            rules={[{ required: true, message: '请输入Salt函数' }]}
          >
            <Input placeholder="例如: cmd.run, pkg.install, service.restart" />
          </Form.Item>

          <Form.Item
            name="arguments"
            label="参数"
          >
            <TextArea
              placeholder="函数参数，每行一个参数"
              rows={3}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input placeholder="命令描述" />
          </Form.Item>

          <Form.Item style={{ textAlign: 'right', marginBottom: 0 }}>
            <Space>
              <Button onClick={() => setSaltCommandModal(false)}>取消</Button>
              <Button type="primary" htmlType="submit">
                执行命令
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default SlurmScalingPage;