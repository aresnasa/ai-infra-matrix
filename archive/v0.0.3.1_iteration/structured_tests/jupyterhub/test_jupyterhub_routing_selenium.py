#!/usr/bin/env python3
"""
JupyterHub路由修复验证测试

验证从projects页面点击JupyterHub菜单的完整流程
"""

import time
import logging
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException, NoSuchElementException
from selenium.webdriver.chrome.service import Service
from webdriver_manager.chrome import ChromeDriverManager

# 设置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_jupyterhub_routing_fix():
    """测试JupyterHub路由修复效果"""
    
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--disable-web-security')
    chrome_options.add_argument('--disable-features=VizDisplayCompositor')
    chrome_options.add_argument('--window-size=1920,1080')
    # chrome_options.add_argument('--headless')  # 取消注释以无头模式运行
    
    driver = None
    
    try:
        logger.info("🚀 启动Chrome浏览器...")
        service = Service(ChromeDriverManager().install())
        driver = webdriver.Chrome(service=service, options=chrome_options)
        driver.set_page_load_timeout(30)
        
        # Step 1: 访问主页并重定向到projects
        logger.info("📍 访问主页...")
        driver.get("http://localhost:8080/")
        
        # 等待重定向到projects页面
        wait = WebDriverWait(driver, 20)
        
        # 检查是否需要登录
        current_url = driver.current_url
        logger.info(f"当前URL: {current_url}")
        
        # 如果在登录页面，输入凭据
        if '/login' in current_url or 'login' in driver.page_source.lower():
            logger.info("🔐 检测到登录页面，尝试登录...")
            
            try:
                # 查找用户名和密码字段
                username_field = wait.until(EC.presence_of_element_located((By.NAME, "username")))
                password_field = driver.find_element(By.NAME, "password")
                
                # 输入测试凭据
                username_field.send_keys("admin")
                password_field.send_keys("admin123")
                
                # 点击登录按钮
                login_button = driver.find_element(By.XPATH, "//button[@type='submit']")
                login_button.click()
                
                # 等待登录成功并重定向
                wait.until(lambda d: '/projects' in d.current_url)
                logger.info("✅ 登录成功")
                
            except Exception as e:
                logger.warning(f"⚠️ 登录过程出现问题: {e}")
                logger.info("继续测试，可能已经登录或不需要登录")
        
        # Step 2: 确保在projects页面
        if '/projects' not in driver.current_url:
            logger.info("🔄 手动导航到projects页面...")
            driver.get("http://localhost:8080/projects")
        
        wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))
        logger.info(f"✅ 成功访问projects页面: {driver.current_url}")
        
        # Step 3: 查找JupyterHub菜单项
        logger.info("🔍 查找JupyterHub菜单项...")
        
        # 尝试多种选择器来找到JupyterHub菜单
        jupyter_selectors = [
            "//span[text()='JupyterHub']",
            "//div[contains(text(), 'JupyterHub')]",
            "//*[contains(@class, 'menu') and contains(text(), 'JupyterHub')]",
            "//li[contains(@data-menu-id, 'jupyterhub')]",
            "//a[contains(@href, 'jupyterhub')]",
            "//*[contains(@key, '/jupyterhub')]"
        ]
        
        jupyter_menu = None
        for selector in jupyter_selectors:
            try:
                element = driver.find_element(By.XPATH, selector)
                if element and element.is_displayed():
                    jupyter_menu = element
                    logger.info(f"✅ 找到JupyterHub菜单: {selector}")
                    break
            except:
                continue
        
        if not jupyter_menu:
            logger.error("❌ 未找到JupyterHub菜单项")
            
            # 尝试查找顶部导航栏
            try:
                nav_bar = driver.find_element(By.TAG_NAME, "nav")
                logger.info("找到导航栏，查看其内容...")
                nav_text = nav_bar.text
                logger.info(f"导航栏内容: {nav_text}")
                
                # 如果导航栏中有JupyterHub文本，尝试点击
                if 'JupyterHub' in nav_text:
                    clickable_elements = nav_bar.find_elements(By.XPATH, ".//*[contains(text(), 'JupyterHub')]")
                    if clickable_elements:
                        jupyter_menu = clickable_elements[0]
                        logger.info("✅ 在导航栏中找到JupyterHub")
                    
            except Exception as e:
                logger.warning(f"⚠️ 查找导航栏失败: {e}")
            
            # 最后尝试：查找所有包含JupyterHub的元素
            try:
                all_elements = driver.find_elements(By.XPATH, "//*[contains(text(), 'JupyterHub')]")
                for elem in all_elements:
                    if elem.is_displayed():
                        jupyter_menu = elem
                        logger.info(f"✅ 找到可见的JupyterHub元素: {elem.tag_name}")
                        break
            except:
                pass
        
        if not jupyter_menu:
            logger.error("❌ 无法找到JupyterHub菜单，测试失败")
            return False
        
        # Step 4: 点击JupyterHub菜单
        logger.info("🖱️ 点击JupyterHub菜单...")
        
        # 记录点击前的URL
        before_click_url = driver.current_url
        logger.info(f"点击前URL: {before_click_url}")
        
        # 点击菜单
        driver.execute_script("arguments[0].click();", jupyter_menu)
        
        # 等待页面变化
        time.sleep(3)
        
        # Step 5: 验证是否成功跳转到JupyterHub
        after_click_url = driver.current_url
        logger.info(f"点击后URL: {after_click_url}")
        
        if '/jupyterhub' in after_click_url:
            logger.info("✅ 成功跳转到JupyterHub路径")
            
            # 检查页面内容
            wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))
            page_title = driver.title
            page_source_snippet = driver.page_source[:500]
            
            logger.info(f"页面标题: {page_title}")
            logger.info(f"页面内容预览: {page_source_snippet}...")
            
            # 检查是否有JupyterHub相关内容
            if 'JupyterHub' in driver.page_source or 'jupyter' in driver.page_source.lower():
                logger.info("✅ 页面包含JupyterHub相关内容")
                
                # 查找iframe
                iframes = driver.find_elements(By.TAG_NAME, "iframe")
                if iframes:
                    logger.info(f"✅ 找到 {len(iframes)} 个iframe")
                    for i, iframe in enumerate(iframes):
                        src = iframe.get_attribute("src")
                        logger.info(f"  iframe[{i}] src: {src}")
                        
                        # 检查iframe是否加载内容
                        try:
                            driver.switch_to.frame(iframe)
                            iframe_body = driver.find_element(By.TAG_NAME, "body")
                            iframe_content_length = len(iframe_body.text)
                            logger.info(f"  iframe[{i}] 内容长度: {iframe_content_length}")
                            
                            if iframe_content_length > 50:
                                logger.info("✅ iframe有实际内容")
                            else:
                                logger.warning("⚠️ iframe内容较少，可能为空")
                            
                            driver.switch_to.default_content()
                            
                        except Exception as e:
                            logger.warning(f"检查iframe[{i}]内容时出错: {e}")
                            driver.switch_to.default_content()
                else:
                    logger.warning("⚠️ 未找到iframe，可能是纯静态页面")
                
                return True
            else:
                logger.error("❌ 页面不包含JupyterHub相关内容")
                return False
        else:
            logger.error("❌ 未能跳转到JupyterHub路径")
            return False
            
    except Exception as e:
        logger.error(f"❌ 测试过程中出错: {e}")
        return False
        
    finally:
        if driver:
            # 截图用于调试
            try:
                driver.save_screenshot('jupyterhub_routing_test.png')
                logger.info("📸 调试截图已保存: jupyterhub_routing_test.png")
            except:
                pass
            
            time.sleep(2)  # 留点时间查看结果
            driver.quit()

if __name__ == "__main__":
    logger.info("🚀 开始JupyterHub路由修复验证测试")
    logger.info("=" * 60)
    
    success = test_jupyterhub_routing_fix()
    
    logger.info("=" * 60)
    if success:
        logger.info("🎉 测试成功！JupyterHub路由修复有效")
        logger.info("✅ 从projects页面点击JupyterHub菜单能够正常跳转")
    else:
        logger.info("❌ 测试失败，可能需要进一步调试")
        
    logger.info("🏁 测试完成")
