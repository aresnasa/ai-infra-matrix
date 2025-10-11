#!/usr/bin/env python3
"""
Template Renderer for AI Infrastructure Matrix
处理模板文件中的变量替换，支持 {{VAR}} 格式
"""

import os
import sys
import re
from datetime import datetime

def get_template_variables():
    """获取模板变量的默认值和环境变量值"""
    
    # 设置默认值
    defaults = {
        'BACKEND_HOST': 'backend',
        'BACKEND_PORT': '8082',
        'FRONTEND_HOST': 'frontend', 
        'FRONTEND_PORT': '80',
        'JUPYTERHUB_HOST': 'jupyterhub',
        'JUPYTERHUB_PORT': '8000',
        'EXTERNAL_SCHEME': 'http',
        'EXTERNAL_HOST': 'localhost',
        'GITEA_ALIAS_ADMIN_TO': 'admin',
        'GITEA_ADMIN_EMAIL': 'admin@example.com',
        
        # JupyterHub特定变量
        'ENVIRONMENT': 'development',
        'AUTH_TYPE': 'local',
        'GENERATION_TIME': datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
        'JUPYTERHUB_HUB_PORT': '8081',
        'JUPYTERHUB_BASE_URL': '/jupyter/',
        'JUPYTERHUB_HUB_CONNECT_HOST': 'jupyterhub',
        'JUPYTERHUB_PUBLIC_URL': 'http://localhost:8080/jupyter/',
        'CONFIGPROXY_AUTH_TOKEN': 'ai-infra-proxy-token-dev',
        'JUPYTERHUB_DB_URL': 'sqlite:///jupyterhub.sqlite',
        'JUPYTERHUB_LOG_LEVEL': 'INFO',
        'SESSION_TIMEOUT_DAYS': '7',
        'SINGLEUSER_IMAGE': 'ai-infra-singleuser:latest',
        'DOCKER_NETWORK': 'ai-infra-matrix_default',
        'JUPYTERHUB_MEM_LIMIT': '2G',
        'JUPYTERHUB_CPU_LIMIT': '1.0',
        'JUPYTERHUB_MEM_GUARANTEE': '1G',
        'JUPYTERHUB_CPU_GUARANTEE': '0.5',
        'USER_STORAGE_CAPACITY': '10Gi',
        'JUPYTERHUB_STORAGE_CLASS': 'default',
        'SHARED_STORAGE_PATH': '/srv/shared-notebooks',
        'AI_INFRA_BACKEND_URL': 'http://backend:8082',
        'KUBERNETES_NAMESPACE': 'ai-infra-users',
        'KUBERNETES_SERVICE_ACCOUNT': 'ai-infra-matrix-jupyterhub',
        'JUPYTERHUB_START_TIMEOUT': '300',
        'JUPYTERHUB_HTTP_TIMEOUT': '30',
        'JWT_SECRET': '',
        'JUPYTERHUB_AUTO_LOGIN': 'False',
        'AUTH_REFRESH_AGE': '3600',
        'ADMIN_USERS': "'admin'",
        
        # 子模板配置 - 从环境变量中读取
        'AUTH_CONFIG': os.environ.get('AUTH_CONFIG', ''),
        'SPAWNER_CONFIG': os.environ.get('SPAWNER_CONFIG', ''),
        'SHARED_STORAGE_CONFIG': os.environ.get('SHARED_STORAGE_CONFIG', ''),
        'ADDITIONAL_CONFIG': os.environ.get('ADDITIONAL_CONFIG', '')
    }
    
    # 合并环境变量值，环境变量优先
    variables = {}
    for key, default_value in defaults.items():
        env_value = os.environ.get(key)
        if env_value is not None and env_value != '':
            variables[key] = env_value
        else:
            variables[key] = default_value
    
    return variables

def render_template(template_file, output_file):
    """渲染模板文件"""
    
    try:
        # 读取模板文件
        with open(template_file, 'r', encoding='utf-8') as f:
            template_content = f.read()
        
        # 获取模板变量
        variables = get_template_variables()
        
        # 替换模板变量 {{VAR}} 格式
        def replace_var(match):
            var_name = match.group(1)
            return variables.get(var_name, match.group(0))  # 如果变量不存在，保持原样
        
        # 使用正则表达式替换所有 {{VAR}} 格式的变量
        rendered_content = re.sub(r'\{\{([^}]+)\}\}', replace_var, template_content)
        
        # 确保输出目录存在
        output_dir = os.path.dirname(output_file)
        if output_dir and not os.path.exists(output_dir):
            os.makedirs(output_dir)
        
        # 写入输出文件
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write(rendered_content)
        
        print(f"✓ 模板渲染完成: {output_file}")
        return True
        
    except Exception as e:
        print(f"✗ 模板渲染失败: {e}", file=sys.stderr)
        return False

def main():
    """主函数"""
    if len(sys.argv) != 3:
        print("用法: python3 template_renderer.py <template_file> <output_file>")
        sys.exit(1)
    
    template_file = sys.argv[1]
    output_file = sys.argv[2]
    
    if not os.path.exists(template_file):
        print(f"✗ 模板文件不存在: {template_file}", file=sys.stderr)
        sys.exit(1)
    
    success = render_template(template_file, output_file)
    sys.exit(0 if success else 1)

if __name__ == '__main__':
    main()
