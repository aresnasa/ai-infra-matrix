#!/usr/bin/env python3
"""
测试增强的SSO功能：
1. 登录状态同步（后端退出时的处理）
2. 用户特定重定向（/gitea/ -> /gitea/$user）
3. 不同用户的登录态状态管理
"""

import requests
import time
import json
import subprocess
import sys
from urllib.parse import urlparse, parse_qs
import re

class EnhancedSSOTester:
    def __init__(self):
        self.base_url = "http://localhost:8080"
        self.session = requests.Session()
        self.session.verify = False  # 忽略SSL警告
        
    def print_result(self, test_name, success, details=""):
        status = "✅ PASS" if success else "❌ FAIL"
        print(f"{status} {test_name}")
        if details:
            print(f"   Details: {details}")
        print()
    
    def check_service_status(self):
        """检查服务状态"""
        print("🔍 检查服务状态...")
        
        services = [
            ("Frontend", f"{self.base_url}/"),
            ("Backend", f"{self.base_url}/api/health"),
            ("Gitea", f"{self.base_url}/gitea/"),
            ("Nginx Auth", f"{self.base_url}/__auth/verify")
        ]
        
        all_healthy = True
        for name, url in services:
            try:
                resp = self.session.get(url, timeout=5)
                if resp.status_code < 500:
                    print(f"   ✅ {name}: HTTP {resp.status_code}")
                else:
                    print(f"   ❌ {name}: HTTP {resp.status_code}")
                    all_healthy = False
            except Exception as e:
                print(f"   ❌ {name}: {str(e)}")
                all_healthy = False
        
        return all_healthy
    
    def test_user_redirection(self):
        """测试 /gitea/ 到 /gitea/$user 的重定向"""
        print("🔄 测试用户特定重定向...")
        
        # 首先登录获取token
        try:
            # 尝试访问SSO登录
            login_resp = self.session.post(f"{self.base_url}/api/auth/login", json={
                "username": "admin",
                "password": "admin123"
            })
            
            if login_resp.status_code == 200:
                token_data = login_resp.json()
                print(f"   ✅ 登录成功: {token_data.get('message', 'OK')}")
                
                # 测试 /gitea/ 重定向
                gitea_resp = self.session.get(f"{self.base_url}/gitea/", allow_redirects=False)
                
                if gitea_resp.status_code in [301, 302]:
                    location = gitea_resp.headers.get('Location', '')
                    if '/gitea/admin' in location or '/gitea/' in location:
                        self.print_result("用户重定向", True, f"重定向到: {location}")
                        return True
                    else:
                        self.print_result("用户重定向", False, f"意外的重定向位置: {location}")
                else:
                    self.print_result("用户重定向", False, f"没有重定向，状态码: {gitea_resp.status_code}")
            else:
                self.print_result("用户重定向", False, f"登录失败: {login_resp.status_code}")
        
        except Exception as e:
            self.print_result("用户重定向", False, f"异常: {str(e)}")
        
        return False
    
    def test_backend_exit_handling(self):
        """测试后端退出时的登录状态同步"""
        print("🛑 测试后端退出处理...")
        
        try:
            # 先确保登录
            login_resp = self.session.post(f"{self.base_url}/api/auth/login", json={
                "username": "admin", 
                "password": "admin123"
            })
            
            if login_resp.status_code == 200:
                print("   ✅ 初始登录成功")
                
                # 停止后端服务来模拟退出
                print("   🔄 停止后端服务...")
                stop_result = subprocess.run(
                    ["docker-compose", "stop", "backend"], 
                    cwd="/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix",
                    capture_output=True, text=True
                )
                
                if stop_result.returncode == 0:
                    print("   ✅ 后端服务已停止")
                    
                    # 等待一下让状态生效
                    time.sleep(2)
                    
                    # 测试访问Gitea时的处理
                    gitea_resp = self.session.get(f"{self.base_url}/gitea/", allow_redirects=False)
                    
                    # 检查响应头中的调试信息
                    debug_action = gitea_resp.headers.get('X-Debug-SSO-Action', '')
                    
                    if 'backend_error' in debug_action or gitea_resp.status_code in [502, 503, 504]:
                        self.print_result("后端退出处理", True, 
                                        f"正确处理后端不可用，状态: {gitea_resp.status_code}, 动作: {debug_action}")
                        result = True
                    else:
                        self.print_result("后端退出处理", False, 
                                        f"未正确处理后端退出，状态: {gitea_resp.status_code}, 动作: {debug_action}")
                        result = False
                    
                    # 重启后端服务
                    print("   🔄 重启后端服务...")
                    start_result = subprocess.run(
                        ["docker-compose", "start", "backend"], 
                        cwd="/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix",
                        capture_output=True, text=True
                    )
                    
                    if start_result.returncode == 0:
                        print("   ✅ 后端服务已重启")
                        time.sleep(3)  # 等待服务完全启动
                    
                    return result
                else:
                    self.print_result("后端退出处理", False, f"无法停止后端服务: {stop_result.stderr}")
            else:
                self.print_result("后端退出处理", False, f"初始登录失败: {login_resp.status_code}")
        
        except Exception as e:
            self.print_result("后端退出处理", False, f"异常: {str(e)}")
        
        return False
    
    def test_token_expiry_handling(self):
        """测试Token过期处理"""
        print("⏰ 测试Token过期处理...")
        
        try:
            # 设置一个过期的token
            expired_token = "expired_token_12345"
            self.session.cookies.set('ai_infra_token', expired_token)
            
            # 访问Gitea触发auth验证
            gitea_resp = self.session.get(f"{self.base_url}/gitea/", allow_redirects=False)
            
            # 检查是否有清除cookie的响应头
            set_cookie_header = gitea_resp.headers.get('Set-Cookie', '')
            debug_action = gitea_resp.headers.get('X-Debug-SSO-Action', '')
            
            # 检查是否清除了过期token
            cookie_cleared = 'ai_infra_token=;' in set_cookie_header or 'Max-Age=0' in set_cookie_header
            
            if cookie_cleared or 'expired' in debug_action:
                self.print_result("Token过期处理", True, f"正确清除过期token，动作: {debug_action}")
                return True
            else:
                self.print_result("Token过期处理", False, f"未正确处理过期token，动作: {debug_action}")
        
        except Exception as e:
            self.print_result("Token过期处理", False, f"异常: {str(e)}")
        
        return False
    
    def test_multi_user_sync(self):
        """测试多用户登录状态同步"""
        print("👥 测试多用户登录状态同步...")
        
        # 创建两个不同的session模拟不同用户
        admin_session = requests.Session()
        user_session = requests.Session()
        
        try:
            # Admin用户登录
            admin_login = admin_session.post(f"{self.base_url}/api/auth/login", json={
                "username": "admin",
                "password": "admin123"
            })
            
            if admin_login.status_code == 200:
                print("   ✅ Admin用户登录成功")
                
                # 测试admin访问gitea
                admin_gitea = admin_session.get(f"{self.base_url}/gitea/", allow_redirects=False)
                admin_location = admin_gitea.headers.get('Location', '')
                
                if '/gitea/' in admin_location:
                    self.print_result("多用户同步 - Admin", True, f"Admin重定向: {admin_location}")
                    admin_success = True
                else:
                    self.print_result("多用户同步 - Admin", False, f"Admin重定向异常: {admin_location}")
                    admin_success = False
                
                # 如果有其他用户，也可以测试
                # 这里暂时只测试admin用户的状态管理
                return admin_success
            else:
                self.print_result("多用户同步", False, f"Admin登录失败: {admin_login.status_code}")
        
        except Exception as e:
            self.print_result("多用户同步", False, f"异常: {str(e)}")
        
        return False
    
    def run_all_tests(self):
        """运行所有测试"""
        print("🚀 开始增强SSO功能测试")
        print("=" * 50)
        
        # 检查服务状态
        if not self.check_service_status():
            print("❌ 服务状态检查失败，跳过测试")
            return False
        
        print()
        
        results = []
        
        # 测试用户重定向
        results.append(self.test_user_redirection())
        
        # 测试后端退出处理
        results.append(self.test_backend_exit_handling())
        
        # 测试Token过期处理
        results.append(self.test_token_expiry_handling())
        
        # 测试多用户同步
        results.append(self.test_multi_user_sync())
        
        # 总结
        print("=" * 50)
        passed = sum(results)
        total = len(results)
        
        print(f"📊 测试总结: {passed}/{total} 测试通过")
        
        if passed == total:
            print("🎉 所有增强功能测试通过！")
            return True
        else:
            print("⚠️ 部分测试失败，需要进一步调试")
            return False

def main():
    tester = EnhancedSSOTester()
    success = tester.run_all_tests()
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
