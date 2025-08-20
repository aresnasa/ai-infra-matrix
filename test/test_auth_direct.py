#!/usr/bin/env python3

import requests
import json

# 测试auth_request问题的脚本

def test_auth_request():
    """测试auth_request是否真的执行了"""
    
    # 1. 获取新的token
    login_response = requests.post('http://localhost:8080/api/auth/login', 
                                 json={"username": "admin", "password": "admin123"})
    if login_response.status_code != 200:
        print(f"❌ 登录失败: {login_response.status_code}")
        return False
    
    token = login_response.json()['token']
    print(f"✅ 获取到token: {token[:50]}...")
    
    # 2. 测试nginx通过api路由调用后端API
    headers = {'Authorization': f'Bearer {token}'}
    api_response = requests.get('http://localhost:8080/api/auth/verify', headers=headers)
    print(f"🔍 通过nginx调用后端API状态: {api_response.status_code}")
    if api_response.status_code == 200:
        print(f"✅ 后端返回的用户头: X-User={api_response.headers.get('X-User', 'None')}")
        print(f"✅ 后端返回的邮箱头: X-Email={api_response.headers.get('X-Email', 'None')}")
    else:
        print(f"❌ 后端API调用失败: {api_response.text[:200] if api_response.text else 'No content'}")
    
    # 3. 测试nginx debug端点
    debug_response = requests.get('http://localhost:8080/debug/verify', headers=headers)
    print(f"🔍 Nginx debug端点状态: {debug_response.status_code}")
    print(f"🔍 Debug响应头:")
    for key, value in debug_response.headers.items():
        if key.startswith('X-Debug'):
            print(f"   {key}: {value}")
    
    # 4. 检查Gitea登录页面是否还要求密码
    gitea_response = requests.get('http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin', 
                                headers=headers)
    print(f"🔍 Gitea登录页面状态: {gitea_response.status_code}")
    
    # 检查是否包含登录表单
    if 'form' in gitea_response.text.lower() and 'password' in gitea_response.text.lower():
        print("❌ Gitea仍然显示登录表单，SSO未生效")
        return False
    else:
        print("✅ Gitea没有显示登录表单，SSO可能已生效")
        return True

if __name__ == "__main__":
    print("🚀 开始测试auth_request问题...")
    result = test_auth_request()
    if result:
        print("🎉 测试通过，SSO正常工作")
    else:
        print("❌ 测试失败，SSO仍有问题")
