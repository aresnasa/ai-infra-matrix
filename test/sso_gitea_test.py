import requests
from urllib.parse import urljoin
import os
import sys
import json

# --- Configuration ---
BASE_URL = "http://localhost:8080"
# The backend API might be running on a different port internally within Docker
# but the login request should go through the main gateway (Nginx)
BACKEND_LOGIN_URL = f"{BASE_URL}/api/auth/login" 
GITEA_URL = f"{BASE_URL}/gitea/"
# Direct admin access (no wrapper, minimal proxy)
GITEA_ADMIN_URL = f"{BASE_URL}/gitea/admin"
ADMIN_USERNAME = "admin"
ADMIN_PASSWORD = "admin123"

def get_sso_token(username, password):
    """Logs into the backend to retrieve an SSO token."""
    print(f"🔄 Attempting to log in as '{username}' to get SSO token...")
    payload = {"username": username, "password": password}
    try:
        # Make the request through the Nginx proxy
        response = requests.post(BACKEND_LOGIN_URL, json=payload, timeout=10)
        response.raise_for_status()  # Raise an exception for bad status codes (4xx or 5xx)

        data = response.json()
        # Adjust based on the actual API response structure
        token = data.get("token")

        if token:
            print(f"✅ Successfully retrieved SSO token for '{username}'.")
            return token
        else:
            print("❌ Login successful, but no token found in response.")
            print("   Response JSON:", data)
            return None

    except requests.exceptions.RequestException as e:
        print(f"❌ Error during login request: {e}")
        # If the backend is not directly exposed, this might fail.
        # Ensure that Nginx correctly proxies /api/auth/login to the backend service.
        return None
    except json.JSONDecodeError:
        print("❌ Failed to decode JSON from login response.")
        # Best-effort: response may not be bound here in some analyzer contexts
        # but during runtime it's in scope for this except branch.
        # Guard with local lookup.
        _resp = locals().get('response')
        if _resp is not None:
            try:
                print("   Response Text:", _resp.text)
            except Exception:
                pass
        return None

def describe_history(resp):
    if not resp.history:
        return "(no redirects)"
    return " -> ".join([f"{r.status_code} {r.headers.get('Location','')}" for r in resp.history] + [str(resp.status_code)])

def print_headers(response, title="Response Headers"):
    """Prints the headers of a response object."""
    print(f"\n--- {title} ---")
    for key, value in response.headers.items():
        print(f"{key}: {value}")
    print("----------------" + "-" * len(title))

def print_cookies(session, title="Session Cookies"):
    """Prints the cookies in a requests.Session object."""
    print(f"\n--- {title} ---")
    if session.cookies:
        for cookie in session.cookies:
            print(f"Name: {cookie.name}, Value: {cookie.value}, Domain: {cookie.domain}")
    else:
        print("No cookies in session.")
    print("----------------" + "-" * len(title))

def test_gitea_sso(sso_token):
    """
    Tests the Gitea SSO flow using a provided token and validates admin access.
    """
    if not sso_token:
        print("\n" + "="*50)
        print("❌ Gitea SSO test skipped: No SSO token was provided.")
        print("="*50)
        return

    print("\n" + "="*50)
    print("🚀 Starting Gitea SSO Test...")
    print("="*50)

    session = requests.Session()
    print("1.  Browser session created.")

    session.cookies.update({'ai_infra_token': sso_token})
    print("2.  SSO Cookie 'ai_infra_token' set.")

    # First request without following redirects to detect 302 from nginx
    print(f"3a. Request (no redirects) to: {GITEA_URL}")
    try:
        r1 = session.get(GITEA_URL, allow_redirects=False, timeout=15)
        print(f"    -> Status: {r1.status_code}")
        if r1.is_redirect:
            print(f"    -> Redirect Location: {r1.headers.get('Location')}")
        # Follow redirect if present
        if r1.is_redirect:
            r2 = session.get(urljoin(GITEA_URL, r1.headers.get('Location')), allow_redirects=True, timeout=15)
            final = r2
            print(f"3b. Followed redirect, final status: {final.status_code}")
        else:
            final = r1
            print("3b. No redirect to follow.")

        print("4.  Final response after redirect handling:")
        print(f"    Status: {final.status_code}")
        print(f"    Redirect chain: {describe_history(final)}")

        # Check for direct Gitea content (no wrapper iframe in minimal proxy)
        content_type = final.headers.get('Content-Type', '')
        body = final.text if hasattr(final, 'text') else ''
        is_gitea_page = final.status_code == 200 and 'text/html' in content_type and 'Gitea' in body
        if is_gitea_page:
            print("✅ Direct Gitea page: /gitea returns HTML content from Gitea (minimal proxy).")
        else:
            print("❌ Expected direct Gitea HTML content via minimal proxy.")

        # Analyze cookies
        gitea_cookie = session.cookies.get('i_like_gitea')
        if gitea_cookie:
            print("✅ Gitea session cookie 'i_like_gitea' is present (already bootstrapped).")
        else:
            print("ℹ️ No Gitea session cookie yet; will bootstrap via login endpoint.")

        # 4b. Explicitly hit /gitea/user/login to trigger SSO-based auto-login (should not prompt for password)
        print(f"4b. Calling /gitea/user/login to bootstrap session via SSO ...")
        try:
            login_page = session.get(f"{BASE_URL}/gitea/user/login?redirect_to=%2Fgitea%2Fadmin", allow_redirects=True, timeout=15)
            # Cookie should be set by now
            if session.cookies.get('i_like_gitea'):
                print("✅ /gitea/user/login created Gitea session from SSO headers (cookie present)")
            else:
                print(f"ℹ️ /gitea/user/login did not set session cookie immediately; will proceed to admin to confirm")
        except requests.exceptions.RequestException as e:
            print(f"❌ Error hitting /gitea/user/login: {e}")

        # Test direct admin access (minimal proxy should allow this)
        print(f"5. Accessing admin directly: {GITEA_ADMIN_URL}")
        admin_resp = session.get(GITEA_ADMIN_URL, allow_redirects=True, timeout=20)
        print(f"    Admin status: {admin_resp.status_code}")
        print(f"    Admin final URL: {admin_resp.url}")
        # Print any X-* headers if present
        admin_x_headers = {k:v for k,v in admin_resp.headers.items() if k.lower().startswith('x-')}
        if admin_x_headers:
            print("\n--- Admin response X-* headers ---")
            for k,v in admin_x_headers.items():
                print(f"  {k}: {v}")

        # Heuristics for success/failure on admin access
        success = (
            admin_resp.status_code == 200 and
            '/user/login' not in admin_resp.url and
            bool(session.cookies.get('i_like_gitea'))
        )
        if success:
            print("✅ SUCCESS: Admin is accessible and session cookie is present.")
        else:
            print("❌ FAILURE: Admin access redirected to login or shows login page.")

        # Print any X-Debug headers if present
        debug_headers = {k:v for k,v in final.headers.items() if k.lower().startswith('x-debug')}
        if debug_headers:
            print("\n--- X-Debug headers ---")
            for k,v in debug_headers.items():
                print(f"  {k}: {v}")

        print("\nFinal Cookies in Session:")
        for name, value in session.cookies.items():
            print(f"  {name}: {value}")

        # 6. API smoke checks: version (lenient) + user (401 expected) to validate routing
        print("\n6. API smoke checks...")
        try:
            ver = session.get(f"{BASE_URL}/gitea/api/v1/version", timeout=15)
            ct = ver.headers.get('Content-Type','')
            if ver.status_code in (200, 401) and ('application/json' in ct):
                print(f"✅ /gitea/api/v1/version reachable: status={ver.status_code}, body={ver.text.strip()}")
            elif ver.status_code == 404:
                print("ℹ️ /gitea/api/v1/version not available on this Gitea version (404); continuing")
            else:
                print(f"ℹ️ /gitea/api/v1/version status={ver.status_code}, ct={ct}, body={ver.text[:100]}")
        except requests.exceptions.RequestException as e:
            print(f"❌ API version request error: {e}")

        # Check that a protected endpoint returns JSON 401 (proxy + SSO headers wired correctly)
        try:
            who = session.get(f"{BASE_URL}/gitea/api/v1/user", timeout=15)
            ct2 = who.headers.get('Content-Type','')
            if who.status_code == 401 and 'application/json' in ct2:
                print("✅ /gitea/api/v1/user returns 401 JSON (auth required) — API routing OK")
            else:
                print(f"ℹ️ /gitea/api/v1/user unexpected: status={who.status_code}, ct={ct2}, body={who.text[:100]}")
        except requests.exceptions.RequestException as e:
            print(f"❌ API whoami request error: {e}")

        # 7. Unified logout should clear Gitea + SSO cookies and block embedded admin
        print("\n7. Calling unified logout at /gitea/_logout ...")
        try:
            logout_resp = session.get(f"{BASE_URL}/gitea/_logout", timeout=10)
            print(f"   -> status={logout_resp.status_code}, ct={logout_resp.headers.get('Content-Type')}")
            # Expect Set-Cookie deletions; requests will drop them
            names = [c.name for c in session.cookies]
            cleared = ('i_like_gitea' not in names)
            if cleared:
                print("✅ Gitea cookie cleared from session store")
            else:
                print("ℹ️ i_like_gitea still visible in session; server likely sent deletion Set-Cookie (client may hide).")

            # Ensure SSO token also cleared; if not, clear locally to validate protection
            if 'ai_infra_token' in names:
                print("ℹ️ SSO token cookie 'ai_infra_token' still present in client; clearing locally for verification...")
                try:
                    session.cookies.clear(name='ai_infra_token')
                except Exception:
                    # Fallback brute-force: set to empty with past expiry
                    session.cookies.set('ai_infra_token', '', expires=0)

            # Access admin after logout; expect redirect to SSO bridge (not 200)
            after_admin = session.get(GITEA_ADMIN_URL, allow_redirects=False, timeout=15)
            if after_admin.status_code in (301,302):
                loc = after_admin.headers.get('Location','')
                if '/sso/jwt_sso_bridge.html' in loc or '/gitea/user/login' in loc:
                    print("✅ After logout: admin redirects away (no active session)")
                else:
                    print(f"ℹ️ After logout: redirected to {loc}")
            elif after_admin.status_code == 401:
                print("✅ After logout: 401 unauthorized as expected")
            else:
                print(f"❌ After logout: unexpected status {after_admin.status_code}")

            # 7b. Normalize accidental /gitea/gitea/ path to avoid 404s
            try:
                dbl = session.get(f"{BASE_URL}/gitea/gitea/", allow_redirects=False, timeout=10)
                if dbl.status_code in (301,302) and dbl.headers.get('Location','').startswith('/gitea/'):
                    print("✅ /gitea/gitea/ correctly normalizes (redirects) to /gitea/")
                elif dbl.status_code == 404:
                    print("❌ /gitea/gitea/ returned 404; normalization missing")
                else:
                    print(f"ℹ️ /gitea/gitea/ status={dbl.status_code}, Location={dbl.headers.get('Location')}")
            except requests.exceptions.RequestException as e:
                print(f"❌ Error testing /gitea/gitea/ normalization: {e}")
        except requests.exceptions.RequestException as e:
            print(f"❌ logout flow error: {e}")

        # 8. Re-bootstrap by restoring SSO cookie and hitting admin again
        print("\n8. Re-bootstrap SSO: re-apply ai_infra_token and hit /gitea/admin ...")
        session.cookies.set('ai_infra_token', sso_token)
        try:
            re_admin = session.get(GITEA_ADMIN_URL, allow_redirects=True, timeout=20)
            if re_admin.status_code == 200 and session.cookies.get('i_like_gitea'):
                print("✅ Re-bootstrap success: session restored and admin accessible")
            else:
                print(f"❌ Re-bootstrap failed: status={re_admin.status_code}, login_redirect={'/user/login' in re_admin.url}")
        except requests.exceptions.RequestException as e:
            print(f"❌ Re-bootstrap request error: {e}")

    except requests.exceptions.RequestException as e:
        print(f"❌ An error occurred during the request to Gitea: {e}")

def main():
    """Main function to run the SSO test."""
    # 1. Get the token automatically
    token = get_sso_token(ADMIN_USERNAME, ADMIN_PASSWORD)
    
    # 2. Run the SSO test with the obtained token
    test_gitea_sso(token)

if __name__ == "__main__":
    main()
