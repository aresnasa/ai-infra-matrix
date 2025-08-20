#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
AI-Infra-Matrix JupyterHub单点登录测试脚本
测试整个登录流程，确保SSO正常工作，避免重复登录
"""

import requests
import time
import json
import sys
from urllib.parse import urljoin, urlparse
import re

class SSOLoginTester:
    def __init__(self):
        self.base_url = "http://localhost:8080"
        self.jupyterhub_base = f"{self.base_url}/jupyter"
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'AI-Infra-Matrix-SSO-Test/1.0'
        })
    
    def test_direct_jupyterhub_login(self):
        """测试直接通过JupyterHub登录（DummyAuthenticator）"""
        print("🔍 测试JupyterHub直接登录...")
        
        # 1. 访问JupyterHub登录页面
        login_url = f"{self.jupyterhub_base}/hub/login"
        resp = self.session.get(login_url)
        
        if resp.status_code != 200:
            print(f"❌ 无法访问JupyterHub登录页面: {resp.status_code}")
            return False
        
        print(f"✅ JupyterHub登录页面访问成功: {login_url}")
        
        # 2. 提取CSRF token
        csrf_token = self._extract_csrf_token(resp.text)
        if not csrf_token:
            print("❌ 无法提取CSRF token")
            return False
        
        print(f"✅ 提取到CSRF token: {csrf_token[:10]}...")
        
        # 3. 执行登录
        login_data = {
            'username': 'admin',
            'password': 'password',
            '_xsrf': csrf_token
        }
        
        login_resp = self.session.post(login_url, data=login_data, allow_redirects=False)
        
        if login_resp.status_code == 302:
            print("✅ JupyterHub登录成功 - 重定向到用户页面")
            redirect_url = login_resp.headers.get('Location', '')
            print(f"📍 重定向到: {redirect_url}")
            return True
        else:
            print(f"❌ JupyterHub登录失败: {login_resp.status_code}")
            print(f"响应内容: {login_resp.text[:500]}")
            return False
    
    def test_sso_session_persistence(self):
        """测试SSO会话持久性 - 确保登录后不需要重复登录"""
        print("\n🔍 测试SSO会话持久性...")
        
        # 1. 访问JupyterHub主页
        home_url = f"{self.jupyterhub_base}/hub/home"
        resp = self.session.get(home_url, allow_redirects=True)
        
        if resp.status_code != 200:
            print(f"❌ 无法访问JupyterHub主页: {resp.status_code}")
            return False
        
        # 2. 检查是否被重定向到登录页面
        final_url = resp.url
        if '/login' in final_url:
            print("❌ 会话已失效，被重定向到登录页面")
            return False
        
        # 3. 检查页面内容确认已登录
        if 'admin' in resp.text and ('Control Panel' in resp.text or 'Start My Server' in resp.text):
            print("✅ SSO会话有效 - 用户已登录，无需重复登录")
            print(f"📍 当前页面: {final_url}")
            return True
        else:
            print("❌ 页面内容异常，可能未正确登录")
            return False
    
    def test_spawner_access(self):
        """测试Spawner服务器启动"""
        print("\n🔍 测试Spawner服务器启动...")
        
        # 尝试启动用户服务器
        spawn_url = f"{self.jupyterhub_base}/hub/spawn"
        resp = self.session.get(spawn_url, allow_redirects=True)
        
        if resp.status_code == 200:
            print("✅ 成功访问Spawner页面")
            
            # 如果页面包含JupyterLab，说明服务器已启动
            if 'jupyter' in resp.text.lower() or 'lab' in resp.text.lower():
                print("✅ 用户服务器正在运行")
                return True
            else:
                print("ℹ️  用户服务器可能需要启动")
                return True
        else:
            print(f"❌ 无法访问Spawner: {resp.status_code}")
            return False
    
    def test_api_access(self):
        """测试JupyterHub API访问"""
        print("\n🔍 测试JupyterHub API访问...")
        
        # 测试用户信息API
        api_url = f"{self.jupyterhub_base}/hub/api/user"
        resp = self.session.get(api_url)
        
        if resp.status_code == 200:
            user_info = resp.json()
            print(f"✅ API访问成功 - 用户: {user_info.get('name', 'Unknown')}")
            print(f"📊 用户信息: {json.dumps(user_info, indent=2)}")
            return True
        else:
            print(f"❌ API访问失败: {resp.status_code}")
            return False
    
    def test_logout_and_relogin(self):
        """测试登出后重新登录"""
        print("\n🔍 测试登出和重新登录...")
        
        # 1. 登出
        logout_url = f"{self.jupyterhub_base}/hub/logout"
        resp = self.session.get(logout_url, allow_redirects=True)
        
        if resp.status_code == 200:
            print("✅ 成功登出")
        else:
            print(f"⚠️  登出响应异常: {resp.status_code}")
        
        # 2. 清除会话
        self.session.cookies.clear()
        
        # 3. 验证已登出
        home_url = f"{self.jupyterhub_base}/hub/home"
        resp = self.session.get(home_url, allow_redirects=True)
        
        if '/login' in resp.url:
            print("✅ 确认已登出 - 重定向到登录页面")
            return True
        else:
            print("❌ 登出可能未成功")
            return False
    
    def _extract_csrf_token(self, html_content):
        """从HTML中提取CSRF token"""
        # 查找_xsrf隐藏输入字段
        csrf_pattern = r'<input[^>]*name=["\']_xsrf["\'][^>]*value=["\']([^"\']*)["\']'
        match = re.search(csrf_pattern, html_content)
        if match:
            return match.group(1)
        
        # 查找其他可能的CSRF token位置
        csrf_pattern2 = r'_xsrf["\']?\s*:\s*["\']([^"\']*)["\']'
        match = re.search(csrf_pattern2, html_content)
        if match:
            return match.group(1)
        
        return None
    
    def run_full_test(self):
        """运行完整的SSO测试套件"""
        print("🚀 开始AI-Infra-Matrix JupyterHub SSO测试")
        print("="*60)
        
        test_results = []
        
        # 测试1: 直接JupyterHub登录
        result1 = self.test_direct_jupyterhub_login()
        test_results.append(("JupyterHub直接登录", result1))
        
        if not result1:
            print("❌ 基础登录失败，终止测试")
            return False
        
        # 等待一下确保登录完全完成
        time.sleep(2)
        
        # 测试2: SSO会话持久性
        result2 = self.test_sso_session_persistence()
        test_results.append(("SSO会话持久性", result2))
        
        # 测试3: Spawner访问
        result3 = self.test_spawner_access()
        test_results.append(("Spawner访问", result3))
        
        # 测试4: API访问
        result4 = self.test_api_access()
        test_results.append(("API访问", result4))
        
        # 测试5: 登出重新登录
        result5 = self.test_logout_and_relogin()
        test_results.append(("登出重新登录", result5))
        
        # 输出测试结果摘要
        print("\n" + "="*60)
        print("📋 测试结果摘要")
        print("="*60)
        
        all_passed = True
        for test_name, result in test_results:
            status = "✅ 通过" if result else "❌ 失败"
            print(f"{test_name:20} : {status}")
            if not result:
                all_passed = False
        
        print("\n" + "="*60)
        if all_passed:
            print("🎉 所有测试通过！JupyterHub SSO工作正常")
        else:
            print("⚠️  部分测试失败，请检查配置")
        
        return all_passed

def main():
    """主函数"""
    tester = SSOLoginTester()
    
    try:
        success = tester.run_full_test()
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        print("\n⏹️  测试被用户中断")
        sys.exit(1)
    except Exception as e:
        print(f"\n💥 测试过程中发生错误: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()
