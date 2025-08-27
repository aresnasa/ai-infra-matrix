#!/usr/bin/env python3
"""
修复 docker-compose.prod.yml 中 backend-init 服务的重启策略
"""

import yaml
import sys

def fix_backend_init_restart(input_file, output_file):
    """修复 backend-init 服务的重启策略"""
    
    # 读取 YAML 文件
    with open(input_file, 'r', encoding='utf-8') as f:
        compose_data = yaml.safe_load(f)
    
    # 检查是否有 services 和 backend-init
    if 'services' not in compose_data:
        print("错误: 未找到 services 配置")
        return False
    
    if 'backend-init' not in compose_data['services']:
        print("错误: 未找到 backend-init 服务")
        return False
    
    # 修复 backend-init 的重启策略
    backend_init = compose_data['services']['backend-init']
    
    # 设置重启策略为 "no"
    backend_init['restart'] = 'no'
    
    print("✓ 已将 backend-init 服务的重启策略设置为 'no'")
    
    # 写入修复后的文件
    with open(output_file, 'w', encoding='utf-8') as f:
        yaml.dump(compose_data, f, default_flow_style=False, allow_unicode=True, sort_keys=False)
    
    print(f"✓ 修复完成: {output_file}")
    return True

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("用法: python3 fix_backend_init_restart.py <input_file> <output_file>")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    try:
        if fix_backend_init_restart(input_file, output_file):
            print("修复成功!")
            sys.exit(0)
        else:
            print("修复失败!")
            sys.exit(1)
    except Exception as e:
        print(f"错误: {e}")
        sys.exit(1)
