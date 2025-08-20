#!/usr/bin/env python3
"""
简化的Chrome WebDriver测试iframe功能
"""

import time
import logging
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

# 设置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_iframe_quick():
    """快速测试iframe功能"""
    
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--disable-web-security')
    chrome_options.add_argument('--window-size=1920,1080')
    
    driver = None
    
    try:
        logger.info("🚀 启动Chrome浏览器...")
        # 使用brew安装的chromedriver
        driver = webdriver.Chrome(options=chrome_options)
        driver.set_page_load_timeout(30)
        
        # 访问测试页面
        logger.info("📍 访问iframe测试页面...")
        driver.get("http://localhost:8080/iframe_test.html")
        
        # 等待页面加载
        time.sleep(10)
        
        page_title = driver.title
        logger.info(f"页面标题: {page_title}")
        
        # 获取浏览器控制台日志
        logs = driver.get_log('browser')
        if logs:
            logger.info("📝 浏览器控制台日志:")
            for log in logs:
                level = log['level']
                message = log['message']
                logger.info(f"  [{level}] {message}")
        
        # 检查iframe状态
        iframes = driver.find_elements(By.TAG_NAME, "iframe")
        logger.info(f"找到 {len(iframes)} 个iframe")
        
        for i, iframe in enumerate(iframes):
            iframe_id = iframe.get_attribute("id")
            iframe_src = iframe.get_attribute("src")
            logger.info(f"iframe[{i}] id={iframe_id}, src={iframe_src}")
            
            # 检查iframe是否加载
            rect = iframe.get_rect()
            logger.info(f"iframe[{i}] 尺寸: {rect['width']}x{rect['height']}")
        
        # 保持浏览器打开以便观察
        logger.info("⏸️ 保持浏览器打开15秒以便观察...")
        time.sleep(15)
        
        return True
        
    except Exception as e:
        logger.error(f"❌ 测试失败: {e}")
        return False
        
    finally:
        if driver:
            driver.save_screenshot('iframe_test_quick.png')
            logger.info("📸 截图已保存: iframe_test_quick.png")
            driver.quit()

if __name__ == "__main__":
    logger.info("🧪 快速iframe测试")
    test_iframe_quick()
