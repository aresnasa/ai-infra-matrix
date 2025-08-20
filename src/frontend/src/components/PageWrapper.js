import React from 'react';
import { usePageMeta } from '../hooks/useFavicon';

/**
 * 页面包装器组件
 * 自动设置页面标题和favicon
 * 
 * @param {Object} props
 * @param {string} props.title - 页面标题
 * @param {string} props.pageType - 页面类型 (jupyter/admin/kubernetes/ansible)
 * @param {React.ReactNode} props.children - 子组件
 */
const PageWrapper = ({ title, pageType, children }) => {
  usePageMeta(title, pageType);
  
  return children;
};

/**
 * 高阶组件：为页面添加favicon和标题管理
 * 
 * @param {React.Component} WrappedComponent - 要包装的组件
 * @param {Object} options - 配置选项
 * @param {string} options.title - 页面标题
 * @param {string} options.pageType - 页面类型
 * @returns {React.Component} - 包装后的组件
 */
export const withPageMeta = (WrappedComponent, options = {}) => {
  const { title, pageType } = options;
  
  return function WithPageMetaComponent(props) {
    return (
      <PageWrapper title={title} pageType={pageType}>
        <WrappedComponent {...props} />
      </PageWrapper>
    );
  };
};

export default PageWrapper;
