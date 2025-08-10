#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
iframe自动登录综合测试脚本
测试各种iframe场景下的自动登录功能
"""

import time
import requests
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException, NoSuchElementException

class IframeAutoLoginTester:
    def __init__(self, username="admin", password="admin123", headless=True):
        self.username = username
        self.password = password
        self.headless = headless
        self.driver = None
        self.test_results = {}
    
    def setup_driver(self):
        """设置Chrome WebDriver"""
        chrome_options = Options()
        if self.headless:
            chrome_options.add_argument('--headless')
        chrome_options.add_argument('--no-sandbox')
        chrome_options.add_argument('--disable-dev-shm-usage')
        chrome_options.add_argument('--window-size=1920,1080')
        chrome_options.add_argument('--disable-blink-features=AutomationControlled')
        chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
        chrome_options.add_experimental_option('useAutomationExtension', False)
        
        self.driver = webdriver.Chrome(options=chrome_options)
        self.driver.execute_script("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")
        print("✅ Chrome WebDriver 已启动")
    
    def perform_auto_login(self, context=""):
        """执行自动登录"""
        print(f"🔐 尝试自动登录{context}...")
        
        try:
            # 等待页面加载完成
            WebDriverWait(self.driver, 10).until(
                EC.presence_of_element_located((By.TAG_NAME, "body"))
            )
            
            # 检查是否已经登录
            current_url = self.driver.current_url
            page_source = self.driver.page_source.lower()
            
            if 'login' not in current_url.lower() and 'login' not in page_source:
                print("   ✅ 已经登录或无需登录")
                return True
            
            # 查找用户名输入框
            username_selectors = [
                "input[name='username']",
                "input[name='user']", 
                "input[type='text']",
                "input[id*='username']",
                "input[id*='user']",
                "input[placeholder*='用户']",
                "input[placeholder*='Username']",
                "input[placeholder*='username']",
                "input[class*='username']"
            ]
            
            username_field = None
            for selector in username_selectors:
                try:
                    username_field = WebDriverWait(self.driver, 2).until(
                        EC.presence_of_element_located((By.CSS_SELECTOR, selector))
                    )
                    print(f"   找到用户名输入框: {selector}")
                    break
                except:
                    continue
            
            if not username_field:
                print("   ⚠️  未找到用户名输入框")
                return False
            
            # 查找密码输入框
            password_selectors = [
                "input[name='password']",
                "input[type='password']",
                "input[id*='password']",
                "input[placeholder*='密码']",
                "input[placeholder*='Password']",
                "input[class*='password']"
            ]
            
            password_field = None
            for selector in password_selectors:
                try:
                    password_field = self.driver.find_element(By.CSS_SELECTOR, selector)
                    print(f"   找到密码输入框: {selector}")
                    break
                except:
                    continue
            
            if not password_field:
                print("   ❌ 未找到密码输入框")
                return False
            
            # 输入凭据
            username_field.clear()
            username_field.send_keys(self.username)
            print(f"   ✅ 已输入用户名: {self.username}")
            
            password_field.clear()
            password_field.send_keys(self.password)
            print(f"   ✅ 已输入密码")
            
            # 查找并点击登录按钮
            login_selectors = [
                "button[type='submit']",
                "input[type='submit']",
                "input[value*='登录']",
                "input[value*='Login']",
                "input[value*='Sign']",
                ".btn-primary",
                ".login-button",
                "#login-submit",
                "button[class*='login']"
            ]
            
            login_button = None
            for selector in login_selectors:
                try:
                    login_button = self.driver.find_element(By.CSS_SELECTOR, selector)
                    print(f"   找到登录按钮: {selector}")
                    break
                except:
                    continue
            
            # 尝试通过文本查找按钮
            if not login_button:
                try:
                    login_button = self.driver.find_element(By.XPATH, "//button[contains(text(), '登录') or contains(text(), 'Login') or contains(text(), 'Sign')]")
                    print("   通过文本找到登录按钮")
                except:
                    pass
            
            if not login_button:
                # 尝试回车提交
                password_field.send_keys("\n")
                print("   ⚠️  未找到登录按钮，尝试回车提交")
            else:
                login_button.click()
                print("   ✅ 已点击登录按钮")
            
            # 等待登录处理
            time.sleep(3)
            
            # 检查登录结果
            new_url = self.driver.current_url
            new_source = self.driver.page_source.lower()
            
            if 'login' not in new_url.lower() and 'login' not in new_source:
                print("   ✅ 登录成功")
                return True
            else:
                print("   ❌ 登录可能失败，仍在登录页面")
                return False
                
        except Exception as e:
            print(f"   ❌ 自动登录失败: {e}")
            return False
    
    def test_iframe_scenario(self, test_name, url, iframe_selector, expected_behavior="login"):
        """测试特定iframe场景"""
        print(f"\n{'='*60}")
        print(f"🧪 测试场景: {test_name}")
        print(f"🔗 URL: {url}")
        print(f"🖼️  iframe选择器: {iframe_selector}")
        print(f"{'='*60}")
        
        test_result = {
            'url_accessible': False,
            'iframe_found': False,
            'iframe_loaded': False,
            'login_attempted': False,
            'login_successful': False,
            'content_visible': False,
            'error_message': None
        }
        
        try:
            # 1. 访问页面
            print("📝 1. 访问页面...")
            self.driver.get(url)
            WebDriverWait(self.driver, 10).until(
                EC.presence_of_element_located((By.TAG_NAME, "body"))
            )
            test_result['url_accessible'] = True
            print(f"   ✅ 页面已加载，标题: {self.driver.title}")
            
            # 2. 查找iframe
            print("🔍 2. 查找iframe...")
            try:
                iframe = WebDriverWait(self.driver, 10).until(
                    EC.presence_of_element_located((By.CSS_SELECTOR, iframe_selector))
                )
                test_result['iframe_found'] = True
                print("   ✅ iframe元素已找到")
                
                src = iframe.get_attribute('src')
                print(f"   iframe源地址: {src}")
                
            except TimeoutException:
                print("   ❌ iframe元素未找到")
                test_result['error_message'] = "iframe元素未找到"
                return test_result
            
            # 3. 等待iframe加载
            print("⏳ 3. 等待iframe加载...")
            time.sleep(5)
            
            # 4. 切换到iframe并检查内容
            print("🔍 4. 检查iframe内容...")
            try:
                self.driver.switch_to.frame(iframe)
                test_result['iframe_loaded'] = True
                
                # 等待iframe内容加载
                WebDriverWait(self.driver, 15).until(
                    lambda d: d.execute_script("return document.readyState") == "complete"
                )
                
                page_source = self.driver.page_source.lower()
                page_text = self.driver.find_element(By.TAG_NAME, "body").text.strip()
                
                print(f"   页面文本长度: {len(page_text)} 字符")
                
                # 5. 检查是否需要登录
                login_indicators = ['login', '登录', 'username', 'password', 'sign in']
                needs_login = any(indicator in page_source for indicator in login_indicators)
                
                if needs_login and expected_behavior == "login":
                    print("   🔐 检测到需要登录...")
                    test_result['login_attempted'] = True
                    
                    login_success = self.perform_auto_login(" (在iframe中)")
                    if login_success:
                        test_result['login_successful'] = True
                        # 重新检查内容
                        time.sleep(3)
                        page_text = self.driver.find_element(By.TAG_NAME, "body").text.strip()
                        print(f"   登录后页面文本长度: {len(page_text)} 字符")
                
                # 6. 检查最终内容
                if len(page_text) > 50:
                    test_result['content_visible'] = True
                    print("   ✅ iframe有足够内容，非白屏")
                else:
                    print("   ❌ iframe内容较少，可能是白屏")
                
                # 检查JupyterHub特征
                jupyter_indicators = ['jupyter', 'hub', 'notebook', 'spawner', 'server']
                jupyter_found = [ind for ind in jupyter_indicators if ind in page_source]
                
                if jupyter_found:
                    print(f"   ✅ 发现JupyterHub特征: {jupyter_found}")
                
            except Exception as e:
                print(f"   ❌ iframe内容检查失败: {e}")
                test_result['error_message'] = str(e)
            finally:
                # 切换回主页面
                self.driver.switch_to.default_content()
            
            # 7. 保存截图
            screenshot_name = f"{test_name.replace(' ', '_').lower()}_test.png"
            self.driver.save_screenshot(screenshot_name)
            print(f"📸 测试截图已保存: {screenshot_name}")
            
        except Exception as e:
            print(f"❌ 测试失败: {e}")
            test_result['error_message'] = str(e)
        
        self.test_results[test_name] = test_result
        return test_result
    
    def run_all_tests(self):
        """运行所有iframe测试场景"""
        print("🚀 开始iframe自动登录综合测试")
        print("="*80)
        
        # 设置WebDriver
        self.setup_driver()
        
        try:
            # 测试场景列表
            test_scenarios = [
                {
                    'name': 'JupyterHub Wrapper',
                    'url': 'http://localhost:8080/jupyterhub/',
                    'iframe_selector': '#jupyter-frame',
                    'expected': 'login'
                },
                {
                    'name': '直接访问JupyterHub',
                    'url': 'http://localhost:8080/jupyter/hub/',
                    'iframe_selector': None,  # 不是iframe，直接页面
                    'expected': 'login'
                },
                {
                    'name': 'Projects页面中的iframe',
                    'url': 'http://localhost:8080/projects',
                    'iframe_selector': 'iframe',  # 通用iframe选择器
                    'expected': 'content'
                }
            ]
            
            # 运行每个测试场景
            for scenario in test_scenarios:
                if scenario['iframe_selector']:
                    self.test_iframe_scenario(
                        scenario['name'],
                        scenario['url'], 
                        scenario['iframe_selector'],
                        scenario['expected']
                    )
                else:
                    # 直接页面测试
                    self.test_direct_page(scenario['name'], scenario['url'])
                
                time.sleep(2)  # 测试间隔
            
            # 打印测试总结
            self.print_test_summary()
            
        finally:
            if self.driver:
                self.driver.quit()
                print("🔚 WebDriver已关闭")
    
    def test_direct_page(self, test_name, url):
        """测试直接页面（非iframe）"""
        print(f"\n{'='*60}")
        print(f"🧪 测试场景: {test_name} (直接页面)")
        print(f"🔗 URL: {url}")
        print(f"{'='*60}")
        
        test_result = {
            'url_accessible': False,
            'login_attempted': False,
            'login_successful': False,
            'content_visible': False,
            'error_message': None
        }
        
        try:
            print("📝 1. 访问页面...")
            self.driver.get(url)
            WebDriverWait(self.driver, 10).until(
                EC.presence_of_element_located((By.TAG_NAME, "body"))
            )
            test_result['url_accessible'] = True
            print(f"   ✅ 页面已加载，标题: {self.driver.title}")
            
            # 检查是否需要登录
            page_source = self.driver.page_source.lower()
            current_url = self.driver.current_url
            
            if 'login' in current_url.lower() or any(indicator in page_source for indicator in ['login', '登录', 'username', 'password']):
                print("   🔐 检测到需要登录...")
                test_result['login_attempted'] = True
                
                login_success = self.perform_auto_login(" (直接页面)")
                if login_success:
                    test_result['login_successful'] = True
            
            # 检查页面内容
            page_text = self.driver.find_element(By.TAG_NAME, "body").text.strip()
            if len(page_text) > 50:
                test_result['content_visible'] = True
                print(f"   ✅ 页面有内容 ({len(page_text)} 字符)")
            else:
                print(f"   ⚠️  页面内容较少 ({len(page_text)} 字符)")
            
            # 保存截图
            screenshot_name = f"{test_name.replace(' ', '_').lower()}_direct_test.png"
            self.driver.save_screenshot(screenshot_name)
            print(f"📸 测试截图已保存: {screenshot_name}")
            
        except Exception as e:
            print(f"❌ 测试失败: {e}")
            test_result['error_message'] = str(e)
        
        self.test_results[test_name] = test_result
        return test_result
    
    def print_test_summary(self):
        """打印测试结果总结"""
        print("\n" + "="*80)
        print("📊 测试结果总结")
        print("="*80)
        
        for test_name, result in self.test_results.items():
            print(f"\n🧪 {test_name}:")
            
            for key, value in result.items():
                if key == 'error_message' and not value:
                    continue
                
                icon = "✅" if value else "❌"
                if key == 'error_message':
                    icon = "❌"
                    
                print(f"   {icon} {key.replace('_', ' ').title()}: {value}")
        
        # 统计
        total_tests = len(self.test_results)
        successful_tests = sum(1 for result in self.test_results.values() 
                             if result.get('content_visible', False))
        
        print(f"\n📈 总体统计:")
        print(f"   总测试数: {total_tests}")
        print(f"   成功测试: {successful_tests}")
        print(f"   成功率: {successful_tests/total_tests*100:.1f}%" if total_tests > 0 else "   成功率: 0%")
        
        if successful_tests >= total_tests * 0.8:
            print("🎉 测试整体通过！")
        else:
            print("⚠️  需要进一步优化")

def main():
    """主函数"""
    print("🔬 iframe自动登录综合测试")
    print("用于测试各种iframe场景下的自动登录功能")
    print("="*80)
    
    # 创建测试器实例
    tester = IframeAutoLoginTester(
        username="admin",
        password="admin123", 
        headless=True  # 设置为False可以看到浏览器操作
    )
    
    try:
        # 运行所有测试
        tester.run_all_tests()
        
    except KeyboardInterrupt:
        print("\n⏹️  测试被用户中断")
    except Exception as e:
        print(f"\n💥 测试过程中发生意外错误: {e}")

if __name__ == "__main__":
    main()
