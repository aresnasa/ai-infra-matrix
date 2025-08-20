#!/usr/bin/env python3
"""
测试JupyterHub API端点是否在iframe中正常工作
"""

import requests
import time
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

def test_api_endpoints():
    """测试常见的JupyterHub API端点"""
    
    base_url = "http://localhost:8080"
    
    # 配置重试策略
    session = requests.Session()
    retry_strategy = Retry(
        total=3,
        status_forcelist=[429, 500, 502, 503, 504],
        allowed_methods=["HEAD", "GET", "OPTIONS"]
    )
    adapter = HTTPAdapter(max_retries=retry_strategy)
    session.mount("http://", adapter)
    session.mount("https://", adapter)
    
    # 测试端点列表
    endpoints = [
        "/jupyter/user/admin/api/sessions",
        "/jupyter/user/admin/api/kernels", 
        "/jupyter/user/admin/api/kernelspecs",
        "/jupyter/user/admin/api/terminals",
        "/jupyter/hub/api/user"
    ]
    
    print("🔍 测试JupyterHub API端点...")
    print("=" * 50)
    
    for endpoint in endpoints:
        url = f"{base_url}{endpoint}"
        print(f"\n📡 测试: {endpoint}")
        
        try:
            # 添加必要的头部信息
            headers = {
                'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36',
                'Accept': 'application/json',
                'Referer': f'{base_url}/jupyterhub',
                'X-Requested-With': 'XMLHttpRequest'
            }
            
            response = session.get(url, headers=headers, timeout=10)
            
            if response.status_code == 200:
                print(f"   ✅ 成功 (200) - 响应长度: {len(response.content)} bytes")
            elif response.status_code == 401:
                print(f"   ⚠️  未授权 (401) - 可能需要登录")
            elif response.status_code == 404:
                print(f"   ❌ 未找到 (404) - 端点不存在或路由错误")
            else:
                print(f"   ⚠️  状态码: {response.status_code}")
                
        except requests.exceptions.Timeout:
            print(f"   ⏰ 超时 - 请求耗时超过10秒")
        except requests.exceptions.ConnectionError:
            print(f"   🔌 连接错误 - 无法连接到服务器")
        except Exception as e:
            print(f"   💥 异常: {str(e)}")
            
        time.sleep(0.5)  # 避免请求过快
    
    print("\n" + "=" * 50)
    print("🏁 测试完成!")
    
    # 测试wrapper页面
    print(f"\n📄 测试wrapper页面...")
    try:
        wrapper_url = f"{base_url}/jupyterhub"
        response = session.get(wrapper_url, timeout=10)
        if response.status_code == 200:
            print(f"   ✅ /jupyterhub 页面正常 (200)")
            if "jupyterhub-frame" in response.text:
                print(f"   ✅ iframe元素存在")
            else:
                print(f"   ⚠️  iframe元素未找到")
        else:
            print(f"   ❌ /jupyterhub 页面错误 ({response.status_code})")
    except Exception as e:
        print(f"   💥 wrapper页面测试失败: {str(e)}")

if __name__ == "__main__":
    test_api_endpoints()
