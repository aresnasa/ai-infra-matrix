import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { Result, Button, Spin } from 'antd';
import { hasRoutePermission } from '../utils/permissions';

/**
 * 路由保护组件
 * 根据用户权限控制路由访问
 */
const ProtectedRoute = ({ children, user, requiredPermission, fallbackPath = '/projects' }) => {
  const location = useLocation();

  // 如果用户未登录，重定向到登录页
  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // 如果需要特定权限但用户没有权限
  if (requiredPermission && !hasRoutePermission(requiredPermission, user)) {
    return (
      <Result
        status="403"
        title="权限不足"
        subTitle={`您没有访问 ${requiredPermission} 的权限。如需访问，请联系管理员。`}
        extra={
          <Button type="primary" onClick={() => window.history.back()}>
            返回上一页
          </Button>
        }
      />
    );
  }

  // 如果路由被限制，重定向到允许的页面
  if (!hasRoutePermission(location.pathname, user)) {
    return (
      <Result
        status="403"
        title="访问被拒绝"
        subTitle="您没有权限访问此页面，将跳转到允许的页面。"
        extra={
          <Button type="primary" onClick={() => window.location.href = fallbackPath}>
            跳转到项目页面
          </Button>
        }
      />
    );
  }

  return children;
};

/**
 * 管理员路由保护组件
 * 只允许管理员访问
 */
export const AdminProtectedRoute = ({ children, user }) => {
  const location = useLocation();

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // 检查是否为管理员
  const isAdmin = user.role === 'admin' || user.role === 'super-admin' ||
                  (user.roles && user.roles.some(role => role.name === 'admin' || role.name === 'super-admin'));

  if (!isAdmin) {
    return (
      <Result
        status="403"
        title="管理员权限 required"
        subTitle="此页面需要管理员权限才能访问。"
        extra={
          <Button type="primary" onClick={() => window.location.href = '/projects'}>
            返回项目页面
          </Button>
        }
      />
    );
  }

  return children;
};

/**
 * 团队权限路由保护组件
 * 根据团队角色控制访问
 */
export const TeamProtectedRoute = ({ children, user, allowedTeams = [] }) => {
  const location = useLocation();

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  // 获取用户团队角色
  const userTeam = user.role_template || user.roleTemplate;

  // 如果没有指定允许的团队，默认允许所有
  if (allowedTeams.length === 0) {
    return children;
  }

  // 检查用户团队是否在允许列表中
  if (!allowedTeams.includes(userTeam)) {
    const teamNames = {
      'data-developer': '数据开发团队',
      'sre': 'SRE运维团队',
      'audit': '审计审核团队'
    };

    const allowedTeamNames = allowedTeams.map(team => teamNames[team] || team).join('、');

    return (
      <Result
        status="403"
        title="团队权限不足"
        subTitle={`此页面只允许 ${allowedTeamNames} 访问。您的团队角色: ${teamNames[userTeam] || '未知'}`}
        extra={
          <Button type="primary" onClick={() => window.location.href = '/projects'}>
            返回项目页面
          </Button>
        }
      />
    );
  }

  return children;
};

export default ProtectedRoute;