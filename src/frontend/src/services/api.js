import axios from 'axios';

// 创建axios实例
const api = axios.create({
  baseURL: process.env.REACT_APP_API_URL || '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// 认证API
export const authAPI = {
  login: (credentials) => api.post('/auth/login', credentials),
  logout: () => api.post('/auth/logout'),
  getCurrentUser: () => api.get('/auth/me'),
  getProfile: () => api.get('/auth/me'), // 添加getProfile方法，指向相同的端点
  refreshToken: () => api.post('/auth/refresh'),
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
  getPods: (id, namespace) => api.get(`/kubernetes/clusters/${id}/namespaces/${namespace || 'default'}/pods`),
  getDeployments: (id, namespace) => api.get(`/kubernetes/clusters/${id}/namespaces/${namespace || 'default'}/deployments`),
  getServices: (id, namespace) => api.get(`/kubernetes/clusters/${id}/namespaces/${namespace || 'default'}/services`),
  getNodesDetail: (id) => api.get(`/kubernetes/clusters/${id}/nodes/detail`),
  getEvents: (id, namespace) => api.get(`/kubernetes/clusters/${id}/namespaces/${namespace || ''}/events`),

  // 操作类（按需在后端实现）
  scaleDeployment: (id, namespace, name, replicas) => api.post(`/kubernetes/clusters/${id}/namespaces/${namespace}/deployments/${name}/scale`, { replicas }),
  deleteResource: (id, namespace, kind, name) => api.delete(`/kubernetes/clusters/${id}/namespaces/${namespace}/${kind.toLowerCase()}s/${name}`),
  getPodLogs: (id, namespace, pod, container) => api.get(`/kubernetes/clusters/${id}/namespaces/${namespace}/pods/${pod}/logs`, { params: { container } }),
  buildPodExecWsUrl: (id, namespace, pod, container, command) => {
    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const host = window.location.host;
    const cmdQuery = command ? `&command=${encodeURIComponent(command)}` : '';
    return `${proto}://${host}/api/kubernetes/clusters/${id}/namespaces/${namespace}/pods/${encodeURIComponent(pod)}/exec?container=${encodeURIComponent(container || '')}${cmdQuery}`;
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
  getUsers: () => api.get('/admin/users'),
  createUser: (userData) => api.post('/admin/users', userData),
  updateUser: (id, userData) => api.put(`/admin/users/${id}`, userData),
  deleteUser: (id) => api.delete(`/admin/users/${id}`),
  
  // 项目管理
  getProjects: () => api.get('/admin/projects'),
  updateProject: (id, projectData) => api.put(`/admin/projects/${id}`, projectData),
  deleteProject: (id) => api.delete(`/admin/projects/${id}`),
  
  // 系统设置
  getSystemSettings: () => api.get('/admin/settings'),
  updateSystemSettings: (settings) => api.put('/admin/settings', settings),
  
  // LDAP设置
  getLDAPSettings: () => api.get('/admin/ldap'),
  updateLDAPSettings: (settings) => api.put('/admin/ldap', settings),
  testLDAPConnection: (settings) => api.post('/admin/ldap/test', settings),
  
  // 认证设置
  getAuthSettings: () => api.get('/admin/auth'),
  updateAuthSettings: (settings) => api.put('/admin/auth', settings),
  
  // 回收站
  getTrashItems: () => api.get('/admin/trash'),
  restoreTrashItem: (id) => api.post(`/admin/trash/${id}/restore`),
  permanentDeleteTrashItem: (id) => api.delete(`/admin/trash/${id}/permanent`),
  
  // 增强用户管理功能
  getUserWithAuthSource: (id) => api.get(`/admin/users/${id}/auth-source`),
  resetUserPassword: (id) => api.post(`/admin/users/${id}/reset-password`),
  
  // 统计信息
  getSystemStats: () => api.get('/admin/stats'),
  getUserStatistics: () => api.get('/admin/user-stats'),
  
  // 增强LDAP管理
  getLDAPConfig: () => api.get('/admin/ldap/config'),
  updateLDAPConfig: (config) => api.put('/admin/ldap/config', config),
  testLDAPConnection: (config) => api.post('/admin/ldap/test', config),
  syncLDAPUsers: (options = {}) => api.post('/admin/ldap/sync', options),
  getLDAPSyncStatus: (syncId) => api.get(`/admin/ldap/sync/${syncId}/status`),
  getLDAPSyncHistory: (limit = 10) => api.get(`/admin/ldap/sync/history?limit=${limit}`),
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
};

// SaltStack API
export const saltStackAPI = {
  getStatus: () => api.get('/saltstack/status'),
  getMinions: () => api.get('/saltstack/minions'),
  getJobs: (limit) => api.get('/saltstack/jobs', { params: { limit } }),
  executeCommand: (command) => api.post('/saltstack/execute', command),
};

// Dashboard API
export const dashboardAPI = {
  getUserDashboard: () => api.get('/dashboard'),
  updateDashboard: (config) => api.put('/dashboard', config),
  resetDashboard: () => api.delete('/dashboard'),
  
  // 增强功能
  getUserDashboardEnhanced: () => api.get('/dashboard/enhanced'),
  getDashboardStats: () => api.get('/dashboard/stats'),
  cloneDashboard: (sourceUserId) => api.post(`/dashboard/clone/${sourceUserId}`),
  exportDashboard: () => api.get('/dashboard/export'),
  importDashboard: (config, overwrite = false) => api.post('/dashboard/import', { config, overwrite }),
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

export default api;