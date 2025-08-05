#!/usr/bin/env python3
"""
JupyterHub自动登录脚本
该脚本创建一个自动登录端点，接收JWT token并自动完成JupyterHub登录
"""

import requests
import sys
import time

def auto_login_jupyterhub(jwt_token, username="admin"):
    """
    使用JWT token自动登录JupyterHub
    """

    print(f"🔐 开始自动登录JupyterHub...")
    print(f"👤 用户: {username}")
    print(f"🎫 Token: {jwt_token[:30]}...")

    # 创建session
    session = requests.Session()

    # 方法1: 直接POST到JupyterHub登录端点
    try:
        login_data = {
            "username": username,
            "token": jwt_token
        }

        # 先获取登录页面以获取CSRF token
        login_page = session.get("http://localhost:8080/jupyter/hub/login", timeout=5)

        if login_page.status_code == 200:
            print("✅ 获取登录页面成功")

            # 提取CSRF token（如果需要的话）
            import re
            csrf_match = re.search(r'name="_xsrf" value="([^"]*)"', login_page.text)
            if csrf_match:
                csrf_token = csrf_match.group(1)
                login_data["_xsrf"] = csrf_token
                print(f"🔐 获取CSRF token: {csrf_token[:20]}...")

            # 提交登录请求
            login_response = session.post(
                "http://localhost:8080/jupyter/hub/login",
                data=login_data,
                timeout=10,
                allow_redirects=True
            )

            print(f"📊 登录响应状态: {login_response.status_code}")

            if login_response.status_code == 200:
                # 检查是否登录成功
                if 'logout' in login_response.text.lower() or 'spawn' in login_response.text.lower():
                    print("🎉 JupyterHub自动登录成功！")
                    return True
                else:
                    print("⚠️  登录页面返回但状态未知")
                    # 可以尝试访问用户页面确认
                    user_page = session.get(f"http://localhost:8080/jupyter/hub/user/{username}/", timeout=5)
                    if user_page.status_code == 200 or user_page.status_code == 302:
                        print("✅ 用户页面可访问，登录可能成功")
                        return True

            else:
                print(f"❌ 登录失败，状态码: {login_response.status_code}")

        else:
            print(f"❌ 无法获取登录页面: {login_page.status_code}")

    except Exception as e:
        print(f"❌ 自动登录异常: {e}")

    return False

if __name__ == "__main__":
    # 测试自动登录
    test_token = "REPLACE_WITH_REAL_TOKEN"
    success = auto_login_jupyterhub(test_token)
    sys.exit(0 if success else 1)
