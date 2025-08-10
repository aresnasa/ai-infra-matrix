#!/usr/bin/env python3
"""
测试JupyterHub路由修复效果
"""

import requests
import time

def test_routing():
    """测试路由配置"""
    base_url = "http://localhost:8080"
    
    print("🧪 测试路由配置...")
    
    # 测试1: 直接访问/jupyterhub
    try:
        response = requests.get(f"{base_url}/jupyterhub", timeout=10)
        print(f"✅ /jupyterhub 直接访问: {response.status_code}")
        
        # 检查是否返回了HTML wrapper页面
        if 'JupyterHub' in response.text:
            print("✅ 返回了JupyterHub wrapper页面")
        else:
            print("❌ 未返回预期的JupyterHub内容")
            
    except Exception as e:
        print(f"❌ /jupyterhub 访问失败: {e}")
    
    # 测试2: 检查nginx health
    try:
        response = requests.get(f"{base_url}/health", timeout=5)
        print(f"✅ nginx健康检查: {response.status_code}")
    except Exception as e:
        print(f"❌ nginx健康检查失败: {e}")
    
    # 测试3: 检查前端应用
    try:
        response = requests.get(f"{base_url}/projects", timeout=10)
        print(f"✅ /projects 访问: {response.status_code}")
    except Exception as e:
        print(f"❌ /projects 访问失败: {e}")

if __name__ == "__main__":
    test_routing()
