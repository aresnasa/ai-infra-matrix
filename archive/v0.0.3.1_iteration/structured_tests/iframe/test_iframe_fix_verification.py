#!/usr/bin/env python3
"""
iframe白屏问题修复验证
"""

import requests
import time
from selenium import webdriver
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

def test_iframe_fix():
    """测试iframe白屏问题是否修复"""
    
    print("🖼️ iframe白屏问题修复验证")
    print("=" * 60)
    
    # 首先测试HTTP访问
    print("📝 步骤1: 测试HTTP直接访问")
    base_url = "http://localhost:8080"
    
    try:
        # 测试jupyterhub包装器页面
        response = requests.get(f"{base_url}/jupyterhub", timeout=10)
        print(f"   JupyterHub包装器: {response.status_code}")
        
        # 测试iframe src URL
        iframe_response = requests.get(f"{base_url}/jupyter/hub/", timeout=10)
        print(f"   iframe目标URL: {iframe_response.status_code}")
        
        if response.status_code == 200 and iframe_response.status_code == 200:
            print("   ✅ HTTP访问正常")
            http_success = True
        else:
            print("   ❌ HTTP访问异常")
            http_success = False
            
    except Exception as e:
        print(f"   ❌ HTTP测试失败: {e}")
        http_success = False
    
    # 浏览器测试
    print("\n📝 步骤2: 浏览器iframe测试")
    driver = None
    
    try:
        # 配置Chrome
        chrome_options = Options()
        chrome_options.add_argument("--no-sandbox")
        chrome_options.add_argument("--disable-dev-shm-usage")
        chrome_options.add_argument("--disable-gpu")
        chrome_options.add_argument("--window-size=1920,1080")
        chrome_options.add_argument("--headless")  # 无头模式更稳定
        
        service = Service('/opt/homebrew/bin/chromedriver')
        driver = webdriver.Chrome(service=service, options=chrome_options)
        
        print("   ✅ Chrome启动成功")
        
        # 访问JupyterHub包装器页面
        driver.get(f"{base_url}/jupyterhub")
        time.sleep(3)
        
        print(f"   当前页面: {driver.current_url}")
        print(f"   页面标题: {driver.title}")
        
        # 查找iframe
        try:
            iframe = WebDriverWait(driver, 10).until(
                EC.presence_of_element_located((By.TAG_NAME, "iframe"))
            )
            print("   ✅ 找到iframe元素")
            
            # 检查iframe的src属性
            iframe_src = iframe.get_attribute('src')
            print(f"   iframe src: {iframe_src}")
            
            # 切换到iframe内容
            driver.switch_to.frame(iframe)
            time.sleep(2)
            
            # 检查iframe内容
            iframe_body = driver.find_element(By.TAG_NAME, "body")
            iframe_text = iframe_body.text.lower()
            
            print(f"   iframe内容长度: {len(iframe_text)} 字符")
            
            # 检查关键词
            keywords = ['jupyter', 'login', 'username', 'password']
            found_keywords = [kw for kw in keywords if kw in iframe_text]
            
            print(f"   找到关键词: {found_keywords}")
            
            if len(iframe_text) > 100 and len(found_keywords) >= 2:
                print("   ✅ iframe内容正常，非白屏")
                iframe_success = True
            else:
                print("   ⚠️ iframe可能仍有白屏问题")
                iframe_success = False
                
            # 切换回主页面
            driver.switch_to.default_content()
            
        except Exception as e:
            print(f"   ❌ iframe测试失败: {e}")
            iframe_success = False
            
        # 截图保存
        try:
            screenshot_path = "iframe_fix_verification.png"
            driver.save_screenshot(screenshot_path)
            print(f"   📸 截图保存: {screenshot_path}")
        except:
            pass
            
    except Exception as e:
        print(f"   ❌ 浏览器测试失败: {e}")
        iframe_success = False
        
    finally:
        if driver:
            driver.quit()
            print("   🔄 浏览器已关闭")
    
    # 生成结果
    print("\n" + "=" * 60)
    print("📊 iframe修复验证结果")
    print("=" * 60)
    
    if http_success and iframe_success:
        print("🎉 iframe白屏问题已完全修复!")
        print()
        print("✅ 确认修复:")
        print("   • HTTP访问正常")
        print("   • iframe加载正常")
        print("   • iframe内容显示正常")
        print("   • 不再出现白屏")
        print()
        print("🔄 用户体验:")
        print("   1. 访问 http://localhost:8080/projects")
        print("   2. 点击Jupyter图标")
        print("   3. iframe正常显示JupyterHub登录页面")
        print("   4. 使用admin/admin123登录")
        print("   5. 正常使用JupyterHub服务")
        
        return True
        
    else:
        print("⚠️ iframe问题仍需进一步调试")
        
        if not http_success:
            print("   - HTTP访问需要修复")
        if not iframe_success:
            print("   - iframe内容加载需要修复")
            
        return False

if __name__ == "__main__":
    test_iframe_fix()
