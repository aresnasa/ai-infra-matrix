#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试JupyterHub别名路径的完整浏览器行为
"""

import requests
import time

def test_jupyterhub_alias():
    """测试/jupyterhub路径的完整行为"""
    print("🔍 测试JupyterHub别名路径 (/jupyterhub) 的浏览器行为")
    print("=" * 60)
    
    session = requests.Session()
    
    # 第一次访问
    print("\n1️⃣ 首次访问 /jupyterhub")
    resp1 = session.get("http://localhost:8080/jupyterhub", allow_redirects=True)
    print(f"   最终URL: {resp1.url}")
    print(f"   状态码: {resp1.status_code}")
    
    # 检查页面内容
    if "JupyterHub" in resp1.text:
        print("   ✅ 成功加载JupyterHub页面")
    else:
        print("   ❌ 页面内容不是JupyterHub")
        
    # 模拟刷新页面
    print("\n2️⃣ 刷新页面 (访问相同URL)")
    resp2 = session.get(resp1.url, allow_redirects=True)
    print(f"   刷新后URL: {resp2.url}")
    print(f"   状态码: {resp2.status_code}")
    
    # 检查URL是否保持
    if "/jupyterhub" in resp1.url and "/jupyterhub" in resp2.url:
        print("   ✅ URL中包含jupyterhub路径")
    elif resp1.url == resp2.url:
        print("   ✅ 刷新后URL保持一致")
    else:
        print("   ⚠️  URL发生了变化")
    
    # 尝试直接访问最终URL
    print("\n3️⃣ 直接访问最终URL")
    resp3 = session.get(resp1.url, allow_redirects=False)
    print(f"   状态码: {resp3.status_code}")
    if resp3.status_code == 200:
        print("   ✅ 直接访问成功，无额外重定向")
    elif resp3.status_code in [301, 302]:
        location = resp3.headers.get('Location', '')
        print(f"   ⚠️  仍有重定向到: {location}")
    
    return resp1.url

def main():
    try:
        final_url = test_jupyterhub_alias()
        
        print(f"\n📋 总结:")
        print(f"   🔗 最终访问URL: {final_url}")
        
        if "jupyterhub" in final_url:
            print("   ✅ 修复成功！URL中保持了 'jupyterhub' 路径")
        elif "/jupyter/" in final_url:
            print("   ⚠️  URL仍然跳转到了 /jupyter/ 路径")
        else:
            print("   ❓ URL跳转到了其他路径")
            
    except Exception as e:
        print(f"❌ 测试异常: {e}")

if __name__ == "__main__":
    main()
