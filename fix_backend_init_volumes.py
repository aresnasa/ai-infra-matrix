#!/usr/bin/env python3
"""
修复生产环境配置文件，为 backend-init 服务添加环境文件 volume 挂载
"""

import yaml
import sys

def fix_backend_init_volumes(input_file, output_file):
    """为 backend-init 服务添加环境文件 volume 挂载"""
    
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
    
    # 修复 backend-init 服务配置
    backend_init = compose_data['services']['backend-init']
    
    # 确保有 volumes 配置
    if 'volumes' not in backend_init:
        backend_init['volumes'] = []
    
    # 添加环境文件挂载（如果不存在）
    env_volume = "./.env.prod:/app/.env:ro"
    if env_volume not in backend_init['volumes']:
        backend_init['volumes'].append(env_volume)
        print("✓ 已添加环境文件挂载: ./.env.prod:/app/.env:ro")
    else:
        print("✓ 环境文件挂载已存在")
    
    # 确保环境变量配置正确
    if 'environment' not in backend_init:
        backend_init['environment'] = []
    
    # 确保有 ENV_FILE 环境变量指向容器内的 .env 文件
    env_file_set = False
    for i, env_var in enumerate(backend_init['environment']):
        if isinstance(env_var, str) and env_var.startswith('ENV_FILE='):
            backend_init['environment'][i] = 'ENV_FILE=/app/.env'
            env_file_set = True
            break
    
    if not env_file_set:
        backend_init['environment'].append('ENV_FILE=/app/.env')
        print("✓ 已添加 ENV_FILE 环境变量")
    
    # 写入修复后的文件
    with open(output_file, 'w', encoding='utf-8') as f:
        yaml.dump(compose_data, f, default_flow_style=False, allow_unicode=True, sort_keys=False)
    
    print(f"✓ 修复完成: {output_file}")
    return True

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("用法: python3 fix_backend_init_volumes.py <input_file> <output_file>")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    try:
        if fix_backend_init_volumes(input_file, output_file):
            print("修复成功!")
            sys.exit(0)
        else:
            print("修复失败!")
            sys.exit(1)
    except Exception as e:
        print(f"错误: {e}")
        sys.exit(1)
