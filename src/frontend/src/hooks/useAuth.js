import { useState, useEffect, useCallback } from 'react';
import { authAPI } from '../services/api';

/**
 * 认证状态管理Hook
 * 提供完整的认证状态管理，确保权限信息的实时性和准确性
 */
export const useAuth = () => {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [authChecked, setAuthChecked] = useState(false);

  /**
   * 检查认证状态
   * 确保从后端API获取最新的用户权限信息
   */
  const checkAuthStatus = useCallback(async () => {
    const token = localStorage.getItem('token');
    
    console.log('useAuth - checking auth status, token:', token ? 'exists' : 'missing');
    
    if (token) {
      try {
        console.log('useAuth - verifying token with backend...');
        const response = await authAPI.getProfile();
        const userData = response.data;
        
        console.log('useAuth - profile response:', userData);
        console.log('useAuth - user role:', userData.role);
        console.log('useAuth - user roles:', userData.roles);
        console.log('useAuth - user role_template:', userData.role_template);
        
        // 使用API返回的最新用户信息
        setUser(userData);
        
        // 更新localStorage中的用户信息为最新状态
        localStorage.setItem('user', JSON.stringify(userData));
        
        console.log('useAuth - user authenticated successfully');
      } catch (error) {
        console.log('useAuth - token verification failed:', error);
        
        // token无效或API调用失败，清除认证状态
        localStorage.removeItem('token');
        localStorage.removeItem('user');
        localStorage.removeItem('token_expires');
        setUser(null);
      }
    } else {
      console.log('useAuth - no token found, user not authenticated');
      setUser(null);
    }
    
    setAuthChecked(true);
    setLoading(false);
  }, []);

  /**
   * 处理用户登录
   * @param {Object} userData - 用户数据
   */
  const handleLogin = useCallback(async (userData) => {
    console.log('useAuth - handling login for user:', userData);
    
    try {
      // 登录后立即获取最新的用户权限信息
      const response = await authAPI.getProfile();
      const latestUserData = response.data;
      
      console.log('useAuth - got latest user data after login:', latestUserData);
      
      setUser(latestUserData);
      localStorage.setItem('user', JSON.stringify(latestUserData));
    } catch (error) {
      console.warn('useAuth - failed to get latest profile after login, using provided data:', error);
      // 如果获取最新信息失败，使用提供的用户数据
      setUser(userData);
      localStorage.setItem('user', JSON.stringify(userData));
    }
    
    setAuthChecked(true);
  }, []);

  /**
   * 处理用户登出
   */
  const handleLogout = useCallback(() => {
    console.log('useAuth - handling logout');
    authAPI.logout();
    setUser(null);
    setAuthChecked(false);
  }, []);

  /**
   * 刷新用户权限信息
   * 用于权限变更后立即同步最新状态
   */
  const refreshUserPermissions = useCallback(async () => {
    if (!user) return;
    
    try {
      console.log('useAuth - refreshing user permissions...');
      const response = await authAPI.getProfile();
      const updatedUserData = response.data;
      
      console.log('useAuth - refreshed user data:', updatedUserData);
      
      setUser(updatedUserData);
      localStorage.setItem('user', JSON.stringify(updatedUserData));
      
      return updatedUserData;
    } catch (error) {
      console.error('useAuth - failed to refresh user permissions:', error);
      throw error;
    }
  }, [user]);

  /**
   * 检查用户是否为管理员
   */
  const isAdmin = useCallback(() => {
    if (!user) return false;
    
    const adminCheck = user.role === 'admin' || user.role === 'super-admin' || 
      (user.roles && user.roles.some(role => role.name === 'admin' || role.name === 'super-admin'));
    
    console.log('useAuth - admin check:', {
      user: user.username,
      role: user.role,
      roles: user.roles,
      isAdmin: adminCheck
    });
    
    return adminCheck;
  }, [user]);

  // 初始化时检查认证状态
  useEffect(() => {
    checkAuthStatus();
  }, [checkAuthStatus]);

  return {
    user,
    loading,
    authChecked,
    isAdmin: isAdmin(),
    handleLogin,
    handleLogout,
    checkAuthStatus,
    refreshUserPermissions
  };
};
