#!/usr/bin/env python3
"""
测试 /jupyterhub 路径的一致性
修复后验证测试
"""

import requests
import time

def test_jupyterhub_consistency():
    url = "http://localhost:8080/jupyterhub"
    
    print("🔧 测试 /jupyterhub 路径修复后的一致性...")
    print("=" * 50)
    
    results = []
    
    # 进行5次测试
    for i in range(1, 6):
        try:
            response = requests.get(url, timeout=10)
            
            print(f"测试 {i}:")
            print(f"  状态码: {response.status_code}")
            print(f"  Content-Type: {response.headers.get('Content-Type', 'N/A')}")
            print(f"  Content-Length: {len(response.text)}")
            print(f"  包含iframe: {'<iframe' in response.text}")
            print(f"  包含JupyterHub标题: {'JupyterHub' in response.text}")
            
            results.append({
                'test': i,
                'status': response.status_code,
                'content_type': response.headers.get('Content-Type', ''),
                'size': len(response.text),
                'has_iframe': '<iframe' in response.text,
                'has_title': 'JupyterHub' in response.text
            })
            
            print()
            time.sleep(2)
            
        except Exception as e:
            print(f"测试 {i} 失败: {e}")
            print()
    
    # 分析结果
    print("📊 结果分析:")
    print("=" * 30)
    
    if results:
        # 检查一致性
        first_size = results[0]['size']
        first_content_type = results[0]['content_type']
        
        sizes = [r['size'] for r in results]
        content_types = [r['content_type'] for r in results]
        
        print(f"内容大小一致性: {len(set(sizes)) == 1}")
        print(f"内容类型一致性: {len(set(content_types)) == 1}")
        print(f"平均响应大小: {sum(sizes) / len(sizes):.0f} 字符")
        print(f"内容类型: {first_content_type}")
        
        # 检查是否都包含iframe
        iframe_count = sum(1 for r in results if r['has_iframe'])
        title_count = sum(1 for r in results if r['has_title'])
        
        print(f"包含iframe的测试: {iframe_count}/{len(results)}")
        print(f"包含JupyterHub标题的测试: {title_count}/{len(results)}")
        
        if len(set(sizes)) == 1 and len(set(content_types)) == 1:
            print("\n✅ /jupyterhub 路径现在表现一致！")
        else:
            print("\n❌ 仍然存在不一致的问题")
    
    return results

if __name__ == "__main__":
    test_jupyterhub_consistency()
