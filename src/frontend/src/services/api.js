import axios from 'axios';
import { message } from 'antd';
import { apiRequestManager } from '../utils/apiRequestManager';

// 创建axios实例
export const api = axios.create({
  baseURL: process.env.REACT_APP_API_URL || '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求重试机制
let retryCount = 0;
const MAX_RETRIES = 2;

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    
    // 添加请求时间戳，用于监控
    config.metadata = { startTime: Date.now() };
    
    return config;
  },
  (error) => {
    console.error('Request interceptor error:', error);
    return Promise.reject(error);
  }
);

// 响应拦截器 - 优化错误处理和重试逻辑
api.interceptors.response.use(
  (response) => {
    // 计算响应时间
    if (response.config.metadata) {
      const responseTime = Date.now() - response.config.metadata.startTime;
      if (responseTime > 5000) {
        console.warn(`Slow API response: ${response.config.url} took ${responseTime}ms`);
      }
    }
    
    // 重置重试计数
    retryCount = 0;
    return response;
  },
  async (error) => {
    const originalRequest = error.config;
    
    // 避免在登录页面显示token过期消息
    const isAuthPage = window.location.pathname.includes('/login') || 
                      window.location.pathname.includes('/auth');
    
    if (error.response?.status === 401 && !originalRequest._retry && !isAuthPage) {
      originalRequest._retry = true;
      
      // 只在用户实际登录状态下尝试刷新token
      const token = localStorage.getItem('token');
      if (token && retryCount < MAX_RETRIES) {
        retryCount++;
        
        try {
          console.log('Token expired, attempting refresh...');
          const refreshResponse = await authAPI.refreshToken();
          const { token: newToken, expires_at } = refreshResponse.data;
          
          localStorage.setItem('token', newToken);
          localStorage.setItem('token_expires', expires_at);
          
          // 更新请求头并重试
          originalRequest.headers.Authorization = `Bearer ${newToken}`;
          return api(originalRequest);
          
        } catch (refreshError) {
          console.log('Token refresh failed:', refreshError.message);
          
          // 清理认证状态和缓存
          localStorage.removeItem('token');
          localStorage.removeItem('user');
          localStorage.removeItem('token_expires');
          apiRequestManager.clearCache();
          
          // 只在非认证页面时跳转到登录
          if (!isAuthPage) {
            message.error('登录已过期，请重新登录');
            setTimeout(() => {
              window.location.href = '/login';
            }, 1000);
          }
        }
      }
    }
    
    // 网络错误处理
    if (!error.response) {
      console.error('Network error:', error.message);
      if (!isAuthPage) {
        message.error('网络连接失败，请检查网络设置');
      }
    }
    
    return Promise.reject(error);
  }
);

// 带缓存的API请求包装器
const createCachedRequest = (requestFn, enableCache = true) => {
  return async (...args) => {
    const cacheKey = apiRequestManager.generateCacheKey(
      requestFn.toString(),
      'GET',
      args[0] || {}
    );
    
    return apiRequestManager.wrapRequest(
      () => requestFn(...args),
      cacheKey,
      enableCache
    );
  };
};

// 认证API - 关键API不缓存，避免安全问题
export const authAPI = {
  login: (credentials) => api.post('/auth/login', credentials),
  register: (userData) => api.post('/auth/register', userData),
  validateLDAP: (credentials) => api.post('/auth/validate-ldap', credentials),
  logout: () => api.post('/auth/logout'),
  getCurrentUser: createCachedRequest(() => api.get('/auth/me'), true),
  getProfile: createCachedRequest(() => api.get('/auth/me'), true), 
  refreshToken: () => api.post('/auth/refresh'), // 不缓存token刷新请求
  changePassword: (data) => api.post('/auth/change-password', data), // 修改密码
  updateProfile: (data) => api.put('/users/profile', data), // 更新个人信息
  // 2FA 登录验证
  verify2FALogin: (data) => api.post('/auth/verify-2fa', data),
};
// Kubernetes集群管理API
export const kubernetesAPI = {
  // 获取集群列表
  getClusters: () => api.get('/kubernetes/clusters'),
  
  // 创建集群
  createCluster: (clusterData) => api.post('/kubernetes/clusters', clusterData),
  
  // 更新集群
  updateCluster: (id, clusterData) => api.put(`/kubernetes/clusters/${id}`, clusterData),
  
  // 删除集群
  deleteCluster: (id) => api.delete(`/kubernetes/clusters/${id}`),
  
  // 测试集群连接
  testConnection: (id) => api.post(`/kubernetes/clusters/${id}/test`),
  
  // 获取集群信息
  getClusterInfo: (id) => api.get(`/kubernetes/clusters/${id}/info`),
  
  // 获取集群节点信息
  getClusterNodes: (id) => api.get(`/kubernetes/clusters/${id}/nodes`),
  
  // 获取集群命名空间
  getClusterNamespaces: (id) => api.get(`/kubernetes/clusters/${id}/namespaces`),

  // 资源浏览（若后端未实现，将在调用处做降级处理）
  getPods: (id, namespace) => api.get(`/kubernetes/clusters/${id}/namespaces/${namespace || 'default'}/resources/pods`),
  getDeployments: (id, namespace) => api.get(`/kubernetes/clusters/${id}/namespaces/${namespace || 'default'}/resources/deployments`),
  getServices: (id, namespace) => api.get(`/kubernetes/clusters/${id}/namespaces/${namespace || 'default'}/resources/services`),
  getNodesDetail: (id) => api.get(`/kubernetes/clusters/${id}/cluster-resources/nodes`),
  getEvents: (id, namespace) => api.get(`/kubernetes/clusters/${id}/namespaces/${namespace || ''}/resources/events`),

  // 操作类（按需在后端实现）
  scaleDeployment: (id, namespace, name, replicas) => api.post(`/kubernetes/clusters/${id}/namespaces/${namespace}/resources/deployments/${name}/scale`, { replicas }),
  deleteResource: (id, namespace, kind, name) => api.delete(`/kubernetes/clusters/${id}/namespaces/${namespace}/resources/${kind.toLowerCase()}s/${name}`),
  getPodLogs: (id, namespace, pod, container) => api.get(`/kubernetes/clusters/${id}/namespaces/${namespace}/resources/pods/${pod}/logs`, { params: { container } }),
  buildPodExecWsUrl: (id, namespace, pod, container, command) => {
    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const host = window.location.host;
    const cmdQuery = command ? `&command=${encodeURIComponent(command)}` : '';
    return `${proto}://${host}/api/kubernetes/clusters/${id}/namespaces/${namespace}/resources/pods/${encodeURIComponent(pod)}/exec?container=${encodeURIComponent(container || '')}${cmdQuery}`;
  },

  // ---- 通用资源发现与CRUD（动态）----
  // 资源发现：返回资源组与资源列表，前端可构建资源树
  discover: (id) => api.get(`/kubernetes/clusters/${id}/discovery`),

  // 命名空间内：通用列表、获取、创建、更新、Patch、删除
  listResources: (id, namespace, resource, params) =>
    api.get(`/kubernetes/clusters/${id}/namespaces/${namespace}/resources/${resource}`, { params }),
  getResource: (id, namespace, resource, name) =>
    api.get(`/kubernetes/clusters/${id}/namespaces/${namespace}/resources/${resource}/${encodeURIComponent(name)}`),
  createResource: (id, namespace, resource, obj) =>
    api.post(`/kubernetes/clusters/${id}/namespaces/${namespace}/resources/${resource}`, obj),
  updateResource: (id, namespace, resource, name, obj) =>
    api.put(`/kubernetes/clusters/${id}/namespaces/${namespace}/resources/${resource}/${encodeURIComponent(name)}`, obj),
  patchResource: (id, namespace, resource, name, patch, patchType = 'application/merge-patch+json') =>
    api.patch(`/kubernetes/clusters/${id}/namespaces/${namespace}/resources/${resource}/${encodeURIComponent(name)}`, patch, {
      headers: { 'Content-Type': patchType },
    }),
  deleteResourceGeneric: (id, namespace, resource, name, params) =>
    api.delete(`/kubernetes/clusters/${id}/namespaces/${namespace}/resources/${resource}/${encodeURIComponent(name)}`, { params }),

  // 集群级资源（无命名空间）
  listClusterResources: (id, resource, params) =>
    api.get(`/kubernetes/clusters/${id}/cluster-resources/${resource}`, { params }),
  getClusterResource: (id, resource, name) =>
    api.get(`/kubernetes/clusters/${id}/cluster-resources/${resource}/${encodeURIComponent(name)}`),
  createClusterResource: (id, resource, obj) =>
    api.post(`/kubernetes/clusters/${id}/cluster-resources/${resource}`, obj),
  updateClusterResource: (id, resource, name, obj) =>
    api.put(`/kubernetes/clusters/${id}/cluster-resources/${resource}/${encodeURIComponent(name)}`, obj),
  patchClusterResource: (id, resource, name, patch, patchType = 'application/merge-patch+json') =>
    api.patch(`/kubernetes/clusters/${id}/cluster-resources/${resource}/${encodeURIComponent(name)}`, patch, {
      headers: { 'Content-Type': patchType },
    }),
  deleteClusterResource: (id, resource, name, params) =>
    api.delete(`/kubernetes/clusters/${id}/cluster-resources/${resource}/${encodeURIComponent(name)}`, { params }),

  // 并发批量列表，用于高性能加载多个资源种类
  batchListResources: (id, namespace, kinds = [], params = {}) => {
    const kindsParam = Array.isArray(kinds) ? kinds.join(',') : String(kinds || '');
    return api.get(`/kubernetes/clusters/${id}/namespaces/${namespace}/resources:batch`, {
      params: { ...params, kinds: kindsParam },
    });
  },

  // ---- Helm 相关 API ----
  // 获取 Helm releases 列表
  getHelmReleases: (id, namespace) => api.get(`/kubernetes/clusters/${id}/helm/releases`, { params: { namespace } }),
  
  // 获取单个 release 详情
  getHelmRelease: (id, namespace, releaseName) => api.get(`/kubernetes/clusters/${id}/helm/releases/${releaseName}`, { params: { namespace } }),
  
  // 安装 Helm chart
  installHelmChart: (id, data) => api.post(`/kubernetes/clusters/${id}/helm/releases`, data),
  
  // 升级 Helm release
  upgradeHelmRelease: (id, namespace, releaseName, data) => api.put(`/kubernetes/clusters/${id}/helm/releases/${releaseName}`, { ...data, namespace }),
  
  // 卸载 Helm release
  uninstallHelmRelease: (id, namespace, releaseName) => api.delete(`/kubernetes/clusters/${id}/helm/releases/${releaseName}`, { params: { namespace } }),
  
  // 回滚 Helm release
  rollbackHelmRelease: (id, namespace, releaseName, revision) => api.post(`/kubernetes/clusters/${id}/helm/releases/${releaseName}/rollback`, { namespace, revision }),
  
  // 获取 Helm release 历史
  getHelmReleaseHistory: (id, namespace, releaseName) => api.get(`/kubernetes/clusters/${id}/helm/releases/${releaseName}/history`, { params: { namespace } }),
  
  // 获取 Helm release values
  getHelmReleaseValues: (id, namespace, releaseName) => api.get(`/kubernetes/clusters/${id}/helm/releases/${releaseName}/values`, { params: { namespace } }),
  
  // Helm 仓库管理
  getHelmRepositories: (id) => api.get(`/kubernetes/clusters/${id}/helm/repositories`),
  addHelmRepository: (id, data) => api.post(`/kubernetes/clusters/${id}/helm/repositories`, data),
  updateHelmRepository: (id, repoName) => api.put(`/kubernetes/clusters/${id}/helm/repositories/${repoName}`),
  removeHelmRepository: (id, repoName) => api.delete(`/kubernetes/clusters/${id}/helm/repositories/${repoName}`),
  
  // 搜索 Helm charts
  searchHelmCharts: (id, keyword) => api.get(`/kubernetes/clusters/${id}/helm/charts/search`, { params: { keyword } }),
  
  // 获取 chart 详情
  getHelmChartInfo: (id, repoName, chartName, version) => api.get(`/kubernetes/clusters/${id}/helm/charts/${repoName}/${chartName}`, { params: { version } }),
  
  // 导入/导出 Helm 配置
  importHelmConfig: (id, config) => api.post(`/kubernetes/clusters/${id}/helm/import`, config),
  exportHelmConfig: (id, namespace, releaseName) => api.get(`/kubernetes/clusters/${id}/helm/export/${releaseName}`, { params: { namespace } }),
};

// Ansible管理API
export const ansibleAPI = {
  // 获取playbook列表 - 暂时返回空数组，因为后端没有实现此端点
  getPlaybooks: () => Promise.resolve({ data: { data: [] } }),
  
  // 创建playbook - 暂时不可用
  createPlaybook: (playbookData) => Promise.reject(new Error('功能暂未实现')),
  
  // 更新playbook - 暂时不可用
  updatePlaybook: (id, playbookData) => Promise.reject(new Error('功能暂未实现')),
  
  // 删除playbook - 暂时不可用
  deletePlaybook: (id) => Promise.reject(new Error('功能暂未实现')),
  
  // 执行playbook - 暂时不可用
  executePlaybook: (id, params) => Promise.reject(new Error('功能暂未实现')),
  
  // 获取执行历史 - 暂时不可用
  getExecutionHistory: (id) => Promise.reject(new Error('功能暂未实现')),
  
  // 测试Ansible连接 - 暂时不可用
  testAnsibleConnection: (config) => Promise.reject(new Error('功能暂未实现')),
  
  // 生成playbook - 使用正确的端点
  generatePlaybook: (config) => api.post('/playbook/generate', config),

  // 模板库（后端可实现以下端点；前端会在页面中做localStorage降级）
  getTemplates: () => api.get('/ansible/templates'),
  createTemplate: (tpl) => api.post('/ansible/templates', tpl),
  updateTemplate: (id, tpl) => api.put(`/ansible/templates/${id}`, tpl),
  deleteTemplate: (id) => api.delete(`/ansible/templates/${id}`),
};

// 项目管理API
export const projectAPI = {
  getProjects: () => api.get('/projects'),
  createProject: (projectData) => api.post('/projects', projectData),
  updateProject: (id, projectData) => api.put(`/projects/${id}`, projectData),
  deleteProject: (id) => api.delete(`/projects/${id}`),
  getProject: (id) => api.get(`/projects/${id}`),
  getProjectDetail: (id) => api.get(`/projects/${id}`), // 兼容性别名
};

// 用户管理API
export const userAPI = {
  // 基础用户管理
  getUsers: (params) => api.get('/users', { params }),
  getUser: (id) => api.get(`/users/${id}`),
  createUser: (userData) => api.post('/users', userData),
  updateUser: (id, userData) => api.put(`/users/${id}`, userData),
  deleteUser: (id) => api.delete(`/users/${id}`),
  resetPassword: (id) => api.post(`/users/${id}/reset-password`),
  
  // 用户角色模板管理（管理员）
  updateUserRoleTemplate: (id, data) => api.put(`/users/${id}/role-template`, data),
  
  // 用户个人信息
  getUserProfile: () => api.get('/users/profile'),
  updateUserProfile: (profileData) => api.put('/users/profile', profileData),
  
  // 用户组管理
  getUserGroups: () => api.get('/user-groups'),
  getUserGroup: (id) => api.get(`/user-groups/${id}`),
  createUserGroup: (groupData) => api.post('/user-groups', groupData),
  updateUserGroup: (id, groupData) => api.put(`/user-groups/${id}`, groupData),
  deleteUserGroup: (id) => api.delete(`/user-groups/${id}`),
  
  // 角色管理
  getRoles: () => api.get('/roles'),
  getRole: (id) => api.get(`/roles/${id}`),
  createRole: (roleData) => api.post('/roles', roleData),
  updateRole: (id, roleData) => api.put(`/roles/${id}`, roleData),
  deleteRole: (id) => api.delete(`/roles/${id}`),
  
  // 权限管理
  getPermissions: () => api.get('/permissions'),
  getUserPermissions: (id) => api.get(`/users/${id}/permissions`),
  assignRoleToUser: (userId, roleId) => api.post(`/users/${userId}/roles/${roleId}`),
  removeRoleFromUser: (userId, roleId) => api.delete(`/users/${userId}/roles/${roleId}`),
  
  // LDAP相关（增强功能）
  getUserWithAuthSource: (id) => api.get(`/users/${id}/auth-source`),
  
  // 用户仪表板相关
  getUserDashboardSettings: () => api.get('/users/dashboard/settings'),
  updateUserDashboardSettings: (settings) => api.put('/users/dashboard/settings', settings),
};

// 主机管理API
export const hostAPI = {
  // 获取项目的主机列表
  getHosts: (projectId) => api.get(`/projects/${projectId}/hosts`),
  
  // 创建主机
  createHost: (hostData) => api.post('/hosts', hostData),
  
  // 更新主机
  updateHost: (id, hostData) => api.put(`/hosts/${id}`, hostData),
  
  // 删除主机
  deleteHost: (id) => api.delete(`/hosts/${id}`),
  
  // 测试主机连接
  testHostConnection: (id) => api.post(`/hosts/${id}/test`),
  
  // 获取主机详情
  getHostDetail: (id) => api.get(`/hosts/${id}`),
};

// 变量管理API
export const variableAPI = {
  // 获取项目变量
  getVariables: (projectId) => api.get(`/projects/${projectId}/variables`),
  
  // 创建变量
  createVariable: (variableData) => api.post('/variables', variableData),
  
  // 更新变量
  updateVariable: (id, variableData) => api.put(`/variables/${id}`, variableData),
  
  // 删除变量
  deleteVariable: (id) => api.delete(`/variables/${id}`),
  
  // 获取变量详情
  getVariableDetail: (id) => api.get(`/variables/${id}`),
};

// 任务管理API
export const taskAPI = {
  // 获取项目任务
  getTasks: (projectId) => api.get(`/projects/${projectId}/tasks`),
  
  // 创建任务
  createTask: (taskData) => api.post('/tasks', taskData),
  
  // 更新任务
  updateTask: (id, taskData) => api.put(`/tasks/${id}`, taskData),
  
  // 删除任务
  deleteTask: (id) => api.delete(`/tasks/${id}`),
  
  // 执行任务
  executeTask: (id, params) => api.post(`/tasks/${id}/execute`, params),
  
  // 获取任务执行日志
  getTaskLogs: (id) => api.get(`/tasks/${id}/logs`),
};

// Playbook管理API
export const playbookAPI = {
  // 获取项目playbooks
  getPlaybooks: (projectId) => api.get(`/projects/${projectId}/playbooks`),
  
  // 创建playbook
  createPlaybook: (playbookData) => api.post('/playbooks', playbookData),
  
  // 更新playbook
  updatePlaybook: (id, playbookData) => api.put(`/playbooks/${id}`, playbookData),
  
  // 删除playbook
  deletePlaybook: (id) => api.delete(`/playbooks/${id}`),
  
  // 执行playbook
  executePlaybook: (id, params) => api.post(`/playbooks/${id}/execute`, params),
  
  // 获取playbook详情
  getPlaybookDetail: (id) => api.get(`/playbooks/${id}`),
  
  // 生成playbook - 使用正确的后端端点
  generatePlaybook: (projectId) => api.post('/playbook/generate', { project_id: projectId }),
  
  // 预览playbook
  previewPlaybook: (config) => api.post('/playbook/preview', config),
  
  // 验证playbook
  validatePlaybook: (content) => api.post('/playbook/validate', content),
  
  // 下载playbook
  downloadPlaybook: (id) => api.get(`/playbook/download/${id}`, { responseType: 'blob' }),
};

// 管理员API
export const adminAPI = {
  // 用户管理
  getUsers: () => api.get('admin/users'),
  createUser: (userData) => api.post('admin/users', userData),
  updateUser: (id, userData) => api.put(`admin/users/${id}`, userData),
  deleteUser: (id) => api.delete(`admin/users/${id}`),
  toggleUserStatus: (id, isActive) => api.put(`users/${id}/status`, { is_active: isActive }),
  
  // 项目管理
  getProjects: () => api.get('admin/projects'),
  updateProject: (id, projectData) => api.put(`admin/projects/${id}`, projectData),
  deleteProject: (id) => api.delete(`admin/projects/${id}`),
  
  // 系统设置
  getSystemSettings: () => api.get('admin/settings'),
  updateSystemSettings: (settings) => api.put('admin/settings', settings),
  
  // LDAP设置
  getLDAPSettings: () => api.get('admin/ldap'),
  updateLDAPSettings: (settings) => api.put('admin/ldap', settings),
  testLDAPConnection: (settings) => api.post('admin/ldap/test', settings),
  
  // 认证设置
  getAuthSettings: () => api.get('admin/auth'),
  updateAuthSettings: (settings) => api.put('admin/auth', settings),
  
  // 回收站
  getTrashItems: () => api.get('admin/trash'),
  restoreTrashItem: (id) => api.post(`admin/trash/${id}/restore`),
  permanentDeleteTrashItem: (id) => api.delete(`admin/trash/${id}/permanent`),
  
  // 增强用户管理功能
  getUserWithAuthSource: (id) => api.get(`admin/users/${id}/auth-source`),
  resetUserPassword: (id, data) => api.post(`admin/enhanced-users/${id}/reset-password`, data),
  updateUserGroups: (id, data) => api.put(`admin/enhanced-users/${id}/groups`, data),
  updateUserStatusEnhanced: (id, data) => api.put(`admin/enhanced-users/${id}/status`, data),
  
  // 注册审批功能
  getPendingApprovals: () => api.get('admin/approvals/pending'),
  approveRegistration: (id) => api.post(`admin/approvals/${id}/approve`),
  rejectRegistration: (id, reason) => api.post(`admin/approvals/${id}/reject`, { reason }),
  
  // 获取所有用户（分页）
  getAllUsers: (params) => api.get('admin/users', { params }),
  
  // 统计信息
  getSystemStats: () => api.get('admin/stats'),
  getUserStatistics: () => api.get('admin/user-stats'),
  
  // 增强LDAP管理
  getLDAPConfig: () => api.get('admin/ldap/config'),
  updateLDAPConfig: (config) => api.put('admin/ldap/config', config),
  testLDAPConnection: (config) => api.post('admin/ldap/test', config),
  syncLDAPUsers: (options = {}) => api.post('admin/ldap/sync', options),
  getLDAPUsers: () => api.get('admin/ldap/users'),
  getLDAPSyncStatus: (syncId) => api.get(`admin/ldap/sync/${syncId}/status`),
  getLDAPSyncHistory: (limit = 10) => api.get(`admin/ldap/sync/history?limit=${limit}`),
};

// AI助手API
export const aiAPI = {
  // 配置管理
  getConfigs: () => api.get('/ai/configs'),
  getConfig: (id) => api.get(`/ai/configs/${id}`),
  createConfig: (config) => api.post('/ai/configs', config),
  updateConfig: (id, config) => api.put(`/ai/configs/${id}`, config),
  deleteConfig: (id) => api.delete(`/ai/configs/${id}`),

  // 对话管理
  getConversations: () => api.get('/ai/conversations'),
  getConversation: (id) => api.get(`/ai/conversations/${id}`),
  createConversation: (data) => api.post('/ai/conversations', data),
  deleteConversation: (id) => api.delete(`/ai/conversations/${id}`),
  clearConversations: () => api.delete('/ai/conversations'),

  // 消息处理（异步版本）
  sendMessage: (conversationId, message) => api.post(`/ai/conversations/${conversationId}/messages`, { message }),
  getMessages: (conversationId) => api.get(`/ai/conversations/${conversationId}/messages`),
  getMessageStatus: (messageId) => api.get(`/ai/messages/${messageId}/status`),
  stopMessage: (messageId) => api.patch(`/ai/messages/${messageId}/stop`),

  // 快速聊天（异步版本）
  quickChat: (message, context) => api.post('/ai/quick-chat', { message, context }),

  // 集群操作
  submitClusterOperation: (operation, parameters) => api.post('/ai/cluster-operations', { 
    operation, 
    parameters 
  }),
  getOperationStatus: (operationId) => api.get(`/ai/operations/${operationId}/status`),

  // 系统健康检查
  getSystemHealth: () => api.get('/ai/health'),

  // 使用统计
  getUsage: (startDate, endDate) => api.get('/ai/usage-stats', {
    params: { start_date: startDate, end_date: endDate }
  }),
  getUsageStats: (startDate, endDate) => api.get('/ai/usage-stats', {
    params: { start_date: startDate, end_date: endDate }
  }),
};

// JupyterHub API
export const jupyterHubAPI = {
  getStatus: () => api.get('/jupyterhub/status'),
  getUserTasks: () => api.get('/jupyterhub/user-tasks'),
  getTaskOutput: (taskId) => api.get(`/jupyterhub/tasks/${taskId}/output`),
};

// Slurm API
export const slurmAPI = {
  getSummary: () => api.get('/slurm/summary'),
  getNodes: () => api.get('/slurm/nodes'),
  getJobs: () => api.get('/slurm/jobs'),
  getPartitions: () => api.get('/slurm/partitions'),
  // 节点管理 API
  manageNodes: (nodeNames, action, reason) => api.post('/slurm/nodes/manage', { 
    node_names: nodeNames, 
    action, 
    reason 
  }),
  // 作业管理 API
  manageJobs: (jobIds, action, signal) => api.post('/slurm/jobs/manage', {
    job_ids: jobIds,
    action,
    signal
  }),
  // 扩缩容相关 API
  getScalingStatus: () => api.get('/slurm/scaling/status'),
  scaleUp: (nodes) => api.post('/slurm/scaling/scale-up/async', { nodes }),
  scaleDown: (nodeIds) => api.post('/slurm/scaling/scale-down', { node_ids: nodeIds }),
  getNodeTemplates: () => api.get('/slurm/node-templates'),
  createNodeTemplate: (template) => api.post('/slurm/node-templates', template),
  updateNodeTemplate: (id, template) => api.put(`/slurm/node-templates/${id}`, template),
  deleteNodeTemplate: (id) => api.delete(`/slurm/node-templates/${id}`),
  // 节点管理 API
  getNode: (nodeId) => api.get(`/slurm/nodes/${nodeId}`),
  listNodes: (clusterId) => api.get(`/slurm/nodes/cluster/${clusterId}`),
  deleteNode: (nodeId, force = false) => api.delete(`/slurm/nodes/${nodeId}`, { params: { force } }),
  deleteNodeByName: (nodeName, force = false) => api.delete(`/slurm/nodes/by-name/${nodeName}`, { params: { force } }),
  // 任务管理 API (基础)
  getTasks: (params) => api.get('/slurm/tasks', { params }),
  getProgress: (opId) => api.get(`/slurm/progress/${opId}`),
  // 增强任务管理 API
  getTaskDetail: (taskId) => api.get(`/slurm/tasks/${taskId}`),
  getTaskStatistics: (params) => api.get('/slurm/tasks/statistics', { params }),
  cancelTask: (taskId, reason) => api.post(`/slurm/tasks/${taskId}/cancel`, { reason }),
  retryTask: (taskId) => api.post(`/slurm/tasks/${taskId}/retry`),
  // SSH连接测试 API
  testSSHConnection: (nodeConfig) => api.post('/slurm/ssh/test-connection', nodeConfig),
  testBatchSSHConnection: (nodes) => api.post('/slurm/ssh/test-batch', { nodes }), // 批量测试
  // 主机初始化 API
  initializeHosts: (hosts) => api.post('/slurm/hosts/initialize', { hosts }),
};

// SaltStack API
export const saltStackAPI = {
  getStatus: () => api.get('/saltstack/status'),
  getMinions: (refresh = false) => api.get('/saltstack/minions', { params: refresh ? { refresh: 'true' } : {} }),
  // 从数据库获取作业历史（持久化数据，支持筛选和搜索）
  getJobs: (limit) => api.get('/saltstack/jobs/history', { params: { page_size: limit } }),
  // 从 Salt API 实时获取作业（仅用于特殊场景）
  getJobsFromSaltAPI: (limit) => api.get('/saltstack/jobs', { params: { limit } }),
  // 获取单个作业详情（优先从数据库查询，确保持久化）
  getJobDetail: (jid) => api.get(`/saltstack/jobs/${jid}`),
  // 通过 TaskID 获取作业详情
  getJobByTaskId: (taskId) => api.get(`/saltstack/jobs/by-task/${taskId}`),
  executeCommand: (command) => api.post('/saltstack/execute', command),
  // 自定义命令（Bash/Python）异步执行与进度
  executeCustomAsync: (payload) => api.post('/saltstack/execute-custom/async', payload),
  getProgress: (opId) => api.get(`/saltstack/progress/${opId}`),
  streamProgressUrl: (opId) => {
    const proto = window.location.protocol === 'https:' ? 'https' : 'http';
    const host = window.location.host;
    return `${proto}://${host}/api/saltstack/progress/${encodeURIComponent(opId)}/stream`;
  },
  // SaltStack 集成 API
  getSaltStackIntegration: () => api.get('/slurm/saltstack/integration'),
  deploySaltMinion: (nodeConfig) => api.post('/slurm/saltstack/deploy-minion', nodeConfig),
  testSSHConnection: (nodeConfig) => api.post('/slurm/ssh/test-connection', nodeConfig),
  executeSaltCommand: (command) => api.post('/slurm/saltstack/execute', command),
  getSaltJobs: () => api.get('/slurm/saltstack/jobs'),
  // SaltStack 状态管理
  acceptMinion: (minionId) => api.post(`/saltstack/minions/${minionId}/accept`),
  rejectMinion: (minionId) => api.post(`/saltstack/minions/${minionId}/reject`),
  deleteMinion: (minionId) => api.delete(`/saltstack/minions/${minionId}`),
  // SaltStack 批量操作
  batchExecute: (targets, command) => api.post('/saltstack/batch-execute', { targets, command }),
  getMinionDetails: (minionId) => api.get(`/saltstack/minions/${minionId}/details`),
  
  // 批量安装 Salt Minion
  batchInstallMinion: (payload) => api.post('/saltstack/batch-install', payload),
  getBatchInstallTask: (taskId) => api.get(`/saltstack/batch-install/${taskId}`),
  listBatchInstallTasks: (params) => api.get('/saltstack/batch-install', { params }),
  calculateParallel: (hostCount, maxParallel) => api.get('/saltstack/batch-install/calculate-parallel', { 
    params: { host_count: hostCount, max_parallel: maxParallel } 
  }),
  getBatchInstallStreamUrl: (taskId) => {
    const proto = window.location.protocol === 'https:' ? 'https' : 'http';
    const host = window.location.host;
    return `${proto}://${host}/api/saltstack/batch-install/${encodeURIComponent(taskId)}/stream`;
  },

  // SSH 测试（含 sudo 权限检查）
  testSSH: (config) => api.post('/saltstack/ssh/test', config),
  batchTestSSH: (payload) => api.post('/saltstack/ssh/test-batch', payload),

  // 主机文件解析
  parseHostFile: (content, filename) => api.post('/saltstack/hosts/parse', { content, filename }),
  // 调试解析接口（返回详细解析过程）
  parseHostFileDebug: (content, filename) => api.post('/saltstack/hosts/parse/debug', { content, filename }),

  // Minion 管理（删除、卸载）
  removeMinionKey: (minionId, force = false) => api.delete(`/saltstack/minion/${minionId}`, { params: { force } }),
  /**
   * 批量删除 Minion 密钥
   * @param {string[]} minionIds - 要删除的 Minion ID 列表
   * @param {Object} options - 删除选项
   * @param {boolean} options.force - 是否强制删除（不等待确认）
   * @param {boolean} options.uninstall - 是否通过 SSH 卸载 salt-minion 组件
   * @param {string} options.ssh_username - SSH 用户名
   * @param {string} options.ssh_password - SSH 密码
   * @param {string} options.ssh_key_path - SSH 密钥路径
   * @param {number} options.ssh_port - SSH 端口（默认22）
   * @param {boolean} options.use_sudo - 是否使用 sudo
   */
  batchRemoveMinionKeys: (minionIds, options = {}) => {
    const { force = false, uninstall = false, ssh_username, ssh_password, ssh_key_path, ssh_port, use_sudo } = options;
    return api.post('/saltstack/minion/batch-delete', { 
      minion_ids: minionIds, 
      force,
      uninstall,
      ssh_username,
      ssh_password,
      ssh_key_path,
      ssh_port,
      use_sudo,
    });
  },
  uninstallMinion: (minionId, sshConfig) => api.post(`/saltstack/minion/${minionId}/uninstall`, sshConfig),
  
  // 删除任务管理（软删除 + 异步真实删除）
  getPendingDeleteMinions: () => api.get('/saltstack/minion/pending-deletes'),
  getDeleteTaskStatus: (minionId) => api.get(`/saltstack/minion/delete-tasks/${minionId}`),
  getDeleteTaskLogs: (minionId) => api.get(`/saltstack/minion/delete-tasks/${minionId}/logs`),
  listDeleteTasks: (params) => api.get('/saltstack/minion/delete-tasks', { params }),
  cancelDeleteTask: (minionId) => api.post(`/saltstack/minion/delete-tasks/${minionId}/cancel`),
  retryDeleteTask: (minionId) => api.post(`/saltstack/minion/delete-tasks/${minionId}/retry`),

  // Minion 分组管理
  listMinionGroups: () => api.get('/saltstack/groups'),
  createMinionGroup: (groupData) => api.post('/saltstack/groups', groupData),
  updateMinionGroup: (id, groupData) => api.put(`/saltstack/groups/${id}`, groupData),
  deleteMinionGroup: (id) => api.delete(`/saltstack/groups/${id}`),
  getGroupMinions: (id) => api.get(`/saltstack/groups/${id}/minions`),
  setMinionGroup: (minionId, groupName) => api.post('/saltstack/minions/set-group', { minion_id: minionId, group_name: groupName }),
  batchSetMinionGroups: (minionGroups) => api.post('/saltstack/minions/batch-set-groups', { minion_groups: minionGroups }),

  // 批量为 Minion 安装 Categraf（通过 Salt State）
  installCategrafOnMinions: (payload) => api.post('/saltstack/minions/install-categraf', payload),
  getCategrafInstallStreamUrl: (taskId) => {
    const proto = window.location.protocol === 'https:' ? 'https' : 'http';
    const host = window.location.host;
    return `${proto}://${host}/api/saltstack/minions/install-categraf/${encodeURIComponent(taskId)}/stream`;
  },

  // 节点指标采集
  getNodeMetrics: (minionId = '') => api.get('/saltstack/node-metrics', { params: minionId ? { minion_id: minionId } : {} }),
  deployNodeMetricsState: (target, interval = 3) => api.post('/saltstack/node-metrics/deploy', { target, interval }),
  
  // IB 端口忽略管理
  getIBPortIgnores: (minionId = '') => api.get('/saltstack/ib-ignores', { params: minionId ? { minion_id: minionId } : {} }),
  addIBPortIgnore: (minionId, portName, portNum = 1, reason = '') => api.post('/saltstack/ib-ignores', { minion_id: minionId, port_name: portName, port_num: portNum, reason }),
  removeIBPortIgnore: (minionId, portName, portNum = 0) => api.delete(`/saltstack/ib-ignores/${encodeURIComponent(minionId)}/${encodeURIComponent(portName)}`, { params: portNum ? { port_num: portNum } : {} }),
  getIBPortAlerts: () => api.get('/saltstack/ib-alerts'),

  // 作业配置管理（清理策略）
  getJobConfig: () => api.get('/saltstack/jobs/config'),
  updateJobConfig: (config) => api.put('/saltstack/jobs/config', config),
  triggerJobCleanup: () => api.post('/saltstack/jobs/cleanup'),
};

// 增强用户管理API
export const enhancedUserAPI = {
  // 增强用户管理功能
  getEnhancedUsers: () => api.get('/admin/enhanced-users'),
  createEnhancedUser: (userData) => api.post('/admin/enhanced-users', userData),
  resetUserPassword: (id) => api.post(`/admin/enhanced-users/${id}/reset-password`),
  
  // 用户组管理
  getUserGroups: () => api.get('/user-groups'),
  createUserGroup: (groupData) => api.post('/user-groups', groupData),
  updateUserGroup: (id, groupData) => api.put(`/user-groups/${id}`, groupData),
  deleteUserGroup: (id) => api.delete(`/user-groups/${id}`),
  addUserToGroup: (groupId, userId) => api.post(`/user-groups/${groupId}/users/${userId}`),
  removeUserFromGroup: (groupId, userId) => api.delete(`/user-groups/${groupId}/users/${userId}`),
  
  // 用户统计和分析
  getUserStatistics: () => api.get('/admin/user-stats'),
  getUserActivityReport: (userId, period) => api.get(`/admin/users/${userId}/activity`, { 
    params: { period } 
  }),
};

// LDAP API (保持向后兼容)
export const ldapAPI = {
  getConfig: () => api.get('/ldap/config'),
  updateConfig: (config) => api.put('/ldap/config', config),
  testConnection: () => api.post('/ldap/test-connection'),
  syncUsers: (options) => api.post('/ldap/sync-users', options),
  searchUsers: (query) => api.get('/ldap/search-users', { params: { query } }),
  getUserGroups: () => api.get('/ldap/groups'),
  syncGroups: () => api.post('/ldap/sync-groups'),
};

// 导航配置API
export const navigationAPI = {
  getUserNavigationConfig: () => api.get('/navigation/config'),
  saveUserNavigationConfig: (config) => api.put('/navigation/config', { items: config }),
  resetNavigationConfig: () => api.delete('/navigation/config'),
  getDefaultNavigationConfig: () => api.get('/navigation/default'),
};

// JupyterLab模板管理API
export const jupyterLabAPI = {
  // 模板管理
  getTemplates: (includeInactive = false) => api.get('/jupyterlab/templates', { 
    params: { include_inactive: includeInactive } 
  }),
  getTemplate: (id) => api.get(`/jupyterlab/templates/${id}`),
  createTemplate: (template) => api.post('/jupyterlab/templates', template),
  updateTemplate: (id, template) => api.put(`/jupyterlab/templates/${id}`, template),
  deleteTemplate: (id) => api.delete(`/jupyterlab/templates/${id}`),
  cloneTemplate: (id, name) => api.post(`/jupyterlab/templates/${id}/clone`, { name }),
  setDefaultTemplate: (id) => api.post(`/jupyterlab/templates/${id}/default`),
  exportTemplate: (id) => api.get(`/jupyterlab/templates/${id}/export`),
  importTemplate: (data) => api.post('/jupyterlab/templates/import', data),
  
  // 实例管理
  getInstances: () => api.get('/jupyterlab/instances'),
  getInstance: (id) => api.get(`/jupyterlab/instances/${id}`),
  createInstance: (instance) => api.post('/jupyterlab/instances', instance),
  deleteInstance: (id) => api.delete(`/jupyterlab/instances/${id}`),
  
  // 管理员功能
  createPredefinedTemplates: () => api.post('/jupyterlab/admin/create-predefined-templates'),
};

// Add Files API for remote file browsing and transfer
export const filesAPI = {
  // List directory contents on a given cluster
  list: (cluster, path) => api.get('/files', { params: { cluster, path } }),
  // Download a file (returns blob)
  download: (cluster, path) => api.get('/files/download', {
    params: { cluster, path },
    responseType: 'blob',
  }),
  // Upload a file (multipart)
  upload: (cluster, path, file) => {
    const form = new FormData();
    form.append('cluster', cluster);
    form.append('path', path);
    form.append('file', file);
    return api.post('/files/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  },
};

// 对象存储相关API
export const objectStorageAPI = {
  // 获取所有存储配置
  getConfigs: () => api.get('/object-storage/configs'),

  // 获取单个存储配置
  getConfig: (id) => api.get(`/object-storage/configs/${id}`),

  // 创建存储配置
  createConfig: (data) => api.post('/object-storage/configs', data),

  // 更新存储配置
  updateConfig: (id, data) => api.put(`/object-storage/configs/${id}`, data),

  // 删除存储配置
  deleteConfig: (id) => api.delete(`/object-storage/configs/${id}`),

  // 设置激活配置
  setActiveConfig: (id) => api.post(`/object-storage/configs/${id}/activate`),

  // 测试连接
  testConnection: (config) => api.post('/object-storage/test-connection', config),

  // 检查连接状态
  checkConnection: (id) => api.get(`/object-storage/configs/${id}/status`),

  // 获取存储统计信息
  getStatistics: (id) => api.get(`/object-storage/configs/${id}/statistics`),

  // 获取存储桶列表
  getBuckets: (id) => api.get(`/object-storage/configs/${id}/buckets`),

  // 创建存储桶
  createBucket: (id, bucketName) => api.post(`/object-storage/configs/${id}/buckets`, { name: bucketName }),

  // 删除存储桶
  deleteBucket: (id, bucketName) => api.delete(`/object-storage/configs/${id}/buckets/${bucketName}`),

  // 获取对象列表
  getObjects: (id, bucketName, prefix = '') => api.get(`/object-storage/configs/${id}/buckets/${bucketName}/objects`, {
    params: { prefix }
  }),

  // 上传对象
  uploadObject: (id, bucketName, file, key) => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('key', key);
    
    return api.post(`/object-storage/configs/${id}/buckets/${bucketName}/objects`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
  },

  // 下载对象
  downloadObject: (id, bucketName, key) => api.get(`/object-storage/configs/${id}/buckets/${bucketName}/objects/${key}/download`, {
    responseType: 'blob'
  }),

  // 删除对象
  deleteObject: (id, bucketName, key) => api.delete(`/object-storage/configs/${id}/buckets/${bucketName}/objects/${key}`),

  // 获取预签名URL
  getPresignedUrl: (id, bucketName, key, expiry = 3600) => api.post(`/object-storage/configs/${id}/buckets/${bucketName}/objects/${key}/presign`, {
    expiry
  })
};

// 角色模板 API
export const roleTemplateAPI = {
  // 获取所有角色模板
  list: (activeOnly = false) => api.get('/rbac/role-templates', { params: { active_only: activeOnly } }),

  // 获取角色模板详情
  get: (id) => api.get(`/rbac/role-templates/${id}`),

  // 创建角色模板
  create: (data) => api.post('/rbac/role-templates', data),

  // 更新角色模板
  update: (id, data) => api.put(`/rbac/role-templates/${id}`, data),

  // 删除角色模板
  delete: (id) => api.delete(`/rbac/role-templates/${id}`),

  // 同步角色模板到角色
  sync: () => api.post('/rbac/role-templates/sync'),

  // 获取可用资源列表
  getResources: () => api.get('/rbac/resources'),

  // 获取可用操作列表
  getVerbs: () => api.get('/rbac/verbs'),
};

// RBAC API
export const rbacAPI = {
  // 检查权限
  checkPermission: (data) => api.post('/rbac/check-permission', data),

  // 角色管理
  getRoles: () => api.get('/rbac/roles'),
  getRole: (id) => api.get(`/rbac/roles/${id}`),
  createRole: (data) => api.post('/rbac/roles', data),
  updateRole: (id, data) => api.put(`/rbac/roles/${id}`, data),
  deleteRole: (id) => api.delete(`/rbac/roles/${id}`),

  // 用户组管理
  getGroups: () => api.get('/rbac/groups'),
  createGroup: (data) => api.post('/rbac/groups', data),
  addUserToGroup: (groupId, userId) => api.post(`/rbac/groups/${groupId}/users/${userId}`),
  removeUserFromGroup: (groupId, userId) => api.delete(`/rbac/groups/${groupId}/users/${userId}`),

  // 权限管理
  getPermissions: () => api.get('/rbac/permissions'),
  createPermission: (data) => api.post('/rbac/permissions', data),

  // 角色分配
  assignRole: (data) => api.post('/rbac/assign-role', data),
  revokeRole: (data) => api.delete('/rbac/revoke-role', { data }),
};

// 安全管理 API
export const securityAPI = {
  // IP 黑名单管理
  getIPBlacklist: (params) => api.get('/security/ip-blacklist', { params }),
  addIPBlacklist: (data) => api.post('/security/ip-blacklist', data),
  updateIPBlacklist: (id, data) => api.put(`/security/ip-blacklist/${id}`, data),
  deleteIPBlacklist: (id) => api.delete(`/security/ip-blacklist/${id}`),
  batchDeleteIPBlacklist: (ids) => api.post('/security/ip-blacklist/batch-delete', { ids }),

  // IP 白名单管理
  getIPWhitelist: (params) => api.get('/security/ip-whitelist', { params }),
  addIPWhitelist: (data) => api.post('/security/ip-whitelist', data),
  deleteIPWhitelist: (id) => api.delete(`/security/ip-whitelist/${id}`),

  // 二次认证 (2FA) 管理
  get2FAStatus: () => api.get('/security/2fa/status'),
  setup2FA: () => api.post('/security/2fa/setup'),
  enable2FA: (data) => api.post('/security/2fa/enable', data),
  disable2FA: (data) => api.post('/security/2fa/disable', data),
  verify2FA: (data) => api.post('/security/2fa/verify', data),
  regenerateRecoveryCodes: () => api.post('/security/2fa/recovery-codes'),

  // 管理员2FA管理（为其他用户管理2FA）
  admin2FAStatus: (userId) => api.get(`/security/admin/2fa/${userId}/status`),
  adminEnable2FA: (userId) => api.post(`/security/admin/2fa/${userId}/enable`),
  adminDisable2FA: (userId) => api.post(`/security/admin/2fa/${userId}/disable`),

  // OAuth 第三方登录配置
  getOAuthProviders: () => api.get('/security/oauth/providers'),
  getOAuthProvider: (id) => api.get(`/security/oauth/providers/${id}`),
  updateOAuthProvider: (id, data) => api.put(`/security/oauth/providers/${id}`, data),

  // 全局安全配置
  getSecurityConfig: () => api.get('/security/config'),
  updateSecurityConfig: (data) => api.put('/security/config', data),

  // 安全审计日志
  getAuditLogs: (params) => api.get('/security/audit-logs', { params }),
};

export default api;