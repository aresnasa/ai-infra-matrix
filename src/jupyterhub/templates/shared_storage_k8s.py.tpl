# Shared storage configuration for Kubernetes
if os.environ.get('SHARED_STORAGE_ENABLED', 'false').lower() == 'true':
    shared_volume = {
        'name': 'shared-notebooks',
        'persistentVolumeClaim': {
            'claimName': 'shared-notebooks-pvc'
        }
    }
    
    shared_volume_mount = {
        'name': 'shared-notebooks',
        'mountPath': '/home/jovyan/shared'
    }
    
    c.KubeSpawner.volumes = [shared_volume]
    c.KubeSpawner.volume_mounts = [shared_volume_mount]
