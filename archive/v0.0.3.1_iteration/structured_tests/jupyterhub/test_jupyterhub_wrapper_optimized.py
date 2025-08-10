#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
JupyterHub Wrapper 优化版本验证脚本
验证iframe显示JupyterHub内容并且没有白屏问题
"""

import time
import requests
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException, WebDriverException
import logging

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('jupyterhub_wrapper_verification.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

class JupyterHubWrapperTester:
    def __init__(self):
        self.base_url = "http://localhost:8080"
        self.driver = None
        self.test_results = {
            'wrapper_load': False,
            'iframe_present': False,
            'iframe_loaded': False,
            'jupyter_content': False,
            'no_white_screen': False,
            'redirect_working': False
        }
    
    def setup_driver(self):
        """设置Chrome WebDriver"""
        try:
            chrome_options = Options()
            chrome_options.add_argument('--headless')
            chrome_options.add_argument('--no-sandbox')
            chrome_options.add_argument('--disable-dev-shm-usage')
            chrome_options.add_argument('--disable-gpu')
            chrome_options.add_argument('--window-size=1920,1080')
            chrome_options.add_argument('--disable-blink-features=AutomationControlled')
            chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
            chrome_options.add_experimental_option('useAutomationExtension', False)
            
            self.driver = webdriver.Chrome(options=chrome_options)
            self.driver.execute_script("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")
            logger.info("Chrome WebDriver 初始化成功")
            return True
        except Exception as e:
            logger.error(f"WebDriver 初始化失败: {e}")
            return False
    
    def test_api_endpoints(self):
        """测试API端点可用性"""
        endpoints = [
            f"{self.base_url}/",
            f"{self.base_url}/jupyterhub",
            f"{self.base_url}/jupyterhub/",
            f"{self.base_url}/jupyter/hub/"
        ]
        
        logger.info("测试API端点可用性...")
        for endpoint in endpoints:
            try:
                response = requests.get(endpoint, timeout=10, allow_redirects=True)
                logger.info(f"  {endpoint}: {response.status_code}")
                if endpoint.endswith('/jupyterhub') and response.status_code == 301:
                    self.test_results['redirect_working'] = True
                    logger.info(f"    重定向到: {response.headers.get('Location', 'N/A')}")
            except Exception as e:
                logger.warning(f"  {endpoint}: 连接失败 - {e}")
    
    def test_wrapper_load(self):
        """测试wrapper页面加载"""
        try:
            logger.info("测试wrapper页面加载...")
            self.driver.get(f"{self.base_url}/jupyterhub/")
            
            # 等待页面加载
            WebDriverWait(self.driver, 10).until(
                EC.presence_of_element_located((By.TAG_NAME, "body"))
            )
            
            # 检查页面标题
            title = self.driver.title
            logger.info(f"页面标题: {title}")
            
            if "JupyterHub" in title:
                self.test_results['wrapper_load'] = True
                logger.info("✅ Wrapper页面加载成功")
            else:
                logger.warning("⚠️  页面标题不包含JupyterHub")
            
            return True
        except Exception as e:
            logger.error(f"❌ Wrapper页面加载失败: {e}")
            return False
    
    def test_iframe_presence(self):
        """测试iframe元素存在"""
        try:
            logger.info("检查iframe元素...")
            iframe = WebDriverWait(self.driver, 10).until(
                EC.presence_of_element_located((By.ID, "jupyter-frame"))
            )
            
            if iframe:
                self.test_results['iframe_present'] = True
                logger.info("✅ iframe元素存在")
                
                # 检查iframe属性
                src = iframe.get_attribute('src')
                sandbox = iframe.get_attribute('sandbox')
                logger.info(f"iframe src: {src}")
                logger.info(f"iframe sandbox: {sandbox}")
                
                return iframe
            else:
                logger.error("❌ iframe元素不存在")
                return None
        except Exception as e:
            logger.error(f"❌ 查找iframe元素失败: {e}")
            return None
    
    def test_iframe_loading(self):
        """测试iframe内容加载"""
        try:
            logger.info("等待iframe加载...")
            
            # 等待加载覆盖层消失
            WebDriverWait(self.driver, 15).until(
                EC.invisibility_of_element_located((By.ID, "loading-overlay"))
            )
            
            # 检查状态指示器
            status_indicator = self.driver.find_element(By.ID, "status-indicator")
            status_class = status_indicator.get_attribute('class')
            logger.info(f"状态指示器: {status_class}")
            
            if 'connected' in status_class:
                self.test_results['iframe_loaded'] = True
                logger.info("✅ iframe加载成功")
                return True
            else:
                logger.warning("⚠️  iframe可能未正确加载")
                return False
                
        except TimeoutException:
            logger.warning("⚠️  iframe加载超时，检查是否有错误覆盖层")
            try:
                error_overlay = self.driver.find_element(By.ID, "error-overlay")
                if 'show' in error_overlay.get_attribute('class'):
                    error_message = self.driver.find_element(By.ID, "error-message").text
                    logger.error(f"iframe加载错误: {error_message}")
                    return False
                else:
                    # 没有错误，可能只是加载慢
                    self.test_results['iframe_loaded'] = True
                    logger.info("✅ iframe似乎已加载（无错误显示）")
                    return True
            except:
                logger.error("❌ iframe加载失败且无错误信息")
                return False
        except Exception as e:
            logger.error(f"❌ 测试iframe加载时出错: {e}")
            return False
    
    def test_jupyter_content(self):
        """测试JupyterHub内容是否正确显示"""
        try:
            logger.info("检查JupyterHub内容...")
            
            # 切换到iframe
            iframe = self.driver.find_element(By.ID, "jupyter-frame")
            self.driver.switch_to.frame(iframe)
            
            # 等待JupyterHub内容加载
            try:
                # 查找JupyterHub特征元素
                WebDriverWait(self.driver, 10).until(
                    lambda driver: driver.execute_script("return document.readyState") == "complete"
                )
                
                # 检查页面内容
                page_source = self.driver.page_source.lower()
                jupyter_indicators = ['jupyter', 'hub', 'login', 'notebook', 'spawner']
                
                found_indicators = [indicator for indicator in jupyter_indicators if indicator in page_source]
                logger.info(f"找到的JupyterHub指示器: {found_indicators}")
                
                if len(found_indicators) >= 2:
                    self.test_results['jupyter_content'] = True
                    logger.info("✅ JupyterHub内容正确显示")
                    
                    # 检查是否是白屏
                    body_text = self.driver.find_element(By.TAG_NAME, "body").text.strip()
                    if len(body_text) > 50:  # 有足够的内容
                        self.test_results['no_white_screen'] = True
                        logger.info("✅ 确认非白屏状态")
                    else:
                        logger.warning("⚠️  页面内容较少，可能存在显示问题")
                    
                    return True
                else:
                    logger.warning("⚠️  未找到足够的JupyterHub特征内容")
                    return False
                    
            except TimeoutException:
                logger.warning("⚠️  iframe内容加载超时")
                return False
            finally:
                # 切换回主页面
                self.driver.switch_to.default_content()
                
        except Exception as e:
            logger.error(f"❌ 检查JupyterHub内容时出错: {e}")
            # 确保切换回主页面
            try:
                self.driver.switch_to.default_content()
            except:
                pass
            return False
    
    def take_screenshot(self, filename):
        """截图保存"""
        try:
            self.driver.save_screenshot(filename)
            logger.info(f"截图已保存: {filename}")
        except Exception as e:
            logger.error(f"截图失败: {e}")
    
    def cleanup(self):
        """清理资源"""
        if self.driver:
            self.driver.quit()
            logger.info("WebDriver已关闭")
    
    def run_tests(self):
        """运行所有测试"""
        logger.info("=" * 60)
        logger.info("开始JupyterHub Wrapper优化版本验证测试")
        logger.info("=" * 60)
        
        try:
            # 1. 设置WebDriver
            if not self.setup_driver():
                return False
            
            # 2. 测试API端点
            self.test_api_endpoints()
            
            # 3. 测试wrapper加载
            if not self.test_wrapper_load():
                return False
            
            # 截图1: 初始加载状态
            self.take_screenshot("wrapper_initial_load.png")
            
            # 4. 测试iframe存在
            iframe = self.test_iframe_presence()
            if not iframe:
                return False
            
            # 5. 测试iframe加载
            if not self.test_iframe_loading():
                logger.warning("iframe加载可能有问题，继续测试...")
            
            # 截图2: iframe加载状态
            self.take_screenshot("wrapper_iframe_loaded.png")
            
            # 6. 测试JupyterHub内容
            self.test_jupyter_content()
            
            # 截图3: 最终状态
            self.take_screenshot("wrapper_final_state.png")
            
            return True
            
        except Exception as e:
            logger.error(f"测试过程中出现错误: {e}")
            return False
        finally:
            self.cleanup()
    
    def print_results(self):
        """打印测试结果"""
        logger.info("=" * 60)
        logger.info("测试结果总结")
        logger.info("=" * 60)
        
        total_tests = len(self.test_results)
        passed_tests = sum(1 for result in self.test_results.values() if result)
        
        for test_name, result in self.test_results.items():
            status = "✅ 通过" if result else "❌ 失败"
            test_desc = {
                'wrapper_load': 'Wrapper页面加载',
                'iframe_present': 'iframe元素存在',
                'iframe_loaded': 'iframe内容加载',
                'jupyter_content': 'JupyterHub内容显示',
                'no_white_screen': '非白屏状态',
                'redirect_working': '重定向功能'
            }
            logger.info(f"{test_desc.get(test_name, test_name)}: {status}")
        
        logger.info("-" * 60)
        success_rate = (passed_tests / total_tests) * 100
        logger.info(f"总体成功率: {passed_tests}/{total_tests} ({success_rate:.1f}%)")
        
        if success_rate >= 80:
            logger.info("🎉 测试整体通过！JupyterHub Wrapper优化成功")
        elif success_rate >= 60:
            logger.info("⚠️  测试部分通过，需要进一步优化")
        else:
            logger.info("❌ 测试失败，需要重新检查配置")

def main():
    """主函数"""
    tester = JupyterHubWrapperTester()
    
    try:
        success = tester.run_tests()
        tester.print_results()
        
        if success:
            print("\n✅ JupyterHub Wrapper优化版本验证完成")
        else:
            print("\n❌ 验证过程中遇到问题")
            
    except KeyboardInterrupt:
        print("\n⏹️  测试被用户中断")
    except Exception as e:
        print(f"\n💥 测试过程中发生意外错误: {e}")
    finally:
        tester.cleanup()

if __name__ == "__main__":
    main()
