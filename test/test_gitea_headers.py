#!/usr/bin/env python3
"""
专门测试Gitea收到的头部
"""

import requests
import json
from datetime import datetime

def log(message):
    print(f"[{datetime.now().strftime('%H:%M:%S')}] {message}")

def main():
    log("🔍 测试Gitea实际收到的认证头部")
    
    # 1. 获取token
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
    log(f"✅ 获取token: {token[:20]}...")
    
    # 2. 设置Bearer token
    session.headers.update({'Authorization': f'Bearer {token}'})
    
    # 3. 访问Gitea登录页面，但不跟随重定向，这样我们可以看到实际的响应
    gitea_url = "http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin"
    
    log("📋 测试Gitea登录页面响应...")
    response = session.get(gitea_url, allow_redirects=False)
    
    log(f"状态码: {response.status_code}")
    log(f"响应头:")
    for key, value in response.headers.items():
        log(f"  {key}: {value}")
        
    if response.status_code == 302:
        log(f"重定向到: {response.headers.get('Location', 'N/A')}")
        if "/gitea/admin" in response.headers.get('Location', ''):
            log("🎉 成功！直接重定向到管理页面！")
            return True
    elif response.status_code == 200:
        # 检查是否是登录页面
        if "password" in response.text.lower():
            log("❌ 返回登录页面，SSO未生效")
        else:
            log("ℹ️ 返回200但内容不明确")
            log(f"内容长度: {len(response.text)}")
            
    return False

if __name__ == "__main__":
    main()
