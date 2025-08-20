#!/usr/bin/env python3
"""
专门测试原始问题：已经登录了localhost:8080，
但是访问Gitea登录页面还需要二次输入密            # 检查是否显示密码输入框
            has_password_field = bool(re.search(r'<input[^>]*type=[\"\']password[\"\']', response.text, re.IGNORECASE))
            has_login_form = bool(re.search(r'<form[^>]*action[^>]*login', response.text, re.IGNORECASE))合预期
"""

import requests
import re
from datetime import datetime

class OriginalProblemTest:
    def __init__(self):
        self.base_url = "http://localhost:8080"
        self.session = requests.Session()
        
    def log(self, message):
        print(f"[{datetime.now().strftime('%H:%M:%S')}] {message}")
        
    def test_main_portal_login(self):
        """测试主门户登录"""
        self.log("🔐 测试主门户登录...")
        
        # 访问主页面
        response = self.session.get(self.base_url)
        self.log(f"主页面状态码: {response.status_code}")
        
        if "用户名" in response.text or "username" in response.text.lower():
            self.log("❌ 主页面显示登录表单，用户未登录")
            return False
        elif "管理员" in response.text or "admin" in response.text.lower():
            self.log("✅ 主页面显示管理内容，用户已登录")
            return True
        else:
            self.log("ℹ️ 主页面内容不明确，但没有登录表单")
            return True
            
    def simulate_login(self):
        """模拟用户登录过程"""
        self.log("🔄 模拟用户登录...")
        
        # 尝试使用admin/admin登录
        login_data = {
            "username": "admin",
            "password": "admin"
        }
        
        # 先获取登录页面
        login_page = self.session.get(f"{self.base_url}/login")
        self.log(f"登录页面状态码: {login_page.status_code}")
        
        # 提交登录表单
        login_response = self.session.post(f"{self.base_url}/login", data=login_data)
        self.log(f"登录提交状态码: {login_response.status_code}")
        
        # 检查是否登录成功
        dashboard_response = self.session.get(self.base_url)
        if dashboard_response.status_code == 200:
            if "管理员" in dashboard_response.text or "admin" in dashboard_response.text.lower():
                self.log("✅ 登录成功")
                return True
        
        self.log("⚠️ 登录状态不明确，继续测试")
        return True
        
    def test_gitea_login_redirect(self):
        """测试核心问题：访问Gitea登录页面是否需要二次密码输入"""
        self.log("🎯 测试核心问题：Gitea登录页面是否需要二次密码...")
        
        gitea_login_url = f"{self.base_url}/gitea/user/login?redirect_to=%2Fgitea%2Fadmin"
        
        try:
            response = self.session.get(gitea_login_url, allow_redirects=True)
            self.log(f"Gitea登录页面状态码: {response.status_code}")
            self.log(f"最终URL: {response.url}")
            
            # 检查响应内容
            content = response.text.lower()
            
            # 检查是否显示密码输入框
            has_password_field = bool(re.search(r'<input[^>]*type=[\"\']password[\"\']', response.text, re.IGNORECASE))
            has_login_form = bool(re.search(r'<form[^>]*action[^>]*login', response.text, re.IGNORECASE))
            
            if has_password_field or has_login_form:
                self.log("❌ 发现问题：Gitea页面仍然显示登录表单/密码输入框")
                self.log("   这意味着用户需要二次输入密码，不符合SSO预期")
                return False
            elif "/gitea/admin" in response.url or "管理员" in response.text:
                self.log("✅ 完美：直接重定向到Gitea管理页面，无需二次密码")
                return True
            elif response.status_code in [401, 403]:
                self.log("❌ 权限问题：收到401/403错误")
                return False
            else:
                self.log("ℹ️ 其他情况：需要进一步分析")
                # 显示一些内容片段用于分析
                if len(response.text) > 100:
                    preview = response.text[:200] + "..."
                    self.log(f"   页面内容预览: {preview}")
                return None
                
        except Exception as e:
            self.log(f"❌ 请求失败: {e}")
            return False
            
    def test_without_session(self):
        """测试没有session的情况（预期显示登录表单）"""
        self.log("🔍 测试无session访问Gitea登录页面...")
        
        # 创建新session（无登录状态）
        new_session = requests.Session()
        gitea_login_url = f"{self.base_url}/gitea/user/login?redirect_to=%2Fgitea%2Fadmin"
        
        try:
            response = new_session.get(gitea_login_url)
            self.log(f"无session状态码: {response.status_code}")
            
            has_password_field = bool(re.search(r'<input[^>]*type=[\"\']password[\"\']', response.text, re.IGNORECASE))
            
            if has_password_field:
                self.log("✅ 正确：无session时显示登录表单")
                return True
            else:
                self.log("⚠️ 无session时没有显示登录表单")
                return False
                
        except Exception as e:
            self.log(f"❌ 无session测试失败: {e}")
            return False
            
    def run_complete_test(self):
        """运行完整测试"""
        self.log("=" * 60)
        self.log("🚀 开始测试原始问题修复情况")
        self.log("问题描述：已经登录localhost:8080，访问gitea登录页面还需要二次密码")
        self.log("=" * 60)
        
        results = {}
        
        # 1. 检查主门户登录状态
        results['main_portal'] = self.test_main_portal_login()
        
        # 2. 如果需要，进行登录
        if not results['main_portal']:
            results['login_simulation'] = self.simulate_login()
        
        # 3. 测试核心问题
        results['gitea_sso'] = self.test_gitea_login_redirect()
        
        # 4. 测试对照组（无session）
        results['no_session'] = self.test_without_session()
        
        # 汇总结果
        self.log("=" * 60)
        self.log("📊 测试结果汇总:")
        
        if results['gitea_sso'] is True:
            self.log("✅ 原始问题已修复：用户无需二次输入密码")
            self.log("   SSO集成工作正常，用户体验符合预期")
        elif results['gitea_sso'] is False:
            self.log("❌ 原始问题仍然存在：用户仍需二次输入密码")
            self.log("   需要进一步检查SSO配置")
        else:
            self.log("⚠️ 测试结果不明确，需要手动验证")
            
        if results['no_session']:
            self.log("✅ 安全检查通过：无session时正确显示登录表单")
        else:
            self.log("⚠️ 安全检查：无session行为需要确认")
            
        self.log("=" * 60)
        return results

if __name__ == "__main__":
    tester = OriginalProblemTest()
    results = tester.run_complete_test()
