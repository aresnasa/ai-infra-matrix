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
  getUsers: () => api.get('/users'),
  createUser: (userData) => api.post('/users', userData),
  updateUser: (id, userData) => api.put(`/users/${id}`, userData),
  deleteUser: (id) => api.delete(`/users/${id}`),
  getUserProfile: () => api.get('/users/profile'),
  updateUserProfile: (profileData) => api.put('/users/profile', profileData),
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

export default api;