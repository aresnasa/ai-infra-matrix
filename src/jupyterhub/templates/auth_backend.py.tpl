# Backend Integrated Authentication Configuration
from backend_integrated_auth import BackendIntegratedAuthenticator

c.JupyterHub.authenticator_class = BackendIntegratedAuthenticator
c.BackendIntegratedAuthenticator.backend_url = '{{AI_INFRA_BACKEND_URL}}'
c.BackendIntegratedAuthenticator.jwt_secret = os.environ.get('JWT_SECRET', '{{JWT_SECRET}}')

# User management configuration
c.Authenticator.allow_all = True  # User permissions controlled by backend
c.Authenticator.admin_users = set()  # Admins determined by backend API

# Auto-login configuration
c.Authenticator.auto_login = {{JUPYTERHUB_AUTO_LOGIN}}
c.Authenticator.auth_refresh_age = {{AUTH_REFRESH_AGE}}
c.Authenticator.refresh_pre_spawn = True
