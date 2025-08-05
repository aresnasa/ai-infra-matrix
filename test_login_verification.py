#!/usr/bin/env python3
"""
简化的JupyterHub验证测试
专注于确认登录状态而不是notebook启动
"""
import requests
import re

def test_simplified_login():
    """简化的登录测试，重点关注登录状态而非notebook启动"""
    
    print("🔍 简化登录状态验证...")
    print("=" * 50)
    
    session = requests.Session()
    base_url = "http://localhost:8080"
    
    # 测试1: 标准用户名密码登录
    print("1️⃣ 测试标准用户名密码登录...")
    
    # 获取登录页面
    login_page = session.get(f"{base_url}/jupyter/", allow_redirects=True)
    print(f"登录页面状态: {login_page.status_code}")
    
    # 提取CSRF token
    csrf_match = re.search(r'name="(_xsrf|csrf_token)"[^>]*value="([^"]*)"', login_page.text)
    csrf_token = csrf_match.group(2) if csrf_match else None
    
    # 执行登录
    login_data = {
        "username": "admin",
        "password": "admin123"
    }
    if csrf_token:
        login_data["_xsrf"] = csrf_token
    
    login_response = session.post(f"{base_url}/hub/login", data=login_data, allow_redirects=False)
    print(f"登录响应状态: {login_response.status_code}")
    
    if login_response.status_code == 302:
        location = login_response.headers.get('Location', '')
        print(f"重定向到: {location}")
        
        if "/hub/spawn" in location or "/user/" in location:
            print("✅ 标准登录成功！")
            
            # 测试登录后的状态
            hub_home = session.get(f"{base_url}/hub/home", allow_redirects=True)
            if hub_home.status_code == 200:
                print("✅ 登录状态确认：可以访问Hub主页")
                
                # 检查是否有logout链接
                if "logout" in hub_home.text.lower():
                    print("✅ 检测到logout链接，确认已登录")
                    return True
            
    print("❌ 标准登录失败")
    return False

def test_token_login_detailed():
    """详细测试token登录"""
    print("\n2️⃣ 详细测试Token登录...")
    
    session = requests.Session()
    base_url = "http://localhost:8080"
    
    # 使用提供的token
    token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOm51bGwsInBlcm1pc3Npb25zIjpudWxsLCJleHAiOjE3NTQ0NDk4NTIsImlhdCI6MTc1NDM2MzQ1Mn0.9LbWjp93eL0lOC-hmHy5l8XTrHcDRjqxYllH0VeD93I"
    username = "admin"
    
    # 尝试token登录
    token_url = f"{base_url}/jupyter/hub/login?token={token}&username={username}"
    token_response = session.get(token_url, allow_redirects=False)
    
    print(f"Token登录状态: {token_response.status_code}")
    
    if token_response.status_code == 302:
        location = token_response.headers.get('Location', '')
        print(f"Token登录重定向到: {location}")
        
        if "/hub/spawn" in location or "/user/" in location:
            print("✅ Token登录成功！")
            return True
    elif token_response.status_code == 200:
        # 检查页面内容
        if "error" in token_response.text.lower():
            print("❌ Token登录页面显示错误")
            # 提取错误信息
            error_match = re.search(r'<div[^>]*class="[^"]*(?:error|alert)[^"]*"[^>]*>([^<]+)', token_response.text)
            if error_match:
                print(f"错误信息: {error_match.group(1).strip()}")
        else:
            print("⚠️ Token登录返回200但没有重定向")
    
    print("❌ Token登录未成功")
    return False

def verify_hub_api_token():
    """验证是否是Hub API token问题"""
    print("\n3️⃣ 分析Token格式问题...")
    
    token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOm51bGwsInBlcm1pc3Npb25zIjpudWxsLCJleHAiOjE3NTQ0NDk4NTIsImlhdCI6MTc1NDM2MzQ1Mn0.9LbWjp93eL0lOC-hmHy5l8XTrHcDRjqxYllH0VeD93I"
    
    # 解码JWT token查看内容
    try:
        import base64
        import json
        
        # 分割token
        parts = token.split('.')
        if len(parts) == 3:
            # 解码payload (添加必要的padding)
            payload = parts[1]
            payload += '=' * (4 - len(payload) % 4)  # 添加padding
            decoded = base64.urlsafe_b64decode(payload)
            token_data = json.loads(decoded)
            
            print("Token内容:")
            print(f"  用户ID: {token_data.get('user_id')}")
            print(f"  用户名: {token_data.get('username')}")
            print(f"  过期时间: {token_data.get('exp')}")
            print(f"  签发时间: {token_data.get('iat')}")
            
            # 检查是否过期
            import time
            current_time = time.time()
            exp_time = token_data.get('exp', 0)
            
            if current_time > exp_time:
                print("❌ Token已过期！")
                return False
            else:
                print("✅ Token未过期")
                print("💡 这是后端API的JWT token，JupyterHub可能不认识这种格式")
                print("💡 JupyterHub通常使用自己的token系统，而不是外部JWT")
                return True
                
    except Exception as e:
        print(f"❌ Token解析失败: {e}")
        return False

if __name__ == "__main__":
    import time
    print("🚀 JupyterHub登录状态详细验证")
    print("时间:", time.strftime("%Y-%m-%d %H:%M:%S"))
    
    # 测试标准登录
    standard_ok = test_simplified_login()
    
    # 测试token登录
    token_ok = test_token_login_detailed()
    
    # 分析token
    token_valid = verify_hub_api_token()
    
    print("\n" + "=" * 50)
    print("📊 验证结果总结")
    print("=" * 50)
    print(f"标准登录 (用户名/密码): {'✅ 成功' if standard_ok else '❌ 失败'}")
    print(f"Token登录: {'✅ 成功' if token_ok else '❌ 失败'}")
    print(f"Token格式: {'✅ 有效' if token_valid else '❌ 无效/过期'}")
    
    if standard_ok and not token_ok:
        print("\n💡 结论：")
        print("- 标准用户名密码登录工作正常")
        print("- 提供的token是后端API的JWT token，JupyterHub不认识")
        print("- JupyterHub使用内部token系统，与后端API的JWT token不同")
        
        print("\n🎯 建议：")
        print("- 使用标准登录方式：http://localhost:8080/jupyter/")
        print("- 使用凭据：admin / admin123")
        print("- 或者在JupyterHub中获取正确的Hub API token")
    elif not standard_ok:
        print("\n⚠️ 问题：标准登录也失败，需要检查系统配置")
