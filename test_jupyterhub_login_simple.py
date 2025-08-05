#!/usr/bin/env python3
"""
JupyterHub登录测试脚本
"""
import requests
import re
from urllib.parse import urljoin, urlparse, parse_qs

def test_jupyterhub_login():
    """测试JupyterHub登录流程"""
    session = requests.Session()
    base_url = "http://localhost:8080"
    
    print("🔄 开始JupyterHub登录测试...")
    
    try:
        # 1. 访问JupyterHub首页
        print("1️⃣ 访问JupyterHub首页...")
        response = session.get(f"{base_url}/jupyter/", allow_redirects=True)
        print(f"   状态码: {response.status_code}")
        print(f"   最终URL: {response.url}")
        
        if response.status_code != 200:
            print(f"❌ 无法访问JupyterHub")
            return False
        
        # 2. 检查是否在登录页面
        if "/hub/login" in response.url:
            print("2️⃣ 重定向到登录页面 ✅")
        else:
            print("2️⃣ 未重定向到登录页面 ⚠️")
            print(f"   当前页面内容片段: {response.text[:200]}")
        
        # 3. 提取CSRF token
        csrf_token = None
        csrf_match = re.search(r'name="(_xsrf|csrf_token)"[^>]*value="([^"]*)"', response.text)
        if csrf_match:
            csrf_token = csrf_match.group(2)
            print(f"3️⃣ 提取CSRF token: {csrf_token[:20]}... ✅")
        else:
            print("3️⃣ 未找到CSRF token ⚠️")
        
        # 4. 准备登录数据
        login_data = {
            "username": "admin",
            "password": "admin123"
        }
        if csrf_token:
            login_data["_xsrf"] = csrf_token
        
        # 5. 执行登录
        print("4️⃣ 执行登录...")
        login_url = f"{base_url}/hub/login"
        headers = {
            'Content-Type': 'application/x-www-form-urlencoded',
            'Referer': response.url
        }
        
        login_response = session.post(login_url, data=login_data, 
                                    headers=headers, allow_redirects=True)
        
        print(f"   登录响应状态码: {login_response.status_code}")
        print(f"   最终URL: {login_response.url}")
        
        # 6. 检查登录结果
        if login_response.status_code == 200:
            if "/hub/spawn" in login_response.url:
                print("5️⃣ 登录成功！重定向到spawn页面 ✅")
                return True
            elif "/user/" in login_response.url:
                print("5️⃣ 登录成功！重定向到用户页面 ✅")
                return True
            elif "logout" in login_response.text.lower():
                print("5️⃣ 登录成功！检测到logout链接 ✅")
                return True
            elif "error" in login_response.text.lower() or "invalid" in login_response.text.lower():
                print("5️⃣ 登录失败！检测到错误信息 ❌")
                # 提取错误信息
                error_match = re.search(r'<div[^>]*class="[^"]*error[^"]*"[^>]*>([^<]+)', login_response.text)
                if error_match:
                    print(f"   错误信息: {error_match.group(1).strip()}")
                return False
            else:
                print("5️⃣ 登录状态不明确 ⚠️")
                print(f"   页面内容片段: {login_response.text[:500]}")
                return False
        else:
            print(f"❌ 登录失败，状态码: {login_response.status_code}")
            return False
            
    except requests.RequestException as e:
        print(f"❌ 请求异常: {e}")
        return False

if __name__ == "__main__":
    success = test_jupyterhub_login()
    print(f"\n📊 最终结果: {'✅ 登录成功' if success else '❌ 登录失败'}")
