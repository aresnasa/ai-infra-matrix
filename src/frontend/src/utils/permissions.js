/**
 * 权限控制工具函数
 * 根据用户角色模板控制页面访问权限
 */

// 角色权限配置
export const ROLE_PERMISSIONS = {
  'data-developer': {
    name: '数据开发团队',
    allowedRoutes: [
      '/projects',
      '/jupyterhub',
      '/slurm',
      '/dashboard',
      '/enhanced-dashboard',
      '/profile'
    ],
    restrictedRoutes: [
      '/admin',
      '/saltstack',
      '/ansible',
      '/kubernetes',
      '/kafka-ui'
    ],
    menuItems: [
      'dashboard',
      'enhanced-dashboard',
      'projects',
      'jupyterhub',
      'slurm'
    ]
  },
  'sre': {
    name: 'SRE运维团队',
    allowedRoutes: [
      '/projects',
      '/jupyterhub',
      '/slurm',
      '/saltstack',
      '/ansible',
      '/kubernetes',
      '/dashboard',
      '/enhanced-dashboard',
      '/admin',
      '/profile'
    ],
    restrictedRoutes: [],
    menuItems: [
      'dashboard',
      'enhanced-dashboard',
      'projects',
      'gitea',
      'kubernetes',
      'ansible',
      'jupyterhub',
      'slurm',
      'saltstack',
      'kafka-ui',
      'admin'
    ]
  },
  'audit': {
    name: '审计审核团队',
    allowedRoutes: [
      '/projects',
      '/kafka-ui',
      '/dashboard',
      '/enhanced-dashboard',
      '/audit-logs',
      '/profile'
    ],
    restrictedRoutes: [
      '/admin',
      '/saltstack',
      '/ansible',
      '/kubernetes',
      '/jupyterhub',
      '/slurm',
      '/gitea'
    ],
    menuItems: [
      'dashboard',
      'enhanced-dashboard',
      'projects',
      'kafka-ui',
      'audit-logs'
    ]
  },
  'admin': {
    name: '系统管理员',
    allowedRoutes: [
      '/', // 根路径
      '/projects',
      '/jupyterhub',
      '/slurm',
      '/saltstack',
      '/ansible',
      '/kubernetes',
      '/kafka-ui',
      '/dashboard',
      '/enhanced-dashboard',
      '/admin',
      '/profile',
      '/audit-logs',
      '/gitea',
      '/users',
      '/settings',
      '/logs',
      '/monitoring',
      '/system-info'
    ],
    restrictedRoutes: [], // 管理员没有任何限制
    menuItems: [
      'dashboard',
      'enhanced-dashboard',
      'projects',
      'gitea',
      'kubernetes',
      'ansible',
      'jupyterhub',
      'slurm',
      'saltstack',
      'kafka-ui',
      'admin',
      'users',
      'settings',
      'logs',
      'monitoring'
    ],
    permissions: [
      '所有系统权限',
      '用户管理',
      '系统配置',
      '审计日志',
      '基础设施管理',
      '数据开发环境管理',
      '安全配置管理'
    ]
  }
};

/**
 * 检查用户是否有权限访问指定路由
 * @param {string} route - 路由路径
 * @param {Object} user - 用户对象
 * @returns {boolean} 是否有权限
 */
export const hasRoutePermission = (route, user) => {
  if (!user) return false;

  // 管理员始终有所有权限
  if (user.role === 'admin' || user.role === 'super-admin') {
    return true;
  }

  // 获取用户角色模板
  const roleTemplate = user.role_template || user.roleTemplate;
  if (!roleTemplate || !ROLE_PERMISSIONS[roleTemplate]) {
    // 如果没有角色模板，默认只允许基本路由
    const basicRoutes = ['/projects', '/dashboard', '/enhanced-dashboard', '/profile'];
    return basicRoutes.some(basicRoute => route.startsWith(basicRoute));
  }

  const permissions = ROLE_PERMISSIONS[roleTemplate];

  // 检查是否在允许路由列表中
  const isAllowed = permissions.allowedRoutes.some(allowedRoute =>
    route.startsWith(allowedRoute)
  );

  // 检查是否在限制路由列表中
  const isRestricted = permissions.restrictedRoutes.some(restrictedRoute =>
    route.startsWith(restrictedRoute)
  );

  return isAllowed && !isRestricted;
};

/**
 * 获取用户可用的菜单项
 * @param {Object} user - 用户对象
 * @returns {Array} 可用菜单项的key数组
 */
export const getAvailableMenuItems = (user) => {
  if (!user) return [];

  // 管理员有所有菜单权限
  if (user.role === 'admin' || user.role === 'super-admin') {
    return [
      'dashboard',
      'enhanced-dashboard',
      'projects',
      'gitea',
      'kubernetes',
      'ansible',
      'jupyterhub',
      'slurm',
      'saltstack',
      'kafka-ui',
      'admin'
    ];
  }

  // 获取用户角色模板
  const roleTemplate = user.role_template || user.roleTemplate;
  if (!roleTemplate || !ROLE_PERMISSIONS[roleTemplate]) {
    // 默认菜单项
    return ['dashboard', 'enhanced-dashboard', 'projects'];
  }

  return ROLE_PERMISSIONS[roleTemplate].menuItems;
};

/**
 * 检查用户是否有管理员权限
 * @param {Object} user - 用户对象
 * @returns {boolean} 是否为管理员
 */
export const isAdmin = (user) => {
  if (!user) return false;
  return user.role === 'admin' || user.role === 'super-admin' ||
         (user.roles && user.roles.some(role => role.name === 'admin' || role.name === 'super-admin'));
};

/**
 * 获取用户角色显示名称
 * @param {Object} user - 用户对象
 * @returns {string} 角色显示名称
 */
export const getUserRoleDisplayName = (user) => {
  if (!user) return '未知用户';

  // 管理员角色
  if (isAdmin(user)) {
    return user.role === 'super-admin' ? '超级管理员' : '管理员';
  }

  // 根据角色模板获取显示名称
  const roleTemplate = user.role_template || user.roleTemplate;
  if (roleTemplate && ROLE_PERMISSIONS[roleTemplate]) {
    return ROLE_PERMISSIONS[roleTemplate].name;
  }

  // 默认角色
  return '普通用户';
};

/**
 * 获取用户权限描述
 * @param {Object} user - 用户对象
 * @returns {Array} 权限描述数组
 */
export const getUserPermissions = (user) => {
  if (!user) return [];

  // 管理员权限
  if (isAdmin(user)) {
    return ['所有系统权限', '用户管理', '系统配置', '审计日志'];
  }

  // 根据角色模板获取权限
  const roleTemplate = user.role_template || user.roleTemplate;
  if (roleTemplate && ROLE_PERMISSIONS[roleTemplate]) {
    return ROLE_PERMISSIONS[roleTemplate].permissions || [];
  }

  return ['基本项目访问权限'];
};