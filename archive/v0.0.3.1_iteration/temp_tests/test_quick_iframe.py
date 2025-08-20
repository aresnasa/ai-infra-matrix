#!/usr/bin/env python3
"""
快速iframe测试脚本
检查JupyterHub iframe的基本功能
"""

import time
import logging
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def quick_iframe_test():
    """快速测试iframe是否能正常加载"""
    
    chrome_options = Options()
    chrome_options.add_argument('--headless')  # 无头模式，更快
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--disable-web-security')
    chrome_options.add_argument('--disable-features=VizDisplayCompositor')
    
    driver = None
    
    try:
        logger.info("🚀 启动Chrome浏览器...")
        driver = webdriver.Chrome(options=chrome_options)
        driver.set_page_load_timeout(30)
        
        # 访问wrapper页面
        url = "http://localhost:8080/jupyterhub"
        logger.info(f"📍 访问: {url}")
        driver.get(url)
        
        # 等待页面加载
        time.sleep(3)
        
        # 检查页面标题
        title = driver.title
        logger.info(f"📄 页面标题: {title}")
        
        # 检查是否有iframe
        try:
            iframe = WebDriverWait(driver, 10).until(
                EC.presence_of_element_located((By.ID, "jupyterhub-frame"))
            )
            logger.info("✅ 找到iframe元素")
            
            # 检查iframe的src属性
            iframe_src = iframe.get_attribute('src')
            logger.info(f"🔗 iframe src: {iframe_src[:100]}...")
            
            # 检查iframe是否可见
            if iframe.is_displayed():
                logger.info("✅ iframe元素可见")
            else:
                logger.warning("⚠️ iframe元素存在但不可见")
                
        except Exception as e:
            logger.error(f"❌ 未找到iframe: {e}")
            
        # 检查页面中的错误元素
        try:
            error_div = driver.find_element(By.ID, "error")
            if "hidden" not in error_div.get_attribute("class"):
                error_msg = driver.find_element(By.ID, "error-message").text
                logger.error(f"❌ 页面显示错误: {error_msg}")
            else:
                logger.info("✅ 没有显示错误信息")
        except:
            logger.info("✅ 没有错误元素")
            
        # 检查加载状态
        try:
            loading_div = driver.find_element(By.ID, "loading")
            if "hidden" not in loading_div.get_attribute("class"):
                logger.info("⏳ 页面仍在加载中")
                time.sleep(5)  # 再等待5秒
            else:
                logger.info("✅ 页面加载完成")
        except:
            logger.info("✅ 没有加载指示器")
            
        # 检查状态指示器
        try:
            status = driver.find_element(By.ID, "status").text
            logger.info(f"📊 状态: {status}")
        except:
            pass
            
        # 获取控制台日志
        logs = driver.get_log('browser')
        if logs:
            logger.info("📝 浏览器控制台日志:")
            for log in logs[-5:]:  # 只显示最后5条
                logger.info(f"   {log['level']}: {log['message']}")
        else:
            logger.info("✅ 没有浏览器控制台错误")
            
        # 最终状态检查
        time.sleep(2)
        
        # 再次检查iframe状态
        try:
            iframe = driver.find_element(By.ID, "jupyterhub-frame")
            if iframe.is_displayed() and iframe.get_attribute('src'):
                logger.info("🎉 iframe测试通过 - 元素存在且可见")
                return True
            else:
                logger.warning("⚠️ iframe存在但可能未正确加载")
                return False
        except:
            logger.error("❌ iframe测试失败 - 元素不存在")
            return False
            
    except Exception as e:
        logger.error(f"❌ 测试过程中出错: {e}")
        return False
        
    finally:
        if driver:
            driver.quit()
            logger.info("🔚 浏览器已关闭")

if __name__ == "__main__":
    logger.info("🧪 开始快速iframe测试")
    logger.info("=" * 50)
    
    success = quick_iframe_test()
    
    logger.info("=" * 50)
    if success:
        logger.info("✅ 测试结果: PASS - iframe功能正常")
    else:
        logger.info("❌ 测试结果: FAIL - iframe存在问题")
        
    logger.info("🏁 测试完成")
