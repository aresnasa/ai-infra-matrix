/**
 * 通用搜索输入组件
 * 配合 useSearch hook 使用，提供统一的搜索UI体验
 */

import React, { useState, useCallback, useRef, useEffect } from 'react';
import { Input, Dropdown, Typography, Space, Tag, Spin, Empty, Tooltip } from 'antd';
import {
  SearchOutlined,
  CloseCircleOutlined,
  HistoryOutlined,
  FilterOutlined,
} from '@ant-design/icons';

const { Text } = Typography;

/**
 * 搜索历史存储 Key 前缀
 */
const SEARCH_HISTORY_KEY = 'ai_infra_search_history_';

/**
 * 获取搜索历史
 */
const getSearchHistory = (namespace) => {
  try {
    const key = SEARCH_HISTORY_KEY + namespace;
    const history = localStorage.getItem(key);
    return history ? JSON.parse(history) : [];
  } catch {
    return [];
  }
};

/**
 * 保存搜索历史
 */
const saveSearchHistory = (namespace, history) => {
  try {
    const key = SEARCH_HISTORY_KEY + namespace;
    localStorage.setItem(key, JSON.stringify(history.slice(0, 10))); // 最多保存10条
  } catch {
    // ignore
  }
};

/**
 * 添加搜索历史
 */
const addToHistory = (namespace, term) => {
  if (!term || term.trim() === '') return;
  
  const history = getSearchHistory(namespace);
  const filtered = history.filter(h => h.toLowerCase() !== term.toLowerCase());
  filtered.unshift(term);
  saveSearchHistory(namespace, filtered);
};

/**
 * SearchInput 组件
 * 
 * @param {object} props
 * @param {string} props.value - 搜索值
 * @param {function} props.onChange - 值变化回调
 * @param {function} props.onSearch - 搜索回调 (按回车或点击搜索按钮)
 * @param {string} props.placeholder - 占位文本
 * @param {boolean} props.loading - 是否加载中
 * @param {string} props.namespace - 搜索历史命名空间
 * @param {array} props.searchFields - 可搜索字段列表 (用于显示提示)
 * @param {number} props.resultCount - 搜索结果数量
 * @param {number} props.totalCount - 总数据量
 * @param {boolean} props.showStats - 是否显示搜索统计
 * @param {boolean} props.showHistory - 是否显示搜索历史
 * @param {boolean} props.allowClear - 是否允许清空
 * @param {string} props.size - 输入框大小 'small' | 'middle' | 'large'
 * @param {object} props.style - 自定义样式
 */
const SearchInput = ({
  value = '',
  onChange,
  onSearch,
  placeholder = '搜索...',
  loading = false,
  namespace = 'default',
  searchFields = [],
  resultCount,
  totalCount,
  showStats = true,
  showHistory = true,
  allowClear = true,
  size = 'middle',
  style = {},
  ...restProps
}) => {
  const [focused, setFocused] = useState(false);
  const [historyVisible, setHistoryVisible] = useState(false);
  const [history, setHistory] = useState([]);
  const inputRef = useRef(null);

  // 加载历史记录
  useEffect(() => {
    if (showHistory && focused) {
      setHistory(getSearchHistory(namespace));
    }
  }, [focused, namespace, showHistory]);

  // 处理输入变化
  const handleChange = useCallback(
    (e) => {
      const newValue = e.target.value;
      onChange?.(newValue);
    },
    [onChange]
  );

  // 处理搜索
  const handleSearch = useCallback(
    (searchValue) => {
      const val = searchValue || value;
      if (val && val.trim()) {
        addToHistory(namespace, val.trim());
        setHistory(getSearchHistory(namespace));
      }
      onSearch?.(val);
      setHistoryVisible(false);
    },
    [value, onSearch, namespace]
  );

  // 处理清空
  const handleClear = useCallback(() => {
    onChange?.('');
    onSearch?.('');
    inputRef.current?.focus();
  }, [onChange, onSearch]);

  // 处理历史项点击
  const handleHistoryClick = useCallback(
    (term) => {
      onChange?.(term);
      handleSearch(term);
    },
    [onChange, handleSearch]
  );

  // 清空历史
  const clearHistory = useCallback(() => {
    saveSearchHistory(namespace, []);
    setHistory([]);
  }, [namespace]);

  // 历史记录下拉内容
  const historyDropdown = showHistory && history.length > 0 && (
    <div
      style={{
        background: '#fff',
        borderRadius: 8,
        boxShadow: '0 3px 6px -4px rgba(0,0,0,.12), 0 6px 16px 0 rgba(0,0,0,.08)',
        padding: '8px 0',
        minWidth: 280,
        maxWidth: 400,
      }}
    >
      <div style={{ 
        padding: '4px 12px', 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center',
        borderBottom: '1px solid #f0f0f0',
        marginBottom: 4
      }}>
        <Text type="secondary" style={{ fontSize: 12 }}>
          <HistoryOutlined /> 搜索历史
        </Text>
        <Text
          type="secondary"
          style={{ fontSize: 12, cursor: 'pointer' }}
          onClick={clearHistory}
        >
          清空
        </Text>
      </div>
      {history.map((term, index) => (
        <div
          key={index}
          style={{
            padding: '6px 12px',
            cursor: 'pointer',
            transition: 'background 0.2s',
          }}
          className="search-history-item"
          onClick={() => handleHistoryClick(term)}
          onMouseEnter={(e) => (e.target.style.background = '#f5f5f5')}
          onMouseLeave={(e) => (e.target.style.background = 'transparent')}
        >
          <Text ellipsis style={{ maxWidth: 350 }}>{term}</Text>
        </div>
      ))}
    </div>
  );

  // 搜索字段提示
  const fieldHint = searchFields.length > 0 && (
    <Tooltip
      title={
        <div>
          <div style={{ marginBottom: 4 }}>可搜索字段：</div>
          {searchFields.map((field, idx) => (
            <Tag key={idx} size="small" style={{ marginBottom: 2 }}>
              {field}
            </Tag>
          ))}
        </div>
      }
    >
      <FilterOutlined style={{ color: '#999', cursor: 'help' }} />
    </Tooltip>
  );

  // 搜索统计
  const statsDisplay = showStats && resultCount !== undefined && totalCount !== undefined && value && (
    <Text type="secondary" style={{ fontSize: 12, marginLeft: 8, whiteSpace: 'nowrap' }}>
      找到 {resultCount}/{totalCount}
    </Text>
  );

  return (
    <Space style={{ width: '100%', ...style }} direction="vertical" size={4}>
      <Space.Compact style={{ width: '100%' }}>
        <Dropdown
          open={historyVisible && focused && !value && history.length > 0}
          dropdownRender={() => historyDropdown}
          placement="bottomLeft"
          trigger={['click']}
        >
          <Input
            ref={inputRef}
            value={value}
            onChange={handleChange}
            onPressEnter={() => handleSearch()}
            onFocus={() => {
              setFocused(true);
              setHistoryVisible(true);
            }}
            onBlur={() => {
              setTimeout(() => {
                setFocused(false);
                setHistoryVisible(false);
              }, 200);
            }}
            placeholder={placeholder}
            prefix={loading ? <Spin size="small" /> : <SearchOutlined />}
            suffix={
              <Space size={4}>
                {value && allowClear && (
                  <CloseCircleOutlined
                    style={{ color: '#999', cursor: 'pointer' }}
                    onClick={handleClear}
                  />
                )}
                {fieldHint}
              </Space>
            }
            size={size}
            allowClear={false}
            {...restProps}
          />
        </Dropdown>
      </Space.Compact>
      {statsDisplay}
    </Space>
  );
};

/**
 * 高级搜索组件 (带筛选器)
 */
export const AdvancedSearchInput = ({
  value = '',
  onChange,
  onSearch,
  filters = [],
  activeFilters = {},
  onFilterChange,
  ...restProps
}) => {
  const [filterValues, setFilterValues] = useState(activeFilters);

  const handleFilterChange = useCallback(
    (key, val) => {
      const newFilters = { ...filterValues, [key]: val };
      setFilterValues(newFilters);
      onFilterChange?.(newFilters);
    },
    [filterValues, onFilterChange]
  );

  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      <SearchInput value={value} onChange={onChange} onSearch={onSearch} {...restProps} />
      {filters.length > 0 && (
        <Space wrap size={[8, 8]}>
          {filters.map((filter) => (
            <Space key={filter.key} size={4}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {filter.label}:
              </Text>
              {filter.render
                ? filter.render(filterValues[filter.key], (val) =>
                    handleFilterChange(filter.key, val)
                  )
                : null}
            </Space>
          ))}
        </Space>
      )}
    </Space>
  );
};

export default SearchInput;
