#!/usr/bin/env python3
"""
JupyterHub URL重定向修复验证
"""

import requests
import time
from urllib.parse import urlparse

def test_jupyter_redirects():
    """测试JupyterHub URL重定向链路"""
    
    print("🔧 JupyterHub URL重定向修复验证")
    print("=" * 60)
    
    base_url = "http://localhost:8080"
    
    # 测试步骤1: /jupyter -> /jupyter/
    print("📝 步骤1: 测试 /jupyter -> /jupyter/ 重定向")
    try:
        response = requests.get(f"{base_url}/jupyter", allow_redirects=False)
        print(f"   状态码: {response.status_code}")
        
        if response.status_code in [301, 302]:
            location = response.headers.get('Location', '')
            print(f"   重定向到: {location}")
            
            parsed = urlparse(location)
            if parsed.netloc == "localhost:8080" and parsed.path == "/jupyter/":
                print("   ✅ 重定向正确")
                step1_success = True
            else:
                print("   ❌ 重定向错误")
                step1_success = False
        else:
            print("   ❌ 应该是重定向响应")
            step1_success = False
            
    except Exception as e:
        print(f"   ❌ 请求失败: {e}")
        step1_success = False
    
    # 测试步骤2: /jupyter/ -> /jupyter/hub/
    print("\n📝 步骤2: 测试 /jupyter/ -> /jupyter/hub/ 重定向")
    try:
        response = requests.get(f"{base_url}/jupyter/", allow_redirects=False)
        print(f"   状态码: {response.status_code}")
        
        if response.status_code in [301, 302]:
            location = response.headers.get('Location', '')
            print(f"   重定向到: {location}")
            
            parsed = urlparse(location)
            if parsed.netloc == "localhost:8080" and parsed.path == "/jupyter/hub/":
                print("   ✅ 重定向正确")
                step2_success = True
            else:
                print("   ❌ 重定向错误")
                step2_success = False
        else:
            print("   ❌ 应该是重定向响应")
            step2_success = False
            
    except Exception as e:
        print(f"   ❌ 请求失败: {e}")
        step2_success = False
    
    # 测试步骤3: /jupyter/hub/ 最终页面
    print("\n📝 步骤3: 测试 /jupyter/hub/ 最终页面")
    try:
        response = requests.get(f"{base_url}/jupyter/hub/", allow_redirects=True, timeout=10)
        print(f"   状态码: {response.status_code}")
        print(f"   最终URL: {response.url}")
        
        if response.status_code == 200:
            content = response.text.lower()
            
            # 检查内容特征
            indicators = [
                ('jupyter' in content, 'Jupyter关键词'),
                ('hub' in content, 'Hub关键词'),
                ('login' in content, '登录页面'),
                ('<!doctype html>' in content, 'HTML文档'),
                (len(content) > 1000, '内容丰富')
            ]
            
            success_count = sum(found for found, _ in indicators)
            print(f"   内容检查: {success_count}/{len(indicators)} 通过")
            
            for found, desc in indicators:
                status = "✅" if found else "❌"
                print(f"     {status} {desc}")
            
            step3_success = response.status_code == 200 and success_count >= 3
            
            if step3_success:
                print("   ✅ JupyterHub页面正常")
            else:
                print("   ⚠️ JupyterHub页面可能有问题")
                
        else:
            print("   ❌ JupyterHub页面访问失败")
            step3_success = False
            
    except Exception as e:
        print(f"   ❌ 请求失败: {e}")
        step3_success = False
    
    # 测试步骤4: 完整重定向链路
    print("\n📝 步骤4: 测试完整重定向链路")
    try:
        session = requests.Session()
        
        # 从 /jupyter 开始，跟随所有重定向
        response = session.get(f"{base_url}/jupyter", allow_redirects=True, timeout=15)
        print(f"   最终状态码: {response.status_code}")
        print(f"   最终URL: {response.url}")
        
        # 检查URL是否正确
        parsed_final = urlparse(response.url)
        url_correct = (parsed_final.netloc == "localhost:8080" and 
                      parsed_final.path.startswith("/jupyter/"))
        
        if response.status_code == 200 and url_correct:
            print("   ✅ 完整链路正常")
            step4_success = True
        else:
            print("   ❌ 完整链路有问题")
            step4_success = False
            
    except Exception as e:
        print(f"   ❌ 完整链路测试失败: {e}")
        step4_success = False
    
    # 生成总结
    print("\n" + "=" * 60)
    print("📊 修复验证结果")
    print("=" * 60)
    
    steps = [
        ("URL重定向 /jupyter", step1_success),
        ("路径重定向 /jupyter/", step2_success),
        ("JupyterHub页面访问", step3_success),
        ("完整重定向链路", step4_success)
    ]
    
    success_count = sum(success for _, success in steps)
    
    for step_name, success in steps:
        status = "✅" if success else "❌"
        print(f"{step_name:<20}: {status}")
    
    overall_success = success_count == len(steps)
    
    print(f"\n🎯 总体结果: {'✅ 修复成功' if overall_success else '❌ 仍有问题'} ({success_count}/{len(steps)})")
    
    if overall_success:
        print("\n🎉 URL重定向问题已完全修复!")
        print("💡 用户现在可以正常访问:")
        print("   • http://localhost:8080/jupyter")
        print("   • http://localhost:8080/jupyter/")
        print("   • http://localhost:8080/jupyter/hub/")
        print("   • 不再跳转到 http://localhost/jupyter")
    else:
        print("\n⚠️ 仍需进一步调试")
        
        if not step1_success:
            print("   - /jupyter 重定向需要修复")
        if not step2_success:
            print("   - /jupyter/ 重定向需要修复")
        if not step3_success:
            print("   - JupyterHub服务可能有问题")
        if not step4_success:
            print("   - 完整访问链路需要检查")
    
    return overall_success

if __name__ == "__main__":
    test_jupyter_redirects()
