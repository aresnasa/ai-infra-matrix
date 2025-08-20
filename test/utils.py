"""
测试工具模块
提供通用的测试功能
"""

import requests
import time
import re
from typing import Dict, Optional, Tuple
from datetime import datetime

class TestSession:
    """测试会话管理器"""
    
    def __init__(self, base_url: str):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        self.token = None
        
    def login(self, username: str, password: str) -> Tuple[bool, str]:
        """登录并获取token"""
        try:
            login_data = {"username": username, "password": password}
            response = self.session.post(
                f"{self.base_url}/api/auth/login",
                json=login_data,
                headers={"Content-Type": "application/json"}
            )
            
            if response.status_code == 200:
                result = response.json()
                self.token = result.get("token")
                if self.token:
                    self.session.headers.update({
                        'Authorization': f'Bearer {self.token}'
                    })
                    self.session.cookies.set('ai_infra_token', self.token)
                    return True, f"登录成功，Token: {self.token[:20]}..."
                else:
                    return False, "响应中没有token"
            else:
                return False, f"登录失败: {response.status_code} - {response.text}"
                
        except Exception as e:
            return False, f"登录异常: {e}"
            
    def get(self, path: str, **kwargs) -> requests.Response:
        """发送GET请求"""
        url = f"{self.base_url}{path}" if path.startswith('/') else f"{self.base_url}/{path}"
        return self.session.get(url, **kwargs)
        
    def post(self, path: str, **kwargs) -> requests.Response:
        """发送POST请求"""
        url = f"{self.base_url}{path}" if path.startswith('/') else f"{self.base_url}/{path}"
        return self.session.post(url, **kwargs)

class TestValidator:
    """测试验证器"""
    
    @staticmethod
    def is_redirect(response: requests.Response) -> bool:
        """检查是否是重定向响应"""
        return response.status_code in [301, 302, 303, 307, 308]
        
    @staticmethod
    def is_success(response: requests.Response) -> bool:
        """检查是否是成功响应"""
        return 200 <= response.status_code < 300
        
    @staticmethod
    def contains_login_form(html: str) -> bool:
        """检查HTML是否包含登录表单"""
        return bool(re.search(r'<input[^>]*type=[\"\']password[\"\']', html, re.IGNORECASE))
        
    @staticmethod
    def extract_redirect_location(response: requests.Response) -> Optional[str]:
        """提取重定向位置"""
        if TestValidator.is_redirect(response):
            return response.headers.get('Location')
        return None

class TestReporter:
    """测试报告器"""
    
    def __init__(self, verbose: bool = False):
        self.verbose = verbose
        self.start_time = time.time()
        self.test_results = []
        
    def log(self, message: str, level: str = "INFO"):
        """记录日志"""
        timestamp = datetime.now().strftime('%H:%M:%S')
        level_icons = {
            "INFO": "ℹ️",
            "SUCCESS": "✅", 
            "ERROR": "❌",
            "WARNING": "⚠️",
            "DEBUG": "🔍"
        }
        icon = level_icons.get(level, "📋")
        formatted_message = f"[{timestamp}] {icon} {message}"
        print(formatted_message)
        
        if self.verbose or level != "DEBUG":
            self.test_results.append({
                "timestamp": timestamp,
                "level": level,
                "message": message
            })
            
    def success(self, message: str):
        self.log(message, "SUCCESS")
        
    def error(self, message: str):
        self.log(message, "ERROR")
        
    def warning(self, message: str):
        self.log(message, "WARNING")
        
    def debug(self, message: str):
        self.log(message, "DEBUG")
        
    def info(self, message: str):
        self.log(message, "INFO")
        
    def report_test_result(self, test_name: str, passed: bool, details: str = ""):
        """报告测试结果"""
        status = "✅ 通过" if passed else "❌ 失败"
        message = f"{test_name}: {status}"
        if details:
            message += f" - {details}"
        self.log(message, "SUCCESS" if passed else "ERROR")
        
    def get_summary(self) -> Dict:
        """获取测试总结"""
        total_time = time.time() - self.start_time
        return {
            "total_time": total_time,
            "total_tests": len(self.test_results),
            "results": self.test_results
        }

def wait_for_service(url: str, timeout: int = 30, interval: int = 1) -> bool:
    """等待服务可用"""
    start_time = time.time()
    while time.time() - start_time < timeout:
        try:
            response = requests.get(url, timeout=5)
            if response.status_code < 500:  # 任何非服务器错误都认为服务可用
                return True
        except requests.RequestException:
            pass
        time.sleep(interval)
    return False

def check_service_health(base_url: str, endpoints: Dict[str, str]) -> Dict[str, bool]:
    """检查多个服务的健康状态"""
    results = {}
    for service_name, endpoint in endpoints.items():
        try:
            url = f"{base_url}{endpoint}"
            response = requests.get(url, timeout=5)
            # 认为2xx, 3xx, 甚至某些4xx状态码都是服务正常的标志
            is_healthy = response.status_code < 500
            results[service_name] = is_healthy
        except Exception:
            results[service_name] = False
    return results
