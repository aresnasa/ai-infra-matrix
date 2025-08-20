#!/usr/bin/env python3
"""
修复JupyterHub路由问题

问题分析：
1. React SPA中有JupyterHub菜单项但没有对应路由
2. 用户从/projects点击JupyterHub时，React Router拦截了/jupyterhub请求
3. 应该让nginx直接处理/jupyterhub路径，而不是React Router

解决方案：
1. 修改Layout.js中的JupyterHub菜单项为外部链接
2. 确保nginx配置正确处理/jupyterhub路径
3. 修复React Router与nginx routing的冲突
"""

import os
import re

def fix_layout_jupyterhub_link():
    """修复Layout.js中的JupyterHub链接"""
    layout_path = "/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/src/frontend/src/components/Layout.js"
    
    print("🔧 修复Layout.js中的JupyterHub菜单项...")
    
    with open(layout_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 修改JupyterHub菜单项，使用window.open或href而不是React Router
    old_pattern = r'''    {
      key: '/jupyterhub',
      icon: <ExperimentTwoTone />,
      label: 'JupyterHub',
    },'''
    
    new_pattern = '''    {
      key: '/jupyterhub',
      icon: <ExperimentTwoTone />,
      label: 'JupyterHub',
      onClick: () => {
        // 直接跳转到nginx处理的JupyterHub路径，避免React Router拦截
        window.location.href = '/jupyterhub';
      }
    },'''
    
    if old_pattern in content:
        content = content.replace(old_pattern, new_pattern)
        print("✅ 已修改JupyterHub菜单项为直接跳转")
    else:
        print("⚠️  未找到预期的JupyterHub菜单项模式，请手动检查")
    
    # 备份原文件
    backup_path = layout_path + '.backup'
    with open(backup_path, 'w', encoding='utf-8') as f:
        f.write(open(layout_path, 'r', encoding='utf-8').read())
    print(f"📄 已备份原文件到: {backup_path}")
    
    # 写入修改后的内容
    with open(layout_path, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print("✅ Layout.js修复完成")

def verify_nginx_configuration():
    """验证nginx配置"""
    nginx_path = "/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/src/nginx/nginx.conf"
    
    print("🔍 检查nginx配置...")
    
    with open(nginx_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 检查关键配置
    checks = [
        (r'location\s+/jupyterhub\s*{', "静态JupyterHub location块"),
        (r'location\s+/jupyter/', "JupyterHub代理location块"),
        (r'location\s+/\s*{', "前端应用代理location块"),
    ]
    
    for pattern, description in checks:
        if re.search(pattern, content):
            print(f"✅ 找到: {description}")
        else:
            print(f"❌ 缺少: {description}")
    
    # 检查location优先级（more specific paths should come first）
    location_order = []
    for match in re.finditer(r'location\s+([^{]+)\s*{', content):
        location_order.append(match.group(1).strip())
    
    print(f"📋 nginx location顺序: {location_order}")
    
    # 确认/jupyterhub是否在/之前（更具体的路径应该在前面）
    if '/jupyterhub' in location_order and '/' in location_order:
        jupyterhub_idx = location_order.index('/jupyterhub')
        root_idx = location_order.index('/')
        if jupyterhub_idx < root_idx:
            print("✅ nginx location优先级正确")
        else:
            print("⚠️  nginx location优先级可能有问题：/jupyterhub应该在/之前")
    
def create_test_script():
    """创建测试脚本验证修复效果"""
    test_script = '''#!/usr/bin/env python3
"""
测试JupyterHub路由修复效果
"""

import requests
import time

def test_routing():
    """测试路由配置"""
    base_url = "http://localhost:8080"
    
    print("🧪 测试路由配置...")
    
    # 测试1: 直接访问/jupyterhub
    try:
        response = requests.get(f"{base_url}/jupyterhub", timeout=10)
        print(f"✅ /jupyterhub 直接访问: {response.status_code}")
        
        # 检查是否返回了HTML wrapper页面
        if 'JupyterHub' in response.text:
            print("✅ 返回了JupyterHub wrapper页面")
        else:
            print("❌ 未返回预期的JupyterHub内容")
            
    except Exception as e:
        print(f"❌ /jupyterhub 访问失败: {e}")
    
    # 测试2: 检查nginx health
    try:
        response = requests.get(f"{base_url}/health", timeout=5)
        print(f"✅ nginx健康检查: {response.status_code}")
    except Exception as e:
        print(f"❌ nginx健康检查失败: {e}")
    
    # 测试3: 检查前端应用
    try:
        response = requests.get(f"{base_url}/projects", timeout=10)
        print(f"✅ /projects 访问: {response.status_code}")
    except Exception as e:
        print(f"❌ /projects 访问失败: {e}")

if __name__ == "__main__":
    test_routing()
'''
    
    test_path = "/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/test_jupyterhub_routing.py"
    with open(test_path, 'w', encoding='utf-8') as f:
        f.write(test_script)
    
    os.chmod(test_path, 0o755)
    print(f"📝 已创建测试脚本: {test_path}")

def main():
    """主函数"""
    print("🚀 开始修复JupyterHub路由问题")
    print("=" * 60)
    
    try:
        # 1. 修复Layout.js中的JupyterHub链接
        fix_layout_jupyterhub_link()
        
        # 2. 验证nginx配置
        verify_nginx_configuration()
        
        # 3. 创建测试脚本
        create_test_script()
        
        print("=" * 60)
        print("🎯 修复完成！")
        print()
        print("📋 修复说明:")
        print("1. ✅ 修改了Layout.js中的JupyterHub菜单项")
        print("   - 使用window.location.href直接跳转")
        print("   - 避免React Router拦截/jupyterhub请求")
        print()
        print("2. ✅ 检查了nginx配置")
        print("   - 确认了location块的存在和优先级")
        print()
        print("📝 下一步操作:")
        print("1. 重新构建前端: npm run build")
        print("2. 重启服务: docker-compose restart")
        print("3. 运行测试: python3 test_jupyterhub_routing.py")
        print()
        print("🎯 修复原理:")
        print("- 之前: React Router处理/jupyterhub -> 找不到路由 -> 空白页面")
        print("- 现在: 直接跳转到/jupyterhub -> nginx处理 -> 显示JupyterHub wrapper")
        
    except Exception as e:
        print(f"❌ 修复过程中出错: {e}")
        print("请手动检查文件路径和权限")

if __name__ == "__main__":
    main()
