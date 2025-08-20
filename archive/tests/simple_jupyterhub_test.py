#!/usr/bin/env python3
"""简化的JupyterHub登录测试脚本"""

import requests
from bs4 import BeautifulSoup
import re

def simple_login_test():
    LOGIN_URL = "http://localhost:8080/jupyter/hub/login"

    session = requests.Session()

    # 1. 获取登录页面
    response = session.get(LOGIN_URL)
    print(f"登录页面状态: {response.status_code}")

    if response.status_code != 200:
        return False

    # 2. 提取CSRF token
    soup = BeautifulSoup(response.text, 'html.parser')
    csrf_input = soup.find('input', {'name': '_xsrf'})
    csrf_token = csrf_input.get('value') if csrf_input else None

    print(f"CSRF Token: {'找到' if csrf_token else '未找到'}")

    # 3. 执行登录
    login_data = {
        'username': 'admin',
        'password': 'password'
    }

    if csrf_token:
        login_data['_xsrf'] = csrf_token

    headers = {
        'Content-Type': 'application/x-www-form-urlencoded',
        'Referer': LOGIN_URL
    }

    login_response = session.post(LOGIN_URL, data=login_data, headers=headers, allow_redirects=False)
    print(f"登录响应: {login_response.status_code}")

    if login_response.status_code in [302, 303]:
        redirect_url = login_response.headers.get('Location', '')
        print(f"重定向到: {redirect_url}")
        return '/hub/home' in redirect_url or '/hub/spawn' in redirect_url

    return False

if __name__ == "__main__":
    success = simple_login_test()
    print(f"登录测试: {'成功' if success else '失败'}")
