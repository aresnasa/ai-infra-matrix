# Kubernetes Spawner Configuration for Production
from kubespawner import KubeSpawner

c.JupyterHub.spawner_class = KubeSpawner

# Kubernetes namespace for single-user pods
c.KubeSpawner.namespace = '{{KUBERNETES_NAMESPACE}}'

# Service account for single-user pods
c.KubeSpawner.service_account = '{{KUBERNETES_SERVICE_ACCOUNT}}'

# Single-user image configuration
c.KubeSpawner.image = os.environ.get('JUPYTERHUB_IMAGE', '{{SINGLEUSER_IMAGE}}')

# Resource limits and guarantees
c.KubeSpawner.mem_limit = '{{JUPYTERHUB_MEM_LIMIT}}'
c.KubeSpawner.cpu_limit = {{JUPYTERHUB_CPU_LIMIT}}
c.KubeSpawner.mem_guarantee = '{{JUPYTERHUB_MEM_GUARANTEE}}'
c.KubeSpawner.cpu_guarantee = {{JUPYTERHUB_CPU_GUARANTEE}}

# Storage configuration
c.KubeSpawner.storage_pvc_ensure = True
c.KubeSpawner.storage_capacity = '{{USER_STORAGE_CAPACITY}}'
c.KubeSpawner.storage_class = '{{JUPYTERHUB_STORAGE_CLASS}}'

# Shared storage configuration
{{SHARED_STORAGE_CONFIG}}

# Environment variables for single-user pods
c.KubeSpawner.environment = {
    'JUPYTER_ENABLE_LAB': '1',
    'AI_INFRA_API_URL': '{{AI_INFRA_BACKEND_URL}}',
    'JUPYTERHUB_K8S_NAMESPACE': '{{KUBERNETES_NAMESPACE}}',
}

# Pod configuration
c.KubeSpawner.pod_name_template = 'jupyter-{username}--{servername}'
c.KubeSpawner.pvc_name_template = 'claim-{username}--{servername}'

# Start timeout
c.KubeSpawner.start_timeout = {{JUPYTERHUB_START_TIMEOUT}}
c.KubeSpawner.http_timeout = {{JUPYTERHUB_HTTP_TIMEOUT}}
