#!/usr/bin/env python3
"""
深度诊断SSO问题
"""

import requests
import re
from datetime import datetime

def log(message):
    print(f"[{datetime.now().strftime('%H:%M:%S')}] {message}")

def main():
    log("🔍 深度诊断SSO配置问题")
    
    # 创建session模拟用户
    session = requests.Session()
    
    # 1. 尝试不同的访问方式
    log("📋 测试1: 直接访问主页面")
    try:
        main_response = session.get("http://localhost:8080")
        log(f"   主页面状态码: {main_response.status_code}")
        log(f"   主页面Cookies: {dict(main_response.cookies)}")
        
        # 检查是否有任何认证相关的内容
        if "login" in main_response.text.lower():
            log("   ⚠️ 主页面包含登录相关内容")
        if "admin" in main_response.text.lower():
            log("   ✅ 主页面包含管理员相关内容")
            
    except Exception as e:
        log(f"   ❌ 访问失败: {e}")
    
    # 2. 检查是否有任何自动登录机制
    log("📋 测试2: 检查SSO自动登录")
    try:
        # 访问可能的SSO端点
        sso_endpoints = [
            "/sso/",
            "/auth/",
            "/__auth/verify",
            "/api/auth/verify"
        ]
        
        for endpoint in sso_endpoints:
            try:
                resp = session.get(f"http://localhost:8080{endpoint}")
                log(f"   {endpoint}: {resp.status_code}")
                if resp.status_code == 200:
                    log(f"      内容长度: {len(resp.text)}")
            except:
                pass
                
    except Exception as e:
        log(f"   ❌ SSO检查失败: {e}")
    
    # 3. 模拟实际的用户登录流程
    log("📋 测试3: 模拟用户登录流程")
    try:
        # 尝试访问登录页面
        login_page = session.get("http://localhost:8080/login")
        log(f"   登录页面状态码: {login_page.status_code}")
        
        if login_page.status_code == 200:
            # 尝试提交登录
            login_data = {"username": "admin", "password": "admin"}
            login_result = session.post("http://localhost:8080/login", data=login_data)
            log(f"   登录提交状态码: {login_result.status_code}")
            log(f"   登录后Cookies: {dict(session.cookies)}")
            
            # 再次检查主页面
            after_login = session.get("http://localhost:8080")
            log(f"   登录后主页面状态码: {after_login.status_code}")
            
            if "admin" in after_login.text.lower():
                log("   ✅ 登录成功，主页面显示管理员内容")
                
                # 现在测试Gitea访问
                log("📋 测试4: 登录后访问Gitea")
                gitea_url = "http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin"
                gitea_response = session.get(gitea_url, allow_redirects=True)
                
                log(f"   Gitea访问状态码: {gitea_response.status_code}")
                log(f"   最终URL: {gitea_response.url}")
                
                # 检查是否有密码字段
                has_password = bool(re.search(r'<input[^>]*type=[\"\']password[\"\']', gitea_response.text, re.IGNORECASE))
                
                if has_password:
                    log("   ❌ 仍然显示密码输入框")
                    
                    # 分析Headers
                    log("   📊 分析请求Headers:")
                    for key, value in session.headers.items():
                        if 'auth' in key.lower() or 'user' in key.lower():
                            log(f"      {key}: {value}")
                            
                    # 检查响应Headers
                    log("   📊 分析响应Headers:")
                    for key, value in gitea_response.headers.items():
                        if 'auth' in key.lower() or 'user' in key.lower() or 'x-' in key.lower():
                            log(f"      {key}: {value}")
                
                else:
                    log("   ✅ 成功：没有显示密码输入框")
                    
            else:
                log("   ❌ 登录失败")
        
    except Exception as e:
        log(f"   ❌ 登录流程失败: {e}")
    
    log("🏁 诊断完成")

if __name__ == "__main__":
    main()
