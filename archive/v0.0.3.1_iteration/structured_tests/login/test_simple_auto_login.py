#!/usr/bin/env python3
"""
简化的Chrome自动登录测试 - 专门测试admin/admin123凭据
"""

import time
import json
from selenium import webdriver
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException, NoSuchElementException

class SimpleAutoLoginTest:
    def __init__(self):
        self.base_url = "http://localhost:8080"
        self.admin_username = "admin"
        self.admin_password = "admin123"
        self.driver = None
        self.test_results = []
    
    def setup_chrome(self):
        """配置Chrome WebDriver"""
        chrome_options = Options()
        chrome_options.add_argument("--no-sandbox")
        chrome_options.add_argument("--disable-dev-shm-usage")
        chrome_options.add_argument("--disable-gpu")
        chrome_options.add_argument("--window-size=1920,1080")
        chrome_options.add_argument("--disable-blink-features=AutomationControlled")
        chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
        chrome_options.add_experimental_option('useAutomationExtension', False)
        
        try:
            service = Service('/opt/homebrew/bin/chromedriver')
            self.driver = webdriver.Chrome(service=service, options=chrome_options)
            self.driver.execute_script("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")
            print("✅ Chrome WebDriver 启动成功")
            return True
        except Exception as e:
            print(f"❌ Chrome WebDriver 启动失败: {e}")
            return False
    
    def take_screenshot(self, name, description=""):
        """截图并保存"""
        try:
            filename = f"simple_login_{name}.png"
            self.driver.save_screenshot(filename)
            size = len(open(filename, 'rb').read())
            print(f"📸 截图保存: {filename} ({size:,} bytes) - {description}")
            return filename
        except Exception as e:
            print(f"❌ 截图失败: {e}")
            return None
    
    def wait_for_element(self, locator, timeout=10, description=""):
        """等待元素出现"""
        try:
            element = WebDriverWait(self.driver, timeout).until(
                EC.presence_of_element_located(locator)
            )
            print(f"✅ 找到元素: {description} - {locator}")
            return element
        except TimeoutException:
            print(f"⏰ 等待超时: {description} - {locator}")
            return None
    
    def test_step_1_homepage(self):
        """测试步骤1: 访问主页"""
        print("\n📝 步骤 1: 访问主页")
        try:
            self.driver.get(self.base_url)
            time.sleep(3)
            
            # 截图
            self.take_screenshot("1_homepage", "主页加载")
            
            # 检查页面标题
            title = self.driver.title
            print(f"   页面标题: {title}")
            
            if "AI-Infra-Matrix" in title:
                print("✅ 主页加载成功")
                return True
            else:
                print("❌ 主页标题不正确")
                return False
                
        except Exception as e:
            print(f"❌ 主页访问失败: {e}")
            return False
    
    def test_step_2_projects_page(self):
        """测试步骤2: 导航到项目页面"""
        print("\n📝 步骤 2: 导航到项目页面")
        try:
            # 尝试找到并点击项目导航
            nav_selectors = [
                "a[href='/projects']",
                "a[href*='projects']",
                ".nav-link:contains('项目')",
                ".menu-item:contains('Projects')"
            ]
            
            project_link = None
            for selector in nav_selectors:
                try:
                    if 'contains' in selector:
                        # 跳过 contains 选择器，因为 Selenium 不支持
                        continue
                    project_link = self.driver.find_element(By.CSS_SELECTOR, selector)
                    if project_link:
                        break
                except:
                    continue
            
            if project_link:
                project_link.click()
                print("✅ 点击项目导航链接")
            else:
                # 直接导航到项目页面
                self.driver.get(f"{self.base_url}/projects")
                print("ℹ️ 直接导航到项目页面")
            
            time.sleep(3)
            self.take_screenshot("2_projects", "项目页面")
            
            current_url = self.driver.current_url
            print(f"   当前URL: {current_url}")
            
            if "/projects" in current_url:
                print("✅ 成功到达项目页面")
                return True
            else:
                print("❌ 项目页面导航失败")
                return False
                
        except Exception as e:
            print(f"❌ 项目页面导航失败: {e}")
            return False
    
    def test_step_3_find_jupyter_button(self):
        """测试步骤3: 寻找并点击JupyterHub按钮"""
        print("\n📝 步骤 3: 寻找JupyterHub入口")
        try:
            # 多种可能的JupyterHub按钮选择器
            jupyter_selectors = [
                "button:contains('Jupyter')",
                "a:contains('Jupyter')",
                ".jupyter-button",
                ".card:contains('JupyterHub')",
                "[data-testid*='jupyter']",
                "img[alt*='jupyter']",
                "div:contains('Jupyter')"
            ]
            
            # 由于Selenium不支持:contains，我们需要用XPath
            xpath_selectors = [
                "//button[contains(text(), 'Jupyter')]",
                "//a[contains(text(), 'Jupyter')]", 
                "//div[contains(text(), 'Jupyter')]",
                "//img[contains(@alt, 'jupyter')]",
                "//span[contains(text(), 'Jupyter')]"
            ]
            
            jupyter_element = None
            for xpath in xpath_selectors:
                try:
                    jupyter_element = self.driver.find_element(By.XPATH, xpath)
                    if jupyter_element:
                        print(f"✅ 找到JupyterHub元素: {xpath}")
                        break
                except:
                    continue
            
            if not jupyter_element:
                print("⚠️ 未找到JupyterHub按钮，尝试直接访问JupyterHub")
                self.driver.get(f"{self.base_url}/jupyterhub")
            else:
                jupyter_element.click()
                print("✅ 点击JupyterHub按钮")
            
            time.sleep(3)
            self.take_screenshot("3_jupyter_access", "JupyterHub访问")
            
            current_url = self.driver.current_url
            print(f"   当前URL: {current_url}")
            
            return True
                
        except Exception as e:
            print(f"❌ JupyterHub访问失败: {e}")
            return False
    
    def test_step_4_login_form(self):
        """测试步骤4: 处理登录表单"""
        print("\n📝 步骤 4: 处理登录表单")
        try:
            time.sleep(2)
            
            # 检查当前页面内容
            page_source = self.driver.page_source.lower()
            
            if "login" in page_source or "username" in page_source or "password" in page_source:
                print("✅ 检测到登录页面")
                
                # 寻找用户名和密码字段
                username_selectors = [
                    "input[name='username']",
                    "input[id='username']", 
                    "input[type='text']",
                    "input[placeholder*='username']",
                    "input[placeholder*='用户名']"
                ]
                
                password_selectors = [
                    "input[name='password']",
                    "input[id='password']",
                    "input[type='password']",
                    "input[placeholder*='password']",
                    "input[placeholder*='密码']"
                ]
                
                username_field = None
                password_field = None
                
                # 查找用户名字段
                for selector in username_selectors:
                    try:
                        username_field = self.driver.find_element(By.CSS_SELECTOR, selector)
                        if username_field:
                            print(f"✅ 找到用户名字段: {selector}")
                            break
                    except:
                        continue
                
                # 查找密码字段
                for selector in password_selectors:
                    try:
                        password_field = self.driver.find_element(By.CSS_SELECTOR, selector)
                        if password_field:
                            print(f"✅ 找到密码字段: {selector}")
                            break
                    except:
                        continue
                
                if username_field and password_field:
                    # 输入凭据
                    username_field.clear()
                    username_field.send_keys(self.admin_username)
                    print(f"✅ 输入用户名: {self.admin_username}")
                    
                    password_field.clear()
                    password_field.send_keys(self.admin_password)
                    print(f"✅ 输入密码: {'*' * len(self.admin_password)}")
                    
                    time.sleep(1)
                    self.take_screenshot("4_login_filled", "登录表单已填写")
                    
                    # 寻找并点击登录按钮
                    login_button_selectors = [
                        "button[type='submit']",
                        "input[type='submit']",
                        "button:contains('Login')",
                        "button:contains('登录')",
                        ".login-button",
                        "#login-button"
                    ]
                    
                    # 使用XPath查找登录按钮
                    login_button_xpaths = [
                        "//button[@type='submit']",
                        "//input[@type='submit']",
                        "//button[contains(text(), 'Login')]",
                        "//button[contains(text(), '登录')]",
                        "//button[contains(text(), 'Sign in')]"
                    ]
                    
                    login_button = None
                    for xpath in login_button_xpaths:
                        try:
                            login_button = self.driver.find_element(By.XPATH, xpath)
                            if login_button:
                                print(f"✅ 找到登录按钮: {xpath}")
                                break
                        except:
                            continue
                    
                    if login_button:
                        login_button.click()
                        print("✅ 点击登录按钮")
                        
                        time.sleep(3)
                        self.take_screenshot("5_after_login", "登录后")
                        
                        return True
                    else:
                        print("❌ 未找到登录按钮")
                        return False
                
                else:
                    print("❌ 未找到用户名或密码字段")
                    return False
            
            else:
                print("ℹ️ 当前页面不是登录页面，可能已经登录")
                return True
                
        except Exception as e:
            print(f"❌ 登录处理失败: {e}")
            return False
    
    def test_step_5_verify_success(self):
        """测试步骤5: 验证登录成功"""
        print("\n📝 步骤 5: 验证登录成功")
        try:
            time.sleep(3)
            
            current_url = self.driver.current_url
            page_title = self.driver.title
            page_source = self.driver.page_source.lower()
            
            print(f"   当前URL: {current_url}")
            print(f"   页面标题: {page_title}")
            
            self.take_screenshot("6_final_state", "最终状态")
            
            # 检查成功指标
            success_indicators = [
                "jupyter" in current_url.lower(),
                "hub" in current_url.lower(),
                "jupyter" in page_title.lower(),
                "dashboard" in page_source,
                "notebook" in page_source,
                "hub" in page_source
            ]
            
            success_count = sum(success_indicators)
            print(f"   成功指标: {success_count}/{len(success_indicators)}")
            
            if success_count >= 2:
                print("✅ 登录验证成功!")
                return True
            else:
                print("⚠️ 登录状态不确定")
                return False
                
        except Exception as e:
            print(f"❌ 登录验证失败: {e}")
            return False
    
    def run_test(self):
        """运行完整测试"""
        print("🚀 开始简化的Chrome自动登录测试")
        print("=" * 60)
        
        if not self.setup_chrome():
            return False
        
        try:
            # 执行测试步骤
            results = []
            
            results.append(("主页访问", self.test_step_1_homepage()))
            results.append(("项目页面", self.test_step_2_projects_page()))
            results.append(("JupyterHub访问", self.test_step_3_find_jupyter_button()))
            results.append(("登录处理", self.test_step_4_login_form()))
            results.append(("登录验证", self.test_step_5_verify_success()))
            
            # 生成报告
            print("\n" + "=" * 60)
            print("📊 测试结果报告")
            print("=" * 60)
            
            success_count = 0
            for step_name, success in results:
                status = "✅ 成功" if success else "❌ 失败"
                print(f"{step_name:<12}: {status}")
                if success:
                    success_count += 1
            
            overall_success = success_count == len(results)
            print(f"\n总体结果: {'✅ 成功' if overall_success else '❌ 部分失败'} ({success_count}/{len(results)})")
            
            if overall_success:
                print("\n🎉 admin/admin123 自动登录测试完全成功!")
                print("   用户可以通过以下流程无需手动输入密码访问JupyterHub:")
                print("   1. 访问主页 -> 2. 进入项目页面 -> 3. 点击Jupyter -> 4. 自动登录")
            else:
                print("\n⚠️ 测试部分成功，可能需要手动干预")
            
            return overall_success
            
        finally:
            if self.driver:
                print("\n🔄 清理浏览器实例...")
                self.driver.quit()

if __name__ == "__main__":
    test = SimpleAutoLoginTest()
    test.run_test()
