#!/usr/bin/env python3
"""
主测试运行器
提供命令行接口运行各种测试
"""

import sys
import argparse
from typing import Optional

try:
    from .sso_tests import SSOTestSuite
    from .utils import TestReporter, check_service_health
    from .config import DEFAULT_CONFIG, HEALTH_ENDPOINTS
except ImportError:
    # 如果作为脚本直接运行，使用相对导入
    from sso_tests import SSOTestSuite
    from utils import TestReporter, check_service_health
    from config import DEFAULT_CONFIG, HEALTH_ENDPOINTS

class TestController:
    """测试控制器"""
    
    def __init__(self, base_url: str, verbose: bool = False):
        self.base_url = base_url
        self.reporter = TestReporter(verbose)
        self.sso_suite = SSOTestSuite(base_url, self.reporter)
        
    def run_health_check(self) -> bool:
        """运行健康检查"""
        self.reporter.info("🏥 系统健康检查")
        
        health_results = check_service_health(self.base_url, HEALTH_ENDPOINTS)
        
        all_healthy = True
        for service, is_healthy in health_results.items():
            if is_healthy:
                self.reporter.success(f"{service}: 健康")
            else:
                self.reporter.error(f"{service}: 异常")
                all_healthy = False
                
        if all_healthy:
            self.reporter.success("✅ 所有服务健康")
        else:
            self.reporter.error("❌ 部分服务异常")
            
        return all_healthy
        
    def run_sso_tests(self) -> bool:
        """运行SSO测试"""
        return self.sso_suite.run_all_tests()
        
    def run_quick_test(self) -> bool:
        """运行快速验证测试"""
        self.reporter.info("🚀 快速验证测试")
        
        # 只运行关键测试
        try:
            if not self.sso_suite.test_authentication():
                return False
            if not self.sso_suite.test_original_problem_resolution():
                return False
            self.reporter.success("✅ 快速验证通过")
            return True
        except Exception as e:
            self.reporter.error(f"快速验证异常: {e}")
            return False
            
    def run_all_tests(self) -> bool:
        """运行完整测试套件"""
        self.reporter.info("🎯 完整测试套件")
        
        results = {}
        
        # 健康检查
        results['health'] = self.run_health_check()
        
        # SSO测试
        results['sso'] = self.run_sso_tests()
        
        # 汇总结果
        passed_count = sum(results.values())
        total_count = len(results)
        
        self.reporter.info(f"📊 总体结果: {passed_count}/{total_count} 通过")
        
        all_passed = passed_count == total_count
        if all_passed:
            self.reporter.success("🎉 所有测试通过！")
        else:
            self.reporter.error("❌ 部分测试失败")
            
        return all_passed

def main():
    """主函数"""
    parser = argparse.ArgumentParser(
        description="AI Infra Matrix 测试运行器",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  %(prog)s --test quick          # 快速验证
  %(prog)s --test sso            # SSO完整测试
  %(prog)s --test health         # 健康检查
  %(prog)s --test all -v         # 完整测试（详细输出）
        """
    )
    
    parser.add_argument(
        "--url", 
        default=DEFAULT_CONFIG["base_url"],
        help=f"基础URL (默认: {DEFAULT_CONFIG['base_url']})"
    )
    
    parser.add_argument(
        "--test",
        choices=["health", "sso", "quick", "all"],
        default="quick",
        help="要运行的测试类型 (默认: quick)"
    )
    
    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="详细输出"
    )
    
    args = parser.parse_args()
    
    controller = TestController(args.url, args.verbose)
    
    # 根据测试类型运行相应测试
    if args.test == "health":
        success = controller.run_health_check()
    elif args.test == "sso":
        success = controller.run_sso_tests()
    elif args.test == "quick":
        success = controller.run_quick_test()
    elif args.test == "all":
        success = controller.run_all_tests()
    else:
        print(f"未知的测试类型: {args.test}")
        sys.exit(1)
    
    # 根据结果设置退出码
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
