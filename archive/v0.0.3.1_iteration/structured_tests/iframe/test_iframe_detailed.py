#!/usr/bin/env python3
"""
详细的JupyterHub iframe诊断脚本 - 使用brew安装的chromedriver
"""

import time
import logging
import json
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

# 设置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_iframe_detailed():
    """详细的iframe诊断"""
    
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--disable-web-security')
    chrome_options.add_argument('--window-size=1920,1080')
    chrome_options.add_argument('--disable-features=VizDisplayCompositor')
    
    # 启用日志记录
    chrome_options.add_argument('--enable-logging')
    chrome_options.add_argument('--log-level=0')
    chrome_options.add_experimental_option('useAutomationExtension', False)
    chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
    
    driver = None
    
    try:
        logger.info("🚀 启动Chrome浏览器 (使用brew安装的chromedriver)...")
        driver = webdriver.Chrome(options=chrome_options)
        driver.set_page_load_timeout(30)
        
        # 访问测试页面
        logger.info("📍 访问iframe测试页面...")
        driver.get("http://localhost:8080/iframe_test.html")
        
        # 等待页面完全加载
        wait = WebDriverWait(driver, 20)
        wait.until(EC.presence_of_element_located((By.TAG_NAME, "h1")))
        
        page_title = driver.title
        logger.info(f"页面标题: {page_title}")
        
        # 等待iframe加载和JavaScript执行
        logger.info("⏳ 等待iframe加载和JavaScript执行（15秒）...")
        time.sleep(15)
        
        # 检查页面状态元素
        logger.info("🔍 检查页面状态元素...")
        status_elements = driver.find_elements(By.CSS_SELECTOR, "div[id^='status-']")
        for status_elem in status_elements:
            status_id = status_elem.get_attribute("id")
            status_text = status_elem.text
            status_class = status_elem.get_attribute("class")
            logger.info(f"状态元素 {status_id}: {status_text} (class: {status_class})")
        
        # 检查iframe元素
        logger.info("🔍 检查iframe元素...")
        iframes = driver.find_elements(By.TAG_NAME, "iframe")
        logger.info(f"找到 {len(iframes)} 个iframe")
        
        for i, iframe in enumerate(iframes):
            iframe_id = iframe.get_attribute("id")
            iframe_src = iframe.get_attribute("src")
            iframe_sandbox = iframe.get_attribute("sandbox")
            
            logger.info(f"iframe[{i}] id={iframe_id}")
            logger.info(f"  src: {iframe_src}")
            logger.info(f"  sandbox: {iframe_sandbox}")
            
            # 检查iframe尺寸
            rect = iframe.get_rect()
            logger.info(f"  尺寸: {rect['width']}x{rect['height']}")
            
            # 尝试检查iframe内容
            try:
                driver.switch_to.frame(iframe)
                
                # 检查iframe内的页面
                iframe_url = driver.current_url
                iframe_title = driver.title
                iframe_body = driver.find_element(By.TAG_NAME, "body")
                iframe_text = iframe_body.text[:200] if iframe_body.text else "空白"
                
                logger.info(f"  iframe内URL: {iframe_url}")
                logger.info(f"  iframe内标题: {iframe_title}")
                logger.info(f"  iframe内容预览: {iframe_text}")
                
                # 检查是否有错误页面
                if "404" in iframe_text or "Not Found" in iframe_text:
                    logger.warning(f"  ⚠️ iframe显示404错误")
                elif "500" in iframe_text or "Internal Server Error" in iframe_text:
                    logger.warning(f"  ⚠️ iframe显示服务器错误")
                elif len(iframe_text.strip()) < 10:
                    logger.warning(f"  ⚠️ iframe内容很少或为空")
                elif "login" in iframe_text.lower() or "sign in" in iframe_text.lower():
                    logger.info(f"  ✅ iframe显示登录页面")
                else:
                    logger.info(f"  ✅ iframe有内容")
                
                # 检查是否有表单
                forms = driver.find_elements(By.TAG_NAME, "form")
                if forms:
                    logger.info(f"  找到 {len(forms)} 个表单")
                    for j, form in enumerate(forms):
                        action = form.get_attribute("action")
                        method = form.get_attribute("method")
                        logger.info(f"    表单[{j}] action={action}, method={method}")
                
                driver.switch_to.default_content()
                
            except Exception as e:
                logger.error(f"  ❌ 检查iframe内容失败: {e}")
                driver.switch_to.default_content()
        
        # 获取浏览器控制台日志
        logger.info("📝 获取浏览器控制台日志...")
        try:
            logs = driver.get_log('browser')
            if logs:
                logger.info("控制台日志:")
                for log in logs:
                    level = log['level']
                    message = log['message']
                    timestamp = log['timestamp']
                    logger.info(f"  [{level}] {timestamp}: {message}")
            else:
                logger.info("无控制台日志")
        except Exception as e:
            logger.warning(f"获取控制台日志失败: {e}")
        
        # 检查网络请求
        logger.info("🌐 检查网络请求...")
        try:
            performance_logs = driver.get_log('performance')
            network_errors = []
            successful_requests = []
            
            for log in performance_logs:
                message = log.get('message', '')
                if isinstance(message, str):
                    try:
                        message_data = json.loads(message)
                        method = message_data.get('message', {}).get('method', '')
                        
                        if method == 'Network.responseReceived':
                            params = message_data.get('message', {}).get('params', {})
                            response = params.get('response', {})
                            status = response.get('status', 0)
                            url = response.get('url', '')
                            
                            if 'jupyter' in url or 'hub' in url:
                                if status >= 400:
                                    network_errors.append(f"{status} - {url}")
                                else:
                                    successful_requests.append(f"{status} - {url}")
                    except:
                        continue
            
            if successful_requests:
                logger.info("成功的JupyterHub相关请求:")
                for req in successful_requests[:5]:
                    logger.info(f"  ✅ {req}")
            
            if network_errors:
                logger.warning("失败的JupyterHub相关请求:")
                for error in network_errors[:5]:
                    logger.warning(f"  ❌ {error}")
                    
        except Exception as e:
            logger.warning(f"检查网络请求失败: {e}")
        
        # 测试直接访问
        logger.info("🔄 测试直接访问JupyterHub页面...")
        test_urls = [
            "http://localhost:8080/jupyter/hub/",
            "http://localhost:8080/jupyter/hub/login",
            "http://localhost:8080/jupyterhub"
        ]
        
        for url in test_urls:
            try:
                logger.info(f"访问: {url}")
                driver.get(url)
                time.sleep(3)
                
                current_url = driver.current_url
                page_title = driver.title
                page_content = driver.find_element(By.TAG_NAME, "body").text[:100]
                
                logger.info(f"  当前URL: {current_url}")
                logger.info(f"  页面标题: {page_title}")
                logger.info(f"  内容预览: {page_content}")
                
                if "404" in page_content or "Not Found" in page_content:
                    logger.warning(f"  ❌ 页面返回404")
                elif "500" in page_content:
                    logger.warning(f"  ❌ 页面返回服务器错误")
                elif len(page_content.strip()) < 10:
                    logger.warning(f"  ⚠️ 页面内容很少")
                else:
                    logger.info(f"  ✅ 页面正常")
                
            except Exception as e:
                logger.error(f"  ❌ 访问失败: {e}")
        
        # 保持浏览器打开以便观察
        logger.info("⏸️ 保持浏览器打开20秒以便观察...")
        driver.get("http://localhost:8080/iframe_test.html")
        time.sleep(20)
        
        return True
        
    except Exception as e:
        logger.error(f"❌ 测试失败: {e}")
        return False
        
    finally:
        if driver:
            try:
                driver.save_screenshot('iframe_detailed_debug.png')
                logger.info("📸 详细截图已保存: iframe_detailed_debug.png")
            except:
                pass
            driver.quit()

if __name__ == "__main__":
    logger.info("🧪 详细iframe诊断测试")
    logger.info("=" * 60)
    test_iframe_detailed()
    logger.info("🏁 诊断完成")
