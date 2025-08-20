"""
测试配置文件
"""

# 默认测试配置
DEFAULT_CONFIG = {
    "base_url": "http://localhost:8080",
    "credentials": {
        "username": "admin",
        "password": "admin123"
    },
    "endpoints": {
        "auth_login": "/api/auth/login",
        "auth_verify": "/api/auth/verify",
        "gitea_login": "/gitea/user/login",
        "gitea_admin": "/gitea/admin",
        "jupyterhub": "/jupyter/",
        "frontend": "/"
    },
    "timeouts": {
        "request": 10,
        "auth": 5
    }
}

# 测试场景配置
TEST_SCENARIOS = {
    "sso_redirect": {
        "name": "SSO自动重定向测试",
        "url": "/gitea/user/login?redirect_to=%2Fgitea%2Fadmin",
        "expected_status": [302, 303],
        "expected_location_contains": "/gitea/"
    },
    "no_token_access": {
        "name": "无token访问测试", 
        "url": "/gitea/user/login",
        "expected_status": [200],
        "expected_content_contains": ["password", "login"]
    },
    "jupyterhub_access": {
        "name": "JupyterHub访问测试",
        "url": "/jupyter/",
        "expected_status": [200, 302, 303]
    }
}

# 健康检查端点
HEALTH_ENDPOINTS = {
    "frontend": "/",
    "backend": "/api/health", 
    "gitea": "/gitea/",
    "jupyterhub": "/jupyter/hub/health"
}
