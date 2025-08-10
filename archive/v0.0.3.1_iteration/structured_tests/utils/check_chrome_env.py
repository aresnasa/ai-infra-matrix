#!/usr/bin/env python3
"""
ChromeDriver环境诊断脚本
"""

import os
import subprocess
import sys

def check_environment():
    """检查ChromeDriver运行环境"""
    
    print("🔍 ChromeDriver环境诊断")
    print("=" * 50)
    
    # 1. 检查Python和Selenium
    print(f"Python版本: {sys.version}")
    
    try:
        import selenium
        print(f"✅ Selenium版本: {selenium.__version__}")
    except ImportError:
        print("❌ Selenium未安装")
        print("   请运行: pip3 install selenium")
        return False
    
    # 2. 检查Chrome浏览器
    chrome_paths = [
        "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
        "/usr/bin/google-chrome",
        "/usr/bin/chromium-browser"
    ]
    
    chrome_path = None
    for path in chrome_paths:
        if os.path.exists(path):
            chrome_path = path
            print(f"✅ Chrome浏览器找到: {path}")
            break
    
    if not chrome_path:
        print("❌ Chrome浏览器未找到")
        print("   请从 https://www.google.com/chrome/ 下载安装Chrome")
        return False
    
    # 获取Chrome版本
    try:
        result = subprocess.run([chrome_path, '--version'], 
                              capture_output=True, text=True, timeout=5)
        chrome_version = result.stdout.strip()
        print(f"Chrome版本: {chrome_version}")
    except Exception as e:
        print(f"⚠️ 无法获取Chrome版本: {e}")
    
    # 3. 检查ChromeDriver
    chromedriver_paths = [
        "/opt/homebrew/bin/chromedriver",
        "/usr/local/bin/chromedriver",
        "/usr/bin/chromedriver"
    ]
    
    chromedriver_path = None
    for path in chromedriver_paths:
        if os.path.exists(path):
            chromedriver_path = path
            print(f"✅ ChromeDriver找到: {path}")
            break
    
    if not chromedriver_path:
        print("❌ ChromeDriver未找到")
        print("   请运行: brew install chromedriver")
        return False
    
    # 获取ChromeDriver版本
    try:
        result = subprocess.run([chromedriver_path, '--version'], 
                              capture_output=True, text=True, timeout=5)
        chromedriver_version = result.stdout.strip()
        print(f"ChromeDriver版本: {chromedriver_version}")
    except Exception as e:
        print(f"⚠️ 无法获取ChromeDriver版本: {e}")
    
    # 4. 检查权限
    try:
        result = subprocess.run([chromedriver_path, '--help'], 
                              capture_output=True, text=True, timeout=5)
        if result.returncode == 0:
            print("✅ ChromeDriver权限正常")
        else:
            print("❌ ChromeDriver权限问题")
            print("   请运行: xattr -d com.apple.quarantine " + chromedriver_path)
            return False
    except Exception as e:
        print(f"❌ ChromeDriver执行失败: {e}")
        print("   请运行: xattr -d com.apple.quarantine " + chromedriver_path)
        return False
    
    # 5. 版本兼容性检查
    print("\n🔄 版本兼容性分析:")
    if 'chrome_version' in locals() and 'chromedriver_version' in locals():
        try:
            # 提取主版本号
            chrome_major = chrome_version.split()[2].split('.')[0] if len(chrome_version.split()) > 2 else "未知"
            chromedriver_major = chromedriver_version.split()[1].split('.')[0] if len(chromedriver_version.split()) > 1 else "未知"
            
            print(f"Chrome主版本: {chrome_major}")
            print(f"ChromeDriver主版本: {chromedriver_major}")
            
            if chrome_major == chromedriver_major:
                print("✅ 版本兼容")
            else:
                print("❌ 版本不兼容")
                print("   建议运行: brew upgrade chromedriver")
                return False
        except Exception as e:
            print(f"⚠️ 版本分析失败: {e}")
    
    print("\n✅ 环境检查完成，可以尝试运行ChromeDriver")
    return True

def test_simple_chrome():
    """简单的Chrome启动测试"""
    print("\n🧪 简单Chrome启动测试")
    print("-" * 30)
    
    try:
        from selenium import webdriver
        from selenium.webdriver.chrome.options import Options
        from selenium.webdriver.chrome.service import Service
        
        # 基本选项
        options = Options()
        options.add_argument('--headless')  # 无头模式
        options.add_argument('--no-sandbox')
        options.add_argument('--disable-dev-shm-usage')
        
        # 服务
        service = Service('/opt/homebrew/bin/chromedriver')
        
        print("🚀 启动Chrome WebDriver (无头模式)...")
        driver = webdriver.Chrome(service=service, options=options)
        
        print("📍 访问测试页面...")
        driver.get("https://www.google.com")
        
        title = driver.title
        print(f"页面标题: {title}")
        
        driver.quit()
        print("✅ 简单测试成功!")
        return True
        
    except Exception as e:
        print(f"❌ 简单测试失败: {e}")
        return False

if __name__ == "__main__":
    env_ok = check_environment()
    
    if env_ok:
        test_simple_chrome()
    else:
        print("\n❌ 环境检查失败，请修复上述问题后重试")
        sys.exit(1)
