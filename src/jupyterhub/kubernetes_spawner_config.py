"""
JupyterHub KubeSpawner配置 - 支持Kubernetes多节点部署
在原有DockerSpawner基础上添加KubeSpawner支持，支持动态切换
"""

import os
import logging
from kubespawner import KubeSpawner
from kubernetes import client
from traitlets import Unicode, Dict, List, Bool, Int

logger = logging.getLogger(__name__)

class AIInfraKubeSpawner(KubeSpawner):
    """
    AI-Infra定制的KubeSpawner
    支持动态配置、资源管理和用户隔离
    """
    
    # 自定义配置属性
    custom_pod_labels = Dict(
        config=True,
        help="自定义Pod标签"
    )
    
    custom_pod_annotations = Dict(
        config=True, 
        help="自定义Pod注解"
    )
    
    enable_shared_storage = Bool(
        True,
        config=True,
        help="是否启用共享存储挂载"
    )
    
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        logger.info("🚀 AIInfraKubeSpawner 初始化")
    
    def _get_pod_manifest(self):
        """生成Pod清单，添加AI-Infra特定配置"""
        manifest = super()._get_pod_manifest()
        
        # 添加自定义标签
        if self.custom_pod_labels:
            if 'labels' not in manifest['metadata']:
                manifest['metadata']['labels'] = {}
            manifest['metadata']['labels'].update(self.custom_pod_labels)
        
        # 添加自定义注解
        if self.custom_pod_annotations:
            if 'annotations' not in manifest['metadata']:
                manifest['metadata']['annotations'] = {}
            manifest['metadata']['annotations'].update(self.custom_pod_annotations)
        
        # 添加AI-Infra标识
        manifest['metadata']['labels']['ai-infra.component'] = 'singleuser-pod'
        manifest['metadata']['labels']['ai-infra.user'] = self.user.name
        
        return manifest
    
    async def start(self):
        """启动单用户Pod，添加AI-Infra特定逻辑"""
        logger.info(f"🎯 启动用户Pod: {self.user.name}")
        
        # 设置用户特定的环境变量
        if not hasattr(self, 'environment') or not self.environment:
            self.environment = {}
        
        self.environment.update({
            'AI_INFRA_USER': self.user.name,
            'AI_INFRA_USER_ID': str(self.user.id),
            'JUPYTER_ENABLE_LAB': 'yes',
            'JUPYTER_LAB_INTERFACE': 'lab',
        })
        
        # 如果用户有认证状态，传递相关信息
        if hasattr(self.user, 'auth_state') and self.user.auth_state:
            user_info = self.user.auth_state.get('user_info', {})
            if user_info:
                self.environment['AI_INFRA_USER_ROLES'] = ','.join(user_info.get('roles', []))
        
        return await super().start()

def configure_kubespawner(c):
    """配置KubeSpawner参数"""
    logger.info("🔧 配置KubeSpawner...")
    
    # 基础配置
    c.JupyterHub.spawner_class = AIInfraKubeSpawner
    
    # Kubernetes命名空间配置
    user_namespace = os.environ.get('KUBERNETES_NAMESPACE', 'ai-infra-users')
    c.KubeSpawner.namespace = user_namespace
    
    # 镜像配置
    singleuser_image = os.environ.get('JUPYTERHUB_IMAGE', 'ai-infra-singleuser:v0.3.6-dev')
    c.KubeSpawner.image = singleuser_image
    c.KubeSpawner.image_pull_policy = 'IfNotPresent'
    
    # 资源限制配置
    mem_limit = os.environ.get('JUPYTERHUB_MEM_LIMIT', '2G')
    cpu_limit = float(os.environ.get('JUPYTERHUB_CPU_LIMIT', '1.0'))
    mem_guarantee = os.environ.get('JUPYTERHUB_MEM_GUARANTEE', '1G')
    cpu_guarantee = float(os.environ.get('JUPYTERHUB_CPU_GUARANTEE', '0.5'))
    
    c.KubeSpawner.mem_limit = mem_limit
    c.KubeSpawner.cpu_limit = cpu_limit
    c.KubeSpawner.mem_guarantee = mem_guarantee
    c.KubeSpawner.cpu_guarantee = cpu_guarantee
    
    # 存储配置
    storage_class = os.environ.get('JUPYTERHUB_STORAGE_CLASS', 'local-path')
    c.KubeSpawner.storage_class = storage_class
    c.KubeSpawner.storage_capacity = '10Gi'
    c.KubeSpawner.storage_access_modes = ['ReadWriteOnce']
    
    # 工作目录配置
    c.KubeSpawner.notebook_dir = '/home/jovyan/work'
    c.KubeSpawner.working_dir = '/home/jovyan'
    
    # 启动命令配置
    c.KubeSpawner.cmd = ['start-singleuser.sh']
    c.KubeSpawner.args = []
    
    # 网络配置
    c.KubeSpawner.port = 8888
    
    # ServiceAccount配置
    c.KubeSpawner.service_account = os.environ.get('KUBERNETES_SERVICE_ACCOUNT', 'default')
    
    # 超时配置
    c.KubeSpawner.start_timeout = int(os.environ.get('JUPYTERHUB_START_TIMEOUT', '300'))
    c.KubeSpawner.http_timeout = int(os.environ.get('JUPYTERHUB_HTTP_TIMEOUT', '120'))
    
    # Pod安全配置
    c.KubeSpawner.fs_gid = 100  # jovyan group
    c.KubeSpawner.supplemental_gids = [100]
    
    # 环境变量配置
    c.KubeSpawner.environment = {
        'JUPYTER_ENABLE_LAB': 'yes',
        'JUPYTER_LAB_INTERFACE': 'lab',
        'GRANT_SUDO': 'no',  # 安全考虑，不授予sudo权限
        'CHOWN_HOME': 'yes',
        'CHOWN_HOME_OPTS': '-R',
    }
    
    # 共享存储配置（如果启用）
    shared_storage_enabled = os.environ.get('SHARED_STORAGE_ENABLED', 'true').lower() == 'true'
    if shared_storage_enabled:
        shared_storage_class = os.environ.get('SHARED_STORAGE_CLASS', 'nfs-client')
        
        # 添加共享存储卷
        c.KubeSpawner.volumes = [
            {
                'name': 'shared-notebooks',
                'persistentVolumeClaim': {
                    'claimName': 'ai-infra-shared-notebooks'
                }
            }
        ]
        
        c.KubeSpawner.volume_mounts = [
            {
                'name': 'shared-notebooks',
                'mountPath': '/home/jovyan/shared-notebooks',
                'readOnly': False
            }
        ]
    
    # 自定义Pod标签
    c.AIInfraKubeSpawner.custom_pod_labels = {
        'app.kubernetes.io/name': 'jupyterhub-singleuser',
        'app.kubernetes.io/component': 'singleuser-pod',
        'ai-infra.project': 'ai-infra-matrix',
        'ai-infra.spawner': 'kubespawner'
    }
    
    # 自定义Pod注解
    c.AIInfraKubeSpawner.custom_pod_annotations = {
        'ai-infra.spawned-by': 'jupyterhub',
        'ai-infra.version': 'v0.3.6-dev'
    }
    
    # Pod模板配置 - 更精细的控制
    c.KubeSpawner.extra_pod_config = {
        'restartPolicy': 'Never',
        'automountServiceAccountToken': False,  # 安全考虑
    }
    
    # 容器额外配置
    c.KubeSpawner.extra_container_config = {
        'securityContext': {
            'runAsNonRoot': True,
            'runAsUser': 1000,  # jovyan user
            'runAsGroup': 100,  # jovyan group
            'allowPrivilegeEscalation': False,
            'capabilities': {
                'drop': ['ALL']
            }
        }
    }
    
    # 删除策略：用户停止时删除Pod
    c.KubeSpawner.delete_grace_period = 30
    c.KubeSpawner.delete_timeout = 60
    
    logger.info("✅ KubeSpawner配置完成")
    logger.info(f"📍 命名空间: {user_namespace}")
    logger.info(f"📍 镜像: {singleuser_image}")
    logger.info(f"📍 资源限制: CPU={cpu_limit}, Memory={mem_limit}")
    logger.info(f"📍 存储类: {storage_class}")
    logger.info(f"📍 共享存储: {'启用' if shared_storage_enabled else '禁用'}")

def get_spawner_config():
    """获取spawner配置信息"""
    spawner_type = os.environ.get('JUPYTERHUB_SPAWNER', 'docker')
    
    config_info = {
        'spawner_type': spawner_type,
        'namespace': os.environ.get('KUBERNETES_NAMESPACE', 'ai-infra-users'),
        'image': os.environ.get('JUPYTERHUB_IMAGE', 'ai-infra-singleuser:v0.3.6-dev'),
        'storage_class': os.environ.get('JUPYTERHUB_STORAGE_CLASS', 'local-path'),
        'mem_limit': os.environ.get('JUPYTERHUB_MEM_LIMIT', '2G'),
        'cpu_limit': os.environ.get('JUPYTERHUB_CPU_LIMIT', '1.0'),
        'shared_storage': os.environ.get('SHARED_STORAGE_ENABLED', 'true').lower() == 'true'
    }
    
    return config_info

# 测试Kubernetes连接
def test_kubernetes_connection():
    """测试Kubernetes API连接"""
    try:
        from kubernetes import config, client
        
        # 尝试加载集群内配置或本地kubeconfig
        try:
            config.load_incluster_config()
            logger.info("✅ 使用集群内Kubernetes配置")
        except:
            config.load_kube_config()
            logger.info("✅ 使用本地Kubernetes配置")
        
        # 测试API连接
        v1 = client.CoreV1Api()
        namespaces = v1.list_namespace()
        logger.info(f"✅ Kubernetes连接成功，发现 {len(namespaces.items)} 个命名空间")
        return True
        
    except Exception as e:
        logger.error(f"❌ Kubernetes连接失败: {e}")
        return False

if __name__ == "__main__":
    # 配置测试
    print("="*60)
    print("🚀 AI-Infra KubeSpawner配置测试")
    print("="*60)
    
    config_info = get_spawner_config()
    for key, value in config_info.items():
        print(f"📍 {key}: {value}")
    
    print("="*60)
    test_kubernetes_connection()
    print("="*60)
