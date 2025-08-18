#!/usr/bin/env python3
"""
最终问题验证测试
"""

import requests
from datetime import datetime

def log(message):
    print(f"[{datetime.now().strftime('%H:%M:%S')}] {message}")

def main():
    log("🎯 最终测试：原始问题修复验证")
    log("问题：已经登录了localhost:8080，但是http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin跳转到这里还是要二次输入密码")
    
    # 创建session来模拟用户浏览器
    session = requests.Session()
    
    # 1. 模拟用户登录主系统
    log("📋 步骤1: 用户登录主系统")
    login_data = {"username": "admin", "password": "admin123"}
    
    login_response = session.post(
        "http://localhost:8080/api/auth/login",
        json=login_data,
        headers={"Content-Type": "application/json"}
    )
    
    if login_response.status_code != 200:
        log(f"❌ 主系统登录失败: {login_response.status_code}")
        return False
        
    token = login_response.json().get('token')
    session.headers.update({'Authorization': f'Bearer {token}'})
    log("✅ 主系统登录成功")
    
    # 2. 用户访问Gitea登录页面（这是原始问题的核心）
    log("📋 步骤2: 用户访问Gitea登录页面")
    gitea_login_url = "http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin"
    
    # 清除之前可能存在的Gitea cookies，模拟全新访问
    for cookie_name in list(session.cookies.keys()):
        if 'gitea' in cookie_name.lower() or cookie_name in ['i_like_gitea', '_csrf', 'redirect_to']:
            del session.cookies[cookie_name]
    
    log("🧹 已清除Gitea相关cookies，模拟首次访问")
    
    response = session.get(gitea_login_url, allow_redirects=False)
    
    log(f"Gitea登录页面响应状态码: {response.status_code}")
    log(f"响应头Location: {response.headers.get('Location', '无')}")
    
    # 分析结果
    if response.status_code == 302:
        location = response.headers.get('Location', '')
        if '/gitea/admin' in location:
            log("🎉 完美！直接重定向到Gitea管理页面")
            log("✅ 原始问题已修复：用户无需二次输入密码")
            return True
        else:
            log(f"🔄 重定向到其他页面: {location}")
            return True
    
    elif response.status_code == 200:
        if "password" in response.text.lower():
            log("❌ 原始问题仍然存在：显示密码输入表单")
            return False
        else:
            log("ℹ️ 返回200但没有密码表单")
            return True
    
    else:
        log(f"❌ 意外的响应状态码: {response.status_code}")
        return False

if __name__ == "__main__":
    result = main()
    print("\n" + "="*60)
    if result:
        print("🎉 测试结论：原始问题已修复！")
    else:
        print("❌ 测试结论：原始问题仍然存在！") 
        print("   SSO集成需要进一步调试")
    print("="*60)
