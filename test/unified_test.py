#!/usr/bin/env python3
"""
AI Infra Matrix 统一测试框架
整合所有SSO、Gitea、JupyterHub相关测试
"""

import sys
import argparse
import time
from datetime import datetime
from typing import Dict, List, Optional
import requests
import re
import json

class TestLogger:
    """统一的测试日志输出"""
    
    def __init__(self, verbose: bool = False):
        self.verbose = verbose
        self.start_time = time.time()
        
    def log(self, message: str, level: str = "INFO"):
        timestamp = datetime.now().strftime('%H:%M:%S')
        level_icons = {
            "INFO": "ℹ️",
            "SUCCESS": "✅", 
            "ERROR": "❌",
            "WARNING": "⚠️",
            "DEBUG": "🔍"
        }
        icon = level_icons.get(level, "📋")
        print(f"[{timestamp}] {icon} {message}")
        
    def success(self, message: str):
        self.log(message, "SUCCESS")
        
    def error(self, message: str):
        self.log(message, "ERROR")
        
    def warning(self, message: str):
        self.log(message, "WARNING")
        
    def debug(self, message: str):
        if self.verbose:
            self.log(message, "DEBUG")

class AuthClient:
    """统一的认证客户端"""
    
    def __init__(self, base_url: str = "http://localhost:8080", logger: Optional[TestLogger] = None):
        self.base_url = base_url
        self.logger = logger or TestLogger()
        self.session = requests.Session()
        self.token = None
        
    def login(self, username: str = "admin", password: str = "admin123") -> bool:
        """执行登录并获取token"""
        self.logger.log(f"🔐 尝试登录用户: {username}")
        
        try:
            login_data = {
                "username": username,
                "password": password
            }
            
            response = self.session.post(
                f"{self.base_url}/api/auth/login",
                json=login_data,
                headers={"Content-Type": "application/json"}
            )
            
            self.logger.debug(f"登录响应状态码: {response.status_code}")
            
            if response.status_code == 200:
                result = response.json()
                self.token = result.get("token")
                if self.token:
                    self.session.headers.update({
                        'Authorization': f'Bearer {self.token}'
                    })
                    # 设置cookie
                    self.session.cookies.set('ai_infra_token', self.token)
                    self.logger.success(f"登录成功，Token: {self.token[:20]}...")
                    return True
                else:
                    self.logger.error("登录响应中没有token")
                    return False
            else:
                self.logger.error(f"登录失败: {response.status_code} - {response.text}")
                return False
                
        except Exception as e:
            self.logger.error(f"登录异常: {e}")
            return False
            
    def verify_token(self) -> bool:
        """验证当前token是否有效"""
        if not self.token:
            return False
            
        try:
            response = self.session.get(f"{self.base_url}/api/auth/verify")
            return response.status_code == 200
        except:
            return False

class SSOTester:
    """SSO功能测试器"""
    
    def __init__(self, auth_client: AuthClient, logger: TestLogger):
        self.auth = auth_client
        self.logger = logger
        
    def test_gitea_login_redirect(self) -> bool:
        """测试Gitea登录页面的SSO重定向"""
        self.logger.log("🧪 测试Gitea登录SSO重定向")
        
        try:
            # 测试登录页面访问
            response = self.auth.session.get(
                f"{self.auth.base_url}/gitea/user/login?redirect_to=%2Fgitea%2Fadmin",
                allow_redirects=False
            )
            
            self.logger.debug(f"Gitea登录页面响应: {response.status_code}")
            self.logger.debug(f"响应头: {dict(response.headers)}")
            
            # 检查是否是重定向响应
            if response.status_code in [302, 303]:
                location = response.headers.get('Location', '')
                self.logger.success(f"SSO重定向成功: {location}")
                return True
            elif response.status_code == 200:
                # 检查是否包含登录表单
                if 'password' in response.text.lower():
                    self.logger.warning("显示登录表单（可能是token无效）")
                    return False
                else:
                    self.logger.success("直接访问成功（可能已认证）")
                    return True
            else:
                self.logger.error(f"意外的响应状态: {response.status_code}")
                return False
                
        except Exception as e:
            self.logger.error(f"测试Gitea SSO异常: {e}")
            return False
            
    def test_gitea_access_without_token(self) -> bool:
        """测试没有token时的Gitea访问"""
        self.logger.log("🧪 测试无token的Gitea访问")
        
        try:
            # 创建新的session（没有认证）
            clean_session = requests.Session()
            response = clean_session.get(
                f"{self.auth.base_url}/gitea/user/login?redirect_to=%2Fgitea%2Fadmin"
            )
            
            self.logger.debug(f"无token访问响应: {response.status_code}")
            
            if response.status_code == 200:
                # 应该显示登录表单
                if 'password' in response.text.lower() or 'login' in response.text.lower():
                    self.logger.success("正确显示登录表单")
                    return True
                else:
                    self.logger.warning("没有显示预期的登录表单")
                    return False
            else:
                self.logger.error(f"意外的响应状态: {response.status_code}")
                return False
                
        except Exception as e:
            self.logger.error(f"测试无token Gitea访问异常: {e}")
            return False

class JupyterHubTester:
    """JupyterHub功能测试器"""
    
    def __init__(self, auth_client: AuthClient, logger: TestLogger):
        self.auth = auth_client
        self.logger = logger
        
    def test_jupyterhub_access(self) -> bool:
        """测试JupyterHub访问"""
        self.logger.log("🧪 测试JupyterHub访问")
        
        try:
            response = self.auth.session.get(f"{self.auth.base_url}/jupyter/")
            self.logger.debug(f"JupyterHub响应: {response.status_code}")
            
            if response.status_code == 200:
                self.logger.success("JupyterHub访问成功")
                return True
            elif response.status_code in [302, 303]:
                location = response.headers.get('Location', '')
                self.logger.success(f"JupyterHub重定向: {location}")
                return True
            else:
                self.logger.error(f"JupyterHub访问失败: {response.status_code}")
                return False
                
        except Exception as e:
            self.logger.error(f"测试JupyterHub访问异常: {e}")
            return False

class SystemHealthTester:
    """系统健康检查"""
    
    def __init__(self, base_url: str, logger: TestLogger):
        self.base_url = base_url
        self.logger = logger
        
    def test_services_health(self) -> Dict[str, bool]:
        """检查各服务健康状态"""
        self.logger.log("🏥 检查系统健康状态")
        
        services = {
            "frontend": "/",
            "backend": "/api/health",
            "gitea": "/gitea/",
            "jupyterhub": "/jupyter/hub/health"
        }
        
        results = {}
        
        for service_name, endpoint in services.items():
            try:
                response = requests.get(f"{self.base_url}{endpoint}", timeout=5)
                is_healthy = response.status_code in [200, 302, 403]  # 403也可能是正常的
                results[service_name] = is_healthy
                
                if is_healthy:
                    self.logger.success(f"{service_name}: 健康")
                else:
                    self.logger.error(f"{service_name}: 异常 ({response.status_code})")
                    
            except Exception as e:
                results[service_name] = False
                self.logger.error(f"{service_name}: 连接失败 - {e}")
                
        return results

class TestRunner:
    """主测试运行器"""
    
    def __init__(self, base_url: str = "http://localhost:8080", verbose: bool = False):
        self.base_url = base_url
        self.logger = TestLogger(verbose)
        self.auth_client = AuthClient(base_url, self.logger)
        self.sso_tester = SSOTester(self.auth_client, self.logger)
        self.jupyterhub_tester = JupyterHubTester(self.auth_client, self.logger)
        self.health_tester = SystemHealthTester(base_url, self.logger)
        
    def run_health_check(self) -> bool:
        """运行系统健康检查"""
        self.logger.log("🚀 开始系统健康检查")
        results = self.health_tester.test_services_health()
        
        all_healthy = all(results.values())
        if all_healthy:
            self.logger.success("所有服务健康")
        else:
            unhealthy = [name for name, healthy in results.items() if not healthy]
            self.logger.error(f"以下服务不健康: {', '.join(unhealthy)}")
            
        return all_healthy
        
    def run_auth_tests(self) -> bool:
        """运行认证相关测试"""
        self.logger.log("🚀 开始认证功能测试")
        
        # 1. 测试登录
        if not self.auth_client.login():
            self.logger.error("登录测试失败")
            return False
            
        # 2. 验证token
        if not self.auth_client.verify_token():
            self.logger.error("Token验证失败")
            return False
            
        self.logger.success("认证功能测试通过")
        return True
        
    def run_sso_tests(self) -> bool:
        """运行SSO功能测试"""
        self.logger.log("🚀 开始SSO功能测试")
        
        # 确保已登录
        if not self.auth_client.token:
            if not self.auth_client.login():
                self.logger.error("无法登录，跳过SSO测试")
                return False
                
        # 测试有token的Gitea访问
        gitea_with_token = self.sso_tester.test_gitea_login_redirect()
        
        # 测试无token的Gitea访问
        gitea_without_token = self.sso_tester.test_gitea_access_without_token()
        
        all_passed = gitea_with_token and gitea_without_token
        
        if all_passed:
            self.logger.success("SSO功能测试通过")
        else:
            self.logger.error("SSO功能测试失败")
            
        return all_passed
        
    def run_jupyterhub_tests(self) -> bool:
        """运行JupyterHub测试"""
        self.logger.log("🚀 开始JupyterHub功能测试")
        
        # 确保已登录
        if not self.auth_client.token:
            if not self.auth_client.login():
                self.logger.error("无法登录，跳过JupyterHub测试")
                return False
                
        result = self.jupyterhub_tester.test_jupyterhub_access()
        
        if result:
            self.logger.success("JupyterHub功能测试通过")
        else:
            self.logger.error("JupyterHub功能测试失败")
            
        return result
        
    def run_original_problem_test(self) -> bool:
        """运行原始问题验证测试"""
        self.logger.log("🚀 验证原始问题已解决")
        
        # 登录
        if not self.auth_client.login():
            self.logger.error("无法登录进行原始问题测试")
            return False
            
        self.logger.log("验证：已登录用户访问Gitea登录页面应该直接重定向")
        
        try:
            response = self.auth_client.session.get(
                f"{self.base_url}/gitea/user/login?redirect_to=%2Fgitea%2Fadmin",
                allow_redirects=False
            )
            
            if response.status_code in [302, 303]:
                location = response.headers.get('Location', '')
                if '/gitea/' in location:
                    self.logger.success("✅ 原始问题已解决：已登录用户无需二次密码输入")
                    return True
                else:
                    self.logger.error(f"重定向位置错误: {location}")
                    return False
            elif response.status_code == 200 and 'password' not in response.text.lower():
                self.logger.success("✅ 原始问题已解决：已登录用户直接访问成功")
                return True
            else:
                self.logger.error("❌ 原始问题未解决：仍需要密码输入")
                return False
                
        except Exception as e:
            self.logger.error(f"原始问题测试异常: {e}")
            return False
            
    def run_all_tests(self) -> bool:
        """运行所有测试"""
        self.logger.log("🎯 开始完整测试套件")
        
        test_results = {}
        
        # 1. 健康检查
        test_results['health'] = self.run_health_check()
        
        # 2. 认证测试
        test_results['auth'] = self.run_auth_tests()
        
        # 3. SSO测试
        test_results['sso'] = self.run_sso_tests()
        
        # 4. JupyterHub测试
        test_results['jupyterhub'] = self.run_jupyterhub_tests()
        
        # 5. 原始问题验证
        test_results['original_problem'] = self.run_original_problem_test()
        
        # 汇总结果
        self.logger.log("📊 测试结果汇总:")
        passed_count = 0
        total_count = len(test_results)
        
        for test_name, result in test_results.items():
            status = "✅ 通过" if result else "❌ 失败"
            self.logger.log(f"   {test_name}: {status}")
            if result:
                passed_count += 1
                
        self.logger.log(f"📈 总体结果: {passed_count}/{total_count} 通过")
        
        all_passed = passed_count == total_count
        if all_passed:
            self.logger.success("🎉 所有测试通过！")
        else:
            self.logger.error("❌ 部分测试失败")
            
        return all_passed

def main():
    """主函数"""
    parser = argparse.ArgumentParser(description="AI Infra Matrix 统一测试框架")
    parser.add_argument("--url", default="http://localhost:8080", help="基础URL")
    parser.add_argument("--verbose", "-v", action="store_true", help="详细输出")
    parser.add_argument("--test", choices=["health", "auth", "sso", "jupyterhub", "original", "all"], 
                       default="all", help="要运行的测试类型")
    
    args = parser.parse_args()
    
    runner = TestRunner(args.url, args.verbose)
    
    if args.test == "health":
        success = runner.run_health_check()
    elif args.test == "auth":
        success = runner.run_auth_tests()
    elif args.test == "sso":
        success = runner.run_sso_tests()
    elif args.test == "jupyterhub":
        success = runner.run_jupyterhub_tests()
    elif args.test == "original":
        success = runner.run_original_problem_test()
    else:  # all
        success = runner.run_all_tests()
    
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
