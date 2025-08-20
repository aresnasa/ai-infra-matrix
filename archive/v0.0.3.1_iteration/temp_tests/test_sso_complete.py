#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
AI-Infra-Matrix SSO单点登录测试脚本
完整测试SSO流程：前端登录 -> Cookie设置 -> JupyterHub自动登录
"""

import requests
import time
import json
import sys
from urllib.parse import urljoin, urlparse, parse_qs
import re

class SSOTester:
    def __init__(self):
        self.base_url = "http://localhost:8080"
        self.backend_url = f"{self.base_url}/api"
        self.jupyterhub_url = f"{self.base_url}/jupyter"
        self.session = requests.Session()
        self.auth_token = None
        
    def test_complete_sso_flow(self):
        """测试完整的SSO流程"""
        print("🚀 开始完整SSO单点登录测试")
        print("=" * 60)
        
        # 1. 后端登录获取JWT token
        if not self.test_backend_login():
            return False
            
        # 2. 设置前端认证cookie
        self.setup_frontend_cookies()
        
        # 3. 测试SSO桥接页面
        if not self.test_sso_bridge():
            return False
            
        # 4. 测试JupyterHub自动登录
        if not self.test_jupyterhub_auto_login():
            return False
            
        # 5. 验证用户状态
        self.verify_user_session()
        
        print("\n🎉 SSO单点登录测试完成！")
        return True
    
    def test_backend_login(self):
        """测试后端认证登录"""
        print("\n1️⃣ 测试后端认证登录...")
        
        login_data = {
            "username": "admin",
            "password": "admin123"
        }
        
        try:
            resp = self.session.post(
                f"{self.backend_url}/auth/login",
                json=login_data,
                timeout=10
            )
            
            if resp.status_code == 200:
                result = resp.json()
                self.auth_token = result.get('token')
                
                if self.auth_token:
                    print(f"   ✅ 后端登录成功")
                    print(f"   📝 获取JWT token: {self.auth_token[:20]}...")
                    print(f"   ⏰ 过期时间: {result.get('expires_at', 'N/A')}")
                    return True
                else:
                    print(f"   ❌ 响应中缺少token: {result}")
                    return False
            else:
                print(f"   ❌ 后端登录失败: {resp.status_code}")
                print(f"   📄 响应内容: {resp.text}")
                return False
                
        except Exception as e:
            print(f"   ❌ 登录请求异常: {e}")
            return False
    
    def setup_frontend_cookies(self):
        """设置前端认证cookies"""
        print("\n2️⃣ 设置前端认证cookies...")
        
        if not self.auth_token:
            print("   ❌ 缺少认证token")
            return
        
        # 设置多种格式的cookie以确保兼容性
        cookie_names = ['ai_infra_token', 'jwt_token', 'auth_token']
        
        for cookie_name in cookie_names:
            self.session.cookies.set(
                cookie_name, 
                self.auth_token, 
                domain='localhost', 
                path='/'
            )
            print(f"   ✅ 设置cookie: {cookie_name}")
        
        # 设置用户信息cookie（模拟前端行为）
        user_info = {
            "username": "admin",
            "roles": ["admin"],
            "permissions": ["all"]
        }
        
        import urllib.parse
        user_info_str = urllib.parse.quote(json.dumps(user_info))
        self.session.cookies.set(
            'user_info',
            user_info_str,
            domain='localhost',
            path='/'
        )
        print(f"   ✅ 设置user_info cookie")
    
    def test_sso_bridge(self):
        """测试SSO桥接页面"""
        print("\n3️⃣ 测试SSO桥接页面...")
        
        sso_url = f"{self.base_url}/sso?next={self.base_url}/jupyter/hub/"
        
        try:
            resp = self.session.get(sso_url, timeout=10, allow_redirects=True)
            
            print(f"   📊 SSO页面状态: {resp.status_code}")
            print(f"   📍 最终URL: {resp.url}")
            
            if resp.status_code == 200:
                # 检查页面内容
                if ('jwt_sso_bridge.html' in resp.url or 
                    '单点登录' in resp.text or 
                    'AI基础设施矩阵' in resp.text or
                    'performSSO' in resp.text):
                    print("   ✅ SSO桥接页面加载成功")
                    
                    # 检查JavaScript自动登录逻辑
                    if 'performSSO' in resp.text:
                        print("   ✅ 发现SSO处理函数")
                    if 'localStorage.getItem' in resp.text:
                        print("   ✅ 发现token读取逻辑")
                    if 'JupyterHub' in resp.text:
                        print("   ✅ 发现JupyterHub集成")
                    
                    return True
                else:
                    print("   ⚠️  页面内容不符合预期")
                    # 显示页面的前几行用于调试
                    print(f"   📄 页面开头: {resp.text[:200]}...")
                    return False
            elif resp.status_code == 302:
                location = resp.headers.get('Location', '')
                print(f"   📍 重定向到: {location}")
                if 'jupyter' in location:
                    print("   ✅ 自动重定向到JupyterHub")
                    return True
                else:
                    print("   ⚠️  重定向目标不明确")
                    return False
            else:
                print(f"   ❌ SSO页面访问失败: {resp.status_code}")
                return False
                
        except Exception as e:
            print(f"   ❌ SSO页面请求异常: {e}")
            return False
    
    def test_jupyterhub_auto_login(self):
        """测试JupyterHub自动登录"""
        print("\n4️⃣ 测试JupyterHub自动登录...")
        
        # 直接访问JupyterHub，测试是否能自动登录
        hub_url = f"{self.jupyterhub_url}/hub/"
        
        try:
            # 添加Authorization header作为备用认证方式
            headers = {
                'Authorization': f'Bearer {self.auth_token}'
            }
            
            resp = self.session.get(
                hub_url, 
                timeout=15, 
                allow_redirects=True,
                headers=headers
            )
            
            print(f"   📊 JupyterHub访问状态: {resp.status_code}")
            print(f"   📍 最终URL: {resp.url}")
            
            # 分析最终URL来判断登录状态
            final_url = resp.url
            
            if '/login' in final_url:
                print("   ⚠️  仍然显示登录页面，检查自动登录逻辑...")
                
                # 检查登录页面是否包含自动登录脚本
                if 'autoLoginWithToken' in resp.text:
                    print("   ✅ 发现自动登录脚本")
                if 'ai_infra_token' in resp.text:
                    print("   ✅ 发现token检测逻辑")
                
                # 尝试手动触发认证
                return self.try_manual_auth()
                
            elif any(keyword in final_url for keyword in ['/hub/home', '/hub/spawn', '/user/']):
                print("   🎉 自动登录成功！已进入JupyterHub用户界面")
                return True
                
            elif resp.status_code == 200 and 'jupyter' in resp.text.lower():
                print("   ✅ 成功访问JupyterHub")
                # 检查页面内容确认登录状态
                if 'logout' in resp.text.lower() or 'spawn' in resp.text.lower():
                    print("   🎉 用户已登录JupyterHub")
                    return True
                else:
                    print("   ⚠️  登录状态不明确")
                    return False
            else:
                print(f"   ❌ 未知响应状态")
                print(f"   📄 页面标题: {self.extract_title(resp.text)}")
                return False
                
        except Exception as e:
            print(f"   ❌ JupyterHub访问异常: {e}")
            return False
    
    def try_manual_auth(self):
        """尝试手动认证"""
        print("\n   🔄 尝试手动认证...")
        
        # 尝试带token参数的URL
        token_url = f"{self.jupyterhub_url}/hub/login?token={self.auth_token}"
        
        try:
            resp = self.session.get(token_url, timeout=10, allow_redirects=True)
            
            print(f"   📊 Token认证状态: {resp.status_code}")
            print(f"   📍 最终URL: {resp.url}")
            
            if '/hub/home' in resp.url or '/hub/spawn' in resp.url or '/user/' in resp.url:
                print("   ✅ Token参数认证成功！")
                return True
            else:
                print("   ⚠️  Token参数认证未成功")
                return False
                
        except Exception as e:
            print(f"   ❌ Token认证异常: {e}")
            return False
    
    def verify_user_session(self):
        """验证用户会话状态"""
        print("\n5️⃣ 验证用户会话状态...")
        
        # 尝试访问JupyterHub API
        api_url = f"{self.jupyterhub_url}/hub/api/user"
        
        try:
            resp = self.session.get(api_url, timeout=10)
            
            if resp.status_code == 200:
                user_info = resp.json()
                print(f"   ✅ API访问成功")
                print(f"   👤 当前用户: {user_info.get('name', 'Unknown')}")
                print(f"   🏠 用户服务器状态: {user_info.get('servers', {})}")
                
            elif resp.status_code == 401:
                print("   ⚠️  API访问需要认证，会话可能未建立")
                
            else:
                print(f"   ⚠️  API访问状态: {resp.status_code}")
                
        except Exception as e:
            print(f"   ⚠️  API访问异常: {e}")
    
    def extract_title(self, html_content):
        """提取页面标题"""
        try:
            title_match = re.search(r'<title[^>]*>([^<]+)</title>', html_content, re.IGNORECASE)
            return title_match.group(1).strip() if title_match else "No Title"
        except:
            return "Title Extraction Failed"
    
    def test_token_verification(self):
        """测试token验证端点"""
        print("\n🔍 额外测试：Token验证端点...")
        
        if not self.auth_token:
            print("   ❌ 缺少认证token")
            return
        
        # 测试新的简单验证端点
        verify_url = f"{self.backend_url}/auth/verify"
        headers = {'Authorization': f'Bearer {self.auth_token}'}
        
        try:
            resp = self.session.get(verify_url, headers=headers, timeout=5)
            
            if resp.status_code == 200:
                user_data = resp.json()
                print(f"   ✅ Token验证成功")
                print(f"   👤 用户: {user_data.get('username')}")
                print(f"   📧 邮箱: {user_data.get('email')}")
                print(f"   🏷️  角色: {user_data.get('roles', [])}")
            else:
                print(f"   ❌ Token验证失败: {resp.status_code}")
                print(f"   📄 响应: {resp.text}")
                
        except Exception as e:
            print(f"   ❌ Token验证异常: {e}")

def main():
    """主函数"""
    print("🔐 AI基础设施矩阵 - SSO单点登录测试")
    print("=" * 60)
    
    tester = SSOTester()
    
    try:
        # 运行完整SSO测试
        success = tester.test_complete_sso_flow()
        
        # 额外的token验证测试
        tester.test_token_verification()
        
        if success:
            print("\n🎉 SSO测试完成 - 单点登录正常工作！")
            print("\n💡 使用方法：")
            print("   1. 在前端 http://localhost:8080 登录")
            print("   2. 直接访问 http://localhost:8080/jupyter/hub/")
            print("   3. 应该自动登录，无需重复输入密码")
            
            print("\n🛠️  如果SSO不工作，尝试：")
            print("   1. 访问 http://localhost:8080/sso 手动触发SSO")
            print("   2. 检查浏览器cookie是否正确设置")
            print("   3. 查看浏览器开发者工具的网络请求")
            
            return 0
        else:
            print("\n❌ SSO测试失败 - 需要检查配置")
            return 1
            
    except KeyboardInterrupt:
        print("\n\n⚠️  测试被用户中断")
        return 1
    except Exception as e:
        print(f"\n❌ 测试过程中发生异常: {e}")
        return 1

if __name__ == "__main__":
    sys.exit(main())
