#!/usr/bin/env python3
"""
浏览器JavaScript执行测试 - 检查是否有客户端重定向
"""

import requests
import time
import re

def test_javascript_redirects():
    """测试是否存在JavaScript重定向或客户端路由干扰"""
    
    print("🔍 JavaScript重定向检测测试")
    print("=" * 60)
    print("🎯 检查 /jupyterhub 是否存在客户端重定向或路由干扰")
    print()
    
    url = "http://localhost:8080/jupyterhub"
    
    # 使用真实浏览器头部
    headers = {
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36',
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8',
        'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
        'Accept-Encoding': 'gzip, deflate',
        'Cache-Control': 'no-cache',
        'Pragma': 'no-cache'
    }
    
    try:
        print("📥 发送请求到 /jupyterhub...")
        response = requests.get(url, headers=headers, timeout=15, allow_redirects=False)
        
        print(f"📊 HTTP状态码: {response.status_code}")
        print(f"📏 内容长度: {len(response.content)} bytes")
        
        # 检查HTTP重定向
        if response.status_code in [301, 302, 303, 307, 308]:
            location = response.headers.get('Location', 'N/A')
            print(f"🔀 HTTP重定向: {response.status_code} -> {location}")
            print("❌ 检测到HTTP级重定向！")
            return False
        
        # 分析内容
        content = response.content.decode('utf-8', errors='ignore')
        
        # 检查内容类型标识
        print("\n🔍 内容分析:")
        
        # 1. 检查是否是门户页面
        is_portal = 'AI基础设施矩阵' in content and 'linear-gradient' in content
        print(f"🏠 门户页面特征: {'✅' if is_portal else '❌'}")
        
        # 2. 检查是否包含React应用代码
        react_indicators = [
            'react', 'useState', 'useEffect', 'ReactDOM',
            'bundle.js', 'app.js', '__REACT_DEVTOOLS',
            'fetchHubStatus', 'userTasks', 'hubStatus'
        ]
        
        found_react = []
        for indicator in react_indicators:
            if indicator.lower() in content.lower():
                found_react.append(indicator)
        
        if found_react:
            print(f"⚛️ React代码标识: ❌ 找到 {found_react}")
        else:
            print(f"⚛️ React代码标识: ✅ 未找到")
        
        # 3. 检查JavaScript重定向
        js_redirect_patterns = [
            r'window\.location\s*=',
            r'window\.location\.href\s*=',
            r'window\.location\.replace\(',
            r'window\.location\.assign\(',
            r'history\.pushState\(',
            r'history\.replaceState\(',
            r'router\.push\(',
            r'router\.replace\(',
            r'location\.hash\s*='
        ]
        
        found_redirects = []
        for pattern in js_redirect_patterns:
            matches = re.findall(pattern, content, re.IGNORECASE)
            if matches:
                found_redirects.append(pattern)
        
        if found_redirects:
            print(f"🔀 JavaScript重定向: ❌ 找到模式 {found_redirects}")
        else:
            print(f"🔀 JavaScript重定向: ✅ 未找到")
        
        # 4. 检查SPA路由器代码
        spa_router_patterns = [
            r'react-router',
            r'BrowserRouter',
            r'Route\s+path',
            r'<Route',
            r'useNavigate',
            r'useHistory'
        ]
        
        found_spa = []
        for pattern in spa_router_patterns:
            matches = re.findall(pattern, content, re.IGNORECASE)
            if matches:
                found_spa.append(pattern)
        
        if found_spa:
            print(f"🛤️ SPA路由器: ❌ 找到 {found_spa}")
        else:
            print(f"🛤️ SPA路由器: ✅ 未找到")
        
        # 5. 检查模块加载器
        module_patterns = [
            r'import\s+.*from',
            r'require\(',
            r'webpack',
            r'__webpack',
            r'module\.exports'
        ]
        
        found_modules = []
        for pattern in module_patterns:
            matches = re.findall(pattern, content, re.IGNORECASE)
            if matches:
                found_modules.append(pattern)
        
        if found_modules:
            print(f"📦 模块加载器: ❌ 找到 {found_modules}")
        else:
            print(f"📦 模块加载器: ✅ 未找到")
        
        # 6. 检查异步加载的脚本
        script_tags = re.findall(r'<script[^>]*src=["\']([^"\']*)["\'][^>]*>', content, re.IGNORECASE)
        external_scripts = [src for src in script_tags if src and not src.startswith('#')]
        
        if external_scripts:
            print(f"📜 外部脚本: ❌ 找到 {external_scripts}")
        else:
            print(f"📜 外部脚本: ✅ 未找到")
        
        # 7. 检查内联脚本内容
        inline_scripts = re.findall(r'<script[^>]*>(.*?)</script>', content, re.DOTALL | re.IGNORECASE)
        problematic_inline = []
        
        for script in inline_scripts:
            script_lower = script.lower()
            if any(keyword in script_lower for keyword in ['location', 'router', 'navigate', 'redirect']):
                problematic_inline.append(script[:100] + '...' if len(script) > 100 else script)
        
        if problematic_inline:
            print(f"📝 可疑内联脚本: ❌ 找到 {len(problematic_inline)} 个")
            for script in problematic_inline[:2]:  # 只显示前2个
                print(f"     {script}")
        else:
            print(f"📝 可疑内联脚本: ✅ 未找到")
        
        print()
        
        # 综合评估
        all_issues = found_react + found_redirects + found_spa + found_modules + external_scripts + problematic_inline
        
        if not all_issues and is_portal:
            print("🎉 检测通过！")
            print("✅ 返回纯净的门户页面")
            print("✅ 没有React应用干扰")
            print("✅ 没有JavaScript重定向")
            print("✅ 没有SPA路由器代码")
            return True
        else:
            print("❌ 检测失败！")
            if not is_portal:
                print("   门户页面特征缺失")
            if all_issues:
                print(f"   发现 {len(all_issues)} 个潜在问题")
            
            print("\n💡 建议解决方案:")
            if found_react:
                print("   1. 确认React应用不会渲染到 /jupyterhub 路径")
            if found_redirects or found_spa:
                print("   2. 检查是否有客户端路由在页面加载后执行")
            if external_scripts:
                print("   3. 检查外部脚本是否来自前端应用")
            
            return False
            
    except requests.RequestException as e:
        print(f"❌ 请求失败: {e}")
        return False

if __name__ == "__main__":
    success = test_javascript_redirects()
    exit(0 if success else 1)
