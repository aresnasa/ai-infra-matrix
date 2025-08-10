#!/usr/bin/env python3
"""
测试JupyterHub在iframe中的实际用户体验
模拟用户登录和API访问过程
"""

import requests
import time
import re
from urllib.parse import urljoin, urlparse, parse_qs

def simulate_user_login():
    """模拟用户登录过程并测试API访问"""
    
    base_url = "http://localhost:8080"
    session = requests.Session()
    session.headers.update({
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36'
    })
    
    print("🔍 模拟用户在iframe中使用JupyterHub...")
    print("=" * 60)
    
    # 第一步：访问wrapper页面
    print("\n📄 第1步：访问 /jupyterhub wrapper页面")
    try:
        wrapper_response = session.get(f"{base_url}/jupyterhub")
        if wrapper_response.status_code == 200:
            print(f"   ✅ wrapper页面正常 (200)")
            if "src=\"/jupyter/hub/\"" in wrapper_response.text:
                print(f"   ✅ iframe指向正确的路径")
            else:
                print(f"   ⚠️  iframe路径可能有问题")
        else:
            print(f"   ❌ wrapper页面错误 ({wrapper_response.status_code})")
            return
    except Exception as e:
        print(f"   💥 wrapper页面访问失败: {str(e)}")
        return
    
    # 第二步：访问JupyterHub主页（iframe内容）
    print("\n🏠 第2步：访问 /jupyter/hub/ (iframe内容)")
    try:
        hub_response = session.get(f"{base_url}/jupyter/hub/")
        print(f"   📊 状态码: {hub_response.status_code}")
        if hub_response.status_code == 302:
            location = hub_response.headers.get('Location')
            print(f"   🔄 重定向到: {location}")
            
            # 跟随重定向
            if location:
                if location.startswith('/'):
                    redirect_url = f"{base_url}{location}"
                else:
                    redirect_url = location
                
                redirect_response = session.get(redirect_url)
                print(f"   📊 重定向页面状态: {redirect_response.status_code}")
                
                if "login" in location:
                    print(f"   🔑 需要登录，这是正常的")
                elif "user" in location:
                    print(f"   👤 已重定向到用户页面")
                    
    except Exception as e:
        print(f"   💥 hub页面访问失败: {str(e)}")
    
    # 第三步：尝试访问API端点（模拟iframe中的JavaScript请求）
    print("\n⚡ 第3步：测试iframe中的API访问模式")
    
    # 模拟从iframe发出的请求
    iframe_headers = {
        'Referer': f'{base_url}/jupyterhub',  # 重要：从wrapper页面发出
        'Origin': base_url,
        'X-Requested-With': 'XMLHttpRequest',
        'Accept': 'application/json',
    }
    
    api_endpoints = [
        "/jupyter/user/admin/api/sessions",
        "/jupyter/user/admin/api/kernels", 
        "/jupyter/user/admin/api/terminals",
    ]
    
    for endpoint in api_endpoints:
        print(f"\n   🔌 测试API: {endpoint}")
        try:
            # 第一次请求（模拟iframe初始加载）
            api_response = session.get(f"{base_url}{endpoint}", headers=iframe_headers, timeout=5)
            print(f"      📊 初始请求: {api_response.status_code}")
            
            if api_response.status_code == 404:
                print(f"      ❌ 404错误 - 这就是需要修复的问题！")
            elif api_response.status_code == 403:
                print(f"      🔒 403认证错误 - 路由正常，需要登录")
            elif api_response.status_code == 200:
                print(f"      ✅ 200成功 - API正常工作")
            else:
                print(f"      ⚠️  其他状态: {api_response.status_code}")
                
            # 等待一下再次请求（模拟页面刷新后的情况）
            time.sleep(1)
            retry_response = session.get(f"{base_url}{endpoint}", headers=iframe_headers, timeout=5)
            print(f"      📊 重试请求: {retry_response.status_code}")
            
            if api_response.status_code != retry_response.status_code:
                print(f"      🔄 状态码发生变化 - 可能存在时序问题")
            else:
                print(f"      ✅ 状态码一致 - 行为稳定")
                
        except requests.exceptions.Timeout:
            print(f"      ⏰ 请求超时")
        except Exception as e:
            print(f"      💥 请求失败: {str(e)}")
    
    print("\n" + "=" * 60)
    print("🏁 iframe用户体验测试完成!")
    
    # 总结
    print("\n📋 问题诊断总结:")
    print("   - 如果API初始请求返回404，需要修复nginx路由配置")
    print("   - 如果API初始请求返回403，说明路由正常，只是需要身份验证")
    print("   - 如果状态码发生变化，说明存在时序或缓存问题")
    print("   - 如果状态码一致，说明行为稳定，问题已解决")

if __name__ == "__main__":
    simulate_user_login()
