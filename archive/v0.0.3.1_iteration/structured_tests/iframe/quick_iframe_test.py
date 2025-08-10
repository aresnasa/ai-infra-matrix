#!/usr/bin/env python3
"""
快速iframe白屏检测脚本
"""

import time
import logging
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

# 设置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def quick_iframe_test():
    """快速iframe白屏检测"""
    
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--window-size=1920,1080')
    chrome_options.binary_location = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
    
    driver = None
    
    try:
        service = Service('/opt/homebrew/bin/chromedriver')
        logger.info("🚀 启动Chrome WebDriver...")
        driver = webdriver.Chrome(service=service, options=chrome_options)
        driver.set_page_load_timeout(20)
        wait = WebDriverWait(driver, 15)
        
        # 1. 访问Projects页面
        logger.info("📍 Step 1: 访问Projects页面")
        driver.get("http://localhost:8080/projects")
        time.sleep(3)
        
        # 2. 查找Jupyter链接
        logger.info("📍 Step 2: 查找Jupyter链接")
        jupyter_link = None
        
        # 尝试不同的选择器
        selectors = [
            "a[href*='jupyter']",
            "a[href='/jupyterhub']", 
            "button[onclick*='jupyter']",
            "*[data-testid*='jupyter']"
        ]
        
        for selector in selectors:
            try:
                elements = driver.find_elements(By.CSS_SELECTOR, selector)
                if elements:
                    jupyter_link = elements[0]
                    logger.info(f"✅ 找到Jupyter链接: {selector}")
                    break
            except:
                continue
        
        if jupyter_link:
            # 3. 点击Jupyter链接
            logger.info("📍 Step 3: 点击Jupyter链接")
            jupyter_link.click()
            time.sleep(5)
        else:
            # 直接导航到JupyterHub
            logger.info("📍 Step 3: 直接导航到JupyterHub")
            driver.get("http://localhost:8080/jupyterhub")
            time.sleep(5)
        
        # 4. 检查是否在iframe环境中
        current_url = driver.current_url
        logger.info(f"当前URL: {current_url}")
        
        if 'projects' in current_url or 'iframe' in current_url:
            logger.info("🔍 检测到iframe环境，检查iframe内容...")
            
            # 查找iframe
            iframes = driver.find_elements(By.TAG_NAME, "iframe")
            logger.info(f"找到 {len(iframes)} 个iframe")
            
            iframe_has_content = False
            
            for i, iframe in enumerate(iframes):
                src = iframe.get_attribute("src")
                logger.info(f"iframe[{i}] src: {src}")
                
                if 'jupyter' in src.lower():
                    try:
                        # 切换到iframe并检查内容
                        driver.switch_to.frame(iframe)
                        
                        # 等待内容加载
                        time.sleep(3)
                        
                        body = driver.find_element(By.TAG_NAME, "body")
                        body_text = body.text.strip()
                        
                        logger.info(f"iframe[{i}] 内容长度: {len(body_text)} 字符")
                        logger.info(f"iframe[{i}] 内容预览: {body_text[:100]}...")
                        
                        if len(body_text) > 50:  # 有足够内容
                            logger.info(f"✅ iframe[{i}] 有内容，无白屏问题")
                            iframe_has_content = True
                        else:
                            logger.warning(f"⚠️ iframe[{i}] 内容较少，可能存在白屏")
                        
                        # 切换回主文档
                        driver.switch_to.default_content()
                        
                    except Exception as e:
                        logger.error(f"❌ iframe[{i}] 检查失败: {e}")
                        driver.switch_to.default_content()
            
            if not iframe_has_content:
                logger.error("❌ 所有iframe都存在白屏问题!")
                driver.save_screenshot('iframe_whitespace_issue.png')
                return False
        else:
            # 直接在JupyterHub页面
            logger.info("🔍 直接在JupyterHub页面，检查内容...")
            body = driver.find_element(By.TAG_NAME, "body")
            body_text = body.text.strip()
            
            if len(body_text) > 50:
                logger.info("✅ JupyterHub页面有内容")
            else:
                logger.warning("⚠️ JupyterHub页面内容较少")
        
        # 5. 保存截图
        driver.save_screenshot('quick_iframe_test_result.png')
        logger.info("📸 测试截图已保存: quick_iframe_test_result.png")
        
        logger.info("✅ 快速iframe测试完成")
        return True
        
    except Exception as e:
        logger.error(f"❌ 快速测试失败: {e}")
        if driver:
            driver.save_screenshot('quick_test_error.png')
        return False
        
    finally:
        if driver:
            driver.quit()

if __name__ == "__main__":
    logger.info("🧪 快速iframe白屏检测开始")
    success = quick_iframe_test()
    
    if success:
        logger.info("🎉 快速测试通过")
    else:
        logger.error("❌ 快速测试发现问题")
    
    logger.info("🏁 快速测试完成")
