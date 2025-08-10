#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
简化版JupyterHub Wrapper验证
验证优化后的wrapper是否正常工作
"""

import requests
import time
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

def test_api_access():
    """测试API访问"""
    print("🔍 测试API访问...")
    
    urls = [
        "http://localhost:8080/",
        "http://localhost:8080/jupyterhub",
        "http://localhost:8080/jupyterhub/",
        "http://localhost:8080/jupyter/hub/"
    ]
    
    for url in urls:
        try:
            response = requests.get(url, timeout=5, allow_redirects=True)
            print(f"  {url}: {response.status_code}")
            if response.status_code == 301:
                print(f"    重定向到: {response.headers.get('Location', 'N/A')}")
        except Exception as e:
            print(f"  {url}: 错误 - {e}")

def test_wrapper_with_browser():
    """用浏览器测试wrapper"""
    print("\n🌐 启动浏览器测试...")
    
    driver = None
    try:
        # 设置Chrome选项
        chrome_options = Options()
        chrome_options.add_argument('--headless')
        chrome_options.add_argument('--no-sandbox')
        chrome_options.add_argument('--disable-dev-shm-usage')
        chrome_options.add_argument('--window-size=1280,720')
        
        driver = webdriver.Chrome(options=chrome_options)
        print("✅ Chrome WebDriver已启动")
        
        # 访问wrapper页面
        print("📄 加载wrapper页面...")
        driver.get("http://localhost:8080/jupyterhub/")
        
        # 等待页面加载
        WebDriverWait(driver, 10).until(
            EC.presence_of_element_located((By.TAG_NAME, "body"))
        )
        
        title = driver.title
        print(f"📋 页面标题: {title}")
        
        # 检查iframe
        try:
            iframe = WebDriverWait(driver, 5).until(
                EC.presence_of_element_located((By.ID, "jupyter-frame"))
            )
            print("✅ iframe元素已找到")
            
            # 检查iframe属性
            src = iframe.get_attribute('src')
            print(f"🔗 iframe源地址: {src}")
            
            # 等待一段时间让iframe加载
            print("⏳ 等待iframe加载...")
            time.sleep(5)
            
            # 检查状态指示器
            try:
                status = driver.find_element(By.ID, "status-indicator")
                status_class = status.get_attribute('class')
                print(f"📊 连接状态: {status_class}")
                
                if 'connected' in status_class:
                    print("✅ iframe已成功连接")
                elif 'loading' in status_class:
                    print("⏳ iframe仍在加载中")
                else:
                    print("❌ iframe连接失败")
            except Exception as e:
                print(f"⚠️  无法获取状态指示器: {e}")
            
            # 检查是否有错误覆盖层
            try:
                error_overlay = driver.find_element(By.ID, "error-overlay")
                if 'show' in error_overlay.get_attribute('class'):
                    error_msg = driver.find_element(By.ID, "error-message").text
                    print(f"❌ 发现错误: {error_msg}")
                else:
                    print("✅ 没有错误显示")
            except:
                print("✅ 没有错误覆盖层")
            
            # 截图
            driver.save_screenshot("wrapper_verification_test.png")
            print("📸 截图已保存: wrapper_verification_test.png")
            
        except Exception as e:
            print(f"❌ iframe检查失败: {e}")
        
        print("✅ 浏览器测试完成")
        
    except Exception as e:
        print(f"❌ 浏览器测试失败: {e}")
    finally:
        if driver:
            driver.quit()
            print("🔚 WebDriver已关闭")

def main():
    print("=" * 60)
    print("🚀 JupyterHub Wrapper 优化版本验证")
    print("=" * 60)
    
    # 1. API测试
    test_api_access()
    
    # 2. 浏览器测试
    test_wrapper_with_browser()
    
    print("\n=" * 60)
    print("✅ 验证测试完成！")
    print("=" * 60)

if __name__ == "__main__":
    main()
