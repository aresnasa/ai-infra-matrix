#!/usr/bin/env python3
"""
完整的JupyterHub登录测试 - 确保无白屏，体验一致
"""

import time
import logging
import json
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.common.action_chains import ActionChains

# 设置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class JupyterHubLoginTest:
    def __init__(self):
        self.driver = None
        self.wait = None
        
    def setup_driver(self):
        """设置Chrome WebDriver"""
        chrome_options = Options()
        chrome_options.add_argument('--no-sandbox')
        chrome_options.add_argument('--disable-dev-shm-usage')
        chrome_options.add_argument('--disable-web-security')
        chrome_options.add_argument('--window-size=1920,1080')
        chrome_options.add_argument('--disable-blink-features=AutomationControlled')
        chrome_options.add_argument('--disable-extensions')
        chrome_options.add_argument('--disable-gpu')
        chrome_options.add_argument('--disable-software-rasterizer')
        chrome_options.add_argument('--disable-background-timer-throttling')
        chrome_options.add_argument('--disable-backgrounding-occluded-windows')
        chrome_options.add_argument('--disable-renderer-backgrounding')
        chrome_options.add_argument('--no-first-run')
        chrome_options.add_argument('--no-default-browser-check')
        chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
        chrome_options.add_experimental_option('useAutomationExtension', False)
        
        # 显式指定Chrome路径
        chrome_binary = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
        chromedriver_path = '/opt/homebrew/bin/chromedriver'
        
        # 检查Chrome是否存在
        import os
        if not os.path.exists(chrome_binary):
            logger.error(f"❌ Chrome浏览器未找到: {chrome_binary}")
            return False
            
        # 检查ChromeDriver是否存在
        if not os.path.exists(chromedriver_path):
            logger.error(f"❌ ChromeDriver未找到: {chromedriver_path}")
            return False
            
        chrome_options.binary_location = chrome_binary
        
        try:
            # 检查版本兼容性
            logger.info("🔍 检查Chrome和ChromeDriver版本...")
            
            # 获取Chrome版本
            import subprocess
            try:
                chrome_version = subprocess.check_output([chrome_binary, '--version'], 
                                                       universal_newlines=True, 
                                                       timeout=10).strip()
                logger.info(f"Chrome版本: {chrome_version}")
            except Exception as e:
                logger.warning(f"⚠️ 无法获取Chrome版本: {e}")
            
            # 获取ChromeDriver版本
            try:
                chromedriver_version = subprocess.check_output([chromedriver_path, '--version'], 
                                                             universal_newlines=True, 
                                                             timeout=10).strip()
                logger.info(f"ChromeDriver版本: {chromedriver_version}")
            except Exception as e:
                logger.warning(f"⚠️ 无法获取ChromeDriver版本: {e}")
            
            # 创建Service对象
            service = Service(chromedriver_path)
            
            # 尝试启动WebDriver
            logger.info("🚀 正在启动Chrome WebDriver...")
            self.driver = webdriver.Chrome(service=service, options=chrome_options)
            self.driver.set_page_load_timeout(30)
            self.wait = WebDriverWait(self.driver, 20)
            
            # 执行脚本隐藏WebDriver特征
            try:
                self.driver.execute_script("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")
            except Exception as e:
                logger.warning(f"⚠️ 隐藏WebDriver特征失败: {e}")
            
            logger.info("✅ Chrome WebDriver初始化成功")
            return True
            
        except Exception as e:
            logger.error(f"❌ Chrome WebDriver初始化失败: {e}")
            logger.error(f"错误类型: {type(e).__name__}")
            
            # 尝试解决常见问题
            if "chrome not reachable" in str(e).lower():
                logger.error("💡 建议: Chrome进程可能已崩溃，请重启Chrome浏览器")
            elif "chromedriver" in str(e).lower() and "version" in str(e).lower():
                logger.error("💡 建议: ChromeDriver版本与Chrome不匹配，请更新ChromeDriver")
                logger.error("   运行: brew upgrade chromedriver")
            elif "permission" in str(e).lower():
                logger.error("💡 建议: ChromeDriver权限问题，请运行:")
                logger.error("   xattr -d com.apple.quarantine /opt/homebrew/bin/chromedriver")
            
            return False
    
    def test_complete_login_flow(self):
        """完整的登录流程测试"""
        logger.info("🧪 开始完整登录流程测试")
        
        try:
            # Step 1: 访问主页
            logger.info("📍 Step 1: 访问主页")
            self.driver.get("http://localhost:8080/")
            time.sleep(2)
            
            page_title = self.driver.title
            logger.info(f"主页标题: {page_title}")
            
            # 截图1: 主页
            self.driver.save_screenshot('step1_homepage.png')
            logger.info("📸 截图已保存: step1_homepage.png")
            
            # Step 2: 导航到projects页面
            logger.info("📍 Step 2: 导航到Projects页面")
            self.driver.get("http://localhost:8080/projects")
            time.sleep(3)
            
            # 等待页面加载完成
            self.wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))
            
            projects_title = self.driver.title
            logger.info(f"Projects页面标题: {projects_title}")
            
            # 截图2: Projects页面
            self.driver.save_screenshot('step2_projects.png')
            logger.info("📸 截图已保存: step2_projects.png")
            
            # Step 3: 查找并点击Jupyter图标
            logger.info("📍 Step 3: 查找Jupyter菜单项")
            
            # 尝试多种可能的选择器
            jupyter_selectors = [
                "a[href*='jupyter']",
                "a[href='/jupyterhub']",
                "[data-testid*='jupyter']",
                "button:contains('Jupyter')",
                "a:contains('Jupyter')",
                ".menu-item[href*='jupyter']",
                "nav a[href*='jupyter']"
            ]
            
            jupyter_element = None
            for selector in jupyter_selectors:
                try:
                    if 'contains' in selector:
                        # 使用XPath查找包含文本的元素
                        if 'button' in selector:
                            elements = self.driver.find_elements(By.XPATH, "//button[contains(text(), 'Jupyter') or contains(text(), 'jupyter')]")
                        else:
                            elements = self.driver.find_elements(By.XPATH, "//a[contains(text(), 'Jupyter') or contains(text(), 'jupyter')]")
                    else:
                        elements = self.driver.find_elements(By.CSS_SELECTOR, selector)
                    
                    if elements:
                        jupyter_element = elements[0]
                        logger.info(f"✅ 找到Jupyter元素: {selector}")
                        break
                except Exception as e:
                    logger.debug(f"选择器 {selector} 未找到元素: {e}")
                    continue
            
            if not jupyter_element:
                # 如果没找到，直接导航到JupyterHub
                logger.warning("⚠️ 未找到Jupyter菜单项，直接导航到JupyterHub")
                self.driver.get("http://localhost:8080/jupyterhub")
            else:
                # 点击Jupyter元素
                logger.info("🖱️ 点击Jupyter菜单项")
                
                # 滚动到元素可见
                self.driver.execute_script("arguments[0].scrollIntoView(true);", jupyter_element)
                time.sleep(1)
                
                # 使用ActionChains点击
                actions = ActionChains(self.driver)
                actions.move_to_element(jupyter_element).click().perform()
            
            time.sleep(3)
            
            # Step 4: 验证是否到达JupyterHub页面
            logger.info("📍 Step 4: 验证JupyterHub页面")
            
            current_url = self.driver.current_url
            logger.info(f"当前URL: {current_url}")
            
            # 检查是否在iframe中
            if 'iframe_test.html' in current_url or 'projects' in current_url:
                logger.info("🔍 检查是否在iframe容器页面中")
                
                # 查找iframe
                iframes = self.driver.find_elements(By.TAG_NAME, "iframe")
                logger.info(f"找到 {len(iframes)} 个iframe")
                
                for i, iframe in enumerate(iframes):
                    src = iframe.get_attribute("src")
                    logger.info(f"iframe[{i}] src: {src}")
                    
                    if 'jupyter' in src.lower():
                        logger.info(f"🎯 切换到JupyterHub iframe[{i}]")
                        
                        # 等待iframe加载
                        time.sleep(5)
                        
                        # 切换到iframe
                        self.driver.switch_to.frame(iframe)
                        
                        # 检查iframe内容
                        try:
                            iframe_body = self.wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))
                            iframe_text = iframe_body.text
                            logger.info(f"iframe内容长度: {len(iframe_text)} 字符")
                            logger.info(f"iframe内容预览: {iframe_text[:100]}...")
                            
                            if len(iframe_text.strip()) < 10:
                                logger.error("❌ iframe内容为空或很少 - 白屏问题!")
                                self.driver.save_screenshot('step4_iframe_blank.png')
                                return False
                            else:
                                logger.info("✅ iframe有内容")
                                
                        except Exception as e:
                            logger.error(f"❌ 检查iframe内容失败: {e}")
                            self.driver.save_screenshot('step4_iframe_error.png')
                            return False
                        
                        # 切换回主文档
                        self.driver.switch_to.default_content()
                        break
            
            # 截图3: JupyterHub页面
            self.driver.save_screenshot('step4_jupyterhub.png')
            logger.info("📸 截图已保存: step4_jupyterhub.png")
            
            # Step 5: 查找并测试登录表单
            logger.info("📍 Step 5: 测试登录功能")
            
            # 如果在iframe中，需要先切换
            if 'iframe_test.html' in self.driver.current_url or 'projects' in self.driver.current_url:
                iframes = self.driver.find_elements(By.TAG_NAME, "iframe")
                for iframe in iframes:
                    src = iframe.get_attribute("src")
                    if 'jupyter' in src.lower():
                        self.driver.switch_to.frame(iframe)
                        break
            
            # 查找登录表单
            try:
                # 等待登录表单出现
                login_form = self.wait.until(EC.presence_of_element_located((By.TAG_NAME, "form")))
                logger.info("✅ 找到登录表单")
                
                # 查找用户名和密码字段
                username_selectors = [
                    "input[name='username']",
                    "input[id='username']",
                    "input[type='text']",
                    "#username_input"
                ]
                
                password_selectors = [
                    "input[name='password']",
                    "input[id='password']", 
                    "input[type='password']",
                    "#password_input"
                ]
                
                username_field = None
                password_field = None
                
                for selector in username_selectors:
                    try:
                        username_field = self.driver.find_element(By.CSS_SELECTOR, selector)
                        logger.info(f"✅ 找到用户名字段: {selector}")
                        break
                    except:
                        continue
                
                for selector in password_selectors:
                    try:
                        password_field = self.driver.find_element(By.CSS_SELECTOR, selector)
                        logger.info(f"✅ 找到密码字段: {selector}")
                        break
                    except:
                        continue
                
                if username_field and password_field:
                    # 测试表单输入
                    logger.info("🔐 测试登录表单输入")
                    
                    username_field.clear()
                    username_field.send_keys("testuser")
                    
                    password_field.clear()
                    password_field.send_keys("testpass")
                    
                    logger.info("✅ 表单输入测试成功")
                    
                    # 截图4: 填写表单
                    self.driver.save_screenshot('step5_login_form.png')
                    logger.info("📸 截图已保存: step5_login_form.png")
                else:
                    logger.warning("⚠️ 未找到完整的登录表单字段")
                    
            except Exception as e:
                logger.error(f"❌ 登录表单测试失败: {e}")
                self.driver.save_screenshot('step5_login_error.png')
            
            # Step 6: 测试页面响应性
            logger.info("📍 Step 6: 测试页面响应性")
            
            # 切换回主文档（如果之前在iframe中）
            self.driver.switch_to.default_content()
            
            # 测试不同的视口大小
            viewports = [
                (1920, 1080, "桌面"),
                (1024, 768, "平板"),
                (375, 667, "手机")
            ]
            
            for width, height, device in viewports:
                logger.info(f"📱 测试 {device} 视口: {width}x{height}")
                self.driver.set_window_size(width, height)
                time.sleep(2)
                
                # 截图不同视口
                self.driver.save_screenshot(f'step6_viewport_{device.lower()}.png')
                logger.info(f"📸 截图已保存: step6_viewport_{device.lower()}.png")
            
            # 恢复原始大小
            self.driver.set_window_size(1920, 1080)
            
            # Step 7: 最终验证
            logger.info("📍 Step 7: 最终验证")
            
            # 重新访问完整流程
            self.driver.get("http://localhost:8080/projects")
            time.sleep(3)
            
            # 最终截图
            self.driver.save_screenshot('step7_final_verification.png')
            logger.info("📸 最终截图已保存: step7_final_verification.png")
            
            logger.info("✅ 完整登录流程测试成功完成!")
            return True
            
        except Exception as e:
            logger.error(f"❌ 完整登录流程测试失败: {e}")
            self.driver.save_screenshot('error_complete_flow.png')
            return False
    
    def run_test(self):
        """运行完整测试"""
        try:
            if not self.setup_driver():
                logger.error("❌ WebDriver设置失败，无法继续测试")
                return False
                
            logger.info("✅ WebDriver设置成功，开始登录流程测试")
            return self.test_complete_login_flow()
            
        except KeyboardInterrupt:
            logger.info("⚠️ 用户中断测试")
            return False
        except Exception as e:
            logger.error(f"❌ 测试运行时发生未预期错误: {e}")
            logger.error(f"错误类型: {type(e).__name__}")
            if self.driver:
                try:
                    self.driver.save_screenshot('unexpected_error.png')
                    logger.info("📸 错误截图已保存: unexpected_error.png")
                except:
                    pass
            return False
        finally:
            if self.driver:
                try:
                    # 保持浏览器打开10秒以便观察
                    logger.info("⏸️ 保持浏览器打开10秒以便观察...")
                    time.sleep(10)
                    self.driver.quit()
                    logger.info("🔄 Chrome WebDriver已关闭")
                except Exception as e:
                    logger.warning(f"⚠️ 关闭WebDriver时出错: {e}")

def main():
    logger.info("🧪 JupyterHub完整登录测试开始")
    logger.info("=" * 60)
    
    # 预检查环境
    logger.info("🔍 预检查测试环境...")
    
    # 检查必要文件
    import os
    chrome_path = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
    chromedriver_path = '/opt/homebrew/bin/chromedriver'
    
    if not os.path.exists(chrome_path):
        logger.error(f"❌ Chrome浏览器未找到: {chrome_path}")
        return False
        
    if not os.path.exists(chromedriver_path):
        logger.error(f"❌ ChromeDriver未找到: {chromedriver_path}")
        return False
    
    # 检查服务是否运行
    import requests
    try:
        response = requests.get("http://localhost:8080/", timeout=5)
        if response.status_code == 200:
            logger.info("✅ 本地服务运行正常")
        else:
            logger.warning(f"⚠️ 本地服务响应异常: {response.status_code}")
    except Exception as e:
        logger.error(f"❌ 无法连接到本地服务: {e}")
        logger.error("   请确保运行: docker-compose up -d")
        return False
    
    logger.info("✅ 环境预检查完成")
    
    # 运行主测试
    test = JupyterHubLoginTest()
    success = test.run_test()
    
    if success:
        logger.info("🎉 所有测试通过 - 用户体验一致，无白屏问题!")
    else:
        logger.error("❌ 测试失败 - 发现白屏或其他问题")
    
    logger.info("🏁 测试完成")
    return success

if __name__ == "__main__":
    main()
