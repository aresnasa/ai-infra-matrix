#!/usr/bin/env python3
"""
脚本用来清理 models.go 文件中重复的 KubernetesCluster 定义
"""

import re
import os

def clean_duplicates():
    # 设置文件路径
    file_path = '/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/apisChecker/deploybot/ansible-playbook-generator/web-v2/backend/internal/models/models.go'
    
    # 读取文件
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
    # 找到所有 KubernetesCluster 定义的开始位置
    pattern = r'// Kubernetes 集群管理模型\s*\n\s*// KubernetesCluster Kubernetes集群表\s*\ntype KubernetesCluster struct {'
    matches = list(re.finditer(pattern, content))
    
    print(f"Found {len(matches)} KubernetesCluster definitions")
    
    if len(matches) <= 1:
        print("No duplicates found or only one definition exists")
        return
    
    # 保留第一个定义，删除其余的
    # 从后往前删除，这样不会影响前面的位置
    for i in range(len(matches) - 1, 0, -1):
        start = matches[i].start()
        
        # 找到这个定义块的结束位置
        # 寻找下一个 type 定义或者文件末尾
        remaining_content = content[start:]
        
        # 寻找这个 KubernetesCluster 定义块的结束
        # 包括 KubernetesClusterCreateRequest 和 KubernetesClusterUpdateRequest
        end_pattern = r'type KubernetesClusterUpdateRequest struct {[^}]*}\s*\n'
        end_match = re.search(end_pattern, remaining_content)
        
        if end_match:
            end = start + end_match.end()
            
            # 查找下一个非 Kubernetes 相关的类型定义
            next_content = content[end:]
            next_type_pattern = r'\n(// [^\n]*\n)*type (?!KubernetesCluster)'
            next_match = re.search(next_type_pattern, next_content)
            
            if next_match:
                # 删除到下一个类型定义之前
                delete_end = end + next_match.start() + 1  # +1 to keep the newline
            else:
                delete_end = end
            
            print(f"Removing duplicate {i}: lines {start} to {delete_end}")
            content = content[:start] + content[delete_end:]
    
    # 写回文件
    with open(file_path, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print("Cleaning completed")

if __name__ == '__main__':
    clean_duplicates()
