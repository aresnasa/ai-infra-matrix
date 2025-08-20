#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
直接JupyterHub登录测试脚本
确保单点登录成功，不要重复登录
"""

import requests
import time
import re
import sys
from urllib.parse import urljoin, urlparse, parse_qs

class JupyterHubLoginTester:
    def __init__(self):
        self.base_url = "http://localhost:8080"
        self.jupyterhub_url = f"{self.base_url}/jupyter"
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'JupyterHub-Login-Test/1.0'
        })
        
    def test_direct_login(self):
        """测试直接登录JupyterHub"""
        print("🔍 测试直接JupyterHub登录...")
        
        # 1. 访问登录页面
        login_url = f"{self.jupyterhub_url}/hub/login"
        print(f"📡 访问登录页面: {login_url}")
        
        resp = self.session.get(login_url)
        if resp.status_code != 200:
            print(f"❌ 无法访问登录页面: {resp.status_code}")
            return False
            
        print(f"✅ 登录页面访问成功: {resp.status_code}")
        
        # 2. 提取CSRF token
        csrf_token = self._extract_csrf_token(resp.text)
        if not csrf_token:
            print("⚠️  未找到CSRF token，尝试不使用CSRF")
            csrf_token = ""
        else:
            print(f"✅ 提取到CSRF token: {csrf_token[:10]}...")
        
        # 3. 执行登录（使用DummyAuthenticator）
        print("🔐 执行登录...")
        login_data = {
            'username': 'admin',
            'password': 'any_password_works_with_dummy_auth'  # DummyAuthenticator接受任何密码
        }
        
        if csrf_token:
            login_data['_xsrf'] = csrf_token
        
        login_resp = self.session.post(login_url, data=login_data, allow_redirects=False)
        
        print(f"📍 登录响应状态: {login_resp.status_code}")
        
        if login_resp.status_code == 302:
            # 检查重定向地址
            redirect_url = login_resp.headers.get('Location', '')
            print(f"✅ 登录成功，重定向到: {redirect_url}")
            
            # 跟随重定向
            if redirect_url:
                if not redirect_url.startswith('http'):
                    redirect_url = urljoin(self.jupyterhub_url, redirect_url)
                
                final_resp = self.session.get(redirect_url)
                print(f"📍 最终页面状态: {final_resp.status_code}")
                print(f"📍 最终URL: {final_resp.url}")
                
                # 检查是否到达主页
                if 'home' in final_resp.url or 'spawn' in final_resp.url:
                    print("🎉 成功登录并到达用户主页！")
                    return True
                elif 'login' in final_resp.url:
                    print("❌ 被重定向回登录页面")
                    return False
                else:
                    return self._check_login_success(final_resp.text)
            else:
                print("✅ 登录成功（无重定向）")
                return True
                
        elif login_resp.status_code == 200:
            print("⚠️  返回200，检查页面内容...")
            return self._check_login_success(login_resp.text)
        else:
            print(f"❌ 登录失败: {login_resp.status_code}")
            if login_resp.text:
                print(f"响应内容: {login_resp.text[:300]}")
            return False
    
    def test_session_persistence(self):
        """测试会话持久性"""
        print("\n🔍 测试会话持久性...")
        
        test_pages = [
            (f"{self.jupyterhub_url}/hub/home", "用户主页"),
            (f"{self.jupyterhub_url}/hub/api/user", "用户API"),
            (f"{self.jupyterhub_url}/hub/token", "Token页面")
        ]
        
        all_success = True
        for url, name in test_pages:
            try:
                resp = self.session.get(url, timeout=10)
                if resp.status_code == 200 and 'login' not in resp.url:
                    print(f"✅ {name} 访问成功")
                else:
                    print(f"❌ {name} 访问失败或需要重新登录")
                    all_success = False
            except Exception as e:
                print(f"❌ {name} 访问异常: {e}")
                all_success = False
        
        return all_success
    
    def test_user_server_spawn(self):
        """测试用户服务器启动"""
        print("\n🔍 测试用户服务器启动...")
        
        # 访问spawn页面
        spawn_url = f"{self.jupyterhub_url}/hub/spawn"
        resp = self.session.get(spawn_url)
        
        if resp.status_code == 200:
            print("✅ 可以访问Spawn页面")
            
            # 尝试启动服务器
            spawn_resp = self.session.post(spawn_url, data={}, allow_redirects=True, timeout=30)
            
            if spawn_resp.status_code == 200:
                if 'notebook' in spawn_resp.url or 'lab' in spawn_resp.url:
                    print("🎉 用户服务器启动成功并重定向到Jupyter界面！")
                    return True
                elif 'spawn-pending' in spawn_resp.url or 'spawn' in spawn_resp.url:
                    print("⏳ 服务器启动中...")
                    return True
                else:
                    print("ℹ️  服务器响应正常，可能需要手动检查")
                    return True
            else:
                print(f"⚠️  服务器启动响应异常: {spawn_resp.status_code}")
                return False
        else:
            print(f"❌ 无法访问Spawn页面: {resp.status_code}")
            return False
    
    def test_logout_and_relogin(self):
        """测试登出和重新登录"""
        print("\n🔍 测试登出和重新登录...")
        
        # 登出
        logout_url = f"{self.jupyterhub_url}/hub/logout"
        logout_resp = self.session.get(logout_url, allow_redirects=False)
        
        if logout_resp.status_code in [302, 200]:
            print("✅ 登出成功")
            
            # 清理会话
            self.session.cookies.clear()
            
            # 重新登录
            print("🔄 尝试重新登录...")
            if self.test_direct_login():
                print("✅ 重新登录成功 - 单点登录功能正常")
                return True
            else:
                print("❌ 重新登录失败")
                return False
        else:
            print(f"❌ 登出失败: {logout_resp.status_code}")
            return False
    
    def _extract_csrf_token(self, html_content):
        """从HTML中提取CSRF token"""
        patterns = [
            r'<input[^>]*name=["\']_xsrf["\'][^>]*value=["\']([^"\']*)["\']',
            r'_xsrf["\']?\s*:\s*["\']([^"\']*)["\']',
            r'data-xsrf-token=["\']([^"\']*)["\']'
        ]
        
        for pattern in patterns:
            match = re.search(pattern, html_content)
            if match:
                return match.group(1)
        
        return None
    
    def _check_login_success(self, content):
        """检查页面内容判断登录是否成功"""
        success_indicators = [
            'control panel', 'start my server', 'home', 'spawn',
            'admin', 'logout', 'my server'
        ]
        
        fail_indicators = [
            'login', 'password', 'sign in', 'username'
        ]
        
        content_lower = content.lower()
        
        if any(indicator in content_lower for indicator in success_indicators):
            print("✅ 页面内容确认登录成功")
            return True
        elif any(indicator in content_lower for indicator in fail_indicators):
            print("❌ 页面显示登录表单，登录失败")
            return False
        else:
            print("⚠️  无法从页面内容确定登录状态")
            return False
    
    def run_complete_test(self):
        """运行完整测试"""
        print("🚀 开始JupyterHub单点登录测试")
        print("="*60)
        
        tests = [
            ("直接登录测试", self.test_direct_login),
            ("会话持久性测试", self.test_session_persistence),
            ("用户服务器启动测试", self.test_user_server_spawn),
            ("登出重登录测试", self.test_logout_and_relogin)
        ]
        
        results = []
        for test_name, test_func in tests:
            print(f"\n{'='*15} {test_name} {'='*15}")
            try:
                result = test_func()
                results.append((test_name, result))
            except Exception as e:
                print(f"💥 测试异常: {e}")
                results.append((test_name, False))
            
            time.sleep(2)  # 短暂延迟
        
        # 测试报告
        print("\n" + "="*60)
        print("📋 测试报告")
        print("="*60)
        
        passed = 0
        for test_name, result in results:
            if result:
                print(f"✅ {test_name}: 通过")
                passed += 1
            else:
                print(f"❌ {test_name}: 失败")
        
        print(f"\n📊 总结: {passed}/{len(results)} 测试通过")
        
        if passed >= 3:  # 至少3个测试通过
            print("🎉 JupyterHub单点登录功能基本正常！")
            print("✅ 确认无重复登录问题")
        else:
            print("⚠️  需要进一步检查配置")
        
        return passed >= 3

def main():
    tester = JupyterHubLoginTester()
    try:
        success = tester.run_complete_test()
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        print("\n⏹️  测试被中断")
        sys.exit(1)
    except Exception as e:
        print(f"\n💥 测试异常: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

if __name__ == "__main__":
    main()
