#!/usr/bin/env python3
"""
测试 /jupyterhub 页面访问和刷新的一致性
验证首次访问和刷新后的体验是否相同
"""

import requests
import time
import hashlib
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

def test_jupyterhub_consistency():
    """测试JupyterHub页面的一致性"""
    
    base_url = "http://localhost:8080"
    session = requests.Session()
    
    # 配置重试策略
    retry_strategy = Retry(
        total=3,
        status_forcelist=[429, 500, 502, 503, 504],
        allowed_methods=["HEAD", "GET", "OPTIONS"]
    )
    adapter = HTTPAdapter(max_retries=retry_strategy)
    session.mount("http://", adapter)
    session.mount("https://", adapter)
    
    session.headers.update({
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36'
    })
    
    print("🔍 测试 /jupyterhub 页面一致性...")
    print("=" * 60)
    
    test_results = []
    
    # 多次访问测试
    for i in range(5):
        print(f"\n📍 第 {i+1} 次访问测试")
        
        try:
            # 访问 /jupyterhub 页面
            response = session.get(f"{base_url}/jupyterhub", timeout=10)
            
            # 记录关键信息
            result = {
                'attempt': i + 1,
                'status_code': response.status_code,
                'content_length': len(response.content),
                'content_hash': hashlib.md5(response.content).hexdigest(),
                'headers': dict(response.headers),
                'response_time': response.elapsed.total_seconds()
            }
            
            test_results.append(result)
            
            print(f"   📊 状态码: {result['status_code']}")
            print(f"   📏 内容长度: {result['content_length']} bytes")
            print(f"   🔑 内容哈希: {result['content_hash'][:16]}...")
            print(f"   ⏱️  响应时间: {result['response_time']:.3f}s")
            
            # 检查关键HTML元素
            html_content = response.text
            has_portal_title = 'JupyterHub 门户' in html_content
            has_logo = '🚀' in html_content
            has_blue_gradient = 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)' in html_content
            has_buttons = 'btn btn-primary' in html_content
            
            print(f"   🏠 门户标题: {'✅' if has_portal_title else '❌'}")
            print(f"   � 门户图标: {'✅' if has_logo else '❌'}")
            print(f"   🎨 蓝色渐变: {'✅' if has_blue_gradient else '❌'}")
            print(f"   🔘 操作按钮: {'✅' if has_buttons else '❌'}")
            
            # 检查缓存头部
            cache_control = response.headers.get('Cache-Control', '')
            etag = response.headers.get('ETag', '')
            
            print(f"   💾 缓存控制: {cache_control}")
            print(f"   🏷️  ETag: {etag[:20]}..." if etag else "   🏷️  ETag: 无")
            
        except requests.exceptions.Timeout:
            print(f"   ⏰ 请求超时")
            test_results.append({'attempt': i + 1, 'error': 'timeout'})
        except Exception as e:
            print(f"   💥 请求失败: {str(e)}")
            test_results.append({'attempt': i + 1, 'error': str(e)})
        
        # 等待一下再次测试
        if i < 4:
            time.sleep(2)
    
    # 分析一致性
    print("\n" + "=" * 60)
    print("📊 一致性分析报告")
    print("=" * 60)
    
    successful_results = [r for r in test_results if 'error' not in r]
    
    if len(successful_results) < 2:
        print("❌ 测试结果不足，无法分析一致性")
        return
    
    # 状态码一致性
    status_codes = [r['status_code'] for r in successful_results]
    status_consistent = all(code == status_codes[0] for code in status_codes)
    print(f"📊 状态码一致性: {'✅ 一致' if status_consistent else '❌ 不一致'}")
    if not status_consistent:
        print(f"   状态码变化: {set(status_codes)}")
    
    # 内容长度一致性
    content_lengths = [r['content_length'] for r in successful_results]
    length_consistent = all(length == content_lengths[0] for length in content_lengths)
    print(f"📏 内容长度一致性: {'✅ 一致' if length_consistent else '❌ 不一致'}")
    if not length_consistent:
        print(f"   长度变化: {set(content_lengths)}")
    
    # 内容哈希一致性
    content_hashes = [r['content_hash'] for r in successful_results]
    hash_consistent = all(hash_val == content_hashes[0] for hash_val in content_hashes)
    print(f"🔑 内容哈希一致性: {'✅ 一致' if hash_consistent else '❌ 不一致'}")
    if not hash_consistent:
        print(f"   哈希变化: {len(set(content_hashes))} 种不同的内容")
    
    # 响应时间分析
    response_times = [r['response_time'] for r in successful_results]
    avg_time = sum(response_times) / len(response_times)
    max_time = max(response_times)
    min_time = min(response_times)
    
    print(f"⏱️  响应时间分析:")
    print(f"   平均: {avg_time:.3f}s")
    print(f"   最快: {min_time:.3f}s")
    print(f"   最慢: {max_time:.3f}s")
    print(f"   稳定性: {'✅ 稳定' if (max_time - min_time) < 1.0 else '⚠️ 波动较大'}")
    
    # 缓存头部分析
    cache_headers = [r['headers'].get('Cache-Control', '') for r in successful_results]
    cache_consistent = all(header == cache_headers[0] for header in cache_headers)
    print(f"💾 缓存头部一致性: {'✅ 一致' if cache_consistent else '❌ 不一致'}")
    
    # 总体评估
    print("\n🏁 总体评估:")
    overall_consistent = status_consistent and length_consistent and hash_consistent
    
    if overall_consistent:
        print("✅ 页面访问体验完全一致")
        print("   - 状态码稳定")
        print("   - 内容完全相同")
        print("   - 响应时间在合理范围内")
    else:
        print("❌ 页面访问体验存在不一致")
        inconsistencies = []
        if not status_consistent:
            inconsistencies.append("状态码不一致")
        if not length_consistent:
            inconsistencies.append("内容长度不一致")
        if not hash_consistent:
            inconsistencies.append("内容不一致")
        
        for issue in inconsistencies:
            print(f"   - {issue}")
    
    # 额外测试：模拟用户刷新
    print("\n🔄 模拟用户刷新测试...")
    try:
        # 第一次访问
        first_response = session.get(f"{base_url}/jupyterhub")
        time.sleep(1)
        
        # 模拟刷新（添加cache-control头部）
        refresh_headers = dict(session.headers)
        refresh_headers.update({
            'Cache-Control': 'no-cache',
            'Pragma': 'no-cache'
        })
        
        refresh_response = requests.get(f"{base_url}/jupyterhub", headers=refresh_headers)
        
        refresh_consistent = (
            first_response.status_code == refresh_response.status_code and
            len(first_response.content) == len(refresh_response.content) and
            hashlib.md5(first_response.content).hexdigest() == hashlib.md5(refresh_response.content).hexdigest()
        )
        
        print(f"🔄 刷新一致性: {'✅ 一致' if refresh_consistent else '❌ 不一致'}")
        if not refresh_consistent:
            print(f"   首次访问状态: {first_response.status_code}")
            print(f"   刷新后状态: {refresh_response.status_code}")
            print(f"   内容长度变化: {len(first_response.content)} -> {len(refresh_response.content)}")
            
    except Exception as e:
        print(f"💥 刷新测试失败: {str(e)}")

if __name__ == "__main__":
    test_jupyterhub_consistency()
