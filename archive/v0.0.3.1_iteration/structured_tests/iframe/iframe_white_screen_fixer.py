#!/usr/bin/env python3
"""
专门的iframe白屏问题诊断和修复脚本
"""

import time
import logging
import os
import subprocess
from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC

# 设置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

class IframeFixer:
    def __init__(self):
        self.driver = None
        self.wait = None
        self.issues_found = []
        self.fixes_applied = []
        
    def setup_driver(self):
        """设置Chrome WebDriver"""
        chrome_options = Options()
        chrome_options.add_argument('--no-sandbox')
        chrome_options.add_argument('--disable-dev-shm-usage')
        chrome_options.add_argument('--disable-web-security')
        chrome_options.add_argument('--disable-features=VizDisplayCompositor')
        chrome_options.add_argument('--window-size=1920,1080')
        
        # 启用性能和网络日志
        chrome_options.add_argument('--enable-logging')
        chrome_options.add_argument('--log-level=0')
        chrome_options.set_capability('goog:loggingPrefs', {
            'browser': 'ALL',
            'performance': 'ALL'
        })
        
        try:
            service = Service('/opt/homebrew/bin/chromedriver')
            self.driver = webdriver.Chrome(service=service, options=chrome_options)
            self.driver.set_page_load_timeout(30)
            self.wait = WebDriverWait(self.driver, 20)
            
            logger.info("✅ Chrome WebDriver启动成功")
            return True
            
        except Exception as e:
            logger.error(f"❌ Chrome WebDriver启动失败: {e}")
            return False
    
    def diagnose_iframe_issues(self):
        """诊断iframe问题"""
        logger.info("🔍 开始iframe白屏问题诊断...")
        
        try:
            # Step 1: 访问Projects页面
            logger.info("📍 Step 1: 访问Projects页面")
            self.driver.get("http://localhost:8080/projects")
            time.sleep(3)
            
            # 截图1: Projects页面
            self.driver.save_screenshot('diagnosis_1_projects_page.png')
            
            # Step 2: 查找Jupyter链接/按钮
            logger.info("📍 Step 2: 查找Jupyter链接")
            
            jupyter_elements = []
            selectors = [
                "a[href*='jupyter']",
                "a[href='/jupyterhub']", 
                "button[onclick*='jupyter']",
                "[data-testid*='jupyter']",
                "a:contains('Jupyter')",
                "button:contains('Jupyter')"
            ]
            
            for selector in selectors:
                try:
                    if 'contains' in selector:
                        if 'button' in selector:
                            elements = self.driver.find_elements(By.XPATH, "//button[contains(translate(text(), 'JUPYTER', 'jupyter'), 'jupyter')]")
                        else:
                            elements = self.driver.find_elements(By.XPATH, "//a[contains(translate(text(), 'JUPYTER', 'jupyter'), 'jupyter')]")
                    else:
                        elements = self.driver.find_elements(By.CSS_SELECTOR, selector)
                    
                    if elements:
                        for elem in elements:
                            href = elem.get_attribute('href') or elem.get_attribute('onclick') or 'button'
                            text = elem.text.strip()
                            jupyter_elements.append({
                                'element': elem,
                                'selector': selector,
                                'href': href,
                                'text': text
                            })
                        logger.info(f"✅ 找到 {len(elements)} 个元素: {selector}")
                except Exception as e:
                    logger.debug(f"选择器 {selector} 未找到: {e}")
            
            if not jupyter_elements:
                logger.error("❌ 未找到Jupyter相关链接或按钮")
                self.issues_found.append("未找到Jupyter导航元素")
                return False
            
            # Step 3: 点击Jupyter链接并分析结果
            logger.info("📍 Step 3: 点击Jupyter链接并分析")
            
            for i, jupyter_info in enumerate(jupyter_elements[:1]):  # 只测试第一个
                element = jupyter_info['element']
                logger.info(f"🖱️ 点击Jupyter元素: {jupyter_info['text']} -> {jupyter_info['href']}")
                
                # 滚动到元素
                self.driver.execute_script("arguments[0].scrollIntoView(true);", element)
                time.sleep(1)
                
                # 点击元素
                element.click()
                time.sleep(5)
                
                # 截图2: 点击后的页面
                self.driver.save_screenshot(f'diagnosis_2_after_click_{i}.png')
                
                current_url = self.driver.current_url
                logger.info(f"点击后URL: {current_url}")
                
                # Step 4: 检查iframe
                logger.info("📍 Step 4: 检查iframe内容")
                
                iframes = self.driver.find_elements(By.TAG_NAME, "iframe")
                logger.info(f"找到 {len(iframes)} 个iframe")
                
                if not iframes:
                    logger.warning("⚠️ 未找到iframe元素")
                    self.issues_found.append("页面中没有iframe")
                    continue
                
                for iframe_idx, iframe in enumerate(iframes):
                    self.analyze_iframe(iframe_idx, iframe)
                
                break  # 只测试第一个链接
            
            return True
            
        except Exception as e:
            logger.error(f"❌ iframe诊断失败: {e}")
            self.issues_found.append(f"诊断过程异常: {e}")
            return False
    
    def analyze_iframe(self, idx, iframe):
        """分析单个iframe"""
        logger.info(f"🔍 分析iframe[{idx}]")
        
        try:
            # 获取iframe属性
            src = iframe.get_attribute("src")
            width = iframe.get_attribute("width") 
            height = iframe.get_attribute("height")
            style = iframe.get_attribute("style")
            sandbox = iframe.get_attribute("sandbox")
            
            logger.info(f"  src: {src}")
            logger.info(f"  尺寸: {width}x{height}")
            logger.info(f"  style: {style}")
            logger.info(f"  sandbox: {sandbox}")
            
            # 检查iframe是否可见
            is_displayed = iframe.is_displayed()
            rect = iframe.rect
            logger.info(f"  显示状态: {is_displayed}")
            logger.info(f"  位置和大小: {rect}")
            
            if not is_displayed:
                self.issues_found.append(f"iframe[{idx}] 不可见")
                return
            
            if rect['width'] <= 0 or rect['height'] <= 0:
                self.issues_found.append(f"iframe[{idx}] 尺寸无效: {rect['width']}x{rect['height']}")
                return
            
            # 切换到iframe检查内容
            logger.info(f"  切换到iframe[{idx}]...")
            self.driver.switch_to.frame(iframe)
            
            # 等待内容加载
            time.sleep(3)
            
            try:
                # 检查iframe内容
                body = self.wait.until(EC.presence_of_element_located((By.TAG_NAME, "body")))
                body_text = body.text.strip()
                inner_html = body.get_attribute("innerHTML")
                
                logger.info(f"  iframe内容长度: {len(body_text)} 字符")
                logger.info(f"  HTML长度: {len(inner_html)} 字符")
                
                if len(body_text) < 10 and len(inner_html) < 100:
                    logger.error(f"  ❌ iframe[{idx}] 内容为空或很少 - 白屏!")
                    self.issues_found.append(f"iframe[{idx}] 白屏 - 内容不足")
                    
                    # 检查具体原因
                    self.check_iframe_loading_issues(idx)
                else:
                    logger.info(f"  ✅ iframe[{idx}] 有内容")
                    logger.info(f"  内容预览: {body_text[:100]}...")
                
                # 检查是否有错误页面
                if "404" in body_text or "Not Found" in body_text:
                    logger.error(f"  ❌ iframe[{idx}] 显示404错误")
                    self.issues_found.append(f"iframe[{idx}] 404错误")
                elif "500" in body_text or "Internal Server Error" in body_text:
                    logger.error(f"  ❌ iframe[{idx}] 显示服务器错误")
                    self.issues_found.append(f"iframe[{idx}] 服务器错误")
                elif "loading" in body_text.lower() and len(body_text) < 50:
                    logger.warning(f"  ⚠️ iframe[{idx}] 可能卡在加载中")
                    self.issues_found.append(f"iframe[{idx}] 加载超时")
                
                # 截图iframe内容
                self.driver.save_screenshot(f'diagnosis_iframe_{idx}_content.png')
                
            except Exception as e:
                logger.error(f"  ❌ 无法检查iframe[{idx}]内容: {e}")
                self.issues_found.append(f"iframe[{idx}] 内容检查失败: {e}")
            
            finally:
                # 切换回主文档
                self.driver.switch_to.default_content()
                
        except Exception as e:
            logger.error(f"❌ 分析iframe[{idx}]失败: {e}")
            self.issues_found.append(f"iframe[{idx}] 分析失败: {e}")
    
    def check_iframe_loading_issues(self, idx):
        """检查iframe加载问题的具体原因"""
        logger.info(f"🔍 检查iframe[{idx}]加载问题...")
        
        try:
            # 检查网络请求
            current_url = self.driver.current_url
            logger.info(f"  iframe当前URL: {current_url}")
            
            # 检查是否有JavaScript错误
            logs = self.driver.get_log('browser')
            error_count = 0
            for log in logs:
                if log['level'] in ['SEVERE', 'ERROR']:
                    logger.error(f"  JS错误: {log['message']}")
                    error_count += 1
            
            if error_count > 0:
                self.issues_found.append(f"iframe[{idx}] 有{error_count}个JavaScript错误")
            
            # 检查页面标题
            title = self.driver.title
            logger.info(f"  iframe标题: {title}")
            
            if not title or title == "":
                self.issues_found.append(f"iframe[{idx}] 页面标题为空")
            
        except Exception as e:
            logger.error(f"检查iframe[{idx}]加载问题失败: {e}")
    
    def apply_fixes(self):
        """应用修复方案"""
        logger.info("🔧 开始应用修复方案...")
        
        for issue in self.issues_found:
            logger.info(f"处理问题: {issue}")
            
            if "未找到Jupyter导航元素" in issue:
                self.fix_missing_jupyter_nav()
            elif "白屏" in issue:
                self.fix_iframe_blank_screen()
            elif "404错误" in issue:
                self.fix_404_error()
            elif "服务器错误" in issue:
                self.fix_server_error()
            elif "加载超时" in issue:
                self.fix_loading_timeout()
    
    def fix_missing_jupyter_nav(self):
        """修复缺失的Jupyter导航"""
        logger.info("🔧 修复Jupyter导航元素...")
        
        # 直接导航到JupyterHub
        try:
            self.driver.get("http://localhost:8080/jupyterhub")
            time.sleep(3)
            self.driver.save_screenshot('fix_direct_jupyterhub.png')
            self.fixes_applied.append("直接导航到JupyterHub")
            logger.info("✅ 应用直接导航修复")
        except Exception as e:
            logger.error(f"❌ 直接导航修复失败: {e}")
    
    def fix_iframe_blank_screen(self):
        """修复iframe白屏"""
        logger.info("🔧 修复iframe白屏问题...")
        
        try:
            # 尝试刷新页面
            self.driver.refresh()
            time.sleep(5)
            
            # 检查iframe是否现在有内容
            iframes = self.driver.find_elements(By.TAG_NAME, "iframe")
            if iframes:
                for i, iframe in enumerate(iframes):
                    try:
                        self.driver.switch_to.frame(iframe)
                        body = self.driver.find_element(By.TAG_NAME, "body")
                        if len(body.text.strip()) > 10:
                            logger.info(f"✅ iframe[{i}] 刷新后有内容")
                            self.fixes_applied.append(f"iframe[{i}] 页面刷新修复")
                        self.driver.switch_to.default_content()
                    except:
                        self.driver.switch_to.default_content()
            
            # 尝试禁用sandbox
            self.driver.execute_script("""
                var iframes = document.querySelectorAll('iframe');
                for (var i = 0; i < iframes.length; i++) {
                    iframes[i].removeAttribute('sandbox');
                    console.log('Removed sandbox from iframe', i);
                }
            """)
            
            time.sleep(2)
            self.driver.save_screenshot('fix_remove_sandbox.png')
            self.fixes_applied.append("移除iframe sandbox属性")
            
        except Exception as e:
            logger.error(f"❌ iframe白屏修复失败: {e}")
    
    def fix_404_error(self):
        """修复404错误"""
        logger.info("🔧 修复404错误...")
        
        # 尝试不同的JupyterHub URL路径
        test_urls = [
            "http://localhost:8080/jupyter/hub/",
            "http://localhost:8080/jupyter/hub/login", 
            "http://localhost:8080/jupyterhub/hub/",
            "http://localhost:8080/hub/"
        ]
        
        for url in test_urls:
            try:
                logger.info(f"尝试URL: {url}")
                self.driver.get(url)
                time.sleep(3)
                
                body_text = self.driver.find_element(By.TAG_NAME, "body").text
                if "404" not in body_text and "Not Found" not in body_text:
                    logger.info(f"✅ URL有效: {url}")
                    self.fixes_applied.append(f"找到有效URL: {url}")
                    break
            except Exception as e:
                logger.debug(f"URL {url} 测试失败: {e}")
    
    def fix_server_error(self):
        """修复服务器错误"""
        logger.info("🔧 检查服务器状态...")
        
        try:
            # 检查JupyterHub服务状态
            result = subprocess.run(['docker-compose', 'ps', 'ai-infra-jupyterhub'], 
                                  capture_output=True, text=True, timeout=10)
            logger.info(f"JupyterHub服务状态: {result.stdout}")
            
            # 重启服务建议
            self.fixes_applied.append("建议检查JupyterHub服务状态")
            
        except Exception as e:
            logger.error(f"检查服务状态失败: {e}")
    
    def fix_loading_timeout(self):
        """修复加载超时"""
        logger.info("🔧 修复加载超时...")
        
        try:
            # 增加等待时间并重新加载
            self.driver.set_page_load_timeout(60)
            time.sleep(10)
            
            self.fixes_applied.append("增加页面加载超时时间")
            
        except Exception as e:
            logger.error(f"修复加载超时失败: {e}")
    
    def generate_report(self):
        """生成诊断报告"""
        logger.info("📋 生成诊断报告...")
        
        report = f"""
# JupyterHub iframe白屏问题诊断报告
生成时间: {time.strftime('%Y-%m-%d %H:%M:%S')}

## 发现的问题 ({len(self.issues_found)} 个)
"""
        for i, issue in enumerate(self.issues_found, 1):
            report += f"{i}. {issue}\n"
        
        report += f"""
## 应用的修复 ({len(self.fixes_applied)} 个)
"""
        for i, fix in enumerate(self.fixes_applied, 1):
            report += f"{i}. {fix}\n"
        
        report += """
## 建议的后续步骤
1. 检查nginx配置中的JupyterHub代理设置
2. 验证JupyterHub服务运行状态
3. 检查CSP (Content Security Policy) 设置
4. 验证iframe src URL的可访问性
5. 检查前端路由配置

## 生成的截图文件
- diagnosis_1_projects_page.png: Projects页面
- diagnosis_2_after_click_*.png: 点击Jupyter后的页面
- diagnosis_iframe_*_content.png: iframe内容截图
- fix_*.png: 修复过程截图
"""
        
        # 保存报告
        with open('iframe_diagnosis_report.md', 'w', encoding='utf-8') as f:
            f.write(report)
        
        logger.info("📄 诊断报告已保存: iframe_diagnosis_report.md")
        print(report)
    
    def run_diagnosis(self):
        """运行完整诊断"""
        try:
            if not self.setup_driver():
                return False
            
            self.diagnose_iframe_issues()
            self.apply_fixes()
            self.generate_report()
            
            return True
            
        finally:
            if self.driver:
                # 最终截图
                try:
                    self.driver.save_screenshot('final_diagnosis_state.png')
                except:
                    pass
                
                # 保持浏览器打开10秒
                logger.info("⏸️ 保持浏览器打开10秒以便观察...")
                time.sleep(10)
                self.driver.quit()

def main():
    logger.info("🚀 JupyterHub iframe白屏问题诊断器启动")
    logger.info("=" * 60)
    
    fixer = IframeFixer()
    success = fixer.run_diagnosis()
    
    if success:
        logger.info("✅ 诊断完成")
    else:
        logger.error("❌ 诊断失败")
    
    logger.info("🏁 程序结束")

if __name__ == "__main__":
    main()
