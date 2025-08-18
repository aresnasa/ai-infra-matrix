#!/usr/bin/env python3
"""
检查cookie状态和SSO流程
"""

import requests
from datetime import datetime

def log(message):
    print(f"[{datetime.now().strftime('%H:%M:%S')}] {message}")

def main():
    log("🔍 检查cookie状态和请求过程")
    
    # 1. 获取token并创建session
    session = requests.Session()
    login_data = {"username": "admin", "password": "admin123"}
    
    login_response = session.post(
        "http://localhost:8080/api/auth/login",
        json=login_data,
        headers={"Content-Type": "application/json"}
    )
    
    if login_response.status_code != 200:
        log(f"❌ 登录失败: {login_response.status_code}")
        return
        
    token = login_response.json().get('token')
    session.headers.update({'Authorization': f'Bearer {token}'})
    
    log(f"✅ 登录成功，当前cookies: {dict(session.cookies)}")
    
    # 2. 首次访问gitea登录页面
    log("📋 首次访问Gitea登录页面...")
    gitea_url = "http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin"
    
    response = session.get(gitea_url, allow_redirects=False)
    log(f"状态码: {response.status_code}")
    log(f"响应头Location: {response.headers.get('Location', '无')}")
    log(f"新增cookies: {dict(session.cookies)}")
    
    # 检查是否有 i_like_gitea cookie
    has_gitea_cookie = 'i_like_gitea' in session.cookies
    log(f"是否有i_like_gitea cookie: {has_gitea_cookie}")
    
    if response.status_code == 302:
        log(f"🔄 重定向到: {response.headers.get('Location')}")
    
    # 3. 清除cookies重新测试
    log("🧹 清除cookies重新测试...")
    session.cookies.clear()
    
    response2 = session.get(gitea_url, allow_redirects=False)
    log(f"清除cookies后状态码: {response2.status_code}")
    log(f"清除cookies后新增cookies: {dict(session.cookies)}")
    
    has_gitea_cookie2 = 'i_like_gitea' in session.cookies
    log(f"清除后是否有i_like_gitea cookie: {has_gitea_cookie2}")

if __name__ == "__main__":
    main()
