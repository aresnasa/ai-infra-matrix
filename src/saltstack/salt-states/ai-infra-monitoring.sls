# AI Infrastructure 监控配置
ai_infra_agent_install:
  pkg.installed:
    - pkgs:
      - python3
      - python3-pip
      - docker.io

# AI Infrastructure 代理服务
ai_infra_agent_config:
  file.managed:
    - name: /etc/ai-infra/agent.conf
    - contents: |
        [agent]
        server_url = http://ai-infra-backend:8082
        agent_id = {{ grains['id'] }}
        heartbeat_interval = 30
        
        [monitoring]
        collect_system_stats = true
        collect_docker_stats = true
        collect_gpu_stats = false
        
        [logging]
        log_level = INFO
        log_file = /var/log/ai-infra/agent.log
    - makedirs: True
    - user: root
    - group: root
    - mode: 644

# 代理服务脚本
ai_infra_agent_service:
  file.managed:
    - name: /opt/ai-infra/agent.py
    - contents: |
        #!/usr/bin/env python3
        # AI Infrastructure Agent
        import requests
        import json
        import time
        import socket
        import subprocess
        
        def get_system_info():
            return {
                'hostname': socket.gethostname(),
                'timestamp': int(time.time()),
                'status': 'online',
                'services': get_running_services()
            }
        
        def get_running_services():
            try:
                result = subprocess.run(['docker', 'ps', '--format', 'json'], 
                                      capture_output=True, text=True)
                return len(result.stdout.strip().split('\n')) if result.stdout.strip() else 0
            except:
                return 0
                
        if __name__ == '__main__':
            print("AI Infrastructure Agent started")
            while True:
                try:
                    data = get_system_info()
                    print(f"Reporting: {data}")
                    time.sleep(30)
                except Exception as e:
                    print(f"Error: {e}")
                    time.sleep(10)
    - mode: 755
    - user: root
    - group: root

# 系统服务配置
ai_infra_systemd_service:
  file.managed:
    - name: /etc/systemd/system/ai-infra-agent.service
    - contents: |
        [Unit]
        Description=AI Infrastructure Agent
        After=network.target
        
        [Service]
        Type=simple
        User=root
        ExecStart=/opt/ai-infra/agent.py
        Restart=always
        RestartSec=10
        
        [Install]
        WantedBy=multi-user.target
    - require:
      - file: ai_infra_agent_service

# 启动服务
ai_infra_agent_running:
  service.running:
    - name: ai-infra-agent
    - enable: True
    - require:
      - file: ai_infra_systemd_service
    - watch:
      - file: ai_infra_agent_service
      - file: ai_infra_agent_config
