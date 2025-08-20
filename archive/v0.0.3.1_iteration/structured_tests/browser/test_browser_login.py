#!/usr/bin/env python3
"""
AI基础设施矩阵 - JupyterHub iframe 浏览器自动化测试
使用Selenium WebDriver模拟真实浏览器行为，诊断iframe白屏问题
"""

import time
import json
import logging
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.chrome.options import Options
from selenium.common.exceptions import TimeoutException, NoSuchElementException
from datetime import datetime

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('jupyterhub_test.log'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

class JupyterHubTester:
    def __init__(self, base_url="http://localhost:8080", headless=False):
        self.base_url = base_url
        self.headless = headless
        self.driver = None
        self.test_results = {}
        
    def setup_driver(self):
        """设置Chrome WebDriver"""
        chrome_options = Options()
        if self.headless:
            chrome_options.add_argument("--headless")
        
        # 禁用安全策略以便测试iframe
        chrome_options.add_argument("--disable-web-security")
        chrome_options.add_argument("--disable-features=VizDisplayCompositor")
        chrome_options.add_argument("--disable-dev-shm-usage")
        chrome_options.add_argument("--no-sandbox")
        chrome_options.add_argument("--disable-gpu")
        
        # 设置窗口大小
        chrome_options.add_argument("--window-size=1920,1080")
        
        try:
            self.driver = webdriver.Chrome(options=chrome_options)
            logger.info("✅ Chrome WebDriver 初始化成功")
            return True
        except Exception as e:
            logger.error(f"❌ Chrome WebDriver 初始化失败: {e}")
            return False
    
    def test_main_page_access(self):
        """测试主页面访问"""
        test_name = "主页面访问测试"
        logger.info(f"🧪 开始 {test_name}")
        
        try:
            self.driver.get(f"{self.base_url}/")
            
            # 等待页面加载
            WebDriverWait(self.driver, 10).until(
                lambda driver: driver.execute_script("return document.readyState") == "complete"
            )
            
            title = self.driver.title
            logger.info(f"📄 页面标题: {title}")
            
            self.test_results[test_name] = {
                "status": "success",
                "title": title,
                "url": self.driver.current_url
            }
            
        except Exception as e:
            logger.error(f"❌ {test_name} 失败: {e}")
            self.test_results[test_name] = {
                "status": "failed",
                "error": str(e)
            }
    
    def test_api_health(self):
        """测试API健康状态"""
        test_name = "API健康检查"
        logger.info(f"🧪 开始 {test_name}")
        
        try:
            # 使用JavaScript在浏览器中测试API
            api_test_script = """
            return fetch('/api/health')
                .then(response => ({
                    status: response.status,
                    ok: response.ok,
                    statusText: response.statusText
                }))
                .catch(error => ({
                    status: 'error',
                    error: error.message
                }));
            """
            
            result = self.driver.execute_async_script(f"""
                var callback = arguments[arguments.length - 1];
                {api_test_script.replace('return', '')}
                    .then(callback)
                    .catch(callback);
            """)
            
            logger.info(f"🔍 API健康检查结果: {result}")
            self.test_results[test_name] = {
                "status": "success" if result.get('ok') else "failed",
                "api_response": result
            }
            
        except Exception as e:
            logger.error(f"❌ {test_name} 失败: {e}")
            self.test_results[test_name] = {
                "status": "failed",
                "error": str(e)
            }
    
    def test_jupyterhub_wrapper_page(self):
        """测试JupyterHub wrapper页面"""
        test_name = "JupyterHub Wrapper页面测试"
        logger.info(f"🧪 开始 {test_name}")
        
        try:
            # 访问JupyterHub wrapper页面
            wrapper_url = f"{self.base_url}/jupyterhub"
            logger.info(f"🌐 访问: {wrapper_url}")
            
            self.driver.get(wrapper_url)
            
            # 等待页面完全加载
            WebDriverWait(self.driver, 15).until(
                lambda driver: driver.execute_script("return document.readyState") == "complete"
            )
            
            # 等待一下让JavaScript执行
            time.sleep(3)
            
            # 检查页面标题
            title = self.driver.title
            logger.info(f"📄 Wrapper页面标题: {title}")
            
            # 检查是否有loading元素
            try:
                loading_element = self.driver.find_element(By.ID, "loading")
                loading_visible = loading_element.is_displayed()
                logger.info(f"⏳ Loading元素可见: {loading_visible}")
            except NoSuchElementException:
                logger.info("⏳ 未找到loading元素")
                loading_visible = False
            
            # 检查是否有错误元素
            try:
                error_element = self.driver.find_element(By.ID, "error")
                error_visible = error_element.is_displayed()
                if error_visible:
                    error_text = error_element.text
                    logger.warning(f"⚠️ 错误元素可见: {error_text}")
                else:
                    logger.info("✅ 没有显示错误")
            except NoSuchElementException:
                logger.info("✅ 未找到错误元素")
                error_visible = False
            
            # 检查iframe元素
            try:
                iframe_element = self.driver.find_element(By.ID, "jupyterhub-frame")
                iframe_src = iframe_element.get_attribute("src")
                iframe_visible = iframe_element.is_displayed()
                
                logger.info(f"🖼️ iframe源: {iframe_src}")
                logger.info(f"🖼️ iframe可见: {iframe_visible}")
                
                # 获取iframe的样式信息
                iframe_style = self.driver.execute_script("""
                    var iframe = document.getElementById('jupyterhub-frame');
                    var computed = window.getComputedStyle(iframe);
                    return {
                        display: computed.display,
                        visibility: computed.visibility,
                        width: computed.width,
                        height: computed.height,
                        opacity: computed.opacity
                    };
                """)
                logger.info(f"🎨 iframe样式: {iframe_style}")
                
            except NoSuchElementException:
                logger.error("❌ 未找到iframe元素")
                iframe_src = None
                iframe_visible = False
                iframe_style = None
            
            # 检查控制台错误
            console_logs = self.driver.get_log('browser')
            console_errors = [log for log in console_logs if log['level'] == 'SEVERE']
            
            if console_errors:
                logger.warning(f"⚠️ 发现 {len(console_errors)} 个控制台错误:")
                for error in console_errors:
                    logger.warning(f"   {error['message']}")
            else:
                logger.info("✅ 没有控制台错误")
            
            # 检查网络请求
            performance_logs = self.driver.get_log('performance')
            network_errors = []
            
            for log in performance_logs:
                message = json.loads(log['message'])
                if message['message']['method'] == 'Network.responseReceived':
                    response = message['message']['params']['response']
                    if response['status'] >= 400:
                        network_errors.append({
                            'url': response['url'],
                            'status': response['status'],
                            'statusText': response['statusText']
                        })
            
            if network_errors:
                logger.warning(f"⚠️ 发现 {len(network_errors)} 个网络错误:")
                for error in network_errors:
                    logger.warning(f"   {error['status']} {error['statusText']}: {error['url']}")
            else:
                logger.info("✅ 没有网络错误")
            
            self.test_results[test_name] = {
                "status": "success",
                "title": title,
                "loading_visible": loading_visible,
                "error_visible": error_visible,
                "iframe_src": iframe_src,
                "iframe_visible": iframe_visible,
                "iframe_style": iframe_style,
                "console_errors": len(console_errors),
                "network_errors": len(network_errors)
            }
            
        except Exception as e:
            logger.error(f"❌ {test_name} 失败: {e}")
            self.test_results[test_name] = {
                "status": "failed",
                "error": str(e)
            }
    
    def test_iframe_content_loading(self):
        """测试iframe内容加载"""
        test_name = "iframe内容加载测试"
        logger.info(f"🧪 开始 {test_name}")
        
        try:
            # 等待iframe加载
            WebDriverWait(self.driver, 20).until(
                EC.presence_of_element_located((By.ID, "jupyterhub-frame"))
            )
            
            iframe = self.driver.find_element(By.ID, "jupyterhub-frame")
            
            # 等待iframe有src属性
            WebDriverWait(self.driver, 10).until(
                lambda driver: iframe.get_attribute("src") is not None
            )
            
            iframe_src = iframe.get_attribute("src")
            logger.info(f"🔗 iframe源URL: {iframe_src}")
            
            # 切换到iframe
            self.driver.switch_to.frame(iframe)
            
            # 等待iframe内容加载
            time.sleep(5)
            
            # 检查iframe内的页面标题
            iframe_title = self.driver.title
            logger.info(f"📄 iframe内页面标题: {iframe_title}")
            
            # 检查iframe内的页面内容
            page_source_length = len(self.driver.page_source)
            logger.info(f"📝 iframe内容长度: {page_source_length} 字符")
            
            # 检查是否有登录表单
            try:
                login_form = self.driver.find_element(By.TAG_NAME, "form")
                logger.info("🔐 发现登录表单")
                has_login_form = True
            except NoSuchElementException:
                logger.info("📝 未发现登录表单")
                has_login_form = False
            
            # 检查是否有JupyterHub特征元素
            jupyter_indicators = []
            try:
                if "jupyter" in self.driver.page_source.lower():
                    jupyter_indicators.append("页面包含'jupyter'文本")
                if "hub" in self.driver.page_source.lower():
                    jupyter_indicators.append("页面包含'hub'文本")
            except:
                pass
            
            logger.info(f"🔍 JupyterHub指标: {jupyter_indicators}")
            
            # 切换回主页面
            self.driver.switch_to.default_content()
            
            self.test_results[test_name] = {
                "status": "success",
                "iframe_src": iframe_src,
                "iframe_title": iframe_title,
                "content_length": page_source_length,
                "has_login_form": has_login_form,
                "jupyter_indicators": jupyter_indicators
            }
            
        except Exception as e:
            logger.error(f"❌ {test_name} 失败: {e}")
            self.driver.switch_to.default_content()  # 确保切换回主页面
            self.test_results[test_name] = {
                "status": "failed",
                "error": str(e)
            }
    
    def test_auth_api(self):
        """测试认证API"""
        test_name = "认证API测试"
        logger.info(f"🧪 开始 {test_name}")
        
        try:
            # 使用JavaScript在浏览器中测试认证API
            auth_test_script = """
            return fetch('/api/auth/login', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    username: 'admin',
                    password: 'admin123'
                })
            })
            .then(response => response.json().then(data => ({
                status: response.status,
                ok: response.ok,
                data: data
            })))
            .catch(error => ({
                status: 'error',
                error: error.message
            }));
            """
            
            result = self.driver.execute_async_script(f"""
                var callback = arguments[arguments.length - 1];
                {auth_test_script.replace('return', '')}
                    .then(callback)
                    .catch(callback);
            """)
            
            logger.info(f"🔑 认证API结果: {result}")
            
            # 检查是否获得了token
            has_token = result.get('data', {}).get('token') is not None
            logger.info(f"🎫 是否获得token: {has_token}")
            
            self.test_results[test_name] = {
                "status": "success" if result.get('ok') else "failed",
                "auth_response": result,
                "has_token": has_token
            }
            
        except Exception as e:
            logger.error(f"❌ {test_name} 失败: {e}")
            self.test_results[test_name] = {
                "status": "failed",
                "error": str(e)
            }
    
    def run_all_tests(self):
        """运行所有测试"""
        logger.info("🚀 开始JupyterHub iframe诊断测试")
        logger.info("=" * 60)
        
        if not self.setup_driver():
            return False
        
        try:
            # 运行测试套件
            self.test_main_page_access()
            time.sleep(2)
            
            self.test_api_health()
            time.sleep(2)
            
            self.test_auth_api()
            time.sleep(2)
            
            self.test_jupyterhub_wrapper_page()
            time.sleep(5)
            
            self.test_iframe_content_loading()
            
            # 生成测试报告
            self.generate_report()
            
        finally:
            if self.driver:
                self.driver.quit()
                logger.info("🔚 浏览器驱动已关闭")
    
    def generate_report(self):
        """生成测试报告"""
        logger.info("=" * 60)
        logger.info("📊 测试报告")
        logger.info("=" * 60)
        
        total_tests = len(self.test_results)
        passed_tests = len([r for r in self.test_results.values() if r.get('status') == 'success'])
        failed_tests = total_tests - passed_tests
        
        logger.info(f"📈 总测试数: {total_tests}")
        logger.info(f"✅ 通过: {passed_tests}")
        logger.info(f"❌ 失败: {failed_tests}")
        logger.info("")
        
        for test_name, result in self.test_results.items():
            status_emoji = "✅" if result.get('status') == 'success' else "❌"
            logger.info(f"{status_emoji} {test_name}: {result.get('status', 'unknown')}")
            
            if result.get('status') == 'failed' and 'error' in result:
                logger.info(f"   错误: {result['error']}")
        
        # 保存详细报告到文件
        report_file = f"jupyterhub_test_report_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json"
        with open(report_file, 'w', encoding='utf-8') as f:
            json.dump(self.test_results, f, indent=2, ensure_ascii=False)
        
        logger.info(f"📄 详细报告已保存到: {report_file}")
        
        # 提供诊断建议
        self.provide_diagnosis()
    
    def provide_diagnosis(self):
        """提供诊断建议"""
        logger.info("=" * 60)
        logger.info("🔍 诊断建议")
        logger.info("=" * 60)
        
        wrapper_test = self.test_results.get("JupyterHub Wrapper页面测试", {})
        iframe_test = self.test_results.get("iframe内容加载测试", {})
        auth_test = self.test_results.get("认证API测试", {})
        
        if not auth_test.get('has_token'):
            logger.warning("🔑 认证问题: 无法获取JWT token")
            logger.info("   建议: 检查后端认证服务是否正常运行")
        
        if wrapper_test.get('loading_visible'):
            logger.warning("⏳ 页面状态: Loading元素仍然可见")
            logger.info("   建议: JavaScript可能没有正确执行或API调用失败")
        
        if wrapper_test.get('error_visible'):
            logger.warning("⚠️ 错误状态: 页面显示错误信息")
            logger.info("   建议: 检查错误详情和网络请求")
        
        if not wrapper_test.get('iframe_visible'):
            logger.warning("🖼️ iframe问题: iframe不可见")
            logger.info("   建议: 检查CSS样式和JavaScript逻辑")
        
        if wrapper_test.get('console_errors', 0) > 0:
            logger.warning(f"💥 JavaScript错误: {wrapper_test.get('console_errors')} 个控制台错误")
            logger.info("   建议: 查看浏览器控制台获取详细错误信息")
        
        if wrapper_test.get('network_errors', 0) > 0:
            logger.warning(f"🌐 网络错误: {wrapper_test.get('network_errors')} 个网络请求失败")
            logger.info("   建议: 检查nginx配置和服务连接")

def main():
    """主函数"""
    import argparse
    
    parser = argparse.ArgumentParser(description='JupyterHub iframe 浏览器自动化测试')
    parser.add_argument('--url', default='http://localhost:8080', help='基础URL')
    parser.add_argument('--headless', action='store_true', help='无头模式运行')
    
    args = parser.parse_args()
    
    tester = JupyterHubTester(base_url=args.url, headless=args.headless)
    tester.run_all_tests()

if __name__ == "__main__":
    main()
