#!/usr/bin/env python3
"""
JupyterHub URL重定向完整修复验证
"""

import requests
from urllib.parse import urlparse

def comprehensive_redirect_test():
    """完整的重定向测试"""
    
    print("🔧 JupyterHub URL重定向完整修复验证")
    print("=" * 70)
    
    base_url = "http://localhost:8080"
    
    test_cases = [
        {
            "name": "精确匹配 /jupyter",
            "url": f"{base_url}/jupyter",
            "expected_redirect": f"{base_url}/jupyter/",
            "expected_status": [301, 302]
        },
        {
            "name": "精确匹配 /jupyter/", 
            "url": f"{base_url}/jupyter/",
            "expected_redirect": f"{base_url}/jupyter/hub/",
            "expected_status": [301, 302]
        },
        {
            "name": "最终页面 /jupyter/hub/",
            "url": f"{base_url}/jupyter/hub/",
            "expected_status": [200, 302],  # 302也是正常的，因为JupyterHub会重定向到登录页
            "expect_content": True
        }
    ]
    
    results = []
    
    for i, test in enumerate(test_cases, 1):
        print(f"\n📝 测试 {i}: {test['name']}")
        print(f"   URL: {test['url']}")
        
        try:
            response = requests.get(test['url'], allow_redirects=False, timeout=10)
            status_code = response.status_code
            
            print(f"   状态码: {status_code}")
            
            # 检查状态码
            if status_code in test['expected_status']:
                print(f"   ✅ 状态码正确")
                status_ok = True
            else:
                print(f"   ❌ 状态码错误，期望: {test['expected_status']}")
                status_ok = False
            
            # 检查重定向
            redirect_ok = True
            if 'expected_redirect' in test:
                location = response.headers.get('Location', '')
                print(f"   重定向到: {location}")
                
                if location == test['expected_redirect']:
                    print(f"   ✅ 重定向正确")
                    redirect_ok = True
                else:
                    print(f"   ❌ 重定向错误，期望: {test['expected_redirect']}")
                    redirect_ok = False
            
            # 检查内容（对于最终页面）
            content_ok = True
            if test.get('expect_content', False):
                # 对于最终页面，使用允许重定向的请求
                final_response = requests.get(test['url'], allow_redirects=True, timeout=15)
                content = final_response.text.lower()
                
                print(f"   最终URL: {final_response.url}")
                print(f"   内容长度: {len(content)} 字符")
                
                content_indicators = [
                    'jupyter' in content,
                    'login' in content or 'hub' in content,
                    len(content) > 500
                ]
                
                content_score = sum(content_indicators)
                print(f"   内容检查: {content_score}/3 通过")
                
                if content_score >= 2:
                    print(f"   ✅ 内容正常")
                    content_ok = True
                else:
                    print(f"   ⚠️ 内容可能异常")
                    content_ok = False
            
            # 记录结果
            test_success = status_ok and redirect_ok and content_ok
            results.append({
                'name': test['name'],
                'success': test_success,
                'status_code': status_code
            })
            
            if test_success:
                print(f"   🎯 测试通过")
            else:
                print(f"   ❌ 测试失败")
                
        except Exception as e:
            print(f"   ❌ 测试异常: {e}")
            results.append({
                'name': test['name'],
                'success': False,
                'error': str(e)
            })
    
    # 测试完整链路
    print(f"\n📝 测试 4: 完整重定向链路")
    try:
        # 从 /jupyter 开始，跟随所有重定向
        response = requests.get(f"{base_url}/jupyter", allow_redirects=True, timeout=20)
        
        print(f"   起始URL: {base_url}/jupyter")
        print(f"   最终URL: {response.url}")
        print(f"   最终状态: {response.status_code}")
        
        # 检查最终URL是否正确
        parsed_url = urlparse(response.url)
        url_correct = (
            parsed_url.netloc == "localhost:8080" and
            parsed_url.path.startswith("/jupyter/") and
            response.status_code == 200
        )
        
        if url_correct:
            print(f"   ✅ 完整链路正常")
            chain_success = True
        else:
            print(f"   ❌ 完整链路异常")
            chain_success = False
            
    except Exception as e:
        print(f"   ❌ 完整链路测试失败: {e}")
        chain_success = False
    
    # 生成最终报告
    print("\n" + "=" * 70)
    print("📊 修复验证结果汇总")
    print("=" * 70)
    
    success_count = sum(1 for r in results if r['success'])
    total_tests = len(results) + 1  # +1 for chain test
    
    if chain_success:
        success_count += 1
    
    for result in results:
        status = "✅" if result['success'] else "❌"
        print(f"{result['name']:<25}: {status}")
    
    chain_status = "✅" if chain_success else "❌"
    print(f"{'完整重定向链路':<25}: {chain_status}")
    
    overall_success = success_count == total_tests
    
    print(f"\n🎯 总体结果: {'✅ 修复成功' if overall_success else '❌ 仍有问题'} ({success_count}/{total_tests})")
    
    if overall_success:
        print("\n🎉 JupyterHub URL重定向问题已完全修复!")
        print("\n✅ 确认修复:")
        print("   • /jupyter 精确匹配重定向正常")
        print("   • /jupyter/ 精确匹配重定向正常") 
        print("   • 最终页面访问正常")
        print("   • 完整重定向链路正常")
        print("   • 不再跳转到错误的 http://localhost/jupyter")
        print("\n🔄 用户使用流程:")
        print("   1. 访问 http://localhost:8080/jupyter")
        print("   2. 自动重定向到 http://localhost:8080/jupyter/")
        print("   3. 再重定向到 http://localhost:8080/jupyter/hub/")
        print("   4. 显示JupyterHub登录页面")
        print("   5. 使用admin/admin123登录")
        
    else:
        print("\n⚠️ 仍有问题需要进一步调试:")
        
        failed_tests = [r for r in results if not r['success']]
        for failed in failed_tests:
            print(f"   - {failed['name']} 失败")
        
        if not chain_success:
            print(f"   - 完整重定向链路失败")
    
    return overall_success

if __name__ == "__main__":
    comprehensive_redirect_test()
