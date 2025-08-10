#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试nginx重定向是否正确保持端口号
"""

import requests
import sys

def test_redirect(url, description):
    """测试单个URL的重定向"""
    print(f"\n🔍 测试 {description}")
    print(f"   URL: {url}")
    
    try:
        resp = requests.get(url, allow_redirects=False, timeout=5)
        print(f"   状态码: {resp.status_code}")
        
        if resp.status_code in [301, 302]:
            location = resp.headers.get('Location', '')
            print(f"   重定向到: {location}")
            
            # 检查端口号是否保持
            if ':8080' in location or location.startswith('/'):
                print("   ✅ 端口保持正确")
                return True
            else:
                print("   ❌ 端口丢失!")
                return False
        elif resp.status_code == 200:
            print("   ✅ 直接访问成功")
            return True
        else:
            print(f"   ⚠️  未预期的状态码: {resp.status_code}")
            return False
            
    except Exception as e:
        print(f"   ❌ 请求异常: {e}")
        return False

def main():
    """主测试函数"""
    print("🔄 AI基础设施矩阵 - 重定向测试")
    print("=" * 50)
    
    test_cases = [
        ("http://localhost:8080/jupyter", "JupyterHub核心路径"),
        ("http://localhost:8080/jupyter/", "JupyterHub根路径"),  
        ("http://localhost:8080/sso", "SSO登录路径"),
        ("http://localhost:8080/jupyterhub", "JupyterHub别名路径"),
    ]
    
    success_count = 0
    
    for url, description in test_cases:
        if test_redirect(url, description):
            success_count += 1
    
    print(f"\n📊 测试结果:")
    print(f"   ✅ 成功: {success_count}/{len(test_cases)}")
    print(f"   ❌ 失败: {len(test_cases) - success_count}/{len(test_cases)}")
    
    if success_count == len(test_cases):
        print("\n🎉 所有重定向测试通过！端口号保持正确。")
        return 0
    else:
        print("\n⚠️  部分重定向测试失败，需要检查nginx配置。")
        return 1

if __name__ == "__main__":
    sys.exit(main())
