#!/usr/bin/env python3
"""
完整的JupyterHub登录测试
模拟真实用户登录流程：获取登录页面 -> 提交用户名密码 -> 使用认证cookie访问
"""
import requests
import re
import sys
from urllib.parse import urljoin, urlparse

class JupyterHubLoginTester:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36'
        })
    
    def test_redirects(self):
        """测试重定向路径"""
        print("🔄 测试重定向路径...")
        redirects = [
            f"{self.base_url}/jupyter",
            f"{self.base_url}/jupyter/",
            f"{self.base_url}/jupyter/hub/login"
        ]
        
        for url in redirects:
            try:
                resp = self.session.get(url, allow_redirects=False)
                print(f"   {url} -> {resp.status_code}")
                if 'Location' in resp.headers:
                    print(f"      重定向到: {resp.headers['Location']}")
            except Exception as e:
                print(f"   ❌ {url} 错误: {e}")
        print()
    
    def get_login_page(self):
        """获取登录页面和XSRF token"""
        print("📄 获取登录页面...")
        login_url = f"{self.base_url}/jupyter/hub/login"
        
        try:
            resp = self.session.get(login_url)
            if resp.status_code != 200:
                print(f"   ❌ 登录页面返回状态码: {resp.status_code}")
                return None, None
            
            print(f"   ✅ 登录页面获取成功 ({len(resp.text)} 字节)")
            
            # 提取XSRF token
            xsrf_match = re.search(r'name="_xsrf".*?value="([^"]*)"', resp.text)
            if xsrf_match:
                xsrf_token = xsrf_match.group(1)
                print(f"   ✅ XSRF token: {xsrf_token[:20]}...")
                return resp.text, xsrf_token
            else:
                print("   ❌ 未找到XSRF token")
                return resp.text, None
                
        except Exception as e:
            print(f"   ❌ 获取登录页面失败: {e}")
            return None, None
    
    def attempt_login(self, username, password, xsrf_token):
        """尝试登录"""
        print(f"🔐 尝试登录 (用户名: {username})...")
        login_url = f"{self.base_url}/jupyter/hub/login"
        
        # 准备登录数据
        login_data = {
            'username': username,
            'password': password,
            '_xsrf': xsrf_token
        }
        
        try:
            # 提交登录表单
            resp = self.session.post(login_url, data=login_data, allow_redirects=False)
            print(f"   登录响应状态码: {resp.status_code}")
            
            if resp.status_code == 302:
                # 登录成功，检查重定向
                redirect_url = resp.headers.get('Location', '')
                print(f"   ✅ 登录成功，重定向到: {redirect_url}")
                
                # 检查是否有认证cookie
                auth_cookies = []
                for cookie in self.session.cookies:
                    if any(keyword in cookie.name.lower() for keyword in ['hub', 'auth', 'session', 'jupyterhub']):
                        auth_cookies.append(f"{cookie.name}={cookie.value[:20]}...")
                
                if auth_cookies:
                    print(f"   ✅ 获得认证cookie: {auth_cookies}")
                else:
                    print("   ⚠️  未检测到明显的认证cookie")
                
                return True, redirect_url
            elif resp.status_code == 200:
                # 可能是登录失败，返回登录页面
                if "incorrect" in resp.text.lower() or "invalid" in resp.text.lower():
                    print("   ❌ 登录失败：用户名或密码错误")
                else:
                    print("   ❌ 登录失败：未知原因")
                return False, None
            else:
                print(f"   ❌ 登录失败：意外状态码 {resp.status_code}")
                return False, None
                
        except Exception as e:
            print(f"   ❌ 登录异常: {e}")
            return False, None
    
    def test_authenticated_access(self):
        """测试使用认证cookie访问JupyterHub"""
        print("🏠 测试认证后的访问...")
        
        # 测试不同的JupyterHub页面
        test_urls = [
            f"{self.base_url}/jupyter/hub/home",
            f"{self.base_url}/jupyter/hub/spawn",
            f"{self.base_url}/jupyter/user-redirect/",
            f"{self.base_url}/jupyter/hub/api/user"
        ]
        
        for url in test_urls:
            try:
                resp = self.session.get(url, allow_redirects=False)
                print(f"   {url}")
                print(f"      状态码: {resp.status_code}")
                
                if resp.status_code == 200:
                    print("      ✅ 访问成功")
                elif resp.status_code == 302:
                    redirect_to = resp.headers.get('Location', '')
                    if 'login' in redirect_to:
                        print("      ❌ 被重定向到登录页面（认证失败）")
                    else:
                        print(f"      ➡️  重定向到: {redirect_to}")
                elif resp.status_code == 403:
                    print("      ❌ 访问被拒绝")
                else:
                    print(f"      ⚠️  状态码: {resp.status_code}")
                    
            except Exception as e:
                print(f"      ❌ 访问异常: {e}")
        print()
    
    def run_full_test(self, username="testuser", password="password"):
        """运行完整的登录测试"""
        print("🧪 开始完整的JupyterHub登录测试...")
        print(f"   基础URL: {self.base_url}")
        print(f"   测试用户: {username}")
        print("=" * 60)
        
        # 1. 测试重定向
        self.test_redirects()
        
        # 2. 获取登录页面
        login_page, xsrf_token = self.get_login_page()
        if not xsrf_token:
            print("❌ 无法获取XSRF token，测试终止")
            return False
        
        print()
        
        # 3. 尝试登录
        login_success, redirect_url = self.attempt_login(username, password, xsrf_token)
        if not login_success:
            print("❌ 登录失败，测试终止")
            return False
        
        print()
        
        # 4. 测试认证后的访问
        self.test_authenticated_access()
        
        # 5. 显示所有cookie
        print("🍪 当前所有cookie:")
        for cookie in self.session.cookies:
            print(f"   {cookie.name}: {cookie.value[:50]}{'...' if len(cookie.value) > 50 else ''}")
            print(f"      域: {cookie.domain}, 路径: {cookie.path}")
        
        print("\n✅ 完整登录测试完成!")
        return True

def main():
    """主函数"""
    # 支持命令行参数
    username = sys.argv[1] if len(sys.argv) > 1 else "testuser"
    password = sys.argv[2] if len(sys.argv) > 2 else "password"
    
    tester = JupyterHubLoginTester()
    success = tester.run_full_test(username, password)
    
    if not success:
        sys.exit(1)

if __name__ == "__main__":
    main()
