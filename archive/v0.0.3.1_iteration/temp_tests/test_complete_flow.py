#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
完整的JupyterHub访问流程测试
模拟从/projects页面点击jupyter图标的完整过程
"""

import time
import requests
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

def setup_driver(headless=True):
    """设置Chrome WebDriver"""
    chrome_options = Options()
    if headless:
        chrome_options.add_argument('--headless')
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')
    chrome_options.add_argument('--window-size=1920,1080')
    chrome_options.add_argument('--disable-blink-features=AutomationControlled')
    chrome_options.add_experimental_option("excludeSwitches", ["enable-automation"])
    chrome_options.add_experimental_option('useAutomationExtension', False)
    
    driver = webdriver.Chrome(options=chrome_options)
    driver.execute_script("Object.defineProperty(navigator, 'webdriver', {get: () => undefined})")
    return driver

def perform_auto_login(driver, username="admin", password="admin123"):
    """执行自动登录"""
    print(f"🔐 尝试自动登录 (用户名: {username})...")
    
    try:
        # 等待页面加载完成
        WebDriverWait(driver, 10).until(
            EC.presence_of_element_located((By.TAG_NAME, "body"))
        )
        
        # 查找用户名输入框的多种可能选择器
        username_selectors = [
            "input[name='username']",
            "input[type='text']",
            "input[id*='username']",
            "input[id*='user']",
            "input[placeholder*='用户']",
            "input[placeholder*='Username']",
            "input[placeholder*='username']"
        ]
        
        username_field = None
        for selector in username_selectors:
            try:
                username_field = WebDriverWait(driver, 3).until(
                    EC.presence_of_element_located((By.CSS_SELECTOR, selector))
                )
                print(f"   找到用户名输入框: {selector}")
                break
            except:
                continue
        
        if not username_field:
            print("   ⚠️  未找到用户名输入框，可能已经登录或页面异常")
            return True
        
        # 查找密码输入框
        password_selectors = [
            "input[name='password']",
            "input[type='password']",
            "input[id*='password']",
            "input[placeholder*='密码']",
            "input[placeholder*='Password']",
            "input[placeholder*='password']"
        ]
        
        password_field = None
        for selector in password_selectors:
            try:
                password_field = driver.find_element(By.CSS_SELECTOR, selector)
                print(f"   找到密码输入框: {selector}")
                break
            except:
                continue
        
        if not password_field:
            print("   ❌ 未找到密码输入框")
            return False
        
        # 输入凭据
        username_field.clear()
        username_field.send_keys(username)
        print(f"   ✅ 已输入用户名: {username}")
        
        password_field.clear()
        password_field.send_keys(password)
        print(f"   ✅ 已输入密码")
        
        # 查找并点击登录按钮
        login_selectors = [
            "button[type='submit']",
            "input[type='submit']",
            "input[value*='登录']",
            "input[value*='Login']",
            ".btn-primary",
            "#login-submit"
        ]
        
        login_button = None
        for selector in login_selectors:
            try:
                login_button = driver.find_element(By.CSS_SELECTOR, selector)
                print(f"   找到登录按钮: {selector}")
                break
            except:
                continue
        
        # 也尝试通过文本查找按钮
        if not login_button:
            try:
                login_button = driver.find_element(By.XPATH, "//button[contains(text(), '登录') or contains(text(), 'Login') or contains(text(), 'Sign')]")
                print("   通过文本找到登录按钮")
            except:
                pass
        
        if not login_button:
            # 尝试回车提交
            password_field.send_keys("\n")
            print("   ⚠️  未找到登录按钮，尝试回车提交")
        else:
            login_button.click()
            print("   ✅ 已点击登录按钮")
        
        # 等待登录处理
        time.sleep(3)
        
        # 检查是否登录成功
        current_url = driver.current_url
        if 'login' not in current_url.lower():
            print("   ✅ 登录成功，已重定向")
            return True
        else:
            print("   ⚠️  可能仍在登录页面，需要进一步检查")
            return False
            
    except Exception as e:
        print(f"   ❌ 自动登录失败: {e}")
        return False

def check_iframe_content(driver, iframe_id, timeout=15):
    """检查iframe内容并尝试自动登录"""
    print(f"🔍 检查iframe内容 (ID: {iframe_id})...")
    
    try:
        # 找到iframe
        iframe = WebDriverWait(driver, 10).until(
            EC.presence_of_element_located((By.ID, iframe_id))
        )
        
        # 等待iframe加载
        time.sleep(3)
        
        # 切换到iframe
        driver.switch_to.frame(iframe)
        
        try:
            # 等待iframe内容加载
            WebDriverWait(driver, timeout).until(
                lambda d: d.execute_script("return document.readyState") == "complete"
            )
            
            # 获取页面源码分析内容
            page_source = driver.page_source.lower()
            page_text = driver.find_element(By.TAG_NAME, "body").text.strip()
            
            print(f"   页面文本长度: {len(page_text)} 字符")
            
            # 检查是否是登录页面
            login_indicators = ['login', '登录', 'username', 'password', 'sign in']
            is_login_page = any(indicator in page_source for indicator in login_indicators)
            
            if is_login_page:
                print("   🔐 检测到登录页面，尝试自动登录...")
                login_success = perform_auto_login(driver)
                
                if login_success:
                    # 登录后重新检查内容
                    time.sleep(3)
                    page_text = driver.find_element(By.TAG_NAME, "body").text.strip()
                    print(f"   登录后页面文本长度: {len(page_text)} 字符")
            
            # 检查JupyterHub特征
            jupyter_indicators = ['jupyter', 'hub', 'notebook', 'spawner', 'server']
            jupyter_found = [ind for ind in jupyter_indicators if ind in page_source]
            
            if jupyter_found:
                print(f"   ✅ 发现JupyterHub特征: {jupyter_found}")
            
            # 检查是否是白屏
            if len(page_text) < 50:
                print("   ❌ 可能是白屏 - 内容过少")
                return False
            else:
                print("   ✅ iframe有内容，非白屏")
                return True
                
        except Exception as e:
            print(f"   ❌ iframe内容检查失败: {e}")
            return False
            
    except Exception as e:
        print(f"   ❌ 无法访问iframe: {e}")
        return False
    finally:
        # 确保切换回主页面
        try:
            driver.switch_to.default_content()
        except:
            pass

def test_complete_flow():
    """测试完整的访问流程"""
    print("🚀 开始完整流程测试...")
    
    driver = setup_driver(headless=True)  # 可以设置为False来看到浏览器
    
    try:
        # 1. 访问主页
        print("📝 1. 访问主页...")
        driver.get("http://localhost:8080/")
        WebDriverWait(driver, 10).until(
            EC.presence_of_element_located((By.TAG_NAME, "body"))
        )
        print(f"   主页标题: {driver.title}")
        driver.save_screenshot("flow_1_homepage.png")
        
        # 2. 等待页面完全加载
        time.sleep(3)
        
        # 3. 导航到projects页面
        print("📁 2. 访问projects页面...")
        driver.get("http://localhost:8080/projects")
        WebDriverWait(driver, 10).until(
            EC.presence_of_element_located((By.TAG_NAME, "body"))
        )
        print(f"   Projects页面标题: {driver.title}")
        driver.save_screenshot("flow_2_projects.png")
        
        # 4. 等待React应用加载
        time.sleep(5)
        
        # 5. 查找并点击JupyterHub链接
        print("🔍 3. 查找JupyterHub访问方式...")
        
        # 方法1: 查找直接的JupyterHub链接
        try:
            jupyter_links = driver.find_elements(By.PARTIAL_LINK_TEXT, "jupyter")
            if not jupyter_links:
                jupyter_links = driver.find_elements(By.PARTIAL_LINK_TEXT, "Jupyter")
            
            if jupyter_links:
                print(f"   找到 {len(jupyter_links)} 个Jupyter链接")
                jupyter_links[0].click()
                print("   点击了Jupyter链接")
            else:
                # 方法2: 直接导航到JupyterHub
                print("   未找到链接，直接导航到JupyterHub...")
                driver.get("http://localhost:8080/jupyterhub/")
        
        except Exception as e:
            print(f"   点击链接失败: {e}")
            print("   直接导航到JupyterHub...")
            driver.get("http://localhost:8080/jupyterhub/")
        
        # 6. 验证JupyterHub wrapper加载
        print("🔧 4. 验证JupyterHub wrapper...")
        WebDriverWait(driver, 15).until(
            EC.presence_of_element_located((By.TAG_NAME, "body"))
        )
        
        current_url = driver.current_url
        title = driver.title
        print(f"   当前URL: {current_url}")
        print(f"   页面标题: {title}")
        
        # 7. 检查iframe
        print("🖼️  5. 检查iframe元素...")
        try:
            iframe = WebDriverWait(driver, 10).until(
                EC.presence_of_element_located((By.ID, "jupyter-frame"))
            )
            print("   ✅ iframe元素存在")
            
            src = iframe.get_attribute('src')
            print(f"   iframe源: {src}")
            
            # 等待iframe加载
            print("⏳ 6. 等待iframe加载...")
            time.sleep(8)
            
            # 检查加载状态
            try:
                loading_overlay = driver.find_element(By.ID, "loading-overlay")
                if 'hidden' in loading_overlay.get_attribute('class'):
                    print("   ✅ 加载覆盖层已隐藏")
                else:
                    print("   ⏳ 仍在加载中...")
            except:
                print("   ✅ 没有加载覆盖层")
            
            # 检查错误状态
            try:
                error_overlay = driver.find_element(By.ID, "error-overlay")
                if 'show' in error_overlay.get_attribute('class'):
                    error_msg = driver.find_element(By.ID, "error-message").text
                    print(f"   ❌ 发现错误: {error_msg}")
                else:
                    print("   ✅ 没有错误显示")
            except:
                print("   ✅ 没有错误覆盖层")
            
            # 检查状态指示器
            try:
                status = driver.find_element(By.ID, "status-indicator")
                status_class = status.get_attribute('class')
                print(f"   📊 连接状态: {status_class}")
                
                if 'connected' in status_class:
                    print("   ✅ iframe连接成功")
                elif 'loading' in status_class:
                    print("   ⏳ iframe仍在连接中")
                else:
                    print("   ❌ iframe连接失败")
            except Exception as e:
                print(f"   ⚠️  无法获取状态: {e}")
                
        except Exception as e:
            print(f"   ❌ iframe检查失败: {e}")
        
        # 8. 最终截图
        driver.save_screenshot("flow_3_jupyterhub_final.png")
        print("📸 7. 最终状态截图已保存")
        
        # 9. 测试iframe内容和自动登录
        print("🔍 8. 检查iframe内容并尝试自动登录...")
        iframe_success = check_iframe_content(driver, "jupyter-frame")
        
        if iframe_success:
            print("   ✅ iframe内容检查通过")
        else:
            print("   ⚠️  iframe内容可能有问题")
        
        # 10. 额外测试：直接访问JupyterHub服务
        print("🔗 9. 测试直接访问JupyterHub服务...")
        driver.get("http://localhost:8080/jupyter/hub/")
        time.sleep(3)
        
        # 检查是否需要登录
        if 'login' in driver.current_url.lower() or 'login' in driver.page_source.lower():
            print("   🔐 检测到JupyterHub登录页面，尝试登录...")
            login_success = perform_auto_login(driver)
            if login_success:
                print("   ✅ 直接登录JupyterHub成功")
                driver.save_screenshot("jupyterhub_direct_login_success.png")
            else:
                print("   ❌ 直接登录JupyterHub失败")
                driver.save_screenshot("jupyterhub_direct_login_failed.png")
        else:
            print("   ✅ 无需登录或已经登录")
            driver.save_screenshot("jupyterhub_direct_access.png")
        
        print("\n🎉 完整流程测试完成！")
        
    except Exception as e:
        print(f"❌ 测试过程中出现错误: {e}")
        # 保存错误时的截图
        try:
            driver.save_screenshot("error_screenshot.png")
        except:
            pass
    finally:
        driver.quit()

def main():
    print("=" * 70)
    print("🔬 JupyterHub 完整访问流程测试")
    print("=" * 70)
    
    # 首先检查服务可用性
    print("🏥 检查服务健康状态...")
    endpoints = [
        "http://localhost:8080/",
        "http://localhost:8080/projects",
        "http://localhost:8080/jupyterhub/",
        "http://localhost:8080/jupyter/hub/"
    ]
    
    for endpoint in endpoints:
        try:
            response = requests.get(endpoint, timeout=5)
            print(f"  {endpoint}: {response.status_code}")
        except Exception as e:
            print(f"  {endpoint}: 错误 - {e}")
    
    print("\n" + "=" * 70)
    
    # 运行完整流程测试
    test_complete_flow()
    
    print("\n" + "=" * 70)
    print("✅ 测试完成！查看生成的截图文件：")
    print("  - flow_1_homepage.png")
    print("  - flow_2_projects.png") 
    print("  - flow_3_jupyterhub_final.png")
    print("=" * 70)

if __name__ == "__main__":
    main()
