#!/usr/bin/env python3
"""
修复生产环境服务移除脚本
精确移除openldap、phpldapadmin和redisinsight服务，避免破坏YAML结构
"""
import sys
import yaml
import io

def remove_production_services(input_file, output_file):
    """移除生产环境中非必须的服务"""
    try:
        # 读取YAML文件
        with open(input_file, 'r', encoding='utf-8') as f:
            data = yaml.safe_load(f)
        
        # 移除非必须服务（LDAP和测试工具）
        services_to_remove = ['openldap', 'phpldapadmin', 'redis-insight']
        for service in services_to_remove:
            if service in data.get('services', {}):
                print(f"移除服务: {service}")
                del data['services'][service]
        
        # 移除其他服务对LDAP的依赖
        for service_name, service_config in data.get('services', {}).items():
            if 'depends_on' in service_config:
                # 处理depends_on字典格式
                if isinstance(service_config['depends_on'], dict):
                    if 'openldap' in service_config['depends_on']:
                        print(f"从 {service_name} 移除openldap依赖")
                        del service_config['depends_on']['openldap']
                # 处理depends_on列表格式
                elif isinstance(service_config['depends_on'], list):
                    if 'openldap' in service_config['depends_on']:
                        print(f"从 {service_name} 移除openldap依赖")
                        service_config['depends_on'].remove('openldap')
            
            # 移除LDAP相关环境变量
            if 'environment' in service_config:
                env = service_config['environment']
                ldap_keys = []
                
                if isinstance(env, dict):
                    ldap_keys = [k for k in env.keys() if 'LDAP' in k]
                elif isinstance(env, list):
                    ldap_keys = [i for i, item in enumerate(env) if isinstance(item, str) and 'LDAP' in item]
                
                for key in ldap_keys:
                    print(f"从 {service_name} 移除环境变量: {key}")
                    if isinstance(env, dict):
                        del env[key]
                    elif isinstance(env, list):
                        env.pop(key)
        
        # 写入文件
        with open(output_file, 'w', encoding='utf-8') as f:
            yaml.dump(data, f, default_flow_style=False, sort_keys=False, 
                     allow_unicode=True, width=120, indent=2)
        
        print(f"✓ LDAP服务移除完成: {output_file}")
        return True
        
    except Exception as e:
        print(f"✗ 处理失败: {e}")
        return False

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("用法: python3 fix_ldap_removal.py input.yml output.yml")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_file = sys.argv[2]
    
    if remove_production_services(input_file, output_file):
        sys.exit(0)
    else:
        sys.exit(1)
