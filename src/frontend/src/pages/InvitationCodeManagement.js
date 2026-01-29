import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  InputNumber,
  DatePicker,
  message,
  Tag,
  Popconfirm,
  Typography,
  Row,
  Col,
  Statistic,
  Tooltip,
  Divider,
  Switch,
  Alert,
} from 'antd';
import {
  PlusOutlined,
  CopyOutlined,
  DeleteOutlined,
  ReloadOutlined,
  StopOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  KeyOutlined,
  BarChartOutlined,
  UserOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { invitationCodeAPI } from '../services/api';
import { useI18n } from '../hooks/useI18n';
import { useTheme } from '../hooks/useTheme';

const { Title, Text, Paragraph } = Typography;
const { Option } = Select;

const InvitationCodeManagement = () => {
  const { t } = useI18n();
  const { isDark } = useTheme();
  const [form] = Form.useForm();
  
  // State
  const [loading, setLoading] = useState(false);
  const [codes, setCodes] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [includeExpired, setIncludeExpired] = useState(false);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [creating, setCreating] = useState(false);
  const [statistics, setStatistics] = useState(null);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedCode, setSelectedCode] = useState(null);
  const [usageRecords, setUsageRecords] = useState([]);

  // 获取邀请码列表
  const fetchCodes = useCallback(async () => {
    setLoading(true);
    try {
      const response = await invitationCodeAPI.list({
        page,
        page_size: pageSize,
        include_expired: includeExpired,
      });
      if (response.data) {
        setCodes(response.data.data || []);
        setTotal(response.data.total || 0);
      }
    } catch (error) {
      console.error('获取邀请码列表失败:', error);
      message.error(t('invitationCode.fetchFailed'));
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, includeExpired, t]);

  // 获取统计数据
  const fetchStatistics = useCallback(async () => {
    try {
      const response = await invitationCodeAPI.getStatistics();
      if (response.data) {
        setStatistics(response.data);
      }
    } catch (error) {
      console.error('获取统计数据失败:', error);
    }
  }, []);

  useEffect(() => {
    fetchCodes();
    fetchStatistics();
  }, [fetchCodes, fetchStatistics]);

  // 创建邀请码
  const handleCreate = async (values) => {
    setCreating(true);
    try {
      // 处理过期时间
      const data = {
        description: values.description,
        role_template: values.role_template,
        max_uses: values.max_uses || 1,
        count: values.count || 1,
      };
      
      if (values.expires_at) {
        data.expires_at = values.expires_at.toISOString();
      }

      const response = await invitationCodeAPI.create(data);
      
      if (response.data) {
        message.success(t('invitationCode.createSuccess'));
        setCreateModalVisible(false);
        form.resetFields();
        fetchCodes();
        fetchStatistics();
        
        // 如果是单个邀请码，显示复制提示
        if (values.count === 1 && response.data.code) {
          Modal.success({
            title: t('invitationCode.createSuccess'),
            content: (
              <div>
                <p>{t('invitationCode.codeGenerated')}:</p>
                <Input.Group compact style={{ marginTop: 8 }}>
                  <Input
                    style={{ width: 'calc(100% - 80px)' }}
                    value={response.data.code.code}
                    readOnly
                  />
                  <Button
                    type="primary"
                    icon={<CopyOutlined />}
                    onClick={() => {
                      navigator.clipboard.writeText(response.data.code.code);
                      message.success(t('common.copySuccess'));
                    }}
                  >
                    {t('common.copy')}
                  </Button>
                </Input.Group>
              </div>
            ),
          });
        }
      }
    } catch (error) {
      console.error('创建邀请码失败:', error);
      message.error(error.response?.data?.error || t('invitationCode.createFailed'));
    } finally {
      setCreating(false);
    }
  };

  // 禁用邀请码
  const handleDisable = async (id) => {
    try {
      await invitationCodeAPI.disable(id);
      message.success(t('invitationCode.disableSuccess'));
      fetchCodes();
      fetchStatistics();
    } catch (error) {
      message.error(t('invitationCode.disableFailed'));
    }
  };

  // 启用邀请码
  const handleEnable = async (id) => {
    try {
      await invitationCodeAPI.enable(id);
      message.success(t('invitationCode.enableSuccess'));
      fetchCodes();
      fetchStatistics();
    } catch (error) {
      message.error(t('invitationCode.enableFailed'));
    }
  };

  // 删除邀请码
  const handleDelete = async (id) => {
    try {
      await invitationCodeAPI.delete(id);
      message.success(t('invitationCode.deleteSuccess'));
      fetchCodes();
      fetchStatistics();
    } catch (error) {
      message.error(t('invitationCode.deleteFailed'));
    }
  };

  // 查看详情
  const handleViewDetail = async (record) => {
    setSelectedCode(record);
    setDetailModalVisible(true);
    
    try {
      const response = await invitationCodeAPI.get(record.id);
      if (response.data) {
        setSelectedCode(response.data.code);
        setUsageRecords(response.data.usages || []);
      }
    } catch (error) {
      console.error('获取邀请码详情失败:', error);
    }
  };

  // 复制邀请码
  const handleCopy = (code) => {
    navigator.clipboard.writeText(code);
    message.success(t('common.copySuccess'));
  };

  // 复制全部邀请码
  const handleCopyAll = () => {
    // 只复制有效的邀请码（未过期、未禁用、未用完）
    const validCodes = codes.filter(item => {
      if (!item.is_active) return false;
      if (item.expires_at && dayjs(item.expires_at).isBefore(dayjs())) return false;
      if (item.max_uses > 0 && item.used_count >= item.max_uses) return false;
      return true;
    });
    
    if (validCodes.length === 0) {
      message.warning(t('invitationCode.noValidCodes'));
      return;
    }
    
    const codesText = validCodes.map(item => item.code).join('\n');
    navigator.clipboard.writeText(codesText);
    message.success(t('invitationCode.copyAllSuccess', { count: validCodes.length }));
  };

  // 获取状态标签
  const getStatusTag = (record) => {
    if (!record.is_active) {
      return <Tag color="default">{t('invitationCode.disabled')}</Tag>;
    }
    if (record.expires_at && dayjs(record.expires_at).isBefore(dayjs())) {
      return <Tag color="red">{t('invitationCode.expired')}</Tag>;
    }
    if (record.max_uses > 0 && record.used_count >= record.max_uses) {
      return <Tag color="orange">{t('invitationCode.usedUp')}</Tag>;
    }
    return <Tag color="green">{t('invitationCode.valid')}</Tag>;
  };

  // 角色模板映射
  const roleTemplateLabels = {
    admin: t('invitationCode.roleAdmin'),
    'data-developer': t('invitationCode.roleDataDeveloper'),
    'model-developer': t('invitationCode.roleModelDeveloper'),
    sre: t('invitationCode.roleSRE'),
    engineer: t('invitationCode.roleEngineer'),
  };

  // 表格列定义
  const columns = [
    {
      title: t('invitationCode.code'),
      dataIndex: 'code',
      key: 'code',
      width: 200,
      render: (code) => (
        <Space>
          <Text code copyable={{ onCopy: () => message.success(t('common.copySuccess')) }}>
            {code}
          </Text>
        </Space>
      ),
    },
    {
      title: t('invitationCode.status'),
      key: 'status',
      width: 100,
      render: (_, record) => getStatusTag(record),
    },
    {
      title: t('invitationCode.roleTemplate'),
      dataIndex: 'role_template',
      key: 'role_template',
      width: 120,
      render: (role) => (
        <Tag color="blue">{roleTemplateLabels[role] || role || t('invitationCode.noPreset')}</Tag>
      ),
    },
    {
      title: t('invitationCode.usageCount'),
      key: 'usage',
      width: 120,
      render: (_, record) => (
        <span>
          {record.used_count} / {record.max_uses === 0 ? '∞' : record.max_uses}
        </span>
      ),
    },
    {
      title: t('invitationCode.expiresAt'),
      dataIndex: 'expires_at',
      key: 'expires_at',
      width: 160,
      render: (time) => time ? dayjs(time).format('YYYY-MM-DD HH:mm') : t('invitationCode.neverExpire'),
    },
    {
      title: t('invitationCode.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('invitationCode.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (time) => dayjs(time).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 200,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Tooltip title={t('common.copy')}>
            <Button
              type="text"
              size="small"
              icon={<CopyOutlined />}
              onClick={() => handleCopy(record.code)}
            />
          </Tooltip>
          <Tooltip title={t('common.details')}>
            <Button
              type="text"
              size="small"
              icon={<UserOutlined />}
              onClick={() => handleViewDetail(record)}
            />
          </Tooltip>
          {record.is_active ? (
            <Popconfirm
              title={t('invitationCode.confirmDisable')}
              onConfirm={() => handleDisable(record.id)}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
            >
              <Tooltip title={t('invitationCode.disable')}>
                <Button
                  type="text"
                  size="small"
                  danger
                  icon={<StopOutlined />}
                />
              </Tooltip>
            </Popconfirm>
          ) : (
            <Tooltip title={t('invitationCode.enable')}>
              <Button
                type="text"
                size="small"
                icon={<CheckCircleOutlined />}
                onClick={() => handleEnable(record.id)}
                style={{ color: '#52c41a' }}
              />
            </Tooltip>
          )}
          <Popconfirm
            title={t('invitationCode.confirmDelete')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
          >
            <Tooltip title={t('common.delete')}>
              <Button
                type="text"
                size="small"
                danger
                icon={<DeleteOutlined />}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px', background: isDark ? '#141414' : '#f0f2f5', minHeight: '100vh' }}>
      {/* 页面标题 */}
      <div style={{ marginBottom: '24px' }}>
        <Title level={2} style={{ color: isDark ? 'rgba(255, 255, 255, 0.85)' : 'inherit' }}>
          <KeyOutlined style={{ marginRight: '8px' }} />
          {t('invitationCode.title')}
        </Title>
        <Paragraph style={{ color: isDark ? 'rgba(255, 255, 255, 0.45)' : '#666' }}>
          {t('invitationCode.description')}
        </Paragraph>
      </div>

      {/* 统计卡片 */}
      {statistics && (
        <Row gutter={16} style={{ marginBottom: '24px' }}>
          <Col xs={12} sm={6}>
            <Card style={{ background: isDark ? '#1f1f1f' : '#fff' }}>
              <Statistic
                title={t('invitationCode.totalCodes')}
                value={statistics.total || 0}
                prefix={<KeyOutlined />}
                valueStyle={{ color: '#1890ff' }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card style={{ background: isDark ? '#1f1f1f' : '#fff' }}>
              <Statistic
                title={t('invitationCode.activeCodes')}
                value={statistics.active || 0}
                prefix={<CheckCircleOutlined />}
                valueStyle={{ color: '#52c41a' }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card style={{ background: isDark ? '#1f1f1f' : '#fff' }}>
              <Statistic
                title={t('invitationCode.usedCount')}
                value={statistics.total_used || 0}
                prefix={<UserOutlined />}
                valueStyle={{ color: '#722ed1' }}
              />
            </Card>
          </Col>
          <Col xs={12} sm={6}>
            <Card style={{ background: isDark ? '#1f1f1f' : '#fff' }}>
              <Statistic
                title={t('invitationCode.expiredCodes')}
                value={statistics.expired || 0}
                prefix={<ClockCircleOutlined />}
                valueStyle={{ color: '#fa8c16' }}
              />
            </Card>
          </Col>
        </Row>
      )}

      {/* 主内容卡片 */}
      <Card
        style={{ background: isDark ? '#1f1f1f' : '#fff' }}
        title={
          <Space>
            <BarChartOutlined />
            {t('invitationCode.codeList')}
          </Space>
        }
        extra={
          <Space>
            <span style={{ marginRight: 8 }}>{t('invitationCode.showExpired')}:</span>
            <Switch
              checked={includeExpired}
              onChange={setIncludeExpired}
              size="small"
            />
            <Divider type="vertical" />
            <Button icon={<ReloadOutlined />} onClick={fetchCodes}>
              {t('common.refresh')}
            </Button>
            <Button
              icon={<CopyOutlined />}
              onClick={handleCopyAll}
              disabled={codes.length === 0}
            >
              {t('invitationCode.copyAll')}
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateModalVisible(true)}
            >
              {t('invitationCode.create')}
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={codes}
          rowKey="id"
          loading={loading}
          scroll={{ x: 1200 }}
          pagination={{
            current: page,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => t('admin.total', { count: total }),
            onChange: (p, ps) => {
              setPage(p);
              setPageSize(ps);
            },
          }}
        />
      </Card>

      {/* 创建邀请码弹窗 */}
      <Modal
        title={t('invitationCode.create')}
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false);
          form.resetFields();
        }}
        footer={null}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreate}
          initialValues={{
            max_uses: 1,
            count: 1,
          }}
        >
          <Alert
            message={t('invitationCode.createTip')}
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
          
          <Form.Item
            name="description"
            label={t('invitationCode.descriptionLabel')}
            rules={[{ max: 255, message: t('invitationCode.descriptionTooLong') }]}
          >
            <Input.TextArea
              placeholder={t('invitationCode.descriptionPlaceholder')}
              rows={2}
            />
          </Form.Item>

          <Form.Item
            name="role_template"
            label={t('invitationCode.roleTemplate')}
          >
            <Select placeholder={t('invitationCode.selectRole')} allowClear>
              <Option value="admin">{t('invitationCode.roleAdmin')}</Option>
              <Option value="data-developer">{t('invitationCode.roleDataDeveloper')}</Option>
              <Option value="model-developer">{t('invitationCode.roleModelDeveloper')}</Option>
              <Option value="sre">{t('invitationCode.roleSRE')}</Option>
              <Option value="engineer">{t('invitationCode.roleEngineer')}</Option>
            </Select>
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="max_uses"
                label={t('invitationCode.maxUses')}
                tooltip={t('invitationCode.maxUsesTooltip')}
              >
                <InputNumber
                  min={0}
                  max={1000}
                  style={{ width: '100%' }}
                  placeholder="0 = 无限制"
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="count"
                label={t('invitationCode.batchCount')}
                tooltip={t('invitationCode.batchCountTooltip')}
              >
                <InputNumber
                  min={1}
                  max={100}
                  style={{ width: '100%' }}
                />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="expires_at"
            label={t('invitationCode.expiresAt')}
          >
            <DatePicker
              showTime
              style={{ width: '100%' }}
              placeholder={t('invitationCode.selectExpireTime')}
              disabledDate={(current) => current && current < dayjs().startOf('day')}
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setCreateModalVisible(false)}>
                {t('common.cancel')}
              </Button>
              <Button type="primary" htmlType="submit" loading={creating}>
                {t('common.create')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 详情弹窗 */}
      <Modal
        title={t('invitationCode.detail')}
        open={detailModalVisible}
        onCancel={() => {
          setDetailModalVisible(false);
          setSelectedCode(null);
          setUsageRecords([]);
        }}
        footer={[
          <Button key="close" onClick={() => setDetailModalVisible(false)}>
            {t('common.close')}
          </Button>,
        ]}
        width={600}
      >
        {selectedCode && (
          <div>
            <Row gutter={[16, 16]}>
              <Col span={24}>
                <Text strong>{t('invitationCode.code')}: </Text>
                <Text code copyable>{selectedCode.code}</Text>
              </Col>
              <Col span={12}>
                <Text strong>{t('invitationCode.status')}: </Text>
                {getStatusTag(selectedCode)}
              </Col>
              <Col span={12}>
                <Text strong>{t('invitationCode.roleTemplate')}: </Text>
                <Tag color="blue">
                  {roleTemplateLabels[selectedCode.role_template] || selectedCode.role_template || t('invitationCode.noPreset')}
                </Tag>
              </Col>
              <Col span={12}>
                <Text strong>{t('invitationCode.usageCount')}: </Text>
                {selectedCode.used_count} / {selectedCode.max_uses === 0 ? '∞' : selectedCode.max_uses}
              </Col>
              <Col span={12}>
                <Text strong>{t('invitationCode.expiresAt')}: </Text>
                {selectedCode.expires_at 
                  ? dayjs(selectedCode.expires_at).format('YYYY-MM-DD HH:mm')
                  : t('invitationCode.neverExpire')
                }
              </Col>
              <Col span={24}>
                <Text strong>{t('invitationCode.description')}: </Text>
                {selectedCode.description || '-'}
              </Col>
              <Col span={24}>
                <Text strong>{t('invitationCode.createdAt')}: </Text>
                {dayjs(selectedCode.created_at).format('YYYY-MM-DD HH:mm:ss')}
              </Col>
            </Row>

            <Divider>{t('invitationCode.usageRecords')}</Divider>

            {usageRecords.length > 0 ? (
              <Table
                dataSource={usageRecords}
                rowKey="id"
                size="small"
                pagination={false}
                columns={[
                  {
                    title: t('invitationCode.usedBy'),
                    dataIndex: ['user', 'username'],
                    key: 'username',
                    render: (_, record) => record.user?.username || '-',
                  },
                  {
                    title: t('invitationCode.usedAt'),
                    dataIndex: 'used_at',
                    key: 'used_at',
                    render: (time) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
                  },
                  {
                    title: 'IP',
                    dataIndex: 'ip_address',
                    key: 'ip_address',
                  },
                ]}
              />
            ) : (
              <Alert
                message={t('invitationCode.noUsageRecords')}
                type="info"
                showIcon
              />
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

export default InvitationCodeManagement;
