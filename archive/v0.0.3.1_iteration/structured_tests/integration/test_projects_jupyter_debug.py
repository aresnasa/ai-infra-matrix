#!/usr/bin/env python3
"""
Projects页面Jupyter图标iframe问题调试
"""

import time
import logging
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.common.exceptions import TimeoutException, NoSuchElementException

# 设置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_projects_to_jupyter_iframe():
    """测试从projects页面访问jupyter iframe的流程"""
    
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--disable-web-security')
    chrome_options.add_argument('--disable-features=VizDisplayCompositor')
    chrome_options.add_argument('--window-size=1920,1080')
    
    driver = None
    
    try:
        logger.info("🚀 启动Chrome浏览器...")
        driver = webdriver.Chrome(options=chrome_options)
        driver.set_page_load_timeout(30)
        
        # Step 1: 访问projects页面
        logger.info("📍 访问projects页面...")
        driver.get("http://localhost:8080/projects")
        
        # 等待页面加载
        wait = WebDriverWait(driver, 20)
        wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))
        
        # 检查当前URL
        current_url = driver.current_url
        logger.info(f"当前页面URL: {current_url}")
        
        # 检查页面标题
        page_title = driver.title
        logger.info(f"页面标题: {page_title}")
        
        # 查找可能的jupyter相关元素
        logger.info("🔍 查找页面中的jupyter相关元素...")
        
        # 尝试多种可能的选择器
        jupyter_selectors = [
            "//button[contains(text(), 'jupyter') or contains(text(), 'Jupyter')]",
            "//a[contains(text(), 'jupyter') or contains(text(), 'Jupyter')]",
            "//div[contains(@class, 'jupyter')]",
            "//span[contains(text(), 'jupyter') or contains(text(), 'Jupyter')]",
            "//*[contains(@title, 'jupyter') or contains(@title, 'Jupyter')]",
            "//img[contains(@alt, 'jupyter')]",
            "//*[@data-testid*='jupyter']",
            "//iframe[contains(@src, 'jupyter')]"
        ]
        
        found_elements = []
        for selector in jupyter_selectors:
            try:
                elements = driver.find_elements(By.XPATH, selector)
                if elements:
                    for element in elements:
                        found_elements.append({
                            'selector': selector,
                            'element': element,
                            'tag': element.tag_name,
                            'text': element.text[:100] if element.text else '',
                            'visible': element.is_displayed()
                        })
                        logger.info(f"找到元素: {element.tag_name} - {element.text[:50]}")
            except Exception as e:
                logger.debug(f"选择器 {selector} 查找失败: {e}")
        
        if not found_elements:
            logger.warning("❌ 在projects页面没有找到任何jupyter相关元素")
            
            # 检查页面源码
            page_source_lower = driver.page_source.lower()
            if 'jupyter' in page_source_lower:
                logger.info("✅ 页面源码中包含'jupyter'关键字")
                # 计算jupyter出现次数
                jupyter_count = page_source_lower.count('jupyter')
                logger.info(f"'jupyter'在页面中出现 {jupyter_count} 次")
            else:
                logger.warning("❌ 页面源码中没有'jupyter'关键字")
        
        # 查找iframe元素
        logger.info("🔍 查找页面中的iframe元素...")
        iframes = driver.find_elements(By.TAG_NAME, "iframe")
        
        if iframes:
            logger.info(f"找到 {len(iframes)} 个iframe元素")
            for i, iframe in enumerate(iframes):
                src = iframe.get_attribute("src")
                logger.info(f"iframe[{i}] src: {src}")
                
                # 检查iframe是否为空白
                if src and 'jupyter' in src.lower():
                    logger.info(f"🎯 发现jupyter相关的iframe[{i}]")
                    
                    # 切换到iframe查看内容
                    try:
                        driver.switch_to.frame(iframe)
                        time.sleep(3)
                        
                        # 检查iframe内容
                        iframe_body = driver.find_element(By.TAG_NAME, "body")
                        iframe_text = iframe_body.text
                        iframe_html_length = len(driver.page_source)
                        
                        logger.info(f"iframe内容长度: {iframe_html_length}")
                        logger.info(f"iframe文本前100字符: {iframe_text[:100]}")
                        
                        if iframe_html_length < 100:
                            logger.error("❌ iframe内容为空或几乎为空")
                        else:
                            logger.info("✅ iframe有内容")
                            
                        # 切换回主页面
                        driver.switch_to.default_content()
                        
                    except Exception as e:
                        logger.error(f"检查iframe内容时出错: {e}")
                        driver.switch_to.default_content()
        else:
            logger.info("📋 当前页面没有iframe元素")
        
        # Step 2: 如果没有找到iframe，尝试手动创建iframe测试
        if not any('jupyter' in elem.get('text', '').lower() for elem in found_elements) and not any('jupyter' in iframe.get_attribute("src") or '' for iframe in iframes):
            logger.info("🧪 手动创建iframe测试...")
            
            # 注入JavaScript来创建iframe测试
            test_script = """
            // 创建测试iframe
            var testDiv = document.createElement('div');
            testDiv.style.position = 'fixed';
            testDiv.style.top = '10px';
            testDiv.style.right = '10px';
            testDiv.style.width = '400px';
            testDiv.style.height = '300px';
            testDiv.style.background = 'white';
            testDiv.style.border = '2px solid red';
            testDiv.style.zIndex = '9999';
            testDiv.innerHTML = '<h3>Jupyter iframe测试</h3><iframe id="test-jupyter-iframe" src="/jupyterhub" width="100%" height="250px"></iframe>';
            
            document.body.appendChild(testDiv);
            
            // 监控iframe加载
            var testIframe = document.getElementById('test-jupyter-iframe');
            testIframe.onload = function() {
                console.log('测试iframe加载完成');
            };
            testIframe.onerror = function() {
                console.log('测试iframe加载失败');
            };
            
            return 'iframe测试已创建';
            """
            
            result = driver.execute_script(test_script)
            logger.info(f"JavaScript执行结果: {result}")
            
            # 等待iframe加载
            time.sleep(5)
            
            # 检查测试iframe
            try:
                test_iframe = driver.find_element(By.ID, "test-jupyter-iframe")
                iframe_src = test_iframe.get_attribute("src")
                logger.info(f"测试iframe src: {iframe_src}")
                
                # 尝试切换到测试iframe
                driver.switch_to.frame(test_iframe)
                time.sleep(3)
                
                iframe_content_length = len(driver.page_source)
                iframe_body_text = driver.find_element(By.TAG_NAME, "body").text
                
                logger.info(f"测试iframe内容长度: {iframe_content_length}")
                logger.info(f"测试iframe body文本: {iframe_body_text[:200]}")
                
                if iframe_content_length < 100:
                    logger.error("❌ 测试iframe内容为空")
                    return False
                else:
                    logger.info("✅ 测试iframe有内容")
                    return True
                    
            except Exception as e:
                logger.error(f"检查测试iframe失败: {e}")
                return False
            finally:
                driver.switch_to.default_content()
        
        # 获取控制台日志
        logs = driver.get_log('browser')
        if logs:
            logger.info("📝 浏览器控制台日志:")
            for log in logs[-10:]:
                logger.info(f"   {log['level']}: {log['message']}")
        
        return True
        
    except Exception as e:
        logger.error(f"❌ 测试过程中出错: {e}")
        return False
        
    finally:
        if driver:
            # 截图用于调试
            try:
                driver.save_screenshot('projects_jupyter_debug.png')
                logger.info("📸 调试截图已保存: projects_jupyter_debug.png")
            except:
                pass
            driver.quit()

def test_direct_jupyterhub_access():
    """测试直接访问jupyterhub wrapper页面"""
    
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--window-size=1920,1080')
    
    driver = None
    
    try:
        logger.info("🧪 测试直接访问jupyterhub wrapper...")
        driver = webdriver.Chrome(options=chrome_options)
        driver.set_page_load_timeout(30)
        
        # 直接访问jupyterhub wrapper
        driver.get("http://localhost:8080/jupyterhub")
        time.sleep(5)
        
        # 检查页面内容
        page_source_length = len(driver.page_source)
        page_title = driver.title
        
        logger.info(f"JupyterHub wrapper页面标题: {page_title}")
        logger.info(f"页面内容长度: {page_source_length}")
        
        # 查找iframe
        iframes = driver.find_elements(By.TAG_NAME, "iframe")
        logger.info(f"发现 {len(iframes)} 个iframe")
        
        for i, iframe in enumerate(iframes):
            src = iframe.get_attribute("src")
            logger.info(f"iframe[{i}] src: {src}")
        
        return page_source_length > 1000  # 基本的内容检查
        
    except Exception as e:
        logger.error(f"直接访问测试失败: {e}")
        return False
        
    finally:
        if driver:
            driver.quit()

if __name__ == "__main__":
    logger.info("🚀 开始Projects页面Jupyter iframe问题诊断")
    logger.info("=" * 60)
    
    # 测试1: 直接访问jupyterhub wrapper
    logger.info("测试1: 直接访问JupyterHub wrapper页面")
    direct_success = test_direct_jupyterhub_access()
    logger.info(f"直接访问结果: {'✅ 成功' if direct_success else '❌ 失败'}")
    
    # 测试2: 从projects页面访问
    logger.info("\n测试2: 从Projects页面访问Jupyter")
    projects_success = test_projects_to_jupyter_iframe()
    logger.info(f"Projects页面访问结果: {'✅ 成功' if projects_success else '❌ 失败'}")
    
    logger.info("=" * 60)
    logger.info("🏁 诊断完成")
    
    if direct_success and not projects_success:
        logger.info("💡 诊断结论: JupyterHub wrapper正常，但在Projects页面context下访问有问题")
        logger.info("建议检查:")
        logger.info("1. Projects页面的iframe src路径")
        logger.info("2. nginx location配置的优先级")
        logger.info("3. 相对路径vs绝对路径问题")
    elif not direct_success:
        logger.info("💡 诊断结论: JupyterHub wrapper本身有问题")
    else:
        logger.info("💡 诊断结论: 两种访问方式都正常")
