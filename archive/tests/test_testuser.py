#!/usr/bin/env python3
import requests
from bs4 import BeautifulSoup

def test_testuser_login():
    session = requests.Session()
    
    # 获取登录页面
    print("🔍 测试testuser登录...")
    login_page = session.get('http://localhost:8080/jupyter/hub/login')
    print(f'登录页面状态: {login_page.status_code}')
    
    # 提取XSRF token
    soup = BeautifulSoup(login_page.text, 'html.parser')
    xsrf_token = soup.find('input', {'name': '_xsrf'})['value']
    
    # 测试testuser登录
    login_data = {
        'username': 'testuser',
        'password': 'any_password',  # DummyAuthenticator allows any password
        '_xsrf': xsrf_token
    }
    
    login_response = session.post('http://localhost:8080/jupyter/hub/login', data=login_data, allow_redirects=False)
    print(f'testuser登录状态: {login_response.status_code}')
    print(f'testuser重定向到: {login_response.headers.get("Location", "无重定向")}')
    
    # 测试认证状态
    auth_check = session.get('http://localhost:8080/jupyter/hub/api/user')
    print(f'testuser认证状态: {auth_check.status_code}')
    if auth_check.status_code == 200:
        user_info = auth_check.json()
        print(f'✅ testuser用户信息: {user_info.get("name", "未知")}')
        print(f'   - 管理员权限: {user_info.get("admin", False)}')
        print(f'   - 服务器状态: {user_info.get("servers", {})}')
    
    return auth_check.status_code == 200

if __name__ == "__main__":
    success = test_testuser_login()
    print(f"\n{'🎉 测试成功' if success else '❌ 测试失败'}")
