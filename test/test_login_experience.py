#!/usr/bin/env python3
"""
测试用户访问 http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin 的完整体验
"""

import requests
import time

def test_login_experience():
    print("🧪 测试用户登录页面体验")
    print("=" * 50)
    
    # 步骤1: 用户登录后端获取SSO token
    print("1. 用户先登录主后端系统...")
    login_response = requests.post(
        "http://localhost:8080/api/auth/login",
        json={"username": "admin", "password": "admin123"},
        headers={"Content-Type": "application/json"}
    )
    
    if login_response.status_code != 200:
        print(f"❌ 后端登录失败: {login_response.status_code}")
        return False
        
    token = login_response.json().get("token")
    print(f"   ✅ 获得SSO token: {token[:20]}...")
    
    # 步骤2: 用户通过浏览器访问Gitea登录页面
    print("\n2. 用户访问 Gitea 登录页面...")
    print("   URL: http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin")
    
    session = requests.Session()
    session.cookies.set('ai_infra_token', token)
    
    response = session.get(
        "http://localhost:8080/gitea/user/login",
        params={"redirect_to": "/gitea/admin"},
        allow_redirects=True
    )
    
    print(f"   → 最终状态码: {response.status_code}")
    print(f"   → 最终URL: {response.url}")
    
    # 检查是否需要输入密码
    content = response.text
    has_password_field = 'type="password"' in content or 'password' in content.lower()
    has_login_form = '<form' in content and ('login' in content.lower() or 'sign in' in content.lower())
    
    if has_password_field or has_login_form:
        print("   ❌ 页面仍然显示登录表单或密码输入框")
        print("   这不符合预期 - 用户已经通过SSO认证，不应该再需要输入密码")
        return False
    else:
        print("   ✅ 页面没有显示登录表单或密码输入框")
    
    # 检查是否建立了Gitea会话
    gitea_cookie = session.cookies.get('i_like_gitea')
    if gitea_cookie:
        print(f"   ✅ Gitea会话已建立: {gitea_cookie[:10]}...")
    else:
        print("   ⚠️ 没有检测到Gitea会话cookie")
    
    # 步骤3: 验证用户可以正常访问管理页面
    print("\n3. 验证管理员权限...")
    admin_response = session.get("http://localhost:8080/gitea/admin")
    print(f"   → 管理页面状态: {admin_response.status_code}")
    
    if admin_response.status_code == 200:
        print("   ✅ 用户可以正常访问管理页面")
        return True
    else:
        print("   ❌ 用户无法访问管理页面")
        return False

if __name__ == "__main__":
    success = test_login_experience()
    
    print("\n" + "=" * 50)
    if success:
        print("🎉 测试通过！")
        print("用户体验符合预期：")
        print("• 已登录用户访问Gitea登录页面时不需要再次输入密码")
        print("• SSO认证自动建立Gitea会话")
        print("• 用户可以正常访问需要权限的页面")
    else:
        print("❌ 测试失败！")
        print("用户仍然需要手动输入密码，SSO统一认证尚未完全实现。")
