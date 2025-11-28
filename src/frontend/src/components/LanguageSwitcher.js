/**
 * 语言切换组件
 * 提供下拉菜单切换语言
 */

import React from 'react';
import { Dropdown, Button, Space, Typography } from 'antd';
import { GlobalOutlined, CheckOutlined } from '@ant-design/icons';
import { useI18n } from '../hooks/useI18n';

const { Text } = Typography;

/**
 * 语言切换下拉组件
 * @param {object} props
 * @param {string} props.size - 按钮大小 'small' | 'middle' | 'large'
 * @param {boolean} props.showLabel - 是否显示当前语言名称
 * @param {string} props.placement - 下拉菜单位置
 */
const LanguageSwitcher = ({
  size = 'middle',
  showLabel = true,
  placement = 'bottomRight',
}) => {
  const { locale, setLocale, availableLocales, t } = useI18n();

  // 获取当前语言名称
  const currentLanguageName = availableLocales.find(l => l.key === locale)?.name || locale;

  // 下拉菜单项
  const menuItems = availableLocales.map(lang => ({
    key: lang.key,
    label: (
      <Space>
        {locale === lang.key && <CheckOutlined style={{ color: '#1890ff' }} />}
        <span style={{ marginLeft: locale === lang.key ? 0 : 18 }}>{lang.name}</span>
      </Space>
    ),
    onClick: () => setLocale(lang.key),
  }));

  return (
    <Dropdown
      menu={{ items: menuItems }}
      placement={placement}
      trigger={['click']}
    >
      <Button size={size} type="text">
        <Space>
          <GlobalOutlined />
          {showLabel && <Text>{currentLanguageName}</Text>}
        </Space>
      </Button>
    </Dropdown>
  );
};

/**
 * 紧凑型语言切换器（仅图标）
 */
export const CompactLanguageSwitcher = ({ size = 'small' }) => {
  return <LanguageSwitcher size={size} showLabel={false} />;
};

/**
 * 内联语言切换器（用于设置页面）
 */
export const InlineLanguageSwitcher = () => {
  const { locale, setLocale, availableLocales, t } = useI18n();

  return (
    <Space direction="vertical" size="small" style={{ width: '100%' }}>
      <Text strong>{t('language.switchTo')}</Text>
      <Space wrap>
        {availableLocales.map(lang => (
          <Button
            key={lang.key}
            type={locale === lang.key ? 'primary' : 'default'}
            onClick={() => setLocale(lang.key)}
            icon={locale === lang.key ? <CheckOutlined /> : null}
          >
            {lang.name}
          </Button>
        ))}
      </Space>
    </Space>
  );
};

export default LanguageSwitcher;
