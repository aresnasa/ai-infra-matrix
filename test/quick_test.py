#!/usr/bin/env python3
"""
简化的测试运行器 - 快速验证SSO功能
"""

import requests
import sys
from datetime import datetime

def log(message):
    print(f"[{datetime.now().strftime('%H:%M:%S')}] {message}")

def quick_sso_test():
    """快速SSO功能验证"""
    log("🚀 快速SSO功能验证")
    
    base_url = "http://localhost:8080"
    
    # 1. 获取新token
    log("1️⃣ 获取认证token...")
    try:
        login_data = {"username": "admin", "password": "admin123"}
        response = requests.post(f"{base_url}/api/auth/login", json=login_data)
        
        if response.status_code != 200:
            log(f"❌ 登录失败: {response.status_code}")
            return False
            
        token = response.json().get("token")
        if not token:
            log("❌ 响应中没有token")
            return False
            
        log(f"✅ 获取token成功: {token[:20]}...")
        
    except Exception as e:
        log(f"❌ 登录异常: {e}")
        return False
    
    # 2. 测试SSO重定向
    log("2️⃣ 测试SSO重定向...")
    try:
        response = requests.get(
            f"{base_url}/gitea/user/login?redirect_to=%2Fgitea%2Fadmin",
            headers={"Cookie": f"ai_infra_token={token}"},
            allow_redirects=False
        )
        
        log(f"响应状态: {response.status_code}")
        
        if response.status_code in [302, 303]:
            location = response.headers.get('Location', '')
            log(f"✅ SSO重定向成功: {location}")
            if '/gitea/' in location:
                log("✅ 原始问题已解决：无需二次密码输入")
                return True
            else:
                log(f"⚠️ 重定向位置异常: {location}")
                return False
        elif response.status_code == 200:
            if 'password' in response.text.lower():
                log("❌ 仍然显示密码输入框")
                return False
            else:
                log("✅ 直接访问成功")
                return True
        else:
            log(f"❌ 意外状态码: {response.status_code}")
            return False
            
    except Exception as e:
        log(f"❌ SSO测试异常: {e}")
        return False
    
    # 3. 测试无token访问
    log("3️⃣ 测试无token访问...")
    try:
        response = requests.get(f"{base_url}/gitea/user/login")
        
        if response.status_code == 200 and 'password' in response.text.lower():
            log("✅ 无token正确显示登录表单")
            return True
        else:
            log("⚠️ 无token访问行为异常")
            return False
            
    except Exception as e:
        log(f"❌ 无token测试异常: {e}")
        return False

def main():
    """主函数"""
    log("🎯 AI Infra Matrix 快速SSO测试")
    
    if quick_sso_test():
        log("🎉 测试通过！SSO功能正常工作")
        sys.exit(0)
    else:
        log("❌ 测试失败！请检查配置")
        sys.exit(1)

if __name__ == "__main__":
    main()
