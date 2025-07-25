#!/usr/bin/env python3
import os
import glob

# 设置工作目录
backend_dir = "/Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/src/backend"
os.chdir(backend_dir)

# 查找所有.go文件
go_files = glob.glob("**/*.go", recursive=True)

old_import = "ansible-playbook-generator-backend"
new_import = "github.com/aresnasa/ai-infra-matrix/src/backend"

print(f"Found {len(go_files)} Go files to process")

for file_path in go_files:
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        if old_import in content:
            updated_content = content.replace(old_import, new_import)
            
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(updated_content)
            
            print(f"Updated: {file_path}")
    except Exception as e:
        print(f"Error processing {file_path}: {e}")

print("Import path replacement completed!")
