#!/usr/bin/env python3
"""
浏览器缓存行为测试 - 专门检测缓存导致的不一致问题
"""

import requests
import time
import hashlib
import json

def test_browser_cache_behavior():
    """测试浏览器缓存行为对页面一致性的影响"""
    
    print("🗄️ 浏览器缓存行为测试")
    print("=" * 60)
    print("🎯 目标: 检测缓存对 /jupyterhub 页面一致性的影响")
    print()
    
    url = "http://localhost:8080/jupyterhub"
    
    # 创建两个不同的会话来模拟不同的浏览器状态
    fresh_session = requests.Session()
    cached_session = requests.Session()
    
    # 基本浏览器头部
    base_headers = {
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36',
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8',
        'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
        'Accept-Encoding': 'gzip, deflate',
        'Connection': 'keep-alive',
        'Upgrade-Insecure-Requests': '1'
    }
    
    fresh_session.headers.update(base_headers)
    cached_session.headers.update(base_headers)
    
    # 测试场景
    test_scenarios = [
        {
            'name': '🆕 全新会话 - 无缓存',
            'session': fresh_session,
            'headers': {
                'Cache-Control': 'no-cache',
                'Pragma': 'no-cache'
            },
            'description': '模拟首次访问浏览器'
        },
        {
            'name': '📦 有缓存会话 - 普通访问',
            'session': cached_session,
            'headers': {},
            'description': '模拟有缓存的浏览器访问'
        },
        {
            'name': '🔄 有缓存会话 - F5刷新',
            'session': cached_session,
            'headers': {
                'Cache-Control': 'max-age=0'
            },
            'description': '模拟F5刷新'
        },
        {
            'name': '⚡ 有缓存会话 - 强制刷新',
            'session': cached_session,
            'headers': {
                'Cache-Control': 'no-cache, no-store, must-revalidate',
                'Pragma': 'no-cache',
                'Expires': '0'
            },
            'description': '模拟Ctrl+F5强制刷新'
        },
        {
            'name': '🌐 条件请求 - If-None-Match',
            'session': cached_session,
            'headers': {
                'If-None-Match': '"68921c50-15d6"'  # 假设的ETag
            },
            'description': '模拟浏览器条件请求'
        },
        {
            'name': '🔗 新窗口访问',
            'session': requests.Session(),  # 新会话
            'headers': {
                'Cache-Control': 'max-age=0'
            },
            'description': '模拟新窗口或标签页'
        }
    ]
    
    results = []
    baseline_hash = None
    baseline_length = None
    
    for i, scenario in enumerate(test_scenarios):
        print(f"📋 测试 {i+1}: {scenario['name']}")
        print(f"   📝 说明: {scenario['description']}")
        
        session = scenario['session']
        test_headers = dict(session.headers)
        test_headers.update(scenario['headers'])
        
        try:
            response = session.get(url, headers=test_headers, timeout=15)
            
            content = response.content.decode('utf-8', errors='ignore')
            content_hash = hashlib.md5(response.content).hexdigest()[:16]
            content_length = len(response.content)
            
            # 设置基准
            if baseline_hash is None:
                baseline_hash = content_hash
                baseline_length = content_length
            
            # 分析页面特征
            is_portal = 'AI基础设施矩阵' in content and 'linear-gradient' in content
            is_react = 'fetchHubStatus' in content or 'userTasks' in content
            has_wrapper = 'jupyterhub_wrapper' in content.lower()
            
            # 获取缓存相关头部
            cache_control = response.headers.get('Cache-Control', 'N/A')
            etag = response.headers.get('ETag', 'N/A')
            last_modified = response.headers.get('Last-Modified', 'N/A')
            
            print(f"   📊 状态码: {response.status_code}")
            print(f"   📏 内容长度: {content_length} bytes")
            print(f"   🔑 内容哈希: {content_hash}...")
            print(f"   🏠 门户页面: {'✅' if is_portal else '❌'}")
            print(f"   ⚛️ React组件: {'❌' if is_react else '✅'}")
            print(f"   📦 缓存控制: {cache_control}")
            print(f"   🏷️ ETag: {etag[:20]}..." if len(etag) > 20 else f"   🏷️ ETag: {etag}")
            
            # 与基准比较
            is_consistent = content_hash == baseline_hash and content_length == baseline_length
            consistency_icon = "✅" if is_consistent else "❌"
            print(f"   🔄 与基准一致: {consistency_icon}")
            
            results.append({
                'scenario': scenario['name'],
                'status': response.status_code,
                'length': content_length,
                'hash': content_hash,
                'is_portal': is_portal,
                'is_react': is_react,
                'is_consistent': is_consistent,
                'cache_control': cache_control,
                'etag': etag
            })
            
            print()
            
        except requests.RequestException as e:
            print(f"   ❌ 请求失败: {e}")
            print()
    
    # 分析结果
    print("=" * 60)
    print("📊 缓存行为分析结果")
    print("=" * 60)
    
    if not results:
        print("❌ 没有成功的测试结果")
        return False
    
    # 统计
    total_tests = len(results)
    consistent_tests = sum(1 for r in results if r['is_consistent'])
    portal_tests = sum(1 for r in results if r['is_portal'])
    react_tests = sum(1 for r in results if r['is_react'])
    
    print(f"📈 测试统计:")
    print(f"   总测试数: {total_tests}")
    print(f"   一致测试: {consistent_tests}")
    print(f"   门户页面: {portal_tests}")
    print(f"   React页面: {react_tests}")
    print()
    
    # 详细结果
    print("📋 详细结果:")
    for result in results:
        status_icon = "✅" if result['is_consistent'] and result['is_portal'] else "❌"
        page_type = "门户" if result['is_portal'] else ("React" if result['is_react'] else "其他")
        print(f"{status_icon} {result['scenario']}: {page_type}页面 ({result['length']} bytes)")
    
    print()
    
    # 问题诊断
    all_consistent = consistent_tests == total_tests
    all_portal = portal_tests == total_tests
    no_react = react_tests == 0
    
    if all_consistent and all_portal and no_react:
        print("🎉 缓存测试通过！")
        print("✅ 所有缓存场景都返回一致的门户页面")
        print("✅ 没有React组件干扰")
        print("✅ 缓存机制不影响页面一致性")
        return True
    else:
        print("❌ 缓存测试失败！")
        
        if not all_consistent:
            print("⚠️ 检测到内容不一致:")
            inconsistent_scenarios = [r['scenario'] for r in results if not r['is_consistent']]
            for scenario in inconsistent_scenarios:
                print(f"   - {scenario}")
        
        if not all_portal:
            print("⚠️ 某些场景未返回门户页面:")
            non_portal_scenarios = [r['scenario'] for r in results if not r['is_portal']]
            for scenario in non_portal_scenarios:
                print(f"   - {scenario}")
        
        if react_tests > 0:
            print("⚠️ 检测到React组件干扰:")
            react_scenarios = [r['scenario'] for r in results if r['is_react']]
            for scenario in react_scenarios:
                print(f"   - {scenario}")
        
        print("\n💡 建议:")
        print("   1. 清除浏览器缓存")
        print("   2. 检查nginx配置的location块优先级")
        print("   3. 确认前端应用不会拦截/jupyterhub路径")
        
        return False

if __name__ == "__main__":
    success = test_browser_cache_behavior()
    exit(0 if success else 1)
