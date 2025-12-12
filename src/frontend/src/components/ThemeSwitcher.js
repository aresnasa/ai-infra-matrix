/**
 * 主题切换组件
 * 提供下拉菜单切换主题模式（光明/暗黑/跟随系统）
 */

import React from 'react';
import { Dropdown, Button, Space, Typography, Tooltip } from 'antd';
import { CheckOutlined } from '@ant-design/icons';
import { useTheme, ThemeMode } from '../hooks/useTheme';
import { useI18n } from '../hooks/useI18n';

const { Text } = Typography;

/**
 * 太阳图标 (光明模式)
 */
const SunIcon = ({ style = {} }) => (
  <svg 
    viewBox="0 0 24 24" 
    width="1em" 
    height="1em" 
    fill="currentColor"
    style={style}
  >
    <path d="M12 7c-2.76 0-5 2.24-5 5s2.24 5 5 5 5-2.24 5-5-2.24-5-5-5zM2 13h2c.55 0 1-.45 1-1s-.45-1-1-1H2c-.55 0-1 .45-1 1s.45 1 1 1zm18 0h2c.55 0 1-.45 1-1s-.45-1-1-1h-2c-.55 0-1 .45-1 1s.45 1 1 1zM11 2v2c0 .55.45 1 1 1s1-.45 1-1V2c0-.55-.45-1-1-1s-1 .45-1 1zm0 18v2c0 .55.45 1 1 1s1-.45 1-1v-2c0-.55-.45-1-1-1s-1 .45-1 1zM5.99 4.58c-.39-.39-1.03-.39-1.41 0-.39.39-.39 1.03 0 1.41l1.06 1.06c.39.39 1.03.39 1.41 0s.39-1.03 0-1.41L5.99 4.58zm12.37 12.37c-.39-.39-1.03-.39-1.41 0-.39.39-.39 1.03 0 1.41l1.06 1.06c.39.39 1.03.39 1.41 0 .39-.39.39-1.03 0-1.41l-1.06-1.06zm1.06-10.96c.39-.39.39-1.03 0-1.41-.39-.39-1.03-.39-1.41 0l-1.06 1.06c-.39.39-.39 1.03 0 1.41s1.03.39 1.41 0l1.06-1.06zM7.05 18.36c.39-.39.39-1.03 0-1.41-.39-.39-1.03-.39-1.41 0l-1.06 1.06c-.39.39-.39 1.03 0 1.41s1.03.39 1.41 0l1.06-1.06z"/>
  </svg>
);

/**
 * 月亮图标 (暗黑模式)
 */
const MoonIcon = ({ style = {} }) => (
  <svg 
    viewBox="0 0 24 24" 
    width="1em" 
    height="1em" 
    fill="currentColor"
    style={style}
  >
    <path d="M12 3c-4.97 0-9 4.03-9 9s4.03 9 9 9 9-4.03 9-9c0-.46-.04-.92-.1-1.36-.98 1.37-2.58 2.26-4.4 2.26-2.98 0-5.4-2.42-5.4-5.4 0-1.81.89-3.42 2.26-4.4-.44-.06-.9-.1-1.36-.1z"/>
  </svg>
);

/**
 * 系统图标 (跟随系统)
 */
const SystemIcon = ({ style = {} }) => (
  <svg 
    viewBox="0 0 24 24" 
    width="1em" 
    height="1em" 
    fill="currentColor"
    style={style}
  >
    <path d="M20 3H4c-1.1 0-2 .9-2 2v11c0 1.1.9 2 2 2h3l-1 1v2h12v-2l-1-1h3c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 13H4V5h16v11z"/>
  </svg>
);

/**
 * 获取主题模式对应的图标
 */
const getThemeIcon = (mode, isDark) => {
  switch (mode) {
    case ThemeMode.LIGHT:
      return <SunIcon />;
    case ThemeMode.DARK:
      return <MoonIcon />;
    case ThemeMode.SYSTEM:
    default:
      return <SystemIcon />;
  }
};

/**
 * 主题切换下拉组件
 * @param {object} props
 * @param {string} props.size - 按钮大小 'small' | 'middle' | 'large'
 * @param {boolean} props.showLabel - 是否显示当前主题名称
 * @param {string} props.placement - 下拉菜单位置
 * @param {boolean} props.darkMode - 是否深色背景（用于Header）
 */
const ThemeSwitcher = ({
  size = 'middle',
  showLabel = true,
  placement = 'bottomRight',
  darkMode = false,
}) => {
  const { themeMode, setThemeMode, isDark } = useTheme();
  const { t } = useI18n();

  // 主题选项
  const themeOptions = [
    { key: ThemeMode.LIGHT, name: t('theme.light'), icon: <SunIcon /> },
    { key: ThemeMode.DARK, name: t('theme.dark'), icon: <MoonIcon /> },
    { key: ThemeMode.SYSTEM, name: t('theme.system'), icon: <SystemIcon /> },
  ];

  // 获取当前主题名称
  const currentThemeName = themeOptions.find(opt => opt.key === themeMode)?.name || themeMode;

  // 下拉菜单项
  const menuItems = themeOptions.map(option => ({
    key: option.key,
    label: (
      <Space>
        {themeMode === option.key && <CheckOutlined style={{ color: '#1890ff' }} />}
        <span style={{ marginLeft: themeMode === option.key ? 0 : 18 }}>
          {option.icon}
        </span>
        <span>{option.name}</span>
      </Space>
    ),
    onClick: () => setThemeMode(option.key),
  }));

  // 按钮样式
  const buttonStyle = darkMode 
    ? { 
        color: '#fff',
        height: '64px',
        padding: '0 12px',
      }
    : {};

  // 当前显示的图标
  const currentIcon = getThemeIcon(themeMode, isDark);

  return (
    <Dropdown
      menu={{ items: menuItems }}
      placement={placement}
      trigger={['click']}
    >
      <Tooltip title={!showLabel ? currentThemeName : undefined}>
        <Button size={size} type="text" style={buttonStyle}>
          <Space>
            <span style={{ fontSize: '16px', display: 'flex', alignItems: 'center' }}>
              {currentIcon}
            </span>
            {showLabel && (
              <Text style={darkMode ? { color: '#fff' } : {}}>
                {currentThemeName}
              </Text>
            )}
          </Space>
        </Button>
      </Tooltip>
    </Dropdown>
  );
};

/**
 * 紧凑型主题切换器（仅图标）
 */
export const CompactThemeSwitcher = ({ size = 'small' }) => {
  return <ThemeSwitcher size={size} showLabel={false} />;
};

/**
 * 简单主题切换按钮（点击直接切换）
 */
export const SimpleThemeToggle = ({ darkMode = false }) => {
  const { toggleTheme, isDark, themeMode } = useTheme();
  const { t } = useI18n();

  const getTooltip = () => {
    switch (themeMode) {
      case ThemeMode.LIGHT:
        return t('theme.switchToDark');
      case ThemeMode.DARK:
        return t('theme.switchToSystem');
      case ThemeMode.SYSTEM:
      default:
        return t('theme.switchToLight');
    }
  };

  return (
    <Tooltip title={getTooltip()}>
      <Button
        type="text"
        onClick={toggleTheme}
        style={darkMode ? { color: '#fff', height: '64px' } : {}}
        icon={getThemeIcon(themeMode, isDark)}
      />
    </Tooltip>
  );
};

export default ThemeSwitcher;
