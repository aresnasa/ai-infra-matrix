#!/usr/bin/env python3
"""
简化的Chrome WebDriver测试
"""

import time
import logging
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service

# 设置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_chrome_simple():
    """简化的Chrome测试"""
    
    logger.info("🧪 开始简化的Chrome WebDriver测试")
    
    # Chrome选项
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--disable-web-security')
    chrome_options.add_argument('--disable-extensions')
    chrome_options.add_argument('--disable-gpu')
    chrome_options.add_argument('--window-size=1920,1080')
    
    # 显式指定Chrome可执行文件路径
    chrome_options.binary_location = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
    
    driver = None
    
    try:
        # 创建Service对象指定chromedriver路径
        service = Service('/opt/homebrew/bin/chromedriver')
        
        logger.info("🚀 尝试启动Chrome WebDriver...")
        logger.info(f"Chrome位置: {chrome_options.binary_location}")
        logger.info(f"ChromeDriver路径: /opt/homebrew/bin/chromedriver")
        
        driver = webdriver.Chrome(service=service, options=chrome_options)
        logger.info("✅ Chrome WebDriver启动成功！")
        
        # 设置超时
        driver.set_page_load_timeout(30)
        
        # 测试访问页面
        logger.info("📍 测试访问本地页面...")
        driver.get("http://localhost:8080/")
        
        page_title = driver.title
        current_url = driver.current_url
        
        logger.info(f"页面标题: {page_title}")
        logger.info(f"当前URL: {current_url}")
        
        # 保持浏览器打开5秒以便观察
        logger.info("⏸️ 保持浏览器打开5秒...")
        time.sleep(5)
        
        logger.info("✅ 测试完成")
        return True
        
    except Exception as e:
        logger.error(f"❌ Chrome WebDriver启动失败: {e}")
        logger.error(f"错误类型: {type(e).__name__}")
        return False
        
    finally:
        if driver:
            try:
                driver.quit()
                logger.info("🔄 Chrome WebDriver已关闭")
            except:
                pass

if __name__ == "__main__":
    test_chrome_simple()
