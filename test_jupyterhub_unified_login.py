#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
JupyterHub统一后端登录测试脚本
测试单点登录功能，验证用户会话管理和状态持久化
"""

import requests
import time
import json
import sys
from urllib.parse import urljoin, urlparse, parse_qs
import logging

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class JupyterHubLoginTester:
    def __init__(self, base_url="http://localhost:8080", 
                 username="admin", password="password"):
        self.base_url = base_url
        self.username = username
        self.password = password
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'JupyterHub-Login-Tester/1.0'
        })
        
    def test_basic_connectivity(self):
        """测试基本连接性"""
        logger.info("🔍 测试基本连接性...")
        try:
            response = self.session.get(f"{self.base_url}/jupyter/")
            logger.info(f"✅ 基本连接成功: {response.status_code}")
            return True
        except Exception as e:
            logger.error(f"❌ 基本连接失败: {e}")
            return False
    
    def get_login_page(self):
        """获取登录页面并提取XSRF token"""
        logger.info("📄 获取登录页面...")
        
        login_url = f"{self.base_url}/jupyter/hub/login"
        response = self.session.get(login_url)
        
        if response.status_code != 200:
            logger.error(f"❌ 登录页面获取失败: {response.status_code}")
            return None, None
            
        # 提取XSRF token
        content = response.text
        xsrf_token = None
        
        # 查找XSRF token的几种可能位置
        if '_xsrf' in content:
            import re
            # 查找隐藏input中的token
            match = re.search(r'name="_xsrf"[^>]*value="([^"]*)"', content)
            if match:
                xsrf_token = match.group(1)
            else:
                # 查找JavaScript中的token
                match = re.search(r'_xsrf["\']?\s*:\s*["\']([^"\']*)["\']', content)
                if match:
                    xsrf_token = match.group(1)
        
        # 从cookies中获取xsrf token
        if not xsrf_token:
            for cookie in self.session.cookies:
                if 'xsrf' in cookie.name.lower():
                    xsrf_token = cookie.value
                    break
        
        logger.info(f"✅ 登录页面获取成功, XSRF Token: {xsrf_token[:20] + '...' if xsrf_token else 'Not Found'}")
        return response, xsrf_token
    
    def perform_login(self, xsrf_token):
        """执行登录操作"""
        logger.info(f"🔐 开始登录用户: {self.username}")
        
        login_url = f"{self.base_url}/jupyter/hub/login"
        
        # 登录数据
        login_data = {
            'username': self.username,
            'password': self.password
        }
        
        # 如果有XSRF token，添加到数据中
        if xsrf_token:
            login_data['_xsrf'] = xsrf_token
        
        # 执行登录POST请求
        response = self.session.post(
            login_url,
            data=login_data,
            allow_redirects=False  # 手动处理重定向以观察登录流程
        )
        
        logger.info(f"📊 登录响应状态: {response.status_code}")
        
        if response.status_code in [302, 303]:
            redirect_url = response.headers.get('Location', '')
            logger.info(f"🔄 登录重定向到: {redirect_url}")
            return True, redirect_url
        elif response.status_code == 200:
            if 'error' in response.text.lower() or 'invalid' in response.text.lower():
                logger.error("❌ 登录失败: 页面包含错误信息")
                return False, None
            else:
                logger.info("✅ 登录成功: 返回200状态")
                return True, None
        else:
            logger.error(f"❌ 登录失败: 状态码 {response.status_code}")
            return False, None
    
    def verify_authenticated_access(self):
        """验证认证后的访问"""
        logger.info("🔍 验证认证状态...")
        
        # 测试访问hub主页
        hub_url = f"{self.base_url}/jupyter/hub/home"
        response = self.session.get(hub_url)
        
        if response.status_code == 200:
            if self.username in response.text:
                logger.info(f"✅ 认证验证成功: 用户 {self.username} 已登录")
                return True
            else:
                logger.warning("⚠️  认证状态未知: 无法在页面中找到用户名")
        
        # 测试访问用户API
        api_url = f"{self.base_url}/jupyter/hub/api/user"
        response = self.session.get(api_url)
        
        if response.status_code == 200:
            try:
                user_info = response.json()
                logger.info(f"✅ API认证成功: {user_info.get('name', 'Unknown')}")
                return True
            except:
                pass
        
        logger.error("❌ 认证验证失败")
        return False
    
    def test_spawn_server(self):
        """测试服务器启动"""
        logger.info("🚀 测试JupyterLab服务器启动...")
        
        spawn_url = f"{self.base_url}/jupyter/hub/spawn"
        response = self.session.get(spawn_url)
        
        if response.status_code in [200, 302]:
            logger.info("✅ 服务器启动请求成功")
            
            # 检查是否重定向到实际的notebook服务器
            if response.status_code == 302:
                redirect_url = response.headers.get('Location', '')
                if '/user/' in redirect_url:
                    logger.info(f"🎯 重定向到用户服务器: {redirect_url}")
                    return True
            
            return True
        else:
            logger.error(f"❌ 服务器启动失败: {response.status_code}")
            return False
    
    def test_notebook_access(self):
        """测试notebook访问"""
        logger.info("📓 测试JupyterLab访问...")
        
        # 尝试访问用户的JupyterLab
        lab_url = f"{self.base_url}/jupyter/user/{self.username}/lab"
        response = self.session.get(lab_url, timeout=30)
        
        if response.status_code == 200:
            if 'JupyterLab' in response.text or 'jupyter-lab' in response.text:
                logger.info("✅ JupyterLab访问成功")
                return True
            else:
                logger.warning("⚠️  JupyterLab可能未完全加载")
                return True
        elif response.status_code == 302:
            logger.info("🔄 JupyterLab重定向（可能在启动中）")
            return True
        else:
            logger.error(f"❌ JupyterLab访问失败: {response.status_code}")
            return False
    
    def test_session_persistence(self):
        """测试会话持久化"""
        logger.info("💾 测试会话持久化...")
        
        # 创建新的session但使用相同的cookies
        new_session = requests.Session()
        new_session.cookies.update(self.session.cookies)
        
        # 尝试访问需要认证的页面
        hub_url = f"{self.base_url}/jupyter/hub/home"
        response = new_session.get(hub_url)
        
        if response.status_code == 200 and self.username in response.text:
            logger.info("✅ 会话持久化成功: 无需重新登录")
            return True
        else:
            logger.error("❌ 会话持久化失败")
            return False
    
    def test_admin_access(self):
        """测试管理员访问（仅限admin用户）"""
        if self.username != 'admin':
            logger.info("⏭️  跳过管理员测试（非admin用户）")
            return True
            
        logger.info("👑 测试管理员权限...")
        
        admin_url = f"{self.base_url}/jupyter/hub/admin"
        response = self.session.get(admin_url)
        
        if response.status_code == 200:
            if 'admin' in response.text.lower():
                logger.info("✅ 管理员权限验证成功")
                return True
            else:
                logger.warning("⚠️  管理员页面访问成功但内容可能不完整")
                return True
        else:
            logger.error(f"❌ 管理员权限验证失败: {response.status_code}")
            return False
    
    def logout(self):
        """测试登出"""
        logger.info("🚪 测试登出...")
        
        logout_url = f"{self.base_url}/jupyter/hub/logout"
        response = self.session.get(logout_url)
        
        if response.status_code in [200, 302]:
            logger.info("✅ 登出成功")
            return True
        else:
            logger.error(f"❌ 登出失败: {response.status_code}")
            return False
    
    def run_full_test(self):
        """运行完整的登录测试流程"""
        logger.info("🎯 开始JupyterHub统一后端登录完整测试")
        logger.info("="*60)
        
        results = {}
        
        # 1. 基本连接测试
        results['connectivity'] = self.test_basic_connectivity()
        if not results['connectivity']:
            logger.error("💥 基本连接失败，终止测试")
            return results
        
        # 2. 获取登录页面
        login_page, xsrf_token = self.get_login_page()
        results['login_page'] = login_page is not None
        
        # 3. 执行登录
        results['login'], redirect_url = self.perform_login(xsrf_token)
        if not results['login']:
            logger.error("💥 登录失败，终止后续测试")
            return results
        
        # 等待一下让会话建立
        time.sleep(2)
        
        # 4. 验证认证状态
        results['authentication'] = self.verify_authenticated_access()
        
        # 5. 测试服务器启动
        results['server_spawn'] = self.test_spawn_server()
        
        # 等待服务器启动
        if results['server_spawn']:
            logger.info("⏳ 等待JupyterLab服务器启动...")
            time.sleep(5)
        
        # 6. 测试notebook访问
        results['notebook_access'] = self.test_notebook_access()
        
        # 7. 测试会话持久化
        results['session_persistence'] = self.test_session_persistence()
        
        # 8. 测试管理员权限（如果是admin用户）
        results['admin_access'] = self.test_admin_access()
        
        # 9. 测试登出
        results['logout'] = self.logout()
        
        # 生成测试报告
        self.generate_report(results)
        
        return results
    
    def generate_report(self, results):
        """生成测试报告"""
        logger.info("\n" + "="*60)
        logger.info("📊 JupyterHub统一后端测试报告")
        logger.info("="*60)
        
        total_tests = len(results)
        passed_tests = sum(1 for result in results.values() if result)
        
        for test_name, result in results.items():
            status = "✅ PASS" if result else "❌ FAIL"
            logger.info(f"{test_name.replace('_', ' ').title():<25} {status}")
        
        logger.info("-"*60)
        logger.info(f"总测试数: {total_tests}")
        logger.info(f"通过测试: {passed_tests}")
        logger.info(f"失败测试: {total_tests - passed_tests}")
        logger.info(f"成功率: {(passed_tests/total_tests)*100:.1f}%")
        
        if passed_tests == total_tests:
            logger.info("🎉 所有测试通过！JupyterHub统一后端工作正常")
        else:
            logger.warning("⚠️  部分测试失败，请检查系统配置")
        
        logger.info("="*60)

def main():
    """主函数"""
    import argparse
    
    parser = argparse.ArgumentParser(description='JupyterHub统一后端登录测试')
    parser.add_argument('--url', default='http://localhost:8080', 
                       help='JupyterHub URL (默认: http://localhost:8080)')
    parser.add_argument('--username', default='admin', 
                       help='测试用户名 (默认: admin)')
    parser.add_argument('--password', default='password', 
                       help='测试密码 (默认: password)')
    parser.add_argument('--test-both-users', action='store_true',
                       help='测试admin和testuser两个用户')
    
    args = parser.parse_args()
    
    if args.test_both_users:
        # 测试两个用户
        users = [
            ('admin', 'password'),
            ('testuser', 'password')
        ]
        
        all_results = {}
        for username, password in users:
            logger.info(f"\n🧪 测试用户: {username}")
            logger.info("="*40)
            
            tester = JupyterHubLoginTester(
                base_url=args.url, 
                username=username, 
                password=password
            )
            
            results = tester.run_full_test()
            all_results[username] = results
            
            time.sleep(3)  # 用户间测试间隔
        
        # 汇总报告
        logger.info("\n" + "="*60)
        logger.info("📈 多用户测试汇总报告")
        logger.info("="*60)
        
        for username, results in all_results.items():
            passed = sum(1 for r in results.values() if r)
            total = len(results)
            logger.info(f"{username:<15} {passed}/{total} 通过 ({(passed/total)*100:.1f}%)")
        
    else:
        # 测试单个用户
        tester = JupyterHubLoginTester(
            base_url=args.url, 
            username=args.username, 
            password=args.password
        )
        
        results = tester.run_full_test()
        
        # 返回适当的退出码
        if all(results.values()):
            sys.exit(0)
        else:
            sys.exit(1)

if __name__ == "__main__":
    main()
