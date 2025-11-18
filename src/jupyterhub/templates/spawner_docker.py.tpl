# Docker Spawner Configuration for Development
from dockerspawner import DockerSpawner

c.JupyterHub.spawner_class = DockerSpawner
c.DockerSpawner.image = os.environ.get('SINGLEUSER_IMAGE', '{{SINGLEUSER_IMAGE}}')
c.DockerSpawner.network_name = '{{DOCKER_NETWORK}}'
c.DockerSpawner.remove = True

# Resource limits
c.DockerSpawner.mem_limit = '{{JUPYTERHUB_MEM_LIMIT}}'
c.DockerSpawner.cpu_limit = {{JUPYTERHUB_CPU_LIMIT}}

# Volume mounts
c.DockerSpawner.volumes = {
    'jupyterhub-user-{username}': '/home/jovyan/work',
    '{{SHARED_STORAGE_PATH}}': '/home/jovyan/shared'
}

# Environment variables for single-user containers
c.DockerSpawner.environment = {
    'JUPYTER_ENABLE_LAB': '1',
    'AI_INFRA_API_URL': '{{AI_INFRA_BACKEND_URL}}',
    'JUPYTERHUB_K8S_NAMESPACE': '{{KUBERNETES_NAMESPACE}}',
}

# Use internal IP for communication
c.DockerSpawner.use_internal_ip = True
