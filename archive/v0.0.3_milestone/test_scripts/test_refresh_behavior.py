#!/usr/bin/env python3
"""
浏览器刷新行为测试：验证首次访问和刷新的一致性
"""

import requests
import time
import hashlib
import json

def test_browser_refresh_behavior():
    """测试浏览器刷新行为"""
    
    print("🔄 浏览器刷新行为测试")
    print("=" * 60)
    print("🎯 目标: 验证首次访问和刷新 /jupyterhub 的一致性")
    print()
    
    url = "http://localhost:8080/jupyterhub"
    
    # 模拟真实浏览器行为的头部
    browser_headers = {
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36',
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7',
        'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
        'Accept-Encoding': 'gzip, deflate',
        'Cache-Control': 'no-cache',
        'Pragma': 'no-cache',
        'Upgrade-Insecure-Requests': '1'
    }
    
    # 模拟F5刷新的头部
    refresh_headers = browser_headers.copy()
    refresh_headers.update({
        'Cache-Control': 'max-age=0',
        'Pragma': 'no-cache'
    })
    
    # 模拟硬刷新(Ctrl+F5)的头部  
    hard_refresh_headers = browser_headers.copy()
    hard_refresh_headers.update({
        'Cache-Control': 'no-cache, no-store, must-revalidate',
        'Pragma': 'no-cache',
        'Expires': '0'
    })
    
    scenarios = [
        ("首次访问", browser_headers),
        ("F5刷新", refresh_headers),
        ("硬刷新(Ctrl+F5)", hard_refresh_headers),
        ("再次访问", browser_headers),
        ("等待后访问", browser_headers)
    ]
    
    results = []
    
    for i, (scenario_name, headers) in enumerate(scenarios):
        print(f"📱 场景 {i+1}: {scenario_name}")
        
        if scenario_name == "等待后访问":
            print("   ⏳ 等待3秒模拟用户停顿...")
            time.sleep(3)
        
        try:
            response = requests.get(url, headers=headers, timeout=10)
            content = response.content.decode('utf-8', errors='ignore')
            content_hash = hashlib.md5(response.content).hexdigest()[:16]
            content_length = len(response.content)
            
            # 分析内容类型
            is_portal_page = 'AI基础设施矩阵' in content and 'linear-gradient' in content
            is_wrapper_page = 'jupyterhub_wrapper' in content or 'iframe' in content
            is_react_page = 'JupyterHub' in content and ('userTasks' in content or 'hubStatus' in content)
            
            content_type = "未知"
            if is_portal_page:
                content_type = "门户页面"
            elif is_wrapper_page:
                content_type = "包装页面"
            elif is_react_page:
                content_type = "React组件"
            
            print(f"   📊 状态码: {response.status_code}")
            print(f"   📏 内容长度: {content_length} bytes")
            print(f"   🔑 内容哈希: {content_hash}...")
            print(f"   📄 页面类型: {content_type}")
            print(f"   🎨 门户特征: {'✅' if is_portal_page else '❌'}")
            
            results.append({
                'scenario': scenario_name,
                'status': response.status_code,
                'length': content_length,
                'hash': content_hash,
                'type': content_type,
                'is_portal': is_portal_page,
                'is_wrapper': is_wrapper_page,
                'is_react': is_react_page
            })
            
            print()
            
        except requests.RequestException as e:
            print(f"   ❌ 请求失败: {e}")
            print()
    
    # 分析结果
    print("=" * 60)
    print("📊 刷新行为分析")
    print("=" * 60)
    
    if not results:
        print("❌ 没有成功的测试结果")
        return False
    
    # 检查一致性
    first_result = results[0]
    all_same_type = all(r['type'] == first_result['type'] for r in results)
    all_same_hash = all(r['hash'] == first_result['hash'] for r in results)
    all_portal = all(r['is_portal'] for r in results)
    
    print(f"📄 页面类型一致性: {'✅' if all_same_type else '❌'}")
    print(f"🔑 内容哈希一致性: {'✅' if all_same_hash else '❌'}")
    print(f"🏠 全部为门户页面: {'✅' if all_portal else '❌'}")
    print()
    
    # 详细结果
    for result in results:
        icon = "✅" if result['is_portal'] else "❌"
        print(f"{icon} {result['scenario']}: {result['type']} ({result['length']} bytes)")
    
    print()
    
    # 最终判断
    if all_same_type and all_portal and all_same_hash:
        print("🎉 测试通过！")
        print("✅ 首次访问和所有刷新方式都返回相同的门户页面")
        print("✅ 没有React路由干扰")
        print("✅ 浏览器刷新行为一致")
        return True
    else:
        print("❌ 测试失败！")
        if not all_same_type:
            print("   不同访问方式返回不同类型的页面")
        if not all_portal:
            print("   某些访问返回了非门户页面")
        if not all_same_hash:
            print("   内容哈希不一致")
        
        # 显示问题详情
        print("\n🔍 问题详情:")
        for result in results:
            if not result['is_portal']:
                print(f"   ⚠️  {result['scenario']}: {result['type']}")
        
        return False

if __name__ == "__main__":
    success = test_browser_refresh_behavior()
    exit(0 if success else 1)
