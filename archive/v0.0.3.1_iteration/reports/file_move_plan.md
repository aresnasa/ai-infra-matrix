# 测试文件分类移动计划
## 测试文件整理完成

### 目录结构：
tests/
├── __init__.py
├── api
│   ├── __init__.py
│   ├── test_api_endpoints.py
│   ├── test_js_redirects.py
│   └── test_redirects.py
├── browser
│   ├── __init__.py
│   ├── monitor_chrome_test.py
│   ├── monitor_iframe.py
│   ├── test_browser_cache.py
│   ├── test_browser_consistency.py
│   ├── test_browser_login.py
│   ├── test_chrome_auto_login.py
│   ├── test_chrome_simple.py
│   └── test_real_browser.py
├── iframe
│   ├── __init__.py
│   ├── iframe_white_screen_fixer.py
│   ├── quick_iframe_test_backup.py
│   ├── quick_iframe_test.py
│   ├── test_fixed_iframe.py
│   ├── test_iframe_auto_login.py
│   ├── test_iframe_chrome.py
│   ├── test_iframe_detailed.py
│   ├── test_iframe_experience.py
│   ├── test_iframe_fix_verification.py
│   ├── test_iframe_quick.py
│   └── test_iframe_simple.py
├── integration
│   ├── __init__.py
│   ├── simple_wrapper_test.py
│   ├── test_complete_flow.py
│   ├── test_complete_redirect_fix.py
│   ├── test_final_verification.py
│   ├── test_jupyter_redirect_fix.py
│   ├── test_projects_jupyter_debug.py
│   ├── test_quick_iframe.py
│   └── test_refresh_behavior.py
├── jupyterhub
│   ├── __init__.py
│   ├── test_jupyterhub_alias.py
│   ├── test_jupyterhub_consistency.py
│   ├── test_jupyterhub_diagnostic.py
│   ├── test_jupyterhub_fixed.py
│   ├── test_jupyterhub_login_complete.ipynb
│   ├── test_jupyterhub_login_complete.py
│   ├── test_jupyterhub_routing_selenium.py
│   ├── test_jupyterhub_routing.py
│   └── test_jupyterhub_wrapper_optimized.py
├── login
│   ├── __init__.py
│   ├── test_quick_login.py
│   ├── test_simple_auto_login.py
│   └── test_sso_complete.py
├── README.md
├── requirements-test.txt
└── utils
    ├── __init__.py
    ├── check_chrome_env.py
    ├── final_verification.py
    ├── fix_jupyterhub_routing.py
    ├── manual_verify.py
    └── verify_portal_consistency.py

8 directories, 57 files
### 各目录文件统计：
- api:        4 个文件
- browser:        9 个文件
- iframe:       12 个文件
- integration:        9 个文件
- jupyterhub:        9 个文件
- login:        4 个文件
- utils:        6 个文件
