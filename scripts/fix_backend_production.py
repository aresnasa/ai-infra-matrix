#!/usr/bin/env python3
"""
修复生产环境 backend 服务问题
确保 backend 服务运行正确的命令而不是初始化脚本
"""

import os
import yaml
import sys
from pathlib import Path

def fix_backend_command(compose_file_path):
    """修复 backend 服务的命令配置"""
    
    if not os.path.exists(compose_file_path):
        print(f"错误: docker-compose 文件不存在: {compose_file_path}")
        return False
    
    try:
        # 读取 docker-compose 文件
        with open(compose_file_path, 'r', encoding='utf-8') as f:
            compose_data = yaml.safe_load(f)
        
        if 'services' not in compose_data:
            print("错误: docker-compose 文件中没有 services 部分")
            return False
        
        if 'backend' not in compose_data['services']:
            print("错误: docker-compose 文件中没有 backend 服务")
            return False
        
        backend_service = compose_data['services']['backend']
        
        # 确保 backend 服务使用正确的命令
        # 移除可能存在的错误 command 配置
        if 'command' in backend_service:
            current_command = backend_service['command']
            print(f"发现 backend 服务有自定义命令: {current_command}")
            
            # 如果命令包含 init 或 wait-for-postgres-init，则删除它
            if isinstance(current_command, list):
                command_str = ' '.join(current_command)
            else:
                command_str = str(current_command)
            
            if 'init' in command_str or 'wait-for-postgres-init' in command_str:
                print("检测到错误的初始化命令，删除自定义命令配置...")
                del backend_service['command']
                print("✓ 已删除错误的命令配置，backend 将使用 Dockerfile 中的默认 CMD")
            else:
                print("命令看起来正确，保持不变")
        else:
            print("backend 服务没有自定义命令，将使用 Dockerfile 中的默认 CMD")
        
        # 确保 backend 服务有正确的健康检查
        if 'healthcheck' not in backend_service:
            print("添加健康检查配置...")
            backend_service['healthcheck'] = {
                'test': ["CMD", "curl", "-f", "http://localhost:8082/api/health"],
                'interval': '30s',
                'timeout': '15s',
                'retries': 5,
                'start_period': '60s'
            }
            print("✓ 已添加健康检查配置")
        
        # 确保正确的依赖关系
        if 'depends_on' in backend_service:
            depends_on = backend_service['depends_on']
            if 'backend-init' in depends_on:
                if isinstance(depends_on['backend-init'], dict):
                    if depends_on['backend-init'].get('condition') != 'service_completed_successfully':
                        depends_on['backend-init']['condition'] = 'service_completed_successfully'
                        print("✓ 修复了 backend-init 依赖条件")
                else:
                    depends_on['backend-init'] = {'condition': 'service_completed_successfully'}
                    print("✓ 更新了 backend-init 依赖配置")
        
        # 写回文件
        with open(compose_file_path, 'w', encoding='utf-8') as f:
            yaml.dump(compose_data, f, default_flow_style=False, allow_unicode=True, indent=2)
        
        print(f"✓ 已修复 backend 服务配置: {compose_file_path}")
        return True
        
    except Exception as e:
        print(f"错误: 处理文件时出错: {e}")
        return False

def main():
    """主函数"""
    if len(sys.argv) < 2:
        print("用法: python3 fix_backend_production.py <docker-compose-file-path>")
        print("示例: python3 fix_backend_production.py docker-compose.prod.yml")
        sys.exit(1)
    
    compose_file = sys.argv[1]
    
    print("🔧 修复生产环境 backend 服务配置...")
    print(f"目标文件: {compose_file}")
    
    if fix_backend_command(compose_file):
        print("✅ 修复完成！")
        print("\n接下来的步骤:")
        print("1. 重新构建 backend 镜像（确保使用正确的 target）:")
        print("   ./build.sh --build backend --env prod --registry YOUR_REGISTRY")
        print("2. 重启 backend 服务:")
        print("   docker-compose -f docker-compose.prod.yml up -d backend")
        print("3. 检查服务状态:")
        print("   docker-compose -f docker-compose.prod.yml logs backend")
    else:
        print("❌ 修复失败")
        sys.exit(1)

if __name__ == "__main__":
    main()
