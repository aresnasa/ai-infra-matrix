#!/usr/bin/env python3
"""
JupyterHub iframe状态监控脚本
"""

import requests
import json
import time
import sys

def check_services():
    """检查所有相关服务的状态"""
    services = {
        "主页": "http://localhost:8080/",
        "JupyterHub包装器": "http://localhost:8080/jupyterhub",
        "认证API": "http://localhost:8080/api/auth/login", 
        "JupyterHub直接访问": "http://localhost:8080/jupyter/hub/",
        "健康检查": "http://localhost:8080/api/health"
    }
    
    print("🔍 检查服务状态...")
    print("=" * 60)
    
    for name, url in services.items():
        try:
            if name == "认证API":
                # 测试POST请求
                response = requests.post(url, 
                    json={"username": "admin", "password": "admin123"},
                    timeout=10
                )
            else:
                response = requests.get(url, timeout=10)
                
            status = "✅ 正常" if response.status_code == 200 else f"⚠️ {response.status_code}"
            print(f"{name:<20}: {status} ({response.status_code})")
            
            # 对于认证API，显示响应内容
            if name == "认证API" and response.status_code == 200:
                try:
                    data = response.json()
                    if 'token' in data:
                        print(f"                    └─ 令牌长度: {len(data['token'])} 字符")
                except:
                    pass
                    
        except requests.exceptions.ConnectionError:
            print(f"{name:<20}: ❌ 连接被拒绝")
        except requests.exceptions.Timeout:
            print(f"{name:<20}: ⏰ 超时")
        except Exception as e:
            print(f"{name:<20}: ❌ 错误 - {str(e)[:50]}")
    
    print("=" * 60)

def monitor_iframe_logs():
    """监控iframe相关的nginx日志"""
    print("\n📊 监控nginx访问日志 (最近10秒)...")
    try:
        # 这里可以添加日志监控逻辑
        import subprocess
        result = subprocess.run([
            "docker", "logs", "--tail", "20", "ai-infra-matrix_nginx_1"
        ], capture_output=True, text=True, timeout=5)
        
        if result.stdout:
            lines = result.stdout.strip().split('\n')[-10:]  # 最后10行
            for line in lines:
                if 'jupyterhub' in line.lower() or 'jupyter' in line.lower():
                    print(f"  🌐 {line}")
    except Exception as e:
        print(f"  ❌ 无法获取nginx日志: {e}")

def test_jwt_authentication():
    """测试JWT认证流程"""
    print("\n🔐 测试JWT认证流程...")
    
    try:
        # 获取JWT令牌
        auth_response = requests.post("http://localhost:8080/api/auth/login", 
            json={"username": "admin", "password": "admin123"},
            timeout=10
        )
        
        if auth_response.status_code != 200:
            print(f"  ❌ 认证失败: {auth_response.status_code}")
            return None
            
        token_data = auth_response.json()
        if 'token' not in token_data:
            print("  ❌ 响应中没有令牌")
            return None
            
        token = token_data['token']
        print(f"  ✅ JWT令牌获取成功，长度: {len(token)}")
        
        # 测试带令牌的JupyterHub访问
        jupyter_url = f"http://localhost:8080/jupyter/hub/?token={token}"
        jupyter_response = requests.get(jupyter_url, timeout=10)
        
        print(f"  🔗 带令牌访问JupyterHub: {jupyter_response.status_code}")
        
        if jupyter_response.status_code == 200:
            content_type = jupyter_response.headers.get('content-type', '')
            if 'text/html' in content_type:
                print("  ✅ 返回HTML内容")
                
                # 检查内容中是否包含JupyterHub相关内容
                content = jupyter_response.text.lower()
                if 'jupyter' in content:
                    print("  ✅ 响应包含JupyterHub内容")
                else:
                    print("  ⚠️ 响应不包含JupyterHub内容")
            else:
                print(f"  ⚠️ 返回非HTML内容: {content_type}")
        
        return token
        
    except Exception as e:
        print(f"  ❌ JWT测试失败: {e}")
        return None

def main():
    """主函数"""
    print("🚀 JupyterHub iframe诊断工具")
    print("=" * 60)
    
    # 检查基础服务
    check_services()
    
    # 测试认证
    token = test_jwt_authentication()
    
    # 监控日志
    monitor_iframe_logs()
    
    print("\n💡 建议:")
    print("1. 在浏览器中访问: http://localhost:8080/jupyterhub?debug=1")
    print("2. 打开浏览器开发者工具查看控制台和网络")
    print("3. 检查iframe是否正确加载JupyterHub内容")
    
    if token:
        print(f"4. 直接测试链接: http://localhost:8080/jupyter/hub/?token={token[:20]}...")
    
    print("\n🔧 如果iframe仍然白屏，可能的原因:")
    print("  - CSP (Content Security Policy) 限制")
    print("  - JupyterHub配置问题")
    print("  - 浏览器缓存问题")
    print("  - JavaScript错误")

if __name__ == "__main__":
    main()
