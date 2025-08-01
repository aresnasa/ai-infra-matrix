
  // 检查token是否有效（本地检查，避免频繁请求后端）
  const isTokenValid = () => {
    const token = localStorage.getItem('token');
    const expires_at = localStorage.getItem('token_expires');

    if (!token || !expires_at) {
      console.log('Token或过期时间不存在');
      return false;
    }

    const expiryTime = new Date(expires_at).getTime();
    const currentTime = new Date().getTime();
    const bufferTime = 5 * 60 * 1000; // 5分钟缓冲时间

    if (currentTime + bufferTime >= expiryTime) {
      console.log('Token即将过期或已过期');
      return false;
    }

    return true;
  };

  // 优化的初始化认证函数
  const initializeAuth = async () => {
    console.log('=== 开始初始化认证状态 ===');

    const token = localStorage.getItem('token');
    console.log('检查token:', token ? '存在' : '不存在');

    if (!token) {
      console.log('无token，用户未认证');
      clearUserState();
      setAuthChecked(true);
      setLoading(false);
      return;
    }

    // 先检查token本地有效性
    if (!isTokenValid()) {
      console.log('Token已过期，尝试刷新...');

      try {
        // 尝试刷新token
        const response = await authAPI.refreshToken();
        const { token: newToken, expires_at } = response.data;

        localStorage.setItem('token', newToken);
        localStorage.setItem('token_expires', expires_at);
        console.log('Token刷新成功');

        // 刷新后验证用户信息
        await verifyUserWithBackend();
      } catch (error) {
        console.log('Token刷新失败，需要重新登录:', error.message);
        clearUserState();
      }
    } else {
      // Token有效，检查是否已有用户信息
      const savedUser = localStorage.getItem('user');
      if (savedUser) {
        try {
          const userData = JSON.parse(savedUser);
          setUser(userData);
          setPermissionsLoaded(true);
          console.log('使用缓存的用户信息');
        } catch (error) {
          console.log('缓存用户信息解析失败，重新获取');
          await verifyUserWithBackend();
        }
      } else {
        // 没有缓存用户信息，从后端获取
        await verifyUserWithBackend();
      }
    }

    setAuthChecked(true);
    setLoading(false);
    console.log('=== 认证状态初始化完成 ===');
  };

  // 优化的后端验证函数
  const verifyUserWithBackend = async () => {
    try {
      console.log('验证token并获取用户权限...');

      const response = await authAPI.getProfile();
      const userData = response.data;

      console.log('后端返回用户数据:', userData);
      console.log('用户角色:', userData.role);
      console.log('用户权限组:', userData.roles);

      // 设置用户状态
      setUser(userData);
      setPermissionsLoaded(true);

      // 更新本地存储
      localStorage.setItem('user', JSON.stringify(userData));

      console.log('✅ 用户权限验证成功');

    } catch (error) {
      console.log('❌ Token验证失败:', error.message);

      // 判断是否是token过期错误
      if (error.response?.status === 401) {
        console.log('认证失败，可能token已过期');
        // 尝试token刷新
        try {
          const refreshResponse = await authAPI.refreshToken();
          const { token: newToken, expires_at } = refreshResponse.data;

          localStorage.setItem('token', newToken);
          localStorage.setItem('token_expires', expires_at);
          console.log('Token刷新成功，重新验证用户信息');

          // 递归调用，但增加计数器防止无限循环
          if (!this.retryCount) this.retryCount = 0;
          if (this.retryCount < 2) {
            this.retryCount++;
            await verifyUserWithBackend();
            return;
          }
        } catch (refreshError) {
          console.log('Token刷新失败:', refreshError.message);
        }
      }

      console.log('清除认证状态...');
      clearUserState();
    }
  };
