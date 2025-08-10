#!/usr/bin/env python3
"""
修复后的iframe测试 - 验证白屏问题是否解决
"""

import time
import logging
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

def test_fixed_iframe():
    """测试修复后的iframe功能"""
    
    chrome_options = Options()
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--disable-web-security')
    chrome_options.add_argument('--window-size=1920,1080')
    
    driver = None
    success_count = 0
    total_tests = 0
    
    try:
        logger.info("🚀 启动Chrome测试修复后的iframe...")
        service = Service('/opt/homebrew/bin/chromedriver')
        driver = webdriver.Chrome(service=service, options=chrome_options)
        driver.set_page_load_timeout(30)
        wait = WebDriverWait(driver, 20)
        
        # 测试1: 访问Projects页面并检查菜单
        logger.info("📍 测试1: 检查Projects页面的JupyterHub菜单")
        total_tests += 1
        
        driver.get("http://localhost:8080/projects")
        time.sleep(3)
        
        # 截图1: Projects页面
        driver.save_screenshot('test_fix_1_projects.png')
        
        # 查找JupyterHub菜单项
        jupyter_menu_found = False
        selectors = [
            "li[data-menu-id='/jupyterhub']",
            "a[href='/jupyterhub']",
            "span:contains('JupyterHub')",
            ".ant-menu-item[data-menu-id='/jupyterhub']"
        ]
        
        for selector in selectors:
            try:
                if 'contains' in selector:
                    elements = driver.find_elements(By.XPATH, "//span[contains(text(), 'JupyterHub')]")
                else:
                    elements = driver.find_elements(By.CSS_SELECTOR, selector)
                
                if elements:
                    logger.info(f"✅ 找到JupyterHub菜单: {selector}")
                    jupyter_menu_found = True
                    
                    # 点击菜单项
                    element = elements[0]
                    logger.info("🖱️ 点击JupyterHub菜单项")
                    
                    # 获取点击前的URL
                    before_url = driver.current_url
                    logger.info(f"点击前URL: {before_url}")
                    
                    # 执行点击
                    driver.execute_script("arguments[0].click();", element)
                    time.sleep(5)
                    
                    # 检查URL变化
                    after_url = driver.current_url
                    logger.info(f"点击后URL: {after_url}")
                    
                    if 'jupyterhub' in after_url:
                        logger.info("✅ 成功导航到JupyterHub")
                        success_count += 1
                    else:
                        logger.warning(f"⚠️ URL未变化为JupyterHub: {after_url}")
                    
                    break
            except Exception as e:
                logger.debug(f"选择器 {selector} 测试失败: {e}")
        
        if not jupyter_menu_found:
            logger.error("❌ 未找到JupyterHub菜单项")
            
            # 检查页面源码
            page_source = driver.page_source
            if 'JupyterHub' in page_source:
                logger.info("页面源码中包含JupyterHub文本")
            else:
                logger.error("页面源码中不包含JupyterHub文本")
        
        # 测试2: 直接访问JupyterHub
        logger.info("📍 测试2: 直接访问JupyterHub")
        total_tests += 1
        
        driver.get("http://localhost:8080/jupyterhub")
        time.sleep(5)
        
        # 截图2: JupyterHub页面
        driver.save_screenshot('test_fix_2_jupyterhub.png')
        
        current_url = driver.current_url
        body_text = driver.find_element(By.TAG_NAME, "body").text
        
        logger.info(f"JupyterHub页面URL: {current_url}")
        logger.info(f"页面内容长度: {len(body_text)} 字符")
        
        if len(body_text) > 100 and ('login' in body_text.lower() or 'jupyter' in body_text.lower()):
            logger.info("✅ JupyterHub页面加载正常")
            success_count += 1
        else:
            logger.error("❌ JupyterHub页面可能有问题")
            logger.info(f"页面内容预览: {body_text[:200]}...")
        
        # 测试3: iframe测试页面
        logger.info("📍 测试3: iframe测试页面")
        total_tests += 1
        
        driver.get("http://localhost:8080/iframe_test.html")
        time.sleep(8)  # 等待iframe加载
        
        # 截图3: iframe测试页面
        driver.save_screenshot('test_fix_3_iframe_test.png')
        
        # 检查iframe
        iframes = driver.find_elements(By.TAG_NAME, "iframe")
        logger.info(f"找到 {len(iframes)} 个iframe")
        
        iframe_success = False
        
        for i, iframe in enumerate(iframes):
            src = iframe.get_attribute("src")
            logger.info(f"检查iframe[{i}]: {src}")
            
            if 'jupyter' in src.lower():
                try:
                    # 切换到iframe
                    driver.switch_to.frame(iframe)
                    time.sleep(3)
                    
                    # 检查内容
                    body = wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))
                    body_text = body.text.strip()
                    
                    logger.info(f"iframe[{i}] 内容长度: {len(body_text)} 字符")
                    
                    if len(body_text) > 50:
                        logger.info(f"✅ iframe[{i}] 内容正常!")
                        logger.info(f"内容预览: {body_text[:100]}...")
                        iframe_success = True
                    else:
                        logger.error(f"❌ iframe[{i}] 仍然白屏! 内容: '{body_text}'")
                    
                    # 截图iframe内容
                    driver.save_screenshot(f'test_fix_iframe_{i}_content.png')
                    
                    driver.switch_to.default_content()
                    
                except Exception as e:
                    logger.error(f"❌ iframe[{i}] 检查失败: {e}")
                    driver.switch_to.default_content()
        
        if iframe_success:
            success_count += 1
        
        # 测试4: 用户体验流程测试
        logger.info("📍 测试4: 完整用户体验流程")
        total_tests += 1
        
        # 从projects页面开始
        driver.get("http://localhost:8080/projects")
        time.sleep(3)
        
        # 尝试点击页面上的任何JupyterHub相关链接
        jupyter_elements = []
        
        # 查找所有可能的JupyterHub元素
        all_elements = driver.find_elements(By.XPATH, "//*[contains(text(), 'Jupyter') or contains(text(), 'jupyter')]")
        
        for elem in all_elements:
            try:
                if elem.is_displayed() and elem.is_enabled():
                    text = elem.text.strip()
                    tag = elem.tag_name
                    logger.info(f"找到Jupyter元素: {tag} - '{text}'")
                    jupyter_elements.append(elem)
            except:
                continue
        
        flow_success = False
        if jupyter_elements:
            try:
                # 点击第一个可用的元素
                element = jupyter_elements[0]
                logger.info("🖱️ 点击用户体验流程中的Jupyter元素")
                
                driver.execute_script("arguments[0].click();", element)
                time.sleep(5)
                
                final_url = driver.current_url
                logger.info(f"流程结束URL: {final_url}")
                
                if 'jupyter' in final_url:
                    logger.info("✅ 用户体验流程成功!")
                    flow_success = True
                else:
                    logger.warning("⚠️ 用户体验流程未达到预期")
                    
            except Exception as e:
                logger.error(f"❌ 用户体验流程失败: {e}")
        else:
            logger.warning("⚠️ 未找到可点击的Jupyter元素")
        
        if flow_success:
            success_count += 1
        
        # 截图4: 最终状态
        driver.save_screenshot('test_fix_4_final_state.png')
        
        # 测试总结
        logger.info("🏁 测试完成!")
        logger.info(f"成功测试: {success_count}/{total_tests}")
        
        if success_count == total_tests:
            logger.info("🎉 所有测试通过! iframe白屏问题已修复!")
            return True
        elif success_count > total_tests // 2:
            logger.warning("⚠️ 大部分测试通过，部分问题仍需解决")
            return False
        else:
            logger.error("❌ 多数测试失败，需要进一步修复")
            return False
            
    except Exception as e:
        logger.error(f"❌ 测试过程异常: {e}")
        return False
        
    finally:
        if driver:
            # 保持浏览器打开15秒以便观察
            logger.info("⏸️ 保持浏览器打开15秒以便观察...")
            time.sleep(15)
            driver.quit()

if __name__ == "__main__":
    logger.info("🧪 开始修复后的iframe测试")
    logger.info("=" * 60)
    
    success = test_fixed_iframe()
    
    if success:
        logger.info("✅ iframe白屏修复验证成功!")
    else:
        logger.error("❌ iframe白屏问题仍需进一步修复")
    
    logger.info("🏁 修复验证测试结束")
