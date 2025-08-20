#!/usr/bin/env python3
"""
专门测试JupyterHub iframe白屏问题的Chrome WebDriver脚本
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

def test_iframe_functionality():
    """测试iframe白屏问题"""
    
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--disable-web-security')
    chrome_options.add_argument('--disable-features=VizDisplayCompositor')
    chrome_options.add_argument('--window-size=1920,1080')
    chrome_options.add_argument('--allow-running-insecure-content')
    chrome_options.add_argument('--disable-extensions')
    # 不使用headless模式，方便观察
    
    driver = None
    
    try:
        logger.info("🚀 启动Chrome浏览器...")
        service = Service(ChromeDriverManager().install())
        driver = webdriver.Chrome(service=service, options=chrome_options)
        driver.set_page_load_timeout(30)
        
        # 测试1: 访问iframe测试页面
        logger.info("📍 访问iframe测试页面...")
        driver.get("http://localhost:8080/iframe_test.html")
        
        # 等待页面加载
        wait = WebDriverWait(driver, 20)
        wait.until(EC.presence_of_element_located((By.TAG_NAME, "h1")))
        
        page_title = driver.title
        logger.info(f"页面标题: {page_title}")
        
        # 等待iframe加载
        logger.info("⏳ 等待iframe加载（10秒）...")
        time.sleep(10)
        
        # 检查第一个iframe（直接嵌入）
        logger.info("🔍 检查第一个iframe（直接嵌入 /jupyter/hub/）...")
        iframe1 = driver.find_element(By.ID, "test-frame-1")
        iframe1_src = iframe1.get_attribute("src")
        logger.info(f"iframe1 src: {iframe1_src}")
        
        # 切换到第一个iframe
        try:
            driver.switch_to.frame(iframe1)
            iframe1_body = driver.find_element(By.TAG_NAME, "body")
            iframe1_content = iframe1_body.text
            iframe1_html_length = len(driver.page_source)
            
            logger.info(f"iframe1 内容长度: {iframe1_html_length}")
            logger.info(f"iframe1 文本内容前200字符: {iframe1_content[:200]}")
            
            if iframe1_html_length > 100:
                logger.info("✅ iframe1 有内容")
            else:
                logger.warning("⚠️ iframe1 内容很少，可能为空")
                
            # 检查是否有JupyterHub相关内容
            if 'jupyter' in driver.page_source.lower() or 'hub' in driver.page_source.lower():
                logger.info("✅ iframe1 包含JupyterHub相关内容")
            else:
                logger.warning("⚠️ iframe1 不包含JupyterHub相关内容")
                
            driver.switch_to.default_content()
            
        except Exception as e:
            logger.error(f"❌ 检查iframe1失败: {e}")
            driver.switch_to.default_content()
        
        # 检查第二个iframe（wrapper页面）
        logger.info("🔍 检查第二个iframe（wrapper页面 /jupyterhub）...")
        iframe2 = driver.find_element(By.ID, "test-frame-2")
        iframe2_src = iframe2.get_attribute("src")
        logger.info(f"iframe2 src: {iframe2_src}")
        
        # 切换到第二个iframe
        try:
            driver.switch_to.frame(iframe2)
            iframe2_body = driver.find_element(By.TAG_NAME, "body")
            iframe2_content = iframe2_body.text
            iframe2_html_length = len(driver.page_source)
            
            logger.info(f"iframe2 内容长度: {iframe2_html_length}")
            logger.info(f"iframe2 文本内容前200字符: {iframe2_content[:200]}")
            
            if iframe2_html_length > 100:
                logger.info("✅ iframe2 有内容")
            else:
                logger.warning("⚠️ iframe2 内容很少，可能为空")
                
            # 检查是否有JupyterHub相关内容
            if 'jupyter' in driver.page_source.lower() or 'hub' in driver.page_source.lower():
                logger.info("✅ iframe2 包含JupyterHub相关内容")
            else:
                logger.warning("⚠️ iframe2 不包含JupyterHub相关内容")
                
            driver.switch_to.default_content()
            
        except Exception as e:
            logger.error(f"❌ 检查iframe2失败: {e}")
            driver.switch_to.default_content()
        
        # 获取浏览器控制台日志
        logger.info("📝 获取浏览器控制台日志...")
        logs = driver.get_log('browser')
        if logs:
            logger.info("控制台日志:")
            for log in logs:
                level = log['level']
                message = log['message']
                logger.info(f"  [{level}] {message}")
        else:
            logger.info("无控制台日志")
            
        # 测试2: 直接访问/jupyterhub页面
        logger.info("🔄 测试直接访问/jupyterhub页面...")
        driver.get("http://localhost:8080/jupyterhub")
        time.sleep(5)
        
        page_title2 = driver.title
        page_content_length = len(driver.page_source)
        logger.info(f"直接访问/jupyterhub页面标题: {page_title2}")
        logger.info(f"页面内容长度: {page_content_length}")
        
        # 检查页面是否有iframe
        iframes_in_jupyterhub = driver.find_elements(By.TAG_NAME, "iframe")
        logger.info(f"在/jupyterhub页面找到 {len(iframes_in_jupyterhub)} 个iframe")
        
        for i, iframe in enumerate(iframes_in_jupyterhub):
            src = iframe.get_attribute("src")
            logger.info(f"  iframe[{i}] src: {src}")
            
        # 测试3: 直接访问/jupyter/hub/
        logger.info("🔄 测试直接访问/jupyter/hub/页面...")
        driver.get("http://localhost:8080/jupyter/hub/")
        time.sleep(5)
        
        page_title3 = driver.title
        page_content_length3 = len(driver.page_source)
        logger.info(f"直接访问/jupyter/hub/页面标题: {page_title3}")
        logger.info(f"页面内容长度: {page_content_length3}")
        
        # 检查是否是登录页面
        if 'login' in driver.page_source.lower() or 'sign in' in driver.page_source.lower():
            logger.info("✅ 检测到JupyterHub登录页面")
        elif 'jupyter' in driver.page_source.lower():
            logger.info("✅ 检测到JupyterHub相关页面")
        else:
            logger.warning("⚠️ 页面不像JupyterHub页面")
            
        # 获取更多诊断信息
        logger.info("📋 获取更多诊断信息...")
        current_url = driver.current_url
        logger.info(f"当前URL: {current_url}")
        
        # 检查网络错误
        performance_logs = driver.get_log('performance')
        network_errors = []
        for log in performance_logs:
            message = log.get('message', {})
            if isinstance(message, str):
                import json
                try:
                    message = json.loads(message)
                except:
                    continue
                    
            method = message.get('message', {}).get('method', '')
            if method == 'Network.responseReceived':
                response = message.get('message', {}).get('params', {}).get('response', {})
                status = response.get('status', 0)
                url = response.get('url', '')
                if status >= 400:
                    network_errors.append(f"{status} - {url}")
                    
        if network_errors:
            logger.warning("⚠️ 发现网络错误:")
            for error in network_errors[:5]:  # 只显示前5个
                logger.warning(f"  {error}")
        else:
            logger.info("✅ 未发现明显的网络错误")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ 测试过程中出错: {e}")
        return False
        
    finally:
        if driver:
            # 截图用于调试
            try:
                driver.save_screenshot('iframe_test_debug.png')
                logger.info("📸 调试截图已保存: iframe_test_debug.png")
            except:
                pass
            
            logger.info("⏸️ 保持浏览器打开10秒以便观察...")
            time.sleep(10)
            driver.quit()

if __name__ == "__main__":
    logger.info("🚀 开始JupyterHub iframe白屏问题诊断")
    logger.info("=" * 60)
    
    success = test_iframe_functionality()
    
    logger.info("=" * 60)
    if success:
        logger.info("🎉 测试完成，请查看日志了解详细情况")
    else:
        logger.info("❌ 测试遇到问题，请查看错误日志")
        
    logger.info("🏁 诊断完成")
