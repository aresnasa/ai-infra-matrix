#!/usr/bin/env python3
"""
最终验证测试：浏览器一致性解决方案验证
"""

import requests
import time
import hashlib

def final_verification_test():
    """最终验证测试"""
    
    print("🏁 最终验证测试")
    print("=" * 60)
    print("🎯 测试目标: 验证 /jupyterhub 路径在所有访问方式下的一致性")
    print()
    
    url = "http://localhost:8080/jupyterhub"
    
    # 测试不同的用户代理和访问模式
    test_cases = [
        {
            'name': 'cURL命令行',
            'headers': {
                'User-Agent': 'curl/8.15.0'
            }
        },
        {
            'name': 'Chrome浏览器',
            'headers': {
                'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36',
                'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8',
                'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
                'Accept-Encoding': 'gzip, deflate',
                'Cache-Control': 'no-cache',
                'Pragma': 'no-cache'
            }
        },
        {
            'name': 'Safari浏览器',
            'headers': {
                'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15',
                'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
                'Accept-Language': 'zh-cn'
            }
        },
        {
            'name': '刷新请求',
            'headers': {
                'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36',
                'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8',
                'Cache-Control': 'max-age=0',
                'Pragma': 'no-cache'
            }
        }
    ]
    
    results = []
    expected_hash = None
    expected_length = None
    
    for i, test_case in enumerate(test_cases):
        print(f"📱 测试 {i+1}: {test_case['name']}")
        
        try:
            response = requests.get(url, headers=test_case['headers'], timeout=10)
            content_hash = hashlib.md5(response.content).hexdigest()[:16]
            content_length = len(response.content)
            
            print(f"   📊 状态码: {response.status_code}")
            print(f"   📏 内容长度: {content_length} bytes")
            print(f"   🔑 内容哈希: {content_hash}...")
            
            # 设置期望值
            if expected_hash is None:
                expected_hash = content_hash
                expected_length = content_length
                print(f"   ✅ 设为基准值")
            else:
                # 验证一致性
                is_consistent = (content_hash == expected_hash and content_length == expected_length)
                icon = "✅" if is_consistent else "❌"
                print(f"   {icon} 与基准值一致性: {'是' if is_consistent else '否'}")
            
            # 验证内容特征
            content_str = response.content.decode('utf-8', errors='ignore')
            has_portal_title = 'AI基础设施矩阵' in content_str
            has_blue_gradient = 'linear-gradient' in content_str and ('#667eea' in content_str or '#764ba2' in content_str)
            has_action_buttons = '集成模式' in content_str or 'JupyterLab' in content_str
            
            print(f"   🏠 门户标题: {'✅' if has_portal_title else '❌'}")
            print(f"   🎨 蓝色渐变: {'✅' if has_blue_gradient else '❌'}")
            print(f"   🔘 操作按钮: {'✅' if has_action_buttons else '❌'}")
            
            results.append({
                'name': test_case['name'],
                'status': response.status_code,
                'length': content_length,
                'hash': content_hash,
                'has_portal_features': has_portal_title and has_blue_gradient and has_action_buttons
            })
            
            print()
            
        except requests.RequestException as e:
            print(f"   ❌ 请求失败: {e}")
            print()
    
    # 最终评估
    print("=" * 60)
    print("📊 最终验证结果")
    print("=" * 60)
    
    all_consistent = True
    all_portal_features = True
    
    for result in results:
        length_ok = result['length'] == expected_length
        hash_ok = result['hash'] == expected_hash
        features_ok = result['has_portal_features']
        
        all_consistent = all_consistent and length_ok and hash_ok
        all_portal_features = all_portal_features and features_ok
        
        status_icon = "✅" if length_ok and hash_ok else "❌"
        features_icon = "✅" if features_ok else "❌"
        
        print(f"{status_icon} {result['name']}: {result['length']} bytes, 门户特征: {features_icon}")
    
    print()
    
    if all_consistent and all_portal_features:
        print("🎉 验证成功！")
        print("✅ 所有访问方式返回相同内容")
        print("✅ 门户页面特征完整")
        print("✅ 浏览器一致性问题已解决")
        print("✅ React前端路由冲突已修复")
        print()
        print("🔧 解决方案总结:")
        print("   - 使用nginx try_files和@frontend处理前端路由")
        print("   - 确保 /jupyterhub 路径优先匹配静态内容")
        print("   - 避免前端应用拦截特定路径")
        return True
    else:
        print("❌ 验证失败！")
        if not all_consistent:
            print("   内容不一致")
        if not all_portal_features:
            print("   门户特征缺失")
        return False

if __name__ == "__main__":
    success = final_verification_test()
    exit(0 if success else 1)
