#!/usr/bin/env python3
"""
快速验证admin/admin123登录功能
"""

import requests
import json
import time

def test_admin_login():
    """测试admin/admin123登录"""
    print("🔐 测试admin/admin123登录功能")
    print("=" * 50)
    
    base_url = "http://localhost:8080"
    
    # 测试1: 直接API登录
    print("\n📝 测试1: API登录")
    try:
        response = requests.post(f"{base_url}/api/auth/login", 
            json={"username": "admin", "password": "admin123"},
            timeout=10
        )
        
        if response.status_code == 200:
            data = response.json()
            print("✅ API登录成功")
            print(f"   Token长度: {len(data.get('token', ''))}")
            print(f"   用户ID: {data.get('user', {}).get('id')}")
            print(f"   用户名: {data.get('user', {}).get('username')}")
            
            token = data.get('token')
            return token
        else:
            print(f"❌ API登录失败: {response.status_code}")
            return None
            
    except Exception as e:
        print(f"❌ API登录异常: {e}")
        return None

def test_jupyterhub_access(token):
    """测试JupyterHub访问"""
    print("\n📝 测试2: JupyterHub访问")
    
    base_url = "http://localhost:8080"
    
    # 测试2a: 无token的JupyterHub访问
    try:
        response = requests.get(f"{base_url}/jupyterhub", timeout=10)
        print(f"   无token访问: {response.status_code}")
        
        if response.status_code == 200:
            print("✅ 可以无token访问JupyterHub页面")
        else:
            print("⚠️ 无token访问需要重定向")
    except Exception as e:
        print(f"❌ 无token访问失败: {e}")
    
    # 测试2b: 带token的JupyterHub访问
    if token:
        try:
            response = requests.get(f"{base_url}/jupyter/hub/?token={token}", timeout=10)
            print(f"   带token访问: {response.status_code}")
            
            if response.status_code == 200:
                print("✅ 带token可以访问JupyterHub")
                
                # 检查响应内容
                content = response.text.lower()
                if 'jupyter' in content:
                    print("✅ 响应包含JupyterHub内容")
                else:
                    print("⚠️ 响应不包含JupyterHub关键词")
            else:
                print("⚠️ 带token访问返回非200状态")
                
        except Exception as e:
            print(f"❌ 带token访问失败: {e}")

def test_sso_flow():
    """测试SSO流程"""
    print("\n📝 测试3: SSO流程")
    
    session = requests.Session()
    base_url = "http://localhost:8080"
    
    try:
        # 步骤1: 获取主页
        print("   1. 访问主页...")
        response = session.get(base_url, timeout=10)
        print(f"      状态: {response.status_code}")
        
        # 步骤2: 访问项目页面
        print("   2. 访问项目页面...")
        response = session.get(f"{base_url}/projects", timeout=10)
        print(f"      状态: {response.status_code}")
        
        # 步骤3: 尝试直接访问JupyterHub
        print("   3. 访问JupyterHub包装器...")
        response = session.get(f"{base_url}/jupyterhub", timeout=10)
        print(f"      状态: {response.status_code}")
        
        if response.status_code == 200:
            content = response.text
            if "login" in content.lower():
                print("⚠️ 需要登录")
                
                # 尝试登录
                print("   4. 执行登录...")
                login_response = session.post(f"{base_url}/api/auth/login",
                    json={"username": "admin", "password": "admin123"},
                    timeout=10
                )
                
                if login_response.status_code == 200:
                    print("✅ 登录成功")
                    
                    # 再次访问JupyterHub
                    print("   5. 登录后重新访问JupyterHub...")
                    response = session.get(f"{base_url}/jupyterhub", timeout=10)
                    print(f"      状态: {response.status_code}")
                    
                    if response.status_code == 200:
                        print("✅ SSO流程正常工作")
                        return True
                    else:
                        print("❌ 登录后JupyterHub访问失败")
                        return False
                else:
                    print("❌ 登录失败")
                    return False
            else:
                print("✅ 无需登录，直接可以访问")
                return True
        else:
            print("❌ JupyterHub访问失败")
            return False
            
    except Exception as e:
        print(f"❌ SSO流程测试失败: {e}")
        return False

def main():
    """主函数"""
    print("🚀 admin/admin123 登录功能快速验证")
    print("=" * 60)
    
    # 测试API登录
    token = test_admin_login()
    
    # 测试JupyterHub访问
    test_jupyterhub_access(token)
    
    # 测试SSO流程
    sso_success = test_sso_flow()
    
    # 生成总结
    print("\n" + "=" * 60)
    print("📊 测试总结")
    print("=" * 60)
    
    if token:
        print("✅ admin/admin123 凭据有效")
    else:
        print("❌ admin/admin123 凭据无效")
    
    if sso_success:
        print("✅ SSO流程正常工作")
        print("💡 用户可以使用admin/admin123登录后无需再次输入密码访问JupyterHub")
    else:
        print("❌ SSO流程存在问题")
        print("💡 可能需要手动登录或修复认证集成")
    
    print("\n🎯 结论:")
    if token and sso_success:
        print("   admin/admin123 自动登录功能正常！")
        print("   用户体验: 登录一次 → 可以无缝访问所有服务")
    else:
        print("   需要进一步调试登录或SSO集成问题")

if __name__ == "__main__":
    main()
