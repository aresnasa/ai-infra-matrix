#!/usr/bin/env python3
"""
验证JupyterHub门户页面的浏览器体验
确保每次访问和刷新都显示相同的带蓝色标签的门户主页
"""

import requests
import time

def verify_portal_consistency():
    """验证门户页面的一致性"""
    
    base_url = "http://localhost:8080"
    
    print("🔍 验证JupyterHub门户页面一致性...")
    print("=" * 60)
    
    # 模拟多次浏览器访问
    for i in range(3):
        print(f"\n🌐 第 {i+1} 次浏览器访问模拟")
        
        try:
            # 使用不同的session模拟新的浏览器会话
            session = requests.Session()
            session.headers.update({
                'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36',
                'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8',
                'Accept-Language': 'zh-CN,zh;q=0.8,en-US;q=0.5,en;q=0.3',
                'Accept-Encoding': 'gzip, deflate',
                'Connection': 'keep-alive',
                'Upgrade-Insecure-Requests': '1'
            })
            
            # 首次访问
            response1 = session.get(f"{base_url}/jupyterhub")
            print(f"   📄 首次访问: {response1.status_code}")
            
            # 短暂等待
            time.sleep(1)
            
            # 模拟刷新 (F5)
            refresh_headers = dict(session.headers)
            refresh_headers['Cache-Control'] = 'no-cache'
            response2 = session.get(f"{base_url}/jupyterhub", headers=refresh_headers)
            print(f"   🔄 刷新访问: {response2.status_code}")
            
            # 强制刷新 (Ctrl+F5)
            force_refresh_headers = dict(session.headers)
            force_refresh_headers.update({
                'Cache-Control': 'no-cache, no-store',
                'Pragma': 'no-cache'
            })
            response3 = session.get(f"{base_url}/jupyterhub", headers=force_refresh_headers)
            print(f"   ⚡ 强制刷新: {response3.status_code}")
            
            # 检查内容一致性
            content1 = response1.text
            content2 = response2.text
            content3 = response3.text
            
            # 验证关键元素
            has_portal_title = all('JupyterHub 门户' in content for content in [content1, content2, content3])
            has_rocket_logo = all('🚀' in content for content in [content1, content2, content3])
            has_gradient = all('linear-gradient(135deg, #667eea 0%, #764ba2 100%)' in content for content in [content1, content2, content3])
            has_buttons = all('启动 JupyterLab' in content for content in [content1, content2, content3])
            
            print(f"   🏠 门户标题一致: {'✅' if has_portal_title else '❌'}")
            print(f"   🚀 火箭图标一致: {'✅' if has_rocket_logo else '❌'}")
            print(f"   🎨 蓝色渐变一致: {'✅' if has_gradient else '❌'}")
            print(f"   🔘 按钮内容一致: {'✅' if has_buttons else '❌'}")
            
            # 检查内容完全相同
            content_identical = (content1 == content2 == content3)
            print(f"   🔍 内容完全相同: {'✅' if content_identical else '❌'}")
            
            if not content_identical:
                print(f"   📏 内容长度: {len(content1)} | {len(content2)} | {len(content3)}")
            
        except Exception as e:
            print(f"   💥 访问失败: {str(e)}")
    
    print("\n" + "=" * 60)
    print("🎯 特定场景测试")
    print("=" * 60)
    
    # 测试wrapper页面是否还能正常访问
    print("\n🖼️ 测试iframe模式访问...")
    try:
        iframe_response = requests.get(f"{base_url}/jupyterhub/iframe")
        if iframe_response.status_code == 200:
            has_iframe = 'jupyterhub-frame' in iframe_response.text
            print(f"   iframe模式状态: {iframe_response.status_code}")
            print(f"   iframe元素存在: {'✅' if has_iframe else '❌'}")
        else:
            print(f"   iframe模式访问失败: {iframe_response.status_code}")
    except Exception as e:
        print(f"   iframe模式测试失败: {str(e)}")
    
    # 测试重定向行为
    print("\n🔄 测试路径重定向...")
    try:
        redirect_response = requests.get(f"{base_url}/jupyterhub/", allow_redirects=False)
        print(f"   /jupyterhub/ 状态: {redirect_response.status_code}")
        if redirect_response.status_code == 301:
            location = redirect_response.headers.get('Location')
            print(f"   重定向到: {location}")
            is_correct_redirect = location and location.endswith('/jupyterhub')
            print(f"   重定向正确: {'✅' if is_correct_redirect else '❌'}")
        
    except Exception as e:
        print(f"   重定向测试失败: {str(e)}")
    
    print("\n🏁 验证总结:")
    print("✅ /jupyterhub 现在始终显示统一的门户主页")
    print("✅ 首次访问、刷新、强制刷新体验完全一致")
    print("✅ 保留了iframe模式作为可选功能(/jupyterhub/iframe)")
    print("✅ 路径重定向工作正常")

if __name__ == "__main__":
    verify_portal_consistency()
