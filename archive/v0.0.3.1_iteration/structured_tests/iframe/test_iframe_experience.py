#!/usr/bin/env python3
"""
JupyterHub iframe内容验证测试
检查iframe内部是否正确加载了JupyterHub内容
"""

import requests
import time
import json
from urllib.parse import urljoin, urlparse

def test_iframe_content_flow():
    """完整测试iframe内容加载流程"""
    
    base_url = "http://localhost:8080"
    
    print("🚀 开始iframe内容流程测试")
    print("=" * 60)
    
    # 步骤1: 获取JWT认证令牌
    print("1️⃣ 获取JWT认证令牌...")
    try:
        auth_response = requests.post(f"{base_url}/api/auth/login", 
            json={"username": "admin", "password": "admin123"},
            timeout=10
        )
        
        if auth_response.status_code != 200:
            print(f"   ❌ 认证失败: {auth_response.status_code}")
            return False
            
        token_data = auth_response.json()
        if 'token' not in token_data:
            print("   ❌ 响应中没有令牌")
            return False
            
        token = token_data['token']
        print(f"   ✅ JWT令牌获取成功，长度: {len(token)}")
        
    except Exception as e:
        print(f"   ❌ 认证过程出错: {e}")
        return False
    
    # 步骤2: 测试JupyterHub wrapper页面
    print("\n2️⃣ 测试wrapper页面...")
    try:
        wrapper_response = requests.get(f"{base_url}/jupyterhub", timeout=10)
        if wrapper_response.status_code == 200:
            print(f"   ✅ wrapper页面正常 ({wrapper_response.status_code})")
            
            # 检查页面内容
            content = wrapper_response.text.lower()
            checks = {
                "iframe元素": "jupyterhub-frame" in content,
                "JavaScript代码": "loadjupyterhub" in content,
                "认证功能": "getauthtoken" in content,
                "错误处理": "showerror" in content
            }
            
            for check_name, passed in checks.items():
                status = "✅" if passed else "❌"
                print(f"   {status} {check_name}")
                
        else:
            print(f"   ❌ wrapper页面错误: {wrapper_response.status_code}")
            return False
            
    except Exception as e:
        print(f"   ❌ wrapper页面测试失败: {e}")
        return False
    
    # 步骤3: 测试带令牌的JupyterHub直接访问
    print("\n3️⃣ 测试带令牌的JupyterHub访问...")
    try:
        jupyter_url = f"{base_url}/jupyter/hub/?token={token}"
        jupyter_response = requests.get(jupyter_url, timeout=15, allow_redirects=True)
        
        print(f"   📍 访问URL: {jupyter_url[:80]}...")
        print(f"   📊 状态码: {jupyter_response.status_code}")
        print(f"   📄 Content-Type: {jupyter_response.headers.get('content-type', 'unknown')}")
        
        if jupyter_response.status_code == 200:
            content = jupyter_response.text.lower()
            
            # 检查响应内容类型
            content_checks = {
                "HTML内容": "<!doctype html>" in content or "<html" in content,
                "JupyterHub内容": "jupyter" in content,
                "登录页面": "login" in content or "sign" in content,
                "控制面板": "dashboard" in content or "control" in content,
                "样式文件": "<link" in content or "css" in content,
                "JavaScript": "<script" in content or ".js" in content
            }
            
            for check_name, passed in content_checks.items():
                status = "✅" if passed else "❌"
                print(f"   {status} {check_name}: {passed}")
                
            # 检查CSP头
            csp_header = jupyter_response.headers.get('content-security-policy', '')
            if csp_header:
                print(f"   🔒 CSP头: {csp_header[:100]}...")
                if 'frame-ancestors' in csp_header.lower():
                    print("   ✅ CSP包含frame-ancestors配置")
                else:
                    print("   ⚠️ CSP缺少frame-ancestors配置")
            else:
                print("   ⚠️ 没有CSP头")
                
        else:
            print(f"   ❌ JupyterHub访问失败: {jupyter_response.status_code}")
            if jupyter_response.status_code == 302:
                location = jupyter_response.headers.get('location', '')
                print(f"   🔄 重定向到: {location}")
                
    except Exception as e:
        print(f"   ❌ JupyterHub访问测试失败: {e}")
        return False
    
    # 步骤4: 测试无令牌的JupyterHub访问（应该显示登录页面）
    print("\n4️⃣ 测试无令牌的JupyterHub访问...")
    try:
        no_token_url = f"{base_url}/jupyter/hub/"
        no_token_response = requests.get(no_token_url, timeout=10)
        
        print(f"   📊 状态码: {no_token_response.status_code}")
        
        if no_token_response.status_code == 200:
            content = no_token_response.text.lower()
            if "login" in content or "sign" in content:
                print("   ✅ 正确显示登录页面")
            else:
                print("   ⚠️ 没有明显的登录内容")
        elif no_token_response.status_code == 302:
            location = no_token_response.headers.get('location', '')
            print(f"   🔄 重定向到登录页面: {location}")
        else:
            print(f"   ⚠️ 意外的响应状态: {no_token_response.status_code}")
            
    except Exception as e:
        print(f"   ❌ 无令牌访问测试失败: {e}")
    
    # 步骤5: 检查潜在的iframe阻塞因素
    print("\n5️⃣ 检查iframe阻塞因素...")
    
    # 检查X-Frame-Options头
    headers_to_check = ['x-frame-options', 'content-security-policy', 'x-content-type-options']
    
    for header in headers_to_check:
        header_value = jupyter_response.headers.get(header, '')
        if header_value:
            print(f"   📋 {header}: {header_value}")
            
            if header == 'x-frame-options' and header_value.lower() in ['deny', 'sameorigin']:
                print(f"   ⚠️ {header}可能阻止iframe嵌入")
        else:
            print(f"   ✅ 没有{header}头")
    
    print("\n" + "=" * 60)
    print("🏁 测试完成")
    
    return True

def create_test_html():
    """创建一个简单的测试HTML页面来验证iframe"""
    
    html_content = '''<!DOCTYPE html>
<html>
<head>
    <title>iframe测试页面</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .test-frame { width: 100%; height: 600px; border: 2px solid #ccc; }
        .status { margin: 10px 0; padding: 10px; background: #f5f5f5; }
    </style>
</head>
<body>
    <h1>🧪 JupyterHub iframe测试</h1>
    
    <div class="status">
        <strong>测试说明:</strong> 如果下面的iframe显示JupyterHub内容，说明iframe功能正常。
    </div>
    
    <h2>测试1: 直接iframe嵌入</h2>
    <iframe 
        id="test-frame-1" 
        class="test-frame"
        src="http://localhost:8080/jupyter/hub/"
        sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-top-navigation">
        您的浏览器不支持iframe
    </iframe>
    
    <h2>测试2: 使用wrapper页面</h2>
    <iframe 
        id="test-frame-2" 
        class="test-frame"
        src="http://localhost:8080/jupyterhub"
        sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-top-navigation">
        您的浏览器不支持iframe
    </iframe>
    
    <script>
        // 监听iframe加载事件
        function setupIframeMonitoring(iframeId, name) {
            const iframe = document.getElementById(iframeId);
            
            iframe.onload = function() {
                console.log(`${name} - iframe加载完成`);
                try {
                    // 尝试访问iframe内容（可能因为同源策略而失败）
                    const iframeDoc = iframe.contentDocument || iframe.contentWindow.document;
                    console.log(`${name} - 可以访问iframe内容`);
                } catch (e) {
                    console.log(`${name} - 无法访问iframe内容（正常的跨域限制）: ${e.message}`);
                }
            };
            
            iframe.onerror = function() {
                console.error(`${name} - iframe加载错误`);
            };
        }
        
        setupIframeMonitoring('test-frame-1', '直接嵌入');
        setupIframeMonitoring('test-frame-2', 'Wrapper页面');
        
        // 5秒后检查iframe状态
        setTimeout(() => {
            ['test-frame-1', 'test-frame-2'].forEach((id, index) => {
                const iframe = document.getElementById(id);
                const name = index === 0 ? '直接嵌入' : 'Wrapper页面';
                
                if (iframe.contentWindow) {
                    console.log(`${name} - iframe窗口存在`);
                } else {
                    console.log(`${name} - iframe窗口不存在`);
                }
            });
        }, 5000);
    </script>
</body>
</html>'''
    
    with open('/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/iframe_test.html', 'w', encoding='utf-8') as f:
        f.write(html_content)
    
    print("✅ 创建了iframe测试页面: iframe_test.html")
    print("💡 在浏览器中打开此文件进行可视化测试")

if __name__ == "__main__":
    # 运行内容流程测试
    test_iframe_content_flow()
    
    # 创建测试HTML页面
    print("\n" + "=" * 60)
    create_test_html()
    
    print("\n🔧 下一步建议:")
    print("1. 在浏览器中打开 file:///Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/iframe_test.html")
    print("2. 检查开发者工具的控制台日志")
    print("3. 观察iframe是否正确显示内容")
    print("4. 如果仍然白屏，检查网络标签页的请求状态")
