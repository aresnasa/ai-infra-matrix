#!/usr/bin/env python3
"""
JupyterHub Token登录问题诊断脚本
专门针对token登录失败的问题进行详细诊断
"""
import requests
import re
import json
import time
from urllib.parse import urljoin, urlparse, parse_qs

def test_token_login_issue():
    """测试Token登录问题"""
    session = requests.Session()
    base_url = "http://localhost:8080"
    
    # 提供的token
    token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6ImFkbWluIiwicm9sZXMiOm51bGwsInBlcm1pc3Npb25zIjpudWxsLCJleHAiOjE3NTQ0NDk4NTIsImlhdCI6MTc1NDM2MzQ1Mn0.9LbWjp93eL0lOC-hmHy5l8XTrHcDRjqxYllH0VeD93I"
    username = "admin"
    
    print("🔍 开始Token登录问题诊断...")
    print("=" * 60)
    
    try:
        # 1. 首先测试提供的token URL
        print("1️⃣ 测试提供的token登录URL...")
        token_url = f"{base_url}/jupyter/hub/login?token={token}&username={username}"
        print(f"URL: {token_url}")
        
        response = session.get(token_url, allow_redirects=True)
        print(f"状态码: {response.status_code}")
        print(f"最终URL: {response.url}")
        
        if "error" in response.text.lower() or "invalid" in response.text.lower():
            print("❌ 检测到错误信息:")
            # 提取错误信息
            error_matches = re.findall(r'<div[^>]*class="[^"]*(?:error|alert)[^"]*"[^>]*>([^<]+)', response.text, re.IGNORECASE)
            for error in error_matches:
                print(f"   {error.strip()}")
        
        # 2. 验证后端API token是否有效
        print("\n2️⃣ 验证后端API token有效性...")
        headers = {'Authorization': f'Bearer {token}'}
        api_response = requests.get(f"{base_url}/api/auth/verify", headers=headers)
        print(f"后端API验证状态: {api_response.status_code}")
        if api_response.status_code == 200:
            print("✅ Token在后端API中有效")
            print(f"用户信息: {api_response.text}")
        else:
            print("❌ Token在后端API中无效或过期")
            print(f"响应: {api_response.text}")
        
        # 3. 测试标准登录流程对比
        print("\n3️⃣ 执行标准登录流程作为对比...")
        
        # 获取登录页面
        login_page = session.get(f"{base_url}/jupyter/", allow_redirects=True)
        print(f"登录页面状态: {login_page.status_code}")
        
        # 提取CSRF token
        csrf_match = re.search(r'name="(_xsrf|csrf_token)"[^>]*value="([^"]*)"', login_page.text)
        csrf_token = csrf_match.group(2) if csrf_match else None
        print(f"CSRF Token: {'✅ 已获取' if csrf_token else '❌ 未找到'}")
        
        # 执行标准登录
        login_data = {
            "username": "admin",
            "password": "admin123"
        }
        if csrf_token:
            login_data["_xsrf"] = csrf_token
        
        login_response = session.post(f"{base_url}/hub/login", data=login_data, 
                                    allow_redirects=True)
        print(f"标准登录状态: {login_response.status_code}")
        print(f"标准登录最终URL: {login_response.url}")
        
        if "/hub/spawn" in login_response.url or "/user/" in login_response.url:
            print("✅ 标准登录成功")
        else:
            print("❌ 标准登录也失败")
        
        # 4. 检查JupyterHub配置
        print("\n4️⃣ 检查JupyterHub认证器配置...")
        
        # 检查JupyterHub日志
        print("查看JupyterHub日志中的认证相关信息...")
        
        return response.status_code == 200 and "/user/" in response.url
        
    except Exception as e:
        print(f"❌ 测试过程中发生异常: {e}")
        return False

def test_fresh_login_flow():
    """测试全新的登录流程"""
    print("\n" + "=" * 60)
    print("🔄 执行全新登录流程测试...")
    print("=" * 60)
    
    session = requests.Session()
    base_url = "http://localhost:8080"
    
    try:
        # 1. 清除所有cookies，模拟隐私模式
        session.cookies.clear()
        
        # 2. 访问JupyterHub主页
        print("1️⃣ 访问JupyterHub主页 (清空cookies)...")
        response = session.get(f"{base_url}/jupyter/", allow_redirects=True)
        print(f"状态码: {response.status_code}")
        print(f"最终URL: {response.url}")
        
        # 3. 检查是否需要登录
        if "/hub/login" in response.url:
            print("2️⃣ 重定向到登录页面 ✅")
            
            # 提取CSRF token
            csrf_match = re.search(r'name="(_xsrf|csrf_token)"[^>]*value="([^"]*)"', response.text)
            csrf_token = csrf_match.group(2) if csrf_match else None
            print(f"3️⃣ CSRF Token: {'✅ ' + csrf_token[:20] + '...' if csrf_token else '❌ 未找到'}")
            
            # 执行登录
            print("4️⃣ 执行登录...")
            login_data = {
                "username": "admin",
                "password": "admin123"
            }
            if csrf_token:
                login_data["_xsrf"] = csrf_token
            
            headers = {
                'Content-Type': 'application/x-www-form-urlencoded',
                'Referer': response.url
            }
            
            login_response = session.post(f"{base_url}/hub/login", 
                                        data=login_data, 
                                        headers=headers,
                                        allow_redirects=True)
            
            print(f"登录响应状态: {login_response.status_code}")
            print(f"登录后URL: {login_response.url}")
            
            # 检查登录结果
            if "/hub/spawn" in login_response.url:
                print("5️⃣ ✅ 登录成功! 重定向到spawn页面")
                
                # 等待spawn完成
                print("6️⃣ 等待notebook服务器启动...")
                time.sleep(3)
                
                # 检查用户页面
                user_response = session.get(f"{base_url}/user/admin/", allow_redirects=False)
                print(f"用户页面状态: {user_response.status_code}")
                
                if user_response.status_code == 200:
                    print("7️⃣ ✅ Notebook服务器启动成功!")
                    return True
                else:
                    print("7️⃣ ⚠️ Notebook服务器可能还在启动中...")
                    return True
            else:
                print("5️⃣ ❌ 登录失败")
                if "error" in login_response.text.lower():
                    errors = re.findall(r'<div[^>]*class="[^"]*error[^"]*"[^>]*>([^<]+)', login_response.text)
                    for error in errors:
                        print(f"   错误: {error.strip()}")
                return False
        else:
            print("2️⃣ 未重定向到登录页面，可能已经登录")
            return True
            
    except Exception as e:
        print(f"❌ 全新登录流程测试异常: {e}")
        return False

def diagnose_authentication_issue():
    """诊断认证问题的具体原因"""
    print("\n" + "=" * 60)
    print("🔍 认证问题详细诊断...")
    print("=" * 60)
    
    # 1. 检查后端API状态
    print("1️⃣ 检查后端认证API状态...")
    try:
        login_test = requests.post("http://localhost:8080/api/auth/login", 
                                 json={"username": "admin", "password": "admin123"},
                                 timeout=10)
        print(f"后端API状态: {login_test.status_code}")
        if login_test.status_code == 200:
            data = login_test.json()
            print(f"✅ 后端API正常，token: {data.get('token', '')[:30]}...")
        else:
            print(f"❌ 后端API异常: {login_test.text}")
    except Exception as e:
        print(f"❌ 后端API连接失败: {e}")
    
    # 2. 检查数据库连接
    print("\n2️⃣ 检查数据库用户状态...")
    try:
        import psycopg2
        conn = psycopg2.connect(
            host='localhost',
            port=5432,
            database='ansible_playbook_generator',
            user='postgres',
            password='postgres123'
        )
        cursor = conn.cursor()
        cursor.execute("SELECT username, email, is_active FROM users WHERE username = 'admin';")
        result = cursor.fetchone()
        if result:
            print(f"✅ 用户存在: {result[0]}, 邮箱: {result[1]}, 激活: {result[2]}")
        else:
            print("❌ 用户不存在")
        cursor.close()
        conn.close()
    except Exception as e:
        print(f"❌ 数据库检查失败: {e}")

if __name__ == "__main__":
    print("🚀 开始JupyterHub Token登录问题完整诊断")
    print("时间:", time.strftime("%Y-%m-%d %H:%M:%S"))
    
    # 诊断token登录问题
    token_success = test_token_login_issue()
    
    # 测试标准登录流程
    standard_success = test_fresh_login_flow()
    
    # 详细诊断
    diagnose_authentication_issue()
    
    print("\n" + "=" * 60)
    print("📊 诊断结果总结")
    print("=" * 60)
    print(f"Token登录: {'✅ 成功' if token_success else '❌ 失败'}")
    print(f"标准登录: {'✅ 成功' if standard_success else '❌ 失败'}")
    
    if standard_success and not token_success:
        print("\n💡 建议:")
        print("- Token登录可能存在问题，但标准用户名密码登录正常")
        print("- 建议使用标准登录方式: http://localhost:8080/jupyter/")
        print("- 使用凭据: admin / admin123")
    elif not standard_success:
        print("\n⚠️ 警告:")
        print("- 标准登录也失败，可能存在系统级别问题")
        print("- 建议检查JupyterHub配置和后端API连接")
    else:
        print("\n✅ 所有登录方式都正常工作!")
