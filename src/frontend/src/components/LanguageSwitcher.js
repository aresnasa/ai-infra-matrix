/**
 * 语言切换组件
 * 提供下拉菜单切换语言
 */

import React from 'react';
import { Dropdown, Button, Space, Typography } from 'antd';
import { CheckOutlined } from '@ant-design/icons';
import { useI18n } from '../hooks/useI18n';

const { Text } = Typography;

/**
 * 网格地球图标 (Grid Globe)
 */
const GridGlobeIcon = ({ style = {} }) => (
  <svg 
    viewBox="0 0 24 24" 
    width="1em" 
    height="1em" 
    fill="currentColor"
    style={style}
  >
    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/>
    <circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" strokeWidth="0.5"/>
    <path d="M12 2v20M2 12h20" stroke="currentColor" strokeWidth="0.3" fill="none"/>
    <ellipse cx="12" cy="12" rx="10" ry="4" fill="none" stroke="currentColor" strokeWidth="0.3" transform="rotate(0)"/>
  </svg>
);

/**
 * 语言切换下拉组件
 * @param {object} props
 * @param {string} props.size - 按钮大小 'small' | 'middle' | 'large'
 * @param {boolean} props.showLabel - 是否显示当前语言名称
 * @param {string} props.placement - 下拉菜单位置
 * @param {boolean} props.darkMode - 是否深色模式（用于深色背景）
 */
const LanguageSwitcher = ({
  size = 'middle',
  showLabel = true,
  placement = 'bottomRight',
  darkMode = false,
}) => {
  const { locale, setLocale, availableLocales } = useI18n();

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

  // 按钮样式
  const buttonStyle = darkMode 
    ? { 
        color: '#fff',
        height: '64px',
        padding: '0 12px',
      }
    : {};

  return (
    <Dropdown
      menu={{ items: menuItems }}
      placement={placement}
      trigger={['click']}
    >
      <Button size={size} type="text" style={buttonStyle}>
        <Space>
          <GridGlobeIcon style={{ fontSize: '16px' }} />
          {showLabel && (
            <Text style={darkMode ? { color: '#fff' } : {}}>
              {currentLanguageName}
            </Text>
          )}
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
