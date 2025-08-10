#!/usr/bin/env python3
"""
浏览器一致性测试脚本
测试浏览器首次访问和刷新页面的一致性
"""

import requests
import time
import hashlib

def test_browser_simulation():
    """模拟浏览器行为测试"""
    
    print("🌐 模拟浏览器测试...")
    print("=" * 60)
    
    url = "http://localhost:8080/jupyterhub"
    
    # 模拟浏览器头部
    browser_headers = {
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7',
        'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
        'Accept-Encoding': 'gzip, deflate',
        'Cache-Control': 'no-cache',
        'Pragma': 'no-cache',
        'Upgrade-Insecure-Requests': '1'
    }
    
    results = []
    
    # 测试多次访问
    for i in range(3):
        print(f"📱 第 {i+1} 次浏览器访问测试")
        
        try:
            # 首次访问
            response1 = requests.get(url, headers=browser_headers, timeout=10)
            hash1 = hashlib.md5(response1.content).hexdigest()[:16]
            
            print(f"   首次访问:")
            print(f"   📊 状态码: {response1.status_code}")
            print(f"   📏 内容长度: {len(response1.content)} bytes")
            print(f"   🔑 内容哈希: {hash1}...")
            
            # 短暂等待
            time.sleep(1)
            
            # 刷新访问 (模拟F5刷新)
            refresh_headers = browser_headers.copy()
            refresh_headers['Cache-Control'] = 'max-age=0'
            
            response2 = requests.get(url, headers=refresh_headers, timeout=10)
            hash2 = hashlib.md5(response2.content).hexdigest()[:16]
            
            print(f"   刷新访问:")
            print(f"   📊 状态码: {response2.status_code}")
            print(f"   📏 内容长度: {len(response2.content)} bytes")
            print(f"   🔑 内容哈希: {hash2}...")
            
            # 检查一致性
            is_consistent = (
                response1.status_code == response2.status_code and
                len(response1.content) == len(response2.content) and
                hash1 == hash2
            )
            
            consistency_icon = "✅" if is_consistent else "❌"
            print(f"   🔄 首次vs刷新一致性: {consistency_icon}")
            
            results.append({
                'round': i + 1,
                'first_status': response1.status_code,
                'first_length': len(response1.content),
                'first_hash': hash1,
                'refresh_status': response2.status_code,
                'refresh_length': len(response2.content),
                'refresh_hash': hash2,
                'consistent': is_consistent
            })
            
            print()
            
        except requests.RequestException as e:
            print(f"   ❌ 请求失败: {e}")
            print()
    
    # 总结报告
    print("=" * 60)
    print("📊 浏览器一致性分析报告")
    print("=" * 60)
    
    consistent_count = sum(1 for r in results if r['consistent'])
    total_count = len(results)
    
    print(f"🔄 测试轮次: {total_count}")
    print(f"✅ 一致轮次: {consistent_count}")
    print(f"❌ 不一致轮次: {total_count - consistent_count}")
    
    if consistent_count == total_count:
        print("🎉 浏览器访问完全一致！")
        print("   - 首次访问和刷新返回相同内容")
        print("   - 不存在路由冲突")
        print("   - 用户体验良好")
    else:
        print("⚠️  检测到浏览器不一致:")
        for r in results:
            if not r['consistent']:
                print(f"   轮次 {r['round']}:")
                print(f"     首次: {r['first_length']} bytes, 哈希: {r['first_hash']}")
                print(f"     刷新: {r['refresh_length']} bytes, 哈希: {r['refresh_hash']}")
    
    print("=" * 60)

if __name__ == "__main__":
    test_browser_simulation()
