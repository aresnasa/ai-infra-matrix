#!/usr/bin/env python3
"""
清理浏览器cookie并测试JupyterHub登录
模拟完整的浏览器访问流程，包括静态资源加载
"""
import requests
import re
import sys

def test_jupyterhub_login():
    """测试JupyterHub登录流程"""
    session = requests.Session()
    session.headers.update({
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36'
    })
    
    print("🧪 测试JupyterHub完整登录流程...")
    
    # 1. 测试重定向
    print("\n1. 测试重定向路径:")
    redirects = [
        "http://localhost:8080/jupyter",
        "http://localhost:8080/jupyter/",
        "http://localhost:8080/jupyter/hub/login"
    ]
    
    for url in redirects:
        try:
            resp = session.get(url, allow_redirects=False)
            print(f"   {url}")
            print(f"   状态码: {resp.status_code}")
            if 'Location' in resp.headers:
                print(f"   重定向到: {resp.headers['Location']}")
            print()
        except Exception as e:
            print(f"   错误: {e}")
    
    # 2. 测试登录页面和静态资源
    print("\n2. 测试登录页面和静态资源:")
    try:
        resp = session.get("http://localhost:8080/jupyter/hub/login")
        print(f"   登录页面状态码: {resp.status_code}")
        print(f"   内容长度: {len(resp.text)} 字节")
        
        # 检查关键元素
        if "JupyterHub" in resp.text:
            print("   ✅ 包含JupyterHub标题")
        if 'name="username"' in resp.text:
            print("   ✅ 包含用户名输入框")
        if 'name="password"' in resp.text:
            print("   ✅ 包含密码输入框")
        if '_xsrf' in resp.text:
            print("   ✅ 包含XSRF token")
        
        # 提取并测试静态资源链接
        print("\n   测试关键静态资源:")
        static_resources = [
            "http://localhost:8080/jupyter/hub/static/css/style.min.css",
            "http://localhost:8080/jupyter/hub/static/components/jquery/dist/jquery.min.js",
            "http://localhost:8080/jupyter/hub/static/components/bootstrap/dist/js/bootstrap.bundle.min.js"
        ]
        
        for resource in static_resources:
            try:
                static_resp = session.head(resource)
                status = "✅" if static_resp.status_code == 200 else "❌"
                print(f"   {status} {resource.split('/')[-1]}: {static_resp.status_code}")
            except Exception as e:
                print(f"   ❌ {resource.split('/')[-1]}: 错误 {e}")
            
        # 检查cookie
        print(f"\n   Cookies: {len(session.cookies)} 个")
        for cookie in session.cookies:
            print(f"   - {cookie.name}: {cookie.value[:30]}...")
        
    except Exception as e:
        print(f"   错误: {e}")

    # 3. 尝试登录
    print("\n3. 尝试模拟登录:")
    try:
        # 获取XSRF token
        login_resp = session.get("http://localhost:8080/jupyter/hub/login")
        xsrf_match = re.search(r'name="_xsrf".*?value="([^"]*)"', login_resp.text)
        
        if xsrf_match:
            xsrf_token = xsrf_match.group(1)
            print(f"   ✅ 获取XSRF token: {xsrf_token[:20]}...")
            
            # 提交登录表单
            login_data = {
                'username': 'testuser',
                'password': 'password',
                '_xsrf': xsrf_token
            }
            
            post_resp = session.post("http://localhost:8080/jupyter/hub/login", 
                                   data=login_data, allow_redirects=False)
            print(f"   登录提交状态码: {post_resp.status_code}")
            
            if post_resp.status_code == 302:
                redirect_to = post_resp.headers.get('Location', '')
                print(f"   ✅ 登录成功，重定向到: {redirect_to}")
                
                # 检查认证cookie
                auth_cookies = []
                for cookie in session.cookies:
                    if any(keyword in cookie.name.lower() for keyword in ['hub', 'auth', 'session']):
                        auth_cookies.append(cookie.name)
                
                if auth_cookies:
                    print(f"   ✅ 获得认证cookie: {auth_cookies}")
                    
                    # 测试认证后的页面访问
                    print("\n4. 测试认证后的访问:")
                    test_pages = [
                        ("用户主页", "http://localhost:8080/jupyter/hub/home"),
                        ("用户API", "http://localhost:8080/jupyter/hub/api/user")
                    ]
                    
                    for name, url in test_pages:
                        try:
                            page_resp = session.get(url, allow_redirects=False)
                            status = "✅" if page_resp.status_code == 200 else "❌"
                            print(f"   {status} {name}: {page_resp.status_code}")
                        except Exception as e:
                            print(f"   ❌ {name}: 错误 {e}")
                else:
                    print("   ⚠️  未检测到认证cookie")
            else:
                print(f"   ❌ 登录失败: {post_resp.status_code}")
        else:
            print("   ❌ 无法获取XSRF token")
            
    except Exception as e:
        print(f"   登录测试错误: {e}")

if __name__ == "__main__":
    test_jupyterhub_login()
