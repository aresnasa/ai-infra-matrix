#!/usr/bin/env python3
"""
完整的SSO登录和Gitea访问测试
"""

import requests
import json
import re
from datetime import datetime

def log(message):
    print(f"[{datetime.now().strftime('%H:%M:%S')}] {message}")

def test_complete_sso_flow():
    log("🚀 测试完整SSO流程")
    
    # 创建session
    session = requests.Session()
    
    # 1. 通过API登录
    log("📋 步骤1: 通过API登录")
    try:
        login_data = {
            "username": "admin",
            "password": "admin123"
        }
        
        login_response = session.post(
            "http://localhost:8080/api/auth/login",
            json=login_data,
            headers={"Content-Type": "application/json"}
        )
        
        log(f"   登录API状态码: {login_response.status_code}")
        
        if login_response.status_code == 200:
            login_result = login_response.json()
            log(f"   ✅ 登录成功")
            log(f"   Token: {login_result.get('token', 'N/A')[:20]}...")
            
            # 设置Authorization header
            token = login_result.get('token')
            if token:
                session.headers.update({
                    'Authorization': f'Bearer {token}'
                })
                
        else:
            log(f"   ❌ 登录失败: {login_response.text}")
            return False
            
    except Exception as e:
        log(f"   ❌ 登录异常: {e}")
        return False
    
    # 2. 验证登录状态
    log("📋 步骤2: 验证登录状态")
    try:
        verify_response = session.get("http://localhost:8080/api/auth/verify")
        log(f"   验证状态码: {verify_response.status_code}")
        
        if verify_response.status_code == 200:
            log("   ✅ 登录状态有效")
        else:
            log("   ⚠️ 登录状态可能无效")
            
    except Exception as e:
        log(f"   ❌ 验证异常: {e}")
    
    # 3. 测试Gitea访问
    log("📋 步骤3: 测试Gitea访问（核心测试）")
    try:
        gitea_url = "http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin"
        gitea_response = session.get(gitea_url, allow_redirects=True)
        
        log(f"   Gitea访问状态码: {gitea_response.status_code}")
        log(f"   最终URL: {gitea_response.url}")
        
        # 检查是否有密码输入框
        has_password_field = bool(re.search(r'<input[^>]*type=[\"\']password[\"\']', gitea_response.text, re.IGNORECASE))
        has_form = bool(re.search(r'<form[^>]*action[^>]*login', gitea_response.text, re.IGNORECASE))
        
        if "/gitea/admin" in gitea_response.url:
            log("   🎉 完美！直接跳转到管理页面，无需二次密码！")
            return True
        elif has_password_field or has_form:
            log("   ❌ 问题：仍然显示登录表单")
            log("   🔍 分析原因...")
            
            # 检查请求头
            log("   📊 当前请求头:")
            for key, value in session.headers.items():
                if any(keyword in key.lower() for keyword in ['auth', 'user', 'token']):
                    log(f"      {key}: {value[:50]}...")
            
            # 检查响应头
            log("   📊 响应头:")
            for key, value in gitea_response.headers.items():
                if any(keyword in key.lower() for keyword in ['auth', 'user', 'x-webauth']):
                    log(f"      {key}: {value}")
                    
            return False
        else:
            log("   ⚠️ 其他情况，需要进一步分析")
            log(f"   页面内容长度: {len(gitea_response.text)}")
            return None
            
    except Exception as e:
        log(f"   ❌ Gitea访问异常: {e}")
        return False

def test_no_auth_scenario():
    """测试无认证场景（对照组）"""
    log("📋 对照测试: 无认证访问Gitea")
    
    no_auth_session = requests.Session()
    try:
        response = no_auth_session.get("http://localhost:8080/gitea/user/login?redirect_to=%2Fgitea%2Fadmin")
        
        has_password_field = bool(re.search(r'<input[^>]*type=[\"\']password[\"\']', response.text, re.IGNORECASE))
        
        if has_password_field:
            log("   ✅ 正确：无认证时显示登录表单")
            return True
        else:
            log("   ⚠️ 无认证时没有显示登录表单")
            return False
            
    except Exception as e:
        log(f"   ❌ 无认证测试失败: {e}")
        return False

def main():
    log("=" * 60)
    log("🎯 测试原始问题：已登录但访问Gitea还需要二次密码")
    log("=" * 60)
    
    # 执行完整测试
    sso_result = test_complete_sso_flow()
    no_auth_result = test_no_auth_scenario()
    
    # 总结
    log("=" * 60)
    log("📊 测试结果总结:")
    
    if sso_result is True:
        log("✅ 原始问题已修复！")
        log("   用户登录后访问Gitea无需二次密码输入")
    elif sso_result is False:
        log("❌ 原始问题仍然存在！")
        log("   用户登录后访问Gitea仍需要二次密码输入")
        log("   需要检查Nginx配置或Gitea设置")
    else:
        log("⚠️ 测试结果不明确，需要手动验证")
    
    if no_auth_result:
        log("✅ 安全检查通过：无认证时正确显示登录表单")
    else:
        log("⚠️ 安全检查需要关注")
        
    log("=" * 60)

if __name__ == "__main__":
    main()
