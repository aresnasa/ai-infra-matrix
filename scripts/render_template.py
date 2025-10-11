#!/usr/bin/env python3
"""
Nginx 配置模板渲染工具
支持 {{VAR}} 格式的变量替换，保留 $nginx_var 不变
"""

import sys
import os
import re

def render_template(template_file, output_file, env_vars=None):
    """
    渲染模板文件
    
    Args:
        template_file: 模板文件路径
        output_file: 输出文件路径
        env_vars: 环境变量字典（可选，默认使用 os.environ）
    """
    if env_vars is None:
        env_vars = os.environ
    
    # 读取模板
    try:
        with open(template_file, 'r', encoding='utf-8') as f:
            content = f.read()
    except Exception as e:
        print(f"错误: 无法读取模板文件 {template_file}: {e}", file=sys.stderr)
        return False
    
    # 替换 {{VAR}} 格式的变量
    def replace_var(match):
        var_name = match.group(1)
        value = env_vars.get(var_name, f"{{{{{var_name}}}}}")  # 如果未找到，保留原样
        return value
    
    # 使用正则表达式替换所有 {{VAR}}
    content = re.sub(r'\{\{([A-Z_][A-Z0-9_]*)\}\}', replace_var, content)
    
    # 写入输出文件
    try:
        # 确保输出目录存在
        output_dir = os.path.dirname(output_file)
        if output_dir and not os.path.exists(output_dir):
            os.makedirs(output_dir, exist_ok=True)
        
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(content)
        
        return True
    except Exception as e:
        print(f"错误: 无法写入输出文件 {output_file}: {e}", file=sys.stderr)
        return False

def main():
    if len(sys.argv) < 3:
        print("用法: render_template.py <template_file> <output_file>", file=sys.stderr)
        print("", file=sys.stderr)
        print("示例:", file=sys.stderr)
        print("  export EXTERNAL_HOST=192.168.1.100", file=sys.stderr)
        print("  python3 render_template.py input.tpl output.conf", file=sys.stderr)
        sys.exit(1)
    
    template_file = sys.argv[1]
    output_file = sys.argv[2]
    
    if not os.path.exists(template_file):
        print(f"错误: 模板文件不存在: {template_file}", file=sys.stderr)
        sys.exit(1)
    
    if render_template(template_file, output_file):
        print(f"✓ 模板渲染成功: {output_file}")
        sys.exit(0)
    else:
        print(f"✗ 模板渲染失败", file=sys.stderr)
        sys.exit(1)

if __name__ == '__main__':
    main()
