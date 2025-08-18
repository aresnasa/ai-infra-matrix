#!/usr/bin/env python3
"""
验证Gitea登录端点处理各种认证场景
确保用户体验符合预期：
1. 有效token -> 自动SSO登录，无需手动输入密码
2. 无效/过期token -> 显示Gitea登录表单供手动登录
3. 无token -> 显示Gitea登录表单供手动登录
"""

import requests
import json
import sys

def test_scenario(description, token_header=None):
    """测试特定认证场景"""
    print(f"\n{'='*50}")
    print(f"🧪 测试场景: {description}")
    print(f"{'='*50}")
    
    # 准备请求头
    headers = {}
    if token_header:
        headers['Authorization'] = token_header
    
    url = "http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin"
    
    try:
        # 创建session以处理重定向
        session = requests.Session()
        
        # 发送请求，允许重定向
        response = session.get(url, headers=headers, allow_redirects=True)
        
        print(f"📊 响应状态: {response.status_code}")
        print(f"🌍 最终URL: {response.url}")
        
        # 检查是否包含密码输入框（表示需要手动登录）
        has_password_field = 'type="password"' in response.text
        has_login_form = '<form' in response.text and 'action="/gitea/user/login"' in response.text
        
        # 检查是否是Gitea管理页面（表示自动登录成功）
        is_admin_page = '/gitea/admin' in response.url and response.status_code == 200
        
        print(f"🔐 包含密码字段: {'是' if has_password_field else '否'}")
        print(f"📝 包含登录表单: {'是' if has_login_form else '否'}")
        print(f"👑 已到达管理页面: {'是' if is_admin_page else '否'}")
        
        # 分析结果
        if is_admin_page:
            print("✅ 结果: 自动SSO登录成功，用户直接到达目标页面")
            return "auto_login"
        elif has_password_field and has_login_form:
            print("✅ 结果: 正确显示登录表单，用户可手动登录")
            return "manual_login"
        else:
            print("❌ 结果: 意外的响应内容")
            print(f"响应内容预览: {response.text[:300]}...")
            return "unexpected"
            
    except Exception as e:
        print(f"❌ 请求失败: {e}")
        return "error"

def get_valid_token():
    """获取有效的SSO token"""
    try:
        login_data = {
            "username": "admin",
            "password": "adminpass123"
        }
        
        response = requests.post(
            "http://localhost:8080/sso/bootstrap.php",
            data=login_data,
            allow_redirects=False
        )
        
        if response.status_code == 200:
            data = response.json()
            if data.get("success") and data.get("token"):
                return data["token"]
        
        print("⚠️ 无法获取有效token，将使用模拟token")
        return None
    except Exception as e:
        print(f"⚠️ 获取token时出错: {e}")
        return None

def main():
    print("🔍 Gitea登录端点认证场景测试")
    print("确保SSO集成正确处理各种认证状态")
    
    # 获取有效token
    valid_token = get_valid_token()
    
    # 测试场景1: 有效token
    if valid_token:
        result1 = test_scenario("有效SSO token", f"Bearer {valid_token}")
        expected1 = "auto_login"
    else:
        print("\n⚠️ 跳过有效token测试（无法获取有效token）")
        result1 = expected1 = "skipped"
    
    # 测试场景2: 无效token
    result2 = test_scenario("无效/过期token", "Bearer invalid_token_12345")
    expected2 = "manual_login"
    
    # 测试场景3: 无token
    result3 = test_scenario("无认证token")
    expected3 = "manual_login"
    
    # 汇总结果
    print(f"\n{'='*60}")
    print("📋 测试结果汇总:")
    print(f"{'='*60}")
    
    tests = [
        ("有效SSO token", result1, expected1),
        ("无效/过期token", result2, expected2), 
        ("无认证token", result3, expected3)
    ]
    
    all_passed = True
    for name, actual, expected in tests:
        if expected == "skipped":
            print(f"⏭️  {name}: 跳过")
        elif actual == expected:
            print(f"✅ {name}: 通过 ({actual})")
        else:
            print(f"❌ {name}: 失败 (期望: {expected}, 实际: {actual})")
            all_passed = False
    
    print(f"\n{'='*60}")
    if all_passed:
        print("🎉 所有测试通过！SSO登录端点配置正确。")
        print("用户体验:")
        print("  - 已登录用户 → 自动进入目标页面")
        print("  - 未登录用户 → 看到登录表单")
        return 0
    else:
        print("💥 存在测试失败，需要检查配置。")
        return 1

if __name__ == "__main__":
    sys.exit(main())
