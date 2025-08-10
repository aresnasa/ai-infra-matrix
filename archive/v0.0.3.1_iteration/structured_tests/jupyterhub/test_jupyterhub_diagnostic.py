#!/usr/bin/env python3
"""
简化版JupyterHub iframe诊断工具
使用requests模拟浏览器行为，不依赖Selenium
"""

import requests
import json
import time
from urllib.parse import urljoin, urlparse
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class JupyterHubDiagnostic:
    def __init__(self, base_url="http://localhost:8080"):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36'
        })
        self.results = {}
    
    def test_main_page(self):
        """测试主页面"""
        logger.info("🧪 测试主页面访问")
        try:
            response = self.session.get(self.base_url)
            logger.info(f"✅ 主页状态码: {response.status_code}")
            self.results['main_page'] = {
                'status_code': response.status_code,
                'content_length': len(response.content)
            }
        except Exception as e:
            logger.error(f"❌ 主页访问失败: {e}")
            self.results['main_page'] = {'error': str(e)}
    
    def test_api_health(self):
        """测试API健康检查"""
        logger.info("🧪 测试API健康检查")
        try:
            response = self.session.get(f"{self.base_url}/api/health")
            logger.info(f"✅ API健康检查状态码: {response.status_code}")
            self.results['api_health'] = {
                'status_code': response.status_code,
                'response': response.text[:200]
            }
        except Exception as e:
            logger.error(f"❌ API健康检查失败: {e}")
            self.results['api_health'] = {'error': str(e)}
    
    def test_auth_login(self):
        """测试认证登录"""
        logger.info("🧪 测试认证API")
        try:
            auth_data = {
                'username': 'admin',
                'password': 'admin123'
            }
            response = self.session.post(
                f"{self.base_url}/api/auth/login",
                json=auth_data,
                headers={'Content-Type': 'application/json'}
            )
            logger.info(f"✅ 认证API状态码: {response.status_code}")
            
            if response.status_code == 200:
                try:
                    data = response.json()
                    has_token = 'token' in data
                    logger.info(f"🎫 是否获得token: {has_token}")
                    self.results['auth_login'] = {
                        'status_code': response.status_code,
                        'has_token': has_token,
                        'token_length': len(data.get('token', '')) if has_token else 0
                    }
                    return data.get('token')
                except json.JSONDecodeError:
                    logger.error("❌ 认证响应不是有效JSON")
                    self.results['auth_login'] = {
                        'status_code': response.status_code,
                        'error': 'Invalid JSON response'
                    }
            else:
                logger.error(f"❌ 认证失败: {response.status_code} - {response.text}")
                self.results['auth_login'] = {
                    'status_code': response.status_code,
                    'error': response.text
                }
        except Exception as e:
            logger.error(f"❌ 认证API测试失败: {e}")
            self.results['auth_login'] = {'error': str(e)}
        return None
    
    def test_jupyterhub_wrapper(self):
        """测试JupyterHub wrapper页面"""
        logger.info("🧪 测试JupyterHub wrapper页面")
        try:
            response = self.session.get(f"{self.base_url}/jupyterhub")
            logger.info(f"✅ Wrapper页面状态码: {response.status_code}")
            
            content = response.text
            has_iframe = 'jupyterhub-frame' in content
            has_auth_script = 'getAuthToken' in content
            has_error_div = 'id="error"' in content
            
            logger.info(f"🖼️ 包含iframe: {has_iframe}")
            logger.info(f"🔑 包含认证脚本: {has_auth_script}")
            logger.info(f"⚠️ 包含错误div: {has_error_div}")
            
            self.results['jupyterhub_wrapper'] = {
                'status_code': response.status_code,
                'content_length': len(content),
                'has_iframe': has_iframe,
                'has_auth_script': has_auth_script,
                'has_error_div': has_error_div
            }
            
        except Exception as e:
            logger.error(f"❌ JupyterHub wrapper测试失败: {e}")
            self.results['jupyterhub_wrapper'] = {'error': str(e)}
    
    def test_jupyterhub_direct(self, token=None):
        """测试直接访问JupyterHub"""
        logger.info("🧪 测试直接JupyterHub访问")
        try:
            # 测试不带token的访问
            response = self.session.get(f"{self.base_url}/jupyter/hub/")
            logger.info(f"✅ JupyterHub直接访问状态码: {response.status_code}")
            
            # 检查是否重定向到登录页面
            is_login_page = 'login' in response.url.lower() or 'login' in response.text.lower()
            logger.info(f"🔐 是否为登录页面: {is_login_page}")
            
            result = {
                'status_code': response.status_code,
                'final_url': response.url,
                'is_login_page': is_login_page,
                'content_length': len(response.content)
            }
            
            # 如果有token，测试带token的访问
            if token:
                logger.info("🧪 测试带token的JupyterHub访问")
                token_url = f"{self.base_url}/jupyter/hub/?token={token}"
                token_response = self.session.get(token_url)
                logger.info(f"✅ 带token访问状态码: {token_response.status_code}")
                
                result['token_access'] = {
                    'status_code': token_response.status_code,
                    'final_url': token_response.url,
                    'content_length': len(token_response.content)
                }
            
            self.results['jupyterhub_direct'] = result
            
        except Exception as e:
            logger.error(f"❌ JupyterHub直接访问测试失败: {e}")
            self.results['jupyterhub_direct'] = {'error': str(e)}
    
    def test_static_files(self):
        """测试静态文件访问"""
        logger.info("🧪 测试静态文件访问")
        
        static_files = [
            "/jupyterhub/iframe_test.html",
            "/jupyterhub/jupyterhub_wrapper.html"
        ]
        
        results = {}
        for file_path in static_files:
            try:
                response = self.session.get(f"{self.base_url}{file_path}")
                results[file_path] = {
                    'status_code': response.status_code,
                    'content_length': len(response.content)
                }
                logger.info(f"✅ {file_path}: {response.status_code}")
            except Exception as e:
                results[file_path] = {'error': str(e)}
                logger.error(f"❌ {file_path}: {e}")
        
        self.results['static_files'] = results
    
    def run_diagnostic(self):
        """运行完整诊断"""
        logger.info("🚀 开始JupyterHub iframe诊断")
        logger.info("=" * 60)
        
        # 运行所有测试
        self.test_main_page()
        time.sleep(1)
        
        self.test_api_health()
        time.sleep(1)
        
        token = self.test_auth_login()
        time.sleep(1)
        
        self.test_jupyterhub_wrapper()
        time.sleep(1)
        
        self.test_jupyterhub_direct(token)
        time.sleep(1)
        
        self.test_static_files()
        
        # 生成报告
        self.generate_report()
        
        # 提供建议
        self.provide_suggestions()
    
    def generate_report(self):
        """生成测试报告"""
        logger.info("=" * 60)
        logger.info("📊 诊断报告")
        logger.info("=" * 60)
        
        for test_name, result in self.results.items():
            if 'error' in result:
                logger.error(f"❌ {test_name}: {result['error']}")
            else:
                status_code = result.get('status_code', 'N/A')
                logger.info(f"✅ {test_name}: HTTP {status_code}")
        
        # 保存详细报告
        report_file = f"diagnostic_report_{int(time.time())}.json"
        with open(report_file, 'w') as f:
            json.dump(self.results, f, indent=2)
        logger.info(f"📄 详细报告已保存到: {report_file}")
    
    def provide_suggestions(self):
        """提供修复建议"""
        logger.info("=" * 60)
        logger.info("🔧 修复建议")
        logger.info("=" * 60)
        
        # 检查认证问题
        auth_result = self.results.get('auth_login', {})
        if not auth_result.get('has_token'):
            logger.warning("🔑 认证问题: 无法获取JWT token")
            logger.info("   建议: 检查后端服务是否正常运行")
            logger.info("   命令: docker-compose logs backend")
        
        # 检查JupyterHub服务
        jupyter_result = self.results.get('jupyterhub_direct', {})
        if jupyter_result.get('status_code') not in [200, 302]:
            logger.warning("🔧 JupyterHub服务问题")
            logger.info("   建议: 检查JupyterHub容器状态")
            logger.info("   命令: docker-compose logs jupyterhub")
        
        # 检查wrapper页面
        wrapper_result = self.results.get('jupyterhub_wrapper', {})
        if not wrapper_result.get('has_auth_script'):
            logger.warning("📝 Wrapper页面缺少认证脚本")
            logger.info("   建议: 检查HTML文件是否正确部署")
        
        # 检查nginx配置
        if self.results.get('main_page', {}).get('status_code') != 200:
            logger.warning("🌐 nginx代理问题")
            logger.info("   建议: 检查nginx配置和服务状态")
            logger.info("   命令: docker-compose logs nginx")

def main():
    diagnostic = JupyterHubDiagnostic()
    diagnostic.run_diagnostic()

if __name__ == "__main__":
    main()
