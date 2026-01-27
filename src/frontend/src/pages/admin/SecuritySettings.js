import React, { useState, useEffect } from 'react';
import {
  Card,
  Tabs,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Switch,
  Space,
  Tag,
  message,
  Popconfirm,
  Tooltip,
  Typography,
  Alert,
  Divider,
  Row,
  Col,
  Statistic,
  Badge,
  QRCode,
  List,
  Spin,
  Empty,
  InputNumber,
} from 'antd';
import {
  SecurityScanOutlined,
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  SafetyOutlined,
  LockOutlined,
  UnlockOutlined,
  StopOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  GlobalOutlined,
  QrcodeOutlined,
  KeyOutlined,
  ReloadOutlined,
  CopyOutlined,
  HistoryOutlined,
  CloudOutlined,
  WechatOutlined,
  DingdingOutlined,
  GithubOutlined,
  GoogleOutlined,
  WindowsOutlined,
  GitlabOutlined,
  ArrowLeftOutlined,
  EnvironmentOutlined,
  CompassOutlined,
  InfoCircleOutlined,
  DesktopOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { securityAPI } from '../../services/api';
import { useI18n } from '../../hooks/useI18n';
import { useTheme } from '../../hooks/useTheme';
import dayjs from 'dayjs';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TextArea } = Input;
const { TabPane } = Tabs;

const SecuritySettings = () => {
  const navigate = useNavigate();
  const { t } = useI18n();
  const { isDark } = useTheme();

  // IP 黑名单状态
  const [blacklist, setBlacklist] = useState([]);
  const [blacklistLoading, setBlacklistLoading] = useState(false);
  const [blacklistModalVisible, setBlacklistModalVisible] = useState(false);
  const [editingBlacklist, setEditingBlacklist] = useState(null);
  const [blacklistForm] = Form.useForm();
  const [selectedBlacklistKeys, setSelectedBlacklistKeys] = useState([]);

  // IP 白名单状态
  const [whitelist, setWhitelist] = useState([]);
  const [whitelistLoading, setWhitelistLoading] = useState(false);
  const [whitelistModalVisible, setWhitelistModalVisible] = useState(false);
  const [whitelistForm] = Form.useForm();

  // 2FA 状态
  const [twoFAStatus, setTwoFAStatus] = useState(null);
  const [twoFALoading, setTwoFALoading] = useState(false);
  const [twoFASetupData, setTwoFASetupData] = useState(null);
  const [twoFASetupVisible, setTwoFASetupVisible] = useState(false);
  const [twoFAVerifyForm] = Form.useForm();
  const [twoFADisableForm] = Form.useForm();
  const [recoveryCodesVisible, setRecoveryCodesVisible] = useState(false);
  const [recoveryCodes, setRecoveryCodes] = useState([]);

  // OAuth 状态
  const [oauthProviders, setOAuthProviders] = useState([]);
  const [oauthLoading, setOAuthLoading] = useState(false);
  const [oauthModalVisible, setOAuthModalVisible] = useState(false);
  const [editingOAuth, setEditingOAuth] = useState(null);
  const [oauthForm] = Form.useForm();

  // 安全配置状态
  const [securityConfig, setSecurityConfig] = useState(null);
  const [configLoading, setConfigLoading] = useState(false);
  const [configForm] = Form.useForm();

  // 审计日志状态
  const [auditLogs, setAuditLogs] = useState([]);
  const [auditLoading, setAuditLoading] = useState(false);
  const [auditPagination, setAuditPagination] = useState({ current: 1, pageSize: 10, total: 0 });

  // 疑似攻击IP状态
  const [suspiciousIPs, setSuspiciousIPs] = useState([]);
  const [suspiciousIPsLoading, setSuspiciousIPsLoading] = useState(false);
  const [suspiciousIPsPagination, setSuspiciousIPsPagination] = useState({ current: 1, pageSize: 10, total: 0 });
  const [loginStats, setLoginStats] = useState(null);
  const [lockedAccounts, setLockedAccounts] = useState([]);
  const [lockedAccountsLoading, setLockedAccountsLoading] = useState(false);

  // 客户端信息状态
  const [clientInfo, setClientInfo] = useState(null);
  const [clientInfoLoading, setClientInfoLoading] = useState(false);
  const [geoIPLookupIP, setGeoIPLookupIP] = useState('');
  const [geoIPLookupResult, setGeoIPLookupResult] = useState(null);
  const [geoIPLookupLoading, setGeoIPLookupLoading] = useState(false);

  // 加载数据
  useEffect(() => {
    fetchBlacklist();
    fetchWhitelist();
    fetch2FAStatus();
    fetchOAuthProviders();
    fetchClientInfo();
    fetchSecurityConfig();
  }, []);

  // ==================== IP 黑名单 ====================
  const fetchBlacklist = async () => {
    setBlacklistLoading(true);
    try {
      const res = await securityAPI.getIPBlacklist();
      setBlacklist(res.data?.data || []);
    } catch (error) {
      message.error(t('security.fetchBlacklistFailed') || '获取 IP 黑名单失败');
    } finally {
      setBlacklistLoading(false);
    }
  };

  const handleAddBlacklist = () => {
    setEditingBlacklist(null);
    blacklistForm.resetFields();
    blacklistForm.setFieldsValue({ type: 'manual', enabled: true });
    setBlacklistModalVisible(true);
  };

  const handleEditBlacklist = (record) => {
    setEditingBlacklist(record);
    blacklistForm.setFieldsValue(record);
    setBlacklistModalVisible(true);
  };

  const handleBlacklistSubmit = async () => {
    try {
      const values = await blacklistForm.validateFields();
      if (editingBlacklist) {
        await securityAPI.updateIPBlacklist(editingBlacklist.id, values);
        message.success(t('security.updateSuccess') || '更新成功');
      } else {
        await securityAPI.addIPBlacklist(values);
        message.success(t('security.addSuccess') || '添加成功');
      }
      setBlacklistModalVisible(false);
      fetchBlacklist();
    } catch (error) {
      message.error(error.response?.data?.error || t('security.operationFailed') || '操作失败');
    }
  };

  const handleDeleteBlacklist = async (id) => {
    try {
      await securityAPI.deleteIPBlacklist(id);
      message.success(t('security.deleteSuccess') || '删除成功');
      fetchBlacklist();
    } catch (error) {
      message.error(t('security.deleteFailed') || '删除失败');
    }
  };

  const handleBatchDeleteBlacklist = async () => {
    try {
      await securityAPI.batchDeleteIPBlacklist(selectedBlacklistKeys);
      message.success(t('security.batchDeleteSuccess') || '批量删除成功');
      setSelectedBlacklistKeys([]);
      fetchBlacklist();
    } catch (error) {
      message.error(t('security.batchDeleteFailed') || '批量删除失败');
    }
  };

  const blacklistColumns = [
    {
      title: 'IP / CIDR',
      dataIndex: 'ip_address',
      key: 'ip_address',
      render: (ip) => <Tag icon={<GlobalOutlined />} color="red">{ip}</Tag>,
    },
    {
      title: t('security.reason') || '原因',
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
    },
    {
      title: t('security.type') || '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type) => {
        const typeConfig = {
          manual: { color: 'blue', text: t('security.manual') || '手动添加' },
          auto: { color: 'orange', text: t('security.auto') || '自动封禁' },
          import: { color: 'green', text: t('security.import') || '批量导入' },
        };
        const config = typeConfig[type] || { color: 'default', text: type };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('security.expireTime') || '过期时间',
      dataIndex: 'expires_at',
      key: 'expires_at',
      render: (time) => time ? dayjs(time).format('YYYY-MM-DD HH:mm') : t('security.permanent') || '永久',
    },
    {
      title: t('security.status') || '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled) => (
        <Badge 
          status={enabled ? 'success' : 'default'} 
          text={enabled ? t('security.enabled') || '启用' : t('security.disabled') || '禁用'} 
        />
      ),
    },
    {
      title: t('common.actions') || '操作',
      key: 'actions',
      render: (_, record) => (
        <Space>
          <Tooltip title={t('common.edit') || '编辑'}>
            <Button type="link" icon={<EditOutlined />} onClick={() => handleEditBlacklist(record)} />
          </Tooltip>
          <Popconfirm
            title={t('security.confirmDelete') || '确定删除此 IP？'}
            onConfirm={() => handleDeleteBlacklist(record.id)}
          >
            <Tooltip title={t('common.delete') || '删除'}>
              <Button type="link" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // ==================== IP 白名单 ====================
  const fetchWhitelist = async () => {
    setWhitelistLoading(true);
    try {
      const res = await securityAPI.getIPWhitelist();
      setWhitelist(res.data?.data || []);
    } catch (error) {
      message.error(t('security.fetchWhitelistFailed') || '获取 IP 白名单失败');
    } finally {
      setWhitelistLoading(false);
    }
  };

  const handleAddWhitelist = () => {
    whitelistForm.resetFields();
    setWhitelistModalVisible(true);
  };

  const handleWhitelistSubmit = async () => {
    try {
      const values = await whitelistForm.validateFields();
      await securityAPI.addIPWhitelist(values);
      message.success(t('security.addSuccess') || '添加成功');
      setWhitelistModalVisible(false);
      fetchWhitelist();
    } catch (error) {
      message.error(error.response?.data?.error || t('security.operationFailed') || '操作失败');
    }
  };

  const handleDeleteWhitelist = async (id) => {
    try {
      await securityAPI.deleteIPWhitelist(id);
      message.success(t('security.deleteSuccess') || '删除成功');
      fetchWhitelist();
    } catch (error) {
      message.error(t('security.deleteFailed') || '删除失败');
    }
  };

  const whitelistColumns = [
    {
      title: 'IP / CIDR',
      dataIndex: 'ip_address',
      key: 'ip_address',
      render: (ip) => <Tag icon={<CheckCircleOutlined />} color="green">{ip}</Tag>,
    },
    {
      title: t('security.description') || '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('security.createdAt') || '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => dayjs(time).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: t('common.actions') || '操作',
      key: 'actions',
      render: (_, record) => (
        <Popconfirm
          title={t('security.confirmDelete') || '确定删除此 IP？'}
          onConfirm={() => handleDeleteWhitelist(record.id)}
        >
          <Tooltip title={t('common.delete') || '删除'}>
            <Button type="link" danger icon={<DeleteOutlined />} />
          </Tooltip>
        </Popconfirm>
      ),
    },
  ];

  // ==================== 二次认证 (2FA) ====================
  const fetch2FAStatus = async () => {
    setTwoFALoading(true);
    try {
      const res = await securityAPI.get2FAStatus();
      // 后端返回 {success: true, data: {...}} 或直接返回数据
      const data = res.data?.data || res.data;
      setTwoFAStatus(data);
    } catch (error) {
      // 如果用户没有设置2FA，返回默认状态
      setTwoFAStatus({ enabled: false });
    } finally {
      setTwoFALoading(false);
    }
  };

  const handleSetup2FA = async () => {
    try {
      setTwoFALoading(true);
      const res = await securityAPI.setup2FA();
      // 后端返回 {success: true, data: {...}}，需要提取 data
      const data = res.data?.data || res.data;
      // 后端返回 qr_code，前端使用 url
      setTwoFASetupData({
        ...data,
        url: data.qr_code || data.url,
      });
      setTwoFASetupVisible(true);
    } catch (error) {
      message.error(t('security.setup2FAFailed') || '设置 2FA 失败');
    } finally {
      setTwoFALoading(false);
    }
  };

  const handleEnable2FA = async () => {
    try {
      const values = await twoFAVerifyForm.validateFields();
      await securityAPI.enable2FA({ code: values.code, secret: twoFASetupData.secret });
      message.success(t('security.enable2FASuccess') || '2FA 启用成功');
      setTwoFASetupVisible(false);
      setTwoFASetupData(null);
      twoFAVerifyForm.resetFields();
      fetch2FAStatus();
    } catch (error) {
      message.error(error.response?.data?.error || t('security.invalidCode') || '验证码无效');
    }
  };

  const handleDisable2FA = async () => {
    try {
      const values = await twoFADisableForm.validateFields();
      await securityAPI.disable2FA({ code: values.code });
      message.success(t('security.disable2FASuccess') || '2FA 已禁用');
      twoFADisableForm.resetFields();
      fetch2FAStatus();
    } catch (error) {
      message.error(error.response?.data?.error || t('security.invalidCode') || '验证码无效');
    }
  };

  const handleRegenerateRecoveryCodes = async () => {
    try {
      const res = await securityAPI.regenerateRecoveryCodes();
      const data = res.data?.data || res.data;
      setRecoveryCodes(data?.recovery_codes || []);
      setRecoveryCodesVisible(true);
    } catch (error) {
      message.error(t('security.regenerateFailed') || '重新生成恢复码失败');
    }
  };

  // 复制到剪贴板（兼容 HTTP 和 HTTPS 环境）
  const copyToClipboard = async (text) => {
    // 首先尝试现代 Clipboard API
    if (navigator.clipboard && window.isSecureContext) {
      try {
        await navigator.clipboard.writeText(text);
        message.success(t('common.copied') || '已复制');
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
        message.success(t('common.copied') || '已复制');
      } else {
        message.error(t('common.copyFailed') || '复制失败，请手动复制');
      }
    } catch (err) {
      console.error('execCommand copy failed:', err);
      message.error(t('common.copyFailed') || '复制失败，请手动复制');
    } finally {
      document.body.removeChild(textArea);
    }
  };

  // ==================== OAuth 配置 ====================
  const fetchOAuthProviders = async () => {
    setOAuthLoading(true);
    try {
      const res = await securityAPI.getOAuthProviders();
      setOAuthProviders(res.data?.data || []);
    } catch (error) {
      // 静默处理
    } finally {
      setOAuthLoading(false);
    }
  };

  const handleEditOAuth = (provider) => {
    setEditingOAuth(provider);
    oauthForm.setFieldsValue(provider);
    setOAuthModalVisible(true);
  };

  const handleOAuthSubmit = async () => {
    try {
      const values = await oauthForm.validateFields();
      await securityAPI.updateOAuthProvider(editingOAuth.id, values);
      message.success(t('security.updateSuccess') || '更新成功');
      setOAuthModalVisible(false);
      fetchOAuthProviders();
    } catch (error) {
      message.error(error.response?.data?.error || t('security.operationFailed') || '操作失败');
    }
  };

  const getProviderIcon = (name) => {
    const iconMap = {
      google: <GoogleOutlined style={{ color: '#4285F4' }} />,
      github: <GithubOutlined style={{ color: '#24292e' }} />,
      wechat: <WechatOutlined style={{ color: '#07C160' }} />,
      wechat_work: <WechatOutlined style={{ color: '#2C87F7' }} />,
      dingtalk: <DingdingOutlined style={{ color: '#0076FF' }} />,
      feishu: <CloudOutlined style={{ color: '#3370FF' }} />,
      teams: <WindowsOutlined style={{ color: '#6264A7' }} />,
      gitlab: <GitlabOutlined style={{ color: '#FC6D26' }} />,
    };
    return iconMap[name] || <CloudOutlined />;
  };

  // ==================== 安全配置 ====================
  const fetchSecurityConfig = async () => {
    setConfigLoading(true);
    try {
      const res = await securityAPI.getSecurityConfig();
      const config = res.data;
      setSecurityConfig(config);
      configForm.setFieldsValue(config);
    } catch (error) {
      // 使用默认配置
      setSecurityConfig({
        enable_ip_blacklist: true,
        enable_ip_whitelist: false,
        enable_2fa_global: false,
        max_login_attempts: 5,
        lockout_duration: 30,
        session_timeout: 480,
      });
    } finally {
      setConfigLoading(false);
    }
  };

  const handleSaveConfig = async () => {
    try {
      const values = await configForm.validateFields();
      await securityAPI.updateSecurityConfig(values);
      message.success(t('security.configSaved') || '配置已保存');
      fetchSecurityConfig();
    } catch (error) {
      message.error(t('security.configSaveFailed') || '保存配置失败');
    }
  };

  // ==================== 审计日志 ====================
  const fetchAuditLogs = async (page = 1, pageSize = 10) => {
    setAuditLoading(true);
    try {
      const res = await securityAPI.getAuditLogs({ page, page_size: pageSize });
      setAuditLogs(res.data?.data || []);
      setAuditPagination({
        current: page,
        pageSize: pageSize,
        total: res.data?.total || 0,
      });
    } catch (error) {
      // 静默处理
    } finally {
      setAuditLoading(false);
    }
  };

  // ==================== 疑似攻击IP ====================
  const fetchSuspiciousIPs = async (page = 1, pageSize = 10) => {
    setSuspiciousIPsLoading(true);
    try {
      const res = await securityAPI.getIPStats({ page, page_size: pageSize, min_risk_score: 30, order_by_risk: true });
      setSuspiciousIPs(res.data?.data || []);
      setSuspiciousIPsPagination({
        current: page,
        pageSize: pageSize,
        total: res.data?.total || 0,
      });
    } catch (error) {
      message.error(t('security.fetchSuspiciousIPsFailed') || '获取疑似攻击IP列表失败');
    } finally {
      setSuspiciousIPsLoading(false);
    }
  };

  const fetchLoginStats = async () => {
    try {
      const res = await securityAPI.getLoginStatsSummary({ hours: 24 });
      setLoginStats(res.data?.data || res.data);
    } catch (error) {
      // 静默处理
    }
  };

  const fetchLockedAccounts = async () => {
    setLockedAccountsLoading(true);
    try {
      const res = await securityAPI.getLockedAccounts({ page: 1, page_size: 100 });
      setLockedAccounts(res.data?.data || []);
    } catch (error) {
      // 静默处理
    } finally {
      setLockedAccountsLoading(false);
    }
  };

  const handleUnlockAccount = async (username) => {
    try {
      await securityAPI.unlockAccount(username);
      message.success(t('security.unlockSuccess') || '解锁成功');
      fetchLockedAccounts();
    } catch (error) {
      message.error(t('security.unlockFailed') || '解锁失败');
    }
  };

  const handleBlockIP = async (ip) => {
    try {
      await securityAPI.blockIP({ ip, reason: '手动封禁', duration_minutes: 1440 });
      message.success(t('security.blockSuccess') || '封禁成功');
      fetchSuspiciousIPs();
    } catch (error) {
      message.error(t('security.blockFailed') || '封禁失败');
    }
  };

  const handleUnblockIP = async (ip) => {
    try {
      await securityAPI.unblockIP(ip);
      message.success(t('security.unblockSuccess') || '解封成功');
      fetchSuspiciousIPs();
    } catch (error) {
      message.error(t('security.unblockFailed') || '解封失败');
    }
  };

  const suspiciousIPsColumns = [
    {
      title: 'IP',
      dataIndex: 'ip',
      key: 'ip',
      render: (ip, record) => (
        <Space>
          <Tag icon={<GlobalOutlined />} color={record.risk_score >= 70 ? 'red' : record.risk_score >= 50 ? 'orange' : 'gold'}>
            {ip}
          </Tag>
          {record.blocked_until && new Date(record.blocked_until) > new Date() && (
            <Tag color="red">{t('security.blocked') || '已封禁'}</Tag>
          )}
        </Space>
      ),
    },
    {
      title: t('security.riskScore') || '风险评分',
      dataIndex: 'risk_score',
      key: 'risk_score',
      render: (score) => {
        let color = 'green';
        if (score >= 70) color = 'red';
        else if (score >= 50) color = 'orange';
        else if (score >= 30) color = 'gold';
        return (
          <Tag color={color}>
            {score}/100
          </Tag>
        );
      },
      sorter: (a, b) => a.risk_score - b.risk_score,
    },
    {
      title: t('security.totalAttempts') || '总尝试',
      dataIndex: 'total_attempts',
      key: 'total_attempts',
    },
    {
      title: t('security.failureCount') || '失败次数',
      dataIndex: 'failure_count',
      key: 'failure_count',
      render: (count, record) => (
        <Text type={count > 5 ? 'danger' : 'secondary'}>
          {count} ({record.consecutive_fails} {t('security.consecutive') || '连续'})
        </Text>
      ),
    },
    {
      title: t('security.successCount') || '成功次数',
      dataIndex: 'success_count',
      key: 'success_count',
    },
    {
      title: t('security.lastAttempt') || '最后尝试',
      dataIndex: 'last_attempt_at',
      key: 'last_attempt_at',
      render: (time) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-',
    },
    {
      title: t('security.blockCount') || '封禁次数',
      dataIndex: 'block_count',
      key: 'block_count',
    },
    {
      title: t('common.actions') || '操作',
      key: 'actions',
      render: (_, record) => (
        <Space>
          {record.blocked_until && new Date(record.blocked_until) > new Date() ? (
            <Popconfirm
              title={t('security.confirmUnblock') || '确定解封此IP？'}
              onConfirm={() => handleUnblockIP(record.ip)}
            >
              <Button size="small" icon={<UnlockOutlined />}>
                {t('security.unblock') || '解封'}
              </Button>
            </Popconfirm>
          ) : (
            <Popconfirm
              title={t('security.confirmBlock') || '确定封禁此IP 24小时？'}
              onConfirm={() => handleBlockIP(record.ip)}
            >
              <Button size="small" danger icon={<StopOutlined />}>
                {t('security.block') || '封禁'}
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  const lockedAccountsColumns = [
    {
      title: t('security.username') || '用户名',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: t('security.failedCount') || '失败次数',
      dataIndex: 'failed_login_count',
      key: 'failed_login_count',
      render: (count) => <Tag color="red">{count}</Tag>,
    },
    {
      title: t('security.lastFailedIP') || '最后失败IP',
      dataIndex: 'last_failed_login_ip',
      key: 'last_failed_login_ip',
    },
    {
      title: t('security.lockedUntil') || '锁定到期',
      dataIndex: 'locked_until',
      key: 'locked_until',
      render: (time) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-',
    },
    {
      title: t('common.actions') || '操作',
      key: 'actions',
      render: (_, record) => (
        <Popconfirm
          title={t('security.confirmUnlockAccount') || '确定解锁此账号？'}
          onConfirm={() => handleUnlockAccount(record.username)}
        >
          <Button size="small" icon={<UnlockOutlined />}>
            {t('security.unlock') || '解锁'}
          </Button>
        </Popconfirm>
      ),
    },
  ];

  const auditColumns = [
    {
      title: t('security.action') || '操作',
      dataIndex: 'action',
      key: 'action',
      render: (action) => {
        const actionColors = {
          login: 'green',
          logout: 'blue',
          failed_login: 'red',
          enable_2fa: 'purple',
          disable_2fa: 'orange',
          ip_blocked: 'red',
        };
        return <Tag color={actionColors[action] || 'default'}>{action}</Tag>;
      },
    },
    {
      title: t('security.user') || '用户',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: 'IP',
      dataIndex: 'ip_address',
      key: 'ip_address',
    },
    {
      title: t('security.details') || '详情',
      dataIndex: 'details',
      key: 'details',
      ellipsis: true,
    },
    {
      title: t('security.time') || '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
  ];

  const tabItems = [
    {
      key: 'blacklist',
      label: (
        <span>
          <StopOutlined /> {t('security.ipBlacklist') || 'IP 黑名单'}
        </span>
      ),
      children: (
        <Card bordered={false}>
          <Space style={{ marginBottom: 16 }}>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddBlacklist}>
              {t('security.addIP') || '添加 IP'}
            </Button>
            {selectedBlacklistKeys.length > 0 && (
              <Popconfirm
                title={t('security.confirmBatchDelete') || `确定删除选中的 ${selectedBlacklistKeys.length} 条记录？`}
                onConfirm={handleBatchDeleteBlacklist}
              >
                <Button danger icon={<DeleteOutlined />}>
                  {t('security.batchDelete') || '批量删除'} ({selectedBlacklistKeys.length})
                </Button>
              </Popconfirm>
            )}
          </Space>
          <Table
            loading={blacklistLoading}
            dataSource={blacklist}
            columns={blacklistColumns}
            rowKey="id"
            rowSelection={{
              selectedRowKeys: selectedBlacklistKeys,
              onChange: setSelectedBlacklistKeys,
            }}
            pagination={{ pageSize: 10 }}
          />
        </Card>
      ),
    },
    {
      key: 'whitelist',
      label: (
        <span>
          <CheckCircleOutlined /> {t('security.ipWhitelist') || 'IP 白名单'}
        </span>
      ),
      children: (
        <Card bordered={false}>
          <Alert
            message={t('security.whitelistTip') || '白名单中的 IP 将跳过黑名单检查和登录限制'}
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddWhitelist} style={{ marginBottom: 16 }}>
            {t('security.addIP') || '添加 IP'}
          </Button>
          <Table
            loading={whitelistLoading}
            dataSource={whitelist}
            columns={whitelistColumns}
            rowKey="id"
            pagination={{ pageSize: 10 }}
          />
        </Card>
      ),
    },
    {
      key: '2fa',
      label: (
        <span>
          <KeyOutlined /> {t('security.twoFactorAuth') || '二次认证'}
        </span>
      ),
      children: (
        <Spin spinning={twoFALoading}>
          <Card bordered={false}>
            <Row gutter={[24, 24]}>
              <Col span={24}>
                <Alert
                  message={t('security.2faTip') || '二次认证可以大幅提高账户安全性'}
                  description={t('security.2faDesc') || '启用后，每次登录除了密码外，还需要输入动态验证码。支持 Google Authenticator、Microsoft Authenticator 等主流认证器。'}
                  type="info"
                  showIcon
                  icon={<SafetyOutlined />}
                />
              </Col>
              <Col span={24}>
                <Card
                  title={
                    <Space>
                      <QrcodeOutlined />
                      {t('security.totp2FA') || 'TOTP 二次认证'}
                    </Space>
                  }
                >
                  {twoFAStatus?.enabled ? (
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <Alert
                        message={t('security.2faEnabled') || '二次认证已启用'}
                        type="success"
                        showIcon
                        icon={<CheckCircleOutlined />}
                      />
                      <Divider />
                      <Form form={twoFADisableForm} layout="inline">
                        <Form.Item
                          name="code"
                          rules={[{ required: true, message: t('security.enterCode') || '请输入验证码' }]}
                        >
                          <Input placeholder={t('security.enterCode') || '输入验证码'} style={{ width: 200 }} />
                        </Form.Item>
                        <Form.Item>
                          <Popconfirm
                            title={t('security.confirmDisable2FA') || '确定禁用二次认证？'}
                            onConfirm={handleDisable2FA}
                          >
                            <Button danger icon={<UnlockOutlined />}>
                              {t('security.disable2FA') || '禁用 2FA'}
                            </Button>
                          </Popconfirm>
                        </Form.Item>
                      </Form>
                      <Divider />
                      <Button icon={<ReloadOutlined />} onClick={handleRegenerateRecoveryCodes}>
                        {t('security.regenerateRecoveryCodes') || '重新生成恢复码'}
                      </Button>
                    </Space>
                  ) : (
                    <Space direction="vertical" style={{ width: '100%' }}>
                      <Alert
                        message={t('security.2faNotEnabled') || '二次认证未启用'}
                        type="warning"
                        showIcon
                        icon={<WarningOutlined />}
                      />
                      <Button type="primary" icon={<LockOutlined />} onClick={handleSetup2FA}>
                        {t('security.setup2FA') || '设置 2FA'}
                      </Button>
                    </Space>
                  )}
                </Card>
              </Col>
            </Row>
          </Card>
        </Spin>
      ),
    },
    {
      key: 'oauth',
      label: (
        <span>
          <CloudOutlined /> {t('security.thirdPartyLogin') || '第三方登录'}
        </span>
      ),
      children: (
        <Card bordered={false}>
          <Alert
            message={t('security.oauthTip') || '配置第三方登录接入，让用户可以使用企业账号快速登录系统'}
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
          <List
            loading={oauthLoading}
            dataSource={oauthProviders}
            renderItem={(provider) => (
              <List.Item
                actions={[
                  <Switch
                    checked={provider.enabled}
                    onChange={() => handleEditOAuth({ ...provider, enabled: !provider.enabled })}
                  />,
                  <Button type="link" icon={<EditOutlined />} onClick={() => handleEditOAuth(provider)}>
                    {t('common.configure') || '配置'}
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  avatar={
                    <div style={{ fontSize: 24 }}>
                      {getProviderIcon(provider.name)}
                    </div>
                  }
                  title={
                    <Space>
                      {provider.display_name}
                      {provider.enabled ? (
                        <Tag color="green">{t('security.enabled') || '已启用'}</Tag>
                      ) : (
                        <Tag>{t('security.disabled') || '未启用'}</Tag>
                      )}
                    </Space>
                  }
                  description={provider.description || `${provider.display_name} OAuth 登录`}
                />
              </List.Item>
            )}
            locale={{ emptyText: <Empty description={t('security.noOAuthProviders') || '暂无第三方登录配置'} /> }}
          />
        </Card>
      ),
    },
    {
      key: 'config',
      label: (
        <span>
          <SafetyOutlined /> {t('security.globalConfig') || '全局配置'}
        </span>
      ),
      children: (
        <Spin spinning={configLoading}>
          <Card bordered={false}>
            <Form form={configForm} layout="vertical" style={{ maxWidth: 600 }}>
              <Divider orientation="left">{t('security.ipRestriction') || 'IP 访问限制'}</Divider>
              <Form.Item
                name="enable_ip_blacklist"
                label={t('security.enableBlacklist') || '启用 IP 黑名单'}
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <Form.Item
                name="enable_ip_whitelist"
                label={t('security.enableWhitelist') || '启用 IP 白名单（仅白名单 IP 可访问）'}
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>

              <Divider orientation="left">{t('security.loginSecurity') || '登录安全'}</Divider>
              <Form.Item
                name="enable_2fa_global"
                label={t('security.enforce2FA') || '强制所有用户启用 2FA'}
                valuePropName="checked"
              >
                <Switch />
              </Form.Item>
              <Form.Item
                name="max_login_attempts"
                label={t('security.maxLoginAttempts') || '最大登录尝试次数'}
              >
                <InputNumber min={3} max={20} style={{ width: 200 }} />
              </Form.Item>
              <Form.Item
                name="lockout_duration"
                label={t('security.lockoutDuration') || '账户锁定时长（分钟）'}
              >
                <InputNumber min={5} max={1440} style={{ width: 200 }} />
              </Form.Item>

              <Divider orientation="left">{t('security.sessionManagement') || '会话管理'}</Divider>
              <Form.Item
                name="session_timeout"
                label={t('security.sessionTimeout') || '会话超时时间（分钟）'}
              >
                <InputNumber min={15} max={10080} style={{ width: 200 }} />
              </Form.Item>

              <Form.Item>
                <Button type="primary" onClick={handleSaveConfig}>
                  {t('common.save') || '保存配置'}
                </Button>
              </Form.Item>
            </Form>
          </Card>
        </Spin>
      ),
    },
    {
      key: 'audit',
      label: (
        <span>
          <HistoryOutlined /> {t('security.auditLog') || '审计日志'}
        </span>
      ),
      children: (
        <Card bordered={false}>
          <Button icon={<ReloadOutlined />} onClick={() => fetchAuditLogs(1, 10)} style={{ marginBottom: 16 }}>
            {t('common.refresh') || '刷新'}
          </Button>
          <Table
            loading={auditLoading}
            dataSource={auditLogs}
            columns={auditColumns}
            rowKey="id"
            pagination={{
              ...auditPagination,
              onChange: (page, pageSize) => fetchAuditLogs(page, pageSize),
            }}
          />
        </Card>
      ),
    },
    {
      key: 'suspicious',
      label: (
        <span>
          <WarningOutlined /> {t('security.suspiciousIPs') || '疑似攻击IP'}
        </span>
      ),
      children: (
        <Card bordered={false}>
          <Alert
            message={t('security.suspiciousIPsDesc') || '展示风险评分较高的IP，可能存在暴力破解或恶意登录行为'}
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
          />
          
          {/* 登录统计概览 */}
          {loginStats && (
            <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
              <Col xs={12} sm={8} md={6}>
                <Card size="small">
                  <Statistic 
                    title={t('security.totalAttempts24h') || '24h登录尝试'} 
                    value={loginStats.total_attempts || 0}
                    prefix={<HistoryOutlined />}
                  />
                </Card>
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Card size="small">
                  <Statistic 
                    title={t('security.failedLogins') || '失败次数'} 
                    value={loginStats.failed_logins || 0}
                    valueStyle={{ color: '#ff4d4f' }}
                    prefix={<WarningOutlined />}
                  />
                </Card>
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Card size="small">
                  <Statistic 
                    title={t('security.lockedAccounts') || '锁定账号'} 
                    value={loginStats.locked_accounts || 0}
                    valueStyle={{ color: '#faad14' }}
                    prefix={<LockOutlined />}
                  />
                </Card>
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Card size="small">
                  <Statistic 
                    title={t('security.blockedIPs') || '封禁IP'} 
                    value={(loginStats.blocked_ips || 0) + (loginStats.auto_blocked_ips || 0)}
                    valueStyle={{ color: '#ff4d4f' }}
                    prefix={<StopOutlined />}
                  />
                </Card>
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Card size="small">
                  <Statistic 
                    title={t('security.highRiskIPs') || '高风险IP'} 
                    value={loginStats.high_risk_ips || 0}
                    valueStyle={{ color: '#ff4d4f' }}
                    prefix={<WarningOutlined />}
                  />
                </Card>
              </Col>
              <Col xs={12} sm={8} md={6}>
                <Card size="small">
                  <Statistic 
                    title={t('security.uniqueIPs') || '独立IP数'} 
                    value={loginStats.unique_ips || 0}
                    prefix={<GlobalOutlined />}
                  />
                </Card>
              </Col>
            </Row>
          )}

          <Space style={{ marginBottom: 16 }}>
            <Button 
              icon={<ReloadOutlined />} 
              onClick={() => {
                fetchSuspiciousIPs();
                fetchLoginStats();
                fetchLockedAccounts();
              }}
            >
              {t('common.refresh') || '刷新'}
            </Button>
          </Space>

          <Divider orientation="left">{t('security.suspiciousIPList') || '疑似攻击IP列表'}</Divider>
          <Table
            loading={suspiciousIPsLoading}
            dataSource={suspiciousIPs}
            columns={suspiciousIPsColumns}
            rowKey="id"
            pagination={{
              ...suspiciousIPsPagination,
              onChange: (page, pageSize) => fetchSuspiciousIPs(page, pageSize),
            }}
          />

          <Divider orientation="left">{t('security.lockedAccountsList') || '已锁定账号'}</Divider>
          <Table
            loading={lockedAccountsLoading}
            dataSource={lockedAccounts}
            columns={lockedAccountsColumns}
            rowKey="id"
            pagination={{ pageSize: 5 }}
            locale={{ emptyText: <Empty description={t('security.noLockedAccounts') || '暂无锁定账号'} /> }}
          />
        </Card>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px', background: isDark ? '#141414' : '#f0f2f5', minHeight: '100vh' }}>
      <div style={{ marginBottom: '24px' }}>
        <Button 
          icon={<ArrowLeftOutlined />} 
          onClick={() => navigate('/admin')}
          style={{ marginBottom: 16 }}
        >
          {t('common.back') || '返回'}
        </Button>
        <Title level={2} style={{ color: isDark ? 'rgba(255, 255, 255, 0.85)' : 'inherit' }}>
          <SecurityScanOutlined style={{ marginRight: '8px' }} />
          {t('security.title') || '安全管理'}
        </Title>
        <Paragraph style={{ fontSize: '16px', color: isDark ? 'rgba(255, 255, 255, 0.45)' : '#666' }}>
          {t('security.description') || '配置系统安全策略，包括 IP 访问控制、二次认证、第三方登录等'}
        </Paragraph>
      </div>

      <Card style={{ background: isDark ? '#1f1f1f' : '#fff' }}>
        <Tabs
          defaultActiveKey="blacklist"
          items={tabItems}
          onChange={(key) => {
            if (key === 'audit') {
              fetchAuditLogs(1, 10);
            } else if (key === 'suspicious') {
              fetchSuspiciousIPs();
              fetchLoginStats();
              fetchLockedAccounts();
            }
          }}
        />
      </Card>

      {/* IP 黑名单编辑模态框 */}
      <Modal
        title={editingBlacklist ? (t('security.editBlacklist') || '编辑黑名单') : (t('security.addBlacklist') || '添加黑名单')}
        open={blacklistModalVisible}
        onOk={handleBlacklistSubmit}
        onCancel={() => setBlacklistModalVisible(false)}
        destroyOnClose
      >
        <Form form={blacklistForm} layout="vertical">
          <Form.Item
            name="ip_address"
            label={t('security.ipOrCIDR') || 'IP 地址或 CIDR'}
            rules={[{ required: true, message: t('security.enterIP') || '请输入 IP 地址' }]}
          >
            <Input placeholder="例如: 192.168.1.100 或 10.0.0.0/24" />
          </Form.Item>
          <Form.Item
            name="reason"
            label={t('security.reason') || '封禁原因'}
          >
            <TextArea rows={2} placeholder={t('security.reasonPlaceholder') || '说明添加此 IP 的原因'} />
          </Form.Item>
          <Form.Item
            name="type"
            label={t('security.type') || '类型'}
            initialValue="manual"
          >
            <Select>
              <Option value="manual">{t('security.manual') || '手动添加'}</Option>
              <Option value="import">{t('security.import') || '批量导入'}</Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="enabled"
            label={t('security.status') || '状态'}
            valuePropName="checked"
            initialValue={true}
          >
            <Switch checkedChildren={t('security.enabled') || '启用'} unCheckedChildren={t('security.disabled') || '禁用'} />
          </Form.Item>
        </Form>
      </Modal>

      {/* IP 白名单添加模态框 */}
      <Modal
        title={t('security.addWhitelist') || '添加白名单'}
        open={whitelistModalVisible}
        onOk={handleWhitelistSubmit}
        onCancel={() => setWhitelistModalVisible(false)}
        destroyOnClose
      >
        <Form form={whitelistForm} layout="vertical">
          <Form.Item
            name="ip_address"
            label={t('security.ipOrCIDR') || 'IP 地址或 CIDR'}
            rules={[{ required: true, message: t('security.enterIP') || '请输入 IP 地址' }]}
          >
            <Input placeholder="例如: 192.168.1.100 或 10.0.0.0/24" />
          </Form.Item>
          <Form.Item
            name="description"
            label={t('security.description') || '描述'}
          >
            <TextArea rows={2} placeholder={t('security.descriptionPlaceholder') || '说明此 IP 的用途'} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 2FA 设置模态框 */}
      <Modal
        title={t('security.setup2FA') || '设置二次认证'}
        open={twoFASetupVisible}
        onCancel={() => {
          setTwoFASetupVisible(false);
          setTwoFASetupData(null);
          twoFAVerifyForm.resetFields();
        }}
        footer={null}
        width={500}
      >
        {twoFASetupData && (
          <Space direction="vertical" style={{ width: '100%' }} size="large">
            <Alert
              message={t('security.scanQRCode') || '请使用认证器扫描二维码'}
              description={t('security.scanQRCodeDesc') || '支持 Google Authenticator、Microsoft Authenticator、1Password 等应用'}
              type="info"
              showIcon
            />
            <div style={{ textAlign: 'center' }}>
              <QRCode value={twoFASetupData.url} size={200} />
            </div>
            <Alert
              message={t('security.manualEntry') || '或手动输入密钥'}
              description={
                <Space>
                  <Text code copyable>{twoFASetupData.secret}</Text>
                </Space>
              }
              type="warning"
            />
            <Divider />
            <Form form={twoFAVerifyForm} onFinish={handleEnable2FA}>
              <Form.Item
                name="code"
                label={t('security.verificationCode') || '验证码'}
                rules={[
                  { required: true, message: t('security.enterCode') || '请输入验证码' },
                  { len: 6, message: t('security.codeLength') || '验证码为6位数字' }
                ]}
              >
                <Input placeholder="000000" style={{ width: '100%' }} maxLength={6} />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" block icon={<CheckCircleOutlined />}>
                  {t('security.verifyAndEnable') || '验证并启用'}
                </Button>
              </Form.Item>
            </Form>
          </Space>
        )}
      </Modal>

      {/* 恢复码显示模态框 */}
      <Modal
        title={t('security.recoveryCodes') || '恢复码'}
        open={recoveryCodesVisible}
        onCancel={() => setRecoveryCodesVisible(false)}
        footer={[
          <Button key="copy" icon={<CopyOutlined />} onClick={() => copyToClipboard(recoveryCodes.join('\n'))}>
            {t('common.copyAll') || '复制全部'}
          </Button>,
          <Button key="close" type="primary" onClick={() => setRecoveryCodesVisible(false)}>
            {t('common.close') || '关闭'}
          </Button>,
        ]}
      >
        <Alert
          message={t('security.saveRecoveryCodes') || '请妥善保存这些恢复码'}
          description={t('security.recoveryCodesDesc') || '每个恢复码只能使用一次。当您无法访问认证器时，可以使用恢复码登录。'}
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <List
          dataSource={recoveryCodes}
          renderItem={(code, index) => (
            <List.Item>
              <Text code>{index + 1}. {code}</Text>
              <Button type="link" icon={<CopyOutlined />} onClick={() => copyToClipboard(code)} />
            </List.Item>
          )}
        />
      </Modal>

      {/* OAuth 配置模态框 */}
      <Modal
        title={t('security.configureOAuth') || '配置第三方登录'}
        open={oauthModalVisible}
        onOk={handleOAuthSubmit}
        onCancel={() => setOAuthModalVisible(false)}
        destroyOnClose
        width={600}
      >
        <Form form={oauthForm} layout="vertical">
          <Form.Item
            name="display_name"
            label={t('security.displayName') || '显示名称'}
          >
            <Input disabled />
          </Form.Item>
          <Form.Item
            name="enabled"
            label={t('security.status') || '状态'}
            valuePropName="checked"
          >
            <Switch checkedChildren={t('security.enabled') || '启用'} unCheckedChildren={t('security.disabled') || '禁用'} />
          </Form.Item>
          <Form.Item
            name="client_id"
            label="Client ID"
            rules={[{ required: true, message: t('security.enterClientId') || '请输入 Client ID' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="client_secret"
            label="Client Secret"
            rules={[{ required: true, message: t('security.enterClientSecret') || '请输入 Client Secret' }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="redirect_url"
            label={t('security.redirectUrl') || '回调地址'}
          >
            <Input placeholder="https://your-domain.com/api/auth/oauth/callback/{provider}" />
          </Form.Item>
          <Form.Item
            name="scopes"
            label={t('security.scopes') || '权限范围'}
          >
            <Input placeholder="openid profile email" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default SecuritySettings;
