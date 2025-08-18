#!/usr/bin/env python3
"""
User Experience Test for Gitea SSO Integration
Tests the complete user workflow to ensure seamless experience
"""

import requests
import time
from urllib.parse import urlparse, parse_qs

def test_user_experience():
    print("🧪 Testing User Experience for Gitea SSO Integration")
    print("=" * 60)
    
    # Step 1: User has SSO token (logged into main portal)
    print("1. User has logged into the main portal and has SSO token")
    
    # Get SSO token from login
    login_response = requests.post(
        "http://localhost:8080/api/auth/login",
        json={"username": "admin", "password": "admin123"},
        headers={"Content-Type": "application/json"}
    )
    
    if login_response.status_code != 200:
        print("❌ Failed to get SSO token")
        return False
        
    token = login_response.json().get("access_token")
    print(f"   ✅ SSO token obtained: {token[:20]}...")
    
    # Step 2: User visits Gitea directly in browser
    print("\n2. User visits http://localhost:8080/gitea/ in browser")
    
    session = requests.Session()
    session.cookies.set('ai_infra_token', token)
    
    # Simulate browser visiting Gitea
    response = session.get("http://localhost:8080/gitea/", allow_redirects=True)
    
    print(f"   → Final URL: {response.url}")
    print(f"   → Status: {response.status_code}")
    
    # Check if we get the auto-login page or Gitea content
    if "SSO Authentication" in response.text:
        print("   ✅ User sees auto-login bridge page (expected first visit)")
        time.sleep(2)  # Simulate auto-login process
        
        # The JavaScript would normally handle this, simulate the next request
        print("\n3. Auto-login JavaScript establishes Gitea session")
        login_response = session.get("http://localhost:8080/gitea/user/login?redirect_to=/gitea/")
        print(f"   → Login endpoint status: {login_response.status_code}")
        
        # Check for Gitea session cookie
        gitea_cookie = session.cookies.get('i_like_gitea')
        if gitea_cookie:
            print(f"   ✅ Gitea session established: {gitea_cookie[:10]}...")
        else:
            print("   ⚠️ No Gitea session cookie found")
            
        # Now try accessing Gitea again
        print("\n4. User automatically redirected to Gitea dashboard")
        final_response = session.get("http://localhost:8080/gitea/")
        print(f"   → Status: {final_response.status_code}")
        
        if final_response.status_code == 200 and "Gitea" in final_response.text:
            print("   ✅ User successfully lands on Gitea dashboard")
            return True
        else:
            print("   ❌ User did not reach Gitea dashboard")
            return False
            
    elif response.status_code == 200 and "Gitea" in response.text:
        print("   ✅ User directly accesses Gitea (session already exists)")
        return True
    else:
        print(f"   ❌ Unexpected response: {response.status_code}")
        print(f"   Response preview: {response.text[:200]}...")
        return False

def test_admin_access():
    print("\n🔐 Testing Admin Access")
    print("-" * 30)
    
    # Get fresh session
    login_response = requests.post(
        "http://localhost:8080/api/auth/login",
        json={"username": "admin", "password": "admin123"},
        headers={"Content-Type": "application/json"}
    )
    
    token = login_response.json().get("access_token")
    
    session = requests.Session()
    session.cookies.set('ai_infra_token', token)
    
    # Establish Gitea session first
    session.get("http://localhost:8080/gitea/user/login")
    
    # Test admin access
    admin_response = session.get("http://localhost:8080/gitea/admin")
    print(f"Admin access status: {admin_response.status_code}")
    
    if admin_response.status_code == 200:
        print("✅ Admin access successful")
        return True
    else:
        print("❌ Admin access failed")
        return False

if __name__ == "__main__":
    user_test_passed = test_user_experience()
    admin_test_passed = test_admin_access()
    
    print("\n" + "=" * 60)
    print("📋 Test Results:")
    print(f"   User Experience: {'✅ PASS' if user_test_passed else '❌ FAIL'}")
    print(f"   Admin Access: {'✅ PASS' if admin_test_passed else '❌ FAIL'}")
    
    if user_test_passed and admin_test_passed:
        print("\n🎉 All tests passed! Gitea SSO integration is working correctly.")
        print("   Users can now seamlessly access Gitea without manual login.")
    else:
        print("\n⚠️ Some tests failed. Please review the configuration.")
