#!/usr/bin/env python3
"""
JupyterHub集成测试脚本
测试主页到JupyterHub的完整流程
"""

import requests
import json
import time
import sys

def test_api_endpoints():
    """测试API端点"""
    base_url = "http://localhost:8080"
    
    print("🧪 Testing JupyterHub Integration...")
    
    # 测试状态API
    print("\n1. Testing JupyterHub Status API...")
    try:
        response = requests.get(f"{base_url}/api/jupyterhub/status", timeout=10)
        if response.status_code == 200:
            status_data = response.json()
            print(f"   ✅ Status API working - Running: {status_data.get('running')}")
            print(f"   📊 Users online: {status_data.get('users_online')}")
            print(f"   🖥️ Servers running: {status_data.get('servers_running')}")
        else:
            print(f"   ❌ Status API failed with status: {response.status_code}")
            return False
    except Exception as e:
        print(f"   ❌ Status API error: {e}")
        return False
    
    # 测试用户任务API
    print("\n2. Testing User Tasks API...")
    try:
        response = requests.get(f"{base_url}/api/jupyterhub/user-tasks", timeout=10)
        if response.status_code == 200:
            tasks_data = response.json()
            tasks = tasks_data.get('tasks', [])
            print(f"   ✅ User Tasks API working - Found {len(tasks)} tasks")
            for task in tasks[:2]:  # 显示前2个任务
                print(f"   📝 Task: {task.get('task_name')} - Status: {task.get('status')}")
        else:
            print(f"   ❌ User Tasks API failed with status: {response.status_code}")
            return False
    except Exception as e:
        print(f"   ❌ User Tasks API error: {e}")
        return False
    
    # 测试前端页面
    print("\n3. Testing Frontend Page...")
    try:
        response = requests.get(f"{base_url}", timeout=10)
        if response.status_code == 200:
            print("   ✅ Frontend page accessible")
        else:
            print(f"   ❌ Frontend page failed with status: {response.status_code}")
            return False
    except Exception as e:
        print(f"   ❌ Frontend page error: {e}")
        return False
    
    return True

def test_jupyterhub_access():
    """测试JupyterHub访问"""
    print("\n4. Testing JupyterHub Direct Access...")
    try:
        response = requests.get("http://localhost:8080/jupyter/", timeout=10, allow_redirects=False)
        if response.status_code in [200, 302, 301]:
            print("   ✅ JupyterHub accessible via nginx proxy")
        else:
            print(f"   ⚠️ JupyterHub response status: {response.status_code}")
    except Exception as e:
        print(f"   ⚠️ JupyterHub access note: {e}")

def main():
    """主测试函数"""
    print("=" * 60)
    print("🚀 AI Infrastructure Matrix - JupyterHub Integration Test")
    print("=" * 60)
    
    # 测试API端点
    if test_api_endpoints():
        print("\n✨ All API tests passed!")
    else:
        print("\n❌ Some API tests failed!")
        sys.exit(1)
    
    # 测试JupyterHub访问
    test_jupyterhub_access()
    
    print("\n" + "=" * 60)
    print("🎉 JupyterHub Integration Test Complete!")
    print("=" * 60)
    print("\n📋 Next Steps:")
    print("   1. Open http://localhost:8080 in your browser")
    print("   2. Navigate to JupyterHub page from the menu")
    print("   3. View status and task information")
    print("   4. Click 'Open JupyterHub' to access Jupyter environment")
    print("\n💡 Note: Login with any username/password (DummyAuthenticator)")

if __name__ == "__main__":
    main()
