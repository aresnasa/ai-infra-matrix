#!/usr/bin/env python3
"""
演示测试 - 展示如何使用测试框架
"""

import sys
import os

# 添加当前目录到Python路径
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from utils import TestSession, TestReporter
from config import DEFAULT_CONFIG

def demo_test():
    """演示基本测试流程"""
    reporter = TestReporter(verbose=True)
    reporter.info("🎭 演示测试开始")
    
    # 创建测试会话
    session = TestSession(DEFAULT_CONFIG["base_url"])
    
    # 测试登录
    reporter.info("测试登录功能...")
    creds = DEFAULT_CONFIG["credentials"]
    success, message = session.login(creds["username"], creds["password"])
    
    if success:
        reporter.success(f"登录成功: {message}")
    else:
        reporter.error(f"登录失败: {message}")
        return False
    
    # 测试API访问
    reporter.info("测试API访问...")
    try:
        response = session.get("/api/auth/verify")
        if response.status_code == 200:
            reporter.success("API访问成功")
        else:
            reporter.warning(f"API访问异常: {response.status_code}")
    except Exception as e:
        reporter.error(f"API访问失败: {e}")
        return False
    
    # 测试Gitea访问
    reporter.info("测试Gitea SSO...")
    try:
        response = session.get("/gitea/user/login", allow_redirects=False)
        if response.status_code in [302, 303]:
            location = response.headers.get('Location', '')
            reporter.success(f"SSO重定向成功: {location}")
        elif response.status_code == 200:
            reporter.success("直接访问成功")
        else:
            reporter.warning(f"访问状态: {response.status_code}")
    except Exception as e:
        reporter.error(f"Gitea访问失败: {e}")
        return False
    
    reporter.success("🎉 演示测试完成！")
    return True

if __name__ == "__main__":
    success = demo_test()
    sys.exit(0 if success else 1)
