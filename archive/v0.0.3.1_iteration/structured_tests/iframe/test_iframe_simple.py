#!/usr/bin/env python3
"""
极简iframe测试 - 快速诊断
"""

import time
import logging
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By

# 设置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_iframe_simple():
    """极简iframe测试"""
    
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--disable-web-security')
    chrome_options.add_argument('--window-size=1920,1080')
    
    driver = None
    
    try:
        logger.info("🚀 启动Chrome...")
        driver = webdriver.Chrome(options=chrome_options)
        driver.set_page_load_timeout(15)
        
        # 访问iframe测试页
        logger.info("📍 访问iframe测试页...")
        driver.get("http://localhost:8080/iframe_test.html")
        
        time.sleep(5)  # 等待5秒加载
        
        # 检查页面标题
        title = driver.title
        logger.info(f"页面标题: {title}")
        
        # 查找iframe
        iframes = driver.find_elements(By.TAG_NAME, "iframe")
        logger.info(f"找到 {len(iframes)} 个iframe")
        
        for i, iframe in enumerate(iframes):
            src = iframe.get_attribute("src")
            logger.info(f"iframe[{i}] src: {src}")
            
            try:
                # 切换到iframe并检查内容
                driver.switch_to.frame(iframe)
                body_text = driver.find_element(By.TAG_NAME, "body").text[:100]
                logger.info(f"iframe[{i}] 内容: {body_text[:50]}...")
                
                if len(body_text.strip()) < 5:
                    logger.warning(f"iframe[{i}] ⚠️ 内容为空或很少")
                elif "login" in body_text.lower():
                    logger.info(f"iframe[{i}] ✅ 显示登录页面")
                else:
                    logger.info(f"iframe[{i}] ✅ 有内容")
                
                driver.switch_to.default_content()
                
            except Exception as e:
                logger.error(f"iframe[{i}] ❌ 检查失败: {e}")
                driver.switch_to.default_content()
        
        # 测试直接访问JupyterHub
        logger.info("🔄 测试直接访问JupyterHub...")
        driver.get("http://localhost:8080/jupyter/hub/login")
        time.sleep(3)
        
        direct_title = driver.title
        direct_text = driver.find_element(By.TAG_NAME, "body").text[:100]
        logger.info(f"直接访问标题: {direct_title}")
        logger.info(f"直接访问内容: {direct_text[:50]}...")
        
        # 保存截图
        screenshot_path = 'iframe_simple_test.png'
        driver.save_screenshot(screenshot_path)
        logger.info(f"📸 截图已保存: {screenshot_path}")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ 测试失败: {e}")
        return False
        
    finally:
        if driver:
            driver.quit()

if __name__ == "__main__":
    logger.info("🧪 极简iframe测试开始")
    test_iframe_simple()
    logger.info("🏁 测试完成")
