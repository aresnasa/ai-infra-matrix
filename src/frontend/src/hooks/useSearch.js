/**
 * 通用模糊搜索 Hook
 * 支持本地模糊匹配和远程全文索引搜索
 * 可适配整个前端项目的搜索需求
 */

import { useState, useMemo, useCallback, useRef, useEffect } from 'react';

/**
 * 简单的 debounce 函数实现
 */
const debounce = (fn, delay) => {
  let timeoutId;
  const debouncedFn = (...args) => {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => fn(...args), delay);
  };
  debouncedFn.cancel = () => clearTimeout(timeoutId);
  return debouncedFn;
};

/**
 * 模糊匹配算法
 * @param {string} text - 待匹配文本
 * @param {string} pattern - 搜索模式
 * @returns {object} - { match: boolean, score: number, highlights: array }
 */
export const fuzzyMatch = (text, pattern) => {
  if (!pattern || !text) {
    return { match: true, score: 0, highlights: [] };
  }

  const textLower = String(text).toLowerCase();
  const patternLower = pattern.toLowerCase().trim();

  // 完全匹配给最高分
  if (textLower === patternLower) {
    return { match: true, score: 100, highlights: [[0, text.length]] };
  }

  // 包含匹配
  const index = textLower.indexOf(patternLower);
  if (index !== -1) {
    return {
      match: true,
      score: 80 - index * 0.5, // 越靠前分数越高
      highlights: [[index, index + patternLower.length]],
    };
  }

  // 模糊字符匹配 (每个字符按顺序匹配)
  let patternIdx = 0;
  let score = 0;
  const highlights = [];
  let consecutiveMatches = 0;

  for (let i = 0; i < textLower.length && patternIdx < patternLower.length; i++) {
    if (textLower[i] === patternLower[patternIdx]) {
      highlights.push([i, i + 1]);
      score += 10 + consecutiveMatches * 5; // 连续匹配加分
      consecutiveMatches++;
      patternIdx++;
    } else {
      consecutiveMatches = 0;
    }
  }

  if (patternIdx === patternLower.length) {
    return { match: true, score, highlights };
  }

  return { match: false, score: 0, highlights: [] };
};

/**
 * 多字段模糊搜索
 * @param {object} item - 数据项
 * @param {string} pattern - 搜索模式
 * @param {string[]} searchFields - 要搜索的字段列表
 * @returns {object} - { match: boolean, score: number, fieldMatches: object }
 */
export const multiFieldFuzzyMatch = (item, pattern, searchFields) => {
  if (!pattern || pattern.trim() === '') {
    return { match: true, score: 0, fieldMatches: {} };
  }

  let totalScore = 0;
  const fieldMatches = {};
  let anyMatch = false;

  // 支持空格分隔的多个搜索词 (AND 逻辑)
  const patterns = pattern.trim().split(/\s+/).filter(p => p);

  for (const field of searchFields) {
    const value = getNestedValue(item, field);
    if (value === null || value === undefined) continue;

    const valueStr = String(value);
    let fieldScore = 0;
    let fieldMatched = true;
    const allHighlights = [];

    // 所有搜索词都要匹配
    for (const p of patterns) {
      const result = fuzzyMatch(valueStr, p);
      if (result.match) {
        fieldScore += result.score;
        allHighlights.push(...result.highlights);
      } else {
        fieldMatched = false;
        break;
      }
    }

    if (fieldMatched) {
      anyMatch = true;
      fieldMatches[field] = {
        value: valueStr,
        highlights: mergeHighlights(allHighlights),
      };
      totalScore += fieldScore;
    }
  }

  return { match: anyMatch, score: totalScore, fieldMatches };
};

/**
 * 获取嵌套对象的值
 * @param {object} obj - 对象
 * @param {string} path - 路径，如 'a.b.c'
 */
const getNestedValue = (obj, path) => {
  return path.split('.').reduce((current, key) => {
    return current && current[key] !== undefined ? current[key] : null;
  }, obj);
};

/**
 * 合并重叠的高亮区间
 */
const mergeHighlights = (highlights) => {
  if (highlights.length === 0) return [];
  
  const sorted = [...highlights].sort((a, b) => a[0] - b[0]);
  const merged = [sorted[0]];
  
  for (let i = 1; i < sorted.length; i++) {
    const last = merged[merged.length - 1];
    const current = sorted[i];
    
    if (current[0] <= last[1]) {
      last[1] = Math.max(last[1], current[1]);
    } else {
      merged.push(current);
    }
  }
  
  return merged;
};

/**
 * 主搜索 Hook
 * @param {object} options - 配置选项
 * @param {array} options.data - 要搜索的数据数组
 * @param {string[]} options.searchFields - 要搜索的字段列表
 * @param {function} options.onRemoteSearch - 远程搜索回调 (可选，用于全文索引)
 * @param {number} options.debounceMs - 防抖延迟毫秒数，默认 300
 * @param {boolean} options.caseSensitive - 是否区分大小写，默认 false
 * @param {number} options.minSearchLength - 最小搜索长度，默认 1
 */
export const useSearch = ({
  data = [],
  searchFields = [],
  onRemoteSearch = null,
  debounceMs = 300,
  caseSensitive = false,
  minSearchLength = 1,
} = {}) => {
  const [searchText, setSearchText] = useState('');
  const [isSearching, setIsSearching] = useState(false);
  const [remoteResults, setRemoteResults] = useState(null);
  const abortControllerRef = useRef(null);

  // 本地搜索结果
  const localSearchResults = useMemo(() => {
    if (!searchText || searchText.length < minSearchLength) {
      return data;
    }

    const results = data
      .map((item) => {
        const matchResult = multiFieldFuzzyMatch(item, searchText, searchFields);
        return {
          ...item,
          _searchMatch: matchResult,
        };
      })
      .filter((item) => item._searchMatch.match)
      .sort((a, b) => b._searchMatch.score - a._searchMatch.score);

    return results;
  }, [data, searchText, searchFields, minSearchLength]);

  // 远程搜索（防抖）
  const debouncedRemoteSearch = useMemo(
    () =>
      debounce(async (query) => {
        if (!onRemoteSearch || query.length < minSearchLength) {
          setRemoteResults(null);
          setIsSearching(false);
          return;
        }

        // 取消之前的请求
        if (abortControllerRef.current) {
          abortControllerRef.current.abort();
        }
        abortControllerRef.current = new AbortController();

        setIsSearching(true);
        try {
          const results = await onRemoteSearch(query, {
            signal: abortControllerRef.current.signal,
          });
          setRemoteResults(results);
        } catch (error) {
          if (error.name !== 'AbortError') {
            console.error('Remote search error:', error);
            setRemoteResults(null);
          }
        } finally {
          setIsSearching(false);
        }
      }, debounceMs),
    [onRemoteSearch, debounceMs, minSearchLength]
  );

  // 搜索文本变化时触发远程搜索
  useEffect(() => {
    if (onRemoteSearch) {
      debouncedRemoteSearch(searchText);
    }
    return () => {
      debouncedRemoteSearch.cancel();
    };
  }, [searchText, debouncedRemoteSearch, onRemoteSearch]);

  // 清理
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
    };
  }, []);

  // 搜索处理函数
  const handleSearch = useCallback((value) => {
    setSearchText(value);
  }, []);

  // 清空搜索
  const clearSearch = useCallback(() => {
    setSearchText('');
    setRemoteResults(null);
  }, []);

  // 最终结果（优先使用远程结果）
  const results = remoteResults !== null ? remoteResults : localSearchResults;

  return {
    searchText,
    setSearchText: handleSearch,
    clearSearch,
    results,
    isSearching,
    // 暴露本地和远程结果，方便调试
    localResults: localSearchResults,
    remoteResults,
    // 高亮工具函数
    highlightText,
  };
};

/**
 * 高亮文本渲染工具
 * @param {string} text - 原始文本
 * @param {array} highlights - 高亮区间数组 [[start, end], ...]
 * @param {string} highlightClass - 高亮样式类名
 * @returns {React.ReactNode}
 */
export const highlightText = (text, highlights = [], highlightClass = 'search-highlight') => {
  if (!highlights || highlights.length === 0) {
    return text;
  }

  const result = [];
  let lastIndex = 0;

  for (const [start, end] of highlights) {
    if (start > lastIndex) {
      result.push(text.substring(lastIndex, start));
    }
    result.push(
      <mark key={start} className={highlightClass} style={{ 
        backgroundColor: '#ffe58f', 
        padding: '0 2px',
        borderRadius: '2px'
      }}>
        {text.substring(start, end)}
      </mark>
    );
    lastIndex = end;
  }

  if (lastIndex < text.length) {
    result.push(text.substring(lastIndex));
  }

  return result;
};

/**
 * 搜索统计信息
 */
export const useSearchStats = (results, totalCount) => {
  return useMemo(() => ({
    matchCount: results.length,
    totalCount,
    matchRate: totalCount > 0 ? ((results.length / totalCount) * 100).toFixed(1) : 0,
  }), [results.length, totalCount]);
};

export default useSearch;
