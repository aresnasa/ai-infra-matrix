#!/usr/bin/env python3
"""
基础测试模块
提供简单的单元测试示例
"""

import sys
import os

# 添加父目录到路径以支持导入
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

try:
    from utils import TestReporter
    from config import DEFAULT_CONFIG
except ImportError as e:
    print(f"导入错误: {e}")
    sys.exit(1)


class BasicTestSuite:
    """基础测试套件"""
    
    def __init__(self, reporter=None):
        self.reporter = reporter or TestReporter(verbose=True)
        
    def test_configuration(self) -> bool:
        """测试配置文件加载"""
        self.reporter.info("测试配置文件")
        
        # 验证必要的配置项存在
        required_keys = ["base_url", "credentials", "endpoints"]
        
        for key in required_keys:
            if key not in DEFAULT_CONFIG:
                self.reporter.error(f"缺少配置项: {key}")
                return False
        
        self.reporter.success("配置文件验证通过")
        return True
    
    def test_reporter(self) -> bool:
        """测试报告器功能"""
        self.reporter.info("测试报告器")
        
        # 测试各种日志级别
        self.reporter.log("普通日志", "INFO")
        self.reporter.success("成功日志")
        self.reporter.warning("警告日志")
        
        self.reporter.success("报告器功能正常")
        return True
    
    def test_basic_math(self) -> bool:
        """测试基本数学运算"""
        self.reporter.info("测试基本运算")
        
        # 简单的断言测试
        assert 1 + 1 == 2, "1 + 1 应该等于 2"
        assert 2 * 3 == 6, "2 * 3 应该等于 6"
        assert 10 - 5 == 5, "10 - 5 应该等于 5"
        
        self.reporter.success("基本运算测试通过")
        return True
    
    def run_all_tests(self) -> bool:
        """运行所有测试"""
        self.reporter.info("🚀 开始基础测试套件")
        
        tests = [
            ("配置验证", self.test_configuration),
            ("报告器测试", self.test_reporter),
            ("基本运算", self.test_basic_math),
        ]
        
        results = []
        for test_name, test_func in tests:
            try:
                result = test_func()
                results.append(result)
                status = "✅ 通过" if result else "❌ 失败"
                self.reporter.log(f"{test_name}: {status}", "SUCCESS" if result else "ERROR")
            except Exception as e:
                self.reporter.error(f"{test_name} 异常: {e}")
                results.append(False)
        
        all_passed = all(results)
        passed_count = sum(results)
        total_count = len(results)
        
        self.reporter.info(f"📊 测试结果: {passed_count}/{total_count} 通过")
        
        if all_passed:
            self.reporter.success("🎉 所有测试通过！")
        else:
            self.reporter.error("❌ 部分测试失败")
        
        return all_passed


def main():
    """主函数"""
    print("=" * 50)
    print("AI Infra Matrix - 基础测试")
    print("=" * 50)
    print()
    
    suite = BasicTestSuite()
    success = suite.run_all_tests()
    
    print()
    print("=" * 50)
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    main()
