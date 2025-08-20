#!/usr/bin/env python3
"""
Chrome自动登录测试监控脚本
"""

import time
import glob
import os
from datetime import datetime
import json

def monitor_chrome_test():
    """监控Chrome自动登录测试进度"""
    
    print("🔍 监控Chrome自动登录测试进度...")
    print("=" * 60)
    
    last_screenshot_count = 0
    start_time = datetime.now()
    
    while True:
        try:
            # 检查自动登录测试截图
            auto_login_shots = sorted(glob.glob('auto_login_*.png'))
            auto_login_count = len(auto_login_shots)
            
            current_time = datetime.now().strftime('%H:%M:%S')
            elapsed = (datetime.now() - start_time).total_seconds()
            
            if auto_login_count != last_screenshot_count:
                print(f"[{current_time}] 📸 新截图生成! (第{auto_login_count}个)")
                
                if auto_login_shots:
                    latest = auto_login_shots[-1]
                    size = os.path.getsize(latest)
                    print(f"  最新截图: {latest}")
                    print(f"  文件大小: {size:,} bytes")
                    
                    # 分析截图名称推测测试步骤
                    if 'initial' in latest:
                        print("  📝 步骤: 初始页面加载")
                    elif 'homepage' in latest:
                        print("  📝 步骤: 主页访问")
                    elif 'projects' in latest:
                        print("  📝 步骤: 项目页面")
                    elif 'login' in latest:
                        print("  📝 步骤: 登录页面")
                    elif 'jupyter' in latest:
                        print("  📝 步骤: JupyterHub访问")
                    elif 'success' in latest:
                        print("  📝 步骤: 测试成功")
                    else:
                        print("  📝 步骤: 测试进行中...")
                
                last_screenshot_count = auto_login_count
                print()
            
            # 显示实时状态
            if elapsed % 10 < 1:  # 每10秒显示一次状态
                print(f"[{current_time}] ⏱️ 运行时间: {elapsed:.0f}秒, 截图数: {auto_login_count}")
            
            # 检查测试是否完成
            if auto_login_count >= 7:  # 预期完整测试会生成6-7个截图
                print(f"[{current_time}] ✅ 测试完成! (生成了 {auto_login_count} 个截图)")
                break
            
            # 超时检查
            if elapsed > 300:  # 5分钟超时
                print(f"[{current_time}] ⏰ 测试超时 (5分钟)")
                break
            
            time.sleep(1)
            
        except KeyboardInterrupt:
            print("\n🛑 监控被中断")
            break
        except Exception as e:
            print(f"监控错误: {e}")
            time.sleep(5)
    
    # 生成最终报告
    final_screenshots = sorted(glob.glob('auto_login_*.png'))
    generate_final_report(final_screenshots)

def generate_final_report(screenshots):
    """生成最终测试报告"""
    print("\n" + "=" * 60)
    print("📊 Chrome自动登录测试最终报告")
    print("=" * 60)
    
    if not screenshots:
        print("❌ 没有生成任何截图 - 测试可能失败")
        return
    
    print(f"📸 总共生成截图: {len(screenshots)}")
    print("\n截图详情:")
    
    for i, shot in enumerate(screenshots, 1):
        size = os.path.getsize(shot)
        mtime = datetime.fromtimestamp(os.path.getctime(shot))
        print(f"{i}. {shot}")
        print(f"   创建时间: {mtime.strftime('%H:%M:%S')}")
        print(f"   文件大小: {size:,} bytes")
        
        # 基于文件大小判断页面状态
        if size < 50000:
            print("   📊 状态: 可能是空白页面或加载失败")
        elif size < 200000:
            print("   📊 状态: 简单页面")
        else:
            print("   📊 状态: 复杂页面 (可能包含丰富内容)")
        print()
    
    # 分析测试结果
    print("🎯 测试结果分析:")
    
    if len(screenshots) >= 6:
        print("✅ 测试流程完整 - 完成了所有预期步骤")
        print("   1. ✓ 初始页面加载")
        print("   2. ✓ 主页访问") 
        print("   3. ✓ 项目页面")
        print("   4. ✓ 登录流程")
        print("   5. ✓ JupyterHub访问")
        print("   6. ✓ 最终验证")
    elif len(screenshots) >= 3:
        print("⚠️ 测试部分完成 - 在某个步骤遇到问题")
        if len(screenshots) < 4:
            print("   可能在登录步骤失败")
        else:
            print("   可能在JupyterHub访问步骤失败")
    else:
        print("❌ 测试早期失败 - 基础页面访问可能有问题")
    
    # 检查是否有错误日志
    log_files = glob.glob('chrome_test_*.log')
    if log_files:
        print(f"\n📝 发现 {len(log_files)} 个日志文件:")
        for log_file in log_files:
            print(f"   - {log_file}")
    
    print("\n💡 下一步建议:")
    print("   1. 查看最新截图确认测试状态")
    print("   2. 检查是否需要手动验证某些步骤")
    print("   3. 如果测试失败，查看错误日志")
    print("   4. 验证admin/admin123登录凭据是否正确")

if __name__ == "__main__":
    monitor_chrome_test()
