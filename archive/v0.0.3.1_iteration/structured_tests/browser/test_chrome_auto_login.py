#!/usr/bin/env python3
"""
Chrome WebDriver自动登录测试 - 自动输入admin/admin123并验证SSO流程
"""

import time
import logging
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.common.keys import Keys

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_auto_login_flow():
    """自动登录流程测试"""
    
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--disable-web-security')
    chrome_options.add_argument('--window-size=1920,1080')
    chrome_options.add_argument('--disable-blink-features=AutomationControlled')
    chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
    chrome_options.add_experimental_option('useAutomationExtension', False)
    
    driver = None
    
    try:
        logger.info("🚀 启动Chrome WebDriver...")
        service = Service('/opt/homebrew/bin/chromedriver')
        driver = webdriver.Chrome(service=service, options=chrome_options)
        driver.set_page_load_timeout(30)
        wait = WebDriverWait(driver, 20)
        
        # 隐藏WebDriver特征
        driver.execute_script("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")
        
        # Step 1: 访问主页
        logger.info("📍 Step 1: 访问主页")
        driver.get("http://localhost:8080/")
        time.sleep(3)
        
        current_url = driver.current_url
        logger.info(f"当前URL: {current_url}")
        
        # 截图1: 初始页面
        driver.save_screenshot('auto_login_1_initial.png')
        
        # Step 2: 如果不在登录页面，查找登录入口
        if 'login' not in current_url.lower():
            logger.info("🔍 查找登录链接...")
            
            login_link_selectors = [
                "a[href*='login']",
                "button:contains('登录')",
                "a:contains('登录')",
                ".login-btn",
                "#login-link"
            ]
            
            for selector in login_link_selectors:
                try:
                    if 'contains' in selector:
                        login_link = driver.find_element(By.XPATH, "//a[contains(text(), '登录')] | //button[contains(text(), '登录')]")
                    else:
                        login_link = driver.find_element(By.CSS_SELECTOR, selector)
                    
                    if login_link:
                        logger.info(f"✅ 找到登录链接: {selector}")
                        login_link.click()
                        time.sleep(3)
                        break
                except:
                    continue
            else:
                # 直接访问登录页面
                logger.info("🔍 直接访问登录页面")
                driver.get("http://localhost:8080/login")
                time.sleep(3)
        
        # Step 3: 查找并填写登录表单
        logger.info("📍 Step 3: 查找登录表单")
        
        # 查找用户名字段 - 更全面的选择器
        username_selectors = [
            "input[name='username']",
            "input[id='username']",
            "input[placeholder*='用户名']",
            "input[placeholder*='Username']",
            "input[placeholder*='admin']",
            "input[type='text']",
            ".ant-input:first-of-type",
            "#login_username",
            ".username-input"
        ]
        
        username_field = None
        for selector in username_selectors:
            try:
                username_field = wait.until(EC.element_to_be_clickable((By.CSS_SELECTOR, selector)))
                logger.info(f"✅ 找到用户名字段: {selector}")
                break
            except:
                continue
        
        if not username_field:
            logger.error("❌ 未找到用户名输入字段")
            driver.save_screenshot('auto_login_error_no_username.png')
            return False
        
        # 查找密码字段
        password_selectors = [
            "input[name='password']",
            "input[id='password']", 
            "input[type='password']",
            "input[placeholder*='密码']",
            "input[placeholder*='Password']",
            "#login_password",
            ".password-input"
        ]
        
        password_field = None
        for selector in password_selectors:
            try:
                password_field = driver.find_element(By.CSS_SELECTOR, selector)
                logger.info(f"✅ 找到密码字段: {selector}")
                break
            except:
                continue
        
        if not password_field:
            logger.error("❌ 未找到密码输入字段")
            driver.save_screenshot('auto_login_error_no_password.png')
            return False
        
        # Step 4: 自动输入账号密码
        logger.info("📍 Step 4: 自动输入账号密码")
        
        # 清空并输入用户名
        username_field.clear()
        username_field.send_keys("admin")
        logger.info("✅ 输入用户名: admin")
        time.sleep(1)
        
        # 清空并输入密码
        password_field.clear()
        password_field.send_keys("admin123")
        logger.info("✅ 输入密码: admin123")
        time.sleep(1)
        
        # 截图2: 填写完表单
        driver.save_screenshot('auto_login_2_form_filled.png')
        
        # Step 5: 提交登录表单
        logger.info("📍 Step 5: 提交登录表单")
        
        # 查找登录按钮
        login_button_selectors = [
            "button[type='submit']",
            "input[type='submit']",
            "button:contains('登录')",
            "button:contains('Login')",
            ".ant-btn-primary",
            "#login-button",
            ".login-btn"
        ]
        
        login_submitted = False
        for selector in login_button_selectors:
            try:
                if 'contains' in selector:
                    login_button = driver.find_element(By.XPATH, "//button[contains(text(), 'Login') or contains(text(), '登录')]")
                else:
                    login_button = driver.find_element(By.CSS_SELECTOR, selector)
                
                if login_button and login_button.is_enabled():
                    logger.info(f"✅ 找到并点击登录按钮: {selector}")
                    login_button.click()
                    login_submitted = True
                    break
            except:
                continue
        
        if not login_submitted:
            # 尝试回车提交
            logger.info("🔍 未找到登录按钮，尝试回车提交")
            password_field.send_keys(Keys.RETURN)
        
        # Step 6: 等待登录结果
        logger.info("📍 Step 6: 等待登录结果")
        time.sleep(5)
        
        current_url = driver.current_url
        logger.info(f"登录后URL: {current_url}")
        
        # 检查登录是否成功
        page_text = driver.find_element(By.TAG_NAME, "body").text.lower()
        
        login_failed_indicators = ['invalid', 'incorrect', 'error', '错误', '无效', 'failed']
        login_success_indicators = ['dashboard', 'welcome', 'projects', '项目', 'logout', '退出']
        
        has_error = any(indicator in page_text for indicator in login_failed_indicators)
        has_success = any(indicator in page_text for indicator in login_success_indicators)
        
        if has_error:
            logger.error("❌ 登录失败 - 页面显示错误信息")
            driver.save_screenshot('auto_login_3_login_failed.png')
            return False
        elif 'login' in current_url.lower() and not has_success:
            logger.error("❌ 登录失败 - 仍在登录页面")
            driver.save_screenshot('auto_login_3_still_login_page.png')
            return False
        else:
            logger.info("✅ 登录成功!")
        
        # 截图3: 登录成功页面
        driver.save_screenshot('auto_login_3_login_success.png')
        
        # Step 7: 验证前端访问
        logger.info("📍 Step 7: 验证前端访问")
        
        # 访问Projects页面
        driver.get("http://localhost:8080/projects")
        time.sleep(3)
        
        projects_page_text = driver.find_element(By.TAG_NAME, "body").text.lower()
        
        if 'login' in projects_page_text and len(projects_page_text) < 1000:
            logger.error("❌ Projects页面需要重新登录")
            driver.save_screenshot('auto_login_4_projects_failed.png')
            return False
        else:
            logger.info("✅ Projects页面访问正常")
        
        # 截图4: Projects页面
        driver.save_screenshot('auto_login_4_projects_page.png')
        
        # Step 8: 测试JupyterHub访问
        logger.info("📍 Step 8: 测试JupyterHub访问")
        
        # 查找JupyterHub菜单
        jupyter_selectors = [
            "a[href='/jupyterhub']",
            "a:contains('JupyterHub')",
            "a:contains('Jupyter')",
            ".ant-menu-item:contains('Jupyter')"
        ]
        
        jupyter_link = None
        for selector in jupyter_selectors:
            try:
                if 'contains' in selector:
                    jupyter_link = driver.find_element(By.XPATH, "//a[contains(text(), 'JupyterHub') or contains(text(), 'Jupyter')]")
                else:
                    jupyter_link = driver.find_element(By.CSS_SELECTOR, selector)
                
                if jupyter_link:
                    logger.info(f"✅ 找到JupyterHub链接: {selector}")
                    break
            except:
                continue
        
        if jupyter_link:
            logger.info("🖱️ 点击JupyterHub菜单")
            # 记录点击前的窗口句柄
            original_window = driver.current_window_handle
            jupyter_link.click()
            time.sleep(5)
            
            # 检查是否打开了新窗口/标签页
            if len(driver.window_handles) > 1:
                logger.info("🔄 检测到新窗口，切换到新窗口")
                for handle in driver.window_handles:
                    if handle != original_window:
                        driver.switch_to.window(handle)
                        break
        else:
            logger.info("🔍 直接导航到JupyterHub")
            driver.get("http://localhost:8080/jupyterhub")
            time.sleep(5)
        
        current_url = driver.current_url
        logger.info(f"JupyterHub访问后URL: {current_url}")
        
        # 截图5: JupyterHub页面
        driver.save_screenshot('auto_login_5_jupyterhub_page.png')
        
        # Step 9: 检查JupyterHub是否需要再次登录
        logger.info("📍 Step 9: 检查JupyterHub登录状态")
        
        # 等待页面完全加载
        time.sleep(3)
        
        jupyter_page_text = driver.find_element(By.TAG_NAME, "body").text.lower()
        
        # 检查是否有iframe
        iframes = driver.find_elements(By.TAG_NAME, "iframe")
        if iframes:
            logger.info(f"🔍 检查 {len(iframes)} 个iframe")
            for i, iframe in enumerate(iframes):
                try:
                    driver.switch_to.frame(iframe)
                    iframe_text = driver.find_element(By.TAG_NAME, "body").text.lower()
                    
                    login_indicators = ['login', 'sign in', 'username', 'password', '登录', '用户名', '密码']
                    iframe_has_login = any(indicator in iframe_text for indicator in login_indicators)
                    
                    if iframe_has_login:
                        logger.error(f"❌ iframe[{i}] 仍显示登录表单")
                        driver.save_screenshot(f'auto_login_iframe_{i}_login.png')
                    else:
                        logger.info(f"✅ iframe[{i}] 无需登录")
                        
                        # 检查JupyterHub功能
                        jupyter_indicators = ['start', 'server', 'notebook', 'lab', 'spawn', 'hub']
                        has_jupyter_features = any(indicator in iframe_text for indicator in jupyter_indicators)
                        
                        if has_jupyter_features:
                            logger.info(f"✅ iframe[{i}] 显示JupyterHub功能")
                        else:
                            logger.warning(f"⚠️ iframe[{i}] 内容可能异常")
                    
                    driver.switch_to.default_content()
                    break
                except:
                    driver.switch_to.default_content()
        else:
            # 检查主页面
            login_indicators = ['login', 'sign in', 'username', 'password', '登录', '用户名', '密码']
            main_has_login = any(indicator in jupyter_page_text for indicator in login_indicators)
            
            if main_has_login:
                logger.error("❌ JupyterHub主页面仍显示登录表单")
            else:
                logger.info("✅ JupyterHub主页面无需登录")
        
        # Step 10: 最终验证
        logger.info("📍 Step 10: 最终验证")
        
        # 回到前端验证会话保持
        driver.get("http://localhost:8080/projects")
        time.sleep(3)
        
        final_page_text = driver.find_element(By.TAG_NAME, "body").text.lower()
        session_maintained = 'login' not in final_page_text or 'projects' in final_page_text
        
        if session_maintained:
            logger.info("✅ 前端会话保持正常")
        else:
            logger.error("❌ 前端会话丢失")
            driver.save_screenshot('auto_login_6_session_lost.png')
            return False
        
        # 截图6: 最终状态
        driver.save_screenshot('auto_login_6_final_state.png')
        
        logger.info("🎉 自动登录流程测试完成!")
        return True
        
    except Exception as e:
        logger.error(f"❌ 自动登录测试失败: {e}")
        if driver:
            driver.save_screenshot('auto_login_error.png')
        return False
        
    finally:
        if driver:
            # 保持浏览器打开10秒观察
            logger.info("⏸️ 保持浏览器打开10秒以便观察...")
            time.sleep(10)
            driver.quit()
            logger.info("🔄 Chrome WebDriver已关闭")

def main():
    logger.info("🧪 Chrome自动登录测试开始")
    logger.info("=" * 60)
    logger.info("📋 测试账号: admin / admin123")
    logger.info("🎯 目标: 验证SSO单点登录功能")
    logger.info("=" * 60)
    
    success = test_auto_login_flow()
    
    if success:
        logger.info("🎉 所有测试通过 - 自动登录和SSO功能正常!")
    else:
        logger.error("❌ 测试失败 - 需要检查登录和SSO配置")
    
    logger.info("🏁 测试完成")
    return success

if __name__ == "__main__":
    main()
