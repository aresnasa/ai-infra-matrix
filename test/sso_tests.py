"""
SSO专项测试模块
"""

from typing import Optional

try:
    from .utils import TestSession, TestValidator, TestReporter
    from .config import DEFAULT_CONFIG, TEST_SCENARIOS
except ImportError:
    from utils import TestSession, TestValidator, TestReporter
    from config import DEFAULT_CONFIG, TEST_SCENARIOS

class SSOTestSuite:
    """SSO测试套件"""
    
    def __init__(self, base_url: Optional[str] = None, reporter: Optional[TestReporter] = None):
        self.base_url = base_url or DEFAULT_CONFIG["base_url"]
        self.reporter = reporter or TestReporter()
        self.session = TestSession(self.base_url)
        self.config = DEFAULT_CONFIG
        
    def test_authentication(self) -> bool:
        """测试基础认证功能"""
        self.reporter.info("开始测试基础认证功能")
        
        creds = self.config["credentials"]
        success, message = self.session.login(creds["username"], creds["password"])
        
        if success:
            self.reporter.success(f"认证成功: {message}")
            return True
        else:
            self.reporter.error(f"认证失败: {message}")
            return False
            
    def test_sso_redirect(self) -> bool:
        """测试SSO自动重定向"""
        self.reporter.info("测试SSO自动重定向功能")
        
        if not self.session.token:
            self.reporter.error("未登录，无法测试SSO")
            return False
            
        scenario = TEST_SCENARIOS["sso_redirect"]
        response = self.session.get(scenario["url"], allow_redirects=False)
        
        # 检查状态码
        if response.status_code in scenario["expected_status"]:
            # 检查重定向位置
            location = TestValidator.extract_redirect_location(response)
            if location and scenario["expected_location_contains"] in location:
                self.reporter.success(f"SSO重定向成功: {location}")
                return True
            else:
                self.reporter.error(f"重定向位置错误: {location}")
                return False
        else:
            self.reporter.error(f"SSO重定向失败，状态码: {response.status_code}")
            return False
            
    def test_no_token_access(self) -> bool:
        """测试无token访问行为"""
        self.reporter.info("测试无token访问行为")
        
        # 创建新的session（无认证）
        clean_session = TestSession(self.base_url)
        scenario = TEST_SCENARIOS["no_token_access"]
        
        response = clean_session.get(scenario["url"])
        
        if response.status_code in scenario["expected_status"]:
            # 检查是否包含登录表单
            if TestValidator.contains_login_form(response.text):
                self.reporter.success("无token正确显示登录表单")
                return True
            else:
                self.reporter.warning("无token访问但未显示登录表单")
                return False
        else:
            self.reporter.error(f"无token访问失败，状态码: {response.status_code}")
            return False
            
    def test_original_problem_resolution(self) -> bool:
        """验证原始问题已解决"""
        self.reporter.info("验证原始问题已解决")
        
        if not self.session.token:
            if not self.test_authentication():
                return False
                
        # 原始问题：已登录访问Gitea登录页面仍需要密码
        response = self.session.get(
            "/gitea/user/login?redirect_to=%2Fgitea%2Fadmin",
            allow_redirects=False
        )
        
        if TestValidator.is_redirect(response):
            location = TestValidator.extract_redirect_location(response)
            if location and '/gitea/' in location:
                self.reporter.success("✅ 原始问题已解决：已登录用户无需二次密码")
                return True
            else:
                self.reporter.error(f"重定向位置错误: {location}")
                return False
        elif TestValidator.is_success(response):
            # 如果是200响应，检查是否包含密码框
            if not TestValidator.contains_login_form(response.text):
                self.reporter.success("✅ 原始问题已解决：已登录用户直接访问成功")
                return True
            else:
                self.reporter.error("❌ 原始问题未解决：仍需要密码输入")
                return False
        else:
            self.reporter.error(f"意外的响应状态: {response.status_code}")
            return False
            
    def run_all_tests(self) -> bool:
        """运行所有SSO测试"""
        self.reporter.info("🚀 开始SSO完整测试")
        
        tests = [
            ("基础认证", self.test_authentication),
            ("SSO重定向", self.test_sso_redirect),
            ("无token访问", self.test_no_token_access),
            ("原始问题验证", self.test_original_problem_resolution)
        ]
        
        results = []
        for test_name, test_func in tests:
            try:
                result = test_func()
                results.append(result)
                self.reporter.report_test_result(test_name, result)
            except Exception as e:
                self.reporter.error(f"{test_name}测试异常: {e}")
                results.append(False)
                
        all_passed = all(results)
        passed_count = sum(results)
        total_count = len(results)
        
        self.reporter.info(f"📊 SSO测试结果: {passed_count}/{total_count} 通过")
        
        if all_passed:
            self.reporter.success("🎉 所有SSO测试通过！")
        else:
            self.reporter.error("❌ 部分SSO测试失败")
            
        return all_passed
