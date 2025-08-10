#!/usr/bin/env python3
"""
手动验证iframe内容的简单脚本
"""

import requests
import time

def manual_check():
    """手动检查各个端点"""
    
    print("🔍 手动验证各个端点...")
    
    endpoints = [
        ("主页", "http://localhost:8080/"),
        ("iframe测试页", "http://localhost:8080/iframe_test.html"),
        ("JupyterHub直接访问", "http://localhost:8080/jupyterhub"),
        ("JupyterHub登录", "http://localhost:8080/jupyterhub/hub/login"),
        ("Projects页面", "http://localhost:8080/projects"),
    ]
    
    for name, url in endpoints:
        try:
            print(f"\n📍 检查 {name}: {url}")
            response = requests.get(url, timeout=10)
            print(f"  状态码: {response.status_code}")
            print(f"  内容长度: {len(response.text)} 字符")
            
            # 检查内容关键词
            content = response.text.lower()
            if "404" in content or "not found" in content:
                print("  ❌ 包含404错误")
            elif "500" in content or "internal server error" in content:
                print("  ❌ 包含服务器错误")
            elif len(response.text.strip()) < 100:
                print("  ⚠️ 内容很少")
            elif "jupyter" in content:
                print("  ✅ 包含JupyterHub相关内容")
            elif "login" in content:
                print("  ✅ 包含登录相关内容")
            else:
                print("  ✅ 有内容")
                
        except Exception as e:
            print(f"  ❌ 请求失败: {e}")
    
    print("\n🏁 手动验证完成")

if __name__ == "__main__":
    manual_check()
