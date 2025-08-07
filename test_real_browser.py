#!/usr/bin/env python3
"""
真实浏览器行为模拟测试
使用更接近浏览器的方式测试，包括会话管理和缓存行为
"""

import requests
import time
import hashlib

def test_real_browser_behavior():
    """模拟真实浏览器行为测试"""
    
    print("🌐 真实浏览器行为模拟测试")
    print("=" * 60)
    
    # 创建一个会话来模拟浏览器的持久连接和cookie管理
    session = requests.Session()
    
    # 设置真实浏览器的头部
    session.headers.update({
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36',
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7',
        'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
        'Accept-Encoding': 'gzip, deflate, br',
        'Connection': 'keep-alive',
        'Upgrade-Insecure-Requests': '1',
        'Sec-Fetch-Dest': 'document',
        'Sec-Fetch-Mode': 'navigate',
        'Sec-Fetch-Site': 'none',
        'Sec-Fetch-User': '?1'
    })
    
    url = "http://localhost:8080/jupyterhub"
    
    scenarios = [
        {
            'name': '🌐 首次浏览器访问',
            'headers': {
                'Cache-Control': 'max-age=0',
                'Sec-Fetch-Mode': 'navigate',
                'Sec-Fetch-Site': 'none',
                'Sec-Fetch-User': '?1'
            }
        },
        {
            'name': '🔄 F5刷新',
            'headers': {
                'Cache-Control': 'max-age=0',
                'Sec-Fetch-Mode': 'navigate',
                'Sec-Fetch-Site': 'same-origin',
                'Sec-Fetch-User': '?1'
            }
        },
        {
            'name': '⚡ Ctrl+F5硬刷新',
            'headers': {
                'Cache-Control': 'no-cache',
                'Pragma': 'no-cache',
                'Sec-Fetch-Mode': 'navigate',
                'Sec-Fetch-Site': 'same-origin',
                'Sec-Fetch-User': '?1'
            }
        },
        {
            'name': '🔗 地址栏重新输入',
            'headers': {
                'Cache-Control': 'max-age=0',
                'Sec-Fetch-Mode': 'navigate',
                'Sec-Fetch-Site': 'none',
                'Sec-Fetch-User': '?1'
            }
        },
        {
            'name': '📱 新标签页访问',
            'headers': {
                'Sec-Fetch-Mode': 'navigate',
                'Sec-Fetch-Site': 'none',
                'Sec-Fetch-User': '?1'
            }
        }
    ]
    
    results = []
    
    for i, scenario in enumerate(scenarios):
        print(f"📋 测试场景 {i+1}: {scenario['name']}")
        
        # 为每个场景设置特定头部
        test_headers = dict(session.headers)
        test_headers.update(scenario['headers'])
        
        try:
            # 发起请求
            response = session.get(url, headers=test_headers, timeout=15, allow_redirects=True)
            
            content = response.content.decode('utf-8', errors='ignore')
            content_hash = hashlib.md5(response.content).hexdigest()[:16]
            content_length = len(response.content)
            
            # 分析页面内容
            is_portal_page = 'AI基础设施矩阵' in content and 'linear-gradient' in content and 'portal-container' in content
            is_wrapper_page = 'jupyterhub_wrapper' in content.lower() or ('iframe' in content and 'JupyterHub' in content)
            is_react_page = 'react' in content.lower() or ('root' in content and 'js/bundle' in content)
            has_backend_features = 'userTasks' in content or 'hubStatus' in content or 'fetchHubStatus' in content
            
            # 确定页面类型
            page_type = "未知页面"
            if is_portal_page:
                page_type = "✅ 门户页面"
            elif is_wrapper_page:
                page_type = "⚠️ 包装页面"
            elif is_react_page or has_backend_features:
                page_type = "❌ React组件页面"
            
            print(f"   📊 状态码: {response.status_code}")
            print(f"   📏 内容长度: {content_length} bytes")
            print(f"   🔑 内容哈希: {content_hash}...")
            print(f"   📄 页面类型: {page_type}")
            
            # 检查关键特征
            print(f"   🏠 门户标题: {'✅' if 'AI基础设施矩阵' in content else '❌'}")
            print(f"   🎨 蓝色渐变: {'✅' if 'linear-gradient' in content else '❌'}")
            print(f"   🔘 操作按钮: {'✅' if '集成模式' in content or 'JupyterLab' in content else '❌'}")
            print(f"   ⚛️ React组件: {'❌' if has_backend_features else '✅'}")
            
            # 检查可能的问题
            if content_length < 3000:
                print(f"   ⚠️ 内容太短，可能是React页面")
            if 'fetchHubStatus' in content:
                print(f"   ⚠️ 检测到React组件代码")
            
            results.append({
                'scenario': scenario['name'],
                'status': response.status_code,
                'length': content_length,
                'hash': content_hash,
                'is_portal': is_portal_page,
                'is_react': is_react_page or has_backend_features,
                'page_type': page_type
            })
            
            print()
            
            # 在测试之间稍作等待，模拟真实用户行为
            time.sleep(0.5)
            
        except requests.RequestException as e:
            print(f"   ❌ 请求失败: {e}")
            print()
    
    # 分析结果
    print("=" * 60)
    print("🔍 真实浏览器行为分析")
    print("=" * 60)
    
    if not results:
        print("❌ 没有成功的测试结果")
        return False
    
    portal_count = sum(1 for r in results if r['is_portal'])
    react_count = sum(1 for r in results if r['is_react'])
    total_count = len(results)
    
    print(f"📊 测试总数: {total_count}")
    print(f"✅ 门户页面: {portal_count}")
    print(f"❌ React页面: {react_count}")
    print(f"❓ 其他页面: {total_count - portal_count - react_count}")
    print()
    
    # 详细结果
    for result in results:
        status_icon = "✅" if result['is_portal'] else "❌"
        print(f"{status_icon} {result['scenario']}: {result['page_type']} ({result['length']} bytes)")
    
    print()
    
    # 一致性检查
    lengths = [r['length'] for r in results]
    hashes = [r['hash'] for r in results]
    
    length_consistent = len(set(lengths)) == 1
    hash_consistent = len(set(hashes)) == 1
    all_portal = portal_count == total_count
    
    print(f"📏 内容长度一致: {'✅' if length_consistent else '❌'}")
    print(f"🔑 内容哈希一致: {'✅' if hash_consistent else '❌'}")
    print(f"🏠 全部门户页面: {'✅' if all_portal else '❌'}")
    print()
    
    if all_portal and length_consistent and hash_consistent:
        print("🎉 测试通过！真实浏览器行为一致")
        return True
    else:
        print("❌ 测试失败！检测到不一致行为")
        print("\n🚨 问题诊断:")
        
        if not all_portal:
            print("   - 某些场景返回了React组件而不是门户页面")
            print("   - 这通常意味着前端路由仍在拦截请求")
        
        if not length_consistent:
            print("   - 不同场景返回不同长度的内容")
            unique_lengths = set(lengths)
            print(f"   - 发现的长度: {list(unique_lengths)}")
        
        if not hash_consistent:
            print("   - 内容哈希不一致，说明返回了不同的页面")
        
        return False

if __name__ == "__main__":
    success = test_real_browser_behavior()
    exit(0 if success else 1)
