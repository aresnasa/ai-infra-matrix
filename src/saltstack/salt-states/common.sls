# 通用基础配置
essential_packages:
  pkg.installed:
    - pkgs:
      - curl
      - wget
      - vim
      - htop
      - net-tools
      - rsync

# 时区设置
timezone:
  timezone.system:
    - name: Asia/Shanghai

# NTP服务
ntp_service:
  service.running:
    - name: systemd-timesyncd
    - enable: True

# 基础目录创建
ai_infra_dirs:
  file.directory:
    - names:
      - /opt/ai-infra
      - /var/log/ai-infra
      - /etc/ai-infra
    - makedirs: True
    - user: root
    - group: root
    - mode: 755

# 系统监控脚本
monitoring_script:
  file.managed:
    - name: /opt/ai-infra/monitor.sh
    - contents: |
        #!/bin/bash
        # AI Infrastructure 系统监控脚本
        echo "$(date): System check - $(hostname)"
        df -h
        free -h
        ps aux | head -10
    - mode: 755
    - user: root
    - group: root
