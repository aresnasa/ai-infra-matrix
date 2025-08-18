#!/usr/bin/env python3
"""最终验证：确保SSO修复完全解决了原始问题"""

import requests

def test_valid_token_scenario():
    """测试有效token场景"""
    print("🔍 测试有效token自动登录...")
    
    # 获取有效token
    login_data = {'username': 'admin', 'password': 'adminpass123'}
    response = requests.post('http://localhost:8080/api/v1/auth/login', json=login_data)
    
    if response.status_code != 200:
        print("❌ 无法获取有效token")
        return False
    
    token = response.json()['token']
    print(f"Token获取成功: {token[:30]}...")
    
    # 测试登录端点自动重定向
    session = requests.Session()
    response = session.get(
        'http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin',
        headers={'Authorization': f'Bearer {token}'},
        allow_redirects=True
    )
    
    print(f"响应状态: {response.status_code}")
    print(f"最终URL: {response.url}")
    
    # 检查是否成功重定向到管理页面
    if '/gitea/admin' in response.url and response.status_code == 200:
        print("✅ 成功: 有效token用户自动重定向到目标页面，无需手动输入密码")
        return True
    else:
        print("❌ 失败: 有效token用户仍需手动操作")
        return False

def test_invalid_token_scenarios():
    """测试无效token场景"""
    print("\n🔍 测试无效token显示登录表单...")
    
    scenarios = [
        ("过期token", "Bearer expired_token_123"),
        ("无效token", "Bearer invalid_token_456"),
        ("无token", None)
    ]
    
    all_passed = True
    
    for name, auth_header in scenarios:
        print(f"\n  测试: {name}")
        
        headers = {}
        if auth_header:
            headers['Authorization'] = auth_header
            
        response = requests.get(
            'http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin',
            headers=headers
        )
        
        has_password_field = 'type="password"' in response.text
        has_login_form = '<form' in response.text and 'action="/gitea/user/login"' in response.text
        
        if has_password_field and has_login_form:
            print(f"  ✅ {name}: 正确显示登录表单")
        else:
            print(f"  ❌ {name}: 未正确显示登录表单")
            all_passed = False
    
    return all_passed

def main():
    print("🚀 最终验证：Gitea SSO登录问题修复")
    print("原问题：已经登录了localhost:8080，但是访问Gitea登录页面还需要二次输入密码")
    print("="*80)
    
    # 测试有效token自动登录
    valid_token_works = test_valid_token_scenario()
    
    # 测试无效token显示表单
    invalid_token_works = test_invalid_token_scenarios()
    
    print("\n" + "="*80)
    print("📋 最终结果:")
    
    if valid_token_works and invalid_token_works:
        print("🎉 修复成功！")
        print("✅ 已登录用户访问Gitea登录页面时自动重定向，无需手动输入密码")
        print("✅ 未登录用户正确看到登录表单")
        print("\n🎯 原始问题已完全解决：")
        print("   用户在主站点已登录后，访问 http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin")
        print("   将自动重定向到目标管理页面，不再需要二次密码输入")
        return True
    else:
        print("❌ 修复不完整，仍存在问题")
        return False

if __name__ == "__main__":
    success = main()
    exit(0 if success else 1)
